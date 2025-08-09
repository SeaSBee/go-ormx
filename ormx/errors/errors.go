// Package errors provides comprehensive error handling for database operations
// with structured error codes, categorization, and detailed error information.
package errors

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ErrorCode represents a specific error code
type ErrorCode string

// Database error codes
const (
	// Connection errors
	ErrCodeConnectionFailed   ErrorCode = "DB_CONNECTION_FAILED"
	ErrCodeConnectionTimeout  ErrorCode = "DB_CONNECTION_TIMEOUT"
	ErrCodeConnectionLeak     ErrorCode = "DB_CONNECTION_LEAK"
	ErrCodeConnectionPoolFull ErrorCode = "DB_CONNECTION_POOL_FULL"

	// Query errors
	ErrCodeQueryTimeout   ErrorCode = "DB_QUERY_TIMEOUT"
	ErrCodeQuerySyntax    ErrorCode = "DB_QUERY_SYNTAX"
	ErrCodeQueryExecution ErrorCode = "DB_QUERY_EXECUTION"
	ErrCodeInvalidQuery   ErrorCode = "DB_INVALID_QUERY"

	// Data errors
	ErrCodeRecordNotFound      ErrorCode = "DB_RECORD_NOT_FOUND"
	ErrCodeDuplicateKey        ErrorCode = "DB_DUPLICATE_KEY"
	ErrCodeConstraintViolation ErrorCode = "DB_CONSTRAINT_VIOLATION"
	ErrCodeDataTooLong         ErrorCode = "DB_DATA_TOO_LONG"
	ErrCodeInvalidData         ErrorCode = "DB_INVALID_DATA"

	// Transaction errors
	ErrCodeTransactionFailed  ErrorCode = "DB_TRANSACTION_FAILED"
	ErrCodeTransactionTimeout ErrorCode = "DB_TRANSACTION_TIMEOUT"
	ErrCodeDeadlock           ErrorCode = "DB_DEADLOCK"
	ErrCodeLockTimeout        ErrorCode = "DB_LOCK_TIMEOUT"

	// Migration errors
	ErrCodeMigrationFailed ErrorCode = "DB_MIGRATION_FAILED"
	ErrCodeSchemaConflict  ErrorCode = "DB_SCHEMA_CONFLICT"
	ErrCodeVersionMismatch ErrorCode = "DB_VERSION_MISMATCH"

	// Security errors
	ErrCodeSQLInjection     ErrorCode = "DB_SQL_INJECTION"
	ErrCodeUnauthorized     ErrorCode = "DB_UNAUTHORIZED"
	ErrCodePermissionDenied ErrorCode = "DB_PERMISSION_DENIED"
	ErrCodeInvalidInput     ErrorCode = "DB_INVALID_INPUT"

	// Configuration errors
	ErrCodeInvalidConfig    ErrorCode = "DB_INVALID_CONFIG"
	ErrCodeMissingConfig    ErrorCode = "DB_MISSING_CONFIG"
	ErrCodeConfigValidation ErrorCode = "DB_CONFIG_VALIDATION"

	// General errors
	ErrCodeUnknown         ErrorCode = "DB_UNKNOWN_ERROR"
	ErrCodeInternal        ErrorCode = "DB_INTERNAL_ERROR"
	ErrCodeNotImplemented  ErrorCode = "DB_NOT_IMPLEMENTED"
	ErrCodeOperationFailed ErrorCode = "DB_OPERATION_FAILED"
)

// ErrorCategory represents the category of an error
type ErrorCategory string

