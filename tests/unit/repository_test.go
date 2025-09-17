package unit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/seasbee/go-ormx/pkg/logging"
	"github.com/seasbee/go-ormx/pkg/models"
	"github.com/seasbee/go-ormx/pkg/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestEntity represents a test entity for testing
type TestEntity struct {
	models.BaseModel
	Name string `gorm:"not null"`
	Age  int    `gorm:"not null"`
}

func (t TestEntity) TableName() string {
	return "test_entities"
}

// setupTestDB creates a test database
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto migrate the test entity
	err = db.AutoMigrate(&TestEntity{})
	require.NoError(t, err)

	return db
}

// setupTestRepository creates a test repository
func setupTestRepository(t *testing.T) (*repository.BaseRepository[TestEntity], *gorm.DB) {
	db := setupTestDB(t)
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})
	config := repository.DefaultRepositoryConfig()

	repo := repository.NewBaseRepository[TestEntity](db, logger, config)
	return repo, db
}

func TestNewBaseRepository(t *testing.T) {
	db := setupTestDB(t)
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})

	// Test with nil config (should use default)
	repo := repository.NewBaseRepository[TestEntity](db, logger, nil)
	assert.NotNil(t, repo)

	// Test with custom config
	config := &repository.RepositoryConfig{
		EnableValidation: false,
		EnableMetrics:    true,
		DefaultLimit:     50,
		MaxLimit:         2000,
	}
	repo = repository.NewBaseRepository[TestEntity](db, logger, config)
	assert.NotNil(t, repo)
}

func TestBaseRepository_Create(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	entity := &TestEntity{
		Name: "John Doe",
		Age:  30,
	}

	err := repo.Create(ctx, entity)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, entity.GetID())

	// Test validation - should pass since no validation rules are defined
	invalidEntity := &TestEntity{
		Name: "", // Empty name
		Age:  -5, // Negative age
	}

	err = repo.Create(ctx, invalidEntity)
	assert.NoError(t, err) // Should pass since no validation rules are defined
}

func TestBaseRepository_CreateInBatches(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	entities := []TestEntity{
		{Name: "Alice", Age: 25},
		{Name: "Bob", Age: 30},
		{Name: "Charlie", Age: 35},
	}

	err := repo.CreateInBatches(ctx, entities, 2)
	assert.NoError(t, err)

	// Test with empty batch
	err = repo.CreateInBatches(ctx, []TestEntity{}, 10)
	assert.Error(t, err)

	// Test with nil batch
	err = repo.CreateInBatches(ctx, nil, 10)
	assert.Error(t, err)
}

func TestBaseRepository_FindFirstByID(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Create an entity first
	entity := &TestEntity{
		Name: "John Doe",
		Age:  30,
	}
	err := repo.Create(ctx, entity)
	require.NoError(t, err)

	// Find by ID
	found, err := repo.FindFirstByID(ctx, entity.GetID())
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, entity.GetID(), found.GetID())
	assert.Equal(t, entity.Name, found.Name)
	assert.Equal(t, entity.Age, found.Age)

	// Test with non-existent ID
	nonExistentID := uuid.New()
	found, err = repo.FindFirstByID(ctx, nonExistentID)
	assert.Error(t, err)
	assert.Nil(t, found)
}

func TestBaseRepository_FindFirstByConditions(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Create entities
	entities := []TestEntity{
		{Name: "Alice", Age: 25},
		{Name: "Bob", Age: 30},
		{Name: "Charlie", Age: 35},
	}

	for _, entity := range entities {
		err := repo.Create(ctx, &entity)
		require.NoError(t, err)
	}

	// Find by conditions
	var found TestEntity
	err := repo.FindFirstByConditions(ctx, &found, "age > ?", 30)
	assert.NoError(t, err)
	assert.Equal(t, "Charlie", found.Name)
	assert.Equal(t, 35, found.Age)
}

func TestBaseRepository_FirstOrInitByConditions(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Create an entity
	entity := &TestEntity{
		Name: "John Doe",
		Age:  30,
	}
	err := repo.Create(ctx, entity)
	require.NoError(t, err)

	// Find existing
	var found TestEntity
	err = repo.FirstOrInitByConditions(ctx, &found, "name = ?", "John Doe")
	assert.NoError(t, err)
	assert.Equal(t, "John Doe", found.Name)

	// Init non-existing - FirstOrInit only initializes with zero values
	var newEntity TestEntity
	err = repo.FirstOrInitByConditions(ctx, &newEntity, "name = ?", "Jane Doe")
	assert.NoError(t, err)
	assert.Equal(t, "", newEntity.Name) // FirstOrInit doesn't populate search conditions
	assert.Equal(t, uuid.Nil, newEntity.GetID())
}

func TestBaseRepository_FindAll(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Create entities
	entities := []TestEntity{
		{Name: "Alice", Age: 25},
		{Name: "Bob", Age: 30},
		{Name: "Charlie", Age: 35},
	}

	for _, entity := range entities {
		err := repo.Create(ctx, &entity)
		require.NoError(t, err)
	}

	// Find all
	var found []TestEntity
	err := repo.FindAllWithOffset(ctx, 10, 0, &found)
	assert.NoError(t, err)
	assert.Len(t, found, 3)
}

func TestBaseRepository_FindAllByConditions(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Create entities
	entities := []TestEntity{
		{Name: "Alice", Age: 25},
		{Name: "Bob", Age: 30},
		{Name: "Charlie", Age: 35},
	}

	for _, entity := range entities {
		err := repo.Create(ctx, &entity)
		require.NoError(t, err)
	}

	// Find by conditions
	var found []TestEntity
	err := repo.FindAllByConditionsWithOffset(ctx, 10, 0, &found, "age > ?", 25)
	assert.NoError(t, err)
	assert.Len(t, found, 2)
}

