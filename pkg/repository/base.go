package repository

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/seasbee/go-ormx/pkg/errors"
	"github.com/seasbee/go-ormx/pkg/logging"
	"github.com/seasbee/go-validatorx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository represents a generic repository interface
type Repository[T any] interface {
	// Basic CRUD operations
	Create(ctx context.Context, entity *T) error
	CreateInBatches(ctx context.Context, entities []T, batchSize int) error

	FindFirstByID(ctx context.Context, id uuid.UUID) (*T, error)
	FindFirstByConditions(ctx context.Context, dest *T, conds ...interface{}) error
	FirstOrInitByConditions(ctx context.Context, dest *T, conds ...interface{}) error

	FindAllWithOffset(ctx context.Context, limit int, offset int, dest *[]T) error
	FindAllInBatchesWithOffset(ctx context.Context, limit int, offset int, dest *[]T, batchSize int, fc func(tx *gorm.DB, batch int) error) error
	FindAllByConditionsWithOffset(ctx context.Context, limit int, offset int, dest *[]T, conds ...interface{}) error
	FindAllInBatchesByConditionsWithOffset(ctx context.Context, limit int, offset int, dest *[]T, batchSize int, fc func(tx *gorm.DB, batch int) error, conds ...interface{}) error

	FindAllWithCursor(ctx context.Context, cursor string, limit int, direction string, dest *[]T) error
	FindAllInBatchesWithCursor(ctx context.Context, cursor string, limit int, direction string, dest *[]T, batchSize int, fc func(tx *gorm.DB, batch int) error) error
	FindAllByConditionsWithCursor(ctx context.Context, cursor string, limit int, direction string, dest *[]T, conds ...interface{}) error
	FindAllInBatchesByConditionsWithCursor(ctx context.Context, cursor string, limit int, direction string, dest *[]T, batchSize int, fc func(tx *gorm.DB, batch int) error, conds ...interface{}) error

	Update(ctx context.Context, entity *T) error
	UpdateByID(ctx context.Context, entity *T, id uuid.UUID) error
	UpdateByConditions(ctx context.Context, entity *T, conds ...interface{}) error

	Upsert(ctx context.Context, entity *T, conflictClause string) error
	UpsertByID(ctx context.Context, entity *T, id uuid.UUID, conflictClause string) error
	UpsertByConditions(ctx context.Context, entity *T, conflictClause string, conds ...interface{}) error
	UpsertInBatches(ctx context.Context, entities []T, batchSize int, conflictClause string) error
	UpsertInBatchesByConditions(ctx context.Context, entities []T, batchSize int, conflictClause string, conds ...interface{}) error

	DeleteByID(ctx context.Context, id uuid.UUID) error
	DeleteByConditions(ctx context.Context, entity *T, conds ...interface{}) error
	DeleteInBatches(ctx context.Context, entities []T, batchSize int) error
	DeleteInBatchesByConditions(ctx context.Context, entities []T, batchSize int, conds ...interface{}) error

	ExistsByID(ctx context.Context, id uuid.UUID) (bool, error)
	ExistsByConditions(ctx context.Context, conds ...interface{}) (bool, error)
	CountByConditions(ctx context.Context, conds ...interface{}) (int64, error)

	TakeByConditions(ctx context.Context, dest *T, conds ...interface{}) error
	LastByConditions(ctx context.Context, dest *T, conds ...interface{}) error

	Begin(ctx context.Context) (*gorm.DB, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error

	// Advanced operations
	WithTransaction(ctx context.Context, fn func(Repository[T]) error) error
}

// RepositoryConfig represents repository configuration
type RepositoryConfig struct {
	EnableValidation bool `json:"enable_validation"`
	EnableMetrics    bool `json:"enable_metrics"`
	DefaultLimit     int  `json:"default_limit"`
	MaxLimit         int  `json:"max_limit"`
}

// DefaultRepositoryConfig returns default repository configuration
func DefaultRepositoryConfig() *RepositoryConfig {
	return &RepositoryConfig{
		EnableValidation: true,
		EnableMetrics:    true,
		DefaultLimit:     20,
		MaxLimit:         1000,
	}
}

// RepositoryMetrics represents repository metrics
type RepositoryMetrics struct {
	TotalOperations      int64         `json:"total_operations"`
	SuccessfulOperations int64         `json:"successful_operations"`
	FailedOperations     int64         `json:"failed_operations"`
	AverageQueryTime     time.Duration `json:"average_query_time"`
	LastReset            time.Time     `json:"last_reset"`
	mu                   sync.RWMutex
}

// NewRepositoryMetrics creates new repository metrics
func NewRepositoryMetrics() *RepositoryMetrics {
	return &RepositoryMetrics{
		LastReset: time.Now(),
	}
}

// IncrementOperations increments operation counters
func (rm *RepositoryMetrics) IncrementOperations(success bool) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.TotalOperations++
	if success {
		rm.SuccessfulOperations++
	} else {
		rm.FailedOperations++
	}
}

