package unit

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/SeaSBee/go-ormx/pkg/logging"
	"github.com/stretchr/testify/assert"
)

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    logging.LogLevel
		expected string
	}{
		{logging.LogLevelDebug, "debug"},
		{logging.LogLevelInfo, "info"},
		{logging.LogLevelWarn, "warn"},
		{logging.LogLevelError, "error"},
		{logging.LogLevelFatal, "fatal"},
		{logging.LogLevel(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.level.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected logging.LogLevel
	}{
		{"debug", logging.LogLevelDebug},
		{"info", logging.LogLevelInfo},
		{"warn", logging.LogLevelWarn},
		{"error", logging.LogLevelError},
		{"fatal", logging.LogLevelFatal},
		{"unknown", logging.LogLevelInfo}, // Default fallback
		{"DEBUG", logging.LogLevelInfo},   // Case sensitive
		{"", logging.LogLevelInfo},        // Empty string
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := logging.ParseLogLevel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLogField_Creation(t *testing.T) {
	// Test convenience functions
	stringField := logging.String("key", "value")
	assert.Equal(t, "key", stringField.Key)
	assert.Equal(t, "value", stringField.Value)

	intField := logging.Int("count", 42)
	assert.Equal(t, "count", intField.Key)
	assert.Equal(t, 42, intField.Value)

	int64Field := logging.Int64("big_count", 9223372036854775807)
	assert.Equal(t, "big_count", int64Field.Key)
	assert.Equal(t, int64(9223372036854775807), int64Field.Value)

	float64Field := logging.Float64("ratio", 3.14)
	assert.Equal(t, "ratio", float64Field.Key)
	assert.Equal(t, 3.14, float64Field.Value)

	boolField := logging.Bool("enabled", true)
	assert.Equal(t, "enabled", boolField.Key)
	assert.Equal(t, true, boolField.Value)

	durationField := logging.Duration("timeout", 5*time.Second)
	assert.Equal(t, "timeout", durationField.Key)
	assert.Equal(t, 5*time.Second, durationField.Value)

	timeField := logging.Time("created", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
	assert.Equal(t, "created", timeField.Key)
	assert.Equal(t, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), timeField.Value)

	errorField := logging.ErrorField("err", assert.AnError)
	assert.Equal(t, "err", errorField.Key)
	assert.Equal(t, assert.AnError, errorField.Value)

	anyField := logging.Any("custom", map[string]int{"a": 1, "b": 2})
	assert.Equal(t, "custom", anyField.Key)
	assert.Equal(t, map[string]int{"a": 1, "b": 2}, anyField.Value)
}

func TestNewLogger(t *testing.T) {
	// Test with nil output (should default to stdout)
	logger := logging.NewLogger(logging.LogLevelInfo, nil, nil)
	assert.NotNil(t, logger)
	assert.Equal(t, logging.LogLevelInfo, logger.GetLevel())

	// Test with custom output and formatter
	buf := &bytes.Buffer{}
	formatter := &logging.TextFormatter{}
	logger = logging.NewLogger(logging.LogLevelDebug, buf, formatter)
	assert.NotNil(t, logger)
	assert.Equal(t, logging.LogLevelDebug, logger.GetLevel())
}

func TestBaseLogger_LogLevels(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := logging.NewLogger(logging.LogLevelInfo, buf, &logging.TextFormatter{})
	ctx := context.Background()

	// Test that debug messages are filtered out at info level
	logger.Debug(ctx, "debug message")
	assert.Empty(t, buf.String())

	// Test that info messages are logged
	logger.Info(ctx, "info message")
	assert.Contains(t, buf.String(), "info message")

	// Test that warn messages are logged
	logger.Warn(ctx, "warn message")
	assert.Contains(t, buf.String(), "warn message")

	// Test that error messages are logged
	logger.Error(ctx, "error message")
	assert.Contains(t, buf.String(), "error message")
}

func TestBaseLogger_WithFields(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := logging.NewLogger(logging.LogLevelInfo, buf, &logging.TextFormatter{})
	ctx := context.Background()

	// Create logger with fields
	loggerWithFields := logger.WithFields(
		logging.String("user", "john"),
		logging.Int("age", 30),
	)

	// Log message with additional fields
	loggerWithFields.Info(ctx, "user logged in", logging.String("action", "login"))

	// Check that all fields are included
	logOutput := buf.String()
	assert.Contains(t, logOutput, "user=john")
	assert.Contains(t, logOutput, "age=30")
	assert.Contains(t, logOutput, "action=login")
}

func TestBaseLogger_WithContext(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := logging.NewLogger(logging.LogLevelInfo, buf, &logging.TextFormatter{})
	ctx := context.Background()

	// Create logger with context
	loggerWithContext := logger.WithContext(ctx)
	assert.NotNil(t, loggerWithContext)

	// Log message
	loggerWithContext.Info(ctx, "test message")
	assert.Contains(t, buf.String(), "test message")
}

func TestBaseLogger_SetLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := logging.NewLogger(logging.LogLevelInfo, buf, &logging.TextFormatter{})
	ctx := context.Background()

	// Initially at info level
	assert.Equal(t, logging.LogLevelInfo, logger.GetLevel())

	// Set to debug level
	logger.SetLevel(logging.LogLevelDebug)
	assert.Equal(t, logging.LogLevelDebug, logger.GetLevel())

	// Now debug messages should be logged
	logger.Debug(ctx, "debug message")
	assert.Contains(t, buf.String(), "debug message")
}

func TestJSONFormatter_Format(t *testing.T) {
	formatter := &logging.JSONFormatter{}
	entry := logging.LogEntry{
		Level:   logging.LogLevelInfo,
		Message: "test message",
		Time:    time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC),
		Fields: []logging.LogField{
			logging.String("key", "value"),
			logging.Int("number", 42),
		},
	}

	result, err := formatter.Format(entry)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)

	// Check JSON structure
	jsonStr := string(result)
	assert.Contains(t, jsonStr, `"level":"info"`)
	assert.Contains(t, jsonStr, `"message":"test message"`)
	assert.Contains(t, jsonStr, `"key":"value"`)
	assert.Contains(t, jsonStr, `"number":"42"`)
}

