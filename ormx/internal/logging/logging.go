// Package logging provides structured logging for database operations.
// It includes GORM logger integration and database-specific logging utilities.
// This package provides a flexible logging interface for database operations.
package logging

import (
	"context"
	"database/sql/driver"
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"time"
	"unicode"

	"gorm.io/gorm/logger"
)

// LogLevel represents the logging level for database operations
type LogLevel int

const (
	// Silent disables all logging
	Silent LogLevel = iota + 1
	// Error only logs errors
	Error
	// Warn logs warnings and errors
	Warn
	// Info logs info, warnings and errors
	Info
)

// Logger interface defines the logging methods used by the database layer
type Logger interface {
	Trace(msg string, fields ...LogField)
	Debug(msg string, fields ...LogField)
	Info(msg string, fields ...LogField)
	Warn(msg string, fields ...LogField)
	Error(msg string, fields ...LogField)
	Fatal(msg string, fields ...LogField)
	With(fields ...LogField) Logger
}

// LogField represents a structured log field
type LogField struct {
	Key   string
	Value interface{}
}

// Field helper functions
func String(key string, value string) LogField {
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

func ErrorField(err error) LogField {
	return LogField{Key: "error", Value: err}
}

func Any(key string, value interface{}) LogField {
	return LogField{Key: key, Value: value}
}

// DBLogger provides structured logging for database operations
type DBLogger struct {
	logger               Logger
	logLevel             LogLevel
	ignoreRecordNotFound bool
	slowThreshold        time.Duration
	colorful             bool
	sourceField          string
}

// NewDBLogger creates a new database logger
func NewDBLogger(logger Logger, config LoggerConfig) *DBLogger {
	return &DBLogger{
		logger:               logger,
		logLevel:             config.LogLevel,
		ignoreRecordNotFound: config.IgnoreRecordNotFound,
		slowThreshold:        config.SlowThreshold,
		colorful:             config.Colorful,
		sourceField:          config.SourceField,
	}
}

// LoggerConfig holds configuration for the database logger
type LoggerConfig struct {
	LogLevel             LogLevel
	IgnoreRecordNotFound bool
	SlowThreshold        time.Duration
	Colorful             bool
	SourceField          string
}

// DefaultLoggerConfig returns a default logger configuration
func DefaultLoggerConfig() LoggerConfig {
	return LoggerConfig{
		LogLevel:             Warn,
		IgnoreRecordNotFound: true,
		SlowThreshold:        200 * time.Millisecond,
		Colorful:             false,
		SourceField:          "source",
	}
}

// LogMode implements the go-ormx logger.Interface
func (l *DBLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	switch level {
	case logger.Silent:
		newLogger.logLevel = Silent
	case logger.Error:
		newLogger.logLevel = Error
	case logger.Warn:
		newLogger.logLevel = Warn
	case logger.Info:
		newLogger.logLevel = Info
	}
	return &newLogger
}

// Info implements the go-ormx logger.Interface
func (l *DBLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= Info {
		fields := l.extractContextFields(ctx)
		if len(data) > 0 {
			fields = append(fields, Any("data", data))
		}
		l.logger.Info(msg, fields...)
	}
}

// Warn implements the go-ormx logger.Interface
func (l *DBLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= Warn {
		fields := l.extractContextFields(ctx)
		if len(data) > 0 {
			fields = append(fields, Any("data", data))
		}
		l.logger.Warn(msg, fields...)
	}
}

// Error implements the go-ormx logger.Interface
func (l *DBLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= Error {
		fields := l.extractContextFields(ctx)
		if len(data) > 0 {
			fields = append(fields, Any("data", data))
		}
		l.logger.Error(msg, fields...)
	}
}

// Trace implements the go-ormx logger.Interface for SQL tracing
func (l *DBLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.logLevel <= Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	fields := l.extractContextFields(ctx)
	fields = append(fields,
		Duration("elapsed", elapsed),
		String("sql", l.sanitizeSQL(sql)),
		Int64("rows_affected", rows),
	)

	// Add source information
	if l.sourceField != "" {
		if file, line := l.getCallerInfo(4); file != "" {
			fields = append(fields, String(l.sourceField, fmt.Sprintf("%s:%d", file, line)))
		}
	}

	switch {
	case err != nil && l.logLevel >= Error && (!l.ignoreRecordNotFound || !isRecordNotFoundError(err)):
		fields = append(fields, ErrorField(err))
		l.logger.Error("Database query failed", fields...)
	case elapsed > l.slowThreshold && l.logLevel >= Warn:
		l.logger.Warn("Slow SQL query detected", fields...)
	case l.logLevel >= Info:
		l.logger.Info("Database query executed", fields...)
	}
}

// extractContextFields extracts relevant fields from context
func (l *DBLogger) extractContextFields(ctx context.Context) []LogField {
	var fields []LogField

	// Extract trace ID if present
	if traceID := getTraceIDFromContext(ctx); traceID != "" {
		fields = append(fields, String("trace_id", traceID))
	}

	// Extract request ID if present
	if requestID := getRequestIDFromContext(ctx); requestID != "" {
		fields = append(fields, String("request_id", requestID))
	}

	// Extract user ID if present
	if userID := getUserIDFromContext(ctx); userID != "" {
		fields = append(fields, String("user_id", userID))
	}

	// Extract tenant ID if present
	if tenantID := getTenantIDFromContext(ctx); tenantID != "" {
		fields = append(fields, String("tenant_id", tenantID))
	}

	return fields
}

// sanitizeSQL removes sensitive information from SQL queries for logging
func (l *DBLogger) sanitizeSQL(sql string) string {
	// Remove potential sensitive data patterns
	sensitivePatterns := []string{
		`(?i)(password\s*=\s*['"])[^'"]*(['"])`,
		`(?i)(token\s*=\s*['"])[^'"]*(['"])`,
		`(?i)(secret\s*=\s*['"])[^'"]*(['"])`,
		`(?i)(key\s*=\s*['"])[^'"]*(['"])`,
	}

	sanitized := sql
	for _, pattern := range sensitivePatterns {
		re := regexp.MustCompile(pattern)
		sanitized = re.ReplaceAllString(sanitized, "${1}***${2}")
	}

	// Limit SQL length for logging
	const maxSQLLength = 1000
	if len(sanitized) > maxSQLLength {
		sanitized = sanitized[:maxSQLLength] + "... (truncated)"
	}

	return sanitized
}

// getCallerInfo returns the file and line number of the caller
func (l *DBLogger) getCallerInfo(skip int) (string, int) {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "", 0
	}

	// Get only the filename, not the full path
	parts := strings.Split(file, "/")
	if len(parts) > 0 {
		file = parts[len(parts)-1]
	}

	return file, line
}

