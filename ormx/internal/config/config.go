// Package config provides configuration management for the GORM data access layer.
// It supports environment-based configuration with secure defaults and validation.
package config

import (
	"crypto/tls"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// DatabaseType represents the type of database
type DatabaseType string

const (
	// PostgreSQL database type
	PostgreSQL DatabaseType = "postgres"
	// MySQL database type
	MySQL DatabaseType = "mysql"
)

// Config holds all configuration options for the database connection
type Config struct {
	// Database connection settings
	Type     DatabaseType `json:"type" yaml:"type"`
	Host     string       `json:"host" yaml:"host"`
	Port     int          `json:"port" yaml:"port"`
	Database string       `json:"database" yaml:"database"`
	Username string       `json:"username" yaml:"username"`
	Password string       `json:"-" yaml:"-"` // Excluded from JSON/YAML for security

	// SSL/TLS settings
	SSLMode   string      `json:"ssl_mode" yaml:"ssl_mode"`
	SSLCert   string      `json:"ssl_cert" yaml:"ssl_cert"`
	SSLKey    string      `json:"ssl_key" yaml:"ssl_key"`
	SSLRootCA string      `json:"ssl_root_ca" yaml:"ssl_root_ca"`
	TLSConfig *tls.Config `json:"-" yaml:"-"`

	// Connection pool settings
	MaxOpenConns    int           `json:"max_open_conns" yaml:"max_open_conns"`
	MaxIdleConns    int           `json:"max_idle_conns" yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime" yaml:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time" yaml:"conn_max_idle_time"`

	// Timeout settings
	ConnectTimeout time.Duration `json:"connect_timeout" yaml:"connect_timeout"`
	QueryTimeout   time.Duration `json:"query_timeout" yaml:"query_timeout"`

	// go-ormx settings
	PreparedStatements bool `json:"prepared_statements" yaml:"prepared_statements"`
	DisableForeignKeys bool `json:"disable_foreign_keys" yaml:"disable_foreign_keys"`
	LogLevel           int  `json:"log_level" yaml:"log_level"`

	// Performance settings
	BatchSize           int  `json:"batch_size" yaml:"batch_size"`
	SkipDefaultTx       bool `json:"skip_default_tx" yaml:"skip_default_tx"`
	EnableQueryOptimize bool `json:"enable_query_optimize" yaml:"enable_query_optimize"`

	// Migration settings
	AutoMigrate   bool   `json:"auto_migrate" yaml:"auto_migrate"`
	MigrationPath string `json:"migration_path" yaml:"migration_path"`

	// Observability settings
	EnableMetrics      bool   `json:"enable_metrics" yaml:"enable_metrics"`
	EnableTracing      bool   `json:"enable_tracing" yaml:"enable_tracing"`
	MetricsNamespace   string `json:"metrics_namespace" yaml:"metrics_namespace"`
	TracingServiceName string `json:"tracing_service_name" yaml:"tracing_service_name"`

	// Field-level encryption settings
	EncryptionEnabled                 bool          `json:"encryption_enabled" yaml:"encryption_enabled"`
	EncryptionAlgorithm               string        `json:"encryption_algorithm" yaml:"encryption_algorithm"`
	EncryptionMasterKey               string        `json:"-" yaml:"-"`
	EncryptionKeySalt                 string        `json:"-" yaml:"-"`
	EncryptionKeyRotationEnabled      bool          `json:"encryption_key_rotation_enabled" yaml:"encryption_key_rotation_enabled"`
	EncryptionKeyRotationPeriod       time.Duration `json:"encryption_key_rotation_period" yaml:"encryption_key_rotation_period"`
	EncryptionUseHardwareAcceleration bool          `json:"encryption_use_hardware_acceleration" yaml:"encryption_use_hardware_acceleration"`

	// PostgreSQL Row-Level Security (RLS)
	RLSEnabled        bool     `json:"rls_enabled" yaml:"rls_enabled"`
	RLSTenantGUC      string   `json:"rls_tenant_guc" yaml:"rls_tenant_guc"`
	RLSTenantColumn   string   `json:"rls_tenant_column" yaml:"rls_tenant_column"`
	RLSManagePolicies bool     `json:"rls_manage_policies" yaml:"rls_manage_policies"`
	RLSPolicyName     string   `json:"rls_policy_name" yaml:"rls_policy_name"`
	RLSTables         []string `json:"rls_tables" yaml:"rls_tables"`
	RLSRequireTenant  bool     `json:"rls_require_tenant" yaml:"rls_require_tenant"`
}

// DefaultConfig returns a configuration with secure defaults
func DefaultConfig() *Config {
	return &Config{
		Type:     PostgreSQL,
		Host:     "localhost",
		Port:     5432,
		Database: "db",
		Username: "postgres",
		Password: "",

		SSLMode: "require",

		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,

		ConnectTimeout: 30 * time.Second,
		QueryTimeout:   30 * time.Second,

		PreparedStatements: true,
		DisableForeignKeys: false,
		LogLevel:           1, // Silent by default

		BatchSize:           100,
		SkipDefaultTx:       true,
		EnableQueryOptimize: true,

		AutoMigrate:   false,
		MigrationPath: "./migrations",

		EnableMetrics:      true,
		EnableTracing:      true,
		MetricsNamespace:   "db",
		TracingServiceName: "db-go-ormx",

		// Encryption defaults
		EncryptionEnabled:                 false,
		EncryptionAlgorithm:               "AES-256-GCM",
		EncryptionKeyRotationEnabled:      false,
		EncryptionKeyRotationPeriod:       30 * 24 * time.Hour,
		EncryptionUseHardwareAcceleration: true,

		// RLS defaults
		RLSEnabled:        false,
		RLSTenantGUC:      "app.tenant_id",
		RLSTenantColumn:   "tenant_id",
		RLSManagePolicies: false,
		RLSPolicyName:     "tenant_isolation",
		RLSRequireTenant:  false,
	}
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() (*Config, error) {
	config := DefaultConfig()

	// Database connection
	if dbType := os.Getenv("DB_TYPE"); dbType != "" {
		config.Type = DatabaseType(strings.ToLower(dbType))
	}

	if host := os.Getenv("DB_HOST"); host != "" {
		config.Host = host
	}

	if portStr := os.Getenv("DB_PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid DB_PORT: %w", err)
		}
		config.Port = port
	}

	if database := os.Getenv("DB_DATABASE"); database != "" {
		config.Database = database
	}

	if username := os.Getenv("DB_USERNAME"); username != "" {
		config.Username = username
	}

	if password := os.Getenv("DB_PASSWORD"); password != "" {
		config.Password = password
	}

	// SSL settings
	if sslMode := os.Getenv("DB_SSL_MODE"); sslMode != "" {
		config.SSLMode = sslMode
	}

	if sslCert := os.Getenv("DB_SSL_CERT"); sslCert != "" {
		config.SSLCert = sslCert
	}

	if sslKey := os.Getenv("DB_SSL_KEY"); sslKey != "" {
		config.SSLKey = sslKey
	}

	if sslRootCA := os.Getenv("DB_SSL_ROOT_CA"); sslRootCA != "" {
		config.SSLRootCA = sslRootCA
	}

	// Connection pool settings
	if maxOpenStr := os.Getenv("DB_MAX_OPEN_CONNS"); maxOpenStr != "" {
		maxOpen, err := strconv.Atoi(maxOpenStr)
		if err != nil {
			return nil, fmt.Errorf("invalid DB_MAX_OPEN_CONNS: %w", err)
		}
		config.MaxOpenConns = maxOpen
	}

	if maxIdleStr := os.Getenv("DB_MAX_IDLE_CONNS"); maxIdleStr != "" {
		maxIdle, err := strconv.Atoi(maxIdleStr)
		if err != nil {
			return nil, fmt.Errorf("invalid DB_MAX_IDLE_CONNS: %w", err)
		}
		config.MaxIdleConns = maxIdle
	}

	if connLifetimeStr := os.Getenv("DB_CONN_MAX_LIFETIME"); connLifetimeStr != "" {
		connLifetime, err := time.ParseDuration(connLifetimeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid DB_CONN_MAX_LIFETIME: %w", err)
		}
		config.ConnMaxLifetime = connLifetime
	}

	if connIdleTimeStr := os.Getenv("DB_CONN_MAX_IDLE_TIME"); connIdleTimeStr != "" {
		connIdleTime, err := time.ParseDuration(connIdleTimeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid DB_CONN_MAX_IDLE_TIME: %w", err)
		}
		config.ConnMaxIdleTime = connIdleTime
	}

	// Timeout settings
	if connectTimeoutStr := os.Getenv("DB_CONNECT_TIMEOUT"); connectTimeoutStr != "" {
		connectTimeout, err := time.ParseDuration(connectTimeoutStr)
		if err != nil {
			return nil, fmt.Errorf("invalid DB_CONNECT_TIMEOUT: %w", err)
		}
		config.ConnectTimeout = connectTimeout
	}

	if queryTimeoutStr := os.Getenv("DB_QUERY_TIMEOUT"); queryTimeoutStr != "" {
		queryTimeout, err := time.ParseDuration(queryTimeoutStr)
		if err != nil {
			return nil, fmt.Errorf("invalid DB_QUERY_TIMEOUT: %w", err)
		}
		config.QueryTimeout = queryTimeout
	}

	// GORM settings
	if preparedStmtsStr := os.Getenv("DB_PREPARED_STATEMENTS"); preparedStmtsStr != "" {
		preparedStmts, err := strconv.ParseBool(preparedStmtsStr)
		if err != nil {
			return nil, fmt.Errorf("invalid DB_PREPARED_STATEMENTS: %w", err)
		}
		config.PreparedStatements = preparedStmts
	}

	if disableFKStr := os.Getenv("DB_DISABLE_FOREIGN_KEYS"); disableFKStr != "" {
		disableFK, err := strconv.ParseBool(disableFKStr)
		if err != nil {
			return nil, fmt.Errorf("invalid DB_DISABLE_FOREIGN_KEYS: %w", err)
		}
		config.DisableForeignKeys = disableFK
	}

	// Auto migration
	if autoMigrateStr := os.Getenv("DB_AUTO_MIGRATE"); autoMigrateStr != "" {
		autoMigrate, err := strconv.ParseBool(autoMigrateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid DB_AUTO_MIGRATE: %w", err)
		}
		config.AutoMigrate = autoMigrate
	}

	if migrationPath := os.Getenv("DB_MIGRATION_PATH"); migrationPath != "" {
		config.MigrationPath = migrationPath
	}

	// Observability
	if enableMetricsStr := os.Getenv("DB_ENABLE_METRICS"); enableMetricsStr != "" {
		enableMetrics, err := strconv.ParseBool(enableMetricsStr)
		if err != nil {
			return nil, fmt.Errorf("invalid DB_ENABLE_METRICS: %w", err)
		}
		config.EnableMetrics = enableMetrics
	}

	if enableTracingStr := os.Getenv("DB_ENABLE_TRACING"); enableTracingStr != "" {
		enableTracing, err := strconv.ParseBool(enableTracingStr)
		if err != nil {
			return nil, fmt.Errorf("invalid DB_ENABLE_TRACING: %w", err)
		}
		config.EnableTracing = enableTracing
	}

	if metricsNamespace := os.Getenv("DB_METRICS_NAMESPACE"); metricsNamespace != "" {
		config.MetricsNamespace = metricsNamespace
	}

	if tracingServiceName := os.Getenv("DB_TRACING_SERVICE_NAME"); tracingServiceName != "" {
		config.TracingServiceName = tracingServiceName
	}

	// Field-level encryption
	if encEnabledStr := os.Getenv("DB_ENCRYPTION_ENABLED"); encEnabledStr != "" {
		enabled, err := strconv.ParseBool(encEnabledStr)
		if err != nil {
			return nil, fmt.Errorf("invalid DB_ENCRYPTION_ENABLED: %w", err)
		}
		config.EncryptionEnabled = enabled
	}

	if alg := os.Getenv("ENCRYPTION_ALGORITHM"); alg != "" {
		config.EncryptionAlgorithm = alg
	}

	if mk := os.Getenv("ENCRYPTION_MASTER_KEY"); mk != "" {
		config.EncryptionMasterKey = mk
	}

	if salt := os.Getenv("ENCRYPTION_KEY_SALT"); salt != "" {
		config.EncryptionKeySalt = salt
	}

	if rotateStr := os.Getenv("ENCRYPTION_KEY_ROTATION_ENABLED"); rotateStr != "" {
		rotate, err := strconv.ParseBool(rotateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid ENCRYPTION_KEY_ROTATION_ENABLED: %w", err)
		}
		config.EncryptionKeyRotationEnabled = rotate
	}

	if rotatePeriod := os.Getenv("ENCRYPTION_KEY_ROTATION_PERIOD"); rotatePeriod != "" {
		d, err := time.ParseDuration(rotatePeriod)
		if err != nil {
			return nil, fmt.Errorf("invalid ENCRYPTION_KEY_ROTATION_PERIOD: %w", err)
		}
		config.EncryptionKeyRotationPeriod = d
	}

	if hwStr := os.Getenv("ENCRYPTION_USE_HARDWARE_ACCELERATION"); hwStr != "" {
		hw, err := strconv.ParseBool(hwStr)
		if err != nil {
			return nil, fmt.Errorf("invalid ENCRYPTION_USE_HARDWARE_ACCELERATION: %w", err)
		}
		config.EncryptionUseHardwareAcceleration = hw
	}

	// PostgreSQL Row-Level Security (RLS)
	if rlsEnabledStr := os.Getenv("DB_RLS_ENABLED"); rlsEnabledStr != "" {
		enabled, err := strconv.ParseBool(rlsEnabledStr)
		if err != nil {
			return nil, fmt.Errorf("invalid DB_RLS_ENABLED: %w", err)
		}
		config.RLSEnabled = enabled
	}

	if guc := os.Getenv("DB_RLS_TENANT_GUC"); guc != "" {
		config.RLSTenantGUC = guc
	}

	if col := os.Getenv("DB_RLS_TENANT_COLUMN"); col != "" {
		config.RLSTenantColumn = col
	}

	if manageStr := os.Getenv("DB_RLS_MANAGE_POLICIES"); manageStr != "" {
		manage, err := strconv.ParseBool(manageStr)
		if err != nil {
			return nil, fmt.Errorf("invalid DB_RLS_MANAGE_POLICIES: %w", err)
		}
		config.RLSManagePolicies = manage
	}

	if policy := os.Getenv("DB_RLS_POLICY_NAME"); policy != "" {
		config.RLSPolicyName = policy
	}

	if tables := os.Getenv("DB_RLS_TABLES"); tables != "" {
		// comma-separated table names
		parts := strings.Split(tables, ",")
		cleaned := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				cleaned = append(cleaned, p)
			}
		}
		config.RLSTables = cleaned
	}

	if reqStr := os.Getenv("DB_RLS_REQUIRE_TENANT"); reqStr != "" {
		req, err := strconv.ParseBool(reqStr)
		if err != nil {
			return nil, fmt.Errorf("invalid DB_RLS_REQUIRE_TENANT: %w", err)
		}
		config.RLSRequireTenant = req
	}

	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Type != PostgreSQL && c.Type != MySQL {
		return fmt.Errorf("unsupported database type: %s", c.Type)
	}

	if c.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("invalid database port: %d", c.Port)
	}

	if c.Database == "" {
		return fmt.Errorf("database name is required")
	}

	if c.Username == "" {
		return fmt.Errorf("database username is required")
	}

	if c.MaxOpenConns <= 0 {
		return fmt.Errorf("max_open_conns must be positive")
	}

	if c.MaxIdleConns < 0 {
		return fmt.Errorf("max_idle_conns cannot be negative")
	}

	if c.MaxIdleConns > c.MaxOpenConns {
		return fmt.Errorf("max_idle_conns cannot be greater than max_open_conns")
	}

	if c.ConnMaxLifetime <= 0 {
		return fmt.Errorf("conn_max_lifetime must be positive")
	}

	if c.ConnMaxIdleTime <= 0 {
		return fmt.Errorf("conn_max_idle_time must be positive")
	}

	if c.ConnectTimeout <= 0 {
		return fmt.Errorf("connect_timeout must be positive")
	}

	if c.QueryTimeout <= 0 {
		return fmt.Errorf("query_timeout must be positive")
	}

	if c.BatchSize <= 0 {
		return fmt.Errorf("batch_size must be positive")
	}

	return nil
}

