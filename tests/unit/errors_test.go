package unit

import (
	"context"
	stderrors "errors"
	"strings"
	"testing"
	"time"

	"github.com/SeaSBee/go-ormx/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestORMError_Error(t *testing.T) {
	tests := []struct {
		name     string
		ormError *errors.ORMError
		expected string
	}{
		{
			name: "error without cause",
			ormError: &errors.ORMError{
				Type:    errors.ErrorTypeConnection,
				Message: "connection failed",
			},
			expected: "connection: connection failed",
		},
		{
			name: "error with cause",
			ormError: &errors.ORMError{
				Type:    errors.ErrorTypeQuery,
				Message: "query failed",
				Cause:   stderrors.New("underlying error"),
			},
			expected: "query: query failed (caused by: underlying error)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ormError.Error()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestORMError_Unwrap(t *testing.T) {
	underlyingErr := stderrors.New("underlying error")
	ormErr := &errors.ORMError{
		Type:    errors.ErrorTypeQuery,
		Message: "query failed",
		Cause:   underlyingErr,
	}

	result := ormErr.Unwrap()
	assert.Equal(t, underlyingErr, result)
}

func TestORMError_IsRetryable(t *testing.T) {
	tests := []struct {
		name           string
		ormError       *errors.ORMError
		expectedResult bool
	}{
		{
			name: "retryable error",
			ormError: &errors.ORMError{
				Type:      errors.ErrorTypeConnection,
				Message:   "connection failed",
				Retryable: true,
			},
			expectedResult: true,
		},
		{
			name: "non-retryable error",
			ormError: &errors.ORMError{
				Type:      errors.ErrorTypeValidation,
				Message:   "validation failed",
				Retryable: false,
			},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ormError.IsRetryable()
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestORMError_GetType(t *testing.T) {
	ormErr := &errors.ORMError{
		Type:    errors.ErrorTypeQuery,
		Message: "query failed",
	}

	result := ormErr.GetType()
	assert.Equal(t, errors.ErrorTypeQuery, result)
}

func TestORMError_GetSeverity(t *testing.T) {
	ormErr := &errors.ORMError{
		Type:     errors.ErrorTypeConnection,
		Message:  "connection failed",
		Severity: errors.ErrorSeverityHigh,
	}

	result := ormErr.GetSeverity()
	assert.Equal(t, errors.ErrorSeverityHigh, result)
}

func TestORMError_GetContext(t *testing.T) {
	ormErr := &errors.ORMError{
		Type:    errors.ErrorTypeQuery,
		Message: "query failed",
		Context: map[string]interface{}{
			"table": "users",
			"user":  "admin",
		},
	}

	result := ormErr.GetContext()
	assert.Equal(t, "users", result["table"])
	assert.Equal(t, "admin", result["user"])
}

func TestORMError_AddContext(t *testing.T) {
	ormErr := &errors.ORMError{
		Type:    errors.ErrorTypeQuery,
		Message: "query failed",
	}

	ormErr.AddContext("table", "users")
	ormErr.AddContext("operation", "select")

	assert.Equal(t, "users", ormErr.Context["table"])
	assert.Equal(t, "select", ormErr.Context["operation"])
}

func TestNew(t *testing.T) {
	ormErr := errors.New(errors.ErrorTypeConnection, "connection failed")

	assert.Equal(t, errors.ErrorTypeConnection, ormErr.Type)
	assert.Equal(t, "connection failed", ormErr.Message)
	assert.Equal(t, errors.ErrorSeverityHigh, ormErr.Severity) // Default severity for connection errors
	assert.False(t, ormErr.Timestamp.IsZero())
	assert.NotEmpty(t, ormErr.Stack)
}

func TestWrap(t *testing.T) {
	underlyingErr := stderrors.New("underlying error")
	ormErr := errors.Wrap(underlyingErr, errors.ErrorTypeQuery, "query failed")

	assert.Equal(t, errors.ErrorTypeQuery, ormErr.Type)
	assert.Equal(t, "query failed", ormErr.Message)
	assert.Equal(t, underlyingErr, ormErr.Cause)
}

func TestORMError_WithSeverity(t *testing.T) {
	ormErr := errors.New(errors.ErrorTypeValidation, "validation failed")
	result := ormErr.WithSeverity(errors.ErrorSeverityCritical)

	assert.Equal(t, errors.ErrorSeverityCritical, result.Severity)
	assert.Equal(t, ormErr, result) // Should return the same instance
}

func TestORMError_WithCode(t *testing.T) {
	ormErr := errors.New(errors.ErrorTypeQuery, "query failed")
	result := ormErr.WithCode("QUERY_001")

	assert.Equal(t, "QUERY_001", result.Code)
	assert.Equal(t, ormErr, result)
}

func TestORMError_WithDetails(t *testing.T) {
	ormErr := errors.New(errors.ErrorTypeValidation, "validation failed")
	result := ormErr.WithDetails("field 'email' is invalid")

	assert.Equal(t, "field 'email' is invalid", result.Details)
	assert.Equal(t, ormErr, result)
}

func TestORMError_WithOperation(t *testing.T) {
	ormErr := errors.New(errors.ErrorTypeTransaction, "transaction failed")
	result := ormErr.WithOperation("user_registration")

	assert.Equal(t, "user_registration", result.Operation)
	assert.Equal(t, ormErr, result)
}

func TestORMError_WithTable(t *testing.T) {
	ormErr := errors.New(errors.ErrorTypeQuery, "query failed")
	result := ormErr.WithTable("users")

	assert.Equal(t, "users", result.Table)
	assert.Equal(t, ormErr, result)
}

func TestORMError_WithField(t *testing.T) {
	ormErr := errors.New(errors.ErrorTypeValidation, "validation failed")
	result := ormErr.WithField("email")

	assert.Equal(t, "email", result.Field)
	assert.Equal(t, ormErr, result)
}

func TestORMError_WithValue(t *testing.T) {
	ormErr := errors.New(errors.ErrorTypeValidation, "validation failed")
	result := ormErr.WithValue("invalid@email")

	assert.Equal(t, "invalid@email", result.Value)
	assert.Equal(t, ormErr, result)
}

func TestORMError_WithQuery(t *testing.T) {
	ormErr := errors.New(errors.ErrorTypeQuery, "query failed")
	result := ormErr.WithQuery("SELECT * FROM users WHERE id = ?", 123)

	assert.Equal(t, "SELECT * FROM users WHERE id = ?", result.Query)
	assert.Equal(t, []interface{}{123}, result.Params)
	assert.Equal(t, ormErr, result)
}

func TestORMError_WithRetry(t *testing.T) {
	ormErr := errors.New(errors.ErrorTypeConnection, "connection failed")
	result := ormErr.WithRetry(true, 3, 5*time.Second)

	assert.True(t, result.Retryable)
	assert.Equal(t, 3, result.RetryCount)
	assert.Equal(t, 5*time.Second, result.RetryDelay)
	assert.Equal(t, ormErr, result)
}

func TestErrorBuilder(t *testing.T) {
	builder := errors.NewBuilder(errors.ErrorTypeQuery, "query failed")
	ormErr := builder.
		Severity(errors.ErrorSeverityHigh).
		Code("QUERY_001").
		Details("syntax error").
		Operation("user_search").
		Table("users").
		Field("email").
		Value("invalid@email").
		Query("SELECT * FROM users WHERE email = ?", "invalid@email").
		Retry(true, 3, 5*time.Second).
		Context("user_id", "123").
		Build()

	assert.Equal(t, errors.ErrorTypeQuery, ormErr.Type)
	assert.Equal(t, "query failed", ormErr.Message)
	assert.Equal(t, errors.ErrorSeverityHigh, ormErr.Severity)
	assert.Equal(t, "QUERY_001", ormErr.Code)
	assert.Equal(t, "syntax error", ormErr.Details)
	assert.Equal(t, "user_search", ormErr.Operation)
	assert.Equal(t, "users", ormErr.Table)
	assert.Equal(t, "email", ormErr.Field)
	assert.Equal(t, "invalid@email", ormErr.Value)
	assert.Equal(t, "SELECT * FROM users WHERE email = ?", ormErr.Query)
	assert.Equal(t, []interface{}{"invalid@email"}, ormErr.Params)
	assert.True(t, ormErr.Retryable)
	assert.Equal(t, 3, ormErr.RetryCount)
	assert.Equal(t, 5*time.Second, ormErr.RetryDelay)
	assert.Equal(t, "123", ormErr.Context["user_id"])
}

func TestErrorClassifier_ClassifyError(t *testing.T) {
	classifier := errors.NewErrorClassifier()

	tests := []struct {
		name              string
		err               error
		operation         string
		expectedType      errors.ErrorType
		expectedRetryable bool
	}{
		{
			name:              "connection timeout error",
			err:               stderrors.New("connection timeout"),
			operation:         "connect",
			expectedType:      errors.ErrorTypeTimeout,
			expectedRetryable: true,
		},
		{
			name:              "duplicate key error",
			err:               stderrors.New("duplicate key"),
			operation:         "insert",
			expectedType:      errors.ErrorTypeDuplicate,
			expectedRetryable: false,
		},
		{
			name:              "unknown error",
			err:               stderrors.New("unknown error"),
			operation:         "unknown",
			expectedType:      errors.ErrorTypeUnknown,
			expectedRetryable: false,
		},
		{
			name:              "nil error",
			err:               nil,
			operation:         "test",
			expectedType:      "",
			expectedRetryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.ClassifyError(tt.err, tt.operation)

			if tt.err == nil {
				assert.Nil(t, result)
				return
			}

			assert.Equal(t, tt.expectedType, result.Type)
			assert.Equal(t, tt.expectedRetryable, result.Retryable)
			assert.Equal(t, tt.operation, result.Operation)
			assert.Equal(t, tt.err, result.Cause)
		})
	}
}

func TestErrorHandler_HandleError(t *testing.T) {
	handler := errors.NewErrorHandler(3, 1*time.Second)

	tests := []struct {
		name               string
		err                error
		operation          string
		expectedRetryable  bool
		expectedRetryCount int
	}{
		{
			name:               "retryable error",
			err:                stderrors.New("connection timeout"),
			operation:          "connect",
			expectedRetryable:  true,
			expectedRetryCount: 1,
		},
		{
			name:               "non-retryable error",
			err:                stderrors.New("duplicate key"),
			operation:          "insert",
			expectedRetryable:  false,
			expectedRetryCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.HandleError(tt.err, tt.operation)

			assert.Equal(t, tt.expectedRetryable, result.Retryable)
			assert.Equal(t, tt.expectedRetryCount, result.RetryCount)
			assert.Equal(t, tt.operation, result.Operation)
			assert.Equal(t, tt.err, result.Cause)
		})
	}
}

func TestErrorHandler_RetryWithContext(t *testing.T) {
	handler := errors.NewErrorHandler(2, 10*time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	attemptCount := 0
	operation := func() error {
		attemptCount++
		if attemptCount < 3 {
			return stderrors.New("temporary failure")
		}
		return nil
	}

	err := handler.RetryWithContext(ctx, operation, "test_operation")
	assert.NoError(t, err)
	assert.Equal(t, 3, attemptCount)
}

func TestErrorHandler_RetryWithContext_ContextCancelled(t *testing.T) {
	handler := errors.NewErrorHandler(5, 100*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context immediately
	cancel()

	operation := func() error {
		return stderrors.New("temporary failure")
	}

	err := handler.RetryWithContext(ctx, operation, "test_operation")
	assert.Equal(t, context.Canceled, err)
}

func TestErrorHandler_RetryWithContext_MaxRetriesExceeded(t *testing.T) {
	handler := errors.NewErrorHandler(2, 10*time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	operation := func() error {
		return stderrors.New("persistent failure")
	}

	err := handler.RetryWithContext(ctx, operation, "test_operation")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "persistent failure")
}

func TestCommonErrorConstructors(t *testing.T) {
	tests := []struct {
		name         string
		constructor  func(string) *errors.ORMError
		expectedType errors.ErrorType
	}{
		{
			name:         "NewConnectionError",
			constructor:  errors.NewConnectionError,
			expectedType: errors.ErrorTypeConnection,
		},
		{
			name:         "NewQueryError",
			constructor:  errors.NewQueryError,
			expectedType: errors.ErrorTypeQuery,
		},
		{
			name:         "NewTransactionError",
			constructor:  errors.NewTransactionError,
			expectedType: errors.ErrorTypeTransaction,
		},
		{
			name:         "NewTimeoutError",
			constructor:  errors.NewTimeoutError,
			expectedType: errors.ErrorTypeTimeout,
		},
		{
			name:         "NewNotFoundError",
			constructor:  errors.NewNotFoundError,
			expectedType: errors.ErrorTypeNotFound,
		},
		{
			name:         "NewDuplicateError",
			constructor:  errors.NewDuplicateError,
			expectedType: errors.ErrorTypeDuplicate,
		},
		{
			name:         "NewValidationError",
			constructor:  errors.NewValidationError,
			expectedType: errors.ErrorTypeValidation,
		},
		{
			name:         "NewConstraintError",
			constructor:  errors.NewConstraintError,
			expectedType: errors.ErrorTypeConstraint,
		},
		{
			name:         "NewSecurityError",
			constructor:  errors.NewSecurityError,
			expectedType: errors.ErrorTypeSecurity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := "test error message"
			result := tt.constructor(message)

			assert.Equal(t, tt.expectedType, result.Type)
			assert.Equal(t, message, result.Message)
			assert.NotZero(t, result.Timestamp)
		})
	}
}

func TestErrorType_Constants(t *testing.T) {
	// Test that all error type constants are defined
	assert.NotEmpty(t, errors.ErrorTypeConnection)
	assert.NotEmpty(t, errors.ErrorTypeQuery)
	assert.NotEmpty(t, errors.ErrorTypeTransaction)
	assert.NotEmpty(t, errors.ErrorTypeTimeout)
	assert.NotEmpty(t, errors.ErrorTypeDeadlock)
	assert.NotEmpty(t, errors.ErrorTypeConstraint)
	assert.NotEmpty(t, errors.ErrorTypeNotFound)
	assert.NotEmpty(t, errors.ErrorTypeDuplicate)
	assert.NotEmpty(t, errors.ErrorTypeValidation)
	assert.NotEmpty(t, errors.ErrorTypeMigration)
	assert.NotEmpty(t, errors.ErrorTypeReplication)
	assert.NotEmpty(t, errors.ErrorTypeModel)
	assert.NotEmpty(t, errors.ErrorTypeRelationship)
	assert.NotEmpty(t, errors.ErrorTypeAssociation)
	assert.NotEmpty(t, errors.ErrorTypeHook)
	assert.NotEmpty(t, errors.ErrorTypeCallback)
	assert.NotEmpty(t, errors.ErrorTypeConfig)
	assert.NotEmpty(t, errors.ErrorTypeInitialization)
	assert.NotEmpty(t, errors.ErrorTypeCache)
	assert.NotEmpty(t, errors.ErrorTypeCacheMiss)
	assert.NotEmpty(t, errors.ErrorTypeCacheInvalid)
	assert.NotEmpty(t, errors.ErrorTypeSecurity)
	assert.NotEmpty(t, errors.ErrorTypeSQLInjection)
	assert.NotEmpty(t, errors.ErrorTypeAccessDenied)
	assert.NotEmpty(t, errors.ErrorTypeSystem)
	assert.NotEmpty(t, errors.ErrorTypeResource)
	assert.NotEmpty(t, errors.ErrorTypeNetwork)
	assert.NotEmpty(t, errors.ErrorTypeUnknown)
}

func TestErrorSeverity_Constants(t *testing.T) {
	// Test that all error severity constants are defined
	assert.NotEmpty(t, errors.ErrorSeverityLow)
	assert.NotEmpty(t, errors.ErrorSeverityMedium)
	assert.NotEmpty(t, errors.ErrorSeverityHigh)
	assert.NotEmpty(t, errors.ErrorSeverityCritical)
}

func TestGetDefaultSeverity(t *testing.T) {
	// This tests the internal function indirectly through the New function
	// Connection errors should have high severity
	connErr := errors.New(errors.ErrorTypeConnection, "connection failed")
	assert.Equal(t, errors.ErrorSeverityHigh, connErr.Severity)

	// Validation errors should have medium severity
	valErr := errors.New(errors.ErrorTypeValidation, "validation failed")
	assert.Equal(t, errors.ErrorSeverityMedium, valErr.Severity)

	// Security errors should have critical severity
	secErr := errors.New(errors.ErrorTypeSecurity, "security violation")
	assert.Equal(t, errors.ErrorSeverityCritical, secErr.Severity)
}

func TestGetStackTrace(t *testing.T) {
	// This tests the internal function indirectly through the New function
	ormErr := errors.New(errors.ErrorTypeQuery, "query failed")
	assert.NotEmpty(t, ormErr.Stack)
	assert.Contains(t, ormErr.Stack, "stack trace placeholder")
}

// Add missing test scenarios
func TestErrorClassifier_AdditionalPatterns(t *testing.T) {
	classifier := errors.NewErrorClassifier()

	tests := []struct {
		name              string
		err               error
		operation         string
		expectedType      errors.ErrorType
		expectedRetryable bool
	}{
		{
			name:              "connection refused error",
			err:               stderrors.New("connection refused"),
			operation:         "connect",
			expectedType:      errors.ErrorTypeConnection,
			expectedRetryable: true,
		},
		{
			name:              "deadlock error",
			err:               stderrors.New("deadlock detected"),
			operation:         "transaction",
			expectedType:      errors.ErrorTypeDeadlock,
			expectedRetryable: true,
		},
		{
			name:              "lock wait timeout error",
			err:               stderrors.New("lock wait timeout"),
			operation:         "update",
			expectedType:      errors.ErrorTypeTimeout,
			expectedRetryable: true,
		},
		{
			name:              "unique constraint error",
			err:               stderrors.New("unique constraint violation"),
			operation:         "insert",
			expectedType:      errors.ErrorTypeConstraint,
			expectedRetryable: false,
		},
		{
			name:              "foreign key constraint error",
			err:               stderrors.New("foreign key constraint fails"),
			operation:         "delete",
			expectedType:      errors.ErrorTypeConstraint,
			expectedRetryable: false,
		},
		{
			name:              "table not found error",
			err:               stderrors.New("table 'nonexistent' doesn't exist"),
			operation:         "query",
			expectedType:      errors.ErrorTypeNotFound,
			expectedRetryable: false,
		},
		{
			name:              "column not found error",
			err:               stderrors.New("unknown column 'nonexistent'"),
			operation:         "query",
			expectedType:      errors.ErrorTypeNotFound,
			expectedRetryable: false,
		},
		{
			name:              "invalid syntax error",
			err:               stderrors.New("syntax error near 'FROM'"),
			operation:         "query",
			expectedType:      errors.ErrorTypeQuery,
			expectedRetryable: false,
		},
		{
			name:              "sql injection attempt",
			err:               stderrors.New("sql injection detected"),
			operation:         "query",
			expectedType:      errors.ErrorTypeSQLInjection,
			expectedRetryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.ClassifyError(tt.err, tt.operation)

			assert.Equal(t, tt.expectedType, result.Type)
			assert.Equal(t, tt.expectedRetryable, result.Retryable)
			assert.Equal(t, tt.operation, result.Operation)
			assert.Equal(t, tt.err, result.Cause)
		})
	}
}

func TestErrorHandler_EdgeCases(t *testing.T) {
	// Test with zero max retries
	handler := errors.NewErrorHandler(0, 1*time.Second)
	result := handler.HandleError(stderrors.New("test error"), "test")
	assert.Equal(t, 0, result.RetryCount)
	assert.False(t, result.Retryable)

	// Test with negative max retries
	handler = errors.NewErrorHandler(-1, 1*time.Second)
	result = handler.HandleError(stderrors.New("test error"), "test")
	assert.Equal(t, 0, result.RetryCount)
	assert.False(t, result.Retryable)

	// Test with zero delay
	handler = errors.NewErrorHandler(3, 0)
	result = handler.HandleError(stderrors.New("test error"), "test")
	assert.Equal(t, 0, result.RetryCount)
	assert.False(t, result.Retryable)

	// Test with negative delay
	handler = errors.NewErrorHandler(3, -1*time.Second)
	result = handler.HandleError(stderrors.New("test error"), "test")
	assert.Equal(t, 0, result.RetryCount)
	assert.False(t, result.Retryable)
}

func TestErrorHandler_RetryWithContext_EdgeCases(t *testing.T) {
	// Test with nil operation
	handler := errors.NewErrorHandler(3, 10*time.Millisecond)
	ctx := context.Background()

	assert.Panics(t, func() {
		handler.RetryWithContext(ctx, nil, "test_operation")
	})

	// Test with empty operation name
	operation := func() error { return nil }
	err := handler.RetryWithContext(ctx, operation, "")
	assert.NoError(t, err)

	// Test with very short context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	slowOperation := func() error {
		time.Sleep(1 * time.Millisecond)
		return stderrors.New("slow operation")
	}

	err = handler.RetryWithContext(ctx, slowOperation, "test_operation")
	// With such a short timeout, we expect the context deadline exceeded error
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestErrorBuilder_EdgeCases(t *testing.T) {
	// Test with empty message
	builder := errors.NewBuilder(errors.ErrorTypeQuery, "")
	ormErr := builder.Build()

	assert.Equal(t, "", ormErr.Message)
	assert.Equal(t, errors.ErrorTypeQuery, ormErr.Type)

	// Test with nil context values
	builder = errors.NewBuilder(errors.ErrorTypeQuery, "test error")
	ormErr = builder.Context("key", nil).Build()

	assert.Equal(t, nil, ormErr.Context["key"])

	// Test with empty context key
	builder = errors.NewBuilder(errors.ErrorTypeQuery, "test error")
	ormErr = builder.Context("", "value").Build()

	assert.Equal(t, "value", ormErr.Context[""])

	// Test with very long message
	longMessage := strings.Repeat("a", 10000)
	builder = errors.NewBuilder(errors.ErrorTypeQuery, longMessage)
	ormErr = builder.Build()

	assert.Equal(t, longMessage, ormErr.Message)

	// Test with special characters in message
	specialMessage := "error with special chars: !@#$%^&*()_+-=[]{}|;':\",./<>?"
	builder = errors.NewBuilder(errors.ErrorTypeQuery, specialMessage)
	ormErr = builder.Build()

	assert.Equal(t, specialMessage, ormErr.Message)
}

func TestORMError_EdgeCases(t *testing.T) {
	// Test with empty error type
	ormErr := &errors.ORMError{
		Type:    "",
		Message: "test error",
	}

	assert.Equal(t, errors.ErrorType(""), ormErr.GetType())
	assert.Equal(t, "test error", ormErr.Error())

	// Test with empty message
	ormErr = &errors.ORMError{
		Type:    errors.ErrorTypeQuery,
		Message: "",
	}

	assert.Equal(t, errors.ErrorTypeQuery, ormErr.GetType())
	assert.Equal(t, "query: ", ormErr.Error())

	// Test with nil context
	ormErr = &errors.ORMError{
		Type:    errors.ErrorTypeQuery,
		Message: "test error",
		Context: nil,
	}

	// AddContext should initialize context if nil
	ormErr.AddContext("key", "value")
	assert.NotNil(t, ormErr.Context)
	assert.Equal(t, "value", ormErr.Context["key"])

	// Test with nil cause
	ormErr = &errors.ORMError{
		Type:    errors.ErrorTypeQuery,
		Message: "test error",
		Cause:   nil,
	}

	assert.Nil(t, ormErr.Unwrap())

	// Test with zero timestamp
	ormErr = &errors.ORMError{
		Type:      errors.ErrorTypeQuery,
		Message:   "test error",
		Timestamp: time.Time{},
	}

	assert.True(t, ormErr.Timestamp.IsZero())
}

func TestErrorClassification_ComplexMessages(t *testing.T) {
	classifier := errors.NewErrorClassifier()

	tests := []struct {
		name              string
		err               error
		expectedType      errors.ErrorType
		expectedRetryable bool
	}{
		{
			name:              "complex connection error",
			err:               stderrors.New("failed to connect to database: connection refused: dial tcp 127.0.0.1:5432: connect: connection refused"),
			expectedType:      errors.ErrorTypeConnection,
			expectedRetryable: true,
		},
		{
			name:              "complex timeout error",
			err:               stderrors.New("query execution failed: context deadline exceeded: operation timed out after 30 seconds"),
			expectedType:      errors.ErrorTypeTimeout,
			expectedRetryable: true,
		},
		{
			name:              "complex constraint error",
			err:               stderrors.New("insert failed: duplicate entry '123' for key 'PRIMARY': unique constraint violation"),
			expectedType:      errors.ErrorTypeConstraint,
			expectedRetryable: false,
		},
		{
			name:              "complex validation error",
			err:               stderrors.New("validation failed: field 'email' is invalid: email format is incorrect"),
			expectedType:      errors.ErrorTypeValidation,
			expectedRetryable: false,
		},
		{
			name:              "complex not found error",
			err:               stderrors.New("record not found: no rows returned from query: SELECT * FROM users WHERE id = 999"),
			expectedType:      errors.ErrorTypeNotFound,
			expectedRetryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.ClassifyError(tt.err, "test_operation")

			assert.Equal(t, tt.expectedType, result.Type)
			assert.Equal(t, tt.expectedRetryable, result.Retryable)
		})
	}
}

func TestErrorSeverity_EdgeCases(t *testing.T) {
	// Test with unknown severity
	unknownSeverity := errors.ErrorSeverity("unknown")
	assert.Equal(t, "unknown", string(unknownSeverity))

	// Test with empty severity
	emptySeverity := errors.ErrorSeverity("")
	assert.Equal(t, "", string(emptySeverity))

	// Test severity comparison
	assert.True(t, errors.ErrorSeverityLow < errors.ErrorSeverityMedium)
	assert.True(t, errors.ErrorSeverityMedium < errors.ErrorSeverityHigh)
	assert.True(t, errors.ErrorSeverityHigh < errors.ErrorSeverityCritical)
}

func TestErrorType_EdgeCases(t *testing.T) {
	// Test with unknown error type
	unknownType := errors.ErrorType("unknown_type")
	assert.Equal(t, "unknown_type", string(unknownType))

	// Test with empty error type
	emptyType := errors.ErrorType("")
	assert.Equal(t, "", string(emptyType))

	// Test error type comparison
	assert.NotEqual(t, errors.ErrorTypeConnection, errors.ErrorTypeQuery)
	assert.NotEqual(t, errors.ErrorTypeQuery, errors.ErrorTypeTransaction)
	assert.NotEqual(t, errors.ErrorTypeTransaction, errors.ErrorTypeTimeout)
}

func TestErrorContext_EdgeCases(t *testing.T) {
	ormErr := &errors.ORMError{
		Type:    errors.ErrorTypeQuery,
		Message: "test error",
	}

	// Test with very long context key
	longKey := strings.Repeat("a", 1000)
	ormErr.AddContext(longKey, "value")
	assert.Equal(t, "value", ormErr.Context[longKey])

	// Test with very long context value
	longValue := strings.Repeat("b", 1000)
	ormErr.AddContext("key", longValue)
	assert.Equal(t, longValue, ormErr.Context["key"])

	// Test with special characters in context
	ormErr.AddContext("special_key!@#$%", "special_value!@#$%")
	assert.Equal(t, "special_value!@#$%", ormErr.Context["special_key!@#$%"])

	// Test with nil context value
	ormErr.AddContext("nil_key", nil)
	assert.Nil(t, ormErr.Context["nil_key"])

	// Test with empty context key and value
	ormErr.AddContext("", "")
	assert.Equal(t, "", ormErr.Context[""])
}

func TestErrorRetry_EdgeCases(t *testing.T) {
	ormErr := &errors.ORMError{
		Type:    errors.ErrorTypeConnection,
		Message: "test error",
	}

	// Test with zero retry count
	result := ormErr.WithRetry(true, 0, 5*time.Second)
	assert.Equal(t, 0, result.RetryCount)
	assert.True(t, result.Retryable)

	// Test with negative retry count
	result = ormErr.WithRetry(true, -1, 5*time.Second)
	assert.Equal(t, -1, result.RetryCount)
	assert.True(t, result.Retryable)

	// Test with zero retry delay
	result = ormErr.WithRetry(true, 3, 0)
	assert.Equal(t, time.Duration(0), result.RetryDelay)
	assert.True(t, result.Retryable)

	// Test with negative retry delay
	result = ormErr.WithRetry(true, 3, -1*time.Second)
	assert.Equal(t, -1*time.Second, result.RetryDelay)
	assert.True(t, result.Retryable)

	// Test with very large retry count
	result = ormErr.WithRetry(true, 999999, 5*time.Second)
	assert.Equal(t, 999999, result.RetryCount)
	assert.True(t, result.Retryable)

	// Test with very long retry delay
	result = ormErr.WithRetry(true, 3, 24*365*time.Hour) // 1 year
	assert.Equal(t, 24*365*time.Hour, result.RetryDelay)
	assert.True(t, result.Retryable)
}
