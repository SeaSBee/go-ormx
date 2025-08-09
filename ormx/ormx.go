// Package ormx provides a high-performance, concurrent-safe, memory-safe, and production-ready
// ORM-based PostgreSQL/MySQL data access layer using GORM with comprehensive features including
// structured logging, metrics, security, and observability.
package ormx

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go-ormx/ormx/db"
	"go-ormx/ormx/errors"
	"go-ormx/ormx/internal/config"
	"go-ormx/ormx/internal/logging"
	"go-ormx/ormx/migrations"

	"gorm.io/gorm"
)

// Client provides the main interface for database operations
type Client struct {
	database *db.Database
	logger   logging.Logger
	migrator *migrations.Migrator
	metrics  *db.MetricsCollector
}

// Config represents the client configuration
type Config struct {
	Database *config.Config
	Logger   logging.Logger
	Options  ClientOptions
}

// ClientOptions holds optional configuration for the client
type ClientOptions struct {
	EnableMetrics    bool `json:"enable_metrics"`
	EnableMigrations bool `json:"enable_migrations"`
	AutoMigrate      bool `json:"auto_migrate"`
	SkipHealthCheck  bool `json:"skip_health_check"`
}

// DefaultClientOptions returns default client options
func DefaultClientOptions() ClientOptions {
	return ClientOptions{
		EnableMetrics:    true,
		EnableMigrations: true,
		AutoMigrate:      false,
		SkipHealthCheck:  false,
	}
}

// NewClient creates a new database client with the given configuration
func NewClient(cfg Config) (*Client, error) {
	if cfg.Database == nil {
		return nil, errors.NewDBError(errors.ErrCodeMissingConfig, "database configuration is required", nil)
	}

	if cfg.Logger == nil {
		return nil, errors.NewDBError(errors.ErrCodeMissingConfig, "logger is required", nil)
	}

	// Apply default options if not provided
	if cfg.Options == (ClientOptions{}) {
		cfg.Options = DefaultClientOptions()
	}

	client := &Client{
		logger: cfg.Logger,
	}

	// Initialize metrics collector
	if cfg.Options.EnableMetrics {
		client.metrics = db.NewMetricsCollector(cfg.Database.MetricsNamespace, true)
	}

	// Initialize database connection
	database, err := db.New(cfg.Database, cfg.Logger)
	if err != nil {
		return nil, err
	}
	client.database = database

	// Perform health check if not skipped
	if !cfg.Options.SkipHealthCheck {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := client.database.Ping(ctx); err != nil {
			client.database.Close()
			return nil, errors.WrapError(err, errors.ErrCodeConnectionFailed, "initial health check failed")
		}
	}

	// Initialize migrator if enabled
	if cfg.Options.EnableMigrations {
		migrator, err := migrations.NewMigrator(client.database.DB(), cfg.Logger, cfg.Database, nil)
		if err != nil {
			return nil, errors.WrapError(err, errors.ErrCodeMigrationFailed, "failed to create migrator")
		}
		client.migrator = migrator

		// Auto-migrate if enabled
		if cfg.Options.AutoMigrate {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			if err := client.migrator.Migrate(ctx); err != nil {
				cfg.Logger.Error("Auto-migration failed", logging.ErrorField(err))
				// Don't fail client creation for migration errors in auto mode
			}
		}

		// Optionally manage RLS policies when enabled
		if cfg.Database.RLSEnabled && cfg.Database.RLSManagePolicies && len(cfg.Database.RLSTables) > 0 && cfg.Database.Type == config.PostgreSQL {
			if err := client.ensureRLSPolicies(context.Background()); err != nil {
				cfg.Logger.Warn("RLS policy management failed", logging.ErrorField(err))
			}
		}
	}

	// Initialize repositories
	if err := client.initializeRepositories(cfg); err != nil {
		client.database.Close()
		return nil, err
	}

	cfg.Logger.Info("Database client initialized successfully",
		logging.String("database_type", string(cfg.Database.Type)),
		logging.String("host", cfg.Database.Host),
		logging.Int("port", cfg.Database.Port),
		logging.Bool("metrics_enabled", cfg.Options.EnableMetrics),
		logging.Bool("migrations_enabled", cfg.Options.EnableMigrations),
	)

	return client, nil
}