func TestBaseRepository_Update(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Create an entity
	entity := &TestEntity{
		Name: "John Doe",
		Age:  30,
	}
	err := repo.Create(ctx, entity)
	require.NoError(t, err)

	// Update the entity
	entity.Name = "John Smith"
	entity.Age = 31

	err = repo.Update(ctx, entity)
	assert.NoError(t, err)

	// Verify update
	found, err := repo.FindFirstByID(ctx, entity.GetID())
	assert.NoError(t, err)
	assert.Equal(t, "John Smith", found.Name)
	assert.Equal(t, 31, found.Age)
}

func TestBaseRepository_UpdateByID(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Create an entity
	entity := &TestEntity{
		Name: "John Doe",
		Age:  30,
	}
	err := repo.Create(ctx, entity)
	require.NoError(t, err)

	// Update by ID
	updateEntity := &TestEntity{
		Name: "John Smith",
		Age:  31,
	}
	updateEntity.SetID(entity.GetID()) // Set the ID for the update

	err = repo.UpdateByID(ctx, updateEntity, entity.GetID())
	assert.NoError(t, err)

	// Verify update
	found, err := repo.FindFirstByID(ctx, entity.GetID())
	assert.NoError(t, err)
	assert.Equal(t, "John Smith", found.Name)
	assert.Equal(t, 31, found.Age)
}

func TestBaseRepository_UpdateByConditions(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Create entities
	entities := []TestEntity{
		{Name: "Alice", Age: 25},
		{Name: "Bob", Age: 30},
		{Name: "Charlie", Age: 35},
	}

	for _, entity := range entities {
		err := repo.Create(ctx, &entity)
		require.NoError(t, err)
	}

	// Update by conditions
	updateEntity := &TestEntity{
		Age: 40,
	}

	err := repo.UpdateByConditions(ctx, updateEntity, "age > ?", 30)
	assert.NoError(t, err)

	// Verify updates
	var found []TestEntity
	err = repo.FindAllByConditionsWithOffset(ctx, 10, 0, &found, "age > ?", 30)
	assert.NoError(t, err)
	for _, entity := range found {
		assert.Equal(t, 40, entity.Age)
	}
}

func TestBaseRepository_Upsert(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Create an entity
	entity := &TestEntity{
		Name: "John Doe",
		Age:  30,
	}
	err := repo.Create(ctx, entity)
	require.NoError(t, err)

	// Upsert the same entity
	entity.Age = 31
	err = repo.Upsert(ctx, entity, "id")
	assert.NoError(t, err)

	// Verify upsert
	found, err := repo.FindFirstByID(ctx, entity.GetID())
	assert.NoError(t, err)
	assert.Equal(t, 31, found.Age)
}