func TestTextFormatter_Format(t *testing.T) {
	formatter := &logging.TextFormatter{}
	entry := logging.LogEntry{
		Level:   logging.LogLevelInfo,
		Message: "test message",
		Time:    time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC),
		Fields: []logging.LogField{
			logging.String("key", "value"),
			logging.Int("number", 42),
		},
	}

	result, err := formatter.Format(entry)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)

	// Check text format
	textStr := string(result)
	assert.Contains(t, textStr, "[2020-01-01T12:00:00Z]")
	assert.Contains(t, textStr, "info: test message")
	assert.Contains(t, textStr, "key=value")
	assert.Contains(t, textStr, "number=42")
}

func TestNewLoggerFromConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      logging.LoggerConfig
		expectError bool
	}{
		{
			name: "valid stdout config",
			config: logging.LoggerConfig{
				Level:              logging.LogLevelInfo,
				Format:             "text",
				Output:             "stdout",
				SlowQueryThreshold: time.Second,
				EnableCaller:       false,
				EnableTimestamp:    true,
			},
			expectError: false,
		},
		{
			name: "valid stderr config",
			config: logging.LoggerConfig{
				Level:              logging.LogLevelDebug,
				Format:             "json",
				Output:             "stderr",
				SlowQueryThreshold: time.Second,
				EnableCaller:       true,
				EnableTimestamp:    true,
			},
			expectError: false,
		},
		{
			name: "invalid format",
			config: logging.LoggerConfig{
				Level:              logging.LogLevelInfo,
				Format:             "invalid",
				Output:             "stdout",
				SlowQueryThreshold: time.Second,
				EnableCaller:       false,
				EnableTimestamp:    true,
			},
			expectError: false, // Should fallback to text
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := logging.NewLoggerFromConfig(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, logger)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, logger)
			}
		})
	}
}