const (
	CategoryConnection    ErrorCategory = "CONNECTION"
	CategoryQuery         ErrorCategory = "QUERY"
	CategoryData          ErrorCategory = "DATA"
	CategoryTransaction   ErrorCategory = "TRANSACTION"
	CategoryMigration     ErrorCategory = "MIGRATION"
	CategorySecurity      ErrorCategory = "SECURITY"
	CategoryConfiguration ErrorCategory = "CONFIGURATION"
	CategoryGeneral       ErrorCategory = "GENERAL"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity string

const (
	SeverityLow      ErrorSeverity = "LOW"
	SeverityMedium   ErrorSeverity = "MEDIUM"
	SeverityHigh     ErrorSeverity = "HIGH"
	SeverityCritical ErrorSeverity = "CRITICAL"
)

// DBError represents a structured database error
type DBError struct {
	Code        ErrorCode     `json:"code"`
	Category    ErrorCategory `json:"category"`
	Severity    ErrorSeverity `json:"severity"`
	Message     string        `json:"message"`
	Details     string        `json:"details,omitempty"`
	Cause       error         `json:"-"`
	Operation   string        `json:"operation,omitempty"`
	Table       string        `json:"table,omitempty"`
	Field       string        `json:"field,omitempty"`
	Timestamp   time.Time     `json:"timestamp"`
	StackTrace  string        `json:"stack_trace,omitempty"`
	Retryable   bool          `json:"retryable"`
	UserMessage string        `json:"user_message,omitempty"`
}

// Error implements the error interface
func (e *DBError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *DBError) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches the target error
func (e *DBError) Is(target error) bool {
	if target == nil {
		return false
	}

	// Check if target is a DBError with the same code
	if dbErr, ok := target.(*DBError); ok {
		return e.Code == dbErr.Code
	}

	// Check if the underlying cause matches
	if e.Cause != nil {
		return errors.Is(e.Cause, target)
	}

	return false
}

// As checks if the error can be assigned to the target
func (e *DBError) As(target interface{}) bool {
	if target == nil {
		return false
	}

	// Check if target is a *DBError
	if dbErr, ok := target.(**DBError); ok {
		*dbErr = e
		return true
	}

	// Check if the underlying cause can be assigned
	if e.Cause != nil {
		return errors.As(e.Cause, target)
	}

	return false
}

// WithOperation adds operation context to the error
func (e *DBError) WithOperation(operation string) *DBError {
	newErr := *e
	newErr.Operation = operation
	return &newErr
}

// WithTable adds table context to the error
func (e *DBError) WithTable(table string) *DBError {
	newErr := *e
	newErr.Table = table
	return &newErr
}

// WithField adds field context to the error
func (e *DBError) WithField(field string) *DBError {
	newErr := *e
	newErr.Field = field
	return &newErr
}

// WithDetails adds additional details to the error
func (e *DBError) WithDetails(details string) *DBError {
	newErr := *e
	newErr.Details = details
	return &newErr
}

// WithUserMessage adds a user-friendly message
func (e *DBError) WithUserMessage(message string) *DBError {
	newErr := *e
	newErr.UserMessage = message
	return &newErr
}

// NewDBError creates a new database error
func NewDBError(code ErrorCode, message string, cause error) *DBError {
	category, severity := getErrorMetadata(code)

	var stackTrace string
	if includeStackTrace(severity) {
		stackTrace = getStackTrace(2) // Skip this function and the caller
	}

	return &DBError{
		Code:       code,
		Category:   category,
		Severity:   severity,
		Message:    message,
		Cause:      cause,
		Timestamp:  time.Now(),
		StackTrace: stackTrace,
		Retryable:  isRetryable(code),
	}
}

// getErrorMetadata returns the category and severity for an error code
func getErrorMetadata(code ErrorCode) (ErrorCategory, ErrorSeverity) {
	switch code {
	// Connection errors
	case ErrCodeConnectionFailed, ErrCodeConnectionTimeout:
		return CategoryConnection, SeverityHigh
	case ErrCodeConnectionLeak, ErrCodeConnectionPoolFull:
		return CategoryConnection, SeverityCritical

	// Query errors
	case ErrCodeQueryTimeout, ErrCodeQuerySyntax:
		return CategoryQuery, SeverityMedium
	case ErrCodeQueryExecution, ErrCodeInvalidQuery:
		return CategoryQuery, SeverityMedium

	// Data errors
	case ErrCodeRecordNotFound:
		return CategoryData, SeverityLow
	case ErrCodeDuplicateKey, ErrCodeConstraintViolation:
		return CategoryData, SeverityMedium
	case ErrCodeDataTooLong, ErrCodeInvalidData:
		return CategoryData, SeverityMedium

	// Transaction errors
	case ErrCodeTransactionFailed, ErrCodeTransactionTimeout:
		return CategoryTransaction, SeverityHigh
	case ErrCodeDeadlock, ErrCodeLockTimeout:
		return CategoryTransaction, SeverityMedium

	// Migration errors
	case ErrCodeMigrationFailed, ErrCodeSchemaConflict, ErrCodeVersionMismatch:
		return CategoryMigration, SeverityCritical

	// Security errors
	case ErrCodeSQLInjection, ErrCodeUnauthorized, ErrCodePermissionDenied:
		return CategorySecurity, SeverityCritical
	case ErrCodeInvalidInput:
		return CategorySecurity, SeverityMedium

	// Configuration errors
	case ErrCodeInvalidConfig, ErrCodeMissingConfig, ErrCodeConfigValidation:
		return CategoryConfiguration, SeverityHigh

	// General errors
	default:
		return CategoryGeneral, SeverityMedium
	}
}

// isRetryable determines if an error is retryable
func isRetryable(code ErrorCode) bool {
	switch code {
	case ErrCodeConnectionFailed, ErrCodeConnectionTimeout, ErrCodeConnectionPoolFull,
		ErrCodeQueryTimeout, ErrCodeTransactionTimeout, ErrCodeDeadlock, ErrCodeLockTimeout:
		return true
	default:
		return false
	}
}

// includeStackTrace determines if a stack trace should be included based on severity
func includeStackTrace(severity ErrorSeverity) bool {
	return severity == SeverityHigh || severity == SeverityCritical
}

// getStackTrace captures the current stack trace
func getStackTrace(skip int) string {
	const maxStackDepth = 10
	pc := make([]uintptr, maxStackDepth)
	n := runtime.Callers(skip+1, pc)

	if n == 0 {
		return "no stack trace available"
	}

	frames := runtime.CallersFrames(pc[:n])
	var stackLines []string

	for {
		frame, more := frames.Next()
		if !more {
			break
		}

		// Format: function (file:line)
		stackLine := fmt.Sprintf("%s (%s:%d)", frame.Function, frame.File, frame.Line)
		stackLines = append(stackLines, stackLine)
	}

	return strings.Join(stackLines, "\n")
}

// WrapGormError wraps a GORM error into a structured DBError
func WrapGormError(err error, operation string) *DBError {
	if err == nil {
		return nil
	}

	// Check for specific GORM errors
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return NewDBError(ErrCodeRecordNotFound, "Record not found", err).
			WithOperation(operation).
			WithUserMessage("The requested record was not found")

	case errors.Is(err, gorm.ErrInvalidTransaction):
		return NewDBError(ErrCodeTransactionFailed, "Invalid transaction", err).
			WithOperation(operation)

	case errors.Is(err, gorm.ErrNotImplemented):
		return NewDBError(ErrCodeNotImplemented, "Operation not implemented", err).
			WithOperation(operation)

	case errors.Is(err, gorm.ErrMissingWhereClause):
		return NewDBError(ErrCodeInvalidQuery, "Missing WHERE clause", err).
			WithOperation(operation).
			WithUserMessage("Query requires a WHERE clause for safety")

	case errors.Is(err, gorm.ErrUnsupportedRelation):
		return NewDBError(ErrCodeQueryExecution, "Unsupported relation", err).
			WithOperation(operation)

	case errors.Is(err, gorm.ErrPrimaryKeyRequired):
		return NewDBError(ErrCodeInvalidData, "Primary key required", err).
			WithOperation(operation)

	case errors.Is(err, gorm.ErrModelValueRequired):
		return NewDBError(ErrCodeInvalidData, "Model value required", err).
			WithOperation(operation)

	case errors.Is(err, gorm.ErrInvalidData):
		return NewDBError(ErrCodeInvalidData, "Invalid data", err).
			WithOperation(operation)

	default:
		// Try to categorize based on error message
		errMsg := strings.ToLower(err.Error())

		switch {
		case strings.Contains(errMsg, "connection"):
			if strings.Contains(errMsg, "timeout") {
				return NewDBError(ErrCodeConnectionTimeout, "Database connection timeout", err).
					WithOperation(operation)
			}
			return NewDBError(ErrCodeConnectionFailed, "Database connection failed", err).
				WithOperation(operation)

		case strings.Contains(errMsg, "timeout"):
			return NewDBError(ErrCodeQueryTimeout, "Query timeout", err).
				WithOperation(operation)

		case strings.Contains(errMsg, "duplicate") || strings.Contains(errMsg, "unique"):
			return NewDBError(ErrCodeDuplicateKey, "Duplicate key violation", err).
				WithOperation(operation).
				WithUserMessage("A record with this identifier already exists")

		case strings.Contains(errMsg, "constraint"):
			return NewDBError(ErrCodeConstraintViolation, "Constraint violation", err).
				WithOperation(operation).
				WithUserMessage("Data violates database constraints")

		case strings.Contains(errMsg, "deadlock"):
			return NewDBError(ErrCodeDeadlock, "Database deadlock detected", err).
				WithOperation(operation)

		case strings.Contains(errMsg, "syntax"):
			return NewDBError(ErrCodeQuerySyntax, "SQL syntax error", err).
				WithOperation(operation)

		case strings.Contains(errMsg, "data too long"):
			return NewDBError(ErrCodeDataTooLong, "Data too long for field", err).
				WithOperation(operation).
				WithUserMessage("One or more fields exceed the maximum length")

		default:
			return NewDBError(ErrCodeUnknown, "Unknown database error", err).
				WithOperation(operation)
		}
	}
}

