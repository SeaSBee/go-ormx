// Package repositories provides repository implementations for data access layer
package repositories

import (
	"context"
	"reflect"
	"time"

	"go-ormx/ormx/errors"
	"go-ormx/ormx/internal/logging"
	"go-ormx/ormx/models"

	"gorm.io/gorm"
)

// BaseRepository provides a generic repository implementation
type BaseRepository[T models.Modelable] struct {
	db        *gorm.DB
	logger    logging.Logger
	options   RepositoryOptions
	model     T
	modelType reflect.Type
	tableName string
}

// NewBaseRepository creates a new base repository
func NewBaseRepository[T models.Modelable](
	db *gorm.DB,
	logger logging.Logger,
	options RepositoryOptions,
) *BaseRepository[T] {
	var model T
	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	return &BaseRepository[T]{
		db:        db,
		logger:    logger,
		options:   options,
		model:     model,
		modelType: modelType,
		tableName: getTableName(model),
	}
}

// Create creates a new entity
func (r *BaseRepository[T]) Create(ctx context.Context, entity T) error {
	start := time.Now()
	opLog := logging.NewOperationLogger(r.logger, "create", r.tableName)

	// Validate input: if T is a pointer, it should not be nil
	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		r.recordMetrics("create", start, false)
		derr := errors.NewDBError(errors.ErrCodeInvalidInput, "entity cannot be nil", nil)
		opLog.Error("Invalid input for create", derr)
		return derr
	}

	// Set audit fields
	r.setAuditFields(ctx, entity, true)

	// Execute create
	if err := r.db.WithContext(ctx).Create(entity).Error; err != nil {
		r.recordMetrics("create", start, false)
		opLog.Error("Failed to create entity", err)
		return errors.WrapError(err, errors.ErrCodeQueryExecution, "failed to create entity")
	}

	r.recordMetrics("create", start, true)
	opLog.Success("Entity created successfully")
	return nil
}

// CreateBatch creates multiple entities in batch
func (r *BaseRepository[T]) CreateBatch(ctx context.Context, entities []T) error {
	start := time.Now()
	opLog := logging.NewOperationLogger(r.logger, "create_batch", r.tableName)

	if len(entities) == 0 {
		opLog.Warn("Empty batch passed to CreateBatch; nothing to do")
		r.recordMetrics("create_batch", start, true)
		return nil
	}

	// Filter out nil entries for pointer types
	filtered := make([]T, 0, len(entities))
	for i := range entities {
		v := reflect.ValueOf(entities[i])
		if v.Kind() == reflect.Ptr && v.IsNil() {
			continue
		}
		filtered = append(filtered, entities[i])
	}
	if len(filtered) == 0 {
		opLog.Warn("All batch entities were nil; nothing to create")
		r.recordMetrics("create_batch", start, true)
		return nil
	}

	// Set audit fields for all entities
	for i := range filtered {
		r.setAuditFields(ctx, filtered[i], true)
	}

	// Execute batch create
	batchSize := r.options.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}
	if err := r.db.WithContext(ctx).CreateInBatches(filtered, batchSize).Error; err != nil {
		r.recordMetrics("create_batch", start, false)
		opLog.Error("Failed to create entities in batch", err)
		return errors.WrapError(err, errors.ErrCodeQueryExecution, "failed to create entities in batch")
	}

	r.recordMetrics("create_batch", start, true)
	opLog.Success("Entities created successfully in batch")
	return nil
}

// GetByID retrieves an entity by its ID
func (r *BaseRepository[T]) GetByID(ctx context.Context, id string) (T, error) {
	start := time.Now()
	opLog := logging.NewOperationLogger(r.logger, "get_by_id", r.tableName)

	// Build query
	query := r.db.WithContext(ctx).Model(r.model)
	query = query.Where("id = ?", id)

	// Apply tenant filter if entity supports it
	if tenantID := r.getTenantIDFromContext(ctx); tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}

	// Execute query
	var entity T
	if err := query.First(&entity).Error; err != nil {
		r.recordMetrics("get_by_id", start, false)
		if gorm.ErrRecordNotFound == err {
			opLog.Warn("Entity not found for ID")
			return entity, errors.NewDBError(errors.ErrCodeRecordNotFound, "entity not found", err)
		}
		return entity, errors.WrapError(err, errors.ErrCodeQueryExecution, "failed to get entity by ID")
	}

	r.recordMetrics("get_by_id", start, true)
	opLog.Success("Entity retrieved successfully")
	return entity, nil
}

