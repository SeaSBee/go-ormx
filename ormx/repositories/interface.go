// Package repositories provides repository interfaces and implementations
// for data access layer with comprehensive CRUD operations, transactions, and caching.
package repositories

import (
	"context"
	"time"

	"go-ormx/ormx/models"

	"gorm.io/gorm"
)

// Repository provides the base interface for all repository operations
type Repository[T models.Modelable] interface {
	// Basic CRUD operations
	Create(ctx context.Context, entity T) error
	CreateBatch(ctx context.Context, entities []T) error
	GetByID(ctx context.Context, id string) (T, error)
	Update(ctx context.Context, entity T) error
	UpdatePartial(ctx context.Context, id string, updates map[string]interface{}) error
	Delete(ctx context.Context, id string) error
	SoftDelete(ctx context.Context, id string) error

	// Query operations
	Find(ctx context.Context, filter Filter) ([]T, error)
	FindOne(ctx context.Context, filter Filter) (T, error)
	Count(ctx context.Context, filter Filter) (int64, error)
	Exists(ctx context.Context, id string) (bool, error)

	// Transaction support
	WithTx(tx *gorm.DB) Repository[T]
	BeginTx(ctx context.Context) (*gorm.DB, error)
	CommitTx(tx *gorm.DB) error
	RollbackTx(tx *gorm.DB) error

	// Advanced querying
	FindByIDs(ctx context.Context, ids []string) ([]T, error)
	FindRaw(ctx context.Context, query string, args ...interface{}) ([]T, error)
	ExecuteRaw(ctx context.Context, query string, args ...interface{}) error
}

// TenantRepository extends Repository for multi-tenant entities
type TenantRepository[T models.Tenantable] interface {
	Repository[T]

	// Tenant-specific operations
	CreateForTenant(ctx context.Context, tenantID string, entity T) error
	GetByIDForTenant(ctx context.Context, tenantID, id string) (T, error)
	FindForTenant(ctx context.Context, tenantID string, filter Filter) ([]T, error)
	CountForTenant(ctx context.Context, tenantID string, filter Filter) (int64, error)
	UpdateForTenant(ctx context.Context, tenantID, id string, entity T) error
	DeleteForTenant(ctx context.Context, tenantID, id string) error
	ExistsForTenant(ctx context.Context, tenantID, id string) (bool, error)
}

// Filter represents query filters
type Filter struct {
	// Basic filters
	Where map[string]WhereCondition `json:"where,omitempty"`

	// Ordering
	OrderBy []OrderBy `json:"order_by,omitempty"`

	// Pagination
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`

	// Scopes
	Scopes []models.Scope `json:"scopes,omitempty"`
}

// WhereCondition represents a where condition
type WhereCondition struct {
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// OrderBy represents ordering
type OrderBy struct {
	Field     string `json:"field"`
	Direction string `json:"direction"` // "ASC" or "DESC"
}

// RepositoryOptions configures repository behavior
type RepositoryOptions struct {
	// Performance
	BatchSize int `json:"batch_size"`

	// Validation
	ValidateOnSave bool `json:"validate_on_save"`
}

// DefaultRepositoryOptions returns default repository options
func DefaultRepositoryOptions() RepositoryOptions {
	return RepositoryOptions{
		BatchSize:      100,
		ValidateOnSave: true,
	}
}

// QueryBuilder provides a fluent interface for building queries
type QueryBuilder[T models.Modelable] interface {
	Where(field string, value interface{}) QueryBuilder[T]
	OrderBy(field string, desc bool) QueryBuilder[T]
	Limit(limit int) QueryBuilder[T]
	Offset(offset int) QueryBuilder[T]
	Build() Filter
	Execute(ctx context.Context) ([]T, error)
	ExecuteOne(ctx context.Context) (T, error)
	Count(ctx context.Context) (int64, error)
}

// TransactionManager provides transaction management
type TransactionManager interface {
	RunInTransaction(ctx context.Context, fn func(ctx context.Context, tx *gorm.DB) error) error
	BeginTransaction(ctx context.Context) (*gorm.DB, error)
	CommitTransaction(tx *gorm.DB) error
	RollbackTransaction(tx *gorm.DB) error
}

// CacheManager provides caching functionality
type CacheManager interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, key string) (bool, error)
	Clear(ctx context.Context, pattern string) error
}

// EventPublisher provides event publishing functionality
type EventPublisher interface {
	PublishCreated(ctx context.Context, entity interface{}) error
	PublishUpdated(ctx context.Context, entity interface{}, changes map[string]interface{}) error
	PublishDeleted(ctx context.Context, entity interface{}) error
}

// MetricsCollector provides metrics collection
type MetricsCollector interface {
	RecordOperation(operation string, duration time.Duration, success bool)
	RecordQueryCount(count int64)
	RecordCacheHit(hit bool)
	RecordBatchSize(size int)
	IncrementError(operation string, errorType string)
}

// RepositoryFactory creates repository instances
type RepositoryFactory interface {
	Create(model interface{}, options RepositoryOptions) interface{}
	CreateWithCache(model interface{}, cache CacheManager, options RepositoryOptions) interface{}
	CreateWithEvents(model interface{}, events EventPublisher, options RepositoryOptions) interface{}
	CreateWithMetrics(model interface{}, metrics MetricsCollector, options RepositoryOptions) interface{}
}

// HealthChecker provides health checking functionality
type HealthChecker interface {
	CheckHealth(ctx context.Context) error
	GetHealthStatus() map[string]interface{}
}

// SchemaManager provides schema management functionality
type SchemaManager interface {
	CreateTable(ctx context.Context, model interface{}) error
	DropTable(ctx context.Context, model interface{}) error
	CreateIndex(ctx context.Context, model interface{}, indexName string, fields []string) error
	DropIndex(ctx context.Context, model interface{}, indexName string) error
	GetTableInfo(ctx context.Context, tableName string) (map[string]interface{}, error)
	ValidateSchema(ctx context.Context, model interface{}) error
}
