// Package examples demonstrates field-level encryption functionality
package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"strings"
	"time"

	"go-ormx/ormx/security"
)

// Example demonstrating field-level encryption
func encryptionExample() {
	fmt.Println("=== Go-ORMX Field-Level Encryption Example ===")

	// 1. Setup encryption configuration
	fmt.Println("1. Setting up encryption configuration...")
	encryptionConfig := setupEncryptionConfig()

	// 2. Create field encryptor
	fmt.Println("2. Creating field encryptor...")
	encryptor, err := security.NewFieldEncryptor(encryptionConfig)
	if err != nil {
		log.Fatalf("Failed to create field encryptor: %v", err)
	}

	// 3. Demonstrate basic encryption/decryption
	fmt.Println("3. Demonstrating basic encryption/decryption...")
	demonstrateBasicEncryption(encryptor)

	// 4. Demonstrate GORM integration
	fmt.Println("4. Demonstrating GORM integration...")
	demonstrateGORMIntegration(encryptor)

	// 5. Demonstrate bulk encryption
	fmt.Println("5. Demonstrating bulk encryption...")
	demonstrateBulkEncryption(encryptor)

	// 6. Demonstrate sensitive data detection
	fmt.Println("6. Demonstrating sensitive data detection...")
	demonstrateSensitiveDataDetection(encryptor)

	fmt.Println("\n=== Encryption Example Complete ===")
}

// setupEncryptionConfig creates a secure encryption configuration
func setupEncryptionConfig() *security.EncryptionConfig {
	// Generate a secure master key (in production, this should be stored securely)
	masterKey := generateSecureKey(32) // 256-bit key

	// Generate a unique salt for key derivation
	keySalt := generateSecureKey(16)

	return &security.EncryptionConfig{
		MasterKey:               masterKey,
		KeySalt:                 keySalt,
		Algorithm:               "AES-256-GCM",
		KeyRotationEnabled:      false,
		KeyRotationPeriod:       30 * 24 * time.Hour, // 30 days
		UseHardwareAcceleration: true,
	}
}

// generateSecureKey generates a secure random key of the specified length
func generateSecureKey(length int) []byte {
	key := make([]byte, length)
	if _, err := rand.Read(key); err != nil {
		log.Fatalf("Failed to generate secure key: %v", err)
	}
	return key
}

// demonstrateBasicEncryption shows basic encryption and decryption
func demonstrateBasicEncryption(encryptor *security.FieldEncryptor) {
	fmt.Println("   - Encrypting sensitive data...")

	// Sensitive data to encrypt
	sensitiveData := map[string]string{
		"password":     "mySecurePassword123!",
		"ssn":          "123-45-6789",
		"credit_card":  "4111-1111-1111-1111",
		"phone_number": "+1-555-123-4567",
		"address":      "123 Main St, Anytown, USA 12345",
	}

	// Encrypt each piece of sensitive data
	encryptedData := make(map[string]*security.EncryptedField)
	for key, value := range sensitiveData {
		encrypted, err := encryptor.Encrypt(value)
		if err != nil {
			log.Printf("Failed to encrypt %s: %v", key, err)
			continue
		}
		encryptedData[key] = encrypted

		fmt.Printf("     ✓ Encrypted %s: %s -> %s\n", key, value, encrypted.Data[:20]+"...")
	}

	// Decrypt the data
	fmt.Println("   - Decrypting sensitive data...")
	for key, encrypted := range encryptedData {
		decrypted, err := encryptor.Decrypt(encrypted)
		if err != nil {
			log.Printf("Failed to decrypt %s: %v", key, err)
			continue
		}

		fmt.Printf("     ✓ Decrypted %s: %s\n", key, decrypted)
	}
}

