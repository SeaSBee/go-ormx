package errors_test

import (
	"database/sql"
	stderrors "errors"
	"strings"
	"testing"

	"go-ormx/ormx/errors"

	"gorm.io/gorm"
)

func TestErrorCode_Constants(t *testing.T) {
	tests := []struct {
		code     errors.ErrorCode
		expected string
	}{
		{errors.ErrCodeConnectionFailed, "DB_CONNECTION_FAILED"},
		{errors.ErrCodeRecordNotFound, "DB_RECORD_NOT_FOUND"},
		{errors.ErrCodeInvalidData, "DB_INVALID_DATA"},
		{errors.ErrCodeTransactionFailed, "DB_TRANSACTION_FAILED"},
		{errors.ErrCodeConstraintViolation, "DB_CONSTRAINT_VIOLATION"},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			result := string(tt.code)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestNewDBError(t *testing.T) {
	tests := []struct {
		name    string
		code    errors.ErrorCode
		message string
		cause   error
	}{
		{
			name:    "simple_error",
			code:    errors.ErrCodeInvalidData,
			message: "validation failed",
			cause:   nil,
		},
		{
			name:    "error_with_cause",
			code:    errors.ErrCodeConnectionFailed,
			message: "database connection failed",
			cause:   stderrors.New("network timeout"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.NewDBError(tt.code, tt.message, tt.cause)

			if err == nil {
				t.Fatal("Expected non-nil error")
			}

			if err.Error() == "" {
				t.Error("Expected non-empty error message")
			}

			if !strings.Contains(err.Error(), tt.message) {
				t.Errorf("Expected error message to contain '%s'", tt.message)
			}
		})
	}
}

func TestWrapGormError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		operation   string
		expectedNil bool
	}{
		{
			name:        "go-ormx_record_not_found",
			err:         gorm.ErrRecordNotFound,
			operation:   "find_user",
			expectedNil: false,
		},
		{
			name:        "go-ormx_invalid_transaction",
			err:         gorm.ErrInvalidTransaction,
			operation:   "transaction",
			expectedNil: false,
		},
		{
			name:        "sql_no_rows",
			err:         sql.ErrNoRows,
			operation:   "query",
			expectedNil: false,
		},
		{
			name:        "nil_error",
			err:         nil,
			operation:   "test",
			expectedNil: true,
		},
		{
			name:        "unknown_error",
			err:         stderrors.New("unknown database error"),
			operation:   "operation",
			expectedNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := errors.WrapGormError(tt.err, tt.operation)

			if tt.expectedNil {
				if wrapped != nil {
					t.Error("Expected nil error")
				}
				return
			}

			if wrapped == nil {
				t.Fatal("Expected non-nil wrapped error")
			}

			if wrapped.Error() == "" {
				t.Error("Expected non-empty error message")
			}
		})
	}
}

func TestWrapError(t *testing.T) {
	originalErr := stderrors.New("original error")

	tests := []struct {
		name    string
		err     error
		code    errors.ErrorCode
		message string
	}{
		{
			name:    "wrap_standard_error",
			err:     originalErr,
			code:    errors.ErrCodeConnectionFailed,
			message: "connection failed",
		},
		{
			name:    "wrap_nil_error",
			err:     nil,
			code:    errors.ErrCodeUnknown,
			message: "unknown error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := errors.WrapError(tt.err, tt.code, tt.message)

			if tt.err == nil {
				if wrapped != nil {
					t.Error("Wrapping nil error should return nil")
				}
				return
			}

			if wrapped == nil {
				t.Fatal("Expected non-nil wrapped error")
			}

			if wrapped.Error() == "" {
				t.Error("Expected non-empty error message")
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "connection_failed",
			err:      errors.NewDBError(errors.ErrCodeConnectionFailed, "connection failed", nil),
			expected: true,
		},
		{
			name:     "invalid_data",
			err:      errors.NewDBError(errors.ErrCodeInvalidData, "validation failed", nil),
			expected: false,
		},
		{
			name:     "standard_error",
			err:      stderrors.New("standard error"),
			expected: false,
		},
		{
			name:     "nil_error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := errors.IsRetryable(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %t, got %t", tt.expected, result)
			}
		})
	}
}

func TestGetUserMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "db_error_with_user_message",
			err:      errors.NewDBError(errors.ErrCodeRecordNotFound, "user not found", nil).WithUserMessage("User not found"),
			expected: "User not found",
		},
		{
			name:     "db_error_without_user_message",
			err:      errors.NewDBError(errors.ErrCodeConnectionFailed, "connection failed", nil),
			expected: "An error occurred while processing your request",
		},
		{
			name:     "standard_error",
			err:      stderrors.New("standard error"),
			expected: "An error occurred while processing your request",
		},
		{
			name:     "nil_error",
			err:      nil,
			expected: "An error occurred while processing your request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := errors.GetUserMessage(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestErrorCollector(t *testing.T) {
	t.Run("basic_functionality", func(t *testing.T) {
		collector := errors.NewErrorCollector(5)

		if collector.HasErrors() {
			t.Error("New collector should not have errors")
		}

		if collector.Count() != 0 {
			t.Error("New collector should have 0 errors")
		}

		// Add errors
		collector.Add(stderrors.New("error 1"))
		collector.Add(stderrors.New("error 2"))

		if !collector.HasErrors() {
			t.Error("Collector should have errors after adding")
		}

		if collector.Count() != 2 {
			t.Error("Collector should have 2 errors")
		}

		first := collector.First()
		if first == nil {
			t.Error("First error should not be nil")
		}

		if first.Error() != "error 1" {
			t.Error("First error should be 'error 1'")
		}
	})

	t.Run("max_count_limit", func(t *testing.T) {
		collector := errors.NewErrorCollector(2)

		collector.Add(stderrors.New("error 1"))
		collector.Add(stderrors.New("error 2"))
		collector.Add(stderrors.New("error 3")) // Should be ignored

		if collector.Count() != 2 {
			t.Error("Collector should not exceed max count")
		}
	})
}

// Benchmark tests
func BenchmarkNewDBError(b *testing.B) {
	cause := stderrors.New("cause error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = errors.NewDBError(errors.ErrCodeInvalidData, "test error", cause)
	}
}

func BenchmarkWrapGormError(b *testing.B) {
	gormErr := gorm.ErrRecordNotFound

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = errors.WrapGormError(gormErr, "test_operation")
	}
}

func BenchmarkIsRetryable(b *testing.B) {
	err := errors.NewDBError(errors.ErrCodeConnectionFailed, "connection failed", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = errors.IsRetryable(err)
	}
}

func BenchmarkGetUserMessage(b *testing.B) {
	err := errors.NewDBError(errors.ErrCodeRecordNotFound, "user not found", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = errors.GetUserMessage(err)
	}
}