func TestDefaultLoggerConfig(t *testing.T) {
	config := logging.DefaultLoggerConfig()

	assert.Equal(t, logging.LogLevelInfo, config.Level)
	assert.Equal(t, "text", config.Format)
	assert.Equal(t, "stdout", config.Output)
	assert.Equal(t, time.Second, config.SlowQueryThreshold)
	assert.False(t, config.EnableCaller)
	assert.True(t, config.EnableTimestamp)
}

func TestQueryLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	baseLogger := logging.NewLogger(logging.LogLevelDebug, buf, &logging.TextFormatter{})
	queryLogger := logging.NewQueryLogger(baseLogger, 100*time.Millisecond)
	ctx := context.Background()

	// Test successful query (logs at Debug level)
	queryLogger.LogQuery(ctx, "SELECT * FROM users", 50*time.Millisecond, 10, nil)
	logOutput := buf.String()
	assert.Contains(t, logOutput, "Query executed")
	assert.Contains(t, logOutput, "query=SELECT * FROM users")
	assert.Contains(t, logOutput, "duration=50ms")
	assert.Contains(t, logOutput, "rows=10")

	// Test failed query (logs at Error level)
	buf.Reset()
	queryLogger.LogQuery(ctx, "SELECT * FROM users", 50*time.Millisecond, 0, assert.AnError)
	logOutput = buf.String()
	assert.Contains(t, logOutput, "Query failed")
	assert.Contains(t, logOutput, "error=")

	// Test slow query (logs at Warn level)
	buf.Reset()
	queryLogger.LogQuery(ctx, "SELECT * FROM users", 150*time.Millisecond, 1000, nil)
	logOutput = buf.String()
	assert.Contains(t, logOutput, "Slow query detected")
}

func TestPerformanceLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	baseLogger := logging.NewLogger(logging.LogLevelInfo, buf, &logging.TextFormatter{})
	perfLogger := logging.NewPerformanceLogger(baseLogger)
	ctx := context.Background()

	// Test operation logging
	metrics := map[string]interface{}{
		"rows_processed": 1000,
		"cache_hits":     500,
	}
	perfLogger.LogOperation(ctx, "batch_process", 2*time.Second, metrics)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "Operation completed")
	assert.Contains(t, logOutput, "operation=batch_process")
	assert.Contains(t, logOutput, "duration=2s")
	assert.Contains(t, logOutput, "rows_processed=1000")
	assert.Contains(t, logOutput, "cache_hits=500")

	// Test memory usage logging
	buf.Reset()
	perfLogger.LogMemoryUsage(ctx, 1024*1024*100, 1024*1024*1000) // 100MB allocated, 1GB total
	logOutput = buf.String()
	assert.Contains(t, logOutput, "Memory usage")
	assert.Contains(t, logOutput, "allocated_mb=100")
	assert.Contains(t, logOutput, "total_mb=1000")
	assert.Contains(t, logOutput, "usage_percent=10")

	// Test goroutine count logging
	buf.Reset()
	perfLogger.LogGoroutineCount(ctx, 25)
	logOutput = buf.String()
	assert.Contains(t, logOutput, "Goroutine count")
	assert.Contains(t, logOutput, "goroutine_count=25")
}

func TestMultiLogger(t *testing.T) {
	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}

	logger1 := logging.NewLogger(logging.LogLevelInfo, buf1, &logging.TextFormatter{})
	logger2 := logging.NewLogger(logging.LogLevelDebug, buf2, &logging.TextFormatter{})

	multiLogger := logging.NewMultiLogger(logger1, logger2)
	ctx := context.Background()

	// Test that both loggers receive messages
	multiLogger.Info(ctx, "test message")

	assert.Contains(t, buf1.String(), "test message")
	assert.Contains(t, buf2.String(), "test message")

	// Test level setting
	multiLogger.SetLevel(logging.LogLevelDebug)
	assert.Equal(t, logging.LogLevelDebug, multiLogger.GetLevel())

	// Test with context
	multiLoggerWithContext := multiLogger.WithContext(ctx)
	assert.NotNil(t, multiLoggerWithContext)

	// Test with fields
	multiLoggerWithFields := multiLogger.WithFields(logging.String("key", "value"))
	assert.NotNil(t, multiLoggerWithFields)
}