// demonstrateGORMIntegration shows how to use encrypted fields with GORM
func demonstrateGORMIntegration(encryptor *security.FieldEncryptor) {
	fmt.Println("   - Creating encrypted string field...")

	// Create an encrypted string field
	encryptedField := security.NewEncryptedString(encryptor)

	// Set a sensitive value (will be encrypted)
	sensitiveValue := "mySecretToken123"
	if err := encryptedField.Set(sensitiveValue); err != nil {
		log.Printf("Failed to set encrypted field: %v", err)
		return
	}

	fmt.Printf("     ✓ Set plaintext value: %s\n", sensitiveValue)

	// Get the encrypted value (for storage)
	encryptedValue := encryptedField.String()
	fmt.Printf("     ✓ Encrypted value: %s\n", encryptedValue[:50]+"...")

	// Get the decrypted value
	decryptedValue, err := encryptedField.Get()
	if err != nil {
		log.Printf("Failed to get decrypted value: %v", err)
		return
	}

	fmt.Printf("     ✓ Decrypted value: %s\n", decryptedValue)

	// Demonstrate JSON marshaling
	fmt.Println("   - Demonstrating JSON marshaling...")
	jsonData, err := encryptedField.MarshalJSON()
	if err != nil {
		log.Printf("Failed to marshal JSON: %v", err)
		return
	}

	fmt.Printf("     ✓ JSON representation: %s\n", string(jsonData)[:50]+"...")

	// Demonstrate JSON unmarshaling
	newField := security.NewEncryptedString(encryptor)
	if err := newField.UnmarshalJSON(jsonData); err != nil {
		log.Printf("Failed to unmarshal JSON: %v", err)
		return
	}

	decryptedAgain, err := newField.Get()
	if err != nil {
		log.Printf("Failed to get decrypted value after unmarshaling: %v", err)
		return
	}

	fmt.Printf("     ✓ Value after JSON round-trip: %s\n", decryptedAgain)
}

// demonstrateBulkEncryption shows bulk encryption operations
func demonstrateBulkEncryption(encryptor *security.FieldEncryptor) {
	fmt.Println("   - Performing bulk encryption...")

	// Sample data with sensitive and non-sensitive fields
	data := map[string]interface{}{
		"username":     "john_doe",
		"email":        "john@example.com",
		"password":     "securePassword123",
		"ssn":          "987-65-4321",
		"credit_card":  "5555-4444-3333-2222",
		"phone_number": "+1-555-987-6543",
		"address":      "456 Oak Ave, Somewhere, USA 54321",
		"age":          30,
		"is_active":    true,
	}

	// Encrypt sensitive data automatically
	encryptedData, err := security.EncryptSensitiveData(encryptor, data)
	if err != nil {
		log.Printf("Failed to encrypt sensitive data: %v", err)
		return
	}

	fmt.Println("     ✓ Encrypted sensitive fields:")
	for key, value := range encryptedData {
		if encryptedField, ok := value.(*security.EncryptedField); ok {
			fmt.Printf("       - %s: %s\n", key, encryptedField.Data[:20]+"...")
		} else {
			fmt.Printf("       - %s: %v (not encrypted)\n", key, value)
		}
	}

	// Decrypt sensitive data
	decryptedData, err := security.DecryptSensitiveData(encryptor, encryptedData)
	if err != nil {
		log.Printf("Failed to decrypt sensitive data: %v", err)
		return
	}

	fmt.Println("     ✓ Decrypted sensitive fields:")
	for key, value := range decryptedData {
		if key == "password" || key == "ssn" || key == "credit_card" || key == "phone_number" || key == "address" {
			fmt.Printf("       - %s: %v\n", key, value)
		}
	}
}

// demonstrateSensitiveDataDetection shows automatic sensitive data detection
func demonstrateSensitiveDataDetection(encryptor *security.FieldEncryptor) {
	fmt.Println("   - Testing sensitive data detection...")

	// Test various field names
	testFields := []string{
		"password",
		"user_password",
		"access_token",
		"api_key",
		"secret_key",
		"ssn",
		"social_security_number",
		"credit_card",
		"card_number",
		"phone",
		"telephone_number",
		"address",
		"street_address",
		"email",
		"username",
		"first_name",
		"age",
		"is_active",
	}

	fmt.Println("     ✓ Field sensitivity analysis:")
	for _, field := range testFields {
		isSensitive := isSensitiveField(field)
		status := "❌"
		if isSensitive {
			status = "✅"
		}
		fmt.Printf("       %s %s\n", status, field)
	}
}

// ExampleUser demonstrates a simple user struct with encrypted fields
type ExampleUser struct {
	ID          string                    `json:"id"`
	Username    string                    `json:"username"`
	Email       string                    `json:"email"`
	Password    *security.EncryptedString `json:"-"` // Never expose in JSON
	SSN         *security.EncryptedString `json:"-"` // Never expose in JSON
	CreditCard  *security.EncryptedString `json:"-"` // Never expose in JSON
	PhoneNumber *security.EncryptedString `json:"-"` // Never expose in JSON
	Address     *security.EncryptedString `json:"-"` // Never expose in JSON
	CreatedAt   time.Time                 `json:"created_at"`
	UpdatedAt   time.Time                 `json:"updated_at"`
}