// RecordQueryTime records query execution time
func (rm *RepositoryMetrics) RecordQueryTime(duration time.Duration) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.AverageQueryTime == 0 {
		rm.AverageQueryTime = duration
	} else {
		rm.AverageQueryTime = (rm.AverageQueryTime + duration) / 2
	}
}

// Reset resets all metrics
func (rm *RepositoryMetrics) Reset() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.TotalOperations = 0
	rm.SuccessfulOperations = 0
	rm.FailedOperations = 0
	rm.AverageQueryTime = 0
	rm.LastReset = time.Now()
}

// GetSuccessRate returns operation success rate
func (rm *RepositoryMetrics) GetSuccessRate() float64 {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if rm.TotalOperations == 0 {
		return 0
	}
	return float64(rm.SuccessfulOperations) / float64(rm.TotalOperations)
}

// BaseRepository provides a base implementation of the Repository interface
type BaseRepository[T any] struct {
	db        *gorm.DB
	logger    logging.Logger
	config    *RepositoryConfig
	metrics   *RepositoryMetrics
	tableName string
	modelType reflect.Type
}

// NewBaseRepository creates a new base repository
func NewBaseRepository[T any](db *gorm.DB, logger logging.Logger, config *RepositoryConfig) *BaseRepository[T] {
	if config == nil {
		config = DefaultRepositoryConfig()
	}

	var entity T
	modelType := reflect.TypeOf(entity)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	tableName := getTableName(entity)

	return &BaseRepository[T]{
		db:        db,
		logger:    logger,
		config:    config,
		metrics:   NewRepositoryMetrics(),
		tableName: tableName,
		modelType: modelType,
	}
}

// getTableName extracts table name from entity
func getTableName(entity interface{}) string {
	// Try to get table name from GORM model
	if tabler, ok := entity.(interface{ TableName() string }); ok {
		return tabler.TableName()
	}

	// Fallback to type name
	t := reflect.TypeOf(entity)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}

// Create creates a new entity
func (r *BaseRepository[T]) Create(ctx context.Context, entity *T) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	// Validate entity if enabled
	if r.config.EnableValidation {
		if result, err := r.Validate(ctx, entity); err != nil {
			r.metrics.IncrementOperations(false)
			return fmt.Errorf("validation failed: %w", err)
		} else if !result.Valid {
			r.metrics.IncrementOperations(false)
			return errors.New(errors.ErrorTypeValidation, fmt.Sprintf("validation failed: %v", result.Errors))
		}
	}

	// Create entity
	if err := r.db.WithContext(ctx).Create(entity).Error; err != nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("failed to create entity: %w", err)
	}

	r.metrics.IncrementOperations(true)
	r.logger.Info(ctx, "Entity created successfully",
		logging.String("table", r.tableName),
		logging.String("id", r.getEntityID(entity).String()))

	return nil
}