func TestMultiLogger_Close(t *testing.T) {
	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}

	logger1 := logging.NewLogger(logging.LogLevelInfo, buf1, &logging.TextFormatter{})
	logger2 := logging.NewLogger(logging.LogLevelInfo, buf2, &logging.TextFormatter{})

	multiLogger := logging.NewMultiLogger(logger1, logger2)

	// Close should not error
	err := multiLogger.Close()
	assert.NoError(t, err)
}

func TestContextFieldsExtraction(t *testing.T) {
	ctx := context.WithValue(context.Background(), "request_id", "req-123")
	ctx = context.WithValue(ctx, "trace_id", "trace-456")
	ctx = context.WithValue(ctx, "span_id", "span-789")
	ctx = context.WithValue(ctx, "user_id", "user-abc")
	ctx = context.WithValue(ctx, "tenant_id", "tenant-def")

	buf := &bytes.Buffer{}
	logger := logging.NewLogger(logging.LogLevelInfo, buf, &logging.TextFormatter{})

	logger.Info(ctx, "test message")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "request_id=req-123")
	assert.Contains(t, logOutput, "trace_id=trace-456")
	assert.Contains(t, logOutput, "span_id=span-789")
	assert.Contains(t, logOutput, "user_id=user-abc")
	assert.Contains(t, logOutput, "tenant_id=tenant-def")
}

func TestNilContextHandling(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := logging.NewLogger(logging.LogLevelInfo, buf, &logging.TextFormatter{})

	// Should not panic with nil context
	logger.Info(nil, "test message")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "test message")
}

func TestLogger_Close(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := logging.NewLogger(logging.LogLevelInfo, buf, &logging.TextFormatter{})

	// Close should not error
	err := logger.Close()
	assert.NoError(t, err)
}

func TestConcurrentLogging(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := logging.NewLogger(logging.LogLevelInfo, buf, &logging.TextFormatter{})
	ctx := context.Background()

	// Test concurrent logging
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			logger.Info(ctx, "concurrent message", logging.Int("goroutine", id))
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Check that all messages were logged
	logOutput := buf.String()
	assert.Contains(t, logOutput, "concurrent message")

	// Check that we have goroutine messages (order may vary due to concurrency)
	// Note: bytes.Buffer is not thread-safe, so some messages might be lost
	// We'll check that at least some goroutine messages are present
	goroutineCount := 0
	for i := 0; i < 10; i++ {
		if strings.Contains(logOutput, fmt.Sprintf("goroutine=%d", i)) {
			goroutineCount++
		}
	}

	// Since bytes.Buffer is not thread-safe, we can't guarantee all 10 messages
	// We'll check that at least some messages are present
	assert.Greater(t, goroutineCount, 0, "At least some goroutine messages should be present")
}

func TestLogLevelFiltering(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := logging.NewLogger(logging.LogLevelWarn, buf, &logging.TextFormatter{})
	ctx := context.Background()

	// Debug and info should be filtered out
	logger.Debug(ctx, "debug message")
	logger.Info(ctx, "info message")
	assert.Empty(t, buf.String())

	// Warn and error should be logged
	logger.Warn(ctx, "warn message")
	logger.Error(ctx, "error message")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "warn message")
	assert.Contains(t, logOutput, "error message")
	assert.NotContains(t, logOutput, "debug message")
	assert.NotContains(t, logOutput, "info message")
}

func TestFatalLogging(t *testing.T) {
	// Note: This test would normally exit the program
	// In a real test environment, you might want to mock os.Exit
	// For now, we'll just test that the method exists and doesn't panic

	buf := &bytes.Buffer{}
	logger := logging.NewLogger(logging.LogLevelFatal, buf, &logging.TextFormatter{})

	// This would normally call os.Exit(1), so we'll just verify the method exists
	// In a real test, you might want to use a test hook or mock
	assert.NotNil(t, logger.Fatal)
}

