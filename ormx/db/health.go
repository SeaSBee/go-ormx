// Package db provides health checking functionality for database connections
package db

import (
	"context"
	"sync"
	"time"

	"go-ormx/ormx/internal/logging"
)

// HealthStatus represents the health status of the database
type HealthStatus struct {
	Healthy          bool          `json:"healthy"`
	LastCheck        time.Time     `json:"last_check"`
	ResponseTime     time.Duration `json:"response_time"`
	Error            string        `json:"error,omitempty"`
	ConsecutiveFails int           `json:"consecutive_fails"`
}

// HealthChecker monitors database health
type HealthChecker struct {
	db     *Database
	logger logging.Logger

	// Health check configuration
	interval            time.Duration
	timeout             time.Duration
	maxConsecutiveFails int

	// Current health status
	mu     sync.RWMutex
	status HealthStatus
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(db *Database, logger logging.Logger) *HealthChecker {
	return &HealthChecker{
		db:                  db,
		logger:              logger,
		interval:            30 * time.Second, // Check every 30 seconds
		timeout:             5 * time.Second,  // 5 second timeout for health checks
		maxConsecutiveFails: 3,                // Mark unhealthy after 3 consecutive failures
		status: HealthStatus{
			Healthy:   true,
			LastCheck: time.Now(),
		},
	}
}

// Start starts the health checker
func (hc *HealthChecker) Start(shutdownCh <-chan struct{}) {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	hc.logger.Info("Health checker started",
		logging.Duration("interval", hc.interval),
		logging.Duration("timeout", hc.timeout),
		logging.Int("max_consecutive_fails", hc.maxConsecutiveFails),
	)

	// Perform initial health check
	hc.performHealthCheck()

	for {
		select {
		case <-ticker.C:
			hc.performHealthCheck()
		case <-shutdownCh:
			hc.logger.Info("Health checker shutting down")
			return
		}
	}
}

// performHealthCheck performs a health check on the database
func (hc *HealthChecker) performHealthCheck() {
	start := time.Now()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), hc.timeout)
	defer cancel()

	// Perform ping
	err := hc.db.Ping(ctx)
	responseTime := time.Since(start)

	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.status.LastCheck = time.Now()
	hc.status.ResponseTime = responseTime

	if err != nil {
		hc.status.ConsecutiveFails++
		hc.status.Error = err.Error()

		// Mark as unhealthy if we've exceeded the consecutive failure threshold
		if hc.status.ConsecutiveFails >= hc.maxConsecutiveFails {
			if hc.status.Healthy {
				hc.logger.Error("Database marked as unhealthy",
					logging.Int("consecutive_fails", hc.status.ConsecutiveFails),
					logging.ErrorField(err),
				)
			}
			hc.status.Healthy = false
		}

		hc.logger.Warn("Database health check failed",
			logging.Int("consecutive_fails", hc.status.ConsecutiveFails),
			logging.Duration("response_time", responseTime),
			logging.ErrorField(err),
		)
	} else {
		// Reset failure count on successful check
		if hc.status.ConsecutiveFails > 0 || !hc.status.Healthy {
			hc.logger.Info("Database health restored",
				logging.Int("previous_consecutive_fails", hc.status.ConsecutiveFails),
				logging.Duration("response_time", responseTime),
			)
		}

		hc.status.Healthy = true
		hc.status.ConsecutiveFails = 0
		hc.status.Error = ""

		hc.logger.Debug("Database health check passed",
			logging.Duration("response_time", responseTime),
		)
	}
}

// IsHealthy returns the current health status
func (hc *HealthChecker) IsHealthy(ctx context.Context) bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.status.Healthy
}

// GetStatus returns the detailed health status
func (hc *HealthChecker) GetStatus() HealthStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.status
}

// SetInterval sets the health check interval
func (hc *HealthChecker) SetInterval(interval time.Duration) {
	hc.interval = interval
}

// SetTimeout sets the health check timeout
func (hc *HealthChecker) SetTimeout(timeout time.Duration) {
	hc.timeout = timeout
}

// SetMaxConsecutiveFails sets the maximum consecutive failures before marking unhealthy
func (hc *HealthChecker) SetMaxConsecutiveFails(maxFails int) {
	hc.maxConsecutiveFails = maxFails
}

// ConnectionMonitor monitors database connection statistics
type ConnectionMonitor struct {
	db     *Database
	logger logging.Logger

	// Monitor configuration
	interval time.Duration

	// Connection stats tracking
	mu    sync.RWMutex
	stats ConnectionStats

	// Alerting thresholds
	maxConnectionUsagePercent float64
	maxWaitDuration           time.Duration
}

