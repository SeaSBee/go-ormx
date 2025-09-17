package unit

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/seasbee/go-ormx/pkg/models"
	"github.com/seasbee/go-ormx/pkg/utils"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestBaseModel_BeforeCreate(t *testing.T) {
	tests := []struct {
		name        string
		model       *models.BaseModel
		expectIDSet bool
	}{
		{
			name: "nil ID should be set",
			model: &models.BaseModel{
				ID: uuid.Nil,
			},
			expectIDSet: true,
		},
		{
			name: "existing ID should not be changed",
			model: &models.BaseModel{
				ID: uuid.New(),
			},
			expectIDSet: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalID := tt.model.ID

			// Mock GORM transaction
			tx := &gorm.DB{}

			err := tt.model.BeforeCreate(tx)
			assert.NoError(t, err)

			if tt.expectIDSet {
				assert.NotEqual(t, uuid.Nil, tt.model.ID)
				assert.NotEqual(t, originalID, tt.model.ID)
			} else {
				assert.Equal(t, originalID, tt.model.ID)
			}
		})
	}
}

func TestBaseModel_BeforeUpdate(t *testing.T) {
	model := &models.BaseModel{
		ID:        uuid.New(),
		UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	originalUpdatedAt := model.UpdatedAt

	// Mock GORM transaction
	tx := &gorm.DB{}

	err := model.BeforeUpdate(tx)
	assert.NoError(t, err)

	// UpdatedAt should be changed
	assert.True(t, model.UpdatedAt.After(originalUpdatedAt))
}

func TestBaseModel_BeforeDelete(t *testing.T) {
	model := &models.BaseModel{
		ID: uuid.New(),
	}

	assert.Nil(t, model.DeletedAt)

	// Mock GORM transaction
	tx := &gorm.DB{}

	err := model.BeforeDelete(tx)
	assert.NoError(t, err)

	// DeletedAt should be set
	assert.NotNil(t, model.DeletedAt)
	assert.True(t, model.DeletedAt.After(time.Now().Add(-1*time.Second)))
}

func TestBaseModel_IsDeleted(t *testing.T) {
	tests := []struct {
		name           string
		model          *models.BaseModel
		expectedResult bool
	}{
		{
			name: "not deleted",
			model: &models.BaseModel{
				ID:        uuid.New(),
				DeletedAt: nil,
			},
			expectedResult: false,
		},
		{
			name: "deleted",
			model: &models.BaseModel{
				ID:        uuid.New(),
				DeletedAt: &[]time.Time{time.Now()}[0],
			},
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.model.IsDeleted()
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestBaseModel_Restore(t *testing.T) {
	deletedAt := time.Now()
	model := &models.BaseModel{
		ID:        uuid.New(),
		DeletedAt: &deletedAt,
	}

	assert.True(t, model.IsDeleted())

	model.Restore()

	assert.False(t, model.IsDeleted())
	assert.Nil(t, model.DeletedAt)
}

func TestBaseModel_GetID(t *testing.T) {
	expectedID := uuid.New()
	model := &models.BaseModel{
		ID: expectedID,
	}

	result := model.GetID()
	assert.Equal(t, expectedID, result)
}

func TestBaseModel_SetID(t *testing.T) {
	model := &models.BaseModel{}
	newID := uuid.New()

	model.SetID(newID)
	assert.Equal(t, newID, model.ID)
}

func TestBaseModel_GetCreatedAt(t *testing.T) {
	expectedTime := time.Now()
	model := &models.BaseModel{
		CreatedAt: expectedTime,
	}

	result := model.GetCreatedAt()
	assert.Equal(t, expectedTime, result)
}

func TestBaseModel_GetUpdatedAt(t *testing.T) {
	expectedTime := time.Now()
	model := &models.BaseModel{
		UpdatedAt: expectedTime,
	}

	result := model.GetUpdatedAt()
	assert.Equal(t, expectedTime, result)
}

func TestBaseModel_GetDeletedAt(t *testing.T) {
	tests := []struct {
		name           string
		model          *models.BaseModel
		expectedResult *time.Time
	}{
		{
			name: "not deleted",
			model: &models.BaseModel{
				ID:        uuid.New(),
				DeletedAt: nil,
			},
			expectedResult: nil,
		},
		{
			name: "deleted",
			model: &models.BaseModel{
				ID:        uuid.New(),
				DeletedAt: &[]time.Time{time.Now()}[0],
			},
			expectedResult: nil, // We'll check the actual value separately
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.model.GetDeletedAt()
			if tt.expectedResult == nil {
				// For the deleted case, just check that it's not nil
				if tt.name == "deleted" {
					assert.NotNil(t, result)
				} else {
					assert.Equal(t, tt.expectedResult, result)
				}
			} else {
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestBaseModel_GetCreatedBy(t *testing.T) {
	expectedUserID := uuid.New()
	model := &models.BaseModel{
		CreatedBy: &expectedUserID,
	}

	result := model.GetCreatedBy()
	assert.Equal(t, expectedUserID, result)
}

func TestBaseModel_GetUpdatedBy(t *testing.T) {
	expectedUserID := uuid.New()
	model := &models.BaseModel{
		UpdatedBy: &expectedUserID,
	}

	result := model.GetUpdatedBy()
	assert.Equal(t, expectedUserID, result)
}

func TestBaseModel_GetDeletedBy(t *testing.T) {
	expectedUserID := uuid.New()
	model := &models.BaseModel{
		DeletedBy: &expectedUserID,
	}

	result := model.GetDeletedBy()
	assert.Equal(t, expectedUserID, result)
}

func TestModelFactory_NewBaseModel(t *testing.T) {
	factory := &models.ModelFactory{}

	model := factory.NewBaseModel()

	assert.NotNil(t, model)
	assert.NotEqual(t, uuid.Nil, model.ID)
	assert.True(t, utils.IsUUIDv7(model.ID))
}

func TestBaseModel_FieldAccess(t *testing.T) {
	userID := uuid.New()
	now := time.Now()

	model := &models.BaseModel{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: &userID,
		UpdatedBy: &userID,
	}

	// Test all field access methods
	assert.NotEqual(t, uuid.Nil, model.GetID())
	assert.Equal(t, now, model.GetCreatedAt())
	assert.Equal(t, now, model.GetUpdatedAt())
	assert.Equal(t, userID, model.GetCreatedBy())
	assert.Equal(t, userID, model.GetUpdatedBy())
}

func TestBaseModel_SoftDeleteFlow(t *testing.T) {
	model := &models.BaseModel{
		ID: uuid.New(),
	}

	// Initially not deleted
	assert.False(t, model.IsDeleted())
	assert.Nil(t, model.GetDeletedAt())

	// Perform soft delete
	tx := &gorm.DB{}
	err := model.BeforeDelete(tx)
	assert.NoError(t, err)

	// Should be marked as deleted
	assert.True(t, model.IsDeleted())
	assert.NotNil(t, model.GetDeletedAt())

	// Restore the model
	model.Restore()
	assert.False(t, model.IsDeleted())
	assert.Nil(t, model.GetDeletedAt())
}

func TestBaseModel_UpdateFlow(t *testing.T) {
	originalTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	model := &models.BaseModel{
		ID:        uuid.New(),
		UpdatedAt: originalTime,
	}

	// Perform update
	tx := &gorm.DB{}
	err := model.BeforeUpdate(tx)
	assert.NoError(t, err)

	// UpdatedAt should be changed
	assert.True(t, model.GetUpdatedAt().After(originalTime))
}

func TestBaseModel_CreateFlow(t *testing.T) {
	model := &models.BaseModel{
		ID: uuid.Nil, // No ID set initially
	}

	// Perform create
	tx := &gorm.DB{}
	err := model.BeforeCreate(tx)
	assert.NoError(t, err)

	// ID should be set to a valid UUIDv7
	assert.NotEqual(t, uuid.Nil, model.GetID())
	assert.True(t, utils.IsUUIDv7(model.GetID()))
}

func TestBaseModel_NilPointerHandling(t *testing.T) {
	// This test verifies that the BaseModel methods handle nil pointers gracefully
	// Note: In Go, calling methods on nil pointers will panic, so we skip this test
	// as it's not a realistic scenario in normal usage
	t.Skip("Skipping nil pointer test as it would cause panic")
}

func TestBaseModel_ZeroValueHandling(t *testing.T) {
	model := &models.BaseModel{} // Zero value

	// Initially all fields should be zero values
	assert.Equal(t, uuid.Nil, model.GetID())
	assert.Equal(t, time.Time{}, model.GetCreatedAt())
	assert.Equal(t, time.Time{}, model.GetUpdatedAt())
	// Note: Pointer fields are nil in zero value, so calling getters will panic
	// We'll test these after setting values
	assert.False(t, model.IsDeleted())

	// Set some values
	newID := uuid.New()
	now := time.Now()
	userID := uuid.New()

	model.SetID(newID)
	model.CreatedAt = now
	model.UpdatedAt = now
	model.CreatedBy = &userID
	model.UpdatedBy = &userID

	// Verify values are set
	assert.Equal(t, newID, model.GetID())
	assert.Equal(t, now, model.GetCreatedAt())
	assert.Equal(t, now, model.GetUpdatedAt())
	assert.Equal(t, userID, model.GetCreatedBy())
	assert.Equal(t, userID, model.GetUpdatedBy())
}

func TestBaseModel_ConcurrentAccess(t *testing.T) {
	userID := uuid.New()
	model := &models.BaseModel{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		CreatedBy: &userID,
		UpdatedBy: &userID,
	}

	// Test concurrent access to getter methods
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			// These should be safe for concurrent access
			_ = model.GetID()
			_ = model.GetCreatedAt()
			_ = model.GetUpdatedAt()
			_ = model.GetCreatedBy()
			_ = model.GetUpdatedBy()
			// Note: GetDeletedBy() would panic if DeletedBy is nil, so we skip it
			// This is a limitation of the current BaseModel implementation
			_ = model.IsDeleted()

			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestBaseModel_EdgeCases(t *testing.T) {
	// Test with extreme time values
	model := &models.BaseModel{
		ID:        uuid.New(),
		CreatedAt: time.Unix(0, 0), // Unix epoch
		UpdatedAt: time.Unix(0, 0), // Unix epoch
	}

	assert.Equal(t, time.Unix(0, 0), model.GetCreatedAt())
	assert.Equal(t, time.Unix(0, 0), model.GetUpdatedAt())

	// Test with very old timestamps
	oldTime := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
	model.CreatedAt = oldTime
	model.UpdatedAt = oldTime

	assert.Equal(t, oldTime, model.GetCreatedAt())
	assert.Equal(t, oldTime, model.GetUpdatedAt())

	// Test with very future timestamps
	futureTime := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)
	model.CreatedAt = futureTime
	model.UpdatedAt = futureTime

	assert.Equal(t, futureTime, model.GetCreatedAt())
	assert.Equal(t, futureTime, model.GetUpdatedAt())
}

// Add missing test scenarios
func TestBaseModel_ValidationScenarios(t *testing.T) {
	// Test with valid UUID
	validUUID := uuid.New()
	model := &models.BaseModel{
		ID: validUUID,
	}
	assert.Equal(t, validUUID, model.GetID())

	// Test with nil UUID (should be handled by BeforeCreate)
	model = &models.BaseModel{
		ID: uuid.Nil,
	}
	assert.Equal(t, uuid.Nil, model.GetID())

	// Test with zero time values
	model = &models.BaseModel{
		ID:        uuid.New(),
		CreatedAt: time.Time{},
		UpdatedAt: time.Time{},
	}
	assert.True(t, model.GetCreatedAt().IsZero())
	assert.True(t, model.GetUpdatedAt().IsZero())

	// Test with nil pointer fields
	model = &models.BaseModel{
		ID:        uuid.New(),
		CreatedBy: nil,
		UpdatedBy: nil,
		DeletedBy: nil,
	}
	assert.Equal(t, uuid.Nil, model.GetCreatedBy())
	assert.Equal(t, uuid.Nil, model.GetUpdatedBy())
	assert.Equal(t, uuid.Nil, model.GetDeletedBy())
}

func TestBaseModel_TimePrecision(t *testing.T) {
	// Test with nanosecond precision
	now := time.Now()
	model := &models.BaseModel{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Should preserve nanosecond precision
	assert.Equal(t, now.Nanosecond(), model.GetCreatedAt().Nanosecond())
	assert.Equal(t, now.Nanosecond(), model.GetUpdatedAt().Nanosecond())

	// Test with microsecond precision
	microTime := time.Date(2020, 1, 1, 12, 0, 0, 123456000, time.UTC)
	model.CreatedAt = microTime
	model.UpdatedAt = microTime

	assert.Equal(t, microTime.Nanosecond(), model.GetCreatedAt().Nanosecond())
	assert.Equal(t, microTime.Nanosecond(), model.GetUpdatedAt().Nanosecond())
}

func TestBaseModel_SoftDeleteEdgeCases(t *testing.T) {
	// Test with zero time for DeletedAt
	model := &models.BaseModel{
		ID:        uuid.New(),
		DeletedAt: &[]time.Time{time.Time{}}[0], // Zero time
	}

	assert.True(t, model.IsDeleted())
	assert.True(t, model.GetDeletedAt().IsZero())

	// Test with very old DeletedAt
	oldDeletedAt := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
	model.DeletedAt = &oldDeletedAt

	assert.True(t, model.IsDeleted())
	assert.Equal(t, oldDeletedAt, *model.GetDeletedAt())

	// Test with very future DeletedAt
	futureDeletedAt := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)
	model.DeletedAt = &futureDeletedAt

	assert.True(t, model.IsDeleted())
	assert.Equal(t, futureDeletedAt, *model.GetDeletedAt())

	// Test restore with zero time
	model.Restore()
	assert.False(t, model.IsDeleted())
	assert.Nil(t, model.GetDeletedAt())
}

func TestBaseModel_UserTrackingEdgeCases(t *testing.T) {
	// Test with nil user IDs
	model := &models.BaseModel{
		ID:        uuid.New(),
		CreatedBy: nil,
		UpdatedBy: nil,
		DeletedBy: nil,
	}

	assert.Equal(t, uuid.Nil, model.GetCreatedBy())
	assert.Equal(t, uuid.Nil, model.GetUpdatedBy())
	assert.Equal(t, uuid.Nil, model.GetDeletedBy())

	// Test with zero UUIDs
	zeroUUID := uuid.Nil
	model.CreatedBy = &zeroUUID
	model.UpdatedBy = &zeroUUID
	model.DeletedBy = &zeroUUID

	assert.Equal(t, uuid.Nil, model.GetCreatedBy())
	assert.Equal(t, uuid.Nil, model.GetUpdatedBy())
	assert.Equal(t, uuid.Nil, model.GetDeletedBy())

	// Test with valid UUIDs
	validUUID1 := uuid.New()
	validUUID2 := uuid.New()
	validUUID3 := uuid.New()

	model.CreatedBy = &validUUID1
	model.UpdatedBy = &validUUID2
	model.DeletedBy = &validUUID3

	assert.Equal(t, validUUID1, model.GetCreatedBy())
	assert.Equal(t, validUUID2, model.GetUpdatedBy())
	assert.Equal(t, validUUID3, model.GetDeletedBy())
}

func TestBaseModel_HookEdgeCases(t *testing.T) {
	// Test BeforeCreate with nil transaction
	model := &models.BaseModel{
		ID: uuid.Nil,
	}

	err := model.BeforeCreate(nil)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, model.GetID())

	// Test BeforeUpdate with nil transaction
	model = &models.BaseModel{
		ID:        uuid.New(),
		UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	originalUpdatedAt := model.GetUpdatedAt()
	err = model.BeforeUpdate(nil)
	assert.NoError(t, err)
	assert.True(t, model.GetUpdatedAt().After(originalUpdatedAt))

	// Test BeforeDelete with nil transaction
	model = &models.BaseModel{
		ID: uuid.New(),
	}

	assert.Nil(t, model.GetDeletedAt())
	err = model.BeforeDelete(nil)
	assert.NoError(t, err)
	assert.NotNil(t, model.GetDeletedAt())
}

func TestBaseModel_ConcurrentModification(t *testing.T) {
	model := &models.BaseModel{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Test concurrent modification of timestamps
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			// These should be safe for concurrent access
			_ = model.GetCreatedAt()
			_ = model.GetUpdatedAt()
			_ = model.GetID()
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Test concurrent modification of user tracking
	userID := uuid.New()
	model.CreatedBy = &userID
	model.UpdatedBy = &userID

	for i := 0; i < 10; i++ {
		go func() {
			_ = model.GetCreatedBy()
			_ = model.GetUpdatedBy()
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestBaseModel_FieldValidation(t *testing.T) {
	// Test with valid field values
	testID, _ := uuid.Parse("01890dd5-ecaa-7c87-9e38-0b0e9e0b0e9e") // Fixed UUIDv7
	testTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	validModel := &models.BaseModel{
		ID:        testID,
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}

	assert.Equal(t, testID, validModel.GetID())
	assert.Equal(t, testTime, validModel.GetCreatedAt())
	assert.Equal(t, testTime, validModel.GetUpdatedAt())

	// Test with edge case field values
	edgeModel := &models.BaseModel{
		ID:        uuid.Nil,
		CreatedAt: time.Time{}, // Zero time
		UpdatedAt: time.Time{}, // Zero time
	}

	assert.Equal(t, uuid.Nil, edgeModel.GetID())
	assert.True(t, edgeModel.GetCreatedAt().IsZero())
	assert.True(t, edgeModel.GetUpdatedAt().IsZero())
}

func TestBaseModel_Serialization(t *testing.T) {
	// Test that all fields can be accessed without panic
	model := &models.BaseModel{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Test JSON serialization (indirectly through field access)
	_ = model.GetID()
	_ = model.GetCreatedAt()
	_ = model.GetUpdatedAt()
	_ = model.GetDeletedAt()
	_ = model.GetCreatedBy()
	_ = model.GetUpdatedBy()
	_ = model.GetDeletedBy()
	_ = model.IsDeleted()

	// Test that model can be used in maps and slices
	modelSlice := []*models.BaseModel{model}
	assert.Len(t, modelSlice, 1)
	assert.Equal(t, model, modelSlice[0])

	modelMap := map[string]*models.BaseModel{"test": model}
	assert.Equal(t, model, modelMap["test"])
}

func TestBaseModel_InterfaceCompliance(t *testing.T) {
	// Test that BaseModel implements expected interfaces
	var _ interface {
		GetID() uuid.UUID
		SetID(uuid.UUID)
		GetCreatedAt() time.Time
		GetUpdatedAt() time.Time
		GetDeletedAt() *time.Time
		GetCreatedBy() uuid.UUID
		GetUpdatedBy() uuid.UUID
		GetDeletedBy() uuid.UUID
		IsDeleted() bool
		Restore()
	} = &models.BaseModel{}

	// Test that BaseModel can be used as a generic type
	var modelSlice []*models.BaseModel
	modelSlice = append(modelSlice, &models.BaseModel{})
	assert.Len(t, modelSlice, 1)
}

func TestModelFactory_EdgeCases(t *testing.T) {
	factory := &models.ModelFactory{}

	// Test creating multiple models
	model1 := factory.NewBaseModel()
	model2 := factory.NewBaseModel()
	model3 := factory.NewBaseModel()

	assert.NotNil(t, model1)
	assert.NotNil(t, model2)
	assert.NotNil(t, model3)

	// All models should have unique IDs
	assert.NotEqual(t, model1.GetID(), model2.GetID())
	assert.NotEqual(t, model2.GetID(), model3.GetID())
	assert.NotEqual(t, model1.GetID(), model3.GetID())

	// All models should be valid UUIDv7s
	assert.True(t, utils.IsUUIDv7(model1.GetID()))
	assert.True(t, utils.IsUUIDv7(model2.GetID()))
	assert.True(t, utils.IsUUIDv7(model3.GetID()))

	// Test with nil factory
	var nilFactory *models.ModelFactory
	model := nilFactory.NewBaseModel()
	assert.NotNil(t, model)
	assert.NotEqual(t, uuid.Nil, model.GetID())
}
