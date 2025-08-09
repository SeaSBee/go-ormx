package config_test

import (
	"os"
	"testing"
	"time"

	"go-ormx/ormx/config"
)

func TestConfig_LoadFromEnv(t *testing.T) {
	// Set up test environment
	os.Setenv("DB_TYPE", "postgres")
	os.Setenv("DB_DATABASE", "test")
	os.Setenv("DB_USERNAME", "user")
	os.Setenv("DB_PASSWORD", "pass")
	defer clearEnv()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cfg == nil {
		t.Fatal("Expected config but got nil")
	}

	if cfg.Type != config.PostgreSQL {
		t.Errorf("Expected DB type PostgreSQL, got %v", cfg.Type)
	}

	if cfg.Database != "test" {
		t.Errorf("Expected database 'test', got %s", cfg.Database)
	}

	if cfg.Username != "user" {
		t.Errorf("Expected username 'user', got %s", cfg.Username)
	}
}

func TestConfig_DefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	if cfg == nil {
		t.Fatal("Expected config but got nil")
	}

	// Check that default values are set
	if cfg.Host == "" {
		t.Error("Default host should be set")
	}

	if cfg.Port == 0 {
		t.Error("Default port should be set")
	}

	if cfg.MaxOpenConns == 0 {
		t.Error("Default MaxOpenConns should be set")
	}
}

func TestDatabaseType_Values(t *testing.T) {
	tests := []struct {
		dbType   config.DatabaseType
		expected string
	}{
		{config.PostgreSQL, "postgres"},
		{config.MySQL, "mysql"},
	}

	for _, tt := range tests {
		t.Run(string(tt.dbType), func(t *testing.T) {
			result := string(tt.dbType)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
	}{
		{
			name: "valid_config",
			config: &config.Config{
				Type:            config.PostgreSQL,
				Host:            "localhost",
				Port:            5432,
				Database:        "test",
				Username:        "user",
				Password:        "pass",
				MaxOpenConns:    25,
				MaxIdleConns:    5,
				ConnMaxLifetime: time.Minute,
				ConnMaxIdleTime: time.Minute,
				ConnectTimeout:  30 * time.Second,
				QueryTimeout:    30 * time.Second,
				BatchSize:       100,
			},
			expectError: false,
		},
		{
			name: "empty_database",
			config: &config.Config{
				Type:         config.PostgreSQL,
				Host:         "localhost",
				Port:         5432,
				Database:     "",
				Username:     "user",
				Password:     "pass",
				MaxOpenConns: 25,
			},
			expectError: true,
		},
		{
			name: "invalid_port",
			config: &config.Config{
				Type:         config.PostgreSQL,
				Host:         "localhost",
				Port:         0,
				Database:     "test",
				Username:     "user",
				Password:     "pass",
				MaxOpenConns: 25,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestConfig_GetDSN(t *testing.T) {
	tests := []struct {
		name   string
		config *config.Config
	}{
		{
			name: "postgresql_config",
			config: &config.Config{
				Type:     config.PostgreSQL,
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				Username: "user",
				Password: "pass",
				SSLMode:  "require",
			},
		},
		{
			name: "mysql_config",
			config: &config.Config{
				Type:     config.MySQL,
				Host:     "localhost",
				Port:     3306,
				Database: "testdb",
				Username: "user",
				Password: "pass",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsn := tt.config.GetDSN()
			if dsn == "" {
				t.Error("DSN should not be empty")
			}

			// DSN should contain the database name
			if !contains(dsn, tt.config.Database) {
				t.Errorf("DSN should contain database name '%s'", tt.config.Database)
			}

			// DSN should contain the username
			if !contains(dsn, tt.config.Username) {
				t.Errorf("DSN should contain username '%s'", tt.config.Username)
			}
		})
	}
}

func TestConfig_BasicFunctionality(t *testing.T) {
	cfg := &config.Config{
		Type:     config.PostgreSQL,
		Host:     "localhost",
		Port:     5432,
		Database: "test",
		Username: "user",
		Password: "pass",
	}

	// Test that methods exist and can be called
	dsn := cfg.GetDSN()
	if dsn == "" {
		t.Error("DSN should not be empty")
	}

	isDev := cfg.IsDevelopment()
	isProd := cfg.IsProduction()

	// These are just testing that the methods exist and return something
	if isDev && isProd {
		t.Error("Cannot be both development and production")
	}
}

// Helper function to clear environment variables
func clearEnv() {
	envVars := []string{
		"DB_TYPE", "DB_HOST", "DB_PORT", "DB_DATABASE", "DB_USERNAME", "DB_PASSWORD",
		"DB_SSL_MODE", "DB_MAX_OPEN_CONNS", "DB_MAX_IDLE_CONNS",
		"APP_ENVIRONMENT", "LOG_LEVEL",
	}

	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) && findIndex(s, substr) >= 0
}

func findIndex(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Benchmark tests
func BenchmarkConfig_LoadFromEnv(b *testing.B) {
	// Set up minimal environment
	os.Setenv("DB_TYPE", "postgres")
	os.Setenv("DB_DATABASE", "test")
	os.Setenv("DB_USERNAME", "user")
	os.Setenv("DB_PASSWORD", "pass")
	defer clearEnv()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := config.LoadFromEnv()
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkConfig_Validate(b *testing.B) {
	cfg := &config.Config{
		Type:            config.PostgreSQL,
		Host:            "localhost",
		Port:            5432,
		Database:        "test",
		Username:        "user",
		Password:        "pass",
		MaxOpenConns:    25,
		ConnMaxLifetime: time.Minute * 5,
		ConnMaxIdleTime: time.Minute,
		ConnectTimeout:  time.Second * 30,
		QueryTimeout:    time.Second * 30,
		BatchSize:       100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := cfg.Validate()
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkConfig_GetDSN(b *testing.B) {
	cfg := &config.Config{
		Type:     config.PostgreSQL,
		Host:     "localhost",
		Port:     5432,
		Database: "test",
		Username: "user",
		Password: "pass",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cfg.GetDSN()
	}
}
