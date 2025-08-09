# Field-Level Encryption

Go-ORMX provides comprehensive field-level encryption for sensitive data using AES-256-GCM encryption. This feature ensures that sensitive information is encrypted at rest in the database while maintaining full GORM compatibility.

## 🔒 Security Features

### **Encryption Algorithm**
- **AES-256-GCM**: Authenticated encryption with associated data
- **Random Nonce**: Each encryption operation uses a unique nonce
- **Key Derivation**: Secure key derivation using SHA-256
- **Hardware Acceleration**: Optional hardware acceleration support

### **Security Benefits**
- **Data at Rest Protection**: Sensitive data is encrypted in the database
- **Transparent Encryption**: Automatic encryption/decryption with GORM
- **Audit Trail**: Encryption metadata includes timestamps and key versions
- **Key Rotation**: Support for key rotation and versioning

## 🚀 Quick Start

### 1. Setup Encryption Configuration

```go
import (
    "crypto/rand"
    "time"
    "go-ormx/internal/security"
)

// Generate secure master key (store securely in production)
masterKey := make([]byte, 32) // 256-bit key
rand.Read(masterKey)

// Generate unique salt for key derivation
keySalt := make([]byte, 16)
rand.Read(keySalt)

// Create encryption configuration
config := &security.EncryptionConfig{
    MasterKey:              masterKey,
    KeySalt:                keySalt,
    Algorithm:              "AES-256-GCM",
    KeyRotationEnabled:     false,
    KeyRotationPeriod:      30 * 24 * time.Hour, // 30 days
    UseHardwareAcceleration: true,
}
```

### 2. Create Field Encryptor

```go
// Create field encryptor
encryptor, err := security.NewFieldEncryptor(config)
if err != nil {
    log.Fatalf("Failed to create field encryptor: %v", err)
}
```

### 3. Use Encrypted Fields in Models

```go
type User struct {
    models.BaseModel
    
    // Basic fields (not encrypted)
    Username string `gorm:"type:varchar(255);uniqueIndex" json:"username"`
    Email    string `gorm:"type:varchar(255);uniqueIndex" json:"email"`
    
    // Encrypted sensitive fields
    Password     security.EncryptedString `gorm:"type:text" json:"-"` // Never expose in JSON
    SSN          security.EncryptedString `gorm:"type:text" json:"-"` // Never expose in JSON
    CreditCard   security.EncryptedString `gorm:"type:text" json:"-"` // Never expose in JSON
    PhoneNumber  security.EncryptedString `gorm:"type:text" json:"-"` // Never expose in JSON
    Address      security.EncryptedString `gorm:"type:text" json:"-"` // Never expose in JSON
}
```

### 4. Set and Get Encrypted Values

```go
// Create user with encrypted fields
user := &User{
    Username: "john_doe",
    Email:    "john@example.com",
}

// Set sensitive data (automatically encrypted)
if err := user.Password.Set("mySecurePassword123!"); err != nil {
    log.Printf("Failed to set password: %v", err)
}

if err := user.SSN.Set("123-45-6789"); err != nil {
    log.Printf("Failed to set SSN: %v", err)
}

// Save to database (encrypted automatically)
db.Create(user)

// Retrieve and decrypt
retrievedUser := &User{}
db.First(retrievedUser, user.ID)

// Get decrypted values
password, err := retrievedUser.Password.Get()
if err != nil {
    log.Printf("Failed to get password: %v", err)
}

ssn, err := retrievedUser.SSN.Get()
if err != nil {
    log.Printf("Failed to get SSN: %v", err)
}
```

## 📋 API Reference

### EncryptionConfig

```go
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
```

### FieldEncryptor

```go
type FieldEncryptor struct {
    config *EncryptionConfig
    aead   cipher.AEAD
}

// NewFieldEncryptor creates a new field encryptor
func NewFieldEncryptor(config *EncryptionConfig) (*FieldEncryptor, error)

// Encrypt encrypts a plaintext string
func (fe *FieldEncryptor) Encrypt(plaintext string) (*EncryptedField, error)

// Decrypt decrypts an encrypted field
func (fe *FieldEncryptor) Decrypt(encryptedField *EncryptedField) (string, error)
```

### EncryptedString

```go
type EncryptedString struct {
    encryptor *FieldEncryptor
    value     *EncryptedField
}

// NewEncryptedString creates a new encrypted string field
func NewEncryptedString(encryptor *FieldEncryptor) *EncryptedString

// Set sets the plaintext value (will be encrypted)
func (es *EncryptedString) Set(plaintext string) error

// Get returns the decrypted plaintext value
func (es *EncryptedString) Get() (string, error)

// GetEncrypted returns the encrypted field value
func (es *EncryptedString) GetEncrypted() *EncryptedField

// SetEncrypted sets the encrypted field value
func (es *EncryptedString) SetEncrypted(encrypted *EncryptedField)
```

