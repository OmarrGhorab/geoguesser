package auth

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User is the registered account model.
type User struct {
	ID              uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Email           string     `gorm:"type:citext;not null;uniqueIndex:users_email_key"`
	PasswordHash    *string    `gorm:"type:text"`
	Role            string     `gorm:"type:text;not null;default:'user'"`
	Status          string     `gorm:"type:text;not null;default:'pending_verification'"`
	EmailVerifiedAt *time.Time `gorm:"type:timestamptz"`
	LastLoginAt     *time.Time `gorm:"type:timestamptz"`
	CreatedAt       time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt       time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

// TableName returns the table name for the model.
func (User) TableName() string {
	return "users"
}

// UserProfile is the public profile for a registered account.
type UserProfile struct {
	UserID      uuid.UUID  `gorm:"type:uuid;primary_key"`
	DisplayName string     `gorm:"type:text;not null"`
	AvatarURL   *string    `gorm:"type:text"`
	CountryCode *string    `gorm:"type:text"`
	Locale      string     `gorm:"type:text;not null;default:'en'"`
	Timezone    *string    `gorm:"type:text"`
	CreatedAt   time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt   time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

// TableName returns the table name for the model.
func (UserProfile) TableName() string {
	return "user_profiles"
}

// AuthSession is a refresh-token session for a registered user.
type AuthSession struct {
	ID               uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID           uuid.UUID  `gorm:"type:uuid;not null;index:auth_sessions_user_id_expires_at"`
	RefreshTokenHash string     `gorm:"type:text;not null;uniqueIndex:auth_sessions_refresh_token_hash_key"`
	UserAgentHash    *string    `gorm:"type:text"`
	IPAddress        *string    `gorm:"type:inet"`
	ExpiresAt        time.Time  `gorm:"type:timestamptz;not null"`
	RevokedAt        *time.Time `gorm:"type:timestamptz"`
	CreatedAt        time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	LastUsedAt       *time.Time `gorm:"type:timestamptz"`
}

// TableName returns the table name for the model.
func (AuthSession) TableName() string {
	return "auth_sessions"
}

// UserOAuthAccount links a registered user to an external OAuth provider.
type UserOAuthAccount struct {
	ID                uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID            uuid.UUID  `gorm:"type:uuid;not null;index:user_oauth_accounts_user_id_provider"`
	Provider          string     `gorm:"type:text;not null"`
	ProviderAccountID string     `gorm:"type:text;not null"`
	Email             *string    `gorm:"type:text"`
	DisplayName       *string    `gorm:"type:text"`
	AvatarURL         *string    `gorm:"type:text"`
	AccessToken       *string    `gorm:"type:text"`
	RefreshToken      *string    `gorm:"type:text"`
	ExpiresAt         *time.Time `gorm:"type:timestamptz"`
	CreatedAt         time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt         time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

// TableName returns the table name for the model.
func (UserOAuthAccount) TableName() string {
	return "user_oauth_accounts"
}

// BeforeCreate ensures UUIDs are generated if absent.
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		u.ID = id
	}
	return nil
}

// BeforeCreate ensures UUIDs are generated if absent.
func (s *AuthSession) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		s.ID = id
	}
	return nil
}

// BeforeCreate ensures UUIDs are generated if absent.
func (o *UserOAuthAccount) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		o.ID = id
	}
	return nil
}
