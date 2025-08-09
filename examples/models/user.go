// Package models provides example models using go-ormx core functionality
package models

import (
	"fmt"
	"strings"
	"time"

	"go-ormx/ormx/models"

	"gorm.io/gorm"
)

// UserStatus represents the status of a user
type UserStatus string

const (
	UserStatusPending   UserStatus = "pending"
	UserStatusActive    UserStatus = "active"
	UserStatusInactive  UserStatus = "inactive"
	UserStatusSuspended UserStatus = "suspended"
)

// UserRole represents the role of a user
type UserRole string

const (
	UserRoleUser      UserRole = "user"
	UserRoleModerator UserRole = "moderator"
	UserRoleAdmin     UserRole = "admin"
)

// User represents a user entity
type User struct {
	models.TenantAuditModel

	Email             string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"email" validate:"required,email"`
	Username          string     `gorm:"type:varchar(100);uniqueIndex;not null" json:"username" validate:"required,min=3,max=100"`
	FirstName         string     `gorm:"type:varchar(100);not null" json:"first_name" validate:"required,min=1,max=100"`
	LastName          string     `gorm:"type:varchar(100);not null" json:"last_name" validate:"required,min=1,max=100"`
	PasswordHash      string     `gorm:"type:varchar(255);not null" json:"-" validate:"required"`
	Salt              string     `gorm:"type:varchar(255);not null" json:"-" validate:"required"`
	Status            UserStatus `gorm:"type:varchar(20);not null;default:'pending'" json:"status" validate:"required,oneof=pending active inactive suspended"`
	Role              UserRole   `gorm:"type:varchar(20);not null;default:'user'" json:"role" validate:"required,oneof=user moderator admin"`
	Avatar            string     `gorm:"type:text" json:"avatar,omitempty"`
	Bio               string     `gorm:"type:text" json:"bio,omitempty"`
	Phone             string     `gorm:"type:varchar(20)" json:"phone,omitempty"`
	DateOfBirth       *time.Time `gorm:"type:date" json:"date_of_birth,omitempty"`
	Timezone          string     `gorm:"type:varchar(50);default:'UTC'" json:"timezone"`
	Locale            string     `gorm:"type:varchar(10);default:'en'" json:"locale"`
	EmailVerified     bool       `gorm:"not null;default:false" json:"email_verified"`
	EmailVerifiedAt   *time.Time `gorm:"type:timestamp" json:"email_verified_at,omitempty"`
	PhoneVerified     bool       `gorm:"not null;default:false" json:"phone_verified"`
	PhoneVerifiedAt   *time.Time `gorm:"type:timestamp" json:"phone_verified_at,omitempty"`
	TwoFactorEnabled  bool       `gorm:"not null;default:false" json:"two_factor_enabled"`
	TwoFactorSecret   string     `gorm:"type:varchar(255)" json:"-"`
	RecoveryCodesHash string     `gorm:"type:text" json:"-"`
	LastLoginAt       *time.Time `gorm:"type:timestamp" json:"last_login_at,omitempty"`
	LastLoginIP       string     `gorm:"type:varchar(45)" json:"last_login_ip,omitempty"`
	LoginAttempts     int        `gorm:"not null;default:0" json:"login_attempts"`
	LockedUntil       *time.Time `gorm:"type:timestamp" json:"locked_until,omitempty"`
	PasswordChangedAt *time.Time `gorm:"type:timestamp" json:"password_changed_at,omitempty"`
	Preferences       string     `gorm:"type:json" json:"preferences,omitempty"`
}

// TableName returns the table name for User
func (u *User) TableName() string {
	return "users"
}

// GetFullName returns the full name of the user
func (u *User) GetFullName() string {
	parts := []string{}
	if strings.TrimSpace(u.FirstName) != "" {
		parts = append(parts, strings.TrimSpace(u.FirstName))
	}
	if strings.TrimSpace(u.LastName) != "" {
		parts = append(parts, strings.TrimSpace(u.LastName))
	}
	return strings.Join(parts, " ")
}

// IsActive returns true if the user is active
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

// IsLocked returns true if the user is locked
func (u *User) IsLocked() bool {
	if u.LockedUntil == nil {
		return false
	}
	return time.Now().Before(*u.LockedUntil)
}

// CanLogin returns true if the user can login
func (u *User) CanLogin() bool {
	return u.IsActive() && !u.IsLocked()
}

// IncrementLoginAttempts increments the login attempts counter
func (u *User) IncrementLoginAttempts() {
	u.LoginAttempts++
}

// ResetLoginAttempts resets the login attempts counter
func (u *User) ResetLoginAttempts() {
	u.LoginAttempts = 0
	u.LockedUntil = nil
}

// Lock locks the user for a specified duration
func (u *User) Lock(duration time.Duration) {
	lockedUntil := time.Now().Add(duration)
	u.LockedUntil = &lockedUntil
}

// Unlock unlocks the user
func (u *User) Unlock() {
	u.LockedUntil = nil
	u.LoginAttempts = 0
}

// MarkEmailVerified marks the user's email as verified
func (u *User) MarkEmailVerified() {
	u.EmailVerified = true
	now := time.Now()
	u.EmailVerifiedAt = &now
}

// MarkPhoneVerified marks the user's phone as verified
func (u *User) MarkPhoneVerified() {
	u.PhoneVerified = true
	now := time.Now()
	u.PhoneVerifiedAt = &now
}

// UpdateLastLogin updates the last login information
func (u *User) UpdateLastLogin(ipAddress string) {
	now := time.Now()
	u.LastLoginAt = &now
	u.LastLoginIP = ipAddress
	u.ResetLoginAttempts()
}

// UpdatePassword updates the password hash and salt
func (u *User) UpdatePassword(passwordHash, salt string) {
	u.PasswordHash = passwordHash
	u.Salt = salt
	now := time.Now()
	u.PasswordChangedAt = &now
}

// EnableTwoFactor enables two-factor authentication
func (u *User) EnableTwoFactor(secret string) {
	u.TwoFactorEnabled = true
	u.TwoFactorSecret = secret
}

// DisableTwoFactor disables two-factor authentication
func (u *User) DisableTwoFactor() {
	u.TwoFactorEnabled = false
	u.TwoFactorSecret = ""
	u.RecoveryCodesHash = ""
}

// SetRecoveryCodes sets the recovery codes hash
func (u *User) SetRecoveryCodes(hash string) {
	u.RecoveryCodesHash = hash
}

// HasRole checks if the user has a specific role
func (u *User) HasRole(role UserRole) bool {
	return u.Role == role
}

// IsAdmin checks if the user is an admin
func (u *User) IsAdmin() bool {
	return u.HasRole(UserRoleAdmin)
}

// IsModerator checks if the user is a moderator
func (u *User) IsModerator() bool {
	return u.HasRole(UserRoleModerator) || u.IsAdmin()
}

// String returns a string representation of the user
func (u *User) String() string {
	return fmt.Sprintf("User{ID: %s, Email: %s, Username: %s, Name: %s, Status: %s, Role: %s}",
		u.ID, u.Email, u.Username, u.GetFullName(), u.Status, u.Role)
}

// BeforeCreate GORM hook
func (u *User) BeforeCreate(tx *gorm.DB) error {
	// Call parent BeforeCreate
	if err := u.TenantAuditModel.BeforeCreate(tx); err != nil {
		return err
	}

	// Set default values
	if u.Status == "" {
		u.Status = UserStatusPending
	}
	if u.Role == "" {
		u.Role = UserRoleUser
	}
	if u.Timezone == "" {
		u.Timezone = "UTC"
	}
	if u.Locale == "" {
		u.Locale = "en"
	}

	return nil
}

// BeforeUpdate GORM hook
func (u *User) BeforeUpdate(tx *gorm.DB) error {
	// Call parent BeforeUpdate
	return u.TenantAuditModel.BeforeUpdate(tx)
}

// AfterFind GORM hook
func (u *User) AfterFind(tx *gorm.DB) error {
	// Any post-processing after finding a user
	return nil
}
