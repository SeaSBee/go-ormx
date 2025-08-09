package logging_test

import (
	"reflect"
	"testing"
	"time"

	"go-ormx/ormx/logging"
)

func TestLogField_Constructors(t *testing.T) {
	tests := []struct {
		name     string
		field    logging.LogField
		wantKey  string
		wantType reflect.Type
	}{
		{
			name:     "string_field",
			field:    logging.String("key", "value"),
			wantKey:  "key",
			wantType: reflect.TypeOf(""),
		},
		{
			name:     "int_field",
			field:    logging.Int("key", 42),
			wantKey:  "key",
			wantType: reflect.TypeOf(0),
		},
		{
			name:     "int64_field",
			field:    logging.Int64("key", int64(42)),
			wantKey:  "key",
			wantType: reflect.TypeOf(int64(0)),
		},
		{
			name:     "float64_field",
			field:    logging.Float64("key", 3.14),
			wantKey:  "key",
			wantType: reflect.TypeOf(0.0),
		},
		{
			name:     "bool_field",
			field:    logging.Bool("key", true),
			wantKey:  "key",
			wantType: reflect.TypeOf(false),
		},
		{
			name:     "duration_field",
			field:    logging.Duration("key", time.Second),
			wantKey:  "key",
			wantType: reflect.TypeOf(time.Duration(0)),
		},
		{
			name:     "any_field",
			field:    logging.Any("key", map[string]int{"test": 1}),
			wantKey:  "key",
			wantType: reflect.TypeOf(map[string]int{}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.field.Key != tt.wantKey {
				t.Errorf("Expected key %s, got %s", tt.wantKey, tt.field.Key)
			}

			valueType := reflect.TypeOf(tt.field.Value)
			if valueType != tt.wantType {
				t.Errorf("Expected type %v, got %v", tt.wantType, valueType)
			}
		})
	}
}

func TestLogLevel_Constants(t *testing.T) {
	// Test that log levels are defined and distinct
	levels := []logging.LogLevel{
		logging.Silent,
		logging.Error,
		logging.Warn,
		logging.Info,
	}

	for i, level := range levels {
		if int(level) != i+1 {
			t.Errorf("Expected level %d to have value %d, got %d", i, i+1, int(level))
		}
	}
}

func TestDBLogger_Creation(t *testing.T) {
	// Mock logger for testing
	mockLogger := &MockLogger{}

	config := logging.LoggerConfig{
		LogLevel:             logging.Info,
		IgnoreRecordNotFound: true,
		SlowThreshold:        time.Second,
		Colorful:             true,
		SourceField:          "source",
	}

	dbLogger := logging.NewDBLogger(mockLogger, config)
	if dbLogger == nil {
		t.Error("Expected non-nil DBLogger")
	}
}

// Mock logger for testing
type MockLogger struct {
	logs []LogEntry
}

type LogEntry struct {
	Level   string
	Message string
	Fields  []logging.LogField
}

func (m *MockLogger) Trace(msg string, fields ...logging.LogField) {
	m.logs = append(m.logs, LogEntry{Level: "TRACE", Message: msg, Fields: fields})
}

func (m *MockLogger) Debug(msg string, fields ...logging.LogField) {
	m.logs = append(m.logs, LogEntry{Level: "DEBUG", Message: msg, Fields: fields})
}

func (m *MockLogger) Info(msg string, fields ...logging.LogField) {
	m.logs = append(m.logs, LogEntry{Level: "INFO", Message: msg, Fields: fields})
}

func (m *MockLogger) Warn(msg string, fields ...logging.LogField) {
	m.logs = append(m.logs, LogEntry{Level: "WARN", Message: msg, Fields: fields})
}

func (m *MockLogger) Error(msg string, fields ...logging.LogField) {
	m.logs = append(m.logs, LogEntry{Level: "ERROR", Message: msg, Fields: fields})
}

func (m *MockLogger) Fatal(msg string, fields ...logging.LogField) {
	m.logs = append(m.logs, LogEntry{Level: "FATAL", Message: msg, Fields: fields})
}

func (m *MockLogger) With(fields ...logging.LogField) logging.Logger {
	return m
}

func (m *MockLogger) GetLogs() []LogEntry {
	return m.logs
}

func (m *MockLogger) Clear() {
	m.logs = nil
}

// Benchmark tests
func BenchmarkLogField_Creation(b *testing.B) {
	b.Run("string_field", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = logging.String("key", "value")
		}
	})

	b.Run("int_field", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = logging.Int("key", 42)
		}
	})

	b.Run("any_field", func(b *testing.B) {
		value := map[string]interface{}{"test": "value"}
		for i := 0; i < b.N; i++ {
			_ = logging.Any("key", value)
		}
	})
}
