// Package models provides example models using go-ormx core functionality
package models

import (
	"fmt"
	"time"

	"go-ormx/ormx/models"

	"gorm.io/gorm"
)

// TenantStatus represents the status of a tenant
type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusInactive  TenantStatus = "inactive"
	TenantStatusSuspended TenantStatus = "suspended"
	TenantStatusPending   TenantStatus = "pending"
)

// TenantPlan represents the plan of a tenant
type TenantPlan string

const (
	TenantPlanFree       TenantPlan = "free"
	TenantPlanBasic      TenantPlan = "basic"
	TenantPlanPro        TenantPlan = "pro"
	TenantPlanEnterprise TenantPlan = "enterprise"
)

// Tenant represents a tenant entity
type Tenant struct {
	models.AuditModel

	Name               string       `gorm:"type:varchar(255);not null" json:"name" validate:"required,min=1,max=255"`
	Slug               string       `gorm:"type:varchar(100);uniqueIndex;not null" json:"slug" validate:"required,min=1,max=100"`
	Domain             string       `gorm:"type:varchar(255)" json:"domain,omitempty"`
	Status             TenantStatus `gorm:"type:varchar(20);not null;default:'pending'" json:"status" validate:"required,oneof=active inactive suspended pending"`
	Plan               TenantPlan   `gorm:"type:varchar(20);not null;default:'free'" json:"plan" validate:"required,oneof=free basic pro enterprise"`
	Description        string       `gorm:"type:text" json:"description,omitempty"`
	Logo               string       `gorm:"type:text" json:"logo,omitempty"`
	Website            string       `gorm:"type:varchar(255)" json:"website,omitempty"`
	ContactEmail       string       `gorm:"type:varchar(255)" json:"contact_email,omitempty"`
	ContactPhone       string       `gorm:"type:varchar(20)" json:"contact_phone,omitempty"`
	Address            string       `gorm:"type:text" json:"address,omitempty"`
	Country            string       `gorm:"type:varchar(2)" json:"country,omitempty"`
	Timezone           string       `gorm:"type:varchar(50);default:'UTC'" json:"timezone"`
	Locale             string       `gorm:"type:varchar(10);default:'en'" json:"locale"`
	Currency           string       `gorm:"type:varchar(3);default:'USD'" json:"currency"`
	MaxUsers           int          `gorm:"not null;default:1" json:"max_users"`
	MaxStorage         int64        `gorm:"not null;default:1073741824" json:"max_storage"` // 1GB in bytes
	MaxProjects        int          `gorm:"not null;default:1" json:"max_projects"`
	Settings           string       `gorm:"type:json" json:"settings,omitempty"`
	Metadata           string       `gorm:"type:json" json:"metadata,omitempty"`
	SubscriptionID     string       `gorm:"type:varchar(255)" json:"subscription_id,omitempty"`
	SubscriptionStatus string       `gorm:"type:varchar(50)" json:"subscription_status,omitempty"`
	BillingEmail       string       `gorm:"type:varchar(255)" json:"billing_email,omitempty"`
	TrialEndsAt        *time.Time   `gorm:"type:timestamp" json:"trial_ends_at,omitempty"`
	PlanExpiresAt      *time.Time   `gorm:"type:timestamp" json:"plan_expires_at,omitempty"`
	LastActivityAt     *time.Time   `gorm:"type:timestamp" json:"last_activity_at,omitempty"`
	EmailVerified      bool         `gorm:"not null;default:false" json:"email_verified"`
	EmailVerifiedAt    *time.Time   `gorm:"type:timestamp" json:"email_verified_at,omitempty"`
	PhoneVerified      bool         `gorm:"not null;default:false" json:"phone_verified"`
	PhoneVerifiedAt    *time.Time   `gorm:"type:timestamp" json:"phone_verified_at,omitempty"`
	TwoFactorRequired  bool         `gorm:"not null;default:false" json:"two_factor_required"`
	PasswordPolicy     string       `gorm:"type:json" json:"password_policy,omitempty"`
	SessionTimeout     int          `gorm:"not null;default:3600" json:"session_timeout"` // in seconds
	MaxLoginAttempts   int          `gorm:"not null;default:5" json:"max_login_attempts"`
	LockoutDuration    int          `gorm:"not null;default:900" json:"lockout_duration"` // in seconds
	IPWhitelist        string       `gorm:"type:json" json:"ip_whitelist,omitempty"`
	IPBlacklist        string       `gorm:"type:json" json:"ip_blacklist,omitempty"`
	AllowedDomains     string       `gorm:"type:json" json:"allowed_domains,omitempty"`
	CustomBranding     bool         `gorm:"not null;default:false" json:"custom_branding"`
	SSOEnabled         bool         `gorm:"not null;default:false" json:"sso_enabled"`
	SSOProvider        string       `gorm:"type:varchar(50)" json:"sso_provider,omitempty"`
	SSOConfig          string       `gorm:"type:json" json:"sso_config,omitempty"`
	APIEnabled         bool         `gorm:"not null;default:false" json:"api_enabled"`
	APIKey             string       `gorm:"type:varchar(255)" json:"-"` // API key hash
	WebhookURL         string       `gorm:"type:varchar(255)" json:"webhook_url,omitempty"`
	WebhookSecret      string       `gorm:"type:varchar(255)" json:"-"` // Webhook secret hash
	AnalyticsEnabled   bool         `gorm:"not null;default:true" json:"analytics_enabled"`
	BackupEnabled      bool         `gorm:"not null;default:false" json:"backup_enabled"`
	BackupFrequency    string       `gorm:"type:varchar(20);default:'daily'" json:"backup_frequency"`
	RetentionDays      int          `gorm:"not null;default:30" json:"retention_days"`
}