// GetDSN returns the database connection string (DSN) for the configured database type
func (c *Config) GetDSN() string {
	switch c.Type {
	case PostgreSQL:
		return c.getPostgresDSN()
	case MySQL:
		return c.getMySQLDSN()
	default:
		return ""
	}
}

// getPostgresDSN returns PostgreSQL connection string
func (c *Config) getPostgresDSN() string {
	dsn := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.Username, c.Database, c.SSLMode)

	if c.Password != "" {
		dsn += fmt.Sprintf(" password=%s", c.Password)
	}

	if c.SSLCert != "" {
		dsn += fmt.Sprintf(" sslcert=%s", c.SSLCert)
	}

	if c.SSLKey != "" {
		dsn += fmt.Sprintf(" sslkey=%s", c.SSLKey)
	}

	if c.SSLRootCA != "" {
		dsn += fmt.Sprintf(" sslrootcert=%s", c.SSLRootCA)
	}

	// Add timeout
	dsn += fmt.Sprintf(" connect_timeout=%d", int(c.ConnectTimeout.Seconds()))

	return dsn
}

// getMySQLDSN returns MySQL connection string
func (c *Config) getMySQLDSN() string {
	// Format: [username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		c.Username, c.Password, c.Host, c.Port, c.Database)

	params := make([]string, 0)

	// Add charset
	params = append(params, "charset=utf8mb4")

	// Add parseTime for proper time handling
	params = append(params, "parseTime=True")

	// Add loc for timezone
	params = append(params, "loc=Local")

	// Add timeout
	params = append(params, fmt.Sprintf("timeout=%s", c.ConnectTimeout))
	params = append(params, fmt.Sprintf("readTimeout=%s", c.QueryTimeout))
	params = append(params, fmt.Sprintf("writeTimeout=%s", c.QueryTimeout))

	// Add TLS config
	if c.SSLMode != "disable" && c.SSLMode != "" {
		params = append(params, "tls=true")
	}

	if len(params) > 0 {
		dsn += "?" + strings.Join(params, "&")
	}

	return dsn
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	env := strings.ToLower(os.Getenv("ENVIRONMENT"))
	return env == "development" || env == "dev" || env == ""
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	env := strings.ToLower(os.Getenv("ENVIRONMENT"))
	return env == "production" || env == "prod"
}
