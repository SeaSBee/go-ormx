package repositories_test

import (
	"testing"

	"go-ormx/ormx/logging"
	"go-ormx/ormx/models"
	"go-ormx/ormx/repositories"

	"gorm.io/gorm"
)

// Mock logger for testing
type MockLogger struct {
	logs []string
}

func (m *MockLogger) Trace(msg string, fields ...logging.LogField) {
	m.logs = append(m.logs, "TRACE: "+msg)
}
func (m *MockLogger) Debug(msg string, fields ...logging.LogField) {
	m.logs = append(m.logs, "DEBUG: "+msg)
}
func (m *MockLogger) Info(msg string, fields ...logging.LogField) {
	m.logs = append(m.logs, "INFO: "+msg)
}
func (m *MockLogger) Warn(msg string, fields ...logging.LogField) {
	m.logs = append(m.logs, "WARN: "+msg)
}
func (m *MockLogger) Error(msg string, fields ...logging.LogField) {
	m.logs = append(m.logs, "ERROR: "+msg)
}
func (m *MockLogger) Fatal(msg string, fields ...logging.LogField) {
	m.logs = append(m.logs, "FATAL: "+msg)
}
func (m *MockLogger) With(fields ...logging.LogField) logging.Logger { return m }

// Simple test model that implements Modelable
type TestModel struct {
	models.BaseModel
	Name string `json:"name"`
}

func (tm *TestModel) GetID() string     { return tm.ID }
func (tm *TestModel) TableName() string { return "test_models" }

func TestRepositoryOptions(t *testing.T) {
	// Test repository options structure
	options := repositories.RepositoryOptions{
		BatchSize:      100,
		ValidateOnSave: true,
	}

	if options.BatchSize != 100 {
		t.Error("BatchSize should be 100")
	}

	if !options.ValidateOnSave {
		t.Error("ValidateOnSave should be true")
	}
}

func TestBaseRepository_Creation(t *testing.T) {
	// Test that we can create a base repository without errors
	// This uses a nil GORM DB which is fine for testing structure
	var db *gorm.DB
	logger := &MockLogger{}
	options := repositories.RepositoryOptions{}

	repo := repositories.NewBaseRepository[*TestModel](db, logger, options)

	if repo == nil {
		t.Error("Expected non-nil repository")
	}
}

func TestFilter_Structure(t *testing.T) {
	// Test filter structure
	filter := repositories.Filter{
		Where: map[string]repositories.WhereCondition{
			"active": {Operator: "=", Value: true},
		},
		OrderBy: []repositories.OrderBy{{Field: "name", Direction: "ASC"}},
	}

	if filter.Where == nil {
		t.Error("Where should not be nil")
	}

	if len(filter.OrderBy) != 1 {
		t.Error("OrderBy should have one item")
	}

	if filter.OrderBy[0].Field != "name" {
		t.Error("OrderBy field should be 'name'")
	}

	if filter.OrderBy[0].Direction != "ASC" {
		t.Error("OrderBy direction should be 'ASC'")
	}
}

// Note: Cursor pagination functionality has been removed from the simplified repository
// These tests are removed as they test non-existent functionality

func TestOrderBy_Structure(t *testing.T) {
	// Test OrderBy structure
	orderBy := repositories.OrderBy{
		Field:     "created_at",
		Direction: "DESC",
	}

	if orderBy.Field != "created_at" {
		t.Error("Field should be 'created_at'")
	}

	if orderBy.Direction != "DESC" {
		t.Error("Direction should be 'DESC'")
	}
}

// Benchmark tests
func BenchmarkBaseRepository_Creation(b *testing.B) {
	var db *gorm.DB
	logger := &MockLogger{}
	options := repositories.RepositoryOptions{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = repositories.NewBaseRepository[*TestModel](db, logger, options)
	}
}