### EncryptedField

```go
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
```

## 🔧 Advanced Usage

### Bulk Encryption

```go
// Encrypt sensitive data automatically
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

// Automatically detect and encrypt sensitive fields
encryptedData, err := security.EncryptSensitiveData(encryptor, data)
if err != nil {
    log.Printf("Failed to encrypt sensitive data: %v", err)
}

// Decrypt sensitive data
decryptedData, err := security.DecryptSensitiveData(encryptor, encryptedData)
if err != nil {
    log.Printf("Failed to decrypt sensitive data: %v", err)
}
```

### Repository Integration

```go
// Create repository with encryption support
type EncryptedUserRepository struct {
    *repositories.BaseRepository[User]
    encryptor *security.FieldEncryptor
}

func NewEncryptedUserRepository(
    db *gorm.DB,
    logger logging.Logger,
    encryptor *security.FieldEncryptor,
    opts repositories.RepositoryOptions,
) *EncryptedUserRepository {
    return &EncryptedUserRepository{
        BaseRepository: repositories.NewBaseRepository[User](db, logger, opts),
        encryptor:      encryptor,
    }
}

// Create with automatic encryption
func (r *EncryptedUserRepository) CreateWithEncryption(ctx context.Context, user *User) error {
    return r.Create(ctx, *user)
}

// Retrieve with automatic decryption
func (r *EncryptedUserRepository) GetByIDWithDecryption(ctx context.Context, id string) (*User, error) {
    user, err := r.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }
    
    return &user, nil
}
```

### JSON Serialization

```go
// EncryptedString implements json.Marshaler and json.Unmarshaler
encryptedField := security.NewEncryptedString(encryptor)
encryptedField.Set("sensitive data")

// Marshal to JSON
jsonData, err := json.Marshal(encryptedField)
if err != nil {
    log.Printf("Failed to marshal: %v", err)
}

// Unmarshal from JSON
newField := security.NewEncryptedString(encryptor)
err = json.Unmarshal(jsonData, newField)
if err != nil {
    log.Printf("Failed to unmarshal: %v", err)
}

// Get decrypted value
value, err := newField.Get()
if err != nil {
    log.Printf("Failed to get value: %v", err)
}
```

## 🛡️ Security Best Practices

### 1. Key Management

```go
// Store master key securely (use environment variables or secure key management)
masterKey := []byte(os.Getenv("ENCRYPTION_MASTER_KEY"))

// Use unique salt per application/environment
keySalt := []byte(os.Getenv("ENCRYPTION_KEY_SALT"))

// Rotate keys regularly
config := &security.EncryptionConfig{
    MasterKey:              masterKey,
    KeySalt:                keySalt,
    KeyRotationEnabled:     true,
    KeyRotationPeriod:      30 * 24 * time.Hour, // 30 days
}
```

### 2. Field Selection

```go
// Only encrypt truly sensitive data
type User struct {
    // Encrypt these fields
    Password     security.EncryptedString `gorm:"type:text" json:"-"`
    SSN          security.EncryptedString `gorm:"type:text" json:"-"`
    CreditCard   security.EncryptedString `gorm:"type:text" json:"-"`
    
    // Don't encrypt these (not sensitive)
    Username     string `gorm:"type:varchar(255)" json:"username"`
    Email        string `gorm:"type:varchar(255)" json:"email"`
    FirstName    string `gorm:"type:varchar(255)" json:"first_name"`
    LastName     string `gorm:"type:varchar(255)" json:"last_name"`
}
```

### 3. JSON Exposure Prevention

```go
// Never expose encrypted fields in JSON responses
type User struct {
    ID       string `json:"id"`
    Username string `json:"username"`
    Email    string `json:"email"`
    
    // Use json:"-" to prevent exposure
    Password security.EncryptedString `gorm:"type:text" json:"-"`
    SSN      security.EncryptedString `gorm:"type:text" json:"-"`
}
```

### 4. Error Handling

```go
// Always handle encryption/decryption errors
password, err := user.Password.Get()
if err != nil {
    log.Printf("Failed to decrypt password: %v", err)
    // Handle error appropriately (e.g., return error to client)
    return err
}

// Validate decrypted data
if password == "" {
    return errors.New("password is empty")
}
```

### 5. Logging Security

```go
// Never log sensitive data
log.Printf("User %s logged in", user.Username) // ✅ Safe
log.Printf("User password: %s", password)      // ❌ Dangerous

// Use structured logging with sensitive data masking
logger.Info("User authentication",
    logging.String("username", user.Username),
    logging.String("status", "success"),
    // Don't log password or other sensitive fields
)
```

## 🔄 Key Rotation

### Automatic Key Rotation