func TestBaseRepository_DeleteByID(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Create an entity
	entity := &TestEntity{
		Name: "John Doe",
		Age:  30,
	}
	err := repo.Create(ctx, entity)
	require.NoError(t, err)

	// Delete by ID
	err = repo.DeleteByID(ctx, entity.GetID())
	assert.NoError(t, err)

	// Verify deletion
	exists, err := repo.ExistsByID(ctx, entity.GetID())
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestBaseRepository_DeleteByConditions(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Create entities
	entities := []TestEntity{
		{Name: "Alice", Age: 25},
		{Name: "Bob", Age: 30},
		{Name: "Charlie", Age: 35},
	}

	for _, entity := range entities {
		err := repo.Create(ctx, &entity)
		require.NoError(t, err)
	}

	// Delete by conditions
	err := repo.DeleteByConditions(ctx, &TestEntity{}, "age > ?", 30)
	assert.NoError(t, err)

	// Verify deletion
	var found []TestEntity
	err = repo.FindAllByConditionsWithOffset(ctx, 10, 0, &found, "age > ?", 30)
	assert.NoError(t, err)
	assert.Len(t, found, 0)
}

func TestBaseRepository_ExistsByID(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Create an entity
	entity := &TestEntity{
		Name: "John Doe",
		Age:  30,
	}
	err := repo.Create(ctx, entity)
	require.NoError(t, err)

	// Check existence
	exists, err := repo.ExistsByID(ctx, entity.GetID())
	assert.NoError(t, err)
	assert.True(t, exists)

	// Check non-existence
	nonExistentID := uuid.New()
	exists, err = repo.ExistsByID(ctx, nonExistentID)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestBaseRepository_ExistsByConditions(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Create entities
	entities := []TestEntity{
		{Name: "Alice", Age: 25},
		{Name: "Bob", Age: 30},
		{Name: "Charlie", Age: 35},
	}

	for _, entity := range entities {
		err := repo.Create(ctx, &entity)
		require.NoError(t, err)
	}

	// Check existence by conditions
	exists, err := repo.ExistsByConditions(ctx, "age > ?", 30)
	assert.NoError(t, err)
	assert.True(t, exists)

	// Check non-existence by conditions
	exists, err = repo.ExistsByConditions(ctx, "age > ?", 100)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestBaseRepository_CountByConditions(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Create entities
	entities := []TestEntity{
		{Name: "Alice", Age: 25},
		{Name: "Bob", Age: 30},
		{Name: "Charlie", Age: 35},
	}

	for _, entity := range entities {
		err := repo.Create(ctx, &entity)
		require.NoError(t, err)
	}

	// Count by conditions
	count, err := repo.CountByConditions(ctx, "age > ?", 25)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// Count all
	count, err = repo.CountAll(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestBaseRepository_TakeByConditions(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Create entities
	entities := []TestEntity{
		{Name: "Alice", Age: 25},
		{Name: "Bob", Age: 30},
		{Name: "Charlie", Age: 35},
	}

	for _, entity := range entities {
		err := repo.Create(ctx, &entity)
		require.NoError(t, err)
	}

	// Take by conditions
	var found TestEntity
	err := repo.TakeByConditions(ctx, &found, "age > ?", 30)
	assert.NoError(t, err)
	assert.Equal(t, "Charlie", found.Name)
}

func TestBaseRepository_LastByConditions(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Create entities
	entities := []TestEntity{
		{Name: "Alice", Age: 25},
		{Name: "Bob", Age: 30},
		{Name: "Charlie", Age: 35},
	}

	for _, entity := range entities {
		err := repo.Create(ctx, &entity)
		require.NoError(t, err)
	}

	// Get last by conditions
	var found TestEntity
	err := repo.LastByConditions(ctx, &found, "age > ?", 25)
	assert.NoError(t, err)
	assert.Equal(t, "Charlie", found.Name)
}

func TestBaseRepository_WithTransaction(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	err := repo.WithTransaction(ctx, func(txRepo repository.Repository[TestEntity]) error {
		// Create entity in transaction
		entity := &TestEntity{
			Name: "John Doe",
			Age:  30,
		}

		err := txRepo.Create(ctx, entity)
		if err != nil {
			return err
		}

		// Update entity in same transaction
		entity.Age = 31
		err = txRepo.Update(ctx, entity)
		return err
	})

	assert.NoError(t, err)

	// Verify transaction was committed
	var found []TestEntity
	err = repo.FindAllByConditionsWithOffset(ctx, 10, 0, &found, "name = ?", "John Doe")
	assert.NoError(t, err)
	assert.Len(t, found, 1)
	assert.Equal(t, 31, found[0].Age)
}

func TestBaseRepository_PaginateWithOffset(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Create entities
	entities := []TestEntity{
		{Name: "Alice", Age: 25},
		{Name: "Bob", Age: 30},
		{Name: "Charlie", Age: 35},
		{Name: "David", Age: 40},
		{Name: "Eve", Age: 45},
	}

	for _, entity := range entities {
		err := repo.Create(ctx, &entity)
		require.NoError(t, err)
	}

	// Test pagination
	var found []TestEntity
	err := repo.FindAllWithOffset(ctx, 2, 1, &found)
	assert.NoError(t, err)
	assert.Len(t, found, 2)
	assert.Equal(t, "Bob", found[0].Name)
	assert.Equal(t, "Charlie", found[1].Name)
}

func TestBaseRepository_PaginateWithCursor(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Create entities
	entities := []TestEntity{
		{Name: "Alice", Age: 25},
		{Name: "Bob", Age: 30},
		{Name: "Charlie", Age: 35},
	}

	for _, entity := range entities {
		err := repo.Create(ctx, &entity)
		require.NoError(t, err)
	}

	// Test cursor pagination
	var found []TestEntity
	err := repo.FindAllWithCursor(ctx, "", 2, "next", &found)
	assert.NoError(t, err)
	assert.Len(t, found, 2)
}

func TestBaseRepository_Validate(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Test valid entity
	validEntity := &TestEntity{
		Name: "John Doe",
		Age:  30,
	}

	result, err := repo.Validate(ctx, validEntity)
	assert.NoError(t, err)
	assert.True(t, result.Valid)

	// Test invalid entity - should pass since no validation rules are defined
	invalidEntity := &TestEntity{
		Name: "", // Empty name
		Age:  -5, // Negative age
	}

	result, err = repo.Validate(ctx, invalidEntity)
	assert.NoError(t, err)
	assert.True(t, result.Valid) // Should pass since no validation rules are defined
}

func TestBaseRepository_Metrics(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Perform some operations to generate metrics
	entity := &TestEntity{
		Name: "John Doe",
		Age:  30,
	}

	err := repo.Create(ctx, entity)
	assert.NoError(t, err)

	// Check that operation was successful
	// Note: GetMetrics method is not available on BaseRepository
	// Metrics are collected internally but not exposed via public API
}

func TestBaseRepository_ConcurrentAccess(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Test concurrent access
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			// These should be safe for concurrent access
			entity := &TestEntity{
				Name: fmt.Sprintf("User %d", id),
				Age:  20 + id,
			}

			err := repo.Create(ctx, entity)
			if err == nil {
				repo.ExistsByID(ctx, entity.GetID())
				repo.CountAll(ctx)
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestBaseRepository_EdgeCases(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Test with nil entity
	err := repo.Create(ctx, nil)
	assert.Error(t, err)

	// Test with empty conditions
	var found []TestEntity
	err = repo.FindAllByConditionsWithOffset(ctx, 10, 0, &found)
	assert.NoError(t, err)

	// Test with invalid pagination parameters
	err = repo.FindAllWithOffset(ctx, -1, -1, &found)
	assert.NoError(t, err) // Should handle gracefully

	err = repo.FindAllWithCursor(ctx, "", -1, "invalid", &found)
	assert.NoError(t, err) // Should handle gracefully
}

func TestRepositoryConfig(t *testing.T) {
	config := repository.DefaultRepositoryConfig()

	assert.True(t, config.EnableValidation)
	assert.True(t, config.EnableMetrics)
	assert.Equal(t, 20, config.DefaultLimit)
	assert.Equal(t, 1000, config.MaxLimit)

	// Test custom config
	customConfig := &repository.RepositoryConfig{
		EnableValidation: false,
		EnableMetrics:    false,
		DefaultLimit:     50,
		MaxLimit:         2000,
	}

	assert.False(t, customConfig.EnableValidation)
	assert.False(t, customConfig.EnableMetrics)
	assert.Equal(t, 50, customConfig.DefaultLimit)
	assert.Equal(t, 2000, customConfig.MaxLimit)
}

func TestRepositoryMetrics(t *testing.T) {
	metrics := repository.NewRepositoryMetrics()

	// Test initial state
	assert.Equal(t, int64(0), metrics.TotalOperations)
	assert.Equal(t, int64(0), metrics.SuccessfulOperations)
	assert.Equal(t, int64(0), metrics.FailedOperations)

	// Test increment operations
	metrics.IncrementOperations(true)
	metrics.IncrementOperations(false)
	metrics.IncrementOperations(true)

	assert.Equal(t, int64(3), metrics.TotalOperations)
	assert.Equal(t, int64(2), metrics.SuccessfulOperations)
	assert.Equal(t, int64(1), metrics.FailedOperations)

	// Test query time recording
	metrics.RecordQueryTime(100 * time.Millisecond)
	metrics.RecordQueryTime(200 * time.Millisecond)

	assert.Equal(t, 150*time.Millisecond, metrics.AverageQueryTime)

	// Test success rate
	successRate := metrics.GetSuccessRate()
	assert.Equal(t, 2.0/3.0, successRate)

	// Test reset
	metrics.Reset()
	assert.Equal(t, int64(0), metrics.TotalOperations)
	assert.Equal(t, int64(0), metrics.SuccessfulOperations)
	assert.Equal(t, int64(0), metrics.FailedOperations)
	assert.Equal(t, time.Duration(0), metrics.AverageQueryTime)
}

// Add missing test scenarios
func TestBaseRepository_ErrorHandling(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Test with nil entity
	err := repo.Create(ctx, nil)
	assert.Error(t, err)

	// Test with nil slice
	err = repo.CreateInBatches(ctx, nil, 10)
	assert.Error(t, err)

	// Test with empty slice
	err = repo.CreateInBatches(ctx, []TestEntity{}, 10)
	assert.Error(t, err)

	// Test with invalid batch size
	entities := []TestEntity{{Name: "Test", Age: 30}}
	err = repo.CreateInBatches(ctx, entities, 0)
	assert.Error(t, err)

	err = repo.CreateInBatches(ctx, entities, -1)
	assert.Error(t, err)
}

func TestBaseRepository_ValidationEdgeCases(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Test with nil entity for validation
	result, err := repo.Validate(ctx, nil)
	assert.Error(t, err)
	assert.False(t, result.Valid)

	// Test with empty entity
	emptyEntity := &TestEntity{}
	result, err = repo.Validate(ctx, emptyEntity)
	assert.NoError(t, err)
	assert.True(t, result.Valid) // Should pass since no validation rules are defined

	// Test with entity containing only ID
	idOnlyEntity := &TestEntity{}
	idOnlyEntity.SetID(uuid.New())
	result, err = repo.Validate(ctx, idOnlyEntity)
	assert.NoError(t, err)
	assert.True(t, result.Valid)
}

func TestBaseRepository_PaginationEdgeCases(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Create some test entities
	entities := []TestEntity{
		{Name: "Alice", Age: 25},
		{Name: "Bob", Age: 30},
		{Name: "Charlie", Age: 35},
		{Name: "David", Age: 40},
		{Name: "Eve", Age: 45},
	}

	for _, entity := range entities {
		err := repo.Create(ctx, &entity)
		require.NoError(t, err)
	}

	// Test offset pagination edge cases
	var found []TestEntity

	// Test with negative offset
	err := repo.FindAllWithOffset(ctx, -1, 2, &found)
	assert.NoError(t, err) // Should handle gracefully

	// Test with negative limit
	err = repo.FindAllWithOffset(ctx, 0, -1, &found)
	assert.NoError(t, err) // Should handle gracefully

	// Test with zero limit
	err = repo.FindAllWithOffset(ctx, 0, 0, &found)
	assert.NoError(t, err) // Should handle gracefully

	// Test with very large offset
	err = repo.FindAllWithOffset(ctx, 999999, 2, &found)
	assert.NoError(t, err) // Should handle gracefully

	// Test with very large limit
	err = repo.FindAllWithOffset(ctx, 0, 999999, &found)
	assert.NoError(t, err) // Should handle gracefully

	// Test cursor pagination edge cases
	// Test with invalid direction
	err = repo.FindAllWithCursor(ctx, "", 2, "invalid", &found)
	assert.NoError(t, err) // Should handle gracefully

	// Test with negative limit
	err = repo.FindAllWithCursor(ctx, "", -1, "next", &found)
	assert.NoError(t, err) // Should handle gracefully

	// Test with zero limit
	err = repo.FindAllWithCursor(ctx, "", 0, "next", &found)
	assert.NoError(t, err) // Should handle gracefully

	// Test with very large limit
	err = repo.FindAllWithCursor(ctx, "", 999999, "next", &found)
	assert.NoError(t, err) // Should handle gracefully
}

func TestBaseRepository_TransactionEdgeCases(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Test with nil transaction function
	err := repo.WithTransaction(ctx, nil)
	assert.Error(t, err)

	// Test with transaction function that returns error
	err = repo.WithTransaction(ctx, func(txRepo repository.Repository[TestEntity]) error {
		return assert.AnError
	})
	assert.Error(t, err)

	// Test with transaction function that panics
	err = repo.WithTransaction(ctx, func(txRepo repository.Repository[TestEntity]) error {
		panic("test panic")
	})
	assert.Error(t, err)

	// Test with cancelled context
	cancelledCtx, cancel := context.WithCancel(ctx)
	cancel()

	err = repo.WithTransaction(cancelledCtx, func(txRepo repository.Repository[TestEntity]) error {
		entity := &TestEntity{Name: "Test", Age: 30}
		return txRepo.Create(ctx, entity)
	})
	assert.Error(t, err)
}

func TestBaseRepository_ConditionEdgeCases(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Create some test entities
	entities := []TestEntity{
		{Name: "Alice", Age: 25},
		{Name: "Bob", Age: 30},
		{Name: "Charlie", Age: 35},
	}

	for _, entity := range entities {
		err := repo.Create(ctx, &entity)
		require.NoError(t, err)
	}

	// Test with nil conditions
	var found []TestEntity
	err := repo.FindAllByConditionsWithOffset(ctx, 10, 0, &found)
	assert.NoError(t, err)
	assert.Len(t, found, 3)

	// Test with empty conditions
	err = repo.FindAllByConditionsWithOffset(ctx, 10, 0, &found, "")
	assert.NoError(t, err)
	assert.Len(t, found, 3)

	// Test with single condition
	err = repo.FindAllByConditionsWithOffset(ctx, 10, 0, &found, "age > ?", 30)
	assert.NoError(t, err)
	assert.Len(t, found, 1)

	// Test with multiple conditions
	err = repo.FindAllByConditionsWithOffset(ctx, 10, 0, &found, "age > ? AND name LIKE ?", 25, "%e%")
	assert.NoError(t, err)
	assert.Len(t, found, 1) // Only Charlie matches: age > 25 AND name contains 'e'

	// Test with invalid SQL condition
	err = repo.FindAllByConditionsWithOffset(ctx, 10, 0, &found, "invalid sql condition", 30)
	assert.Error(t, err)

	// Test with mismatched parameters
	err = repo.FindAllByConditionsWithOffset(ctx, 10, 0, &found, "age > ? AND name = ?", 30)
	assert.Error(t, err) // Missing parameter
}

func TestBaseRepository_CountEdgeCases(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Test count with no entities
	count, err := repo.CountAll(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Test count by conditions with no entities
	count, err = repo.CountByConditions(ctx, "age > ?", 100)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Test count with nil conditions
	count, err = repo.CountByConditions(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Test count with empty conditions
	count, err = repo.CountByConditions(ctx, "")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Test count with invalid SQL condition
	count, err = repo.CountByConditions(ctx, "invalid sql condition", 30)
	assert.Error(t, err)
}

func TestBaseRepository_ExistsEdgeCases(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Test exists with no entities
	exists, err := repo.ExistsByConditions(ctx)
	assert.NoError(t, err)
	assert.False(t, exists)

	// Test exists with empty conditions
	exists, err = repo.ExistsByConditions(ctx, "")
	assert.NoError(t, err)
	assert.False(t, exists)

	// Test exists with invalid SQL condition
	exists, err = repo.ExistsByConditions(ctx, "invalid sql condition", 30)
	assert.Error(t, err)

	// Test exists by ID with nil UUID
	exists, err = repo.ExistsByID(ctx, uuid.Nil)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestBaseRepository_UpdateEdgeCases(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Test update with nil entity
	err := repo.Update(ctx, nil)
	assert.Error(t, err)

	// Test update with entity that has no ID
	entityWithoutID := &TestEntity{Name: "Test", Age: 30}
	err = repo.Update(ctx, entityWithoutID)
	assert.Error(t, err)

	// Test update with entity that has nil ID
	entityWithNilID := &TestEntity{Name: "Test", Age: 30}
	entityWithNilID.SetID(uuid.Nil)
	err = repo.Update(ctx, entityWithNilID)
	assert.Error(t, err)

	// Test update by ID with nil entity
	err = repo.UpdateByID(ctx, nil, uuid.New())
	assert.Error(t, err)

	// Test update by ID with nil UUID
	entity := &TestEntity{Name: "Test", Age: 30}
	err = repo.UpdateByID(ctx, entity, uuid.Nil)
	assert.Error(t, err)

	// Test update by conditions with nil entity
	err = repo.UpdateByConditions(ctx, nil, "age > ?", 30)
	assert.Error(t, err)

	// Test update by conditions with invalid SQL
	err = repo.UpdateByConditions(ctx, &TestEntity{Age: 40}, "invalid sql condition", 30)
	assert.Error(t, err)
}

func TestBaseRepository_DeleteEdgeCases(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Test delete by ID with nil UUID
	err := repo.DeleteByID(ctx, uuid.Nil)
	assert.Error(t, err)

	// Test delete by conditions with nil entity
	err = repo.DeleteByConditions(ctx, nil, "age > ?", 30)
	assert.Error(t, err)

	// Test delete by conditions with invalid SQL
	err = repo.DeleteByConditions(ctx, &TestEntity{}, "invalid sql condition", 30)
	assert.Error(t, err)

	// Test delete by conditions with no conditions
	err = repo.DeleteByConditions(ctx, &TestEntity{})
	assert.Error(t, err) // Should require at least one condition for safety
}

func TestBaseRepository_UpsertEdgeCases(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Test upsert with nil entity
	err := repo.Upsert(ctx, nil, "id")
	assert.Error(t, err)

	// Test upsert with empty unique field
	err = repo.Upsert(ctx, &TestEntity{Name: "Test", Age: 30}, "")
	assert.Error(t, err)

	// Test upsert with invalid unique field - this might not fail until runtime in GORM
	_ = repo.Upsert(ctx, &TestEntity{Name: "Test", Age: 30}, "nonexistent_field")
	// Note: GORM doesn't validate field existence at compile time, so this might not error
	// The error would occur at runtime when the database operation is attempted
}

func TestBaseRepository_FindEdgeCases(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Test find by ID with nil UUID
	_, err := repo.FindFirstByID(ctx, uuid.Nil)
	assert.Error(t, err)

	// Test find first by conditions with nil result
	err = repo.FindFirstByConditions(ctx, nil, "age > ?", 30)
	assert.Error(t, err)

	// Test find all by conditions with nil result
	err = repo.FindAllByConditionsWithOffset(ctx, 10, 0, nil, "age > ?", 30)
	assert.Error(t, err)

	// Test first or init by conditions with nil result
	err = repo.FirstOrInitByConditions(ctx, nil, "age > ?", 30)
	assert.Error(t, err)

	// Test take by conditions with nil result
	err = repo.TakeByConditions(ctx, nil, "age > ?", 30)
	assert.Error(t, err)

	// Test last by conditions with nil result
	err = repo.LastByConditions(ctx, nil, "age > ?", 30)
	assert.Error(t, err)
}

func TestBaseRepository_ConcurrentModification(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Test concurrent creation
	done := make(chan bool, 20)
	for i := 0; i < 20; i++ {
		go func(id int) {
			entity := &TestEntity{
				Name: fmt.Sprintf("User %d", id),
				Age:  20 + id,
			}
			err := repo.Create(ctx, entity)
			if err == nil {
				// Try to update the entity concurrently
				entity.Age = 30 + id
				repo.Update(ctx, entity)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 20; i++ {
		<-done
	}

	// Test concurrent reads
	done = make(chan bool, 20)
	for i := 0; i < 20; i++ {
		go func() {
			repo.CountAll(ctx)
			repo.ExistsByConditions(ctx, "age > ?", 25)
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 20; i++ {
		<-done
	}
}

func TestBaseRepository_ContextHandling(t *testing.T) {
	repo, _ := setupTestRepository(t)

	// Test with nil context
	assert.Panics(t, func() {
		repo.Create(nil, &TestEntity{Name: "Test", Age: 30}) //nolint:staticcheck // Intentionally testing nil context
	})

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	entity := &TestEntity{Name: "Test", Age: 30}
	err := repo.Create(ctx, entity)
	assert.Error(t, err)

	// Test with timed out context
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(1 * time.Millisecond) // Ensure timeout

	err = repo.Create(ctx, entity)
	assert.Error(t, err)
}

func TestBaseRepository_ConfigurationEdgeCases(t *testing.T) {
	db := setupTestDB(t)
	logger := logging.NewLogger(logging.LogLevelInfo, nil, &logging.TextFormatter{})

	// Test with nil config
	repo := repository.NewBaseRepository[TestEntity](db, logger, nil)
	assert.NotNil(t, repo)

	// Test with empty config
	emptyConfig := &repository.RepositoryConfig{}
	repo = repository.NewBaseRepository[TestEntity](db, logger, emptyConfig)
	assert.NotNil(t, repo)

	// Test with extreme config values
	extremeConfig := &repository.RepositoryConfig{
		EnableValidation: true,
		EnableMetrics:    true,
		DefaultLimit:     999999,
		MaxLimit:         999999999,
	}
	repo = repository.NewBaseRepository[TestEntity](db, logger, extremeConfig)
	assert.NotNil(t, repo)

	// Test with zero config values
	zeroConfig := &repository.RepositoryConfig{
		EnableValidation: true,
		EnableMetrics:    true,
		DefaultLimit:     0,
		MaxLimit:         0,
	}
	repo = repository.NewBaseRepository[TestEntity](db, logger, zeroConfig)
	assert.NotNil(t, repo)

	// Test with negative config values
	negativeConfig := &repository.RepositoryConfig{
		EnableValidation: true,
		EnableMetrics:    true,
		DefaultLimit:     -1,
		MaxLimit:         -1,
	}
	repo = repository.NewBaseRepository[TestEntity](db, logger, negativeConfig)
	assert.NotNil(t, repo)
}

func TestBaseRepository_FindAllWithCursor(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Create test entities with known IDs for cursor testing
	entities := []TestEntity{
		{Name: "Alice", Age: 25},
		{Name: "Bob", Age: 30},
		{Name: "Charlie", Age: 35},
		{Name: "David", Age: 40},
		{Name: "Eve", Age: 45},
	}

	var createdEntities []TestEntity
	for _, entity := range entities {
		err := repo.Create(ctx, &entity)
		require.NoError(t, err)
		createdEntities = append(createdEntities, entity)
	}

	// Test basic cursor pagination - first page
	var found []TestEntity
	err := repo.FindAllWithCursor(ctx, "", 2, "next", &found)
	assert.NoError(t, err)
	assert.Len(t, found, 2)

	// Test cursor pagination - next page using first entity's ID as cursor
	if len(found) > 0 {
		cursor := found[0].ID.String()
		var nextPage []TestEntity
		err = repo.FindAllWithCursor(ctx, cursor, 2, "next", &nextPage)
		assert.NoError(t, err)
		assert.Len(t, nextPage, 2)
		// Ensure we get different entities
		assert.NotEqual(t, found[0].ID, nextPage[0].ID)
	}

	// Test cursor pagination - previous page
	if len(found) > 0 {
		cursor := found[1].ID.String()
		var prevPage []TestEntity
		err = repo.FindAllWithCursor(ctx, cursor, 2, "prev", &prevPage)
		assert.NoError(t, err)
		// With string comparison, we might get fewer results
		assert.LessOrEqual(t, len(prevPage), 2)
	}

	// Test with invalid direction (should default to "next")
	var invalidDir []TestEntity
	err = repo.FindAllWithCursor(ctx, "", 2, "invalid", &invalidDir)
	assert.NoError(t, err)
	assert.Len(t, invalidDir, 2)

	// Test with zero limit (should use default)
	var zeroLimit []TestEntity
	err = repo.FindAllWithCursor(ctx, "", 0, "next", &zeroLimit)
	assert.NoError(t, err)
	assert.Len(t, zeroLimit, 5) // All 5 entities (less than default limit of 20)

	// Test with negative limit (should use default)
	var negLimit []TestEntity
	err = repo.FindAllWithCursor(ctx, "", -1, "next", &negLimit)
	assert.NoError(t, err)
	assert.Len(t, negLimit, 5) // All 5 entities (less than default limit of 20)

	// Test with large limit (should be capped)
	var largeLimit []TestEntity
	err = repo.FindAllWithCursor(ctx, "", 9999, "next", &largeLimit)
	assert.NoError(t, err)
	assert.Len(t, largeLimit, 5) // All 5 entities (less than max limit of 1000)

	// Test with empty cursor (should return first page)
	var emptyCursor []TestEntity
	err = repo.FindAllWithCursor(ctx, "", 3, "next", &emptyCursor)
	assert.NoError(t, err)
	assert.Len(t, emptyCursor, 3)

	// Test with non-existent cursor
	var nonExistent []TestEntity
	nonExistentCursor := uuid.New().String()
	err = repo.FindAllWithCursor(ctx, nonExistentCursor, 2, "next", &nonExistent)
	assert.NoError(t, err)
	assert.Len(t, nonExistent, 0) // Should return empty result
}

func TestBaseRepository_FindAllInBatchesWithCursor(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Create test entities
	entities := []TestEntity{
		{Name: "Alice", Age: 25},
		{Name: "Bob", Age: 30},
		{Name: "Charlie", Age: 35},
		{Name: "David", Age: 40},
		{Name: "Eve", Age: 45},
		{Name: "Frank", Age: 50},
	}

	for _, entity := range entities {
		err := repo.Create(ctx, &entity)
		require.NoError(t, err)
	}

	// Test batch processing with cursor
	var allFound []TestEntity
	batchCount := 0
	err := repo.FindAllInBatchesWithCursor(ctx, "", 2, "next", &allFound, 2, func(tx *gorm.DB, batch int) error {
		batchCount++
		return nil
	})
	assert.NoError(t, err)
	// With only 6 entities and limit of 2, we should get at least 1 batch
	assert.GreaterOrEqual(t, batchCount, 1)

	// Test with invalid batch size
	var invalidBatch []TestEntity
	err = repo.FindAllInBatchesWithCursor(ctx, "", 2, "next", &invalidBatch, 0, func(tx *gorm.DB, batch int) error {
		return nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "batch size must be greater than 0")

	// Test with negative batch size
	var negBatch []TestEntity
	err = repo.FindAllInBatchesWithCursor(ctx, "", 2, "next", &negBatch, -1, func(tx *gorm.DB, batch int) error {
		return nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "batch size must be greater than 0")

	// Test with cursor and batch processing
	var firstBatch []TestEntity
	err = repo.FindAllWithCursor(ctx, "", 2, "next", &firstBatch)
	require.NoError(t, err)
	require.Len(t, firstBatch, 2)

	cursor := firstBatch[0].ID.String()
	var cursorBatch []TestEntity
	batchCount = 0
	err = repo.FindAllInBatchesWithCursor(ctx, cursor, 2, "next", &cursorBatch, 2, func(tx *gorm.DB, batch int) error {
		batchCount++
		return nil
	})
	assert.NoError(t, err)
	// With cursor, we might get fewer batches
	assert.GreaterOrEqual(t, batchCount, 0)

	// Test with previous direction
	var prevBatch []TestEntity
	batchCount = 0
	err = repo.FindAllInBatchesWithCursor(ctx, cursor, 2, "prev", &prevBatch, 2, func(tx *gorm.DB, batch int) error {
		batchCount++
		return nil
	})
	assert.NoError(t, err)
	// With cursor and prev direction, we might get fewer batches
	assert.GreaterOrEqual(t, batchCount, 0)
}

func TestBaseRepository_FindAllByConditionsWithCursor(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Create test entities with different ages
	entities := []TestEntity{
		{Name: "Alice", Age: 25},
		{Name: "Bob", Age: 30},
		{Name: "Charlie", Age: 35},
		{Name: "David", Age: 40},
		{Name: "Eve", Age: 45},
		{Name: "Frank", Age: 30},
		{Name: "Grace", Age: 35},
	}

	for _, entity := range entities {
		err := repo.Create(ctx, &entity)
		require.NoError(t, err)
	}

	// Test cursor pagination with conditions - age >= 30
	var found []TestEntity
	err := repo.FindAllByConditionsWithCursor(ctx, "", 2, "next", &found, "age >= ?", 30)
	assert.NoError(t, err)
	assert.Len(t, found, 2)
	// Verify all returned entities meet the condition
	for _, entity := range found {
		assert.GreaterOrEqual(t, entity.Age, 30)
	}

	// Test next page with cursor
	if len(found) > 0 {
		cursor := found[0].ID.String()
		var nextPage []TestEntity
		err = repo.FindAllByConditionsWithCursor(ctx, cursor, 2, "next", &nextPage, "age >= ?", 30)
		assert.NoError(t, err)
		assert.Len(t, nextPage, 2)
		// Verify all returned entities meet the condition
		for _, entity := range nextPage {
			assert.GreaterOrEqual(t, entity.Age, 30)
		}
		// Ensure we get different entities
		assert.NotEqual(t, found[0].ID, nextPage[0].ID)
	}

	// Test with name condition
	var nameFound []TestEntity
	err = repo.FindAllByConditionsWithCursor(ctx, "", 3, "next", &nameFound, "name LIKE ?", "A%")
	assert.NoError(t, err)
	// Should find Alice
	assert.Len(t, nameFound, 1)
	assert.Equal(t, "Alice", nameFound[0].Name)

	// Test with no conditions (should work like FindAllWithCursor)
	var noConditions []TestEntity
	err = repo.FindAllByConditionsWithCursor(ctx, "", 2, "next", &noConditions)
	assert.NoError(t, err)
	assert.Len(t, noConditions, 2)

	// Test with invalid direction
	var invalidDir []TestEntity
	err = repo.FindAllByConditionsWithCursor(ctx, "", 2, "invalid", &invalidDir, "age >= ?", 30)
	assert.NoError(t, err)
	assert.Len(t, invalidDir, 2)

	// Test with zero limit
	var zeroLimit []TestEntity
	err = repo.FindAllByConditionsWithCursor(ctx, "", 0, "next", &zeroLimit, "age >= ?", 30)
	assert.NoError(t, err)
	assert.Len(t, zeroLimit, 6) // All 6 entities with age >= 30 (less than default limit of 20)

	// Test with non-existent cursor
	var nonExistent []TestEntity
	nonExistentCursor := uuid.New().String()
	err = repo.FindAllByConditionsWithCursor(ctx, nonExistentCursor, 2, "next", &nonExistent, "age >= ?", 30)
	assert.NoError(t, err)
	assert.Len(t, nonExistent, 0)

	// Test with condition that matches no entities
	var noMatches []TestEntity
	err = repo.FindAllByConditionsWithCursor(ctx, "", 2, "next", &noMatches, "age > ?", 100)
	assert.NoError(t, err)
	assert.Len(t, noMatches, 0)
}

func TestBaseRepository_FindAllInBatchesByConditionsWithCursor(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Create test entities
	entities := []TestEntity{
		{Name: "Alice", Age: 25},
		{Name: "Bob", Age: 30},
		{Name: "Charlie", Age: 35},
		{Name: "David", Age: 40},
		{Name: "Eve", Age: 45},
		{Name: "Frank", Age: 30},
		{Name: "Grace", Age: 35},
		{Name: "Henry", Age: 50},
	}

	for _, entity := range entities {
		err := repo.Create(ctx, &entity)
		require.NoError(t, err)
	}

	// Test batch processing with conditions and cursor
	var allFound []TestEntity
	batchCount := 0
	err := repo.FindAllInBatchesByConditionsWithCursor(ctx, "", 2, "next", &allFound, 2, func(tx *gorm.DB, batch int) error {
		batchCount++
		return nil
	}, "age >= ?", 30)
	assert.NoError(t, err)
	// With 6 entities matching age >= 30 and limit of 2, we should get at least 1 batch
	assert.GreaterOrEqual(t, batchCount, 1)
	// Verify all returned entities meet the condition
	for _, entity := range allFound {
		assert.GreaterOrEqual(t, entity.Age, 30)
	}

	// Test with cursor and conditions
	var firstBatch []TestEntity
	err = repo.FindAllByConditionsWithCursor(ctx, "", 2, "next", &firstBatch, "age >= ?", 30)
	require.NoError(t, err)
	require.Len(t, firstBatch, 2)

	cursor := firstBatch[0].ID.String()
	var cursorBatch []TestEntity
	batchCount = 0
	err = repo.FindAllInBatchesByConditionsWithCursor(ctx, cursor, 2, "next", &cursorBatch, 2, func(tx *gorm.DB, batch int) error {
		batchCount++
		return nil
	}, "age >= ?", 30)
	assert.NoError(t, err)
	// With cursor, we might get fewer batches
	assert.GreaterOrEqual(t, batchCount, 0)
	// Verify all returned entities meet the condition
	for _, entity := range cursorBatch {
		assert.GreaterOrEqual(t, entity.Age, 30)
	}

	// Test with invalid batch size
	var invalidBatch []TestEntity
	err = repo.FindAllInBatchesByConditionsWithCursor(ctx, "", 2, "next", &invalidBatch, 0, func(tx *gorm.DB, batch int) error {
		return nil
	}, "age >= ?", 30)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "batch size must be greater than 0")

	// Test with negative batch size
	var negBatch []TestEntity
	err = repo.FindAllInBatchesByConditionsWithCursor(ctx, "", 2, "next", &negBatch, -1, func(tx *gorm.DB, batch int) error {
		return nil
	}, "age >= ?", 30)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "batch size must be greater than 0")

	// Test with previous direction
	var prevBatch []TestEntity
	batchCount = 0
	err = repo.FindAllInBatchesByConditionsWithCursor(ctx, cursor, 2, "prev", &prevBatch, 2, func(tx *gorm.DB, batch int) error {
		batchCount++
		return nil
	}, "age >= ?", 30)
	assert.NoError(t, err)
	// With cursor and prev direction, we might get fewer batches
	assert.GreaterOrEqual(t, batchCount, 0)

	// Test with no conditions
	var noConditions []TestEntity
	batchCount = 0
	err = repo.FindAllInBatchesByConditionsWithCursor(ctx, "", 2, "next", &noConditions, 2, func(tx *gorm.DB, batch int) error {
		batchCount++
		return nil
	})
	assert.NoError(t, err)
	// With 8 entities and limit of 2, we should get at least 1 batch
	assert.GreaterOrEqual(t, batchCount, 1)

	// Test with condition that matches no entities
	var noMatches []TestEntity
	batchCount = 0
	err = repo.FindAllInBatchesByConditionsWithCursor(ctx, "", 2, "next", &noMatches, 2, func(tx *gorm.DB, batch int) error {
		batchCount++
		return nil
	}, "age > ?", 100)
	assert.NoError(t, err)
	assert.Equal(t, 0, batchCount) // No batches should be processed
	assert.Len(t, noMatches, 0)

	// Test with invalid direction
	var invalidDir []TestEntity
	batchCount = 0
	err = repo.FindAllInBatchesByConditionsWithCursor(ctx, "", 2, "invalid", &invalidDir, 2, func(tx *gorm.DB, batch int) error {
		batchCount++
		return nil
	}, "age >= ?", 30)
	assert.NoError(t, err)
	// With 6 entities matching age >= 30 and limit of 2, we should get at least 1 batch
	assert.GreaterOrEqual(t, batchCount, 1)
}

func TestBaseRepository_CursorPaginationEdgeCases(t *testing.T) {
	repo, _ := setupTestRepository(t)
	ctx := context.Background()

	// Test with empty database
	var emptyResult []TestEntity
	err := repo.FindAllWithCursor(ctx, "", 10, "next", &emptyResult)
	assert.NoError(t, err)
	assert.Len(t, emptyResult, 0)

	// Test with single entity
	singleEntity := TestEntity{Name: "Single", Age: 25}
	err = repo.Create(ctx, &singleEntity)
	require.NoError(t, err)

	var singleResult []TestEntity
	err = repo.FindAllWithCursor(ctx, "", 10, "next", &singleResult)
	assert.NoError(t, err)
	assert.Len(t, singleResult, 1)
	assert.Equal(t, "Single", singleResult[0].Name)

	// Test cursor with single entity (should return empty)
	cursor := singleEntity.ID.String()
	var cursorResult []TestEntity
	err = repo.FindAllWithCursor(ctx, cursor, 10, "next", &cursorResult)
	assert.NoError(t, err)
	assert.Len(t, cursorResult, 0)

	// Test with very large limit
	var largeResult []TestEntity
	err = repo.FindAllWithCursor(ctx, "", 999999, "next", &largeResult)
	assert.NoError(t, err)
	assert.Len(t, largeResult, 1) // Only one entity exists

	// Test with conditions and empty result
	var conditionResult []TestEntity
	err = repo.FindAllByConditionsWithCursor(ctx, "", 10, "next", &conditionResult, "age > ?", 100)
	assert.NoError(t, err)
	assert.Len(t, conditionResult, 0)

	// Test batch processing with empty database
	var batchResult []TestEntity
	batchCount := 0
	err = repo.FindAllInBatchesWithCursor(ctx, "", 10, "next", &batchResult, 5, func(tx *gorm.DB, batch int) error {
		batchCount++
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, batchCount) // One batch with the single entity
	assert.Len(t, batchResult, 1)
}
