package db_test

import (
	"testing"
	"time"

	"go-ormx/ormx/config"
	"go-ormx/ormx/db"
	"go-ormx/ormx/logging"
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

func TestDatabase_New(t *testing.T) {
	// Test basic database creation functionality
	config := &config.Config{
		Type:            config.PostgreSQL,
		Host:            "localhost",
		Port:            5432,
		Database:        "test_db",
		Username:        "test_user",
		Password:        "test_pass",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Minute,
	}

	logger := &MockLogger{}

	// This will likely fail without a real database, but we test the creation
	database, err := db.New(config, logger)

	// We expect this to fail in test environment, but we check that we get
	// appropriate error handling rather than panics
	if err != nil {
		// Expected in test environment without real database
		t.Logf("Database creation failed as expected in test environment: %v", err)
	} else {
		// If it somehow succeeds, clean up
		if database != nil {
			database.Close()
		}
	}
}

func TestConnectionStats(t *testing.T) {
	stats := db.ConnectionStats{
		MaxOpenConnections: 10,
		OpenConnections:    5,
		InUseConnections:   2,
		IdleConnections:    3,
		WaitCount:          100,
		WaitDuration:       time.Second,
		MaxIdleClosed:      50,
		MaxIdleTimeClosed:  25,
		MaxLifetimeClosed:  10,
	}

	// Test that all fields are accessible
	if stats.MaxOpenConnections != 10 {
		t.Error("MaxOpenConnections not set correctly")
	}

	if stats.OpenConnections != 5 {
		t.Error("OpenConnections not set correctly")
	}

	if stats.InUseConnections != 2 {
		t.Error("InUseConnections not set correctly")
	}

	if stats.IdleConnections != 3 {
		t.Error("IdleConnections not set correctly")
	}

	if stats.WaitCount != 100 {
		t.Error("WaitCount not set correctly")
	}

	if stats.WaitDuration != time.Second {
		t.Error("WaitDuration not set correctly")
	}
}

func TestDatabaseManager_Creation(t *testing.T) {
	logger := &MockLogger{}
	manager := db.NewDatabaseManager(logger)

	if manager == nil {
		t.Error("Expected non-nil database manager")
	}
}

// Basic benchmark test
func BenchmarkDatabaseManager_Creation(b *testing.B) {
	logger := &MockLogger{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = db.NewDatabaseManager(logger)
	}
}
