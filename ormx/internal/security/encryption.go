// Package security provides field-level encryption for sensitive data
package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// defaultEncryptor holds a global encryptor used by EncryptedString when no instance encryptor is provided
var (
	defaultEncryptor     *FieldEncryptor
	defaultEncryptorLock sync.RWMutex
)

// SetDefaultEncryptor sets the global default field encryptor
func SetDefaultEncryptor(encryptor *FieldEncryptor) {
	defaultEncryptorLock.Lock()
	defer defaultEncryptorLock.Unlock()
	defaultEncryptor = encryptor
}

// getDefaultEncryptor returns the global default field encryptor (internal)
func getDefaultEncryptor() *FieldEncryptor {
	defaultEncryptorLock.RLock()
	defer defaultEncryptorLock.RUnlock()
	return defaultEncryptor
}

// EncryptionConfig holds encryption configuration
type EncryptionConfig struct {
	// Master key for encryption (should be stored securely)
	MasterKey []byte

	// Key derivation salt (should be unique per application)
	KeySalt []byte

	// Algorithm to use for encryption
	Algorithm string

	// Key rotation settings
	KeyRotationEnabled bool
	KeyRotationPeriod  time.Duration

	// Performance settings
	UseHardwareAcceleration bool
}

// DefaultEncryptionConfig returns a default encryption configuration
func DefaultEncryptionConfig() *EncryptionConfig {
	return &EncryptionConfig{
		Algorithm:               "AES-256-GCM",
		KeyRotationEnabled:      false,
		KeyRotationPeriod:       30 * 24 * time.Hour, // 30 days
		UseHardwareAcceleration: true,
	}
}

// EncryptedField represents an encrypted field value
type EncryptedField struct {
	// Encrypted data (base64 encoded)
	Data string `json:"data"`

	// Nonce used for encryption (base64 encoded)
	Nonce string `json:"nonce"`

	// Key version for key rotation
	KeyVersion int `json:"key_version"`

	// Algorithm used for encryption
	Algorithm string `json:"algorithm"`

	// Timestamp when encrypted
	Timestamp time.Time `json:"timestamp"`
}

// String returns the encrypted field as a JSON string
func (ef *EncryptedField) String() string {
	data, _ := json.Marshal(ef)
	return string(data)
}

// MarshalJSON implements json.Marshaler
func (ef *EncryptedField) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"data":        ef.Data,
		"nonce":       ef.Nonce,
		"key_version": ef.KeyVersion,
		"algorithm":   ef.Algorithm,
		"timestamp":   ef.Timestamp,
	})
}

// UnmarshalJSON implements json.Unmarshaler
func (ef *EncryptedField) UnmarshalJSON(data []byte) error {
	var temp struct {
		Data       string    `json:"data"`
		Nonce      string    `json:"nonce"`
		KeyVersion int       `json:"key_version"`
		Algorithm  string    `json:"algorithm"`
		Timestamp  time.Time `json:"timestamp"`
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	ef.Data = temp.Data
	ef.Nonce = temp.Nonce
	ef.KeyVersion = temp.KeyVersion
	ef.Algorithm = temp.Algorithm
	ef.Timestamp = temp.Timestamp

	return nil
}

// FieldEncryptor provides field-level encryption functionality
type FieldEncryptor struct {
	config *EncryptionConfig
	aead   cipher.AEAD
}

// NewFieldEncryptor creates a new field encryptor
func NewFieldEncryptor(config *EncryptionConfig) (*FieldEncryptor, error) {
	if config == nil {
		config = DefaultEncryptionConfig()
	}

	if len(config.MasterKey) == 0 {
		return nil, fmt.Errorf("master key is required")
	}

	// Derive encryption key from master key
	key := deriveKey(config.MasterKey, config.KeySalt)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create GCM mode
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM mode: %w", err)
	}

	return &FieldEncryptor{
		config: config,
		aead:   aead,
	}, nil
}

// deriveKey derives an encryption key from the master key and salt
func deriveKey(masterKey, salt []byte) []byte {
	// Use SHA-256 for key derivation
	hash := sha256.New()
	hash.Write(masterKey)
	hash.Write(salt)
	return hash.Sum(nil)
}

