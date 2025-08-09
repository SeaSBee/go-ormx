//go:build integration_old
// +build integration_old

package config_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	gormx "go-ormx/ormx"
	"go-ormx/ormx/config"
	"go-ormx/ormx/models"
	"go-ormx/ormx/repositories"
	"go-ormx/tests/integration/utils"
)

func TestConfig_DatabaseConnection_Integration(t *testing.T) {
	utils.SkipIfShort(t)

	tests := []struct {
		name          string
		config        *config.Config
		expectSuccess bool
		expectedError string
	}{
		{
			name: "valid_postgresql_config",
			config: &config.Config{
				Type:            config.PostgreSQL,
				Host:            utils.GetEnvWithDefault("TEST_POSTGRES_HOST", "localhost"),
				Port:            5432,
				Database:        utils.GetEnvWithDefault("TEST_POSTGRES_DB", "go-ormx_test"),
				Username:        utils.GetEnvWithDefault("TEST_POSTGRES_USER", "postgres"),
				Password:        utils.GetEnvWithDefault("TEST_POSTGRES_PASSWORD", "password"),
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: 5 * time.Minute,
				ConnMaxIdleTime: 1 * time.Minute,
				ConnectTimeout:  10 * time.Second,
				QueryTimeout:    30 * time.Second,
				BatchSize:       100,
			},
			expectSuccess: true,
		},
		{
			name: "valid_mysql_config",
			config: &config.Config{
				Type:            config.MySQL,
				Host:            utils.GetEnvWithDefault("TEST_MYSQL_HOST", "localhost"),
				Port:            3306,
				Database:        utils.GetEnvWithDefault("TEST_MYSQL_DB", "go-ormx_test"),
				Username:        utils.GetEnvWithDefault("TEST_MYSQL_USER", "root"),
				Password:        utils.GetEnvWithDefault("TEST_MYSQL_PASSWORD", "password"),
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: 5 * time.Minute,
				ConnMaxIdleTime: 1 * time.Minute,
				ConnectTimeout:  10 * time.Second,
				QueryTimeout:    30 * time.Second,
				BatchSize:       100,
			},
			expectSuccess: os.Getenv("TEST_MYSQL_ENABLED") == "true",
		},
		{
			name: "invalid_host_config",
			config: &config.Config{
				Type:            config.PostgreSQL,
				Host:            "nonexistent-host-12345",
				Port:            5432,
				Database:        "test",
				Username:        "postgres",
				Password:        "password",
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: 5 * time.Minute,
				ConnMaxIdleTime: 1 * time.Minute,
				ConnectTimeout:  5 * time.Second,
				QueryTimeout:    30 * time.Second,
				BatchSize:       100,
			},
			expectSuccess: false,
			expectedError: "connection",
		},
		{
			name: "invalid_credentials_config",
			config: &config.Config{
				Type:            config.PostgreSQL,
				Host:            utils.GetEnvWithDefault("TEST_POSTGRES_HOST", "localhost"),
				Port:            5432,
				Database:        "test",
				Username:        "invalid_user",
				Password:        "invalid_password",
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: 5 * time.Minute,
				ConnMaxIdleTime: 1 * time.Minute,
				ConnectTimeout:  5 * time.Second,
				QueryTimeout:    30 * time.Second,
				BatchSize:       100,
			},
			expectSuccess: false,
			expectedError: "authentication",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.expectSuccess && tt.name == "valid_mysql_config" {
				t.Skip("MySQL test skipped - set TEST_MYSQL_ENABLED=true to enable")
			}

			logger := utils.NewTestLogger(t, "config")

			clientConfig := gormx.Config{
				Database: tt.config,
				Logger:   logger,
				Options: gormx.ClientOptions{
					EnableMetrics:    true,
					EnableMigrations: false, // Don't auto-migrate for invalid configs
					AutoMigrate:      false,
				},
			}

			client, err := gormx.NewClient(clientConfig)

			if tt.expectSuccess {
				if err != nil {
					t.Fatalf("Expected successful connection but got error: %v", err)
				}

				// Test basic functionality
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				err = client.Health(ctx)
				if err != nil {
					t.Errorf("Health check failed: %v", err)
				}

				client.Close()
			} else {
				if err == nil {
					if client != nil {
						client.Close()
					}
					t.Fatalf("Expected connection to fail but it succeeded")
				}

				if tt.expectedError != "" && !containsString(err.Error(), tt.expectedError) {
					t.Errorf("Expected error to contain '%s' but got: %v", tt.expectedError, err)
				}
			}
		})
	}
}

