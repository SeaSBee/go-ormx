// Package db provides database connection management, health checks, and graceful shutdown
// functionality for GORM-based data access layer.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"go-ormx/ormx/errors"
	"go-ormx/ormx/internal/config"
	"go-ormx/ormx/internal/logging"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// Database represents the database connection and its associated components
type Database struct {
	db     *gorm.DB
	sqlDB  *sql.DB
	config *config.Config
	logger logging.Logger
	mu     sync.RWMutex
	closed bool

	// Health check components
	healthChecker *HealthChecker

	// Connection monitoring
	connectionMonitor *ConnectionMonitor

	// Graceful shutdown
	shutdownCh chan struct{}
	shutdownWg sync.WaitGroup
}

// New creates a new database instance with the given configuration
func New(cfg *config.Config, logger logging.Logger) (*Database, error) {
	if cfg == nil {
		return nil, errors.NewDBError(errors.ErrCodeMissingConfig, "database configuration is required", nil)
	}

	if err := cfg.Validate(); err != nil {
		return nil, errors.WrapError(err, errors.ErrCodeConfigValidation, "invalid database configuration")
	}

	if logger == nil {
		return nil, errors.NewDBError(errors.ErrCodeMissingConfig, "logger is required", nil)
	}

	db := &Database{
		config:     cfg,
		logger:     logger,
		shutdownCh: make(chan struct{}),
	}

	if err := db.connect(); err != nil {
		return nil, err
	}

	// Initialize health checker
	db.healthChecker = NewHealthChecker(db, logger)

	// Initialize connection monitor
	db.connectionMonitor = NewConnectionMonitor(db, logger)

	// Start background tasks
	db.startBackgroundTasks()

	return db, nil
}

// connect establishes the database connection
func (d *Database) connect() error {
	gormConfig := &gorm.Config{
		Logger: logging.NewDBLogger(d.logger, logging.DefaultLoggerConfig()),
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "",    // No table prefix by default
			SingularTable: false, // Use plural table names
		},
		PrepareStmt:                              d.config.PreparedStatements,
		DisableForeignKeyConstraintWhenMigrating: d.config.DisableForeignKeys,
		SkipDefaultTransaction:                   d.config.SkipDefaultTx,
	}

	var dialector gorm.Dialector
	var err error

	switch d.config.Type {
	case config.PostgreSQL:
		dialector, err = d.createPostgresDialector()
	case config.MySQL:
		dialector, err = d.createMySQLDialector()
	default:
		return errors.NewDBError(errors.ErrCodeInvalidConfig,
			fmt.Sprintf("unsupported database type: %s", d.config.Type), nil)
	}

	if err != nil {
		return errors.WrapError(err, errors.ErrCodeConnectionFailed, "failed to create database dialector")
	}

	// Open database connection
	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return errors.WrapGormError(err, "connect")
	}

	// Get underlying SQL DB for connection pool configuration
	sqlDB, err := db.DB()
	if err != nil {
		return errors.WrapError(err, errors.ErrCodeConnectionFailed, "failed to get underlying SQL DB")
	}

	// Configure connection pool
	if err := d.configureConnectionPool(sqlDB); err != nil {
		return errors.WrapError(err, errors.ErrCodeConnectionFailed, "failed to configure connection pool")
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), d.config.ConnectTimeout)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return errors.WrapError(err, errors.ErrCodeConnectionFailed, "database ping failed")
	}

	// If RLS is enabled and PostgreSQL, set session GUC hook via GORM callbacks
	if d.config.RLSEnabled && d.config.Type == config.PostgreSQL {
		// Register a callback to set the app.tenant_id GUC from context if present
		d.registerRLSCallbacks(db)
		d.logger.Info("RLS enabled; callbacks registered",
			logging.String("tenant_guc", d.config.RLSTenantGUC),
			logging.String("tenant_column", d.config.RLSTenantColumn),
		)
	}

	d.db = db
	d.sqlDB = sqlDB

	d.logger.Info("Database connection established successfully",
		logging.String("database_type", string(d.config.Type)),
		logging.String("host", d.config.Host),
		logging.Int("port", d.config.Port),
		logging.String("database", d.config.Database),
	)

	return nil
}

// registerRLSCallbacks registers GORM callbacks to set PostgreSQL RLS GUC per request
func (d *Database) registerRLSCallbacks(gormDB *gorm.DB) {
	// Before each query, set the GUC based on context tenant_id
	gormDB.Callback().Query().Before("gorm:query").Register("rls:set_tenant_guc_before_query", func(db *gorm.DB) {
		d.setTenantGUCFromContext(db)
	})
	gormDB.Callback().Row().Before("gorm:row").Register("rls:set_tenant_guc_before_row", func(db *gorm.DB) {
		d.setTenantGUCFromContext(db)
	})
	gormDB.Callback().Create().Before("gorm:create").Register("rls:set_tenant_guc_before_create", func(db *gorm.DB) {
		d.setTenantGUCFromContext(db)
	})
	gormDB.Callback().Update().Before("gorm:update").Register("rls:set_tenant_guc_before_update", func(db *gorm.DB) {
		d.setTenantGUCFromContext(db)
	})
	gormDB.Callback().Delete().Before("gorm:delete").Register("rls:set_tenant_guc_before_delete", func(db *gorm.DB) {
		d.setTenantGUCFromContext(db)
	})
}