// WrapError wraps a generic error into a DBError
func WrapError(err error, code ErrorCode, message string) *DBError {
	if err == nil {
		return nil
	}
	return NewDBError(code, message, err)
}

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	if dbErr, ok := err.(*DBError); ok {
		return dbErr.Retryable
	}
	return false
}

// IsConnectionError checks if an error is related to database connection
func IsConnectionError(err error) bool {
	if dbErr, ok := err.(*DBError); ok {
		return dbErr.Category == CategoryConnection
	}
	return false
}

// IsDataError checks if an error is related to data validation or constraints
func IsDataError(err error) bool {
	if dbErr, ok := err.(*DBError); ok {
		return dbErr.Category == CategoryData
	}
	return false
}

// IsSecurityError checks if an error is related to security
func IsSecurityError(err error) bool {
	if dbErr, ok := err.(*DBError); ok {
		return dbErr.Category == CategorySecurity
	}
	return false
}

// GetErrorCode extracts the error code from an error
func GetErrorCode(err error) ErrorCode {
	if dbErr, ok := err.(*DBError); ok {
		return dbErr.Code
	}
	return ErrCodeUnknown
}

// GetUserMessage extracts a user-friendly message from an error
func GetUserMessage(err error) string {
	if dbErr, ok := err.(*DBError); ok && dbErr.UserMessage != "" {
		return dbErr.UserMessage
	}

	// Return a generic message for unknown errors
	return "An error occurred while processing your request"
}