// CreateInBatches creates multiple entities in batches
func (r *BaseRepository[T]) CreateInBatches(ctx context.Context, entities []T, batchSize int) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	// Validate entity if enabled
	if r.config.EnableValidation {
		if entities == nil {
			r.metrics.IncrementOperations(false)
			return fmt.Errorf("entities cannot be nil")
		}

		if len(entities) == 0 {
			r.metrics.IncrementOperations(false)
			return fmt.Errorf("entities cannot be empty")
		}

		// Validate batch size
		if batchSize <= 0 {
			r.metrics.IncrementOperations(false)
			return fmt.Errorf("batch size must be greater than 0, got %d", batchSize)
		}

		// Validate each entity in the batch
		for i, entity := range entities {
			if result, err := r.Validate(ctx, &entity); err != nil {
				r.metrics.IncrementOperations(false)
				return fmt.Errorf("validation failed for entity %d: %w", i, err)
			} else if !result.Valid {
				r.metrics.IncrementOperations(false)
				return errors.New(errors.ErrorTypeValidation,
					fmt.Sprintf("validation failed for entity %d: %v", i, result.Errors))
			}
		}
	}

	// Create entities
	if err := r.db.WithContext(ctx).CreateInBatches(entities, batchSize).Error; err != nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("failed to create entities: %w", err)
	}

	r.metrics.IncrementOperations(true)
	r.logger.Info(ctx, "Entities created successfully",
		logging.String("table", r.tableName),
		logging.Int("batch_size", batchSize),
		logging.Int("total_entities", len(entities)))

	return nil
}

// FindFirstByID finds entity by ID
func (r *BaseRepository[T]) FindFirstByID(ctx context.Context, id uuid.UUID) (*T, error) {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	var entity T
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&entity).Error; err != nil {
		r.metrics.IncrementOperations(false)
		return nil, fmt.Errorf("failed to find entity by ID: %w", err)
	}

	r.metrics.IncrementOperations(true)
	return &entity, nil
}

// FindFirstByConditions finds first entity by conditions
func (r *BaseRepository[T]) FindFirstByConditions(ctx context.Context, dest *T, conds ...interface{}) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	var err error
	if len(conds) == 0 {
		err = r.db.WithContext(ctx).First(dest).Error
	} else {
		err = r.db.WithContext(ctx).Where(conds[0], conds[1:]...).First(dest).Error
	}

	if err != nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("failed to find entity by conditions: %w", err)
	}

	r.metrics.IncrementOperations(true)
	return nil
}

// FirstOrInitByConditions finds first entity by conditions or initializes a new entity
func (r *BaseRepository[T]) FirstOrInitByConditions(ctx context.Context, dest *T, conds ...interface{}) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	var err error
	if len(conds) == 0 {
		err = r.db.WithContext(ctx).FirstOrInit(dest).Error
	} else {
		err = r.db.WithContext(ctx).Where(conds[0], conds[1:]...).FirstOrInit(dest).Error
	}

	if err != nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("failed to find entity by conditions: %w", err)
	}

	r.metrics.IncrementOperations(true)
	return nil
}

// FindAllWithOffset finds all entities
func (r *BaseRepository[T]) FindAllWithOffset(ctx context.Context, limit int, offset int, dest *[]T) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	limit, offset = r.validateOffsetPaginationParams(limit, offset)

	if err := r.db.WithContext(ctx).Limit(limit).Offset(offset).Find(dest).Error; err != nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("failed to find all entities: %w", err)
	}

	r.metrics.IncrementOperations(true)
	return nil
}

// FindAllInBatchesWithOffset finds all entities in batches
func (r *BaseRepository[T]) FindAllInBatchesWithOffset(ctx context.Context, limit int, offset int, dest *[]T, batchSize int, fc func(tx *gorm.DB, batch int) error) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	limit, offset = r.validateOffsetPaginationParams(limit, offset)

	// Validate batch size
	if batchSize <= 0 {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("batch size must be greater than 0, got %d", batchSize)
	}

	if err := r.db.WithContext(ctx).Limit(limit).Offset(offset).FindInBatches(dest, batchSize, fc).Error; err != nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("failed to find all entities in batches: %w", err)
	}

	r.metrics.IncrementOperations(true)
	return nil
}

