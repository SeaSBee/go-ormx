package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/seasbee/go-ormx/pkg/config"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ConnectionManager manages database connections with pooling and health checks
type ConnectionManager struct {
	config     *config.DatabaseConfig
	primaryDB  *gorm.DB
	readDBs    []*gorm.DB
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	healthChan chan HealthCheckResult
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	DB      *gorm.DB
	Healthy bool
	Error   error
	Time    time.Time
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(cfg *config.DatabaseConfig) (*ConnectionManager, error) {
	if cfg == nil {
		return nil, errors.New("database config cannot be nil")
	}

	if err := cfg.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid database config")
	}

	ctx, cancel := context.WithCancel(context.Background())

	cm := &ConnectionManager{
		config:     cfg,
		ctx:        ctx,
		cancel:     cancel,
		healthChan: make(chan HealthCheckResult, 100),
	}

	// Initialize primary connection
	if err := cm.initializePrimaryConnection(); err != nil {
		cancel()
		return nil, errors.Wrap(err, "failed to initialize primary connection")
	}

	// Initialize read replicas
	if err := cm.initializeReadReplicas(); err != nil {
		cancel()
		return nil, errors.Wrap(err, "failed to initialize read replicas")
	}

	// Start health check goroutine
	go cm.startHealthChecks()

	return cm, nil
}

// initializePrimaryConnection sets up the primary database connection
func (cm *ConnectionManager) initializePrimaryConnection() error {
	db, err := cm.createConnection(*cm.config)
	if err != nil {
		return errors.Wrap(err, "failed to create primary connection")
	}

	// Configure connection pool using GORM's underlying sql.DB
	sqlDB, err := db.DB()
	if err != nil {
		return errors.Wrap(err, "failed to get underlying sql.DB")
	}

	// Set connection pool settings
	sqlDB.SetMaxOpenConns(cm.config.MaxConnections)
	sqlDB.SetMaxIdleConns(cm.config.MaxIdleConnections)
	sqlDB.SetConnMaxLifetime(cm.config.MaxLifetime)
	sqlDB.SetConnMaxIdleTime(cm.config.IdleTimeout)

	cm.primaryDB = db
	return nil
}

// initializeReadReplicas sets up read replica connections
func (cm *ConnectionManager) initializeReadReplicas() error {
	// For now, we'll skip read replicas since they're not in the current config
	// This can be extended later when read replica support is added
	return nil
}

// createConnection creates a GORM connection based on the driver type
func (cm *ConnectionManager) createConnection(connConfig config.DatabaseConfig) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch connConfig.Driver {
	case "postgres":
		dsn := connConfig.ConnectionString()
		dialector = postgres.Open(dsn)
	case "mysql":
		dsn := connConfig.ConnectionString()
		dialector = mysql.Open(dsn)
	case "sqlite":
		dsn := connConfig.ConnectionString()
		dialector = sqlite.Open(dsn)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", connConfig.Driver)
	}

	// Configure GORM logger
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	// Add timeout configuration
	if connConfig.ConnectionTimeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), connConfig.ConnectionTimeout)
		defer cancel()
		db, err := gorm.Open(dialector, gormConfig)
		if err != nil {
			return nil, err
		}
		return db.WithContext(ctx), nil
	}

	return gorm.Open(dialector, gormConfig)
}

// GetPrimaryDB returns the primary database connection
func (cm *ConnectionManager) GetPrimaryDB() *gorm.DB {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.primaryDB
}

// GetReadDB returns a read replica database connection (round-robin)
func (cm *ConnectionManager) GetReadDB() *gorm.DB {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if len(cm.readDBs) == 0 {
		return cm.primaryDB
	}

	// Simple round-robin selection
	index := time.Now().UnixNano() % int64(len(cm.readDBs))
	return cm.readDBs[index]
}

// GetAllReadDBs returns all read replica connections
func (cm *ConnectionManager) GetAllReadDBs() []*gorm.DB {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	result := make([]*gorm.DB, len(cm.readDBs))
	copy(result, cm.readDBs)
	return result
}

// startHealthChecks starts the health check monitoring goroutine
func (cm *ConnectionManager) startHealthChecks() {
	ticker := time.NewTicker(cm.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cm.ctx.Done():
			return
		case <-ticker.C:
			cm.performHealthChecks()
		}
	}
}