```go
// Enable key rotation
config := &security.EncryptionConfig{
    MasterKey:              masterKey,
    KeySalt:                keySalt,
    KeyRotationEnabled:     true,
    KeyRotationPeriod:      30 * 24 * time.Hour, // 30 days
}

// Key rotation is handled automatically
// Old keys are kept for decryption of existing data
// New data is encrypted with the current key
```

### Manual Key Rotation

```go
// For manual key rotation, you may need to re-encrypt existing data
func RotateKeys(oldEncryptor, newEncryptor *security.FieldEncryptor, users []User) error {
    for _, user := range users {
        // Decrypt with old key
        oldPassword, err := user.Password.Get()
        if err != nil {
            return err
        }
        
        // Re-encrypt with new key
        if err := user.Password.Set(oldPassword); err != nil {
            return err
        }
        
        // Save back to database
        db.Save(&user)
    }
    return nil
}
```

## 📊 Performance Considerations

### Encryption Overhead

- **Encryption**: ~1-5ms per field (depending on data size)
- **Decryption**: ~1-3ms per field
- **Memory**: Minimal overhead for encrypted fields
- **Storage**: ~30-50% increase in storage size for encrypted data

### Optimization Tips

```go
// Use hardware acceleration when available
config := &security.EncryptionConfig{
    UseHardwareAcceleration: true,
}

// Batch operations for better performance
func BatchEncrypt(encryptor *security.FieldEncryptor, data []map[string]string) error {
    for _, item := range data {
        // Process in batches
        if err := encryptSensitiveFields(encryptor, item); err != nil {
            return err
        }
    }
    return nil
}

// Cache frequently accessed decrypted data (with appropriate TTL)
type CachedUser struct {
    User      *User
    Decrypted map[string]string
    ExpiresAt time.Time
}
```

## 🧪 Testing

### Unit Tests

```go
func TestEncryption(t *testing.T) {
    // Setup
    config := &security.EncryptionConfig{
        MasterKey: []byte("test-master-key-32-bytes-long"),
        KeySalt:   []byte("test-salt-16-bytes"),
    }
    
    encryptor, err := security.NewFieldEncryptor(config)
    require.NoError(t, err)
    
    // Test encryption/decryption
    plaintext := "sensitive data"
    encrypted, err := encryptor.Encrypt(plaintext)
    require.NoError(t, err)
    
    decrypted, err := encryptor.Decrypt(encrypted)
    require.NoError(t, err)
    require.Equal(t, plaintext, decrypted)
}

func TestEncryptedString(t *testing.T) {
    // Setup
    encryptor := setupTestEncryptor(t)
    field := security.NewEncryptedString(encryptor)
    
    // Test Set/Get
    testValue := "test password"
    err := field.Set(testValue)
    require.NoError(t, err)
    
    retrieved, err := field.Get()
    require.NoError(t, err)
    require.Equal(t, testValue, retrieved)
}
```

### Integration Tests

```go
func TestEncryptedUserRepository(t *testing.T) {
    // Setup database and encryptor
    db := setupTestDB(t)
    encryptor := setupTestEncryptor(t)
    
    repo := NewEncryptedUserRepository(db, logger, encryptor, repositories.DefaultRepositoryOptions())
    
    // Test create with encryption
    user := &User{
        Username: "testuser",
        Email:    "test@example.com",
    }
    user.Password.Set("testpassword")
    
    err := repo.CreateWithEncryption(context.Background(), user)
    require.NoError(t, err)
    
    // Test retrieve with decryption
    retrieved, err := repo.GetByIDWithDecryption(context.Background(), user.ID)
    require.NoError(t, err)
    
    password, err := retrieved.Password.Get()
    require.NoError(t, err)
    require.Equal(t, "testpassword", password)
}
```

## 🚨 Security Considerations

### 1. Key Storage
- **Never hardcode encryption keys**
- **Use secure key management systems** (AWS KMS, Azure Key Vault, etc.)
- **Rotate keys regularly**
- **Backup keys securely**

### 2. Access Control
- **Limit access to encryption keys**
- **Use principle of least privilege**
- **Monitor key access and usage**

### 3. Data Classification
- **Only encrypt truly sensitive data**
- **Consider performance impact**
- **Balance security with usability**

### 4. Compliance
- **GDPR**: Encrypt personal data at rest
- **HIPAA**: Encrypt protected health information
- **PCI DSS**: Encrypt cardholder data
- **SOX**: Encrypt financial data

## 📚 Examples

See the following examples for complete implementations:

- `examples/encryption_example.go` - Basic encryption usage
- `examples/models/encrypted_user.go` - GORM model with encrypted fields
- `examples/repositories/encrypted_user_repository.go` - Repository with encryption support

## 🔗 Related Documentation

- [Security Overview](../SECURITY.md)
- [Production Readiness](../PRODUCTION_ANALYSIS.md)
- [API Reference](../API.md)
- [Best Practices](../BEST_PRACTICES.md)