// setTenantGUCFromContext sets the configured GUC (e.g., app.tenant_id) using the tenant ID from context if present
func (d *Database) setTenantGUCFromContext(db *gorm.DB) {
	if db == nil || db.Statement == nil || db.Statement.Context == nil {
		return
	}
	ctx := db.Statement.Context
	val := ctx.Value("tenant_id")
	tenantID, _ := val.(string)
	if strings.TrimSpace(tenantID) == "" {
		if d.config.RLSRequireTenant {
			d.logger.Warn("RLS requires tenant but no tenant_id present in context")
		}
		return
	}
	// Only for PostgreSQL
	if d.config.Type != config.PostgreSQL {
		return
	}
	guc := d.config.RLSTenantGUC
	if strings.TrimSpace(guc) == "" {
		guc = "app.tenant_id"
	}
	// Set the GUC on the session
	if err := db.Exec("select set_config(?, ?, true)", guc, tenantID).Error; err != nil {
		d.logger.Warn("Failed to set RLS tenant GUC",
			logging.String("guc", guc),
			logging.String("tenant_id", tenantID),
			logging.ErrorField(err),
		)
	}
}

// createPostgresDialector creates a PostgreSQL dialector
func (d *Database) createPostgresDialector() (gorm.Dialector, error) {
	dsn := d.config.GetDSN()
	if dsn == "" {
		return nil, errors.NewDBError(errors.ErrCodeInvalidConfig, "failed to generate PostgreSQL DSN", nil)
	}

	return postgres.Open(dsn), nil
}

// createMySQLDialector creates a MySQL dialector
func (d *Database) createMySQLDialector() (gorm.Dialector, error) {
	dsn := d.config.GetDSN()
	if dsn == "" {
		return nil, errors.NewDBError(errors.ErrCodeInvalidConfig, "failed to generate MySQL DSN", nil)
	}

	return mysql.Open(dsn), nil
}

// configureConnectionPool configures the database connection pool
func (d *Database) configureConnectionPool(sqlDB *sql.DB) error {
	// Set maximum number of open connections
	sqlDB.SetMaxOpenConns(d.config.MaxOpenConns)

	// Set maximum number of idle connections
	sqlDB.SetMaxIdleConns(d.config.MaxIdleConns)

	// Set maximum lifetime of connections
	sqlDB.SetConnMaxLifetime(d.config.ConnMaxLifetime)

	// Set maximum idle time of connections
	sqlDB.SetConnMaxIdleTime(d.config.ConnMaxIdleTime)

	d.logger.Info("Database connection pool configured",
		logging.Int("max_open_conns", d.config.MaxOpenConns),
		logging.Int("max_idle_conns", d.config.MaxIdleConns),
		logging.Duration("conn_max_lifetime", d.config.ConnMaxLifetime),
		logging.Duration("conn_max_idle_time", d.config.ConnMaxIdleTime),
	)

	return nil
}

// startBackgroundTasks starts background monitoring and maintenance tasks
func (d *Database) startBackgroundTasks() {
	// Start health checker
	d.shutdownWg.Add(1)
	go func() {
		defer d.shutdownWg.Done()
		d.healthChecker.Start(d.shutdownCh)
	}()

	// Start connection monitor
	d.shutdownWg.Add(1)
	go func() {
		defer d.shutdownWg.Done()
		d.connectionMonitor.Start(d.shutdownCh)
	}()
}

// DB returns the GORM database instance
func (d *Database) DB() *gorm.DB {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.db
}

// SqlDB returns the underlying SQL database instance
func (d *Database) SqlDB() *sql.DB {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.sqlDB
}

// Config returns the database configuration
func (d *Database) Config() *config.Config {
	return d.config
}

// WithContext returns a new GORM DB instance with the given context
func (d *Database) WithContext(ctx context.Context) *gorm.DB {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.closed {
		return nil
	}

	return d.db.WithContext(ctx)
}

// Transaction executes a function within a database transaction
func (d *Database) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.closed {
		return errors.NewDBError(errors.ErrCodeConnectionFailed, "database is closed", nil)
	}

	// Create a context with timeout for the transaction
	txCtx, cancel := context.WithTimeout(ctx, d.config.QueryTimeout)
	defer cancel()

	return d.db.WithContext(txCtx).Transaction(fn)
}

