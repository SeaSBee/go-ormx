package config

import (
	"fmt"
	"time"
)

// DatabaseConfig represents comprehensive database configuration with connection pooling and timeout support
type DatabaseConfig struct {
	// Basic connection settings
	Driver   string `yaml:"driver" json:"driver" validate:"required,oneof=postgres mysql sqlite sqlserver,max=20"`
	Host     string `yaml:"host" json:"host" validate:"required,hostname,max=255"`
	Port     int    `yaml:"port" json:"port" validate:"required,min=1,max=65535"`
	Database string `yaml:"database" json:"database" validate:"required,min=1,max=64"`
	Username string `yaml:"username" json:"username" validate:"required,min=1,max=32"`
	Password string `yaml:"password" json:"password" validate:"required,min=1,max=128"`
	SSLMode  string `yaml:"ssl_mode" json:"ssl_mode" validate:"oneof=disable require verify-ca verify-full,max=20"`

	// Connection Pool Configuration
	MaxConnections     int           `yaml:"max_connections" json:"max_connections" validate:"required,min=1,max=10000"`
	MinConnections     int           `yaml:"min_connections" json:"min_connections" validate:"required,min=0,max=1000"`
	MaxIdleConnections int           `yaml:"max_idle_connections" json:"max_idle_connections" validate:"required,min=0,max=1000"`
	MaxLifetime        time.Duration `yaml:"max_lifetime" json:"max_lifetime" validate:"required,min=1m,max=12h"`
	IdleTimeout        time.Duration `yaml:"idle_timeout" json:"idle_timeout" validate:"required,min=30s,max=1h"`
	AcquireTimeout     time.Duration `yaml:"acquire_timeout" json:"acquire_timeout" validate:"required,min=1s,max=30s"`
	LeakDetection      bool          `yaml:"leak_detection" json:"leak_detection"`
	LeakTimeout        time.Duration `yaml:"leak_timeout" json:"leak_timeout" validate:"omitempty,min=1s,max=5m"`

	// Timeout Configuration
	ConnectionTimeout  time.Duration `yaml:"connection_timeout" json:"connection_timeout" validate:"required,min=1s,max=30s"`
	QueryTimeout       time.Duration `yaml:"query_timeout" json:"query_timeout" validate:"required,min=100ms,max=5m"`
	TransactionTimeout time.Duration `yaml:"transaction_timeout" json:"transaction_timeout" validate:"required,min=1s,max=10m"`
	StatementTimeout   time.Duration `yaml:"statement_timeout" json:"statement_timeout" validate:"required,min=100ms,max=1m"`
	CancelTimeout      time.Duration `yaml:"cancel_timeout" json:"cancel_timeout" validate:"required,min=100ms,max=30s"`

	// Retry Configuration
	Retry RetryConfig `yaml:"retry" json:"retry" validate:"required"`

	// Health Check Configuration
	HealthCheck HealthCheckConfig `yaml:"health_check" json:"health_check" validate:"required"`

	// Logging Configuration
	LogLevel  string            `yaml:"log_level" json:"log_level" validate:"required,oneof=debug info warn error fatal,max=10"`
	LogFormat string            `yaml:"log_format" json:"log_format" validate:"required,oneof=json text,max=1000"`
	LogFields map[string]string `yaml:"log_fields" json:"log_fields" validate:"omitempty,max=500"`

	// Observability Configuration
	Metrics             bool          `yaml:"metrics" json:"metrics"`
	Tracing             bool          `yaml:"tracing" json:"tracing"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval" json:"health_check_interval" validate:"required,min=5s,max=5m"`

	// Security Configuration
	MaxQuerySize       int  `yaml:"max_query_size" json:"max_query_size" validate:"required,min=1024,max=1048576"`
	MaxResultSize      int  `yaml:"max_result_size" json:"max_result_size" validate:"required,min=1000,max=1000000"`
	EnableQueryLogging bool `yaml:"enable_query_logging" json:"enable_query_logging" validate:"required,oneof=true false,default=true"`
	MaskSensitiveData  bool `yaml:"mask_sensitive_data" json:"mask_sensitive_data" validate:"required,oneof=true false,default=true"`

	// Pagination Configuration
	Pagination *PaginationConfig `yaml:"pagination" json:"pagination" validate:"required"`

	// Encryption Configuration
	Encryption *EncryptionConfig `yaml:"encryption" json:"encryption" validate:"omitempty"`

	// Read Replicas Configuration
	ReadReplicas []ReadReplicaConfig `yaml:"read_replicas" json:"read_replicas" validate:"omitempty,dive,max=10"`
}

