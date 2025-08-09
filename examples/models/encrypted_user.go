// Package models provides example models with encrypted fields
package models

import (
	"context"
	"time"

	"go-ormx/ormx/errors"
	"go-ormx/ormx/logging"
	"go-ormx/ormx/models"
	"go-ormx/ormx/repositories"
	"go-ormx/ormx/security"

	"gorm.io/gorm"
)

// EncryptedUser represents a user with encrypted sensitive fields
type EncryptedUser struct {
	models.BaseModel

	// Basic user information (not encrypted)
	Username  string `gorm:"type:varchar(255);uniqueIndex;not null" json:"username"`
	FirstName string `gorm:"type:varchar(255);not null" json:"first_name"`
	LastName  string `gorm:"type:varchar(255);not null" json:"last_name"`
	Email     string `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`

	// Encrypted sensitive fields
	Password    security.EncryptedString `gorm:"type:text" json:"-"` // Never expose in JSON
	SSN         security.EncryptedString `gorm:"type:text" json:"-"` // Never expose in JSON
	CreditCard  security.EncryptedString `gorm:"type:text" json:"-"` // Never expose in JSON
	PhoneNumber security.EncryptedString `gorm:"type:text" json:"-"` // Never expose in JSON
	Address     security.EncryptedString `gorm:"type:text" json:"-"` // Never expose in JSON

	// Non-sensitive fields
	Status      UserStatus `gorm:"type:varchar(50);not null;default:'active'" json:"status"`
	Role        UserRole   `gorm:"type:varchar(50);not null;default:'user'" json:"role"`
	LastLoginAt *time.Time `gorm:"type:timestamp" json:"last_login_at,omitempty"`
}

// TableName returns the table name for EncryptedUser
func (u *EncryptedUser) TableName() string { // satisfies models.Modelable
	return "encrypted_users"
}

// SetPassword sets the encrypted password
func (u *EncryptedUser) SetPassword(plaintextPassword string) error {
	return u.Password.Set(plaintextPassword)
}

// GetPassword returns the decrypted password
func (u *EncryptedUser) GetPassword() (string, error) {
	return u.Password.Get()
}

// SetSSN sets the encrypted SSN
func (u *EncryptedUser) SetSSN(plaintextSSN string) error {
	return u.SSN.Set(plaintextSSN)
}

// GetSSN returns the decrypted SSN
func (u *EncryptedUser) GetSSN() (string, error) {
	return u.SSN.Get()
}

// SetCreditCard sets the encrypted credit card number
func (u *EncryptedUser) SetCreditCard(plaintextCard string) error {
	return u.CreditCard.Set(plaintextCard)
}

// GetCreditCard returns the decrypted credit card number
func (u *EncryptedUser) GetCreditCard() (string, error) {
	return u.CreditCard.Get()
}

// SetPhoneNumber sets the encrypted phone number
func (u *EncryptedUser) SetPhoneNumber(plaintextPhone string) error {
	return u.PhoneNumber.Set(plaintextPhone)
}

// GetPhoneNumber returns the decrypted phone number
func (u *EncryptedUser) GetPhoneNumber() (string, error) {
	return u.PhoneNumber.Get()
}

// SetAddress sets the encrypted address
func (u *EncryptedUser) SetAddress(plaintextAddress string) error {
	return u.Address.Set(plaintextAddress)
}

// GetAddress returns the decrypted address
func (u *EncryptedUser) GetAddress() (string, error) {
	return u.Address.Get()
}

// GetFullName returns the user's full name
func (u *EncryptedUser) GetFullName() string {
	return u.FirstName + " " + u.LastName
}

// IsActive returns true if the user is active
func (u *EncryptedUser) IsActive() bool {
	return u.Status == UserStatusActive
}

// IsAdmin returns true if the user is an admin
func (u *EncryptedUser) IsAdmin() bool {
	return u.Role == UserRoleAdmin
}

// BeforeCreate GORM hook
func (u *EncryptedUser) BeforeCreate(tx *gorm.DB) error {
	// Set default values
	if u.Status == "" {
		u.Status = UserStatusActive
	}
	if u.Role == "" {
		u.Role = UserRoleUser
	}
	return nil
}

// BeforeUpdate GORM hook
func (u *EncryptedUser) BeforeUpdate(tx *gorm.DB) error {
	// Any pre-update logic can go here
	return nil
}

// AfterFind GORM hook
func (u *EncryptedUser) AfterFind(tx *gorm.DB) error {
	// Any post-find logic can go here
	return nil
}

// EncryptedUserRepository provides repository operations for EncryptedUser
type EncryptedUserRepository struct {
	*repositories.BaseRepository[*EncryptedUser] //nolint:gocritic
	encryptor                                    *security.FieldEncryptor
}

// NewEncryptedUserRepository creates a new encrypted user repository
func NewEncryptedUserRepository(
	db *gorm.DB,
	logger logging.Logger,
	encryptor *security.FieldEncryptor,
	opts repositories.RepositoryOptions,
) *EncryptedUserRepository {
	return &EncryptedUserRepository{
		BaseRepository: repositories.NewBaseRepository[*EncryptedUser](db, logger, opts),
		encryptor:      encryptor,
	}
}

