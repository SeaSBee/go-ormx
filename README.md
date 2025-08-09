# Go-ORMX

A production-ready, open-source GORM extension for Go applications with comprehensive database management, migrations, and enterprise features.

## 🚀 Features

### Core Functionality
- **Database Management**: Multi-database support (PostgreSQL, MySQL)
- **Migration System**: File-based migrations using `golang-migrate/migrate`
- **Repository Pattern**: Generic repository with comprehensive CRUD operations
- **Error Handling**: Structured error handling with error codes and categorization
- **Logging**: Structured logging with context support
- **Configuration**: Environment-based configuration management
- **Health Checks**: Database health monitoring
- **Security**: Input validation and security utilities
- **Transactions**: ACID transaction support
- **Connection Pooling**: Optimized connection management

### Enterprise Features
- **Multi-tenancy**: Built-in tenant support
- **Audit Trail**: Comprehensive audit logging
- **Soft Deletes**: Safe data deletion with recovery
- **Validation**: Struct validation with custom rules
- **Batch Operations**: Efficient bulk operations
- **ULID Support**: Universally unique lexicographically sortable identifiers

## 📦 Installation

```bash
go get github.com/your-org/go-ormx
```

## 🏗️ Architecture

### Core Module (`go-ormx`)
The core module provides the foundation for database operations:

```
go-ormx/
├── db/                 # Database connection and management
├── errors/            # Error handling and categorization
├── internal/          # Internal packages
│   ├── config/       # Configuration management
│   ├── logging/      # Structured logging
│   └── security/     # Security utilities
├── migrations/        # Migration system (golang-migrate)
├── models/           # Base models and interfaces
├── repositories/     # Repository pattern implementation
└── gorm.go          # Main client interface
```

### Examples (`examples/`)
The examples demonstrate how to use the core functionality:

```
examples/
├── models/           # Example models (User, Tenant)
├── repositories/     # Example repositories
├── migrations/       # Example migration files
│   ├── postgresql/   # PostgreSQL migrations
│   └── mysql/        # MySQL migrations
├── app.go           # Complete example application
├── basic_usage.go   # Basic usage examples
└── migration_example.go  # Migration examples
```

## 🚀 Quick Start

### 1. Basic Setup

```go
package main

import (
    "context"
    "log"
    
    gormx "go-ormx"
    "go-ormx/ormx/config"
)

func main() {
    // Create configuration
    cfg := gormx.Config{
        Database: &config.Config{
        Type:     config.PostgreSQL,
        Host:     "localhost",
        Port:     5432,
        Database: "myapp",
        Username: "postgres",
        Password: "password",
        },
        Options: gormx.ClientOptions{
            EnableMigrations: true,
            AutoMigrate:      true,
        },
    }
    
    // Create client
    client, err := gormx.NewClient(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // Run migrations
    ctx := context.Background()
    if err := client.Migrate(ctx); err != nil {
        log.Fatal(err)
    }

    fmt.Println("Database setup complete!")
}
```

### 2. Using Base Models

```go
import (
    "go-ormx/ormx/models"
    "time"
)

// Create a basic model
type Product struct {
    models.BaseModel
    Name        string  `gorm:"type:varchar(255);not null" json:"name"`
    Description string  `gorm:"type:text" json:"description"`
    Price       float64 `gorm:"type:decimal(10,2);not null" json:"price"`
}

// Create a tenant-aware model
type Order struct {
    models.TenantAuditModel
    ProductID string    `gorm:"type:varchar(26);not null" json:"product_id"`
    Quantity  int       `gorm:"not null" json:"quantity"`
    OrderDate time.Time `gorm:"not null" json:"order_date"`
}

// Set tenant ID for tenant-aware models
order := &Order{
    ProductID: "01HXYZ123456789ABCDEFGHIJKL",
    Quantity:  5,
    OrderDate: time.Now(),
}
order.SetTenantID("01HXYZ123456789ABCDEFGHIJKM")
```

### 3. Using Repository Pattern