// RetryConfig represents retry configuration
type RetryConfig struct {
	Enabled            bool          `yaml:"enabled" json:"enabled"`
	MaxAttempts        int           `yaml:"max_attempts" json:"max_attempts" validate:"required_if=Enabled true,min=1,max=10"`
	InitialDelay       time.Duration `yaml:"initial_delay" json:"initial_delay" validate:"required_if=Enabled true,min=100ms,max=30s"`
	MaxDelay           time.Duration `yaml:"max_delay" json:"max_delay" validate:"required_if=Enabled true,min=1s,max=5m"`
	BackoffMultiplier  float64       `yaml:"backoff_multiplier" json:"backoff_multiplier" validate:"required_if=Enabled true,min=1.0,max=5.0"`
	Jitter             bool          `yaml:"jitter" json:"jitter"`
	RetryableErrors    []string      `yaml:"retryable_errors" json:"retryable_errors" validate:"omitempty,dive,min=1,max=50,max=20"`
	NonRetryableErrors []string      `yaml:"non_retryable_errors" json:"non_retryable_errors" validate:"omitempty,dive,min=1,max=50,max=20"`
}

// HealthCheckConfig represents health check configuration
type HealthCheckConfig struct {
	Enabled      bool          `yaml:"enabled" json:"enabled"`
	Interval     time.Duration `yaml:"interval" json:"interval" validate:"required_if=Enabled true,min=5s,max=5m"`
	Timeout      time.Duration `yaml:"timeout" json:"timeout" validate:"required_if=Enabled true,min=1s,max=30s"`
	Query        string        `yaml:"query" json:"query" validate:"required_if=Enabled true,min=1,max=1000"`
	MaxFailures  int           `yaml:"max_failures" json:"max_failures" validate:"required_if=Enabled true,min=1,max=10"`
	RecoveryTime time.Duration `yaml:"recovery_time" json:"recovery_time" validate:"required_if=Enabled true,min=1s,max=10m"`
}

// EncryptionConfig represents encryption configuration
type EncryptionConfig struct {
	// Global encryption settings
	Enabled   bool   `yaml:"enabled" json:"enabled"`
	Algorithm string `yaml:"algorithm" json:"algorithm" validate:"required_if=Enabled true,oneof=aes-256-gcm aes-256-cbc chacha20-poly1305,max=32"`
}

// ReadReplicaConfig represents read replica configuration
type ReadReplicaConfig struct {
	// Basic connection settings for read replica
	Host     string `yaml:"host" json:"host" validate:"required,hostname,max=255"`
	Port     int    `yaml:"port" json:"port" validate:"required,min=1,max=65535"`
	Database string `yaml:"database" json:"database" validate:"omitempty,min=1,max=64"`
	Username string `yaml:"username" json:"username" validate:"omitempty,min=1,max=32"`
	Password string `yaml:"password" json:"password" validate:"omitempty,min=1,max=128"`
	SSLMode  string `yaml:"ssl_mode" json:"ssl_mode" validate:"omitempty,oneof=disable require verify-ca verify-full,max=20"`

	// Read replica specific settings
	Weight     int           `yaml:"weight" json:"weight" validate:"omitempty,min=1,max=100"`            // Load balancing weight
	MaxLatency time.Duration `yaml:"max_latency" json:"max_latency" validate:"omitempty,min=1ms,max=5s"` // Maximum acceptable latency
	Enabled    bool          `yaml:"enabled" json:"enabled" validate:"omitempty,default=true"`           // Whether this replica is enabled

	// Connection pool settings for read replica (can override main config)
	MaxConnections     int           `yaml:"max_connections" json:"max_connections" validate:"omitempty,min=1,max=1000"`
	MinConnections     int           `yaml:"min_connections" json:"min_connections" validate:"omitempty,min=0,max=100"`
	MaxIdleConnections int           `yaml:"max_idle_connections" json:"max_idle_connections" validate:"omitempty,min=0,max=100"`
	MaxLifetime        time.Duration `yaml:"max_lifetime" json:"max_lifetime" validate:"omitempty,min=1m,max=12h"`
	IdleTimeout        time.Duration `yaml:"idle_timeout" json:"idle_timeout" validate:"omitempty,min=30s,max=1h"`
}

