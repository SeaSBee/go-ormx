package logging

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// LogLevel represents the logging level
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "debug"
	case LogLevelInfo:
		return "info"
	case LogLevelWarn:
		return "warn"
	case LogLevelError:
		return "error"
	case LogLevelFatal:
		return "fatal"
	default:
		return "unknown"
	}
}

// ParseLogLevel parses a string to LogLevel
func ParseLogLevel(level string) LogLevel {
	switch level {
	case "debug":
		return LogLevelDebug
	case "info":
		return LogLevelInfo
	case "warn":
		return LogLevelWarn
	case "error":
		return LogLevelError
	case "fatal":
		return LogLevelFatal
	default:
		return LogLevelInfo
	}
}

// LogField represents a structured log field
type LogField struct {
	Key   string
	Value interface{}
}

// LogEntry represents a log entry
type LogEntry struct {
	Level     LogLevel
	Message   string
	Time      time.Time
	Fields    []LogField
	Context   context.Context
	Caller    string
	Timestamp string
}

// Logger interface defines the logging contract
type Logger interface {
	Debug(ctx context.Context, message string, fields ...LogField)
	Info(ctx context.Context, message string, fields ...LogField)
	Warn(ctx context.Context, message string, fields ...LogField)
	Error(ctx context.Context, message string, fields ...LogField)
	Fatal(ctx context.Context, message string, fields ...LogField)
	WithContext(ctx context.Context) Logger
	WithFields(fields ...LogField) Logger
	SetLevel(level LogLevel)
	GetLevel() LogLevel
	Close() error
}

// BaseLogger implements the base logging functionality
type BaseLogger struct {
	level     LogLevel
	output    io.Writer
	formatter LogFormatter
	mutex     sync.RWMutex
	fields    []LogField
}

// LogFormatter interface for formatting log entries
type LogFormatter interface {
	Format(entry LogEntry) ([]byte, error)
}

// JSONFormatter formats log entries as JSON
type JSONFormatter struct{}

// Format formats a log entry as JSON
func (f *JSONFormatter) Format(entry LogEntry) ([]byte, error) {
	// Simplified JSON formatting - in a real implementation, you'd use encoding/json
	json := fmt.Sprintf(`{"level":"%s","message":"%s","time":"%s"`,
		entry.Level.String(), entry.Message, entry.Time.Format(time.RFC3339))

	if len(entry.Fields) > 0 {
		json += `,"fields":{`
		for i, field := range entry.Fields {
			if i > 0 {
				json += ","
			}
			json += fmt.Sprintf(`"%s":"%v"`, field.Key, field.Value)
		}
		json += "}"
	}

	json += "}\n"
	return []byte(json), nil
}

// TextFormatter formats log entries as text
type TextFormatter struct{}

// Format formats a log entry as text
func (f *TextFormatter) Format(entry LogEntry) ([]byte, error) {
	text := fmt.Sprintf("[%s] %s: %s",
		entry.Time.Format("2006-01-02T15:04:05Z07:00"),
		entry.Level.String(),
		entry.Message)

	if len(entry.Fields) > 0 {
		text += " | "
		for i, field := range entry.Fields {
			if i > 0 {
				text += ", "
			}
			text += fmt.Sprintf("%s=%v", field.Key, field.Value)
		}
	}

	text += "\n"
	return []byte(text), nil
}

// NewLogger creates a new logger
func NewLogger(level LogLevel, output io.Writer, formatter LogFormatter) *BaseLogger {
	if output == nil {
		output = os.Stdout
	}
	if formatter == nil {
		formatter = &TextFormatter{}
	}

	return &BaseLogger{
		level:     level,
		output:    output,
		formatter: formatter,
		fields:    make([]LogField, 0),
	}
}