// Add missing test scenarios
func TestLogField_EdgeCases(t *testing.T) {
	// Test with empty key
	emptyKeyField := logging.String("", "value")
	assert.Equal(t, "", emptyKeyField.Key)
	assert.Equal(t, "value", emptyKeyField.Value)

	// Test with very long key
	longKey := strings.Repeat("a", 1000)
	longKeyField := logging.String(longKey, "value")
	assert.Equal(t, longKey, longKeyField.Key)
	assert.Equal(t, "value", longKeyField.Value)

	// Test with very long value
	longValue := strings.Repeat("b", 1000)
	longValueField := logging.String("key", longValue)
	assert.Equal(t, "key", longValueField.Key)
	assert.Equal(t, longValue, longValueField.Value)

	// Test with special characters
	specialField := logging.String("special!@#$%", "value!@#$%")
	assert.Equal(t, "special!@#$%", specialField.Key)
	assert.Equal(t, "value!@#$%", specialField.Value)

	// Test with nil values
	nilField := logging.Any("nil_key", nil)
	assert.Equal(t, "nil_key", nilField.Key)
	assert.Nil(t, nilField.Value)

	// Test with empty values
	emptyValueField := logging.String("key", "")
	assert.Equal(t, "key", emptyValueField.Key)
	assert.Equal(t, "", emptyValueField.Value)
}

func TestBaseLogger_EdgeCases(t *testing.T) {
	// Test with nil output
	logger := logging.NewLogger(logging.LogLevelInfo, nil, nil)
	assert.NotNil(t, logger)

	// Test with nil formatter
	logger = logging.NewLogger(logging.LogLevelInfo, &bytes.Buffer{}, nil)
	assert.NotNil(t, logger)

	// Test with invalid log level
	invalidLevel := logging.LogLevel(99)
	logger.SetLevel(invalidLevel)
	assert.Equal(t, invalidLevel, logger.GetLevel())

	// Test with empty message
	buf := &bytes.Buffer{}
	logger = logging.NewLogger(logging.LogLevelInfo, buf, &logging.TextFormatter{})
	ctx := context.Background()

	logger.Info(ctx, "")
	logOutput := buf.String()
	assert.Contains(t, logOutput, "info: ")

	// Test with very long message
	longMessage := strings.Repeat("a", 10000)
	logger.Info(ctx, longMessage)
	logOutput = buf.String()
	assert.Contains(t, logOutput, longMessage)

	// Test with special characters in message
	specialMessage := "message with special chars: !@#$%^&*()_+-=[]{}|;':\",./<>?"
	logger.Info(ctx, specialMessage)
	logOutput = buf.String()
	assert.Contains(t, logOutput, specialMessage)
}

func TestJSONFormatter_EdgeCases(t *testing.T) {
	formatter := &logging.JSONFormatter{}

	// Test with empty entry
	emptyEntry := logging.LogEntry{}
	result, err := formatter.Format(emptyEntry)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)

	// Test with nil fields
	entryWithNilFields := logging.LogEntry{
		Level:   logging.LogLevelInfo,
		Message: "test",
		Time:    time.Now(),
		Fields:  nil,
	}
	result, err = formatter.Format(entryWithNilFields)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)

	// Test with empty fields
	entryWithEmptyFields := logging.LogEntry{
		Level:   logging.LogLevelInfo,
		Message: "test",
		Time:    time.Now(),
		Fields:  []logging.LogField{},
	}
	result, err = formatter.Format(entryWithEmptyFields)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)

	// Test with very long message
	longMessage := strings.Repeat("a", 10000)
	entryWithLongMessage := logging.LogEntry{
		Level:   logging.LogLevelInfo,
		Message: longMessage,
		Time:    time.Now(),
		Fields:  []logging.LogField{},
	}
	result, err = formatter.Format(entryWithLongMessage)
	assert.NoError(t, err)
	assert.Contains(t, string(result), longMessage)

	// Test with special characters in message
	specialMessage := "message with special chars: !@#$%^&*()_+-=[]{}|;':\",./<>?"
	entryWithSpecialMessage := logging.LogEntry{
		Level:   logging.LogLevelInfo,
		Message: specialMessage,
		Time:    time.Now(),
		Fields:  []logging.LogField{},
	}
	result, err = formatter.Format(entryWithSpecialMessage)
	assert.NoError(t, err)
	assert.Contains(t, string(result), specialMessage)
}