// FindAllByConditionsWithOffset finds all entities by conditions
func (r *BaseRepository[T]) FindAllByConditionsWithOffset(ctx context.Context, limit int, offset int, dest *[]T, conds ...interface{}) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	limit, offset = r.validateOffsetPaginationParams(limit, offset)

	var err error
	if len(conds) == 0 {
		err = r.db.WithContext(ctx).Limit(limit).Offset(offset).Find(dest).Error
	} else {
		err = r.db.WithContext(ctx).Limit(limit).Offset(offset).Where(conds[0], conds[1:]...).Find(dest).Error
	}

	if err != nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("failed to find all entities by conditions: %w", err)
	}

	r.metrics.IncrementOperations(true)
	return nil
}

// FindAllInBatchesByConditionsWithOffset finds all entities in batches by conditions
func (r *BaseRepository[T]) FindAllInBatchesByConditionsWithOffset(ctx context.Context, limit int, offset int, dest *[]T, batchSize int, fc func(tx *gorm.DB, batch int) error, conds ...interface{}) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	limit, offset = r.validateOffsetPaginationParams(limit, offset)

	// Validate batch size
	if batchSize <= 0 {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("batch size must be greater than 0, got %d", batchSize)
	}

	var err error
	if len(conds) == 0 {
		err = r.db.WithContext(ctx).Limit(limit).Offset(offset).FindInBatches(dest, batchSize, fc).Error
	} else {
		err = r.db.WithContext(ctx).Limit(limit).Offset(offset).Where(conds[0], conds[1:]...).FindInBatches(dest, batchSize, fc).Error
	}

	if err != nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("failed to find all entities in batches by conditions: %w", err)
	}

	r.metrics.IncrementOperations(true)
	return nil
}

// FindAllWithCursor finds all entities
func (r *BaseRepository[T]) FindAllWithCursor(ctx context.Context, cursor string, limit int, direction string, dest *[]T) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	cursor, limit, direction = r.validateCursorPaginationParams(cursor, limit, direction)

	// Build query with cursor
	query := r.db.WithContext(ctx).Limit(limit)

	if cursor != "" {
		// For cursor-based pagination, we need to know the cursor field
		// This is a simplified implementation - in practice, you'd need to specify the cursor field
		if direction == "next" {
			query = query.Where("id > ?", cursor)
		} else {
			query = query.Where("id < ?", cursor)
		}
	}

	if err := query.Find(dest).Error; err != nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("failed to find all entities: %w", err)
	}

	r.metrics.IncrementOperations(true)
	return nil
}

// FindAllInBatchesWithCursor finds all entities in batches
func (r *BaseRepository[T]) FindAllInBatchesWithCursor(ctx context.Context, cursor string, limit int, direction string, dest *[]T, batchSize int, fc func(tx *gorm.DB, batch int) error) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	cursor, limit, direction = r.validateCursorPaginationParams(cursor, limit, direction)

	// Build query with cursor
	query := r.db.WithContext(ctx).Limit(limit)

	if cursor != "" {
		// For cursor-based pagination, we need to know the cursor field
		// This is a simplified implementation - in practice, you'd need to specify the cursor field
		if direction == "next" {
			query = query.Where("id > ?", cursor)
		} else {
			query = query.Where("id < ?", cursor)
		}
	}

	// Validate batch size
	if batchSize <= 0 {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("batch size must be greater than 0, got %d", batchSize)
	}

	if err := query.FindInBatches(dest, batchSize, fc).Error; err != nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("failed to find all entities in batches: %w", err)
	}

	r.metrics.IncrementOperations(true)
	return nil
}

// FindAllByConditionsWithCursor finds all entities by conditions
func (r *BaseRepository[T]) FindAllByConditionsWithCursor(ctx context.Context, cursor string, limit int, direction string, dest *[]T, conds ...interface{}) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	cursor, limit, direction = r.validateCursorPaginationParams(cursor, limit, direction)

	// Build query with cursor
	query := r.db.WithContext(ctx).Limit(limit)

	if cursor != "" {
		// For cursor-based pagination, we need to know the cursor field
		// This is a simplified implementation - in practice, you'd need to specify the cursor field
		if direction == "next" {
			query = query.Where("id > ?", cursor)
		} else {
			query = query.Where("id < ?", cursor)
		}
	}

	var err error
	if len(conds) == 0 {
		err = query.Find(dest).Error
	} else {
		err = query.Where(conds[0], conds[1:]...).Find(dest).Error
	}

	if err != nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("failed to find all entities by conditions: %w", err)
	}

	r.metrics.IncrementOperations(true)
	return nil
}