// TableName returns the table name for Tenant
func (t *Tenant) TableName() string {
	return "tenants"
}

// IsActive returns true if the tenant is active
func (t *Tenant) IsActive() bool {
	return t.Status == TenantStatusActive
}

// IsSuspended returns true if the tenant is suspended
func (t *Tenant) IsSuspended() bool {
	return t.Status == TenantStatusSuspended
}

// IsInactive returns true if the tenant is inactive
func (t *Tenant) IsInactive() bool {
	return t.Status == TenantStatusInactive
}

// IsPending returns true if the tenant is pending
func (t *Tenant) IsPending() bool {
	return t.Status == TenantStatusPending
}

// IsTrialActive returns true if the tenant is in trial period
func (t *Tenant) IsTrialActive() bool {
	if t.TrialEndsAt == nil {
		return false
	}
	return time.Now().Before(*t.TrialEndsAt)
}

// IsPlanExpired returns true if the tenant's plan has expired
func (t *Tenant) IsPlanExpired() bool {
	if t.PlanExpiresAt == nil {
		return false
	}
	return time.Now().After(*t.PlanExpiresAt)
}

// HasPlan returns true if the tenant has a specific plan
func (t *Tenant) HasPlan(plan TenantPlan) bool {
	return t.Plan == plan
}

// IsFreePlan returns true if the tenant is on free plan
func (t *Tenant) IsFreePlan() bool {
	return t.HasPlan(TenantPlanFree)
}

// IsPaidPlan returns true if the tenant is on a paid plan
func (t *Tenant) IsPaidPlan() bool {
	return !t.IsFreePlan()
}

// CanAddUser checks if the tenant can add more users
func (t *Tenant) CanAddUser(currentUserCount int) bool {
	return currentUserCount < t.MaxUsers
}

// CanAddProject checks if the tenant can add more projects
func (t *Tenant) CanAddProject(currentProjectCount int) bool {
	return currentProjectCount < t.MaxProjects
}

// CanUseStorage checks if the tenant can use more storage
func (t *Tenant) CanUseStorage(currentStorage int64) bool {
	return currentStorage < t.MaxStorage
}

// UpdateLastActivity updates the last activity timestamp
func (t *Tenant) UpdateLastActivity() {
	now := time.Now()
	t.LastActivityAt = &now
}

// MarkEmailVerified marks the tenant's email as verified
func (t *Tenant) MarkEmailVerified() {
	t.EmailVerified = true
	now := time.Now()
	t.EmailVerifiedAt = &now
}

// MarkPhoneVerified marks the tenant's phone as verified
func (t *Tenant) MarkPhoneVerified() {
	t.PhoneVerified = true
	now := time.Now()
	t.PhoneVerifiedAt = &now
}

// EnableSSO enables SSO for the tenant
func (t *Tenant) EnableSSO(provider string, config string) {
	t.SSOEnabled = true
	t.SSOProvider = provider
	t.SSOConfig = config
}

// DisableSSO disables SSO for the tenant
func (t *Tenant) DisableSSO() {
	t.SSOEnabled = false
	t.SSOProvider = ""
	t.SSOConfig = ""
}

// EnableAPI enables API access for the tenant
func (t *Tenant) EnableAPI() {
	t.APIEnabled = true
}

// DisableAPI disables API access for the tenant
func (t *Tenant) DisableAPI() {
	t.APIEnabled = false
	t.APIKey = ""
}

// SetAPIKey sets the API key hash
func (t *Tenant) SetAPIKey(keyHash string) {
	t.APIKey = keyHash
}

// SetWebhookSecret sets the webhook secret hash
func (t *Tenant) SetWebhookSecret(secretHash string) {
	t.WebhookSecret = secretHash
}

// String returns a string representation of the tenant
func (t *Tenant) String() string {
	return fmt.Sprintf("Tenant{ID: %s, Name: %s, Slug: %s, Status: %s, Plan: %s}",
		t.ID, t.Name, t.Slug, t.Status, t.Plan)
}

// BeforeCreate GORM hook
func (t *Tenant) BeforeCreate(tx *gorm.DB) error {
	// Call parent BeforeCreate
	if err := t.AuditModel.BeforeCreate(tx); err != nil {
		return err
	}

	// Set default values
	if t.Status == "" {
		t.Status = TenantStatusPending
	}
	if t.Plan == "" {
		t.Plan = TenantPlanFree
	}
	if t.Timezone == "" {
		t.Timezone = "UTC"
	}
	if t.Locale == "" {
		t.Locale = "en"
	}
	if t.Currency == "" {
		t.Currency = "USD"
	}
	if t.MaxUsers <= 0 {
		t.MaxUsers = 1
	}
	if t.MaxStorage <= 0 {
		t.MaxStorage = 1073741824 // 1GB
	}
	if t.MaxProjects <= 0 {
		t.MaxProjects = 1
	}
	if t.SessionTimeout <= 0 {
		t.SessionTimeout = 3600 // 1 hour
	}
	if t.MaxLoginAttempts <= 0 {
		t.MaxLoginAttempts = 5
	}
	if t.LockoutDuration <= 0 {
		t.LockoutDuration = 900 // 15 minutes
	}
	if t.RetentionDays <= 0 {
		t.RetentionDays = 30
	}

	return nil
}

// BeforeUpdate GORM hook
func (t *Tenant) BeforeUpdate(tx *gorm.DB) error {
	// Call parent BeforeUpdate
	return t.AuditModel.BeforeUpdate(tx)
}

// AfterFind GORM hook
func (t *Tenant) AfterFind(tx *gorm.DB) error {
	// Any post-processing after finding a tenant
	return nil
}
