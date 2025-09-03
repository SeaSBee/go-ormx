package errors

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ErrorType represents the type of error
type ErrorType string

const (
	// Database errors
	ErrorTypeConnection  ErrorType = "connection"
	ErrorTypeQuery       ErrorType = "query"
	ErrorTypeTransaction ErrorType = "transaction"
	ErrorTypeTimeout     ErrorType = "timeout"
	ErrorTypeDeadlock    ErrorType = "deadlock"
	ErrorTypeConstraint  ErrorType = "constraint"
	ErrorTypeNotFound    ErrorType = "not_found"
	ErrorTypeDuplicate   ErrorType = "duplicate"
	ErrorTypeValidation  ErrorType = "validation"
	ErrorTypeMigration   ErrorType = "migration"
	ErrorTypeReplication ErrorType = "replication"

	// ORM specific errors
	ErrorTypeModel        ErrorType = "model"
	ErrorTypeRelationship ErrorType = "relationship"
	ErrorTypeAssociation  ErrorType = "association"
	ErrorTypeHook         ErrorType = "hook"
	ErrorTypeCallback     ErrorType = "callback"

	// Configuration errors
	ErrorTypeConfig         ErrorType = "config"
	ErrorTypeInitialization ErrorType = "initialization"

	// Cache errors
	ErrorTypeCache        ErrorType = "cache"
	ErrorTypeCacheMiss    ErrorType = "cache_miss"
	ErrorTypeCacheInvalid ErrorType = "cache_invalid"

	// Security errors
	ErrorTypeSecurity     ErrorType = "security"
	ErrorTypeSQLInjection ErrorType = "sql_injection"
	ErrorTypeAccessDenied ErrorType = "access_denied"

	// System errors
	ErrorTypeSystem   ErrorType = "system"
	ErrorTypeResource ErrorType = "resource"
	ErrorTypeNetwork  ErrorType = "network"
	ErrorTypeUnknown  ErrorType = "unknown"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity string

const (
	ErrorSeverityLow      ErrorSeverity = "1_low"
	ErrorSeverityMedium   ErrorSeverity = "2_medium"
	ErrorSeverityHigh     ErrorSeverity = "3_high"
	ErrorSeverityCritical ErrorSeverity = "4_critical"
)

// ORMError represents a comprehensive ORM error
type ORMError struct {
	Type       ErrorType              `json:"type"`
	Severity   ErrorSeverity          `json:"severity"`
	Message    string                 `json:"message"`
	Code       string                 `json:"code,omitempty"`
	Details    string                 `json:"details,omitempty"`
	Operation  string                 `json:"operation,omitempty"`
	Table      string                 `json:"table,omitempty"`
	Field      string                 `json:"field,omitempty"`
	Value      interface{}            `json:"value,omitempty"`
	Query      string                 `json:"query,omitempty"`
	Params     []interface{}          `json:"params,omitempty"`
	Retryable  bool                   `json:"retryable"`
	RetryCount int                    `json:"retry_count,omitempty"`
	RetryDelay time.Duration          `json:"retry_delay,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Stack      string                 `json:"stack,omitempty"`
	Cause      error                  `json:"-"`
}

// Error implements the error interface
func (e *ORMError) Error() string {
	if e.Type == "" {
		// If no type is specified, just return the message
		if e.Cause != nil {
			return fmt.Sprintf("%s (caused by: %v)", e.Message, e.Cause)
		}
		return e.Message
	}

	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying cause error
func (e *ORMError) Unwrap() error {
	return e.Cause
}

// IsRetryable returns true if the error is retryable
func (e *ORMError) IsRetryable() bool {
	return e.Retryable
}

// GetType returns the error type
func (e *ORMError) GetType() ErrorType {
	return e.Type
}

// GetSeverity returns the error severity
func (e *ORMError) GetSeverity() ErrorSeverity {
	return e.Severity
}

// GetContext returns the error context
func (e *ORMError) GetContext() map[string]interface{} {
	return e.Context
}

// AddContext adds context information to the error
func (e *ORMError) AddContext(key string, value interface{}) {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
}

// New creates a new ORM error
func New(errorType ErrorType, message string) *ORMError {
	return &ORMError{
		Type:      errorType,
		Severity:  getDefaultSeverity(errorType),
		Message:   message,
		Timestamp: time.Now(),
		Stack:     getStackTrace(),
	}
}

// Wrap wraps an existing error with ORM error information
func Wrap(err error, errorType ErrorType, message string) *ORMError {
	ormErr := New(errorType, message)
	ormErr.Cause = err
	return ormErr
}

// WithSeverity sets the error severity
func (e *ORMError) WithSeverity(severity ErrorSeverity) *ORMError {
	e.Severity = severity
	return e
}

// WithCode sets the error code
func (e *ORMError) WithCode(code string) *ORMError {
	e.Code = code
	return e
}

// WithDetails sets the error details
func (e *ORMError) WithDetails(details string) *ORMError {
	e.Details = details
	return e
}

// WithOperation sets the operation that caused the error
func (e *ORMError) WithOperation(operation string) *ORMError {
	e.Operation = operation
	return e
}

// WithTable sets the table involved in the error
func (e *ORMError) WithTable(table string) *ORMError {
	e.Table = table
	return e
}

// WithField sets the field involved in the error
func (e *ORMError) WithField(field string) *ORMError {
	e.Field = field
	return e
}

// WithValue sets the value involved in the error
func (e *ORMError) WithValue(value interface{}) *ORMError {
	e.Value = value
	return e
}

// WithQuery sets the query that caused the error
func (e *ORMError) WithQuery(query string, params ...interface{}) *ORMError {
	e.Query = query
	e.Params = params
	return e
}

// WithRetry sets retry information
func (e *ORMError) WithRetry(retryable bool, retryCount int, retryDelay time.Duration) *ORMError {
	e.Retryable = retryable
	e.RetryCount = retryCount
	e.RetryDelay = retryDelay
	return e
}

// ErrorBuilder provides a fluent interface for building errors
type ErrorBuilder struct {
	error *ORMError
}

// NewBuilder creates a new error builder
func NewBuilder(errorType ErrorType, message string) *ErrorBuilder {
	return &ErrorBuilder{
		error: New(errorType, message),
	}
}

// Severity sets the error severity
func (b *ErrorBuilder) Severity(severity ErrorSeverity) *ErrorBuilder {
	b.error.Severity = severity
	return b
}

// Code sets the error code
func (b *ErrorBuilder) Code(code string) *ErrorBuilder {
	b.error.Code = code
	return b
}

// Details sets the error details
func (b *ErrorBuilder) Details(details string) *ErrorBuilder {
	b.error.Details = details
	return b
}

// Operation sets the operation
func (b *ErrorBuilder) Operation(operation string) *ErrorBuilder {
	b.error.Operation = operation
	return b
}

// Table sets the table
func (b *ErrorBuilder) Table(table string) *ErrorBuilder {
	b.error.Table = table
	return b
}

// Field sets the field
func (b *ErrorBuilder) Field(field string) *ErrorBuilder {
	b.error.Field = field
	return b
}

// Value sets the value
func (b *ErrorBuilder) Value(value interface{}) *ErrorBuilder {
	b.error.Value = value
	return b
}

// Query sets the query
func (b *ErrorBuilder) Query(query string, params ...interface{}) *ErrorBuilder {
	b.error.Query = query
	b.error.Params = params
	return b
}

// Retry sets retry information
func (b *ErrorBuilder) Retry(retryable bool, retryCount int, retryDelay time.Duration) *ErrorBuilder {
	b.error.Retryable = retryable
	b.error.RetryCount = retryCount
	b.error.RetryDelay = retryDelay
	return b
}

// Context adds context information
func (b *ErrorBuilder) Context(key string, value interface{}) *ErrorBuilder {
	b.error.AddContext(key, value)
	return b
}

// Build returns the built error
func (b *ErrorBuilder) Build() *ORMError {
	return b.error
}

// ErrorClassifier classifies errors based on their type and content
type ErrorClassifier struct {
	// Retryable error patterns
	retryablePatterns []string
	// Non-retryable error patterns
	nonRetryablePatterns []string
	// Error type mappings (ordered for priority)
	typeMappings []struct {
		pattern   string
		errorType ErrorType
	}
}

// NewErrorClassifier creates a new error classifier
func NewErrorClassifier() *ErrorClassifier {
	return &ErrorClassifier{
		retryablePatterns: []string{
			"connection refused",
			"connection reset",
			"connection timeout",
			"context deadline exceeded",
			"operation timed out",
			"deadlock",
			"lock wait timeout",
			"temporary failure",
			"persistent failure",
			"try again",
		},
		nonRetryablePatterns: []string{
			"duplicate key",
			"unique constraint",
			"foreign key constraint",
			"check constraint",
			"not null constraint",
			"invalid syntax",
			"table does not exist",
			"column does not exist",
		},
		typeMappings: []struct {
			pattern   string
			errorType ErrorType
		}{
			{"unique constraint violation", ErrorTypeConstraint},
			{"foreign key constraint", ErrorTypeConstraint},
			{"connection timeout", ErrorTypeTimeout},
			{"timeout", ErrorTypeTimeout},
			{"context deadline exceeded", ErrorTypeTimeout},
			{"operation timed out", ErrorTypeTimeout},
			{"connection", ErrorTypeConnection},
			{"deadlock", ErrorTypeDeadlock},
			{"duplicate", ErrorTypeDuplicate},
			{"not found", ErrorTypeNotFound},
			{"validation", ErrorTypeValidation},
			{"sql injection", ErrorTypeSQLInjection},
			{"table does not exist", ErrorTypeNotFound},
			{"doesn't exist", ErrorTypeNotFound},
			{"column does not exist", ErrorTypeNotFound},
			{"unknown column", ErrorTypeNotFound},
			{"syntax error", ErrorTypeQuery},
		},
	}
}

// ClassifyError classifies an error and returns an ORMError
func (ec *ErrorClassifier) ClassifyError(err error, operation string) *ORMError {
	if err == nil {
		return nil
	}

	// Check if it's already an ORMError
	if ormErr, ok := err.(*ORMError); ok {
		return ormErr
	}

	// Get error message
	message := err.Error()
	lowerMessage := strings.ToLower(message)

	// Determine error type
	errorType := ec.determineErrorType(lowerMessage)

	// Determine if retryable
	retryable := ec.isRetryable(lowerMessage)

	// Create ORM error
	ormErr := New(errorType, message)
	ormErr.Operation = operation
	ormErr.Retryable = retryable
	ormErr.Cause = err

	return ormErr
}

// determineErrorType determines the error type based on the error message
func (ec *ErrorClassifier) determineErrorType(message string) ErrorType {
	for _, mapping := range ec.typeMappings {
		if strings.Contains(message, mapping.pattern) {
			return mapping.errorType
		}
	}
	return ErrorTypeUnknown
}

// isRetryable determines if an error is retryable
func (ec *ErrorClassifier) isRetryable(message string) bool {
	// Check non-retryable patterns first
	for _, pattern := range ec.nonRetryablePatterns {
		if strings.Contains(message, pattern) {
			return false
		}
	}

	// Check retryable patterns
	for _, pattern := range ec.retryablePatterns {
		if strings.Contains(message, pattern) {
			return true
		}
	}

	return false
}

// ErrorHandler handles errors with retry logic and logging
type ErrorHandler struct {
	classifier *ErrorClassifier
	maxRetries int
	retryDelay time.Duration
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(maxRetries int, retryDelay time.Duration) *ErrorHandler {
	return &ErrorHandler{
		classifier: NewErrorClassifier(),
		maxRetries: maxRetries,
		retryDelay: retryDelay,
	}
}

// HandleError handles an error with retry logic
func (eh *ErrorHandler) HandleError(err error, operation string) *ORMError {
	if err == nil {
		return nil
	}

	// Classify the error
	ormErr := eh.classifier.ClassifyError(err, operation)

	// If retryable and under max retries, set retry information
	if ormErr.Retryable && ormErr.RetryCount < eh.maxRetries {
		ormErr.RetryCount++
		ormErr.RetryDelay = eh.retryDelay
	}

	return ormErr
}

// RetryWithContext retries an operation with context
func (eh *ErrorHandler) RetryWithContext(ctx context.Context, operation func() error, operationName string) error {
	var lastErr error

	for attempt := 0; attempt <= eh.maxRetries; attempt++ {
		// Check context before operation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err
		ormErr := eh.HandleError(err, operationName)

		// If not retryable or max retries reached, return error
		if !ormErr.Retryable || attempt == eh.maxRetries {
			return ormErr
		}

		// Wait before retry, but check context during wait
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(ormErr.RetryDelay):
			// Check context again after wait
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}
	}

	return lastErr
}

// getDefaultSeverity returns the default severity for an error type
func getDefaultSeverity(errorType ErrorType) ErrorSeverity {
	switch errorType {
	case ErrorTypeConnection, ErrorTypeTimeout, ErrorTypeDeadlock:
		return ErrorSeverityHigh
	case ErrorTypeConstraint, ErrorTypeValidation, ErrorTypeNotFound:
		return ErrorSeverityMedium
	case ErrorTypeSecurity, ErrorTypeSQLInjection:
		return ErrorSeverityCritical
	default:
		return ErrorSeverityMedium
	}
}

// getStackTrace returns the current stack trace (placeholder)
func getStackTrace() string {
	return "stack trace placeholder"
}

// Common error constructors
func NewConnectionError(message string) *ORMError {
	return New(ErrorTypeConnection, message)
}

func NewQueryError(message string) *ORMError {
	return New(ErrorTypeQuery, message)
}

func NewTransactionError(message string) *ORMError {
	return New(ErrorTypeTransaction, message)
}

func NewTimeoutError(message string) *ORMError {
	return New(ErrorTypeTimeout, message)
}

func NewNotFoundError(message string) *ORMError {
	return New(ErrorTypeNotFound, message)
}

func NewDuplicateError(message string) *ORMError {
	return New(ErrorTypeDuplicate, message)
}

func NewValidationError(message string) *ORMError {
	return New(ErrorTypeValidation, message)
}

func NewConstraintError(message string) *ORMError {
	return New(ErrorTypeConstraint, message)
}

func NewSecurityError(message string) *ORMError {
	return New(ErrorTypeSecurity, message)
}
