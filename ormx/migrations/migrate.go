// Package migrations provides database migration management using golang-migrate/migrate
package migrations

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"go-ormx/ormx/internal/config"
	"go-ormx/ormx/internal/logging"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"gorm.io/gorm"
)

// Migrator provides database migration management using golang-migrate
type Migrator struct {
	db      *gorm.DB
	logger  logging.Logger
	migrate *migrate.Migrate
	config  *config.Config
}

// MigrationStatus represents the current migration status
type MigrationStatus struct {
	Version      uint `json:"version"`
	Dirty        bool `json:"dirty"`
	AppliedCount int  `json:"applied_count"`
	PendingCount int  `json:"pending_count"`
}

// MigrationConfig holds configuration for migrations
type MigrationConfig struct {
	MigrationsPath string
	DatabaseType   config.DatabaseType
	Timeout        time.Duration
	MaxRetries     int
}

// DefaultMigrationConfig returns default migration configuration
func DefaultMigrationConfig() *MigrationConfig {
	return &MigrationConfig{
		MigrationsPath: "examples/migrations",
		DatabaseType:   config.PostgreSQL,
		Timeout:        5 * time.Minute,
		MaxRetries:     3,
	}
}

// NewMigrator creates a new migrator instance
func NewMigrator(db *gorm.DB, logger logging.Logger, dbConfig *config.Config, migrationConfig *MigrationConfig) (*Migrator, error) {
	if migrationConfig == nil {
		migrationConfig = DefaultMigrationConfig()
	}

	// Get the underlying sql.DB
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Create database driver based on database type
	var driver database.Driver
	switch migrationConfig.DatabaseType {
	case config.PostgreSQL:
		driver, err = postgres.WithInstance(sqlDB, &postgres.Config{})
		if err != nil {
			return nil, fmt.Errorf("failed to create postgres driver: %w", err)
		}
	case config.MySQL:
		driver, err = mysql.WithInstance(sqlDB, &mysql.Config{})
		if err != nil {
			return nil, fmt.Errorf("failed to create mysql driver: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported database type: %s", migrationConfig.DatabaseType)
	}

	// Determine migrations path based on database type
	migrationsPath := filepath.Join(migrationConfig.MigrationsPath, string(migrationConfig.DatabaseType))
	sourceURL := fmt.Sprintf("file://%s", migrationsPath)

	// Create migrate instance
	m, err := migrate.NewWithDatabaseInstance(sourceURL, string(migrationConfig.DatabaseType), driver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return &Migrator{
		db:      db,
		logger:  logger,
		migrate: m,
		config:  dbConfig,
	}, nil
}

// Migrate runs all pending migrations
func (m *Migrator) Migrate(ctx context.Context) error {
	m.logger.Info("Starting database migration")

	// Run migrations
	err := m.migrate.Up()
	if err != nil && err != migrate.ErrNoChange {
		m.logger.Error("Migration failed", logging.ErrorField(err))
		return fmt.Errorf("migration failed: %w", err)
	}

	if err == migrate.ErrNoChange {
		m.logger.Info("No pending migrations found")
		return nil
	}

	m.logger.Info("Database migration completed successfully")
	return nil
}

// MigrateTo runs migrations up to a specific version
func (m *Migrator) MigrateTo(ctx context.Context, version uint) error {
	m.logger.Info("Starting migration to specific version", logging.Int("target_version", int(version)))

	// Run migrations to specific version
	err := m.migrate.Migrate(version)
	if err != nil && err != migrate.ErrNoChange {
		m.logger.Error("Migration failed", logging.ErrorField(err))
		return fmt.Errorf("migration to version %d failed: %w", version, err)
	}

	if err == migrate.ErrNoChange {
		m.logger.Info("No migration needed")
		return nil
	}

	m.logger.Info("Migration to version completed successfully")
	return nil
}

// Rollback rolls back the last migration
func (m *Migrator) Rollback(ctx context.Context) error {
	m.logger.Info("Rolling back last migration")

	// Rollback one step
	if err := m.migrate.Steps(-1); err != nil {
		m.logger.Error("Rollback failed", logging.ErrorField(err))
		return fmt.Errorf("rollback failed: %w", err)
	}

	m.logger.Info("Rollback completed successfully")
	return nil
}

// RollbackTo rolls back to a specific version
func (m *Migrator) RollbackTo(ctx context.Context, version uint) error {
	m.logger.Info("Rolling back to specific version", logging.Int("target_version", int(version)))

	// Get current version
	currentVersion, _, err := m.migrate.Version()
	if err != nil {
		m.logger.Error("Failed to get current version", logging.ErrorField(err))
		return fmt.Errorf("failed to get current version: %w", err)
	}

	// Calculate steps to rollback
	steps := int(currentVersion - version)
	if steps <= 0 {
		m.logger.Info("Already at or below target version")
		return nil
	}

	// Rollback steps
	if err := m.migrate.Steps(-steps); err != nil {
		m.logger.Error("Rollback failed", logging.ErrorField(err))
		return fmt.Errorf("rollback to version %d failed: %w", version, err)
	}

	m.logger.Info("Rollback to version completed successfully")
	return nil
}

// GetStatus returns the current migration status
func (m *Migrator) GetStatus(ctx context.Context) (*MigrationStatus, error) {
	// Get current version and dirty status
	version, dirty, err := m.migrate.Version()
	if err != nil && err != migrate.ErrNilVersion {
		m.logger.Error("Failed to get migration status", logging.ErrorField(err))
		return nil, fmt.Errorf("failed to get migration status: %w", err)
	}

	// For now, we'll set pending to 0 as golang-migrate doesn't provide this directly
	pending := 0

	status := &MigrationStatus{
		Version:      version,
		Dirty:        dirty,
		AppliedCount: int(version),
		PendingCount: pending,
	}

	m.logger.Info("Migration status retrieved successfully")
	return status, nil
}

// Force sets the migration version without running migrations
func (m *Migrator) Force(ctx context.Context, version int) error {
	m.logger.Info("Forcing migration version", logging.Int("version", version))

	// Force version
	if err := m.migrate.Force(version); err != nil {
		m.logger.Error("Force version failed", logging.ErrorField(err))
		return fmt.Errorf("force version failed: %w", err)
	}

	m.logger.Info("Migration version forced successfully")
	return nil
}

// Drop drops all tables (use with caution!)
func (m *Migrator) Drop(ctx context.Context) error {
	m.logger.Warn("Dropping all tables - this is destructive!")

	// Drop all tables
	if err := m.migrate.Drop(); err != nil {
		m.logger.Error("Drop failed", logging.ErrorField(err))
		return fmt.Errorf("drop failed: %w", err)
	}

	m.logger.Info("All tables dropped successfully")
	return nil
}

// Close closes the migrator and releases resources
func (m *Migrator) Close() error {
	if m.migrate != nil {
		if _, err := m.migrate.Close(); err != nil {
			return fmt.Errorf("failed to close migrator: %w", err)
		}
	}
	return nil
}

// GetVersion returns the current migration version
func (m *Migrator) GetVersion(ctx context.Context) (uint, bool, error) {
	version, dirty, err := m.migrate.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return 0, false, fmt.Errorf("failed to get version: %w", err)
	}
	return version, dirty, nil
}

// GetAppliedMigrations returns the list of applied migrations
func (m *Migrator) GetAppliedMigrations(ctx context.Context) ([]uint, error) {
	// golang-migrate doesn't provide a direct way to get all applied migrations
	// This is a simplified implementation
	currentVersion, _, err := m.migrate.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return nil, fmt.Errorf("failed to get current version: %w", err)
	}

	var applied []uint
	for i := uint(1); i <= currentVersion; i++ {
		applied = append(applied, i)
	}

	return applied, nil
}