func TestConfig_ConnectionPool_Integration(t *testing.T) {
	utils.SkipIfShort(t)

	setup := utils.NewTestSetup(t)
	defer setup.TearDown(t)

	t.Run("connection_pool_limits", func(t *testing.T) {
		// Test that connection pool respects limits
		ctx := setup.Context()

		// Create multiple concurrent operations
		concurrency := 20
		done := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(id int) {
				user := setup.CreateTestUser(t, fmt.Sprintf("pool_test_%d@example.com", id), fmt.Sprintf("pooluser%d", id))
				_, err := setup.Client.Users().GetByID(ctx, user.ID)
				done <- err
			}(i)
		}

		// Wait for all operations to complete
		for i := 0; i < concurrency; i++ {
			err := <-done
			if err != nil {
				t.Errorf("Concurrent operation %d failed: %v", i, err)
			}
		}
	})

	t.Run("connection_timeout", func(t *testing.T) {
		// Test connection timeout behavior
		ctx, cancel := context.WithTimeout(setup.Context(), 1*time.Millisecond)
		defer cancel()

		// This should timeout
		_, err := setup.Client.Users().Find(ctx, repositories.Filter{})
		if err == nil {
			t.Error("Expected timeout error but operation succeeded")
		}
	})

	t.Run("connection_lifecycle", func(t *testing.T) {
		// Test connection lifecycle and cleanup
		ctx := setup.Context()

		// Perform operations to establish connections
		for i := 0; i < 5; i++ {
			user := setup.CreateTestUser(t, fmt.Sprintf("lifecycle_%d@example.com", i), fmt.Sprintf("lifecycleuser%d", i))
			_, err := setup.Client.Users().GetByID(ctx, user.ID)
			if err != nil {
				t.Errorf("Operation %d failed: %v", i, err)
			}
		}

		// Get connection stats
		stats := setup.Client.GetConnectionStats()
		if stats.OpenConnections <= 0 {
			t.Error("Expected active connections but found none")
		}

		t.Logf("Connection stats: Open=%d, InUse=%d, Idle=%d",
			stats.OpenConnections, stats.InUseConnections, stats.IdleConnections)
	})
}

func TestConfig_DatabaseTypes_Integration(t *testing.T) {
	utils.SkipIfShort(t)

	testCases := []struct {
		name       string
		dbType     config.DatabaseType
		envEnabled string
		skipReason string
	}{
		{
			name:       "postgresql",
			dbType:     config.PostgreSQL,
			envEnabled: "TEST_POSTGRES_ENABLED",
		},
		{
			name:       "mysql",
			dbType:     config.MySQL,
			envEnabled: "TEST_MYSQL_ENABLED",
			skipReason: "MySQL test skipped - set TEST_MYSQL_ENABLED=true to enable",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envEnabled != "" && os.Getenv(tc.envEnabled) != "true" {
				if tc.skipReason != "" {
					t.Skip(tc.skipReason)
				} else {
					t.Skipf("Skipping %s test - set %s=true to enable", tc.name, tc.envEnabled)
				}
			}

			// Create database-specific configuration
			dbConfig := createDatabaseConfig(tc.dbType)

			logger := utils.NewTestLogger(t, tc.name)
			clientConfig := gormx.Config{
				Database: dbConfig,
				Logger:   logger,
				Options: gormx.ClientOptions{
					EnableMetrics:    true,
					EnableMigrations: true,
					AutoMigrate:      true,
				},
			}

			client, err := gormx.NewClient(clientConfig)
			if err != nil {
				t.Fatalf("Failed to create client for %s: %v", tc.name, err)
			}
			defer client.Close()

			// Test basic CRUD operations
			ctx := context.Background()

			// Test user creation
			user := &models.User{
				Email:     fmt.Sprintf("test_%s@example.com", tc.name),
				Username:  fmt.Sprintf("testuser_%s", tc.name),
				FirstName: "Test",
				LastName:  "User",
				Status:    models.UserStatusActive,
				Role:      models.UserRoleUser,
			}

			err = client.Users().Create(ctx, user)
			if err != nil {
				t.Fatalf("Failed to create user: %v", err)
			}

			// Test user retrieval
			retrieved, err := client.Users().GetByID(ctx, user.ID)
			if err != nil {
				t.Fatalf("Failed to retrieve user: %v", err)
			}

			if retrieved.Email != user.Email {
				t.Errorf("Expected email %s, got %s", user.Email, retrieved.Email)
			}

			// Test user update
			retrieved.FirstName = "Updated"
			err = client.Users().Update(ctx, retrieved)
			if err != nil {
				t.Fatalf("Failed to update user: %v", err)
			}

			// Test user deletion
			err = client.Users().Delete(ctx, retrieved.ID)
			if err != nil {
				t.Fatalf("Failed to delete user: %v", err)
			}

			// Verify deletion
			_, err = client.Users().GetByID(ctx, retrieved.ID)
			if err == nil {
				t.Error("Expected error when retrieving deleted user")
			}
		})
	}
}

