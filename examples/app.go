// Package examples provides a comprehensive example application using go-ormx
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go-ormx/examples/models"
	exampleRepos "go-ormx/examples/repositories"
	gormx "go-ormx/ormx"
	"go-ormx/ormx/config"
	"go-ormx/ormx/logging"
	"go-ormx/ormx/repositories"

	"github.com/oklog/ulid/v2"
)

// ExampleApp demonstrates a complete application using go-ormx
type ExampleApp struct {
	client   *gormx.Client
	userRepo *exampleRepos.UserRepository
	logger   logging.Logger
}

// NewExampleApp creates a new example application
func NewExampleApp() (*ExampleApp, error) {
	// Create logger
	logger := &SimpleLogger{}

	// Create configuration
	cfg := gormx.Config{
		Database: &config.Config{
			Type:     config.PostgreSQL,
			Host:     "localhost",
			Port:     5432,
			Database: "example_db",
			Username: "postgres",
			Password: "password",

			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
			ConnMaxIdleTime: 1 * time.Minute,

			PreparedStatements: true,
			BatchSize:          100,
			SkipDefaultTx:      false,
		},
		Options: gormx.ClientOptions{
			EnableMigrations: true,
			AutoMigrate:      true,
			EnableMetrics:    true,
			SkipHealthCheck:  false,
		},
	}

	// Create client
	client, err := gormx.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Create user repository
	userRepo := exampleRepos.NewUserRepository(
		client.Database().DB(),
		logger,
		repositories.RepositoryOptions{
			BatchSize:      100,
			ValidateOnSave: true,
		},
	)

	return &ExampleApp{
		client:   client,
		userRepo: userRepo,
		logger:   logger,
	}, nil
}

// Run runs the example application
func (app *ExampleApp) Run() error {
	ctx := context.Background()

	fmt.Println("=== Go-ORMX Example Application ===")

	// Run migrations
	if err := app.runMigrations(ctx); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Run user management examples
	if err := app.runUserExamples(ctx); err != nil {
		return fmt.Errorf("user examples failed: %w", err)
	}

	// Run tenant examples
	if err := app.runTenantExamples(ctx); err != nil {
		return fmt.Errorf("tenant examples failed: %w", err)
	}

	// Run repository examples
	if err := app.runRepositoryExamples(ctx); err != nil {
		return fmt.Errorf("repository examples failed: %w", err)
	}

	// Run transaction examples
	if err := app.runTransactionExamples(ctx); err != nil {
		return fmt.Errorf("transaction examples failed: %w", err)
	}

	fmt.Println("\n=== Example Application Completed Successfully ===")
	return nil
}

// runMigrations demonstrates migration functionality
func (app *ExampleApp) runMigrations(ctx context.Context) error {
	fmt.Println("\n1. Running Migrations...")

	// Get migration status
	status, err := app.client.GetMigrationStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}

	fmt.Printf("✓ Current version: %d\n", status.Version)
	fmt.Printf("✓ Dirty: %t\n", status.Dirty)
	fmt.Printf("✓ Applied count: %d\n", status.AppliedCount)
	fmt.Printf("✓ Pending count: %d\n", status.PendingCount)

	// Run migrations if there are pending ones
	if status.PendingCount > 0 {
		if err := app.client.Migrate(ctx); err != nil {
			return fmt.Errorf("failed to run migrations: %w", err)
		}
		fmt.Println("✓ Migrations completed successfully")
	} else {
		fmt.Println("✓ No pending migrations")
	}

	return nil
}