// FindAllInBatchesByConditionsWithCursor finds all entities in batches by conditions
func (r *BaseRepository[T]) FindAllInBatchesByConditionsWithCursor(ctx context.Context, cursor string, limit int, direction string, dest *[]T, batchSize int, fc func(tx *gorm.DB, batch int) error, conds ...interface{}) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	cursor, limit, direction = r.validateCursorPaginationParams(cursor, limit, direction)

	// Build query with cursor
	query := r.db.WithContext(ctx).Limit(limit)

	if cursor != "" {
		// For cursor-based pagination, we need to know the cursor field
		// This is a simplified implementation - in practice, you'd need to specify the cursor field
		if direction == "next" {
			query = query.Where("id > ?", cursor)
		} else {
			query = query.Where("id < ?", cursor)
		}
	}

	// Validate batch size
	if batchSize <= 0 {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("batch size must be greater than 0, got %d", batchSize)
	}

	var err error
	if len(conds) == 0 {
		err = query.FindInBatches(dest, batchSize, fc).Error
	} else {
		err = query.Where(conds[0], conds[1:]...).FindInBatches(dest, batchSize, fc).Error
	}

	if err != nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("failed to find all entities in batches by conditions: %w", err)
	}

	r.metrics.IncrementOperations(true)
	return nil
}

// Update updates an entity
func (r *BaseRepository[T]) Update(ctx context.Context, entity *T) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	// Check for nil entity
	if entity == nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("entity cannot be nil")
	}

	// Check if entity has a valid ID
	entityID := r.getEntityID(entity)
	if entityID == uuid.Nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("entity must have a valid ID")
	}

	// Validate entity if enabled
	if r.config.EnableValidation {
		if result, err := r.Validate(ctx, entity); err != nil {
			r.metrics.IncrementOperations(false)
			return fmt.Errorf("validation failed: %w", err)
		} else if !result.Valid {
			r.metrics.IncrementOperations(false)
			return errors.New(errors.ErrorTypeValidation, fmt.Sprintf("validation failed: %v", result.Errors))
		}
	}

	// Update entity
	if err := r.db.WithContext(ctx).Save(entity).Error; err != nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("failed to update entity: %w", err)
	}

	r.metrics.IncrementOperations(true)
	r.logger.Info(ctx, "Entity updated successfully",
		logging.String("table", r.tableName),
		logging.String("id", r.getEntityID(entity).String()))

	return nil
}

// UpdateByID updates an entity by ID
func (r *BaseRepository[T]) UpdateByID(ctx context.Context, entity *T, id uuid.UUID) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	// Check for nil entity
	if entity == nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("entity cannot be nil")
	}

	// Check if ID is valid
	if id == uuid.Nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("ID cannot be nil")
	}

	// Check if entity has a valid ID
	entityID := r.getEntityID(entity)
	if entityID == uuid.Nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("entity must have a valid ID")
	}

	if entityID != id {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("entity id must match id")
	}

	// Validate entity if enabled
	if r.config.EnableValidation {
		if result, err := r.Validate(ctx, entity); err != nil {
			r.metrics.IncrementOperations(false)
			return fmt.Errorf("validation failed: %w", err)
		} else if !result.Valid {
			r.metrics.IncrementOperations(false)
			return errors.New(errors.ErrorTypeValidation, fmt.Sprintf("validation failed: %v", result.Errors))
		}
	}

	if err := r.db.WithContext(ctx).Where("id = ?", id).Save(entity).Error; err != nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("failed to update entity by ID: %w", err)
	}

	r.metrics.IncrementOperations(true)
	return nil
}