// ObservabilityConfig represents observability configuration
type ObservabilityConfig struct {
	EnableMetrics bool   `yaml:"enable_metrics" json:"enable_metrics"`
	EnableTracing bool   `yaml:"enable_tracing" json:"enable_tracing"`
	EnableLogging bool   `yaml:"enable_logging" json:"enable_logging"`
	LogLevel      string `yaml:"log_level" json:"log_level" validate:"omitempty,oneof=debug info warn error fatal,max=10"`
}

// PaginationConfig represents pagination configuration
type PaginationConfig struct {
	DefaultLimit int `yaml:"default_limit" json:"default_limit" validate:"required,min=1,max=1000"`
	MaxLimit     int `yaml:"max_limit" json:"max_limit" validate:"required,min=1,max=1000"`
	MinLimit     int `yaml:"min_limit" json:"min_limit" validate:"required,min=1,max=1000"`
}

// ConnectionString returns the database connection string
func (c *DatabaseConfig) ConnectionString() string {
	switch c.Driver {
	case "sqlite":
		// SQLite only requires database path
		if c.Database == "" {
			return ""
		}
		return c.Database
	default:
		// Validate required fields for other drivers
		if c.Host == "" || c.Port <= 0 || c.Database == "" || c.Username == "" || c.Password == "" {
			return ""
		}
	}

	switch c.Driver {
	case "postgres":
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			c.Host, c.Port, c.Username, c.Password, c.Database, c.SSLMode)
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
			c.Username, c.Password, c.Host, c.Port, c.Database)
	case "sqlserver":
		return fmt.Sprintf("server=%s;user id=%s;password=%s;database=%s;port=%d",
			c.Host, c.Username, c.Password, c.Database, c.Port)
	default:
		return ""
	}
}