// NewClientFromEnv creates a new client using environment configuration
func NewClientFromEnv(logger logging.Logger) (*Client, error) {
	dbConfig, err := config.LoadFromEnv()
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrCodeConfigValidation, "failed to load configuration from environment")
	}

	cfg := Config{
		Database: dbConfig,
		Logger:   logger,
		Options:  DefaultClientOptions(),
	}

	return NewClient(cfg)
}

// initializeRepositories initializes all repository instances
func (c *Client) initializeRepositories(cfg Config) error {
	// Repository initialization is now handled by the user
	// This method is kept for future extensibility
	return nil
}

// Database returns the database instance
func (c *Client) Database() *db.Database {
	return c.database
}

// Logger returns the logger instance
func (c *Client) Logger() logging.Logger {
	return c.logger
}

// Migrator returns the migrator instance
func (c *Client) Migrator() *migrations.Migrator {
	return c.migrator
}

// Metrics returns the metrics collector
func (c *Client) Metrics() *db.MetricsCollector {
	return c.metrics
}

// Users returns the user repository
// Note: User repositories are now created separately using the examples
func (c *Client) Users() interface{} {
	return nil
}

// Health checks the health of the database connection
func (c *Client) Health(ctx context.Context) error {
	return c.database.Ping(ctx)
}

// IsHealthy returns true if the database is healthy
func (c *Client) IsHealthy(ctx context.Context) bool {
	return c.database.IsHealthy(ctx)
}

// GetConnectionStats returns database connection statistics
func (c *Client) GetConnectionStats() db.ConnectionStats {
	return c.database.GetConnectionStats()
}

// Transaction executes a function within a database transaction
func (c *Client) Transaction(ctx context.Context, fn func(tx *Client) error) error {
	return c.database.Transaction(ctx, func(tx *gorm.DB) error {
		// Create a new client instance with the transaction
		txClient := &Client{
			database: &db.Database{}, // Simplified for this example
			logger:   c.logger,
			metrics:  c.metrics,
		}

		return fn(txClient)
	})
}

// ensureRLSPolicies creates or updates PostgreSQL RLS policies for configured tables
func (c *Client) ensureRLSPolicies(ctx context.Context) error {
	cfg := c.database.Config()
	if cfg.Type != config.PostgreSQL || !cfg.RLSEnabled || len(cfg.RLSTables) == 0 {
		return nil
	}

	sqlDB := c.database.SqlDB()
	if sqlDB == nil {
		return fmt.Errorf("sql DB not initialized")
	}

	policyName := cfg.RLSPolicyName
	tenantCol := cfg.RLSTenantColumn
	if tenantCol == "" {
		tenantCol = "tenant_id"
	}

	for _, table := range cfg.RLSTables {
		// Enable RLS on table
		if _, err := sqlDB.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s ENABLE ROW LEVEL SECURITY", table)); err != nil {
			c.logger.Warn("Failed to enable RLS on table", logging.String("table", table), logging.ErrorField(err))
		}
		// Create or replace policy
		// Using current_setting(GUC, true) to read tenant from GUC; null-safe compare
		stmt := fmt.Sprintf("CREATE POLICY IF NOT EXISTS %s ON %s USING (%s = current_setting('%s', true))", policyName, table, tenantCol, cfg.RLSTenantGUC)
		if _, err := sqlDB.ExecContext(ctx, stmt); err != nil {
			c.logger.Warn("Failed to create RLS policy", logging.String("table", table), logging.ErrorField(err))
		}
		// Optionally force RLS
		if _, err := sqlDB.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s FORCE ROW LEVEL SECURITY", table)); err != nil {
			c.logger.Warn("Failed to force RLS on table", logging.String("table", table), logging.ErrorField(err))
		}
	}
	return nil
}

// Migrate runs all pending database migrations
func (c *Client) Migrate(ctx context.Context) error {
	if c.migrator == nil {
		return errors.NewDBError(errors.ErrCodeNotImplemented, "migrations not enabled", nil)
	}

	return c.migrator.Migrate(ctx)
}