// Helper functions to extract context values
func getTraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if val := ctx.Value("trace_id"); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	// Try OpenTelemetry trace ID
	if val := ctx.Value("otel_trace_id"); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getRequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if val := ctx.Value("request_id"); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getUserIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if val := ctx.Value("user_id"); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getTenantIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if val := ctx.Value("tenant_id"); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// isRecordNotFoundError checks if the error is a "record not found" error
func isRecordNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "record not found") ||
		strings.Contains(errStr, "no rows") ||
		strings.Contains(errStr, "not found")
}

// isPrintable checks if a value is printable
func isPrintable(s string) bool {
	for _, r := range s {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}

// formatValue formats a driver.Value for logging
func formatValue(value driver.Value) string {
	if value == nil {
		return "NULL"
	}

	switch v := value.(type) {
	case string:
		if isPrintable(v) {
			return fmt.Sprintf("'%s'", v)
		}
		return "'<binary>'"
	case []byte:
		if isPrintable(string(v)) {
			return fmt.Sprintf("'%s'", string(v))
		}
		return "'<binary>'"
	case time.Time:
		return fmt.Sprintf("'%s'", v.Format(time.RFC3339))
	case fmt.Stringer:
		return fmt.Sprintf("'%s'", v.String())
	default:
		return fmt.Sprintf("%v", v)
	}
}

// OperationLogger provides structured logging for database operations
type OperationLogger struct {
	logger    Logger
	operation string
	table     string
	startTime time.Time
	fields    []LogField
}

// NewOperationLogger creates a new operation logger
func NewOperationLogger(logger Logger, operation, table string) *OperationLogger {
	return &OperationLogger{
		logger:    logger,
		operation: operation,
		table:     table,
		startTime: time.Now(),
		fields:    make([]LogField, 0),
	}
}

// WithField adds a field to the operation logger
func (ol *OperationLogger) WithField(key string, value interface{}) *OperationLogger {
	ol.fields = append(ol.fields, LogField{Key: key, Value: value})
	return ol
}

// WithFields adds multiple fields to the operation logger
func (ol *OperationLogger) WithFields(fields ...LogField) *OperationLogger {
	ol.fields = append(ol.fields, fields...)
	return ol
}

// Success logs a successful operation
func (ol *OperationLogger) Success(message string) {
	duration := time.Since(ol.startTime)
	fields := append(ol.fields,
		String("operation", ol.operation),
		String("table", ol.table),
		Duration("duration", duration),
		String("status", "success"),
	)
	ol.logger.Info(message, fields...)
}

// Error logs a failed operation
func (ol *OperationLogger) Error(message string, err error) {
	duration := time.Since(ol.startTime)
	fields := append(ol.fields,
		String("operation", ol.operation),
		String("table", ol.table),
		Duration("duration", duration),
		String("status", "error"),
		ErrorField(err),
	)
	ol.logger.Error(message, fields...)
}

// Warn logs a warning for an operation
func (ol *OperationLogger) Warn(message string) {
	duration := time.Since(ol.startTime)
	fields := append(ol.fields,
		String("operation", ol.operation),
		String("table", ol.table),
		Duration("duration", duration),
		String("status", "warning"),
	)
	ol.logger.Warn(message, fields...)
}