// ErrorCollector collects multiple errors and provides aggregated error handling
type ErrorCollector struct {
	errors   []error
	maxCount int
}

// NewErrorCollector creates a new error collector
func NewErrorCollector(maxCount int) *ErrorCollector {
	return &ErrorCollector{
		errors:   make([]error, 0),
		maxCount: maxCount,
	}
}

// Add adds an error to the collector
func (ec *ErrorCollector) Add(err error) {
	if err != nil && len(ec.errors) < ec.maxCount {
		ec.errors = append(ec.errors, err)
	}
}

// HasErrors returns true if there are any errors
func (ec *ErrorCollector) HasErrors() bool {
	return len(ec.errors) > 0
}

// Count returns the number of errors
func (ec *ErrorCollector) Count() int {
	return len(ec.errors)
}

// Errors returns all collected errors
func (ec *ErrorCollector) Errors() []error {
	return ec.errors
}

// First returns the first error
func (ec *ErrorCollector) First() error {
	if len(ec.errors) > 0 {
		return ec.errors[0]
	}
	return nil
}

// Error returns a combined error message
func (ec *ErrorCollector) Error() string {
	if len(ec.errors) == 0 {
		return ""
	}

	if len(ec.errors) == 1 {
		return ec.errors[0].Error()
	}

	var messages []string
	for i, err := range ec.errors {
		messages = append(messages, fmt.Sprintf("%d: %s", i+1, err.Error()))
	}

	return fmt.Sprintf("Multiple errors occurred:\n%s", strings.Join(messages, "\n"))
}

// ToDBError converts the collector's errors to a single DBError
func (ec *ErrorCollector) ToDBError(operation string) *DBError {
	if len(ec.errors) == 0 {
		return nil
	}

	if len(ec.errors) == 1 {
		if dbErr, ok := ec.errors[0].(*DBError); ok {
			return dbErr.WithOperation(operation)
		}
		return WrapError(ec.errors[0], ErrCodeOperationFailed, "Operation failed").
			WithOperation(operation)
	}

	return NewDBError(ErrCodeOperationFailed, "Multiple errors occurred", fmt.Errorf("%s", ec.Error())).
		WithOperation(operation)
}
