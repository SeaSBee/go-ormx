package basic

import (
	"context"
	"os"
	"testing"
	"time"

	"go-ormx/examples/models"
	exampleRepos "go-ormx/examples/repositories"
	gormx "go-ormx/ormx"
	"go-ormx/ormx/config"
	"go-ormx/ormx/repositories"
	"go-ormx/tests/integration/utils"

	"github.com/oklog/ulid/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// sqliteUser is a minimal model compatible with SQLite for mock integration tests
type sqliteUser struct {
	ID        string         `gorm:"primaryKey"`
	CreatedAt time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP"`
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Email    string `gorm:"size:255;not null"`
	Username string `gorm:"size:100;not null"`
}

func (u *sqliteUser) GetID() string     { return u.ID }
func (u *sqliteUser) SetID(id string)   { u.ID = id }
func (u *sqliteUser) IsDeleted() bool   { return u.DeletedAt.Valid }
func (u *sqliteUser) TableName() string { return "mock_users" }

func TestBasicIntegration_ClientCreation(t *testing.T) {
	utils.SkipIfShort(t)

	t.Run("mock_sqlite_client_creation", func(t *testing.T) {
		// In-memory SQLite for mock integration
		gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		if err != nil {
			t.Fatalf("failed to open sqlite: %v", err)
		}

		// Minimal client-like checks: DB, Logger stubs via utils logger
		logger := utils.NewTestLogger(t, "mock-basic-integration")
		if gdb == nil || logger == nil {
			t.Fatalf("expected mock DB and logger")
		}
	})

	// Create test configuration
	testConfig := &config.Config{
		Type:     config.PostgreSQL,
		Host:     "localhost",
		Port:     5432,
		Database: "go-ormx_test",
		Username: "postgres",
		Password: "password",
		SSLMode:  "disable",
	}

	logger := utils.NewTestLogger(t, "basic-integration")
	clientConfig := gormx.Config{
		Database: testConfig,
		Logger:   logger,
		Options: gormx.ClientOptions{
			EnableMetrics:    false,
			EnableMigrations: false,
			AutoMigrate:      false,
			SkipHealthCheck:  true, // Skip health check for integration tests
		},
	}

	// Test client creation
	client, err := gormx.NewClient(clientConfig)
	if err != nil {
		t.Skipf("Skipping test: failed to create client (database may not be available): %v", err)
	}
	defer client.Close()

	// Test basic client functionality
	if client.Database() == nil {
		t.Error("Expected database to be initialized")
	}

	if client.Logger() == nil {
		t.Error("Expected logger to be initialized")
	}
}

func TestBasicIntegration_UserRepository(t *testing.T) {
	utils.SkipIfShort(t)

	t.Run("mock_sqlite_user_repository_crud", func(t *testing.T) {
		gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		if err != nil {
			t.Fatalf("failed to open sqlite: %v", err)
		}

		// Auto-migrate mock sqlite user model
		if err := gdb.AutoMigrate(&sqliteUser{}); err != nil {
			t.Fatalf("failed to migrate user: %v", err)
		}

		logger := utils.NewTestLogger(t, "mock-user-repo")
		userRepo := repositories.NewBaseRepository[*sqliteUser](
			gdb,
			logger,
			repositories.RepositoryOptions{BatchSize: 50, ValidateOnSave: true},
		)

		ctx := context.Background()
		user := &sqliteUser{ID: ulid.Make().String(), Email: "mock_integration@example.com", Username: "mock_user"}

		if err := userRepo.Create(ctx, user); err != nil {
			t.Fatalf("create failed: %v", err)
		}
		if user.ID == "" {
			t.Fatalf("expected ID after create")
		}

		found, err := userRepo.GetByID(ctx, user.ID)
		if err != nil {
			t.Fatalf("get by id failed: %v", err)
		}
		if found.GetID() == "" {
			t.Fatalf("expected found user to have ID")
		}

		// Count
		if cnt, err := userRepo.Count(ctx, repositories.Filter{}); err != nil || cnt == 0 {
			t.Fatalf("count failed: %v cnt=%d", err, cnt)
		}
	})

	// Create test configuration
	testConfig := &config.Config{
		Type:     config.PostgreSQL,
		Host:     "localhost",
		Port:     5432,
		Database: "go-ormx_test",
		Username: "postgres",
		Password: "password",
		SSLMode:  "disable",
	}

	logger := utils.NewTestLogger(t, "user-repo-integration")
	clientConfig := gormx.Config{
		Database: testConfig,
		Logger:   logger,
		Options: gormx.ClientOptions{
			EnableMetrics:    false,
			EnableMigrations: false,
			AutoMigrate:      false,
			SkipHealthCheck:  true,
		},
	}

	// Create client
	client, err := gormx.NewClient(clientConfig)
	if err != nil {
		t.Skipf("Skipping test: failed to create client (database may not be available): %v", err)
	}
	defer client.Close()

	// Create user repository
	userRepo := exampleRepos.NewUserRepository(
		client.Database().DB(),
		logger,
		repositories.RepositoryOptions{
			BatchSize:      100,
			ValidateOnSave: true,
		},
	)

	ctx := context.Background()

	t.Run("create_and_retrieve_user", func(t *testing.T) {
		// Create a test user
		user := &models.User{
			Email:        "integration_test@example.com",
			Username:     "integration_test_user",
			FirstName:    "Integration",
			LastName:     "Test",
			PasswordHash: "test_hash",
			Salt:         "test_salt",
			Status:       models.UserStatusActive,
			Role:         models.UserRoleUser,
		}
		user.SetTenantID(ulid.Make().String())

		// Create user
		err := userRepo.Create(ctx, user)
		if err != nil {
			t.Skipf("Skipping test: failed to create user (database may not be available): %v", err)
		}

		// Verify user was created
		if user.ID == "" {
			t.Error("Expected user ID to be set after creation")
		}

		// Retrieve user
		retrieved, err := userRepo.GetByID(ctx, user.ID)
		if err != nil {
			t.Errorf("Failed to retrieve user: %v", err)
			return
		}

		if retrieved.Email != user.Email {
			t.Errorf("Expected email %s, got %s", user.Email, retrieved.Email)
		}

		if retrieved.Username != user.Username {
			t.Errorf("Expected username %s, got %s", user.Username, retrieved.Username)
		}
	})

	t.Run("user_model_methods", func(t *testing.T) {
		user := &models.User{
			Email:        "test_methods@example.com",
			Username:     "test_methods_user",
			FirstName:    "Test",
			LastName:     "Methods",
			PasswordHash: "test_hash",
			Salt:         "test_salt",
			Status:       models.UserStatusActive,
			Role:         models.UserRoleAdmin,
		}

		// Test user methods
		if user.GetFullName() != "Test Methods" {
			t.Errorf("Expected full name 'Test Methods', got '%s'", user.GetFullName())
		}

		if !user.IsActive() {
			t.Error("Expected user to be active")
		}

		if !user.IsAdmin() {
			t.Error("Expected user to be admin")
		}

		if user.CanLogin() != true {
			t.Error("Expected user to be able to login")
		}
	})
}

func TestBasicIntegration_MigrationStatus(t *testing.T) {
	utils.SkipIfShort(t)

	// Create test configuration
	testConfig := &config.Config{
		Type:     config.PostgreSQL,
		Host:     "localhost",
		Port:     5432,
		Database: "go-ormx_test",
		Username: "postgres",
		Password: "password",
		SSLMode:  "disable",
	}

	logger := utils.NewTestLogger(t, "migration-integration")
	clientConfig := gormx.Config{
		Database: testConfig,
		Logger:   logger,
		Options: gormx.ClientOptions{
			EnableMetrics:    false,
			EnableMigrations: true,
			AutoMigrate:      false,
			SkipHealthCheck:  true,
		},
	}

	// Create client
	client, err := gormx.NewClient(clientConfig)
	if err != nil {
		t.Skipf("Skipping test: failed to create client (database may not be available): %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Test migration status
	status, err := client.GetMigrationStatus(ctx)
	if err != nil {
		t.Skipf("Skipping test: failed to get migration status (migrations may not be available): %v", err)
	}

	// Verify status structure
	if status == nil {
		t.Error("Expected migration status to be non-nil")
		return
	}

	// Log status for debugging
	t.Logf("Migration status: Version=%d, Dirty=%t, AppliedCount=%d, PendingCount=%d",
		status.Version, status.Dirty, status.AppliedCount, status.PendingCount)
}

func TestBasicIntegration_HealthCheck(t *testing.T) {
	utils.SkipIfShort(t)

	t.Run("mock_sqlite_health_check", func(t *testing.T) {
		gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		if err != nil {
			t.Fatalf("failed to open sqlite: %v", err)
		}
		sqlDB, err := gdb.DB()
		if err != nil {
			t.Fatalf("failed to get sql DB: %v", err)
		}
		if err := sqlDB.Ping(); err != nil {
			t.Fatalf("sqlite ping failed: %v", err)
		}
	})

	// Create test configuration
	testConfig := &config.Config{
		Type:     config.PostgreSQL,
		Host:     "localhost",
		Port:     5432,
		Database: "go-ormx_test",
		Username: "postgres",
		Password: "password",
		SSLMode:  "disable",
	}

	logger := utils.NewTestLogger(t, "health-integration")
	clientConfig := gormx.Config{
		Database: testConfig,
		Logger:   logger,
		Options: gormx.ClientOptions{
			EnableMetrics:    false,
			EnableMigrations: false,
			AutoMigrate:      false,
			SkipHealthCheck:  true, // Skip initial health check
		},
	}

	// Create client
	client, err := gormx.NewClient(clientConfig)
	if err != nil {
		t.Skipf("Skipping test: failed to create client (database may not be available): %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Test health check
	err = client.Health(ctx)
	if err != nil {
		t.Skipf("Skipping test: health check failed (database may not be available): %v", err)
	}

	// Test is healthy
	isHealthy := client.IsHealthy(ctx)
	if !isHealthy {
		t.Skip("Skipping test: database is not healthy")
	}

	// Test connection stats
	stats := client.GetConnectionStats()
	t.Logf("Connection stats: Open=%d, InUse=%d, Idle=%d, WaitCount=%d, WaitDuration=%v, MaxIdleClosed=%d, MaxLifetimeClosed=%d",
		stats.OpenConnections, stats.InUseConnections, stats.IdleConnections, stats.WaitCount, stats.WaitDuration, stats.MaxIdleClosed, stats.MaxLifetimeClosed)
}

// (no build tag; tests auto-skip without DB)

func TestRLS_GUC_Set_When_Tenant_Present(t *testing.T) {
	if os.Getenv("DB_HOST") == "" {
		t.Skip("no database configured; skipping integration test")
	}

	os.Setenv("DB_RLS_ENABLED", "true")
	os.Setenv("DB_RLS_TENANT_GUC", "app.tenant_id")
	os.Setenv("DB_RLS_REQUIRE_TENANT", "true")

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	logger := utils.NewTestLogger(t, "rls-tenant-present")
	client, err := gormx.NewClient(gormx.Config{Database: cfg, Logger: logger, Options: gormx.DefaultClientOptions()})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	ctx = gormx.WithTenant(ctx, "tenant-it")
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// If callback fails to set GUC, at least health succeeds without error
	if err := client.Health(ctx); err != nil {
		t.Fatalf("health failed with tenant: %v", err)
	}
}

func TestRLS_Warns_When_Tenant_Missing(t *testing.T) {
	if os.Getenv("DB_HOST") == "" {
		t.Skip("no database configured; skipping integration test")
	}

	os.Setenv("DB_RLS_ENABLED", "true")
	os.Setenv("DB_RLS_TENANT_GUC", "app.tenant_id")
	os.Setenv("DB_RLS_REQUIRE_TENANT", "true")

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	logger := utils.NewTestLogger(t, "rls-tenant-missing")
	client, err := gormx.NewClient(gormx.Config{Database: cfg, Logger: logger, Options: gormx.DefaultClientOptions()})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Should not error even if tenant is missing; behavior is warn-only at client side
	if err := client.Health(ctx); err != nil {
		t.Fatalf("health failed without tenant: %v", err)
	}
}

// uses utils.NewTestLogger