// NewConnectionMonitor creates a new connection monitor
func NewConnectionMonitor(db *Database, logger logging.Logger) *ConnectionMonitor {
	return &ConnectionMonitor{
		db:                        db,
		logger:                    logger,
		interval:                  60 * time.Second, // Monitor every minute
		maxConnectionUsagePercent: 80.0,             // Alert when 80% of connections are in use
		maxWaitDuration:           1 * time.Second,  // Alert when average wait > 1 second
	}
}

// Start starts the connection monitor
func (cm *ConnectionMonitor) Start(shutdownCh <-chan struct{}) {
	ticker := time.NewTicker(cm.interval)
	defer ticker.Stop()

	cm.logger.Info("Connection monitor started",
		logging.Duration("interval", cm.interval),
		logging.Float64("max_connection_usage_percent", cm.maxConnectionUsagePercent),
		logging.Duration("max_wait_duration", cm.maxWaitDuration),
	)

	for {
		select {
		case <-ticker.C:
			cm.monitorConnections()
		case <-shutdownCh:
			cm.logger.Info("Connection monitor shutting down")
			return
		}
	}
}

// monitorConnections monitors connection statistics
func (cm *ConnectionMonitor) monitorConnections() {
	stats := cm.db.GetConnectionStats()

	cm.mu.Lock()
	cm.stats = stats
	cm.mu.Unlock()

	// Calculate usage percentage
	usagePercent := float64(stats.InUseConnections) / float64(stats.MaxOpenConnections) * 100

	// Log current statistics
	cm.logger.Debug("Connection statistics",
		logging.Int("max_open", stats.MaxOpenConnections),
		logging.Int("open", stats.OpenConnections),
		logging.Int("in_use", stats.InUseConnections),
		logging.Int("idle", stats.IdleConnections),
		logging.Float64("usage_percent", usagePercent),
		logging.Int64("wait_count", stats.WaitCount),
		logging.Duration("wait_duration", stats.WaitDuration),
	)

	// Check for high connection usage
	if usagePercent > cm.maxConnectionUsagePercent {
		cm.logger.Warn("High database connection usage detected",
			logging.Float64("usage_percent", usagePercent),
			logging.Float64("threshold_percent", cm.maxConnectionUsagePercent),
			logging.Int("in_use_connections", stats.InUseConnections),
			logging.Int("max_connections", stats.MaxOpenConnections),
		)
	}

	// Check for high wait times
	if stats.WaitCount > 0 {
		avgWaitDuration := stats.WaitDuration / time.Duration(stats.WaitCount)
		if avgWaitDuration > cm.maxWaitDuration {
			cm.logger.Warn("High database connection wait time detected",
				logging.Duration("avg_wait_duration", avgWaitDuration),
				logging.Duration("threshold_duration", cm.maxWaitDuration),
				logging.Int64("wait_count", stats.WaitCount),
				logging.Duration("total_wait_duration", stats.WaitDuration),
			)
		}
	}

	// Check for connection leaks (connections that are closed due to max lifetime)
	if stats.MaxLifetimeClosed > 0 {
		cm.logger.Info("Connections closed due to max lifetime",
			logging.Int64("closed_count", stats.MaxLifetimeClosed),
		)
	}

	// Check for idle connection cleanup
	if stats.MaxIdleClosed > 0 {
		cm.logger.Debug("Idle connections cleaned up",
			logging.Int64("closed_count", stats.MaxIdleClosed),
		)
	}

	if stats.MaxIdleTimeClosed > 0 {
		cm.logger.Debug("Connections closed due to max idle time",
			logging.Int64("closed_count", stats.MaxIdleTimeClosed),
		)
	}
}

// GetStats returns the current connection statistics
func (cm *ConnectionMonitor) GetStats() ConnectionStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.stats
}

// SetInterval sets the monitoring interval
func (cm *ConnectionMonitor) SetInterval(interval time.Duration) {
	cm.interval = interval
}

// SetMaxConnectionUsagePercent sets the threshold for connection usage alerts
func (cm *ConnectionMonitor) SetMaxConnectionUsagePercent(percent float64) {
	cm.maxConnectionUsagePercent = percent
}

// SetMaxWaitDuration sets the threshold for wait duration alerts
func (cm *ConnectionMonitor) SetMaxWaitDuration(duration time.Duration) {
	cm.maxWaitDuration = duration
}