// Encrypt encrypts a plaintext string
func (fe *FieldEncryptor) Encrypt(plaintext string) (*EncryptedField, error) {
	if plaintext == "" {
		return &EncryptedField{}, nil
	}

	// Generate random nonce
	nonce := make([]byte, fe.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the plaintext
	ciphertext := fe.aead.Seal(nil, nonce, []byte(plaintext), nil)

	// Create encrypted field
	encryptedField := &EncryptedField{
		Data:       base64.StdEncoding.EncodeToString(ciphertext),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		KeyVersion: 1, // Current key version
		Algorithm:  fe.config.Algorithm,
		Timestamp:  time.Now(),
	}

	return encryptedField, nil
}

// Decrypt decrypts an encrypted field
func (fe *FieldEncryptor) Decrypt(encryptedField *EncryptedField) (string, error) {
	if encryptedField == nil || encryptedField.Data == "" {
		return "", nil
	}

	// Decode base64 data
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedField.Data)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	// Decode base64 nonce
	nonce, err := base64.StdEncoding.DecodeString(encryptedField.Nonce)
	if err != nil {
		return "", fmt.Errorf("failed to decode nonce: %w", err)
	}

	// Decrypt the ciphertext
	plaintext, err := fe.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// EncryptedString represents an encrypted string field for GORM
type EncryptedString struct {
	encryptor *FieldEncryptor
	value     *EncryptedField
}

// NewEncryptedString creates a new encrypted string field
func NewEncryptedString(encryptor *FieldEncryptor) *EncryptedString {
	return &EncryptedString{
		encryptor: encryptor,
		value:     &EncryptedField{},
	}
}

// Set sets the plaintext value (will be encrypted)
func (es *EncryptedString) Set(plaintext string) error {
	if plaintext == "" {
		es.value = &EncryptedField{}
		return nil
	}

	encryptor := es.encryptor
	if encryptor == nil {
		encryptor = getDefaultEncryptor()
		if encryptor == nil {
			return fmt.Errorf("field encryption not configured")
		}
		es.encryptor = encryptor
	}

	encrypted, err := encryptor.Encrypt(plaintext)
	if err != nil {
		return err
	}

	es.value = encrypted
	return nil
}

// Get returns the decrypted plaintext value
func (es *EncryptedString) Get() (string, error) {
	if es.value == nil || es.value.Data == "" {
		return "", nil
	}
	encryptor := es.encryptor
	if encryptor == nil {
		encryptor = getDefaultEncryptor()
		if encryptor == nil {
			return "", fmt.Errorf("field encryption not configured")
		}
		es.encryptor = encryptor
	}
	return encryptor.Decrypt(es.value)
}

// GetEncrypted returns the encrypted field value
func (es *EncryptedString) GetEncrypted() *EncryptedField {
	return es.value
}

// SetEncrypted sets the encrypted field value
func (es *EncryptedString) SetEncrypted(encrypted *EncryptedField) {
	es.value = encrypted
}

// String returns the encrypted field as a JSON string
func (es *EncryptedString) String() string {
	if es.value == nil {
		return ""
	}
	return es.value.String()
}

// MarshalJSON implements json.Marshaler
func (es *EncryptedString) MarshalJSON() ([]byte, error) {
	if es.value == nil {
		return json.Marshal("")
	}
	return json.Marshal(es.value)
}

// UnmarshalJSON implements json.Unmarshaler
func (es *EncryptedString) UnmarshalJSON(data []byte) error {
	var encryptedField EncryptedField
	if err := json.Unmarshal(data, &encryptedField); err != nil {
		return err
	}

	es.value = &encryptedField
	return nil
}

// Value implements driver.Valuer for GORM
func (es *EncryptedString) Value() (interface{}, error) {
	if es.value == nil {
		return "", nil
	}
	return es.value.String(), nil
}

// Scan implements sql.Scanner for GORM
func (es *EncryptedString) Scan(value interface{}) error {
	if value == nil {
		es.value = &EncryptedField{}
		return nil
	}

	switch v := value.(type) {
	case string:
		if v == "" {
			es.value = &EncryptedField{}
			return nil
		}

		var encryptedField EncryptedField
		if err := json.Unmarshal([]byte(v), &encryptedField); err != nil {
			return fmt.Errorf("failed to unmarshal encrypted field: %w", err)
		}

		es.value = &encryptedField
		return nil

	case []byte:
		if len(v) == 0 {
			es.value = &EncryptedField{}
			return nil
		}

		var encryptedField EncryptedField
		if err := json.Unmarshal(v, &encryptedField); err != nil {
			return fmt.Errorf("failed to unmarshal encrypted field: %w", err)
		}

		es.value = &encryptedField
		return nil

	default:
		return fmt.Errorf("unsupported type for encrypted field: %T", value)
	}
}

// EncryptedFieldManager manages encrypted fields globally
type EncryptedFieldManager struct {
	encryptor *FieldEncryptor
	fields    map[string]*EncryptedString
}

// NewEncryptedFieldManager creates a new encrypted field manager
func NewEncryptedFieldManager(config *EncryptionConfig) (*EncryptedFieldManager, error) {
	encryptor, err := NewFieldEncryptor(config)
	if err != nil {
		return nil, err
	}

	return &EncryptedFieldManager{
		encryptor: encryptor,
		fields:    make(map[string]*EncryptedString),
	}, nil
}

// CreateField creates a new encrypted field
func (efm *EncryptedFieldManager) CreateField(fieldName string) *EncryptedString {
	field := NewEncryptedString(efm.encryptor)
	efm.fields[fieldName] = field
	return field
}

// GetField returns an existing encrypted field
func (efm *EncryptedFieldManager) GetField(fieldName string) (*EncryptedString, bool) {
	field, exists := efm.fields[fieldName]
	return field, exists
}

// EncryptField encrypts a field value
func (efm *EncryptedFieldManager) EncryptField(fieldName, plaintext string) error {
	field := efm.CreateField(fieldName)
	return field.Set(plaintext)
}

// DecryptField decrypts a field value
func (efm *EncryptedFieldManager) DecryptField(fieldName string) (string, error) {
	field, exists := efm.GetField(fieldName)
	if !exists {
		return "", fmt.Errorf("field %s not found", fieldName)
	}

	return field.Get()
}

// GORM integration for encrypted fields
type EncryptedFieldGORM struct {
	*EncryptedString
}

// NewEncryptedFieldGORM creates a new GORM-compatible encrypted field
func NewEncryptedFieldGORM(encryptor *FieldEncryptor) *EncryptedFieldGORM {
	return &EncryptedFieldGORM{
		EncryptedString: NewEncryptedString(encryptor),
	}
}

// GormDataType returns the GORM data type
func (efg *EncryptedFieldGORM) GormDataType() string {
	return "text"
}

// GormDBDataType returns the database data type
func (efg *EncryptedFieldGORM) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "postgres":
		return "text"
	case "mysql":
		return "text"
	case "sqlite":
		return "text"
	default:
		return "text"
	}
}