// log logs a message at the specified level
func (l *BaseLogger) log(ctx context.Context, level LogLevel, message string, fields ...LogField) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	if level < l.level {
		return
	}

	// Extract context fields
	contextFields := extractContextFields(ctx)

	// Combine all fields
	allFields := make([]LogField, 0, len(l.fields)+len(contextFields)+len(fields))
	allFields = append(allFields, l.fields...)
	allFields = append(allFields, contextFields...)
	allFields = append(allFields, fields...)

	entry := LogEntry{
		Level:   level,
		Message: message,
		Time:    time.Now(),
		Fields:  allFields,
		Context: ctx,
	}

	formatted, err := l.formatter.Format(entry)
	if err != nil {
		// Fallback to simple logging
		fmt.Fprintf(l.output, "[ERROR] Failed to format log entry: %v\n", err)
		return
	}

	_, err = l.output.Write(formatted)
	if err != nil {
		// Log to stderr if we can't write to output
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to write log entry: %v\n", err)
	}
}

// Debug logs a debug message
func (l *BaseLogger) Debug(ctx context.Context, message string, fields ...LogField) {
	l.log(ctx, LogLevelDebug, message, fields...)
}

// Info logs an info message
func (l *BaseLogger) Info(ctx context.Context, message string, fields ...LogField) {
	l.log(ctx, LogLevelInfo, message, fields...)
}

// Warn logs a warning message
func (l *BaseLogger) Warn(ctx context.Context, message string, fields ...LogField) {
	l.log(ctx, LogLevelWarn, message, fields...)
}

// Error logs an error message
func (l *BaseLogger) Error(ctx context.Context, message string, fields ...LogField) {
	l.log(ctx, LogLevelError, message, fields...)
}

// Fatal logs a fatal message and exits
func (l *BaseLogger) Fatal(ctx context.Context, message string, fields ...LogField) {
	l.log(ctx, LogLevelFatal, message, fields...)
	os.Exit(1)
}

// WithContext creates a new logger with context
func (l *BaseLogger) WithContext(ctx context.Context) Logger {
	return &BaseLogger{
		level:     l.level,
		output:    l.output,
		formatter: l.formatter,
		fields:    l.fields,
	}
}

// WithFields creates a new logger with additional fields
func (l *BaseLogger) WithFields(fields ...LogField) Logger {
	newFields := make([]LogField, len(l.fields)+len(fields))
	copy(newFields, l.fields)
	copy(newFields[len(l.fields):], fields)

	return &BaseLogger{
		level:     l.level,
		output:    l.output,
		formatter: l.formatter,
		fields:    newFields,
	}
}

// SetLevel sets the log level
func (l *BaseLogger) SetLevel(level LogLevel) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.level = level
}

// GetLevel gets the current log level
func (l *BaseLogger) GetLevel() LogLevel {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.level
}

// Close closes the logger
func (l *BaseLogger) Close() error {
	return nil
}

// extractContextFields extracts fields from context
func extractContextFields(ctx context.Context) []LogField {
	if ctx == nil {
		return nil
	}

	var fields []LogField

	// Extract request ID
	if requestID := ctx.Value("request_id"); requestID != nil {
		fields = append(fields, LogField{Key: "request_id", Value: requestID})
	}

	// Extract trace ID
	if traceID := ctx.Value("trace_id"); traceID != nil {
		fields = append(fields, LogField{Key: "trace_id", Value: traceID})
	}

	// Extract span ID
	if spanID := ctx.Value("span_id"); spanID != nil {
		fields = append(fields, LogField{Key: "span_id", Value: spanID})
	}

	// Extract user ID
	if userID := ctx.Value("user_id"); userID != nil {
		fields = append(fields, LogField{Key: "user_id", Value: userID})
	}

	// Extract tenant ID
	if tenantID := ctx.Value("tenant_id"); tenantID != nil {
		fields = append(fields, LogField{Key: "tenant_id", Value: tenantID})
	}

	return fields
}

// Convenience functions for creating log fields
func String(key, value string) LogField {
	return LogField{Key: key, Value: value}
}