// runUserExamples demonstrates user model functionality
func (app *ExampleApp) runUserExamples(ctx context.Context) error {
	fmt.Println("\n2. User Model Examples...")

	// Create a tenant ID
	tenantID := ulid.Make().String()

	// Create a user
	user := &models.User{
		Email:        "john.doe@example.com",
		Username:     "johndoe",
		FirstName:    "John",
		LastName:     "Doe",
		PasswordHash: "hashed_password_123",
		Salt:         "salt_123",
		Status:       models.UserStatusActive,
		Role:         models.UserRoleUser,
	}
	user.SetTenantID(tenantID)

	// Demonstrate user methods
	fmt.Printf("✓ User full name: %s\n", user.GetFullName())
	fmt.Printf("✓ User is active: %t\n", user.IsActive())
	fmt.Printf("✓ User can login: %t\n", user.CanLogin())
	fmt.Printf("✓ User is admin: %t\n", user.IsAdmin())

	// Update user
	user.MarkEmailVerified()
	user.UpdateLastLogin("192.168.1.1")
	user.EnableTwoFactor("secret_key_123")

	fmt.Printf("✓ Email verified: %t\n", user.EmailVerified)
	fmt.Printf("✓ Two-factor enabled: %t\n", user.TwoFactorEnabled)

	return nil
}

// runTenantExamples demonstrates tenant model functionality
func (app *ExampleApp) runTenantExamples(ctx context.Context) error {
	fmt.Println("\n3. Tenant Model Examples...")

	// Create a tenant
	tenant := &models.Tenant{
		Name:        "Example Corp",
		Slug:        "example-corp",
		Domain:      "example.com",
		Status:      models.TenantStatusActive,
		Plan:        models.TenantPlanPro,
		Description: "A sample tenant for demonstration",
		MaxUsers:    100,
		MaxStorage:  10737418240, // 10GB
		MaxProjects: 50,
	}

	// Demonstrate tenant methods
	fmt.Printf("✓ Tenant is active: %t\n", tenant.IsActive())
	fmt.Printf("✓ Tenant is paid plan: %t\n", tenant.IsPaidPlan())
	fmt.Printf("✓ Can add user: %t\n", tenant.CanAddUser(50))
	fmt.Printf("✓ Can add project: %t\n", tenant.CanAddProject(25))
	fmt.Printf("✓ Can use storage: %t\n", tenant.CanUseStorage(5368709120)) // 5GB

	// Update tenant
	tenant.UpdateLastActivity()
	tenant.EnableAPI()
	tenant.EnableSSO("google", `{"client_id": "123", "client_secret": "456"}`)

	fmt.Printf("✓ API enabled: %t\n", tenant.APIEnabled)
	fmt.Printf("✓ SSO enabled: %t\n", tenant.SSOEnabled)

	return nil
}

// runRepositoryExamples demonstrates repository functionality
func (app *ExampleApp) runRepositoryExamples(ctx context.Context) error {
	fmt.Println("\n4. Repository Examples...")

	// Create a tenant ID
	tenantID := ulid.Make().String()

	// Create users
	users := []*models.User{
		{
			Email:        "alice@example.com",
			Username:     "alice",
			FirstName:    "Alice",
			LastName:     "Johnson",
			PasswordHash: "hash1",
			Salt:         "salt1",
			Status:       models.UserStatusActive,
			Role:         models.UserRoleAdmin,
		},
		{
			Email:        "bob@example.com",
			Username:     "bob",
			FirstName:    "Bob",
			LastName:     "Smith",
			PasswordHash: "hash2",
			Salt:         "salt2",
			Status:       models.UserStatusActive,
			Role:         models.UserRoleUser,
		},
		{
			Email:        "charlie@example.com",
			Username:     "charlie",
			FirstName:    "Charlie",
			LastName:     "Brown",
			PasswordHash: "hash3",
			Salt:         "salt3",
			Status:       models.UserStatusPending,
			Role:         models.UserRoleUser,
		},
	}

	// Set tenant ID for all users
	for _, user := range users {
		user.SetTenantID(tenantID)
	}

	// Create users in batch
	if err := app.userRepo.CreateBatch(ctx, users); err != nil {
		return fmt.Errorf("failed to create users: %w", err)
	}
	fmt.Println("✓ Created 3 users successfully")

	// Get user by email
	user, err := app.userRepo.GetByEmail(ctx, "alice@example.com")
	if err != nil {
		return fmt.Errorf("failed to get user by email: %w", err)
	}
	fmt.Printf("✓ Retrieved user: %s\n", user.GetFullName())

	// Get active users
	activeUsers, err := app.userRepo.GetActiveUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active users: %w", err)
	}
	fmt.Printf("✓ Found %d active users\n", len(activeUsers))

	// Get admins
	admins, err := app.userRepo.GetAdmins(ctx)
	if err != nil {
		return fmt.Errorf("failed to get admins: %w", err)
	}
	fmt.Printf("✓ Found %d admin users\n", len(admins))

	// Search users
	searchResults, err := app.userRepo.SearchUsers(ctx, "alice")
	if err != nil {
		return fmt.Errorf("failed to search users: %w", err)
	}
	fmt.Printf("✓ Search found %d users\n", len(searchResults))

	// Update user status
	if err := app.userRepo.UpdateUserStatus(ctx, user.ID, models.UserStatusSuspended); err != nil {
		return fmt.Errorf("failed to update user status: %w", err)
	}
	fmt.Println("✓ Updated user status to suspended")

	// Get user count
	count, err := app.userRepo.GetUserCount(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user count: %w", err)
	}
	fmt.Printf("✓ Total users: %d\n", count)

	return nil
}

