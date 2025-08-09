// Package repositories provides example repositories using go-ormx core functionality
package repositories

import (
	"context"

	exampleModels "go-ormx/examples/models"
	"go-ormx/ormx/logging"
	"go-ormx/ormx/models"
	"go-ormx/ormx/repositories"

	"gorm.io/gorm"
)

// UserRepository provides user-specific repository operations
type UserRepository struct {
	*repositories.BaseRepository[*exampleModels.User]
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB, logger logging.Logger, options repositories.RepositoryOptions) *UserRepository {
	baseRepo := repositories.NewBaseRepository[*exampleModels.User](db, logger, options)
	return &UserRepository{
		BaseRepository: baseRepo,
	}
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*exampleModels.User, error) {
	filter := repositories.Filter{
		Where: map[string]repositories.WhereCondition{
			"email": {Operator: "eq", Value: email},
		},
	}
	return r.FindOne(ctx, filter)
}

// GetByUsername retrieves a user by username
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*exampleModels.User, error) {
	filter := repositories.Filter{
		Where: map[string]repositories.WhereCondition{
			"username": {Operator: "eq", Value: username},
		},
	}
	return r.FindOne(ctx, filter)
}

// GetByEmailAndTenant retrieves a user by email within a specific tenant
func (r *UserRepository) GetByEmailAndTenant(ctx context.Context, email, tenantID string) (*exampleModels.User, error) {
	filter := repositories.Filter{
		Where: map[string]repositories.WhereCondition{
			"email":     {Operator: "eq", Value: email},
			"tenant_id": {Operator: "eq", Value: tenantID},
		},
	}
	return r.FindOne(ctx, filter)
}

// GetByUsernameAndTenant retrieves a user by username within a specific tenant
func (r *UserRepository) GetByUsernameAndTenant(ctx context.Context, username, tenantID string) (*exampleModels.User, error) {
	filter := repositories.Filter{
		Where: map[string]repositories.WhereCondition{
			"username":  {Operator: "eq", Value: username},
			"tenant_id": {Operator: "eq", Value: tenantID},
		},
	}
	return r.FindOne(ctx, filter)
}

// GetActiveUsers retrieves all active users
func (r *UserRepository) GetActiveUsers(ctx context.Context) ([]*exampleModels.User, error) {
	filter := repositories.Filter{
		Where: map[string]repositories.WhereCondition{
			"status": {Operator: "eq", Value: exampleModels.UserStatusActive},
		},
	}
	return r.Find(ctx, filter)
}

// GetActiveUsersByTenant retrieves all active users within a specific tenant
func (r *UserRepository) GetActiveUsersByTenant(ctx context.Context, tenantID string) ([]*exampleModels.User, error) {
	filter := repositories.Filter{
		Where: map[string]repositories.WhereCondition{
			"status":    {Operator: "eq", Value: exampleModels.UserStatusActive},
			"tenant_id": {Operator: "eq", Value: tenantID},
		},
	}
	return r.Find(ctx, filter)
}

// GetUsersByRole retrieves users by role
func (r *UserRepository) GetUsersByRole(ctx context.Context, role exampleModels.UserRole) ([]*exampleModels.User, error) {
	filter := repositories.Filter{
		Where: map[string]repositories.WhereCondition{
			"role": {Operator: "eq", Value: role},
		},
	}
	return r.Find(ctx, filter)
}

// GetUsersByRoleAndTenant retrieves users by role within a specific tenant
func (r *UserRepository) GetUsersByRoleAndTenant(ctx context.Context, role exampleModels.UserRole, tenantID string) ([]*exampleModels.User, error) {
	filter := repositories.Filter{
		Where: map[string]repositories.WhereCondition{
			"role":      {Operator: "eq", Value: role},
			"tenant_id": {Operator: "eq", Value: tenantID},
		},
	}
	return r.Find(ctx, filter)
}

// GetAdmins retrieves all admin users
func (r *UserRepository) GetAdmins(ctx context.Context) ([]*exampleModels.User, error) {
	return r.GetUsersByRole(ctx, exampleModels.UserRoleAdmin)
}

// GetAdminsByTenant retrieves all admin users within a specific tenant
func (r *UserRepository) GetAdminsByTenant(ctx context.Context, tenantID string) ([]*exampleModels.User, error) {
	return r.GetUsersByRoleAndTenant(ctx, exampleModels.UserRoleAdmin, tenantID)
}

// GetModerators retrieves all moderator users
func (r *UserRepository) GetModerators(ctx context.Context) ([]*exampleModels.User, error) {
	return r.GetUsersByRole(ctx, exampleModels.UserRoleModerator)
}

// GetModeratorsByTenant retrieves all moderator users within a specific tenant
func (r *UserRepository) GetModeratorsByTenant(ctx context.Context, tenantID string) ([]*exampleModels.User, error) {
	return r.GetUsersByRoleAndTenant(ctx, exampleModels.UserRoleModerator, tenantID)
}