func (r *BaseRepository[T]) UpdateByConditions(ctx context.Context, entity *T, conds ...interface{}) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	// Check for nil entity
	if entity == nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("entity cannot be nil")
	}

	// // Check if entity has a valid ID
	// entityID := r.getEntityID(entity)
	// if entityID == uuid.Nil {
	// 	r.metrics.IncrementOperations(false)
	// 	return fmt.Errorf("entity must have a valid ID")
	// }

	// Validate entity if enabled
	if r.config.EnableValidation {
		if result, err := r.Validate(ctx, entity); err != nil {
			r.metrics.IncrementOperations(false)
			return fmt.Errorf("validation failed: %w", err)
		} else if !result.Valid {
			r.metrics.IncrementOperations(false)
			return errors.New(errors.ErrorTypeValidation, fmt.Sprintf("validation failed: %v", result.Errors))
		}
	}

	var err error
	if len(conds) == 0 {
		err = r.db.WithContext(ctx).Save(entity).Error
	} else {
		// For bulk updates by conditions, use Updates instead of Save
		err = r.db.WithContext(ctx).Model(new(T)).Where(conds[0], conds[1:]...).Updates(entity).Error
	}

	if err != nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("failed to update entity by conditions: %w", err)
	}

	r.metrics.IncrementOperations(true)
	return nil
}

func (r *BaseRepository[T]) Upsert(ctx context.Context, entity *T, conflictClause string) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	// Check for nil entity
	if entity == nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("entity cannot be nil")
	}

	// Check for empty conflict clause
	if conflictClause == "" {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("conflict clause cannot be empty")
	}

	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Save(entity).Error; err != nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("failed to upsert entity: %w", err)
	}

	r.metrics.IncrementOperations(true)
	return nil
}

func (r *BaseRepository[T]) UpsertByID(ctx context.Context, entity *T, id uuid.UUID, conflictClause string) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Save(entity).Error; err != nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("failed to upsert entity by ID: %w", err)
	}

	r.metrics.IncrementOperations(true)
	return nil
}

func (r *BaseRepository[T]) UpsertByConditions(ctx context.Context, entity *T, conflictClause string, conds ...interface{}) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Save(entity).Error; err != nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("failed to upsert entity by conditions: %w", err)
	}

	r.metrics.IncrementOperations(true)
	return nil
}

func (r *BaseRepository[T]) UpsertInBatches(ctx context.Context, entities []T, batchSize int, conflictClause string) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	// Validate batch size
	if batchSize <= 0 {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("batch size must be greater than 0, got %d", batchSize)
	}

	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Save(&entities).Error; err != nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("failed to upsert entities in batches: %w", err)
	}

	r.metrics.IncrementOperations(true)
	return nil
}

func (r *BaseRepository[T]) UpsertInBatchesByConditions(ctx context.Context, entities []T, batchSize int, conflictClause string, conds ...interface{}) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	// Validate batch size
	if batchSize <= 0 {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("batch size must be greater than 0, got %d", batchSize)
	}

	var err error
	if len(conds) == 0 {
		err = r.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Save(&entities).Error
	} else {
		err = r.db.WithContext(ctx).Where(conds[0], conds[1:]...).Clauses(clause.OnConflict{UpdateAll: true}).Save(&entities).Error
	}

	if err != nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("failed to upsert entities in batches by conditions: %w", err)
	}

	r.metrics.IncrementOperations(true)
	return nil
}

// Delete deletes an entity
func (r *BaseRepository[T]) Delete(ctx context.Context, entity *T) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	if err := r.db.WithContext(ctx).Delete(entity).Error; err != nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("failed to delete entity: %w", err)
	}

	r.metrics.IncrementOperations(true)
	return nil
}

// DeleteByID deletes an entity
func (r *BaseRepository[T]) DeleteByID(ctx context.Context, id uuid.UUID) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	// Check if ID is valid
	if id == uuid.Nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("ID cannot be nil")
	}

	if err := r.db.WithContext(ctx).Delete(new(T), "id = ?", id).Error; err != nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("failed to delete entity: %w", err)
	}

	r.metrics.IncrementOperations(true)
	r.logger.Info(ctx, "Entity deleted successfully",
		logging.String("table", r.tableName),
		logging.String("id", id.String()))

	return nil
}

