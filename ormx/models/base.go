// Package models provides GORM model definitions with validation and annotations.
// All models include comprehensive validation, soft delete support, and audit fields.
package models

import (
	"time"

	"github.com/oklog/ulid/v2"
	"gorm.io/gorm"
)

// BaseModel provides common fields for all models
type BaseModel struct {
	ID        string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// BeforeCreate GORM hook to set ID before creating
func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = ulid.Make().String()
	}
	return nil
}

// IsDeleted returns true if the model is soft deleted
func (b *BaseModel) IsDeleted() bool {
	return b.DeletedAt.Valid
}

// GetID returns the model ID
func (b *BaseModel) GetID() string {
	return b.ID
}

// SetID sets the model ID
func (b *BaseModel) SetID(id string) {
	b.ID = id
}

// TenantModel provides multi-tenancy support
type TenantModel struct {
	BaseModel
	TenantID string `gorm:"type:uuid;not null;index" json:"tenant_id"`
}

// AuditModel provides comprehensive audit trail
type AuditModel struct {
	BaseModel
	CreatedBy string  `gorm:"type:uuid;index" json:"created_by,omitempty"`
	UpdatedBy *string `gorm:"type:uuid;index" json:"updated_by,omitempty"`
	DeletedBy *string `gorm:"type:uuid;index" json:"deleted_by,omitempty"`
	Version   int64   `gorm:"not null;default:1" json:"version"`
}

// BeforeUpdate GORM hook to increment version
func (a *AuditModel) BeforeUpdate(tx *gorm.DB) error {
	a.Version++
	return nil
}

// TenantAuditModel combines tenant and audit functionality
type TenantAuditModel struct {
	BaseModel
	TenantID  string  `gorm:"type:uuid;not null;index" json:"tenant_id"`
	CreatedBy string  `gorm:"type:uuid;index" json:"created_by,omitempty"`
	UpdatedBy *string `gorm:"type:uuid;index" json:"updated_by,omitempty"`
	DeletedBy *string `gorm:"type:uuid;index" json:"deleted_by,omitempty"`
	Version   int64   `gorm:"not null;default:1" json:"version"`
}

// BeforeUpdate GORM hook to increment version
func (tam *TenantAuditModel) BeforeUpdate(tx *gorm.DB) error {
	tam.Version++
	return nil
}

// Modelable interface defines common model operations
// Note: All methods use pointer receivers for consistency
type Modelable interface {
	GetID() string
	SetID(string)
	IsDeleted() bool
	TableName() string
}

// Tenantable interface for multi-tenant models
type Tenantable interface {
	Modelable
	GetTenantID() string
	SetTenantID(string)
}

// GetTenantID returns the tenant ID for TenantModel
func (t *TenantModel) GetTenantID() string {
	return t.TenantID
}

// SetTenantID sets the tenant ID for TenantModel
func (t *TenantModel) SetTenantID(tenantID string) {
	t.TenantID = tenantID
}

// GetTenantID returns the tenant ID for TenantAuditModel
func (tam *TenantAuditModel) GetTenantID() string {
	return tam.TenantID
}

// SetTenantID sets the tenant ID for TenantAuditModel
func (tam *TenantAuditModel) SetTenantID(tenantID string) {
	tam.TenantID = tenantID
}

// Auditable interface for models with audit trail
type Auditable interface {
	Modelable
	GetCreatedBy() string
	SetCreatedBy(string)
	GetUpdatedBy() *string
	SetUpdatedBy(*string)
	GetVersion() int64
}

// GetCreatedBy returns the created by user ID for AuditModel
func (a *AuditModel) GetCreatedBy() string {
	return a.CreatedBy
}

// SetCreatedBy sets the created by user ID for AuditModel
func (a *AuditModel) SetCreatedBy(userID string) {
	a.CreatedBy = userID
}

// GetUpdatedBy returns the updated by user ID for AuditModel
func (a *AuditModel) GetUpdatedBy() *string {
	return a.UpdatedBy
}