func TestConfig_SSLConnection_Integration(t *testing.T) {
	utils.SkipIfShort(t)
	utils.RequireEnvironment(t, "TEST_SSL_ENABLED")

	tests := []struct {
		name      string
		sslMode   string
		expectErr bool
	}{
		{
			name:      "ssl_require",
			sslMode:   "require",
			expectErr: false,
		},
		{
			name:      "ssl_disable",
			sslMode:   "disable",
			expectErr: false,
		},
		{
			name:      "ssl_prefer",
			sslMode:   "prefer",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbConfig := &config.Config{
				Type:            config.PostgreSQL,
				Host:            utils.GetEnvWithDefault("TEST_POSTGRES_HOST", "localhost"),
				Port:            5432,
				Database:        utils.GetEnvWithDefault("TEST_POSTGRES_DB", "go-ormx_test"),
				Username:        utils.GetEnvWithDefault("TEST_POSTGRES_USER", "postgres"),
				Password:        utils.GetEnvWithDefault("TEST_POSTGRES_PASSWORD", "password"),
				SSLMode:         tt.sslMode,
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: 5 * time.Minute,
				ConnectTimeout:  10 * time.Second,
				QueryTimeout:    30 * time.Second,
				BatchSize:       100,
			}

			logger := utils.NewTestLogger(t, "ssl")
			clientConfig := gormx.Config{
				Database: dbConfig,
				Logger:   logger,
				Options: gormx.ClientOptions{
					EnableMetrics:    false,
					EnableMigrations: false,
					AutoMigrate:      false,
				},
			}

			client, err := gormx.NewClient(clientConfig)

			if tt.expectErr {
				if err == nil {
					client.Close()
					t.Fatal("Expected SSL connection to fail but it succeeded")
				}
				return
			}

			if err != nil {
				t.Fatalf("SSL connection failed: %v", err)
			}
			defer client.Close()

			// Test basic operation with SSL
			ctx := context.Background()
			err = client.Health(ctx)
			if err != nil {
				t.Errorf("Health check failed with SSL: %v", err)
			}
		})
	}
}

// Helper functions

func createDatabaseConfig(dbType config.DatabaseType) *config.Config {
	baseConfig := &config.Config{
		Type:            dbType,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
		ConnectTimeout:  10 * time.Second,
		QueryTimeout:    30 * time.Second,
		BatchSize:       100,
	}

	switch dbType {
	case config.PostgreSQL:
		baseConfig.Host = utils.GetEnvWithDefault("TEST_POSTGRES_HOST", "localhost")
		baseConfig.Port = 5432
		baseConfig.Database = utils.GetEnvWithDefault("TEST_POSTGRES_DB", "go-ormx_test")
		baseConfig.Username = utils.GetEnvWithDefault("TEST_POSTGRES_USER", "postgres")
		baseConfig.Password = utils.GetEnvWithDefault("TEST_POSTGRES_PASSWORD", "password")
		baseConfig.SSLMode = "disable"
	case config.MySQL:
		baseConfig.Host = utils.GetEnvWithDefault("TEST_MYSQL_HOST", "localhost")
		baseConfig.Port = 3306
		baseConfig.Database = utils.GetEnvWithDefault("TEST_MYSQL_DB", "go-ormx_test")
		baseConfig.Username = utils.GetEnvWithDefault("TEST_MYSQL_USER", "root")
		baseConfig.Password = utils.GetEnvWithDefault("TEST_MYSQL_PASSWORD", "password")
	}

	return baseConfig
}

func containsString(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) && findStringIndex(s, substr) >= 0
}

func findStringIndex(s, substr string) int {
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