// Update updates an entity
func (r *BaseRepository[T]) Update(ctx context.Context, entity T) error {
	start := time.Now()
	opLog := logging.NewOperationLogger(r.logger, "update", r.tableName).
		WithField("entity_id", entity.GetID())

	// Set audit fields
	r.setAuditFields(ctx, entity, false)

	// Execute update
	result := r.db.WithContext(ctx).Save(entity)
	if result.Error != nil {
		r.recordMetrics("update", start, false)
		opLog.Error("Failed to update entity", result.Error)
		return errors.WrapError(result.Error, errors.ErrCodeQueryExecution, "failed to update entity")
	}

	if result.RowsAffected == 0 {
		r.recordMetrics("update", start, false)
		opLog.Warn("No rows affected for update")
		return errors.NewDBError(errors.ErrCodeRecordNotFound, "entity not found for update", nil)
	}

	r.recordMetrics("update", start, true)
	opLog.Success("Entity updated successfully")
	return nil
}

// UpdatePartial updates specific fields of an entity
func (r *BaseRepository[T]) UpdatePartial(ctx context.Context, id string, updates map[string]interface{}) error {
	start := time.Now()
	opLog := logging.NewOperationLogger(r.logger, "update_partial", r.tableName).
		WithField("entity_id", id)

	if len(updates) == 0 {
		r.recordMetrics("update_partial", start, false)
		derr := errors.NewDBError(errors.ErrCodeInvalidInput, "updates map cannot be empty", nil)
		opLog.Error("Invalid input for partial update", derr)
		return derr
	}

	// Add audit fields to updates
	r.addAuditFieldsToMap(ctx, updates, false)

	// Build query
	query := r.db.WithContext(ctx).Model(r.model).Where("id = ?", id)

	// Apply tenant filter if entity supports it
	if tenantID := r.getTenantIDFromContext(ctx); tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}

	// Execute update
	result := query.Updates(updates)
	if result.Error != nil {
		r.recordMetrics("update_partial", start, false)
		opLog.Error("Failed to update entity partially", result.Error)
		return errors.WrapError(result.Error, errors.ErrCodeQueryExecution, "failed to update entity partially")
	}

	if result.RowsAffected == 0 {
		r.recordMetrics("update_partial", start, false)
		return errors.NewDBError(errors.ErrCodeRecordNotFound, "entity not found for partial update", nil)
	}

	r.recordMetrics("update_partial", start, true)
	opLog.Success("Entity updated partially successfully")
	return nil
}

// Delete deletes an entity
func (r *BaseRepository[T]) Delete(ctx context.Context, id string) error {
	start := time.Now()
	opLog := logging.NewOperationLogger(r.logger, "delete", r.tableName).
		WithField("entity_id", id)

	// Build query
	query := r.db.WithContext(ctx).Model(r.model).Where("id = ?", id)

	// Apply tenant filter if entity supports it
	if tenantID := r.getTenantIDFromContext(ctx); tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}

	// Execute delete
	result := query.Delete(r.model)
	if result.Error != nil {
		r.recordMetrics("delete", start, false)
		opLog.Error("Failed to delete entity", result.Error)
		return errors.WrapError(result.Error, errors.ErrCodeQueryExecution, "failed to delete entity")
	}

	if result.RowsAffected == 0 {
		r.recordMetrics("delete", start, false)
		opLog.Warn("No rows affected for delete")
		return errors.NewDBError(errors.ErrCodeRecordNotFound, "entity not found for deletion", nil)
	}

	r.recordMetrics("delete", start, true)
	opLog.Success("Entity deleted successfully")
	return nil
}

// SoftDelete soft deletes an entity
func (r *BaseRepository[T]) SoftDelete(ctx context.Context, id string) error {
	start := time.Now()
	opLog := logging.NewOperationLogger(r.logger, "soft_delete", r.tableName).
		WithField("entity_id", id)

	// Build query
	query := r.db.WithContext(ctx).Model(r.model).Where("id = ?", id)

	// Apply tenant filter if entity supports it
	if tenantID := r.getTenantIDFromContext(ctx); tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}

	// Execute soft delete
	result := query.Update("deleted_at", time.Now())
	if result.Error != nil {
		r.recordMetrics("soft_delete", start, false)
		opLog.Error("Failed to soft delete entity", result.Error)
		return errors.WrapError(result.Error, errors.ErrCodeQueryExecution, "failed to soft delete entity")
	}

	if result.RowsAffected == 0 {
		r.recordMetrics("soft_delete", start, false)
		opLog.Warn("No rows affected for soft delete")
		return errors.NewDBError(errors.ErrCodeRecordNotFound, "entity not found for soft deletion", nil)
	}

	r.recordMetrics("soft_delete", start, true)
	opLog.Success("Entity soft deleted successfully")
	return nil
}

