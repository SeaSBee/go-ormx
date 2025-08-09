// Package security provides security utilities for database operations including
// input validation, SQL injection prevention, and data sanitization.
package security

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
}

// ValidationResult holds the result of validation
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
}

// Validator interface for input validation
type Validator interface {
	Validate(ctx context.Context, value interface{}) ValidationResult
}

// SQLInjectionPattern holds patterns that might indicate SQL injection attempts
var SQLInjectionPatterns = []string{
	`(?i)(union\s+select)`,
	`(?i)(select\s+.*\s+from)`,
	`(?i)(insert\s+into)`,
	`(?i)(update\s+.*\s+set)`,
	`(?i)(delete\s+from)`,
	`(?i)(drop\s+(table|database|schema))`,
	`(?i)(alter\s+table)`,
	`(?i)(create\s+(table|database|schema))`,
	`(?i)(exec\s*\()`,
	`(?i)(execute\s*\()`,
	`(?i)(sp_executesql)`,
	`(?i)(xp_cmdshell)`,
	`(?i)((and|or)\s+1\s*=\s*1)`,
	`(?i)((and|or)\s+1\s*=\s*0)`,
	`(?i)(;\s*(select|insert|update|delete|drop|alter|create))`,
	`(?i)(--\s*$)`,
	`(?i)(/\*.*\*/)`,
}