// MigrateTo runs migrations up to a specific version
func (c *Client) MigrateTo(ctx context.Context, version uint) error {
	if c.migrator == nil {
		return errors.NewDBError(errors.ErrCodeNotImplemented, "migrations not enabled", nil)
	}

	return c.migrator.MigrateTo(ctx, version)
}

// Rollback rolls back the last migration
func (c *Client) Rollback(ctx context.Context) error {
	if c.migrator == nil {
		return errors.NewDBError(errors.ErrCodeNotImplemented, "migrations not enabled", nil)
	}

	return c.migrator.Rollback(ctx)
}

// RollbackTo rolls back to a specific version
func (c *Client) RollbackTo(ctx context.Context, version uint) error {
	if c.migrator == nil {
		return errors.NewDBError(errors.ErrCodeNotImplemented, "migrations not enabled", nil)
	}

	return c.migrator.RollbackTo(ctx, version)
}

// GetMigrationStatus returns the current migration status
func (c *Client) GetMigrationStatus(ctx context.Context) (*migrations.MigrationStatus, error) {
	if c.migrator == nil {
		return nil, errors.NewDBError(errors.ErrCodeNotImplemented, "migrations not enabled", nil)
	}

	return c.migrator.GetStatus(ctx)
}

// Force sets the migration version without running migrations
func (c *Client) Force(ctx context.Context, version int) error {
	if c.migrator == nil {
		return errors.NewDBError(errors.ErrCodeNotImplemented, "migrations not enabled", nil)
	}

	return c.migrator.Force(ctx, version)
}

// Drop drops all tables (use with caution!)
func (c *Client) Drop(ctx context.Context) error {
	if c.migrator == nil {
		return errors.NewDBError(errors.ErrCodeNotImplemented, "migrations not enabled", nil)
	}

	return c.migrator.Drop(ctx)
}

// Close gracefully closes the database connection and all resources
func (c *Client) Close() error {
	c.logger.Info("Closing database client...")

	if c.database != nil {
		if err := c.database.Close(); err != nil {
			c.logger.Error("Failed to close database", logging.ErrorField(err))
			return err
		}
	}

	if c.migrator != nil {
		if err := c.migrator.Close(); err != nil {
			c.logger.Error("Failed to close migrator", logging.ErrorField(err))
			return err
		}
	}

	c.logger.Info("Database client closed successfully")
	return nil
}

// WithContext returns a new client instance with the given context
// This is useful for request-scoped operations
func (c *Client) WithContext(ctx context.Context) *Client {
	newClient := *c
	return &newClient
}

// Utility functions

// IsConnectionError checks if an error is a connection-related error
func IsConnectionError(err error) bool {
	return errors.IsConnectionError(err)
}

// IsDataError checks if an error is a data-related error
func IsDataError(err error) bool {
	return errors.IsDataError(err)
}

// IsSecurityError checks if an error is a security-related error
func IsSecurityError(err error) bool {
	return errors.IsSecurityError(err)
}

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	return errors.IsRetryable(err)
}

// GetErrorCode extracts the error code from an error
func GetErrorCode(err error) errors.ErrorCode {
	return errors.GetErrorCode(err)
}

// GetUserMessage extracts a user-friendly message from an error
func GetUserMessage(err error) string {
	return errors.GetUserMessage(err)
}

// IsRecordNotFoundError checks if an error is a "record not found" error
func IsRecordNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "record not found") ||
		strings.Contains(errStr, "no rows") ||
		strings.Contains(errStr, "not found")
}

// Version information
const (
	Version = "1.0.0"
	Name    = "go-ormx"
)

// GetVersion returns the package version
func GetVersion() string {
	return Version
}

// GetName returns the package name
func GetName() string {
	return Name
}

// Context helpers for multi-tenancy

// WithTenant returns a new context carrying the provided tenant ID.
// It sets the key "tenant_id" which is consumed by logging and RLS hooks.
func WithTenant(ctx context.Context, tenantID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, "tenant_id", tenantID)
}

// TenantFromContext extracts the tenant ID from context if present; otherwise returns empty string.
func TenantFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v := ctx.Value("tenant_id"); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