// GetLockedUsers retrieves all locked users
func (r *UserRepository) GetLockedUsers(ctx context.Context) ([]*exampleModels.User, error) {
	filter := repositories.Filter{
		Where: map[string]repositories.WhereCondition{
			"locked_until": {Operator: "is_not_null", Value: nil},
		},
	}
	return r.Find(ctx, filter)
}

// GetLockedUsersByTenant retrieves all locked users within a specific tenant
func (r *UserRepository) GetLockedUsersByTenant(ctx context.Context, tenantID string) ([]*exampleModels.User, error) {
	filter := repositories.Filter{
		Where: map[string]repositories.WhereCondition{
			"locked_until": {Operator: "is_not_null", Value: nil},
			"tenant_id":    {Operator: "eq", Value: tenantID},
		},
	}
	return r.Find(ctx, filter)
}

// GetUsersWithTwoFactor retrieves users with two-factor authentication enabled
func (r *UserRepository) GetUsersWithTwoFactor(ctx context.Context) ([]*exampleModels.User, error) {
	filter := repositories.Filter{
		Where: map[string]repositories.WhereCondition{
			"two_factor_enabled": {Operator: "eq", Value: true},
		},
	}
	return r.Find(ctx, filter)
}

// GetUsersWithTwoFactorByTenant retrieves users with two-factor authentication enabled within a specific tenant
func (r *UserRepository) GetUsersWithTwoFactorByTenant(ctx context.Context, tenantID string) ([]*exampleModels.User, error) {
	filter := repositories.Filter{
		Where: map[string]repositories.WhereCondition{
			"two_factor_enabled": {Operator: "eq", Value: true},
			"tenant_id":          {Operator: "eq", Value: tenantID},
		},
	}
	return r.Find(ctx, filter)
}

// GetUsersByStatus retrieves users by status
func (r *UserRepository) GetUsersByStatus(ctx context.Context, status exampleModels.UserStatus) ([]*exampleModels.User, error) {
	filter := repositories.Filter{
		Where: map[string]repositories.WhereCondition{
			"status": {Operator: "eq", Value: status},
		},
	}
	return r.Find(ctx, filter)
}

// GetUsersByStatusAndTenant retrieves users by status within a specific tenant
func (r *UserRepository) GetUsersByStatusAndTenant(ctx context.Context, status exampleModels.UserStatus, tenantID string) ([]*exampleModels.User, error) {
	filter := repositories.Filter{
		Where: map[string]repositories.WhereCondition{
			"status":    {Operator: "eq", Value: status},
			"tenant_id": {Operator: "eq", Value: tenantID},
		},
	}
	return r.Find(ctx, filter)
}

// SearchUsers searches users by name, email, or username
func (r *UserRepository) SearchUsers(ctx context.Context, query string) ([]*exampleModels.User, error) {
	filter := repositories.Filter{
		Where: map[string]repositories.WhereCondition{
			"first_name": {Operator: "like", Value: "%" + query + "%"},
		},
		Scopes: []models.Scope{
			func(db *gorm.DB) *gorm.DB {
				return db.Or("last_name LIKE ?", "%"+query+"%").
					Or("email LIKE ?", "%"+query+"%").
					Or("username LIKE ?", "%"+query+"%")
			},
		},
	}
	return r.Find(ctx, filter)
}

// SearchUsersByTenant searches users by name, email, or username within a specific tenant
func (r *UserRepository) SearchUsersByTenant(ctx context.Context, query, tenantID string) ([]*exampleModels.User, error) {
	filter := repositories.Filter{
		Where: map[string]repositories.WhereCondition{
			"tenant_id": {Operator: "eq", Value: tenantID},
		},
		Scopes: []models.Scope{
			func(db *gorm.DB) *gorm.DB {
				return db.Where("first_name LIKE ? OR last_name LIKE ? OR email LIKE ? OR username LIKE ?",
					"%"+query+"%", "%"+query+"%", "%"+query+"%", "%"+query+"%")
			},
		},
	}
	return r.Find(ctx, filter)
}

// GetUserCount returns the total number of users
func (r *UserRepository) GetUserCount(ctx context.Context) (int64, error) {
	filter := repositories.Filter{}
	return r.Count(ctx, filter)
}

// GetUserCountByTenant returns the total number of users within a specific tenant
func (r *UserRepository) GetUserCountByTenant(ctx context.Context, tenantID string) (int64, error) {
	filter := repositories.Filter{
		Where: map[string]repositories.WhereCondition{
			"tenant_id": {Operator: "eq", Value: tenantID},
		},
	}
	return r.Count(ctx, filter)
}

// GetActiveUserCount returns the total number of active users
func (r *UserRepository) GetActiveUserCount(ctx context.Context) (int64, error) {
	filter := repositories.Filter{
		Where: map[string]repositories.WhereCondition{
			"status": {Operator: "eq", Value: exampleModels.UserStatusActive},
		},
	}
	return r.Count(ctx, filter)
}

