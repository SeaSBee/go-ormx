package models_test

import (
	"testing"
	"time"

	"go-ormx/ormx/models"

	"github.com/oklog/ulid/v2"
	"gorm.io/gorm"
)

// TestProduct represents a test product model
type TestProduct struct {
	models.AuditModel
	Name        string  `gorm:"type:varchar(255);not null" json:"name"`
	Description string  `gorm:"type:text" json:"description"`
	Price       float64 `gorm:"type:decimal(10,2);not null" json:"price"`
}

// TableName returns the table name for TestProduct
func (p *TestProduct) TableName() string {
	return "test_products"
}

func TestBaseModel_Structure(t *testing.T) {
	product := &TestProduct{
		Name:        "Test Product",
		Description: "A test product",
		Price:       29.99,
	}

	// Test that fields are accessible
	if product.Name != "Test Product" {
		t.Error("Expected Name to be set")
	}

	if product.Description != "A test product" {
		t.Error("Expected Description to be set")
	}

	if product.Price != 29.99 {
		t.Error("Expected Price to be set")
	}

	// Note: ID and timestamps are set by GORM hooks during database operations
	// They are not automatically set when creating struct instances
}

func TestBaseModel_GetID(t *testing.T) {
	product := &TestProduct{
		Name:  "Test Product",
		Price: 29.99,
	}

	// Set ID manually for testing
	product.SetID("test-id-123")

	id := product.GetID()
	if id != "test-id-123" {
		t.Error("Expected GetID to return the set ID")
	}
}

func TestBaseModel_GetCreatedAt(t *testing.T) {
	product := &TestProduct{
		Name:  "Test Product",
		Price: 29.99,
	}

	// Note: CreatedAt is zero for new instances
	// Timestamps are set by GORM during database operations
	createdAt := product.CreatedAt
	if !createdAt.IsZero() {
		t.Error("Expected CreatedAt to be zero for new instances")
	}
}

func TestBaseModel_GetUpdatedAt(t *testing.T) {
	product := &TestProduct{
		Name:  "Test Product",
		Price: 29.99,
	}

	// Note: UpdatedAt is zero for new instances
	// Timestamps are set by GORM during database operations
	updatedAt := product.UpdatedAt
	if !updatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be zero for new instances")
	}
}

func TestBaseModel_GetCreatedBy(t *testing.T) {
	product := &TestProduct{
		Name:  "Test Product",
		Price: 29.99,
	}

	createdBy := product.CreatedBy
	if createdBy != "" {
		t.Error("Expected GetCreatedBy to return empty string by default")
	}
}

func TestBaseModel_GetUpdatedBy(t *testing.T) {
	product := &TestProduct{
		Name:  "Test Product",
		Price: 29.99,
	}

	updatedBy := product.GetUpdatedBy()
	if updatedBy != nil {
		t.Error("Expected GetUpdatedBy to return nil by default")
	}
}

// Note: SetCreatedAt and SetUpdatedAt methods don't exist on BaseModel
// These tests are removed as they test non-existent functionality

func TestBaseModel_SetCreatedBy(t *testing.T) {
	product := &TestProduct{
		Name:  "Test Product",
		Price: 29.99,
	}

	userID := "user123"
	product.SetCreatedBy(userID)

	if product.CreatedBy != userID {
		t.Error("Expected CreatedBy to be set to the specified user ID")
	}
}

func TestBaseModel_SetUpdatedBy(t *testing.T) {
	product := &TestProduct{
		Name:  "Test Product",
		Price: 29.99,
	}

	userID := "user123"
	product.SetUpdatedBy(&userID)

	if *product.UpdatedBy != userID {
		t.Error("Expected UpdatedBy to be set to the specified user ID")
	}
}

func TestBaseModel_IsDeleted(t *testing.T) {
	product := &TestProduct{
		Name:  "Test Product",
		Price: 29.99,
	}

	// Test that non-deleted product returns false
	if product.IsDeleted() {
		t.Error("Expected IsDeleted to return false for non-deleted product")
	}

	// Test that deleted product returns true
	product.DeletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}
	if !product.IsDeleted() {
		t.Error("Expected IsDeleted to return true for deleted product")
	}
}

func TestBaseModel_TableName(t *testing.T) {
	product := &TestProduct{
		Name:  "Test Product",
		Price: 29.99,
	}

	expected := "test_products"
	if product.TableName() != expected {
		t.Errorf("Expected TableName to return %s, got %s", expected, product.TableName())
	}
}

func TestBaseModel_ID_Generation(t *testing.T) {
	product1 := &TestProduct{Name: "Product 1", Price: 10.0}
	product2 := &TestProduct{Name: "Product 2", Price: 20.0}

	// Set IDs manually for testing
	product1.SetID("test-id-1")
	product2.SetID("test-id-2")

	// Test that IDs are different
	if product1.ID == product2.ID {
		t.Error("Expected products to have different IDs")
	}

	// Test that IDs are set correctly
	if product1.ID != "test-id-1" {
		t.Errorf("Expected ID to be 'test-id-1', got %s", product1.ID)
	}

	if product2.ID != "test-id-2" {
		t.Errorf("Expected ID to be 'test-id-2', got %s", product2.ID)
	}
}

func BenchmarkBaseModel_Creation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = &TestProduct{
			Name:        "Benchmark Product",
			Description: "A product for benchmarking",
			Price:       99.99,
		}
	}
}

func BenchmarkBaseModel_ID_Generation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = ulid.Make().String()
	}
}