func (r *BaseRepository[T]) DeleteByConditions(ctx context.Context, entity *T, conds ...interface{}) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	// Check for nil entity
	if entity == nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("entity cannot be nil")
	}

	// Require at least one condition for safety
	if len(conds) == 0 {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("WHERE conditions required")
	}

	var err = r.db.WithContext(ctx).Where(conds[0], conds[1:]...).Delete(entity).Error
	if err != nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("failed to delete entity by conditions: %w", err)
	}

	r.metrics.IncrementOperations(true)
	return nil
}

// DeleteInBatches deletes entities in batches
func (r *BaseRepository[T]) DeleteInBatches(ctx context.Context, entities []T, batchSize int) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	// Validate batch size
	if batchSize <= 0 {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("batch size must be greater than 0, got %d", batchSize)
	}

	if err := r.db.WithContext(ctx).Delete(&entities, batchSize).Error; err != nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("failed to delete entities in batches: %w", err)
	}

	r.metrics.IncrementOperations(true)
	return nil
}

func (r *BaseRepository[T]) DeleteInBatchesByConditions(ctx context.Context, entities []T, batchSize int, conds ...interface{}) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	// Validate batch size
	if batchSize <= 0 {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("batch size must be greater than 0, got %d", batchSize)
	}

	var err error
	if len(conds) == 0 {
		err = r.db.WithContext(ctx).Delete(&entities, batchSize).Error
	} else {
		err = r.db.WithContext(ctx).Where(conds[0], conds[1:]...).Delete(&entities, batchSize).Error
	}

	if err != nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("failed to delete entities in batches by conditions: %w", err)
	}

	r.metrics.IncrementOperations(true)
	return nil
}

// ExistsByID checks if an entity exists
func (r *BaseRepository[T]) ExistsByID(ctx context.Context, id uuid.UUID) (bool, error) {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	var count int64
	if err := r.db.WithContext(ctx).Model(new(T)).Where("id = ?", id).Count(&count).Error; err != nil {
		r.metrics.IncrementOperations(false)
		return false, fmt.Errorf("failed to check entity existence: %w", err)
	}

	r.metrics.IncrementOperations(true)
	return count > 0, nil
}

// ExistsByConditions checks if an entity exists by conditions
func (r *BaseRepository[T]) ExistsByConditions(ctx context.Context, conds ...interface{}) (bool, error) {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	var count int64
	var err error
	if len(conds) == 0 {
		err = r.db.WithContext(ctx).Model(new(T)).Count(&count).Error
	} else {
		err = r.db.WithContext(ctx).Model(new(T)).Where(conds[0], conds[1:]...).Count(&count).Error
	}

	if err != nil {
		r.metrics.IncrementOperations(false)
		return false, fmt.Errorf("failed to check entity existence by conditions: %w", err)
	}

	r.metrics.IncrementOperations(true)
	return count > 0, nil
}

// CountByConditions counts entities by conditions
func (r *BaseRepository[T]) CountByConditions(ctx context.Context, conds ...interface{}) (int64, error) {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	var count int64
	var err error
	if len(conds) == 0 {
		err = r.db.WithContext(ctx).Model(new(T)).Count(&count).Error
	} else {
		err = r.db.WithContext(ctx).Model(new(T)).Where(conds[0], conds[1:]...).Count(&count).Error
	}

	if err != nil {
		r.metrics.IncrementOperations(false)
		return 0, fmt.Errorf("failed to count entities by conditions: %w", err)
	}

	r.metrics.IncrementOperations(true)
	return count, nil
}

// CountAll counts all entities
func (r *BaseRepository[T]) CountAll(ctx context.Context) (int64, error) {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	var count int64
	if err := r.db.WithContext(ctx).Model(new(T)).Count(&count).Error; err != nil {
		r.metrics.IncrementOperations(false)
		return 0, fmt.Errorf("failed to count entities: %w", err)
	}

	r.metrics.IncrementOperations(true)
	return count, nil
}

// TakeByConditions finds first entity by conditions
func (r *BaseRepository[T]) TakeByConditions(ctx context.Context, dest *T, conds ...interface{}) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	var err error
	if len(conds) == 0 {
		err = r.db.WithContext(ctx).Take(dest).Error
	} else {
		err = r.db.WithContext(ctx).Where(conds[0], conds[1:]...).Take(dest).Error
	}

	if err != nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("failed to take entity by conditions: %w", err)
	}

	r.metrics.IncrementOperations(true)
	return nil
}