// BeforeCreate GORM hook for encryption
func (efg *EncryptedFieldGORM) BeforeCreate(tx *gorm.DB) error {
	// Encryption is handled automatically by the EncryptedString type
	return nil
}

// BeforeUpdate GORM hook for encryption
func (efg *EncryptedFieldGORM) BeforeUpdate(tx *gorm.DB) error {
	// Encryption is handled automatically by the EncryptedString type
	return nil
}

// AfterFind GORM hook for decryption
func (efg *EncryptedFieldGORM) AfterFind(tx *gorm.DB) error {
	// Decryption is handled automatically by the EncryptedString type
	return nil
}

// Utility functions for common encryption operations

// EncryptSensitiveData encrypts sensitive data with automatic field detection
func EncryptSensitiveData(encryptor *FieldEncryptor, data map[string]interface{}) (map[string]interface{}, error) {
	encrypted := make(map[string]interface{})

	for key, value := range data {
		if isSensitiveField(key) {
			if strValue, ok := value.(string); ok {
				encryptedField, err := encryptor.Encrypt(strValue)
				if err != nil {
					return nil, fmt.Errorf("failed to encrypt field %s: %w", key, err)
				}
				encrypted[key] = encryptedField
			} else {
				// For non-string values, convert to string and encrypt
				strValue := fmt.Sprintf("%v", value)
				encryptedField, err := encryptor.Encrypt(strValue)
				if err != nil {
					return nil, fmt.Errorf("failed to encrypt field %s: %w", key, err)
				}
				encrypted[key] = encryptedField
			}
		} else {
			encrypted[key] = value
		}
	}

	return encrypted, nil
}

// DecryptSensitiveData decrypts sensitive data with automatic field detection
func DecryptSensitiveData(encryptor *FieldEncryptor, data map[string]interface{}) (map[string]interface{}, error) {
	decrypted := make(map[string]interface{})

	for key, value := range data {
		if isSensitiveField(key) {
			if encryptedField, ok := value.(*EncryptedField); ok {
				plaintext, err := encryptor.Decrypt(encryptedField)
				if err != nil {
					return nil, fmt.Errorf("failed to decrypt field %s: %w", key, err)
				}
				decrypted[key] = plaintext
			} else if strValue, ok := value.(string); ok && strValue != "" {
				// Try to unmarshal as encrypted field
				var encryptedField EncryptedField
				if err := json.Unmarshal([]byte(strValue), &encryptedField); err == nil {
					plaintext, err := encryptor.Decrypt(&encryptedField)
					if err != nil {
						return nil, fmt.Errorf("failed to decrypt field %s: %w", key, err)
					}
					decrypted[key] = plaintext
				} else {
					// Not an encrypted field, keep as is
					decrypted[key] = value
				}
			} else {
				decrypted[key] = value
			}
		} else {
			decrypted[key] = value
		}
	}

	return decrypted, nil
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
