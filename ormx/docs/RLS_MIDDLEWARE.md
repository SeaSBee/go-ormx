# RLS Middleware Examples

This guide shows how to inject a tenant ID into `context.Context` so PostgreSQL Row-Level Security (RLS) works seamlessly with go-ormx.

The go-ormx client reads the tenant from the context key `"tenant_id"`. Use `ormx.WithTenant(ctx, id)` to attach it and pass that context to all DB operations in a request.

Before using these middlewares, enable RLS:

```bash
export DB_RLS_ENABLED=true
export DB_RLS_TENANT_GUC=app.tenant_id          # optional, default: app.tenant_id
export DB_RLS_REQUIRE_TENANT=true               # optional, warn-only when missing
```

## net/http Middleware

```go
package middleware

import (
    "net/http"
    gormx "go-ormx"
)

const HeaderTenantID = "X-Tenant-ID"

func TenantMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        tenantID := r.Header.Get(HeaderTenantID)
        ctx := gormx.WithTenant(r.Context(), tenantID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

Usage:

```go
mux := http.NewServeMux()
mux.Handle("/api", TenantMiddleware(apiHandler))
```

## Gin Middleware

```go
package middleware

import (
    "github.com/gin-gonic/gin"
    gormx "go-ormx"
)

const HeaderTenantID = "X-Tenant-ID"

func GinTenantMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        tenantID := c.GetHeader(HeaderTenantID)
        ctx := gormx.WithTenant(c.Request.Context(), tenantID)
        c.Request = c.Request.WithContext(ctx)
        c.Next()
    }
}
```

Usage:

```go
r := gin.Default()
r.Use(GinTenantMiddleware())

r.GET("/health", func(c *gin.Context) {
    // Pass c.Request.Context() to DB operations
    if err := client.Health(c.Request.Context()); err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, gin.H{"status": "ok"})
})

// Example: Repository usage with tenant-scoped context
type Product struct {
    models.TenantAuditModel
    Name string `gorm:"type:varchar(255);not null"`
}

type ProductRepo struct { *repositories.BaseRepository[Product] }

func NewProductRepo(db *gorm.DB, logger logging.Logger) *ProductRepo {
    return &ProductRepo{ BaseRepository: repositories.NewBaseRepository[Product](db, logger, repositories.DefaultRepositoryOptions()) }
}

productRepo := NewProductRepo(client.Database().DB(), logger)

r.GET("/products", func(c *gin.Context) {
    // tenant_id already injected by middleware
    ctx := c.Request.Context()
    filter := repositories.Filter{
        Where: map[string]repositories.WhereCondition{
            "name": {Operator: "like", Value: "%widget%"},
        },
        Limit: 50,
    }
    products, err := productRepo.Find(ctx, filter)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, products)
})
```

## Echo Middleware

```go
package middleware

import (
    "github.com/labstack/echo/v4"
    gormx "go-ormx"
)

const HeaderTenantID = "X-Tenant-ID"

func EchoTenantMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        tenantID := c.Request().Header.Get(HeaderTenantID)
        ctx := gormx.WithTenant(c.Request().Context(), tenantID)
        req := c.Request().WithContext(ctx)
        c.SetRequest(req)
        return next(c)
    }
}
```

Usage:

```go
e := echo.New()
e.Use(EchoTenantMiddleware)

e.GET("/health", func(c echo.Context) error {
    if err := client.Health(c.Request().Context()); err != nil {
        return c.JSON(500, map[string]string{"error": err.Error()})
    }
    return c.JSON(200, map[string]string{"status": "ok"})
})
```

## Chi Middleware

```go
package middleware

import (
    "net/http"
    gormx "go-ormx"
)

const HeaderTenantID = "X-Tenant-ID"

func ChiTenantMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        tenantID := r.Header.Get(HeaderTenantID)
        ctx := gormx.WithTenant(r.Context(), tenantID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

## gRPC Unary Interceptor

```go
import (
    "context"
    "google.golang.org/grpc"
    "google.golang.org/grpc/metadata"
    gormx "go-ormx"
)

const HeaderTenantID = "x-tenant-id"

func TenantUnaryServerInterceptor() grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (interface{}, error) {
        if md, ok := metadata.FromIncomingContext(ctx); ok {
            if vals := md.Get(HeaderTenantID); len(vals) > 0 {
                ctx = gormx.WithTenant(ctx, vals[0])
            }
        }
        return handler(ctx, req)
    }
}
```

## Usage Notes

- Always pass the request-scoped context (with tenant) to repository or client calls:

```go
ctx := c.Request().Context() // framework context
// now call DB ops
_ = client.Health(ctx)
// or repo.Find(ctx, filter)
```

- If `DB_RLS_REQUIRE_TENANT=true` and a request is missing the tenant, go-ormx will log a warning before each operation. Database-side enforcement still depends on your RLS policies.

- Combine with application-side tenant filters if your models include `tenant_id` (repositories already add a tenant filter when present).