```go
import (
    "context"
    "go-ormx/ormx/repositories"
    "go-ormx/examples/repositories"
)

// Create repository for your custom model
type ProductRepository struct {
    *repositories.BaseRepository[Product]
}

func NewProductRepository(db *gorm.DB, logger logging.Logger, opts repositories.RepositoryOptions) *ProductRepository {
    return &ProductRepository{
        BaseRepository: repositories.NewBaseRepository[Product](db, logger, opts),
    }
}

// Usage
repo := NewProductRepository(
    client.Database().DB(),
    logger,
    repositories.RepositoryOptions{
        BatchSize:      100,
        ValidateOnSave: true,
    },
)

// CRUD operations
ctx := context.Background()

// Create
product := &Product{
    Name:        "Sample Product",
    Description: "A sample product",
    Price:       29.99,
}
err := repo.Create(ctx, product)

// Read
foundProduct, err := repo.GetByID(ctx, product.ID)

// Update
foundProduct.Price = 39.99
err = repo.Update(ctx, foundProduct)

// Delete
err = repo.Delete(ctx, product.ID)

// Find with filters
filter := repositories.Filter{
    Where: map[string]repositories.WhereCondition{
        "price": {Operator: "gte", Value: 20.0},
    },
    OrderBy: []repositories.OrderBy{
        {Field: "created_at", Direction: "DESC"},
    },
    Limit: 10,
}
products, err := repo.Find(ctx, filter)
```

### 4. Multi-tenancy and PostgreSQL Row-Level Security (RLS)

Enable RLS via env and scope operations using context:

```go
// Environment
os.Setenv("DB_RLS_ENABLED", "true")              // turn on RLS handling
os.Setenv("DB_RLS_TENANT_GUC", "app.tenant_id")  // optional, defaults to app.tenant_id
os.Setenv("DB_RLS_REQUIRE_TENANT", "true")       // optional, warn when tenant missing

// Client
cfg, _ := config.LoadFromEnv()
client, _ := gormx.NewClient(gormx.Config{ Database: cfg, Logger: logger, Options: gormx.DefaultClientOptions() })
defer client.Close()

// Scope a request to a tenant
ctx := gormx.WithTenant(context.Background(), "tenant-123")
_ = client.Health(ctx) // callbacks set the GUC per query when using PostgreSQL
```

See `docs/RLS_MIDDLEWARE.md` for framework middleware examples (net/http, Gin, Echo, Chi, gRPC) to automatically inject tenant IDs into request contexts.

## 📋 Migration System

### Migration Files Structure

```
examples/migrations/
├── postgresql/
│   ├── 000001_create_users_table.up.sql
│   ├── 000001_create_users_table.down.sql
│   ├── 000002_create_user_sessions_table.up.sql
│   └── 000002_create_user_sessions_table.down.sql
└── mysql/
    ├── 000001_create_users_table.up.sql
    ├── 000001_create_users_table.down.sql
    ├── 000002_create_user_sessions_table.up.sql
    └── 000002_create_user_sessions_table.down.sql
```

### Migration Commands

```go
// Run all pending migrations
err := client.Migrate(ctx)

// Migrate to specific version
err := client.MigrateTo(ctx, 3)

// Rollback last migration
err := client.Rollback(ctx)

// Rollback to specific version
err := client.RollbackTo(ctx, 1)

// Get migration status
status, err := client.GetMigrationStatus(ctx)
fmt.Printf("Version: %d, Dirty: %t\n", status.Version, status.Dirty)

// Force migration version (for recovery)
err := client.Force(ctx, 5)

// Drop all tables (dangerous!)
err := client.Drop(ctx)
```

## 🔧 Configuration

### Environment Variables

```bash
# Database Configuration
DB_TYPE=postgresql
DB_HOST=localhost
DB_PORT=5432
DB_NAME=myapp
DB_USERNAME=postgres
DB_PASSWORD=password

# Connection Pool
DB_MAX_OPEN_CONNS=10
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m
DB_CONN_MAX_IDLE_TIME=1m

# Features
DB_ENABLE_MIGRATIONS=true
DB_AUTO_MIGRATE=true
DB_ENABLE_HEALTH=true
```

