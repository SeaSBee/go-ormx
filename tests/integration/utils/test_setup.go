// Package utils provides common utilities and setup for integration tests
package utils

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	gormx "go-ormx/ormx"
	"go-ormx/ormx/config"
	"go-ormx/ormx/logging"
	"go-ormx/ormx/models"

	"github.com/oklog/ulid/v2"
)

// TestConfig holds configuration for integration tests
type TestConfig struct {
	DatabaseURL      string
	DatabaseType     config.DatabaseType
	TestDatabaseName string
	CleanupAfterTest bool
	UseTestContainer bool
	LogLevel         logging.LogLevel
}

// DefaultTestConfig returns default configuration for integration tests
func DefaultTestConfig() *TestConfig {
	return &TestConfig{
		DatabaseURL:      GetEnvWithDefault("TEST_DATABASE_URL", "postgres://postgres:password@localhost:5432/go-ormx_test?sslmode=disable"),
		DatabaseType:     config.PostgreSQL,
		TestDatabaseName: fmt.Sprintf("go-ormx_test_%s", ulid.Make().String()[:8]),
		CleanupAfterTest: true,
		UseTestContainer: GetEnvBool("USE_TEST_CONTAINER", false),
		LogLevel:         logging.Info,
	}
}

// TestLogger implements the logging.Logger interface for tests
type TestLogger struct {
	t      *testing.T
	prefix string
}

// NewTestLogger creates a new test logger
func NewTestLogger(t *testing.T, prefix string) *TestLogger {
	return &TestLogger{t: t, prefix: prefix}
}

func (tl *TestLogger) Trace(msg string, fields ...logging.LogField) {
	tl.t.Logf("[TRACE] %s: %s %v", tl.prefix, msg, fields)
}

func (tl *TestLogger) Debug(msg string, fields ...logging.LogField) {
	tl.t.Logf("[DEBUG] %s: %s %v", tl.prefix, msg, fields)
}

func (tl *TestLogger) Info(msg string, fields ...logging.LogField) {
	tl.t.Logf("[INFO] %s: %s %v", tl.prefix, msg, fields)
}

func (tl *TestLogger) Warn(msg string, fields ...logging.LogField) {
	tl.t.Logf("[WARN] %s: %s %v", tl.prefix, msg, fields)
}

func (tl *TestLogger) Error(msg string, fields ...logging.LogField) {
	tl.t.Logf("[ERROR] %s: %s %v", tl.prefix, msg, fields)
}

func (tl *TestLogger) Fatal(msg string, fields ...logging.LogField) {
	tl.t.Fatalf("[FATAL] %s: %s %v", tl.prefix, msg, fields)
}

func (tl *TestLogger) With(fields ...logging.LogField) logging.Logger {
	return tl
}

// TestSetup provides setup and teardown for integration tests
type TestSetup struct {
	Config *TestConfig
	Client *gormx.Client
	Logger *TestLogger
	ctx    context.Context
}

// NewTestSetup creates a new test setup with default configuration
func NewTestSetup(t *testing.T) *TestSetup {
	return NewTestSetupWithConfig(t, DefaultTestConfig())
}

// NewTestSetupWithConfig creates a new test setup with custom configuration
func NewTestSetupWithConfig(t *testing.T, testConfig *TestConfig) *TestSetup {
	logger := NewTestLogger(t, "integration")

	// Create database configuration
	dbConfig := &config.Config{
		Type:               testConfig.DatabaseType,
		Host:               "localhost",
		Port:               5432,
		Database:           testConfig.TestDatabaseName,
		Username:           "postgres",
		Password:           "password",
		MaxOpenConns:       10,
		MaxIdleConns:       5,
		ConnMaxLifetime:    5 * time.Minute,
		ConnMaxIdleTime:    1 * time.Minute,
		PreparedStatements: true,
		BatchSize:          100,
		SkipDefaultTx:      false,
	}

	// Create client configuration
	clientConfig := gormx.Config{
		Database: dbConfig,
		Logger:   logger,
		Options: gormx.ClientOptions{
			EnableMigrations: true,
			AutoMigrate:      true,
			EnableMetrics:    true,
			SkipHealthCheck:  false,
		},
	}

	// Create client
	client, err := gormx.NewClient(clientConfig)
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	return &TestSetup{
		Config: testConfig,
		Client: client,
		Logger: logger,
		ctx:    context.Background(),
	}
}

// Context returns the test context
func (ts *TestSetup) Context() context.Context {
	return ts.ctx
}

// WithTimeout returns a context with timeout
func (ts *TestSetup) WithTimeout(duration time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ts.ctx, duration)
}

// TearDown cleans up test resources
func (ts *TestSetup) TearDown(t *testing.T) {
	if ts.Client != nil {
		ts.Client.Close()
	}
}

// SkipIfShort skips the test if -short flag is used
func SkipIfShort(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}
}

// RequireEnvironment requires specific environment variables to be set
func RequireEnvironment(t *testing.T, envVars ...string) {
	for _, envVar := range envVars {
		if os.Getenv(envVar) == "" {
			t.Skipf("Skipping test: %s environment variable not set", envVar)
		}
	}
}

// Helper functions
func GetEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func GetEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}

// DatabaseHealth checks if the database is healthy
func (ts *TestSetup) DatabaseHealth(t *testing.T) {
	if !ts.Client.IsHealthy(ts.ctx) {
		t.Fatal("Database is not healthy")
	}
}

// TestProduct represents a test product model
type TestProduct struct {
	models.BaseModel
	Name        string  `gorm:"type:varchar(255);not null" json:"name"`
	Description string  `gorm:"type:text" json:"description"`
	Price       float64 `gorm:"type:decimal(10,2);not null" json:"price"`
}

// TableName returns the table name for TestProduct
func (p *TestProduct) TableName() string {
	return "test_products"
}

// CreateTestProduct creates a test product
func (ts *TestSetup) CreateTestProduct(t *testing.T, name, description string, price float64) *TestProduct {
	product := &TestProduct{
		Name:        name,
		Description: description,
		Price:       price,
	}

	// This would typically use a repository to create the product
	// For now, we'll just return the product
	return product
}