// CreateWithEncryption creates a new encrypted user with automatic encryption
func (r *EncryptedUserRepository) CreateWithEncryption(ctx context.Context, user *EncryptedUser) error {
	// Encrypt sensitive fields before saving
	if err := r.encryptSensitiveFields(user); err != nil {
		return err
	}

	return r.Create(ctx, user)
}

// UpdateWithEncryption updates an encrypted user with automatic encryption
func (r *EncryptedUserRepository) UpdateWithEncryption(ctx context.Context, user *EncryptedUser) error {
	// Encrypt sensitive fields before updating
	if err := r.encryptSensitiveFields(user); err != nil {
		return err
	}

	return r.Update(ctx, user)
}

// GetByIDWithDecryption retrieves an encrypted user with automatic decryption
func (r *EncryptedUserRepository) GetByIDWithDecryption(ctx context.Context, id string) (*EncryptedUser, error) {
	user, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Decrypt sensitive fields after retrieval
	if err := r.decryptSensitiveFields(user); err != nil {
		return nil, err
	}

	return user, nil
}

// FindWithDecryption retrieves encrypted users with automatic decryption
func (r *EncryptedUserRepository) FindWithDecryption(ctx context.Context, filter repositories.Filter) ([]*EncryptedUser, error) {
	users, err := r.Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Decrypt sensitive fields for all users
	result := make([]*EncryptedUser, len(users))
	for i, user := range users {
		if err := r.decryptSensitiveFields(user); err != nil {
			return nil, err
		}
		result[i] = user
	}

	return result, nil
}

// encryptSensitiveFields encrypts all sensitive fields in the user
func (r *EncryptedUserRepository) encryptSensitiveFields(user *EncryptedUser) error {
	// Password is already handled by EncryptedString
	// SSN is already handled by EncryptedString
	// CreditCard is already handled by EncryptedString
	// PhoneNumber is already handled by EncryptedString
	// Address is already handled by EncryptedString

	// The encryption is handled automatically by the EncryptedString type
	// when the Set() method is called
	return nil
}

// decryptSensitiveFields decrypts all sensitive fields in the user
func (r *EncryptedUserRepository) decryptSensitiveFields(_ *EncryptedUser) error {
	// The decryption is handled automatically by the EncryptedString type
	// when the Get() method is called
	return nil
}

// FindByEmailWithDecryption finds a user by email with decryption
func (r *EncryptedUserRepository) FindByEmailWithDecryption(ctx context.Context, email string) (*EncryptedUser, error) {
	filter := repositories.Filter{
		Where: map[string]repositories.WhereCondition{
			"email": {Operator: "=", Value: email},
		},
	}

	users, err := r.FindWithDecryption(ctx, filter)
	if err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return nil, errors.NewDBError(errors.ErrCodeRecordNotFound, "user not found", nil)
	}

	return users[0], nil
}

// ValidatePassword validates a password against the stored encrypted password
func (r *EncryptedUserRepository) ValidatePassword(ctx context.Context, userID, plaintextPassword string) (bool, error) {
	user, err := r.GetByIDWithDecryption(ctx, userID)
	if err != nil {
		return false, err
	}

	storedPassword, err := user.GetPassword()
	if err != nil {
		return false, err
	}

	return storedPassword == plaintextPassword, nil
}

// UpdatePassword updates the user's password with encryption
func (r *EncryptedUserRepository) UpdatePassword(ctx context.Context, userID, newPassword string) error {
	user, err := r.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if err := user.SetPassword(newPassword); err != nil {
		return err
	}

	return r.Update(ctx, user)
}

// GetSensitiveData returns a map of sensitive data (for authorized access only)
func (r *EncryptedUserRepository) GetSensitiveData(ctx context.Context, userID string) (map[string]interface{}, error) {
	user, err := r.GetByIDWithDecryption(ctx, userID)
	if err != nil {
		return nil, err
	}

	sensitiveData := make(map[string]interface{})

	if ssn, err := user.GetSSN(); err == nil {
		sensitiveData["ssn"] = ssn
	}

	if creditCard, err := user.GetCreditCard(); err == nil {
		sensitiveData["credit_card"] = creditCard
	}

	if phone, err := user.GetPhoneNumber(); err == nil {
		sensitiveData["phone_number"] = phone
	}

	if address, err := user.GetAddress(); err == nil {
		sensitiveData["address"] = address
	}

	return sensitiveData, nil
}

// UpdateSensitiveData updates sensitive data with encryption
func (r *EncryptedUserRepository) UpdateSensitiveData(ctx context.Context, userID string, sensitiveData map[string]string) error {
	user, err := r.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// Update SSN if provided
	if ssn, ok := sensitiveData["ssn"]; ok {
		if err := user.SetSSN(ssn); err != nil {
			return err
		}
	}

	// Update credit card if provided
	if creditCard, ok := sensitiveData["credit_card"]; ok {
		if err := user.SetCreditCard(creditCard); err != nil {
			return err
		}
	}

	// Update phone number if provided
	if phone, ok := sensitiveData["phone_number"]; ok {
		if err := user.SetPhoneNumber(phone); err != nil {
			return err
		}
	}

	// Update address if provided
	if address, ok := sensitiveData["address"]; ok {
		if err := user.SetAddress(address); err != nil {
			return err
		}
	}

	return r.Update(ctx, user)
}