### Configuration Struct

```go
type Config struct {
    Database *config.Config
    Options  ClientOptions
}

type ClientOptions struct {
    EnableMigrations bool
    AutoMigrate      bool
    EnableHealth     bool
}
```

## 🛡️ Error Handling

```go
import "go-ormx/ormx/errors"

// Check error types
if errors.IsConnectionError(err) {
    // Handle connection errors
}

if errors.IsDataError(err) {
    // Handle data errors
}

if errors.IsRetryable(err) {
    // Retry the operation
}

// Get error details
errorCode := errors.GetErrorCode(err)
userMessage := errors.GetUserMessage(err)
```

## 📊 Monitoring

### Health Checks

```go
// Check database health
healthy := client.IsHealthy(ctx)
if !healthy {
    log.Println("Database is unhealthy")
}

// Get connection stats
stats := client.GetConnectionStats()
fmt.Printf("Open connections: %d\n", stats.OpenConnections)
fmt.Printf("In use connections: %d\n", stats.InUseConnections)
fmt.Printf("Idle connections: %d\n", stats.IdleConnections)
```

## 🔒 Security

### Input Validation

```go
import "go-ormx/ormx/security"

// Validate struct
result := security.ValidateStruct(ctx, user)
if !result.Valid {
    for _, err := range result.Errors {
        log.Printf("Validation error: %s", err)
    }
}
```

### Security Context

```go
// Create security context
secCtx := security.NewContext("user123", "tenant456")
ctx := security.WithContext(context.Background(), secCtx)

// Get security info from context
userID := security.GetUserIDFromContext(ctx)
tenantID := security.GetTenantIDFromContext(ctx)
```

## 🧪 Testing

### Unit Tests

```bash
# Run all unit tests
go test ./tests/unit/... -v

# Run specific test package
go test ./tests/unit/repositories/... -v
```

### Integration Tests

```bash
# Run integration tests
go test ./tests/integration/... -v

# Run with database
DB_HOST=localhost go test ./tests/integration/... -v
```

## 📚 Examples

### Complete Example Application

See `examples/app.go` for a comprehensive example that demonstrates:

- Database setup and configuration
- Migration management
- Model creation and usage
- Repository operations
- Transaction handling
- Error handling
- Logging

### Running Examples

```bash
# Run the example application
go run examples/app.go

# Run specific examples
go run examples/basic_usage.go
go run examples/migration_example.go
```

## 🧪 Comprehensive Test Report

See `TEST_REPORT.md` for the full, up-to-date test summary, benchmarks, coverage notes, and quality assessment.

Quick summary:

- Unit tests: PASS
- Integration tests: SKIPPED by default (require DB); guarded via env checks
- Benchmarks: Completed (config, logging, errors, models, repositories, security)
- Coverage: 0.0% shown due to external `tests/...` layout; tests exercise core features without contaminating library packages

## 🔄 Key Changes in v2.0

### Open Source Refactoring
- **Core/Examples Separation**: Core functionality is now in the main module, while user-specific examples are in the `examples/` directory
- **Standard ULID**: Replaced custom UUIDv7 with `github.com/oklog/ulid/v2` for better compatibility
- **Migration System**: Integrated `golang-migrate/migrate` for production-grade SQL migrations
- **Clean Architecture**: Removed enterprise-specific features from core module

### Breaking Changes
- Module path changed to `go-ormx`
- User models and repositories moved to `examples/`
- UUIDv7 replaced with ULID
- Migration system completely rewritten
- Repository interfaces simplified

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

- **Documentation**: [docs/](docs/)
- **Issues**: [GitHub Issues](https://github.com/your-org/go-ormx/issues)
- **Discussions**: [GitHub Discussions](https://github.com/your-org/go-ormx/discussions)

## 🔄 Changelog

See [CHANGELOG.md](CHANGELOG.md) for a list of changes and version history.

---

**Go-ORMX** - Production-ready, open-source GORM extension for Go applications