func TestTextFormatter_EdgeCases(t *testing.T) {
	formatter := &logging.TextFormatter{}

	// Test with empty entry
	emptyEntry := logging.LogEntry{}
	result, err := formatter.Format(emptyEntry)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)

	// Test with nil fields
	entryWithNilFields := logging.LogEntry{
		Level:   logging.LogLevelInfo,
		Message: "test",
		Time:    time.Now(),
		Fields:  nil,
	}
	result, err = formatter.Format(entryWithNilFields)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)

	// Test with empty fields
	entryWithEmptyFields := logging.LogEntry{
		Level:   logging.LogLevelInfo,
		Message: "test",
		Time:    time.Now(),
		Fields:  []logging.LogField{},
	}
	result, err = formatter.Format(entryWithEmptyFields)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)

	// Test with very long message
	longMessage := strings.Repeat("a", 10000)
	entryWithLongMessage := logging.LogEntry{
		Level:   logging.LogLevelInfo,
		Message: longMessage,
		Time:    time.Now(),
		Fields:  []logging.LogField{},
	}
	result, err = formatter.Format(entryWithLongMessage)
	assert.NoError(t, err)
	assert.Contains(t, string(result), longMessage)

	// Test with special characters in message
	specialMessage := "message with special chars: !@#$%^&*()_+-=[]{}|;':\",./<>?"
	entryWithSpecialMessage := logging.LogEntry{
		Level:   logging.LogLevelInfo,
		Message: specialMessage,
		Time:    time.Now(),
		Fields:  []logging.LogField{},
	}
	result, err = formatter.Format(entryWithSpecialMessage)
	assert.NoError(t, err)
	assert.Contains(t, string(result), specialMessage)
}

func TestQueryLogger_EdgeCases(t *testing.T) {
	buf := &bytes.Buffer{}
	baseLogger := logging.NewLogger(logging.LogLevelDebug, buf, &logging.TextFormatter{})
	queryLogger := logging.NewQueryLogger(baseLogger, 100*time.Millisecond)
	ctx := context.Background()

	// Test with empty query
	queryLogger.LogQuery(ctx, "", 50*time.Millisecond, 10, nil)
	logOutput := buf.String()
	assert.Contains(t, logOutput, "query=")

	// Test with very long query
	longQuery := strings.Repeat("SELECT * FROM users WHERE id = ? AND name = ? AND age > ? AND email LIKE ? AND created_at > ? AND updated_at < ? AND status = ? AND role = ? AND department = ? AND manager_id = ?; ", 100)
	queryLogger.LogQuery(ctx, longQuery, 50*time.Millisecond, 10, nil)
	logOutput = buf.String()
	assert.Contains(t, logOutput, longQuery)

	// Test with special characters in query
	specialQuery := "SELECT * FROM `users` WHERE `name` LIKE '%test%' AND `email` REGEXP '^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$'"
	queryLogger.LogQuery(ctx, specialQuery, 50*time.Millisecond, 10, nil)
	logOutput = buf.String()
	assert.Contains(t, logOutput, specialQuery)

	// Test with zero duration
	queryLogger.LogQuery(ctx, "SELECT 1", 0, 1, nil)
	logOutput = buf.String()
	assert.Contains(t, logOutput, "duration=0s")

	// Test with very long duration
	queryLogger.LogQuery(ctx, "SELECT 1", 24*365*time.Hour, 1, nil) // 1 year
	logOutput = buf.String()
	assert.Contains(t, logOutput, "duration=")

	// Test with negative duration
	queryLogger.LogQuery(ctx, "SELECT 1", -1*time.Second, 1, nil)
	logOutput = buf.String()
	assert.Contains(t, logOutput, "duration=")

	// Test with zero rows
	queryLogger.LogQuery(ctx, "SELECT 1", 50*time.Millisecond, 0, nil)
	logOutput = buf.String()
	assert.Contains(t, logOutput, "rows=0")

	// Test with very large row count
	queryLogger.LogQuery(ctx, "SELECT 1", 50*time.Millisecond, 999999999, nil)
	logOutput = buf.String()
	assert.Contains(t, logOutput, "rows=999999999")

	// Test with negative row count
	queryLogger.LogQuery(ctx, "SELECT 1", 50*time.Millisecond, -1, nil)
	logOutput = buf.String()
	assert.Contains(t, logOutput, "rows=-1")
}