// runTransactionExamples demonstrates transaction functionality
func (app *ExampleApp) runTransactionExamples(ctx context.Context) error {
	fmt.Println("\n5. Transaction Examples...")

	// Start a transaction
	tx, err := app.userRepo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Create a repository with transaction
	txUserRepo := app.userRepo.WithTx(tx)

	// Create a user within transaction
	user := &models.User{
		Email:        "transaction@example.com",
		Username:     "transaction_user",
		FirstName:    "Transaction",
		LastName:     "User",
		PasswordHash: "hash_tx",
		Salt:         "salt_tx",
		Status:       models.UserStatusActive,
		Role:         models.UserRoleUser,
	}
	user.SetTenantID(ulid.Make().String())

	if err := txUserRepo.Create(ctx, user); err != nil {
		// Rollback on error
		if rollbackErr := app.userRepo.RollbackTx(tx); rollbackErr != nil {
			return fmt.Errorf("failed to rollback transaction: %w", rollbackErr)
		}
		return fmt.Errorf("failed to create user in transaction: %w", err)
	}

	// Update user within transaction
	if err := txUserRepo.UpdateUserRole(ctx, user.ID, models.UserRoleModerator); err != nil {
		// Rollback on error
		if rollbackErr := app.userRepo.RollbackTx(tx); rollbackErr != nil {
			return fmt.Errorf("failed to rollback transaction: %w", rollbackErr)
		}
		return fmt.Errorf("failed to update user role in transaction: %w", err)
	}

	// Commit transaction
	if err := app.userRepo.CommitTx(tx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	fmt.Println("✓ Transaction completed successfully")

	// Verify the user was created
	createdUser, err := app.userRepo.GetByID(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("failed to get created user: %w", err)
	}
	fmt.Printf("✓ Verified user created: %s (Role: %s)\n", createdUser.GetFullName(), createdUser.Role)

	return nil
}

// Close closes the application
func (app *ExampleApp) Close() error {
	return app.client.Close()
}

// SimpleLogger implements a simple logger for examples
type SimpleLogger struct{}

func (l *SimpleLogger) Trace(msg string, fields ...logging.LogField)   { log.Printf("TRACE: %s", msg) }
func (l *SimpleLogger) Debug(msg string, fields ...logging.LogField)   { log.Printf("DEBUG: %s", msg) }
func (l *SimpleLogger) Info(msg string, fields ...logging.LogField)    { log.Printf("INFO: %s", msg) }
func (l *SimpleLogger) Warn(msg string, fields ...logging.LogField)    { log.Printf("WARN: %s", msg) }
func (l *SimpleLogger) Error(msg string, fields ...logging.LogField)   { log.Printf("ERROR: %s", msg) }
func (l *SimpleLogger) Fatal(msg string, fields ...logging.LogField)   { log.Fatalf("FATAL: %s", msg) }
func (l *SimpleLogger) With(fields ...logging.LogField) logging.Logger { return l }

// Main function to run the example application
func main() {
	app, err := NewExampleApp()
	if err != nil {
		log.Fatalf("Failed to create example app: %v", err)
	}
	defer app.Close()

	if err := app.Run(); err != nil {
		log.Fatalf("Example app failed: %v", err)
	}
}