// Validate validates the DatabaseConfig
func (c *DatabaseConfig) Validate() error {
	// Validate required fields
	if c.Driver == "" {
		return fmt.Errorf("driver is required")
	}
	if c.Host == "" {
		return fmt.Errorf("host is required")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", c.Port)
	}
	if c.Database == "" {
		return fmt.Errorf("database is required")
	}
	if c.Username == "" {
		return fmt.Errorf("username is required")
	}
	if c.Password == "" {
		return fmt.Errorf("password is required")
	}

	// Validate pool configuration
	if c.MaxConnections <= 0 || c.MaxConnections > 10000 {
		return fmt.Errorf("max_connections must be between 1 and 10000, got %d", c.MaxConnections)
	}
	if c.MinConnections < 0 || c.MinConnections > 1000 {
		return fmt.Errorf("min_connections must be between 0 and 1000, got %d", c.MinConnections)
	}
	if c.MaxIdleConnections < 0 || c.MaxIdleConnections > 1000 {
		return fmt.Errorf("max_idle_connections must be between 0 and 1000, got %d", c.MaxIdleConnections)
	}
	if c.MinConnections > c.MaxConnections {
		return fmt.Errorf("min_connections (%d) cannot be greater than max_connections (%d)", c.MinConnections, c.MaxConnections)
	}
	if c.MaxIdleConnections > c.MaxConnections {
		return fmt.Errorf("max_idle_connections (%d) cannot be greater than max_connections (%d)", c.MaxIdleConnections, c.MaxConnections)
	}

	// Validate timeouts with reasonable bounds
	if c.MaxLifetime < 1*time.Minute || c.MaxLifetime > 12*time.Hour {
		return fmt.Errorf("max_lifetime must be between 1 minute and 12 hours, got %v", c.MaxLifetime)
	}
	if c.IdleTimeout < 30*time.Second || c.IdleTimeout > 1*time.Hour {
		return fmt.Errorf("idle_timeout must be between 30 seconds and 1 hour, got %v", c.IdleTimeout)
	}
	if c.MaxLifetime < c.IdleTimeout {
		return fmt.Errorf("max_lifetime (%v) must be greater than idle_timeout (%v)", c.MaxLifetime, c.IdleTimeout)
	}

	// Validate timeouts
	if c.ConnectionTimeout < 1*time.Second || c.ConnectionTimeout > 30*time.Second {
		return fmt.Errorf("connection_timeout must be between 1 second and 30 seconds, got %v", c.ConnectionTimeout)
	}
	if c.QueryTimeout < 100*time.Millisecond || c.QueryTimeout > 5*time.Minute {
		return fmt.Errorf("query_timeout must be between 100ms and 5 minutes, got %v", c.QueryTimeout)
	}
	if c.TransactionTimeout < 1*time.Second || c.TransactionTimeout > 10*time.Minute {
		return fmt.Errorf("transaction_timeout must be between 1 second and 10 minutes, got %v", c.TransactionTimeout)
	}
	if c.QueryTimeout > c.TransactionTimeout {
		return fmt.Errorf("query_timeout (%v) cannot be greater than transaction_timeout (%v)", c.QueryTimeout, c.TransactionTimeout)
	}

	// Validate retry configuration
	if c.Retry.Enabled {
		if c.Retry.MaxAttempts <= 0 || c.Retry.MaxAttempts > 10 {
			return fmt.Errorf("retry max_attempts must be between 1 and 10, got %d", c.Retry.MaxAttempts)
		}
		if c.Retry.InitialDelay <= 0 || c.Retry.InitialDelay > 30*time.Second {
			return fmt.Errorf("retry initial_delay must be between 100ms and 30 seconds, got %v", c.Retry.InitialDelay)
		}
		if c.Retry.MaxDelay < 1*time.Second || c.Retry.MaxDelay > 5*time.Minute {
			return fmt.Errorf("retry max_delay must be between 1 second and 5 minutes, got %v", c.Retry.MaxDelay)
		}
		if c.Retry.MaxDelay < c.Retry.InitialDelay {
			return fmt.Errorf("max_delay (%v) cannot be less than initial_delay (%v)", c.Retry.MaxDelay, c.Retry.InitialDelay)
		}
	}

	// Validate health check configuration
	if c.HealthCheck.Enabled {
		if c.HealthCheck.Interval < 5*time.Second || c.HealthCheck.Interval > 5*time.Minute {
			return fmt.Errorf("health_check interval must be between 5 seconds and 5 minutes, got %v", c.HealthCheck.Interval)
		}
		if c.HealthCheck.Timeout < 1*time.Second || c.HealthCheck.Timeout > 30*time.Second {
			return fmt.Errorf("health_check timeout must be between 1 second and 30 seconds, got %v", c.HealthCheck.Timeout)
		}
		if c.HealthCheck.Timeout >= c.HealthCheck.Interval {
			return fmt.Errorf("health_check_timeout (%v) must be less than health_check_interval (%v)", c.HealthCheck.Timeout, c.HealthCheck.Interval)
		}
		if c.HealthCheck.MaxFailures <= 0 || c.HealthCheck.MaxFailures > 10 {
			return fmt.Errorf("health_check max_failures must be between 1 and 10, got %d", c.HealthCheck.MaxFailures)
		}
		if c.HealthCheck.Query == "" {
			return fmt.Errorf("health_check query is required when enabled")
		}
	}

	// Validate read replica configuration
	if err := c.ValidateReadReplicas(); err != nil {
		return fmt.Errorf("read replica validation failed: %w", err)
	}

	// Validate pagination consistency
	if c.Pagination != nil {
		if c.Pagination.MinLimit <= 0 || c.Pagination.MinLimit > 1000 {
			return fmt.Errorf("pagination min_limit must be between 1 and 1000, got %d", c.Pagination.MinLimit)
		}
		if c.Pagination.DefaultLimit <= 0 || c.Pagination.DefaultLimit > 10000 {
			return fmt.Errorf("pagination default_limit must be between 1 and 10000, got %d", c.Pagination.DefaultLimit)
		}
		if c.Pagination.MaxLimit <= 0 || c.Pagination.MaxLimit > 100000 {
			return fmt.Errorf("pagination max_limit must be between 1 and 100000, got %d", c.Pagination.MaxLimit)
		}
		if c.Pagination.MinLimit > c.Pagination.DefaultLimit ||
			c.Pagination.DefaultLimit > c.Pagination.MaxLimit {
			return fmt.Errorf("pagination limits must be: min_limit (%d) <= default_limit (%d) <= max_limit (%d)",
				c.Pagination.MinLimit, c.Pagination.DefaultLimit, c.Pagination.MaxLimit)
		}
	}

	// Validate result size consistency
	if c.MaxQuerySize < 1024 || c.MaxQuerySize > 1048576 {
		return fmt.Errorf("max_query_size must be between 1KB and 1MB, got %d", c.MaxQuerySize)
	}
	if c.MaxResultSize < 1000 || c.MaxResultSize > 1000000 {
		return fmt.Errorf("max_result_size must be between 1KB and 1MB, got %d", c.MaxResultSize)
	}
	if c.MaxResultSize > c.MaxQuerySize {
		return fmt.Errorf("max_result_size (%d) cannot be greater than max_query_size (%d)",
			c.MaxResultSize, c.MaxQuerySize)
	}

	// Validate health check interval consistency
	if c.HealthCheckInterval < 5*time.Second || c.HealthCheckInterval > 5*time.Minute {
		return fmt.Errorf("health_check_interval must be between 5 seconds and 5 minutes, got %v", c.HealthCheckInterval)
	}
	if c.HealthCheck.Enabled && c.HealthCheckInterval < c.HealthCheck.Timeout {
		return fmt.Errorf("health_check_interval (%v) must be greater than health_check_timeout (%v)",
			c.HealthCheckInterval, c.HealthCheck.Timeout)
	}

	return nil
}