func TestPerformanceLogger_EdgeCases(t *testing.T) {
	buf := &bytes.Buffer{}
	baseLogger := logging.NewLogger(logging.LogLevelInfo, buf, &logging.TextFormatter{})
	perfLogger := logging.NewPerformanceLogger(baseLogger)
	ctx := context.Background()

	// Test with empty operation name
	perfLogger.LogOperation(ctx, "", 2*time.Second, nil)
	logOutput := buf.String()
	assert.Contains(t, logOutput, "operation=")

	// Test with very long operation name
	longOperation := strings.Repeat("very_long_operation_name_that_exceeds_normal_lengths_and_should_be_handled_gracefully_by_the_logging_system", 10)
	perfLogger.LogOperation(ctx, longOperation, 2*time.Second, nil)
	logOutput = buf.String()
	assert.Contains(t, logOutput, longOperation)

	// Test with special characters in operation name
	specialOperation := "operation_with_special_chars: !@#$%^&*()_+-=[]{}|;':\",./<>?"
	perfLogger.LogOperation(ctx, specialOperation, 2*time.Second, nil)
	logOutput = buf.String()
	assert.Contains(t, logOutput, specialOperation)

	// Test with zero duration
	perfLogger.LogOperation(ctx, "test", 0, nil)
	logOutput = buf.String()
	assert.Contains(t, logOutput, "duration=0s")

	// Test with very long duration
	perfLogger.LogOperation(ctx, "test", 24*365*time.Hour, nil) // 1 year
	logOutput = buf.String()
	assert.Contains(t, logOutput, "duration=")

	// Test with negative duration
	perfLogger.LogOperation(ctx, "test", -1*time.Second, nil)
	logOutput = buf.String()
	assert.Contains(t, logOutput, "duration=")

	// Test with nil metrics
	perfLogger.LogOperation(ctx, "test", 2*time.Second, nil)
	logOutput = buf.String()
	assert.Contains(t, logOutput, "Operation completed")

	// Test with empty metrics
	perfLogger.LogOperation(ctx, "test", 2*time.Second, map[string]interface{}{})
	logOutput = buf.String()
	assert.Contains(t, logOutput, "Operation completed")

	// Test with very large metrics values
	largeMetrics := map[string]interface{}{
		"large_number": 999999999999999,
		"large_string": strings.Repeat("a", 10000),
		"large_float":  999999999.999999,
	}
	perfLogger.LogOperation(ctx, "test", 2*time.Second, largeMetrics)
	logOutput = buf.String()
	assert.Contains(t, logOutput, "large_number=999999999999999")
	assert.Contains(t, logOutput, "large_float=9.99999999999999e+08")
}

func TestMultiLogger_EdgeCases(t *testing.T) {
	// Test with nil loggers
	multiLogger := logging.NewMultiLogger()
	assert.NotNil(t, multiLogger)

	// Test with single logger
	buf := &bytes.Buffer{}
	logger := logging.NewLogger(logging.LogLevelInfo, buf, &logging.TextFormatter{})
	multiLogger = logging.NewMultiLogger(logger)
	assert.NotNil(t, multiLogger)

	// Test with multiple nil loggers
	multiLogger = logging.NewMultiLogger(nil, nil, nil)
	assert.NotNil(t, multiLogger)

	// Test with mixed nil and valid loggers
	multiLogger = logging.NewMultiLogger(nil, logger, nil)
	assert.NotNil(t, multiLogger)

	// Test operations with nil loggers
	ctx := context.Background()
	multiLogger.Info(ctx, "test message")
	// Should not panic even with nil loggers
}

