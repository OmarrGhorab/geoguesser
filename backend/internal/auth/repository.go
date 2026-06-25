package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository owns auth-related persistence queries.
type Repository struct {
	db *gorm.DB
}

// NewRepository returns a new auth repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// CreateUser inserts a new registered account.
func (r *Repository) CreateUser(ctx context.Context, tx *gorm.DB, user *User) error {
	db := r.dbOrTx(tx)
	if err := db.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// GetUserByEmail returns a user by email address.
func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return &user, nil
}

// GetUserByID returns a user by id.
func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}
	return &user, nil
}

// CreateProfile inserts a user profile.
func (r *Repository) CreateProfile(ctx context.Context, tx *gorm.DB, profile *UserProfile) error {
	db := r.dbOrTx(tx)
	if err := db.WithContext(ctx).Create(profile).Error; err != nil {
		return fmt.Errorf("failed to create profile: %w", err)
	}
	return nil
}

// GetProfileByUserID returns a profile by user id.
func (r *Repository) GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*UserProfile, error) {
	var profile UserProfile
	if err := r.db.WithContext(ctx).First(&profile, "user_id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}
	return &profile, nil
}

// UpdateUserLastLogin updates the user's last login timestamp.
func (r *Repository) UpdateUserLastLogin(ctx context.Context, tx *gorm.DB, userID uuid.UUID, t time.Time) error {
	db := r.dbOrTx(tx)
	if err := db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Update("last_login_at", t).Error; err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}
	return nil
}

// CreateOAuthAccount inserts an OAuth connection.
func (r *Repository) CreateOAuthAccount(ctx context.Context, tx *gorm.DB, account *UserOAuthAccount) error {
	db := r.dbOrTx(tx)
	if err := db.WithContext(ctx).Create(account).Error; err != nil {
		return fmt.Errorf("failed to create oauth account: %w", err)
	}
	return nil
}

// GetOAuthAccount returns an OAuth account by provider and provider account id.
func (r *Repository) GetOAuthAccount(ctx context.Context, provider OAuthProvider, providerAccountID string) (*UserOAuthAccount, error) {
	var account UserOAuthAccount
	if err := r.db.WithContext(ctx).Where("provider = ? AND provider_account_id = ?", string(provider), providerAccountID).First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get oauth account: %w", err)
	}
	return &account, nil
}

// GetOAuthAccountByUserAndProvider returns an OAuth account by user and provider.
func (r *Repository) GetOAuthAccountByUserAndProvider(ctx context.Context, userID uuid.UUID, provider OAuthProvider) (*UserOAuthAccount, error) {
	var account UserOAuthAccount
	if err := r.db.WithContext(ctx).Where("user_id = ? AND provider = ?", userID, string(provider)).First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get oauth account by user: %w", err)
	}
	return &account, nil
}

// UpdateOAuthTokens updates OAuth token metadata for an account.
func (r *Repository) UpdateOAuthTokens(ctx context.Context, tx *gorm.DB, accountID uuid.UUID, accessToken, refreshToken *string, expiresAt *time.Time) error {
	db := r.dbOrTx(tx)
	updates := map[string]any{}
	if accessToken != nil {
		updates["access_token"] = *accessToken
	}
	if refreshToken != nil {
		updates["refresh_token"] = *refreshToken
	}
	if expiresAt != nil {
		updates["expires_at"] = *expiresAt
	}
	if len(updates) == 0 {
		return nil
	}
	if err := db.WithContext(ctx).Model(&UserOAuthAccount{}).Where("id = ?", accountID).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update oauth tokens: %w", err)
	}
	return nil
}

func (r *Repository) dbOrTx(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return r.db
}