// ReadReplicaDSNs returns DSN strings for all enabled read replicas (GORM format)
func (c *DatabaseConfig) ReadReplicaDSNs() []string {
	var dsns []string

	for _, replica := range c.ReadReplicas {
		if !replica.Enabled {
			continue
		}

		// Use replica-specific settings or fall back to main config
		username := replica.Username
		if username == "" {
			username = c.Username
		}

		password := replica.Password
		if password == "" {
			password = c.Password
		}

		database := replica.Database
		if database == "" {
			database = c.Database
		}

		sslMode := replica.SSLMode
		if sslMode == "" {
			sslMode = c.SSLMode
		}

		dsn := c.buildDSN(replica.Host, replica.Port, username, password, database, sslMode)
		dsns = append(dsns, dsn)
	}

	return dsns
}

// buildDSN builds a DSN string for GORM (Data Source Name)
func (c *DatabaseConfig) buildDSN(host string, port int, username, password, database, sslMode string) string {
	switch c.Driver {
	case "postgres":
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			host, port, username, password, database, sslMode)
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
			username, password, host, port, database)
	case "sqlite":
		return database
	case "sqlserver":
		return fmt.Sprintf("server=%s;user id=%s;password=%s;database=%s;port=%d",
			host, username, password, database, port)
	default:
		return ""
	}
}

// ReadReplicaConnectionStrings returns connection strings for all enabled read replicas
func (c *DatabaseConfig) ReadReplicaConnectionStrings() []string {
	var connections []string

	for _, replica := range c.ReadReplicas {
		if !replica.Enabled {
			continue
		}

		// Use replica-specific settings or fall back to main config
		username := replica.Username
		if username == "" {
			username = c.Username
		}

		password := replica.Password
		if password == "" {
			password = c.Password
		}

		database := replica.Database
		if database == "" {
			database = c.Database
		}

		sslMode := replica.SSLMode
		if sslMode == "" {
			sslMode = c.SSLMode
		}

		connStr := c.buildConnectionString(replica.Host, fmt.Sprintf("%d", replica.Port), username, password, database, sslMode)
		connections = append(connections, connStr)
	}

	return connections
}

// GetEnabledReadReplicas returns only the enabled read replicas
func (c *DatabaseConfig) GetEnabledReadReplicas() []ReadReplicaConfig {
	var enabled []ReadReplicaConfig

	for _, replica := range c.ReadReplicas {
		if replica.Enabled {
			enabled = append(enabled, replica)
		}
	}

	return enabled
}

// HasReadReplicas returns true if there are any enabled read replicas
func (c *DatabaseConfig) HasReadReplicas() bool {
	return len(c.GetEnabledReadReplicas()) > 0
}

// buildConnectionString builds a connection string for the given parameters
func (c *DatabaseConfig) buildConnectionString(host, port, username, password, database, sslMode string) string {
	switch c.Driver {
	case "postgres":
		return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			host, port, username, password, database, sslMode)
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
			username, password, host, port, database)
	case "sqlite":
		return database
	case "sqlserver":
		return fmt.Sprintf("server=%s;user id=%s;password=%s;database=%s;port=%s",
			host, username, password, database, port)
	default:
		return ""
	}
}