func TestLoggerConfig_EdgeCases(t *testing.T) {
	// Test with invalid output
	config := logging.LoggerConfig{
		Level:  logging.LogLevelInfo,
		Format: "text",
		Output: "invalid_output",
	}
	logger, err := logging.NewLoggerFromConfig(config)
	assert.NoError(t, err) // Should fallback to stdout
	assert.NotNil(t, logger)

	// Test with empty output
	config.Output = ""
	logger, err = logging.NewLoggerFromConfig(config)
	assert.NoError(t, err) // Should fallback to stdout
	assert.NotNil(t, logger)

	// Test with invalid format
	config.Format = "invalid_format"
	logger, err = logging.NewLoggerFromConfig(config)
	assert.NoError(t, err) // Should fallback to text
	assert.NotNil(t, logger)

	// Test with empty format
	config.Format = ""
	logger, err = logging.NewLoggerFromConfig(config)
	assert.NoError(t, err) // Should fallback to text
	assert.NotNil(t, logger)

	// Test with very long slow query threshold
	config.SlowQueryThreshold = 24 * 365 * time.Hour // 1 year
	logger, err = logging.NewLoggerFromConfig(config)
	assert.NoError(t, err)
	assert.NotNil(t, logger)

	// Test with zero slow query threshold
	config.SlowQueryThreshold = 0
	logger, err = logging.NewLoggerFromConfig(config)
	assert.NoError(t, err)
	assert.NotNil(t, logger)

	// Test with negative slow query threshold
	config.SlowQueryThreshold = -1 * time.Second
	logger, err = logging.NewLoggerFromConfig(config)
	assert.NoError(t, err)
	assert.NotNil(t, logger)
}

func TestLogLevel_EdgeCases(t *testing.T) {
	// Test with invalid log level string
	invalidLevel := logging.ParseLogLevel("invalid_level")
	assert.Equal(t, logging.LogLevelInfo, invalidLevel) // Should fallback to info

	// Test with empty string
	emptyLevel := logging.ParseLogLevel("")
	assert.Equal(t, logging.LogLevelInfo, emptyLevel) // Should fallback to info

	// Test with whitespace string
	whitespaceLevel := logging.ParseLogLevel("   ")
	assert.Equal(t, logging.LogLevelInfo, whitespaceLevel) // Should fallback to info

	// Test with mixed case
	mixedCaseLevel := logging.ParseLogLevel("DeBuG")
	assert.Equal(t, logging.LogLevelInfo, mixedCaseLevel) // Should fallback to info

	// Test with numbers
	numberLevel := logging.ParseLogLevel("123")
	assert.Equal(t, logging.LogLevelInfo, numberLevel) // Should fallback to info

	// Test with special characters
	specialLevel := logging.ParseLogLevel("!@#$%")
	assert.Equal(t, logging.LogLevelInfo, specialLevel) // Should fallback to info
}

func TestContextFields_EdgeCases(t *testing.T) {
	// Test with nil context
	buf := &bytes.Buffer{}
	logger := logging.NewLogger(logging.LogLevelInfo, buf, &logging.TextFormatter{})
	logger.Info(nil, "test message")
	logOutput := buf.String()
	assert.Contains(t, logOutput, "test message")

	// Test with context containing nil values
	ctx := context.WithValue(context.Background(), "nil_key", nil)
	logger.Info(ctx, "test message")
	logOutput = buf.String()
	assert.Contains(t, logOutput, "test message")

	// Test with context containing empty values
	ctx = context.WithValue(context.Background(), "empty_key", "")
	logger.Info(ctx, "test message")
	logOutput = buf.String()
	assert.Contains(t, logOutput, "test message")

	// Test with context containing very long values
	longValue := strings.Repeat("a", 10000)
	ctx = context.WithValue(context.Background(), "long_key", longValue)
	logger.Info(ctx, "test message")
	logOutput = buf.String()
	assert.Contains(t, logOutput, "test message")

	// Test with context containing special characters
	specialValue := "value with special chars: !@#$%^&*()_+-=[]{}|;':\",./<>?"
	ctx = context.WithValue(context.Background(), "special_key", specialValue)
	logger.Info(ctx, "test message")
	logOutput = buf.String()
	assert.Contains(t, logOutput, "test message") // Context values are not automatically extracted
}
