package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/SeaSBee/go-ormx/pkg/utils"
	"gorm.io/gorm"
)

// BaseModel provides the foundation for all models with UUIDv7 primary key
type BaseModel struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key" json:"id" validate:"required"`
	CreatedAt time.Time  `gorm:"autoCreateTime;not null" json:"created_at" validate:"required"`
	UpdatedAt time.Time  `gorm:"autoUpdateTime;not null" json:"updated_at" validate:"required"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"`
	CreatedBy *uuid.UUID `gorm:"type:uuid;index" json:"created_by,omitempty"`
	UpdatedBy *uuid.UUID `gorm:"type:uuid;index" json:"updated_by,omitempty"`
	DeletedBy *uuid.UUID `gorm:"type:uuid;index" json:"deleted_by,omitempty"`
}

// BeforeCreate is called before creating a new record
func (m *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = utils.GenerateUUIDv7()
	}
	// Set CreatedBy from context if available
	if userID := getUserIDFromContext(tx); userID != uuid.Nil {
		m.CreatedBy = &userID
	}
	return nil
}

// BeforeUpdate is called before updating a record
func (m *BaseModel) BeforeUpdate(tx *gorm.DB) error {
	m.UpdatedAt = time.Now()
	// Set UpdatedBy from context if available
	if userID := getUserIDFromContext(tx); userID != uuid.Nil {
		m.UpdatedBy = &userID
	}
	return nil
}

// BeforeDelete is called before deleting a record
func (m *BaseModel) BeforeDelete(tx *gorm.DB) error {
	// Soft delete by setting DeletedAt
	now := time.Now()
	m.DeletedAt = &now
	return nil
}

// IsDeleted checks if the model is soft deleted
func (m *BaseModel) IsDeleted() bool {
	return m.DeletedAt != nil
}

// Restore restores a soft deleted model
func (m *BaseModel) Restore() {
	m.DeletedAt = nil
}

// GetID returns the model's ID
func (m *BaseModel) GetID() uuid.UUID {
	return m.ID
}

// SetID sets the model's ID
func (m *BaseModel) SetID(id uuid.UUID) {
	m.ID = id
}

// GetCreatedAt returns the creation timestamp
func (m *BaseModel) GetCreatedAt() time.Time {
	return m.CreatedAt
}

// GetUpdatedAt returns the last update timestamp
func (m *BaseModel) GetUpdatedAt() time.Time {
	return m.UpdatedAt
}

// GetDeletedAt returns the deletion timestamp
func (m *BaseModel) GetDeletedAt() *time.Time {
	return m.DeletedAt
}

// GetCreatedBy returns the user who created the record
func (m *BaseModel) GetCreatedBy() uuid.UUID {
	if m.CreatedBy == nil {
		return uuid.Nil
	}
	return *m.CreatedBy
}

// GetUpdatedBy returns the user who last updated the record
func (m *BaseModel) GetUpdatedBy() uuid.UUID {
	if m.UpdatedBy == nil {
		return uuid.Nil
	}
	return *m.UpdatedBy
}

// GetDeletedBy returns the user who deleted the record
func (m *BaseModel) GetDeletedBy() uuid.UUID {
	if m.DeletedBy == nil {
		return uuid.Nil
	}
	return *m.DeletedBy
}

// getUserIDFromContext extracts user ID from GORM context
func getUserIDFromContext(tx *gorm.DB) uuid.UUID {
	// TODO: Implement actual context extraction
	// This would typically extract user ID from the request context
	return uuid.Nil
}

// ModelFactory provides factory methods for creating models
type ModelFactory struct{}

// NewBaseModel creates a new BaseModel
func (f *ModelFactory) NewBaseModel() *BaseModel {
	return &BaseModel{
		ID: utils.GenerateUUIDv7(),
	}
}