// Find finds entities based on filter
func (r *BaseRepository[T]) Find(ctx context.Context, filter Filter) ([]T, error) {
	start := time.Now()
	opLog := logging.NewOperationLogger(r.logger, "find", r.tableName)

	query := r.buildQuery(ctx, filter)

	var entities []T
	if err := query.Find(&entities).Error; err != nil {
		r.recordMetrics("find", start, false)
		opLog.Error("Failed to find entities", err)
		return nil, errors.WrapError(err, errors.ErrCodeQueryExecution, "failed to find entities")
	}

	r.recordMetrics("find", start, true)
	opLog.Success("Entities found successfully")
	if entities == nil {
		return []T{}, nil
	}
	return entities, nil
}

// FindOne finds a single entity based on filter
func (r *BaseRepository[T]) FindOne(ctx context.Context, filter Filter) (T, error) {
	start := time.Now()
	opLog := logging.NewOperationLogger(r.logger, "find_one", r.tableName)

	query := r.buildQuery(ctx, filter)

	var entity T
	if err := query.First(&entity).Error; err != nil {
		r.recordMetrics("find_one", start, false)
		if gorm.ErrRecordNotFound == err {
			opLog.Warn("Entity not found for filter")
			return entity, errors.NewDBError(errors.ErrCodeRecordNotFound, "entity not found", err)
		}
		opLog.Error("Failed to find entity", err)
		return entity, errors.WrapError(err, errors.ErrCodeQueryExecution, "failed to find entity")
	}

	r.recordMetrics("find_one", start, true)
	opLog.Success("Entity found successfully")
	return entity, nil
}

// Count counts entities based on filter
func (r *BaseRepository[T]) Count(ctx context.Context, filter Filter) (int64, error) {
	start := time.Now()
	opLog := logging.NewOperationLogger(r.logger, "count", r.tableName)

	query := r.buildQueryForCount(ctx, filter)

	var count int64
	if err := query.Count(&count).Error; err != nil {
		r.recordMetrics("count", start, false)
		opLog.Error("Failed to count entities", err)
		return 0, errors.WrapError(err, errors.ErrCodeQueryExecution, "failed to count entities")
	}

	r.recordMetrics("count", start, true)
	opLog.Success("Entities counted successfully")
	return count, nil
}

// Exists checks if an entity exists
func (r *BaseRepository[T]) Exists(ctx context.Context, id string) (bool, error) {
	start := time.Now()
	opLog := logging.NewOperationLogger(r.logger, "exists", r.tableName)

	query := r.db.WithContext(ctx).Model(r.model).Where("id = ?", id)

	// Apply tenant filter if entity supports it
	if tenantID := r.getTenantIDFromContext(ctx); tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		r.recordMetrics("exists", start, false)
		opLog.Error("Failed to check entity existence", err)
		return false, errors.WrapError(err, errors.ErrCodeQueryExecution, "failed to check entity existence")
	}

	r.recordMetrics("exists", start, true)
	opLog.Success("Entity existence checked successfully")
	return count > 0, nil
}

// FindByIDs finds entities by their IDs
func (r *BaseRepository[T]) FindByIDs(ctx context.Context, ids []string) ([]T, error) {
	start := time.Now()
	opLog := logging.NewOperationLogger(r.logger, "find_by_ids", r.tableName)

	if len(ids) == 0 {
		return []T{}, nil
	}

	query := r.db.WithContext(ctx).Model(r.model).Where("id IN ?", ids)

	// Apply tenant filter if entity supports it
	if tenantID := r.getTenantIDFromContext(ctx); tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}

	var entities []T
	if err := query.Find(&entities).Error; err != nil {
		r.recordMetrics("find_by_ids", start, false)
		opLog.Error("Failed to find entities by IDs", err)
		return nil, errors.WrapError(err, errors.ErrCodeQueryExecution, "failed to find entities by IDs")
	}

	r.recordMetrics("find_by_ids", start, true)
	opLog.Success("Entities found by IDs successfully")
	if entities == nil {
		return []T{}, nil
	}
	return entities, nil
}

// FindRaw executes a raw query and returns entities
func (r *BaseRepository[T]) FindRaw(ctx context.Context, query string, args ...interface{}) ([]T, error) {
	start := time.Now()
	opLog := logging.NewOperationLogger(r.logger, "find_raw", r.tableName)

	var entities []T
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&entities).Error; err != nil {
		r.recordMetrics("find_raw", start, false)
		opLog.Error("Failed to execute raw query", err)
		return nil, errors.WrapError(err, errors.ErrCodeQueryExecution, "failed to execute raw query")
	}

	r.recordMetrics("find_raw", start, true)
	opLog.Success("Raw query executed successfully")
	if entities == nil {
		return []T{}, nil
	}
	return entities, nil
}