// ValidateReadReplicas validates the read replica configuration
func (c *DatabaseConfig) ValidateReadReplicas() error {
	for i, replica := range c.ReadReplicas {
		if !replica.Enabled {
			continue
		}

		// Validate required fields
		if replica.Host == "" {
			return fmt.Errorf("read_replica[%d]: host is required", i)
		}
		if len(replica.Host) > 255 {
			return fmt.Errorf("read_replica[%d]: host length cannot exceed 255 characters", i)
		}
		if replica.Port < 1 || replica.Port > 65535 {
			return fmt.Errorf("read_replica[%d]: port must be between 1 and 65535", i)
		}

		// Validate weight if specified
		if replica.Weight > 0 && (replica.Weight < 1 || replica.Weight > 100) {
			return fmt.Errorf("read_replica[%d]: weight must be between 1 and 100", i)
		}

		// Validate max latency if specified
		if replica.MaxLatency > 0 && (replica.MaxLatency < time.Millisecond || replica.MaxLatency > 5*time.Second) {
			return fmt.Errorf("read_replica[%d]: max_latency must be between 1ms and 5s", i)
		}

		// Validate connection pool settings if specified
		if replica.MaxConnections > 0 {
			if replica.MinConnections > replica.MaxConnections {
				return fmt.Errorf("read_replica[%d]: min_connections cannot be greater than max_connections", i)
			}
			if replica.MaxIdleConnections > replica.MaxConnections {
				return fmt.Errorf("read_replica[%d]: max_idle_connections cannot be greater than max_connections", i)
			}
		}

		if replica.MaxLifetime > 0 && replica.IdleTimeout > 0 {
			if replica.MaxLifetime < replica.IdleTimeout {
				return fmt.Errorf("read_replica[%d]: max_lifetime must be greater than idle_timeout", i)
			}

			// Validate string lengths
			if replica.Database != "" && len(replica.Database) > 64 {
				return fmt.Errorf("read_replica[%d]: database length cannot exceed 64 characters", i)
			}
			if replica.Username != "" && len(replica.Username) > 32 {
				return fmt.Errorf("read_replica[%d]: username length cannot exceed 32 characters", i)
			}
			if replica.Password != "" && len(replica.Password) > 128 {
				return fmt.Errorf("read_replica[%d]: password length cannot exceed 128 characters", i)
			}
			if replica.SSLMode != "" && len(replica.SSLMode) > 20 {
				return fmt.Errorf("read_replica[%d]: ssl_mode length cannot exceed 20 characters", i)
			}
		}
	}

	return nil
}

// DefaultDatabaseConfig returns a default database configuration
func DefaultDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Driver:   "postgres",
		Host:     "localhost",
		Port:     5432,
		Database: "ormx",
		Username: "postgres",
		Password: "password",
		SSLMode:  "disable",

		// Connection Pool Configuration
		MaxConnections:     100,
		MinConnections:     10,
		MaxIdleConnections: 20,
		MaxLifetime:        1 * time.Hour,
		IdleTimeout:        5 * time.Minute,
		AcquireTimeout:     10 * time.Second,
		LeakDetection:      true,
		LeakTimeout:        1 * time.Minute,

		// Timeout Configuration
		ConnectionTimeout:  10 * time.Second,
		QueryTimeout:       30 * time.Second,
		TransactionTimeout: 5 * time.Minute,
		StatementTimeout:   1 * time.Second,
		CancelTimeout:      5 * time.Second,

		// Retry Configuration
		Retry: RetryConfig{
			Enabled:           true,
			MaxAttempts:       3,
			InitialDelay:      1 * time.Second,
			MaxDelay:          30 * time.Second,
			BackoffMultiplier: 2.0,
			Jitter:            true,
			RetryableErrors: []string{
				"connection refused",
				"connection reset",
				"timeout",
				"deadlock",
			},
			NonRetryableErrors: []string{
				"syntax error",
				"permission denied",
				"table not found",
			},
		},

		// Health Check Configuration
		HealthCheck: HealthCheckConfig{
			Enabled:      true,
			Interval:     30 * time.Second,
			Timeout:      5 * time.Second,
			Query:        "SELECT 1",
			MaxFailures:  3,
			RecoveryTime: 1 * time.Minute,
		},

		// Logging Configuration
		LogLevel:  "info",
		LogFormat: "json",
		LogFields: make(map[string]string),

		// Observability Configuration
		Metrics:             true,
		Tracing:             true,
		HealthCheckInterval: 30 * time.Second,

		// Security Configuration
		MaxQuerySize:       65536,  // 64KB
		MaxResultSize:      100000, // 100KB
		EnableQueryLogging: false,
		MaskSensitiveData:  true,

		// Pagination Configuration
		Pagination: &PaginationConfig{
			DefaultLimit: 20,
			MaxLimit:     100,
			MinLimit:     1,
		},

		// Encryption Configuration
		Encryption: &EncryptionConfig{
			Enabled:   true,
			Algorithm: "aes-256-gcm",
		},

		// Read Replicas Configuration
		ReadReplicas: []ReadReplicaConfig{},
	}
}
