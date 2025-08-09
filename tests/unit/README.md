# Unit Tests for Go-ORMX Package

This directory contains comprehensive unit tests for the Go-ORMX data access layer. The tests cover all major components with extensive edge cases and scenarios.

## Test Structure

```
tests/unit/
├── config/          # Configuration management tests
├── db/              # Database layer tests
├── errors/          # Error handling tests
├── logging/         # Structured logging tests  
├── models/          # Data model tests
├── repositories/    # Repository pattern tests
└── security/        # Security utilities tests
```

## Test Coverage

### 1. Config Package (`config/`)
- **config_test.go**: Configuration loading, validation, and environment variable handling
- **Coverage**: Environment loading, validation rules, DSN generation, default values
- **Edge Cases**: Invalid configurations, missing values, boundary conditions

### 2. DB Package (`db/`)
- **db_test.go**: Database connection and management
- **Coverage**: Connection pooling, health checks, connection stats
- **Edge Cases**: Connection failures, timeout scenarios, concurrent access

### 3. Errors Package (`errors/`)
- **errors_test.go**: Structured error handling
- **Coverage**: Error creation, wrapping, categorization, user messages
- **Edge Cases**: Error chaining, concurrent access, error transformation

### 4. Logging Package (`logging/`)
- **logging_test.go**: Structured logging with field support
- **Coverage**: Log field creation, GORM logger adapter, operation logging, context logging
- **Edge Cases**: Sensitive data masking, error conditions, concurrent access

### 5. Models Package (`models/`)
- **models_test.go**: Base model functionality
- **Coverage**: Model validation, hooks, timestamps, soft delete, ULID generation
- **Edge Cases**: Unicode data, boundary values, concurrent operations

### 6. Repositories Package (`repositories/`)
- **repositories_test.go**: Generic repository functionality
- **Coverage**: CRUD operations, filtering, ordering, batch operations
- **Edge Cases**: Invalid data, concurrent operations, large datasets

### 7. Security Package (`security/`)
- **security_test.go**: Security utilities and validation
- **Coverage**: Input validation, security context, struct validation
- **Edge Cases**: Unicode handling, large inputs, security edge cases

## Running Tests

### Run All Unit Tests
```bash
cd go-ormx
go test ./tests/unit/... -v
```

### Run Specific Package Tests
```bash
# Config tests
go test ./tests/unit/config -v

# Security tests  
go test ./tests/unit/security -v

# Models tests
go test ./tests/unit/models -v
```

### Run with Coverage
```bash
go test ./tests/unit/... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Run Benchmarks
```bash
go test ./tests/unit/... -bench=. -benchmem
```

## Test Features

### Comprehensive Coverage
- **Happy Path**: All normal operation scenarios
- **Error Conditions**: Invalid inputs, connection failures, constraint violations
- **Edge Cases**: Boundary values, unicode data, concurrent access
- **Performance**: Benchmark tests for critical operations

### Mock Infrastructure
- **Mock Logger**: Captures log output for verification
- **Mock Database**: In-memory SQLite for fast, isolated tests

### Key Testing Areas
- **ULID Generation**: Testing ULID-based ID generation in models
- **Repository Pattern**: Testing generic repository with type constraints
- **Error Handling**: Testing structured error categorization and wrapping
- **Configuration**: Testing environment-based configuration loading
- **Security**: Testing input validation and security context

## Test Data

### Sample Models
```go
type TestProduct struct {
    models.AuditModel
    Name        string  `gorm:"type:varchar(255);not null"`
    Description string  `gorm:"type:text"`
    Price       float64 `gorm:"type:decimal(10,2);not null"`
}
```

### Sample Repository
```go
type TestRepository struct {
    *repositories.BaseRepository[TestProduct]
}
```

## Notes

- All tests use the current architecture with ULID-based IDs
- Tests focus on core functionality without external dependencies
- Mock implementations are used for database and logging dependencies
- Tests are designed to be fast and reliable for CI/CD pipelines