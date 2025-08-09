package security_test

import (
	"context"
	"testing"

	"go-ormx/ormx/security"
)

// Test struct for validation
type TestStruct struct {
	Email    string   `validate:"required,email"`
	Age      string   `validate:"required,min=1,max=3"`
	Username string   `validate:"required,min=3,max=50"`
	Password string   `validate:"required,min=8"`
	Website  string   `validate:"omitempty"`
	Tags     []string `validate:"required"`
}

func TestValidateStruct(t *testing.T) {
	tests := []struct {
		name        string
		input       TestStruct
		expectValid bool
	}{
		{
			name: "valid_struct",
			input: TestStruct{
				Email:    "test@example.com",
				Age:      "25",
				Username: "testuser123",
				Password: "password123",
				Website:  "https://example.com",
				Tags:     []string{"tag1", "tag2"},
			},
			expectValid: true,
		},
		{
			name: "invalid_email",
			input: TestStruct{
				Email:    "invalid-email",
				Age:      "25",
				Username: "testuser123",
				Password: "password123",
			},
			expectValid: false,
		},
		{
			name: "missing_required_field",
			input: TestStruct{
				Age:      "25",
				Username: "testuser123",
				Password: "password123",
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result := security.ValidateStruct(ctx, tt.input)

			if result.Valid != tt.expectValid {
				t.Errorf("Expected valid=%t, got valid=%t", tt.expectValid, result.Valid)
			}

			if tt.expectValid && len(result.Errors) > 0 {
				t.Errorf("Expected no errors for valid input, got: %v", result.Errors)
			}

			if !tt.expectValid && len(result.Errors) == 0 {
				t.Error("Expected validation errors for invalid input")
			}
		})
	}
}

func TestSecurityContext(t *testing.T) {
	t.Run("new_security_context", func(t *testing.T) {
		userID := "test-user-id"
		tenantID := "test-tenant-id"
		ipAddress := "192.168.1.1"
		userAgent := "test-agent/1.0"

		secCtx := security.NewSecurityContext(userID, tenantID, ipAddress, userAgent)

		if secCtx.UserID != userID {
			t.Errorf("Expected UserID %s, got %s", userID, secCtx.UserID)
		}
		if secCtx.TenantID != tenantID {
			t.Errorf("Expected TenantID %s, got %s", tenantID, secCtx.TenantID)
		}
		if secCtx.IPAddress != ipAddress {
			t.Errorf("Expected IPAddress %s, got %s", ipAddress, secCtx.IPAddress)
		}
		if secCtx.UserAgent != userAgent {
			t.Errorf("Expected UserAgent %s, got %s", userAgent, secCtx.UserAgent)
		}
		if secCtx.Timestamp.IsZero() {
			t.Error("Timestamp should be set")
		}
	})

	t.Run("context_operations", func(t *testing.T) {
		secCtx := security.NewSecurityContext("user1", "tenant1", "127.0.0.1", "test")
		ctx := context.Background()

		// Add to context
		ctxWithSec := secCtx.ToContext(ctx)

		// Retrieve from context
		retrieved := security.FromContext(ctxWithSec)

		if retrieved == nil {
			t.Fatal("Expected security context from context")
		}

		if retrieved.UserID != secCtx.UserID {
			t.Errorf("Expected UserID %s, got %s", secCtx.UserID, retrieved.UserID)
		}
	})

	t.Run("context_without_security", func(t *testing.T) {
		ctx := context.Background()
		retrieved := security.FromContext(ctx)

		if retrieved != nil {
			t.Error("Expected nil security context from empty context")
		}
	})
}

// Benchmark tests
func BenchmarkValidateStruct(b *testing.B) {
	testStruct := TestStruct{
		Email:    "test@example.com",
		Age:      "25",
		Username: "testuser",
		Password: "password123",
		Website:  "https://example.com",
		Tags:     []string{"tag1", "tag2"},
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = security.ValidateStruct(ctx, testStruct)
	}
}