// SetUpdatedBy sets the updated by user ID for AuditModel
func (a *AuditModel) SetUpdatedBy(userID *string) {
	a.UpdatedBy = userID
}

// GetVersion returns the version for AuditModel
func (a *AuditModel) GetVersion() int64 {
	return a.Version
}

// GetCreatedBy returns the created by user ID for TenantAuditModel
func (tam *TenantAuditModel) GetCreatedBy() string {
	return tam.CreatedBy
}

// SetCreatedBy sets the created by user ID for TenantAuditModel
func (tam *TenantAuditModel) SetCreatedBy(userID string) {
	tam.CreatedBy = userID
}

// GetUpdatedBy returns the updated by user ID for TenantAuditModel
func (tam *TenantAuditModel) GetUpdatedBy() *string {
	return tam.UpdatedBy
}

// SetUpdatedBy sets the updated by user ID for TenantAuditModel
func (tam *TenantAuditModel) SetUpdatedBy(userID *string) {
	tam.UpdatedBy = userID
}

// GetVersion returns the version for TenantAuditModel
func (tam *TenantAuditModel) GetVersion() int64 {
	return tam.Version
}

// ModelMetadata provides metadata about a model
type ModelMetadata struct {
	TableName     string            `json:"table_name"`
	PrimaryKey    string            `json:"primary_key"`
	Indexes       []string          `json:"indexes"`
	Constraints   []string          `json:"constraints"`
	Relationships []string          `json:"relationships"`
	Tags          map[string]string `json:"tags"`
}

// GetMetadata returns metadata about the model
func (b *BaseModel) GetMetadata() ModelMetadata {
	return ModelMetadata{
		TableName:  b.TableName(),
		PrimaryKey: "id",
		Indexes:    []string{"idx_created_at", "idx_updated_at", "idx_deleted_at"},
		Tags:       make(map[string]string),
	}
}

// TableName returns the table name for the model
func (b *BaseModel) TableName() string {
	return "base_models"
}

// Scope defines a database query scope
type Scope func(*gorm.DB) *gorm.DB

// NotDeleted returns a scope that filters out soft-deleted records
func NotDeleted() Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Unscoped().Where("deleted_at IS NULL")
	}
}

// ByTenant returns a scope that filters by tenant ID
func ByTenant(tenantID string) Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("tenant_id = ?", tenantID)
	}
}

// ByCreatedBy returns a scope that filters by created by user ID
func ByCreatedBy(userID string) Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("created_by = ?", userID)
	}
}

// CreatedAfter returns a scope that filters records created after a date
func CreatedAfter(date time.Time) Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("created_at > ?", date)
	}
}

// CreatedBefore returns a scope that filters records created before a date
func CreatedBefore(date time.Time) Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("created_at < ?", date)
	}
}

// UpdatedAfter returns a scope that filters records updated after a date
func UpdatedAfter(date time.Time) Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("updated_at > ?", date)
	}
}

// UpdatedBefore returns a scope that filters records updated before a date
func UpdatedBefore(date time.Time) Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("updated_at < ?", date)
	}
}

// OrderByCreatedAt returns a scope that orders by created_at
func OrderByCreatedAt(desc bool) Scope {
	return func(db *gorm.DB) *gorm.DB {
		if desc {
			return db.Order("created_at DESC")
		}
		return db.Order("created_at ASC")
	}
}

// OrderByUpdatedAt returns a scope that orders by updated_at
func OrderByUpdatedAt(desc bool) Scope {
	return func(db *gorm.DB) *gorm.DB {
		if desc {
			return db.Order("updated_at DESC")
		}
		return db.Order("updated_at ASC")
	}
}

// Limit returns a scope that limits the number of results
func Limit(limit int) Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Limit(limit)
	}
}

// Offset returns a scope that offsets the results
func Offset(offset int) Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Offset(offset)
	}
}

// ApplyScopes applies multiple scopes to a database query
func ApplyScopes(db *gorm.DB, scopes ...Scope) *gorm.DB {
	for _, scope := range scopes {
		db = scope(db)
	}
	return db
}