// performHealthChecks performs health checks on all connections
func (cm *ConnectionManager) performHealthChecks() {
	// Check primary connection
	if cm.primaryDB != nil {
		result := cm.checkConnectionHealth(cm.primaryDB, "primary")
		select {
		case cm.healthChan <- result:
		default:
			// Channel is full, skip this result
		}
	}

	// Check read replicas
	for i, readDB := range cm.readDBs {
		if readDB != nil {
			result := cm.checkConnectionHealth(readDB, fmt.Sprintf("read-replica-%d", i))
			select {
			case cm.healthChan <- result:
			default:
				// Channel is full, skip this result
			}
		}
	}
}

// checkConnectionHealth performs a health check on a single connection
func (cm *ConnectionManager) checkConnectionHealth(db *gorm.DB, name string) HealthCheckResult {
	ctx, cancel := context.WithTimeout(context.Background(), cm.config.HealthCheck.Timeout)
	defer cancel()

	result := HealthCheckResult{
		DB:   db,
		Time: time.Now(),
	}

	// Simple ping test
	sqlDB, err := db.DB()
	if err != nil {
		result.Healthy = false
		result.Error = errors.Wrap(err, "failed to get underlying sql.DB")
		return result
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		result.Healthy = false
		result.Error = errors.Wrap(err, "ping failed")
		return result
	}

	result.Healthy = true
	return result
}

// GetHealthResults returns the health check results channel
func (cm *ConnectionManager) GetHealthResults() <-chan HealthCheckResult {
	return cm.healthChan
}

// Close closes all database connections
func (cm *ConnectionManager) Close() error {
	cm.cancel()

	var errs []error

	// Close primary connection
	if cm.primaryDB != nil {
		if sqlDB, err := cm.primaryDB.DB(); err == nil {
			if err := sqlDB.Close(); err != nil {
				errs = append(errs, errors.Wrap(err, "failed to close primary connection"))
			}
		}
	}

	// Close read replica connections
	for i, readDB := range cm.readDBs {
		if readDB != nil {
			if sqlDB, err := readDB.DB(); err == nil {
				if err := sqlDB.Close(); err != nil {
					errs = append(errs, errors.Wrapf(err, "failed to close read replica %d", i))
				}
			}
		}
	}

	// Close health channel
	close(cm.healthChan)

	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}

	return nil
}

// IsHealthy returns true if all connections are healthy
func (cm *ConnectionManager) IsHealthy() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Check primary connection
	if cm.primaryDB != nil {
		if sqlDB, err := cm.primaryDB.DB(); err == nil {
			if err := sqlDB.Ping(); err != nil {
				return false
			}
		} else {
			return false
		}
	}

	// Check read replicas
	for _, readDB := range cm.readDBs {
		if readDB != nil {
			if sqlDB, err := readDB.DB(); err == nil {
				if err := sqlDB.Ping(); err != nil {
					return false
				}
			} else {
				return false
			}
		}
	}

	return true
}

// GetStats returns connection statistics
func (cm *ConnectionManager) GetStats() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	stats := make(map[string]interface{})

	// Primary connection stats
	if cm.primaryDB != nil {
		if sqlDB, err := cm.primaryDB.DB(); err == nil {
			stats["primary"] = map[string]interface{}{
				"max_open_connections": sqlDB.Stats().MaxOpenConnections,
				"open_connections":     sqlDB.Stats().OpenConnections,
				"in_use":               sqlDB.Stats().InUse,
				"idle":                 sqlDB.Stats().Idle,
			}
		}
	}

	// Read replica stats
	readStats := make([]map[string]interface{}, len(cm.readDBs))
	for i, readDB := range cm.readDBs {
		if readDB != nil {
			if sqlDB, err := readDB.DB(); err == nil {
				readStats[i] = map[string]interface{}{
					"max_open_connections": sqlDB.Stats().MaxOpenConnections,
					"open_connections":     sqlDB.Stats().OpenConnections,
					"in_use":               sqlDB.Stats().InUse,
					"idle":                 sqlDB.Stats().Idle,
				}
			}
		}
	}
	stats["read_replicas"] = readStats

	return stats
}