// ExecuteRaw executes a raw query without returning results
func (r *BaseRepository[T]) ExecuteRaw(ctx context.Context, query string, args ...interface{}) error {
	start := time.Now()
	opLog := logging.NewOperationLogger(r.logger, "execute_raw", r.tableName)

	if err := r.db.WithContext(ctx).Exec(query, args...).Error; err != nil {
		r.recordMetrics("execute_raw", start, false)
		opLog.Error("Failed to execute raw query", err)
		return errors.WrapError(err, errors.ErrCodeQueryExecution, "failed to execute raw query")
	}

	r.recordMetrics("execute_raw", start, true)
	opLog.Success("Raw query executed successfully")
	return nil
}

// WithTx returns a repository with transaction
func (r *BaseRepository[T]) WithTx(tx *gorm.DB) Repository[T] {
	txRepo := *r
	txRepo.db = tx
	return &txRepo
}

// BeginTx begins a transaction
func (r *BaseRepository[T]) BeginTx(ctx context.Context) (*gorm.DB, error) {
	tx := r.db.WithContext(ctx).Begin()
	return tx, tx.Error
}

// CommitTx commits a transaction
func (r *BaseRepository[T]) CommitTx(tx *gorm.DB) error {
	return tx.Commit().Error
}

// RollbackTx rolls back a transaction
func (r *BaseRepository[T]) RollbackTx(tx *gorm.DB) error {
	return tx.Rollback().Error
}

// buildQuery builds a query based on filter
func (r *BaseRepository[T]) buildQuery(ctx context.Context, filter Filter) *gorm.DB {
	query := r.db.WithContext(ctx).Model(r.model)

	// Apply tenant filter if entity supports it
	if tenantID := r.getTenantIDFromContext(ctx); tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}

	// Apply filter conditions
	query = r.applyWhereConditions(query, filter)

	// Apply ordering
	for _, orderBy := range filter.OrderBy {
		direction := "ASC"
		if orderBy.Direction == "DESC" {
			direction = "DESC"
		}
		query = query.Order(orderBy.Field + " " + direction)
	}

	// Apply pagination
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	return query
}

// buildQueryForCount builds a count query based on filter
func (r *BaseRepository[T]) buildQueryForCount(ctx context.Context, filter Filter) *gorm.DB {
	query := r.db.WithContext(ctx).Model(r.model)

	// Apply tenant filter if entity supports it
	if tenantID := r.getTenantIDFromContext(ctx); tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}

	// Apply filter conditions
	query = r.applyWhereConditions(query, filter)

	return query
}

// applyWhereConditions applies where conditions to a query
func (r *BaseRepository[T]) applyWhereConditions(query *gorm.DB, filter Filter) *gorm.DB {
	for field, condition := range filter.Where {
		switch condition.Operator {
		case "eq":
			query = query.Where(field+" = ?", condition.Value)
		case "ne":
			query = query.Where(field+" != ?", condition.Value)
		case "gt":
			query = query.Where(field+" > ?", condition.Value)
		case "gte":
			query = query.Where(field+" >= ?", condition.Value)
		case "lt":
			query = query.Where(field+" < ?", condition.Value)
		case "lte":
			query = query.Where(field+" <= ?", condition.Value)
		case "like":
			query = query.Where(field+" LIKE ?", condition.Value)
		case "in":
			query = query.Where(field+" IN ?", condition.Value)
		default:
			query = query.Where(field+" = ?", condition.Value)
		}
	}

	return query
}

// setAuditFields sets audit fields on an entity
func (r *BaseRepository[T]) setAuditFields(ctx context.Context, entity T, isCreate bool) {
	// This would typically set audit fields on the entity
	// For now, do nothing as the interface methods are not available
}

// addAuditFieldsToMap adds audit fields to a map
func (r *BaseRepository[T]) addAuditFieldsToMap(ctx context.Context, updates map[string]interface{}, isCreate bool) {
	now := time.Now()
	userID := r.getUserIDFromContext(ctx)

	if isCreate {
		updates["created_at"] = now
		updates["created_by"] = userID
	}
	updates["updated_at"] = now
	updates["updated_by"] = userID
}

// getTenantIDFromContext extracts tenant ID from context
func (r *BaseRepository[T]) getTenantIDFromContext(ctx context.Context) string {
	// This would typically extract tenant ID from context
	// For now, return empty string
	return ""
}

// getUserIDFromContext extracts user ID from context
func (r *BaseRepository[T]) getUserIDFromContext(ctx context.Context) string {
	// This would typically extract user ID from context
	// For now, return empty string
	return ""
}

// recordMetrics records operation metrics
func (r *BaseRepository[T]) recordMetrics(operation string, start time.Time, success bool) {
	// This would typically record metrics
	// For now, do nothing
}

// getTableName gets the table name for a model
func getTableName(model interface{}) string {
	if tabler, ok := model.(interface{ TableName() string }); ok {
		return tabler.TableName()
	}
	return ""
}