// LastByConditions finds last entity by conditions
func (r *BaseRepository[T]) LastByConditions(ctx context.Context, dest *T, conds ...interface{}) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	var err error
	if len(conds) == 0 {
		err = r.db.WithContext(ctx).Last(dest).Error
	} else {
		err = r.db.WithContext(ctx).Where(conds[0], conds[1:]...).Last(dest).Error
	}

	if err != nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("failed to last entity by conditions: %w", err)
	}

	r.metrics.IncrementOperations(true)
	return nil
}

// WithTransaction executes a function within a transaction
func (r *BaseRepository[T]) WithTransaction(ctx context.Context, fn func(Repository[T]) error) error {
	start := time.Now()
	defer func() {
		r.metrics.RecordQueryTime(time.Since(start))
	}()

	// Check if function is nil
	if fn == nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("transaction function cannot be nil")
	}

	var txErr error
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := NewBaseRepository[T](tx, r.logger, r.config)

		// Add panic recovery
		defer func() {
			if r := recover(); r != nil {
				txErr = fmt.Errorf("transaction function panicked: %v", r)
			}
		}()

		return fn(txRepo)
	}); err != nil {
		r.metrics.IncrementOperations(false)
		return fmt.Errorf("failed to execute function within transaction: %w", err)
	}

	// Check for panic
	if txErr != nil {
		r.metrics.IncrementOperations(false)
		return txErr
	}

	r.metrics.IncrementOperations(true)
	return nil
}

func (r *BaseRepository[T]) Begin(ctx context.Context) (*gorm.DB, error) {
	return r.db.WithContext(ctx).Begin(), nil //nolint:wrapcheck
}

func (r *BaseRepository[T]) Commit(ctx context.Context) error {
	return r.db.WithContext(ctx).Commit().Error //nolint:wrapcheck
}

func (r *BaseRepository[T]) Rollback(ctx context.Context) error {
	return r.db.WithContext(ctx).Rollback().Error //nolint:wrapcheck
}

// Validate validates an entity
func (r *BaseRepository[T]) Validate(ctx context.Context, entity *T) (*validatorx.ValidationResult, error) {
	if !r.config.EnableValidation {
		return &validatorx.ValidationResult{Valid: true}, nil
	}

	// Check if entity is nil
	if entity == nil {
		return &validatorx.ValidationResult{Valid: false}, fmt.Errorf("entity cannot be nil")
	}

	// Use context-aware validation if available
	result := validatorx.ValidateStructWithContext(ctx, entity)
	return result, nil
}

// Helper methods

// validateOffsetPaginationParams validates and normalizes pagination parameters
func (r *BaseRepository[T]) validateOffsetPaginationParams(limit, offset int) (int, int) {
	limit = r.validateLimit(limit)

	if offset < 0 {
		offset = 0
	}

	return limit, offset
}

// validateLimit validates and normalizes limit parameter
func (r *BaseRepository[T]) validateLimit(limit int) int {
	if limit < 1 {
		limit = r.config.DefaultLimit
	}
	if limit > r.config.MaxLimit {
		limit = r.config.MaxLimit
	}

	return limit
}

// validateCursorPaginationParams validates and normalizes cursor pagination parameters
func (r *BaseRepository[T]) validateCursorPaginationParams(cursor string, limit int, direction string) (string, int, string) {
	limit = r.validateLimit(limit)

	// Normalize direction
	if direction != "next" && direction != "prev" {
		direction = "next"
	}

	return cursor, limit, direction
}

// getEntityID extracts ID from entity
func (r *BaseRepository[T]) getEntityID(entity *T) uuid.UUID {
	if entity == nil {
		return uuid.Nil
	}

	// Try to get ID using reflection
	val := reflect.ValueOf(entity)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if idField := val.FieldByName("ID"); idField.IsValid() {
		if id, ok := idField.Interface().(uuid.UUID); ok {
			return id
		}
	}

	return uuid.Nil
}