// NewExampleUser creates a new example user with encrypted fields
func NewExampleUser(encryptor *security.FieldEncryptor) *ExampleUser {
	return &ExampleUser{
		Password:    security.NewEncryptedString(encryptor),
		SSN:         security.NewEncryptedString(encryptor),
		CreditCard:  security.NewEncryptedString(encryptor),
		PhoneNumber: security.NewEncryptedString(encryptor),
		Address:     security.NewEncryptedString(encryptor),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// SetSensitiveData sets sensitive data with encryption
func (u *ExampleUser) SetSensitiveData(data map[string]string) error {
	if password, ok := data["password"]; ok {
		if err := u.Password.Set(password); err != nil {
			return fmt.Errorf("failed to set password: %w", err)
		}
	}

	if ssn, ok := data["ssn"]; ok {
		if err := u.SSN.Set(ssn); err != nil {
			return fmt.Errorf("failed to set SSN: %w", err)
		}
	}

	if creditCard, ok := data["credit_card"]; ok {
		if err := u.CreditCard.Set(creditCard); err != nil {
			return fmt.Errorf("failed to set credit card: %w", err)
		}
	}

	if phone, ok := data["phone"]; ok {
		if err := u.PhoneNumber.Set(phone); err != nil {
			return fmt.Errorf("failed to set phone number: %w", err)
		}
	}

	if address, ok := data["address"]; ok {
		if err := u.Address.Set(address); err != nil {
			return fmt.Errorf("failed to set address: %w", err)
		}
	}

	return nil
}

// GetSensitiveData returns decrypted sensitive data
func (u *ExampleUser) GetSensitiveData() (map[string]string, error) {
	data := make(map[string]string)

	if password, err := u.Password.Get(); err == nil {
		data["password"] = password
	}

	if ssn, err := u.SSN.Get(); err == nil {
		data["ssn"] = ssn
	}

	if creditCard, err := u.CreditCard.Get(); err == nil {
		data["credit_card"] = creditCard
	}

	if phone, err := u.PhoneNumber.Get(); err == nil {
		data["phone"] = phone
	}

	if address, err := u.Address.Get(); err == nil {
		data["address"] = address
	}

	return data, nil
}

// demonstrateUserExample shows how to use encrypted fields in a user struct
func demonstrateUserExample(encryptor *security.FieldEncryptor) {
	fmt.Println("   - Demonstrating user struct with encrypted fields...")

	// Create a new user
	user := NewExampleUser(encryptor)
	user.ID = "user_123"
	user.Username = "john_doe"
	user.Email = "john@example.com"

	// Set sensitive data
	sensitiveData := map[string]string{
		"password":    "mySecurePassword123!",
		"ssn":         "123-45-6789",
		"credit_card": "4111-1111-1111-1111",
		"phone":       "+1-555-123-4567",
		"address":     "123 Main St, Anytown, USA 12345",
	}

	if err := user.SetSensitiveData(sensitiveData); err != nil {
		log.Printf("Failed to set sensitive data: %v", err)
		return
	}

	fmt.Printf("     ✓ Created user: %s (%s)\n", user.Username, user.Email)

	// Retrieve sensitive data
	retrievedData, err := user.GetSensitiveData()
	if err != nil {
		log.Printf("Failed to get sensitive data: %v", err)
		return
	}

	fmt.Println("     ✓ Retrieved sensitive data:")
	for key, value := range retrievedData {
		fmt.Printf("       - %s: %s\n", key, value)
	}
}

// isSensitiveField checks if a field name indicates sensitive data
func isSensitiveField(fieldName string) bool {
	fieldName = strings.ToLower(fieldName)

	sensitivePatterns := []string{
		"password", "passwd", "pwd",
		"token", "access_token", "refresh_token", "api_key",
		"secret", "private_key", "secret_key",
		"ssn", "social_security",
		"credit_card", "card_number",
		"phone", "telephone",
		"address", "street_address",
		"email", "mail",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(fieldName, pattern) {
			return true
		}
	}

	return false
}