func Int(key string, value int) LogField {
	return LogField{Key: key, Value: value}
}

func Int64(key string, value int64) LogField {
	return LogField{Key: key, Value: value}
}

func Float64(key string, value float64) LogField {
	return LogField{Key: key, Value: value}
}

func Bool(key string, value bool) LogField {
	return LogField{Key: key, Value: value}
}

func Duration(key string, value time.Duration) LogField {
	return LogField{Key: key, Value: value}
}

func Time(key string, value time.Time) LogField {
	return LogField{Key: key, Value: value}
}

func ErrorField(key string, value error) LogField {
	return LogField{Key: key, Value: value}
}

func Any(key string, value interface{}) LogField {
	return LogField{Key: key, Value: value}
}

// QueryLogger provides specialized logging for database queries
type QueryLogger struct {
	Logger
	slowQueryThreshold time.Duration
}

// NewQueryLogger creates a new query logger
func NewQueryLogger(logger Logger, slowQueryThreshold time.Duration) *QueryLogger {
	return &QueryLogger{
		Logger:             logger,
		slowQueryThreshold: slowQueryThreshold,
	}
}

// LogQuery logs a database query
func (ql *QueryLogger) LogQuery(ctx context.Context, query string, duration time.Duration, rows int64, err error) {
	fields := []LogField{
		String("query", query),
		Duration("duration", duration),
		Int64("rows", rows),
	}

	if err != nil {
		fields = append(fields, ErrorField("error", err))
		ql.Error(ctx, "Query failed", fields...)
		return
	}

	if duration > ql.slowQueryThreshold {
		ql.Warn(ctx, "Slow query detected", fields...)
	} else {
		ql.Debug(ctx, "Query executed", fields...)
	}
}

// LogTransaction logs a database transaction
func (ql *QueryLogger) LogTransaction(ctx context.Context, operation string, duration time.Duration, err error) {
	fields := []LogField{
		String("operation", operation),
		Duration("duration", duration),
	}

	if err != nil {
		fields = append(fields, ErrorField("error", err))
		ql.Error(ctx, "Transaction failed", fields...)
	} else {
		ql.Info(ctx, "Transaction completed", fields...)
	}
}

// PerformanceLogger provides specialized logging for performance metrics
type PerformanceLogger struct {
	Logger
}

// NewPerformanceLogger creates a new performance logger
func NewPerformanceLogger(logger Logger) *PerformanceLogger {
	return &PerformanceLogger{
		Logger: logger,
	}
}

// LogOperation logs a performance operation
func (ql *PerformanceLogger) LogOperation(ctx context.Context, operation string, duration time.Duration, metrics map[string]interface{}) {
	fields := []LogField{
		String("operation", operation),
		Duration("duration", duration),
	}

	for key, value := range metrics {
		fields = append(fields, Any(key, value))
	}

	ql.Info(ctx, "Operation completed", fields...)
}

// LogMemoryUsage logs memory usage
func (ql *PerformanceLogger) LogMemoryUsage(ctx context.Context, allocated, total int64) {
	fields := []LogField{
		Int64("allocated_mb", allocated/1024/1024),
		Int64("total_mb", total/1024/1024),
		Float64("usage_percent", float64(allocated)/float64(total)*100),
	}

	ql.Info(ctx, "Memory usage", fields...)
}

// LogGoroutineCount logs goroutine count
func (ql *PerformanceLogger) LogGoroutineCount(ctx context.Context, count int) {
	fields := []LogField{
		Int("goroutine_count", count),
	}

	ql.Info(ctx, "Goroutine count", fields...)
}

// LoggerConfig represents logger configuration
type LoggerConfig struct {
	Level              LogLevel
	Format             string // "json" or "text"
	Output             string // "stdout", "stderr", or file path
	SlowQueryThreshold time.Duration
	EnableCaller       bool
	EnableTimestamp    bool
}

