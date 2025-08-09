# Go-ORMX Integration Tests

This directory contains integration tests for the Go-ORMX module. These tests verify the complete system functionality with real database connections and end-to-end workflows.

## 📋 Test Coverage

### Core Components
- **Basic Integration** (`basic/`) - Client creation, user repository operations, migration status, health checks
- **Test Utilities** (`utils/`) - Common test infrastructure, database setup/teardown, mock utilities

### Test Utilities
- **Test Setup & Helpers** (`utils/`) - Common test infrastructure, database setup/teardown, mock utilities

## 🚀 Quick Start

### Prerequisites

1. **PostgreSQL Database** (primary test database):
   ```bash
   # Using Docker
   docker run --name go-ormx-postgres -e POSTGRES_PASSWORD=password -p 5432:5432 -d postgres:13
   
   # Or install locally
   brew install postgresql  # macOS
   sudo apt-get install postgresql  # Ubuntu
   ```

2. **MySQL Database** (optional, for multi-database tests):
   ```bash
   # Using Docker
   docker run --name go-ormx-mysql -e MYSQL_ROOT_PASSWORD=password -p 3306:3306 -d mysql:8.0
   ```

3. **Environment Variables**:
   ```bash
   export TEST_POSTGRES_HOST=localhost
   export TEST_POSTGRES_USER=postgres
   export TEST_POSTGRES_PASSWORD=password
   export TEST_POSTGRES_DB=go-ormx_test
   
   # Optional for MySQL tests
   export TEST_MYSQL_ENABLED=true
   export TEST_MYSQL_HOST=localhost
   export TEST_MYSQL_USER=root
   export TEST_MYSQL_PASSWORD=password
   export TEST_MYSQL_DB=go-ormx_test
   ```

### Running Tests

1. **All Integration Tests**:
   ```bash
   # From go-ormx root directory
   go test ./tests/integration/... -v
   
   # Or use the test runner
   ./tests/run_integration_tests.sh
   ```

2. **Specific Test Suites**:
   ```bash
   # Basic integration tests
   go test ./tests/integration/basic -v
   
   # Test utilities (no actual tests)
   go test ./tests/integration/utils -v
   ```

3. **Short Tests Only** (skip long-running tests):
   ```bash
   go test ./tests/integration/... -short
   ```

## 📊 Test Categories

### Functional Tests
- **Client Creation**: Database client initialization and configuration
- **User Repository**: CRUD operations with example User model
- **Migration Status**: Migration system functionality
- **Health Checks**: Database health monitoring and connection stats

### Test Utilities
- **Test Setup**: Common test infrastructure and helpers
- **Database Setup**: Database connection and cleanup utilities
- **Mock Utilities**: Test doubles for external dependencies

## 🧪 Test Examples

### Basic Integration Tests

```go
func TestBasicIntegration_ClientCreation(t *testing.T) {
    testConfig := &config.Config{
        Type:     config.PostgreSQL,
        Host:     "localhost",
        Port:     5432,
        Database: "test_db",
        Username: "postgres",
        Password: "password",
    }
    
    client, err := gormx.NewClient(gormx.Config{
        Database: testConfig,
        Options:  gormx.ClientOptions{},
    })
    
    if err != nil {
        t.Skipf("Skipping test: failed to create client (database may not be available): %v", err)
    }
    defer client.Close()
    
    // Test assertions...
}
```

### User Repository Tests

```go
func TestBasicIntegration_UserRepository(t *testing.T) {
    // Setup client and repository
    userRepo := examples.NewUserRepository(
        client.Database().DB(),
        logger,
        repositories.RepositoryOptions{
            BatchSize:      100,
            ValidateOnSave: true,
        },
    )
    
    // Test CRUD operations
    user := &models.User{
        Email:        "test@example.com",
        Username:     "testuser",
        FirstName:    "Test",
        LastName:     "User",
        PasswordHash: "hashed_password",
        Salt:         "salt",
        Status:       models.UserStatusActive,
        Role:         models.UserRoleUser,
    }
    user.SetTenantID(ulid.Make().String())
    
    err := userRepo.Create(ctx, user)
    if err != nil {
        t.Skipf("Skipping test: failed to create user (database may not be available): %v", err)
    }
    
    // Test assertions...
}
```

## 🔧 Test Configuration

### Environment Variables

```bash
# Required for database tests
TEST_POSTGRES_HOST=localhost
TEST_POSTGRES_PORT=5432
TEST_POSTGRES_USER=postgres
TEST_POSTGRES_PASSWORD=password
TEST_POSTGRES_DB=go-ormx_test

# Optional for MySQL tests
TEST_MYSQL_ENABLED=true
TEST_MYSQL_HOST=localhost
TEST_MYSQL_PORT=3306
TEST_MYSQL_USER=root
TEST_MYSQL_PASSWORD=password
TEST_MYSQL_DB=go-ormx_test
```

### Test Utilities

```go
// Skip tests if database is not available
utils.SkipIfShort(t)

// Create test logger
logger := utils.NewTestLogger(t, "integration-test")

// Get environment with defaults
host := utils.GetEnvWithDefault("TEST_POSTGRES_HOST", "localhost")
port := utils.GetEnvBool("TEST_SSL_ENABLED", false)
```

## 📈 Test Results

### Expected Behavior
- **With Database**: Tests run and verify actual database operations
- **Without Database**: Tests gracefully skip with informative messages
- **Short Mode**: Long-running tests are skipped

### Test Output Example
```
=== RUN   TestBasicIntegration_ClientCreation
    basic_integration_test.go:46: Skipping test: failed to create client (database may not be available): [DB_CONFIG_VALIDATION] invalid database configuration
--- SKIP: TestBasicIntegration_ClientCreation (0.00s)
PASS
```

## 🛠️ Troubleshooting

### Common Issues

**Tests skipping due to database connection**
```bash
# Start PostgreSQL container
docker run --name go-ormx-postgres -e POSTGRES_PASSWORD=password -p 5432:5432 -d postgres:13

# Wait for database to be ready
sleep 5

# Run tests
go test ./tests/integration/... -v
```

**Permission denied errors**
```bash
# Ensure test user has proper permissions
psql -h localhost -U postgres -c "CREATE DATABASE go_ormx_test;"
psql -h localhost -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE go_ormx_test TO postgres;"
```

**Port conflicts**
```bash
# Check if port is in use
lsof -i :5432

# Use different port
export TEST_POSTGRES_PORT=5433
```

## 📝 Notes

- Integration tests are designed to work with or without a database
- Tests gracefully skip when database is not available
- Focus on testing the current architecture with examples
- Tests use ULID-based IDs and current model structure
- Integration tests complement unit tests for end-to-end validation

## 🔄 Migration from v1.0

The integration test suite has been simplified and refactored to match the current architecture:

- **Removed**: Old integration tests for previous architecture
- **Added**: New basic integration tests for current functionality
- **Updated**: Test utilities to work with current models and repositories
- **Simplified**: Focus on core functionality rather than enterprise features

For more information, see the main project documentation and individual test files for specific examples and patterns.