// TransactionWithOptions executes a function within a database transaction with options
func (d *Database) TransactionWithOptions(ctx context.Context, opts *sql.TxOptions, fn func(tx *gorm.DB) error) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.closed {
		return errors.NewDBError(errors.ErrCodeConnectionFailed, "database is closed", nil)
	}

	// Create a context with timeout for the transaction
	txCtx, cancel := context.WithTimeout(ctx, d.config.QueryTimeout)
	defer cancel()

	return d.db.WithContext(txCtx).Transaction(fn, opts)
}

// Ping tests the database connection
func (d *Database) Ping(ctx context.Context) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.closed {
		return errors.NewDBError(errors.ErrCodeConnectionFailed, "database is closed", nil)
	}

	if d.sqlDB == nil {
		return errors.NewDBError(errors.ErrCodeConnectionFailed, "database connection not initialized", nil)
	}

	if err := d.sqlDB.PingContext(ctx); err != nil {
		return errors.WrapError(err, errors.ErrCodeConnectionFailed, "database ping failed")
	}

	return nil
}

// IsHealthy checks if the database is healthy
func (d *Database) IsHealthy(ctx context.Context) bool {
	return d.healthChecker.IsHealthy(ctx)
}

// GetConnectionStats returns database connection statistics
func (d *Database) GetConnectionStats() ConnectionStats {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.closed || d.sqlDB == nil {
		return ConnectionStats{}
	}

	stats := d.sqlDB.Stats()
	return ConnectionStats{
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUseConnections:   stats.InUse,
		IdleConnections:    stats.Idle,
		WaitCount:          stats.WaitCount,
		WaitDuration:       stats.WaitDuration,
		MaxIdleClosed:      stats.MaxIdleClosed,
		MaxIdleTimeClosed:  stats.MaxIdleTimeClosed,
		MaxLifetimeClosed:  stats.MaxLifetimeClosed,
	}
}

// Close gracefully closes the database connection
func (d *Database) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return nil
	}

	d.logger.Info("Starting database shutdown...")

	// Signal background tasks to stop
	close(d.shutdownCh)

	// Wait for background tasks to complete
	d.shutdownWg.Wait()

	// Close the database connection
	if d.sqlDB != nil {
		if err := d.sqlDB.Close(); err != nil {
			d.logger.Error("Failed to close database connection", logging.ErrorField(err))
			return errors.WrapError(err, errors.ErrCodeConnectionFailed, "failed to close database connection")
		}
	}

	d.closed = true
	d.logger.Info("Database shutdown completed successfully")

	return nil
}

// IsClosed returns true if the database is closed
func (d *Database) IsClosed() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.closed
}

// ConnectionStats represents database connection statistics
type ConnectionStats struct {
	MaxOpenConnections int           `json:"max_open_connections"`
	OpenConnections    int           `json:"open_connections"`
	InUseConnections   int           `json:"in_use_connections"`
	IdleConnections    int           `json:"idle_connections"`
	WaitCount          int64         `json:"wait_count"`
	WaitDuration       time.Duration `json:"wait_duration"`
	MaxIdleClosed      int64         `json:"max_idle_closed"`
	MaxIdleTimeClosed  int64         `json:"max_idle_time_closed"`
	MaxLifetimeClosed  int64         `json:"max_lifetime_closed"`
}

// DatabaseManager manages multiple database instances
type DatabaseManager struct {
	databases map[string]*Database
	mu        sync.RWMutex
	logger    logging.Logger
}

// NewDatabaseManager creates a new database manager
func NewDatabaseManager(logger logging.Logger) *DatabaseManager {
	return &DatabaseManager{
		databases: make(map[string]*Database),
		logger:    logger,
	}
}

// Add adds a database instance with a given name
func (dm *DatabaseManager) Add(name string, db *Database) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.databases[name] = db
}

// Get retrieves a database instance by name
func (dm *DatabaseManager) Get(name string) (*Database, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	db, exists := dm.databases[name]
	return db, exists
}

// Remove removes a database instance by name
func (dm *DatabaseManager) Remove(name string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	if db, exists := dm.databases[name]; exists {
		db.Close()
		delete(dm.databases, name)
	}
}

// List returns all database names
func (dm *DatabaseManager) List() []string {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	names := make([]string, 0, len(dm.databases))
	for name := range dm.databases {
		names = append(names, name)
	}
	return names
}

// CloseAll closes all database instances
func (dm *DatabaseManager) CloseAll() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	var lastErr error
	for name, db := range dm.databases {
		if err := db.Close(); err != nil {
			dm.logger.Error("Failed to close database",
				logging.String("database_name", name),
				logging.ErrorField(err))
			lastErr = err
		}
	}

	// Clear the map
	dm.databases = make(map[string]*Database)

	return lastErr
}

// HealthStatus returns the health status of all databases
func (dm *DatabaseManager) HealthStatus(ctx context.Context) map[string]bool {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	status := make(map[string]bool)
	for name, db := range dm.databases {
		status[name] = db.IsHealthy(ctx)
	}
	return status
}