// SensitiveDataPatterns holds patterns for sensitive data that should be masked
var SensitiveDataPatterns = map[string]*regexp.Regexp{
	"password":    regexp.MustCompile(`(?i)(password|passwd|pwd)`),
	"token":       regexp.MustCompile(`(?i)(token|access_token|refresh_token|api_key)`),
	"secret":      regexp.MustCompile(`(?i)(secret|private_key|secret_key)`),
	"email":       regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`),
	"phone":       regexp.MustCompile(`(\+?1)?[-.\s]?\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4}`),
	"ssn":         regexp.MustCompile(`\b\d{3}-?\d{2}-?\d{4}\b`),
	"credit_card": regexp.MustCompile(`\b(?:\d{4}[-\s]?){3}\d{4}\b`),
}

// InputSanitizer provides methods for sanitizing user input
type InputSanitizer struct {
	maxStringLength int
	allowedChars    map[rune]bool
}

// NewInputSanitizer creates a new input sanitizer
func NewInputSanitizer(maxStringLength int) *InputSanitizer {
	return &InputSanitizer{
		maxStringLength: maxStringLength,
		allowedChars:    make(map[rune]bool),
	}
}

// SanitizeString sanitizes a string input
func (is *InputSanitizer) SanitizeString(input string) (string, error) {
	if input == "" {
		return input, nil
	}

	// Check for SQL injection patterns
	if err := is.checkSQLInjection(input); err != nil {
		return "", err
	}

	// Validate UTF-8
	if !utf8.ValidString(input) {
		return "", ValidationError{
			Field:   "input",
			Message: "invalid UTF-8 encoding",
			Code:    "INVALID_ENCODING",
		}
	}

	// Trim whitespace
	sanitized := strings.TrimSpace(input)

	// Check length
	if is.maxStringLength > 0 && len(sanitized) > is.maxStringLength {
		return "", ValidationError{
			Field:   "input",
			Message: fmt.Sprintf("string too long, maximum %d characters", is.maxStringLength),
			Code:    "STRING_TOO_LONG",
		}
	}

	// Remove control characters except newline and tab
	sanitized = is.removeControlChars(sanitized)

	// Normalize whitespace
	sanitized = is.normalizeWhitespace(sanitized)

	return sanitized, nil
}

// checkSQLInjection checks for potential SQL injection patterns
func (is *InputSanitizer) checkSQLInjection(input string) error {
	for _, pattern := range SQLInjectionPatterns {
		matched, err := regexp.MatchString(pattern, input)
		if err != nil {
			continue // Skip malformed patterns
		}
		if matched {
			return ValidationError{
				Field:   "input",
				Message: "potential SQL injection detected",
				Code:    "SQL_INJECTION_DETECTED",
			}
		}
	}
	return nil
}

// removeControlChars removes control characters except newline and tab
func (is *InputSanitizer) removeControlChars(input string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' && r != '\r' {
			return -1 // Remove the character
		}
		return r
	}, input)
}

// normalizeWhitespace normalizes whitespace characters
func (is *InputSanitizer) normalizeWhitespace(input string) string {
	// Replace multiple consecutive whitespace with single space
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(input, " ")
}

// UUIDValidator validates UUID strings
type UUIDValidator struct{}

// Validate validates a UUID
func (uv *UUIDValidator) Validate(ctx context.Context, value interface{}) ValidationResult {
	result := ValidationResult{Valid: true}

	str, ok := value.(string)
	if !ok {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "uuid",
			Message: "value must be a string",
			Code:    "INVALID_TYPE",
		})
		return result
	}

	if str == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "uuid",
			Message: "UUID cannot be empty",
			Code:    "EMPTY_VALUE",
		})
		return result
	}

	if _, err := uuid.Parse(str); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "uuid",
			Message: "invalid UUID format",
			Code:    "INVALID_UUID",
		})
	}

	return result
}

// EmailValidator validates email addresses
type EmailValidator struct{}

// Validate validates an email address
func (ev *EmailValidator) Validate(ctx context.Context, value interface{}) ValidationResult {
	result := ValidationResult{Valid: true}

	str, ok := value.(string)
	if !ok {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "email",
			Message: "value must be a string",
			Code:    "INVALID_TYPE",
		})
		return result
	}

	if str == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "email",
			Message: "email cannot be empty",
			Code:    "EMPTY_VALUE",
		})
		return result
	}

	// Basic email validation regex
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(str) {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "email",
			Message: "invalid email format",
			Code:    "INVALID_EMAIL",
		})
	}

	return result
}

// LengthValidator validates string length
type LengthValidator struct {
	MinLength int
	MaxLength int
}

// Validate validates string length
func (lv *LengthValidator) Validate(ctx context.Context, value interface{}) ValidationResult {
	result := ValidationResult{Valid: true}

	str, ok := value.(string)
	if !ok {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "string",
			Message: "value must be a string",
			Code:    "INVALID_TYPE",
		})
		return result
	}

	length := len(str)

	if lv.MinLength > 0 && length < lv.MinLength {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "string",
			Message: fmt.Sprintf("minimum length is %d characters", lv.MinLength),
			Code:    "STRING_TOO_SHORT",
		})
	}

	if lv.MaxLength > 0 && length > lv.MaxLength {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "string",
			Message: fmt.Sprintf("maximum length is %d characters", lv.MaxLength),
			Code:    "STRING_TOO_LONG",
		})
	}

	return result
}

// RequiredValidator validates that a value is not nil/empty
type RequiredValidator struct{}

// Validate validates that a value is not nil/empty
func (rv *RequiredValidator) Validate(ctx context.Context, value interface{}) ValidationResult {
	result := ValidationResult{Valid: true}

	if value == nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "value",
			Message: "value is required",
			Code:    "REQUIRED_VALUE",
		})
		return result
	}

	// Check for empty strings
	if str, ok := value.(string); ok && str == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "value",
			Message: "value cannot be empty",
			Code:    "EMPTY_VALUE",
		})
		return result
	}

	// Check for zero values using reflection
	rv_value := reflect.ValueOf(value)
	if rv_value.Kind() == reflect.Ptr && rv_value.IsNil() {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "value",
			Message: "value cannot be nil",
			Code:    "NIL_VALUE",
		})
	}

	return result
}

// CompositeValidator combines multiple validators
type CompositeValidator struct {
	validators []Validator
}

// NewCompositeValidator creates a new composite validator
func NewCompositeValidator(validators ...Validator) *CompositeValidator {
	return &CompositeValidator{validators: validators}
}

// Validate runs all validators and aggregates results
func (cv *CompositeValidator) Validate(ctx context.Context, value interface{}) ValidationResult {
	result := ValidationResult{Valid: true}

	for _, validator := range cv.validators {
		validationResult := validator.Validate(ctx, value)
		if !validationResult.Valid {
			result.Valid = false
			result.Errors = append(result.Errors, validationResult.Errors...)
		}
	}

	return result
}

// MaskSensitiveData masks sensitive data in logs and outputs
func MaskSensitiveData(key string, value interface{}) interface{} {
	if value == nil {
		return value
	}

	str, ok := value.(string)
	if !ok {
		return value
	}

	// Check if the key indicates sensitive data
	for sensitiveType, pattern := range SensitiveDataPatterns {
		if pattern.MatchString(key) {
			return maskString(str, sensitiveType)
		}
	}

	// Check if the value itself contains sensitive patterns
	for sensitiveType, pattern := range SensitiveDataPatterns {
		if pattern.MatchString(str) {
			return maskString(str, sensitiveType)
		}
	}

	return value
}

// maskString masks a string based on its type
func maskString(str, sensitiveType string) string {
	if str == "" {
		return str
	}

	switch sensitiveType {
	case "password", "token", "secret":
		return "***"
	case "email":
		parts := strings.Split(str, "@")
		if len(parts) == 2 {
			username := parts[0]
			domain := parts[1]
			if len(username) > 2 {
				return username[:2] + "***@" + domain
			}
			return "***@" + domain
		}
		return "***"
	case "phone":
		// Mask middle digits: +1 (555) ***-1234
		re := regexp.MustCompile(`(\d{3})(\d{4})`)
		return re.ReplaceAllString(str, "${1}****")
	case "ssn":
		// Mask middle digits: ***-**-1234
		re := regexp.MustCompile(`(\d{3})-?(\d{2})-?(\d{4})`)
		return re.ReplaceAllString(str, "***-**-${3}")
	case "credit_card":
		// Mask middle digits: ****-****-****-1234
		re := regexp.MustCompile(`(\d{4})[-\s]?(\d{4})[-\s]?(\d{4})[-\s]?(\d{4})`)
		return re.ReplaceAllString(str, "****-****-****-${4}")
	default:
		// Generic masking
		if len(str) <= 4 {
			return "***"
		}
		return str[:2] + strings.Repeat("*", len(str)-4) + str[len(str)-2:]
	}
}

// GenerateSecureToken generates a cryptographically secure random token
func GenerateSecureToken(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("token length must be positive")
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure token: %w", err)
	}

	return hex.EncodeToString(bytes), nil
}

// SanitizeForLog sanitizes a value for logging purposes
func SanitizeForLog(key string, value interface{}) interface{} {
	// First mask sensitive data
	masked := MaskSensitiveData(key, value)

	// If it's a string, apply additional sanitization
	if str, ok := masked.(string); ok {
		sanitizer := NewInputSanitizer(500) // Limit log entries to 500 chars
		sanitized, err := sanitizer.SanitizeString(str)
		if err != nil {
			return "***sanitization_error***"
		}
		return sanitized
	}

	return masked
}

// ValidateStruct validates a struct using field tags and custom validators
func ValidateStruct(ctx context.Context, s interface{}) ValidationResult {
	result := ValidationResult{Valid: true}

	v := reflect.ValueOf(s)
	t := reflect.TypeOf(s)

	// Handle pointer to struct
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   "struct",
				Message: "struct cannot be nil",
				Code:    "NIL_STRUCT",
			})
			return result
		}
		v = v.Elem()
		t = t.Elem()
	}

	if v.Kind() != reflect.Struct {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "struct",
			Message: "value must be a struct",
			Code:    "INVALID_TYPE",
		})
		return result
	}

	// Validate each field
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		fieldName := fieldType.Name

		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}

		// Check validation tags
		if tag := fieldType.Tag.Get("validate"); tag != "" {
			fieldResult := validateFieldWithTag(ctx, fieldName, field.Interface(), tag)
			if !fieldResult.Valid {
				result.Valid = false
				result.Errors = append(result.Errors, fieldResult.Errors...)
			}
		}
	}

	return result
}

// validateFieldWithTag validates a field based on its validation tag
func validateFieldWithTag(ctx context.Context, fieldName string, value interface{}, tag string) ValidationResult {
	result := ValidationResult{Valid: true}

	// Parse validation tags (simplified version)
	rules := strings.Split(tag, ",")
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}

		var validator Validator

		switch {
		case rule == "required":
			validator = &RequiredValidator{}
		case rule == "email":
			validator = &EmailValidator{}
		case rule == "uuid":
			validator = &UUIDValidator{}
		case strings.HasPrefix(rule, "min="):
			// Parse min length
			if length, err := parseIntFromTag(rule, "min="); err == nil {
				validator = &LengthValidator{MinLength: length}
			}
		case strings.HasPrefix(rule, "max="):
			// Parse max length
			if length, err := parseIntFromTag(rule, "max="); err == nil {
				validator = &LengthValidator{MaxLength: length}
			}
		}

		if validator != nil {
			fieldResult := validator.Validate(ctx, value)
			if !fieldResult.Valid {
				result.Valid = false
				// Update field name in errors
				for _, err := range fieldResult.Errors {
					err.Field = fieldName
					result.Errors = append(result.Errors, err)
				}
			}
		}
	}

	return result
}

// parseIntFromTag parses an integer from a validation tag
func parseIntFromTag(tag, prefix string) (int, error) {
	if !strings.HasPrefix(tag, prefix) {
		return 0, fmt.Errorf("tag does not start with %s", prefix)
	}

	valueStr := strings.TrimPrefix(tag, prefix)
	return parseInt(valueStr)
}

// parseInt parses a string to int with basic error handling
func parseInt(s string) (int, error) {
	var result int
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid integer: %s", s)
		}
		result = result*10 + int(r-'0')
	}
	return result, nil
}

// SecurityContext provides security-related context for database operations
type SecurityContext struct {
	UserID    string
	TenantID  string
	IPAddress string
	UserAgent string
	Timestamp time.Time
}

// NewSecurityContext creates a new security context
func NewSecurityContext(userID, tenantID, ipAddress, userAgent string) *SecurityContext {
	return &SecurityContext{
		UserID:    userID,
		TenantID:  tenantID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Timestamp: time.Now(),
	}
}

// ToContext adds security context to a Go context
func (sc *SecurityContext) ToContext(ctx context.Context) context.Context {
	ctx = context.WithValue(ctx, "security_context", sc)
	ctx = context.WithValue(ctx, "user_id", sc.UserID)
	ctx = context.WithValue(ctx, "tenant_id", sc.TenantID)
	return ctx
}

// FromContext extracts security context from a Go context
func FromContext(ctx context.Context) *SecurityContext {
	if sc, ok := ctx.Value("security_context").(*SecurityContext); ok {
		return sc
	}
	return nil
}