// DefaultLoggerConfig returns default logger configuration
func DefaultLoggerConfig() LoggerConfig {
	return LoggerConfig{
		Level:              LogLevelInfo,
		Format:             "text",
		Output:             "stdout",
		SlowQueryThreshold: time.Second,
		EnableCaller:       false,
		EnableTimestamp:    true,
	}
}

// NewLoggerFromConfig creates a new logger from configuration
func NewLoggerFromConfig(config LoggerConfig) (Logger, error) {
	var output io.Writer
	switch config.Output {
	case "stdout", "":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	default:
		file, err := os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			// Fallback to stdout if file can't be opened
			output = os.Stdout
		} else {
			output = file
		}
	}

	var formatter LogFormatter
	switch config.Format {
	case "json":
		formatter = &JSONFormatter{}
	case "text":
		formatter = &TextFormatter{}
	default:
		formatter = &TextFormatter{}
	}

	return NewLogger(config.Level, output, formatter), nil
}

// MultiLogger logs to multiple outputs
type MultiLogger struct {
	loggers []Logger
}

// NewMultiLogger creates a new multi-logger
func NewMultiLogger(loggers ...Logger) *MultiLogger {
	return &MultiLogger{
		loggers: loggers,
	}
}

// Debug logs to all loggers
func (ml *MultiLogger) Debug(ctx context.Context, message string, fields ...LogField) {
	for _, logger := range ml.loggers {
		if logger != nil {
			logger.Debug(ctx, message, fields...)
		}
	}
}

// Info logs to all loggers
func (ml *MultiLogger) Info(ctx context.Context, message string, fields ...LogField) {
	for _, logger := range ml.loggers {
		if logger != nil {
			logger.Info(ctx, message, fields...)
		}
	}
}

// Warn logs to all loggers
func (ml *MultiLogger) Warn(ctx context.Context, message string, fields ...LogField) {
	for _, logger := range ml.loggers {
		if logger != nil {
			logger.Warn(ctx, message, fields...)
		}
	}
}

// Error logs to all loggers
func (ml *MultiLogger) Error(ctx context.Context, message string, fields ...LogField) {
	for _, logger := range ml.loggers {
		if logger != nil {
			logger.Error(ctx, message, fields...)
		}
	}
}

// Fatal logs to all loggers and exits
func (ml *MultiLogger) Fatal(ctx context.Context, message string, fields ...LogField) {
	for _, logger := range ml.loggers {
		if logger != nil {
			logger.Fatal(ctx, message, fields...)
		}
	}
}

// WithContext creates a new logger with context
func (ml *MultiLogger) WithContext(ctx context.Context) Logger {
	loggers := make([]Logger, len(ml.loggers))
	for i, logger := range ml.loggers {
		if logger != nil {
			loggers[i] = logger.WithContext(ctx)
		} else {
			loggers[i] = nil
		}
	}
	return NewMultiLogger(loggers...)
}

// WithFields creates a new logger with additional fields
func (ml *MultiLogger) WithFields(fields ...LogField) Logger {
	loggers := make([]Logger, len(ml.loggers))
	for i, logger := range ml.loggers {
		if logger != nil {
			loggers[i] = logger.WithFields(fields...)
		} else {
			loggers[i] = nil
		}
	}
	return NewMultiLogger(loggers...)
}

// SetLevel sets the log level for all loggers
func (ml *MultiLogger) SetLevel(level LogLevel) {
	for _, logger := range ml.loggers {
		logger.SetLevel(level)
	}
}

// GetLevel gets the log level (returns the first logger's level)
func (ml *MultiLogger) GetLevel() LogLevel {
	if len(ml.loggers) > 0 {
		return ml.loggers[0].GetLevel()
	}
	return LogLevelInfo
}

// Close closes all loggers
func (ml *MultiLogger) Close() error {
	var errs []error
	for _, logger := range ml.loggers {
		if err := logger.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to close loggers: %v", errs)
	}
	return nil
}