// GetActiveUserCountByTenant returns the total number of active users within a specific tenant
func (r *UserRepository) GetActiveUserCountByTenant(ctx context.Context, tenantID string) (int64, error) {
	filter := repositories.Filter{
		Where: map[string]repositories.WhereCondition{
			"status":    {Operator: "eq", Value: exampleModels.UserStatusActive},
			"tenant_id": {Operator: "eq", Value: tenantID},
		},
	}
	return r.Count(ctx, filter)
}

// UpdateUserStatus updates a user's status
func (r *UserRepository) UpdateUserStatus(ctx context.Context, userID string, status exampleModels.UserStatus) error {
	updates := map[string]interface{}{
		"status": status,
	}
	return r.UpdatePartial(ctx, userID, updates)
}

// UpdateUserRole updates a user's role
func (r *UserRepository) UpdateUserRole(ctx context.Context, userID string, role exampleModels.UserRole) error {
	updates := map[string]interface{}{
		"role": role,
	}
	return r.UpdatePartial(ctx, userID, updates)
}

// LockUser locks a user for a specified duration
func (r *UserRepository) LockUser(ctx context.Context, userID string, duration int) error {
	updates := map[string]interface{}{
		"locked_until": gorm.Expr("NOW() + INTERVAL ? SECOND", duration),
	}
	return r.UpdatePartial(ctx, userID, updates)
}

// UnlockUser unlocks a user
func (r *UserRepository) UnlockUser(ctx context.Context, userID string) error {
	updates := map[string]interface{}{
		"locked_until":   nil,
		"login_attempts": 0,
	}
	return r.UpdatePartial(ctx, userID, updates)
}

// IncrementLoginAttempts increments a user's login attempts
func (r *UserRepository) IncrementLoginAttempts(ctx context.Context, userID string) error {
	updates := map[string]interface{}{
		"login_attempts": gorm.Expr("login_attempts + 1"),
	}
	return r.UpdatePartial(ctx, userID, updates)
}

// ResetLoginAttempts resets a user's login attempts
func (r *UserRepository) ResetLoginAttempts(ctx context.Context, userID string) error {
	updates := map[string]interface{}{
		"login_attempts": 0,
		"locked_until":   nil,
	}
	return r.UpdatePartial(ctx, userID, updates)
}

// UpdateLastLogin updates a user's last login information
func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID, ipAddress string) error {
	updates := map[string]interface{}{
		"last_login_at": gorm.Expr("NOW()"),
		"last_login_ip": ipAddress,
	}
	return r.UpdatePartial(ctx, userID, updates)
}

// MarkEmailVerified marks a user's email as verified
func (r *UserRepository) MarkEmailVerified(ctx context.Context, userID string) error {
	updates := map[string]interface{}{
		"email_verified":    true,
		"email_verified_at": gorm.Expr("NOW()"),
	}
	return r.UpdatePartial(ctx, userID, updates)
}

// MarkPhoneVerified marks a user's phone as verified
func (r *UserRepository) MarkPhoneVerified(ctx context.Context, userID string) error {
	updates := map[string]interface{}{
		"phone_verified":    true,
		"phone_verified_at": gorm.Expr("NOW()"),
	}
	return r.UpdatePartial(ctx, userID, updates)
}

// EnableTwoFactor enables two-factor authentication for a user
func (r *UserRepository) EnableTwoFactor(ctx context.Context, userID, secret string) error {
	updates := map[string]interface{}{
		"two_factor_enabled": true,
		"two_factor_secret":  secret,
	}
	return r.UpdatePartial(ctx, userID, updates)
}

// DisableTwoFactor disables two-factor authentication for a user
func (r *UserRepository) DisableTwoFactor(ctx context.Context, userID string) error {
	updates := map[string]interface{}{
		"two_factor_enabled":  false,
		"two_factor_secret":   "",
		"recovery_codes_hash": "",
	}
	return r.UpdatePartial(ctx, userID, updates)
}

// SetRecoveryCodes sets recovery codes for a user
func (r *UserRepository) SetRecoveryCodes(ctx context.Context, userID, hash string) error {
	updates := map[string]interface{}{
		"recovery_codes_hash": hash,
	}
	return r.UpdatePartial(ctx, userID, updates)
}

// UpdatePassword updates a user's password
func (r *UserRepository) UpdatePassword(ctx context.Context, userID, passwordHash, salt string) error {
	updates := map[string]interface{}{
		"password_hash":       passwordHash,
		"salt":                salt,
		"password_changed_at": gorm.Expr("NOW()"),
	}
	return r.UpdatePartial(ctx, userID, updates)
}

// WithTx returns a new repository instance with the given transaction
func (r *UserRepository) WithTx(tx *gorm.DB) *UserRepository {
	baseRepo := r.BaseRepository.WithTx(tx)
	return &UserRepository{
		BaseRepository: baseRepo.(*repositories.BaseRepository[*exampleModels.User]),
	}
}
