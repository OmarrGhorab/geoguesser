package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PostgresSessionStore stores refresh-token sessions in the durable auth_sessions table.
type PostgresSessionStore struct {
	db *gorm.DB
}

// NewPostgresSessionStore returns a Postgres-backed session store.
func NewPostgresSessionStore(db *gorm.DB) *PostgresSessionStore {
	return &PostgresSessionStore{db: db}
}

// Create stores a new refresh session. ttl is enforced by ExpiresAt in the row.
func (p *PostgresSessionStore) Create(ctx context.Context, tokenHash string, session *RefreshSession, ttl time.Duration) error {
	row := &AuthSession{
		UserID:           session.UserID,
		RefreshTokenHash: tokenHash,
		ExpiresAt:        session.ExpiresAt,
		CreatedAt:        session.CreatedAt,
	}
	if err := p.db.WithContext(ctx).Create(row).Error; err != nil {
		return fmt.Errorf("failed to create postgres session: %w", err)
	}
	return nil
}

// Get retrieves an active refresh session by token hash.
func (p *PostgresSessionStore) Get(ctx context.Context, tokenHash string) (*RefreshSession, error) {
	var row AuthSession
	if err := p.db.WithContext(ctx).
		Where("refresh_token_hash = ? AND revoked_at IS NULL", tokenHash).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get postgres session: %w", err)
	}

	return &RefreshSession{
		UserID:    row.UserID,
		CreatedAt: row.CreatedAt,
		ExpiresAt: row.ExpiresAt,
	}, nil
}

// Revoke marks a refresh session as revoked.
func (p *PostgresSessionStore) Revoke(ctx context.Context, tokenHash string) error {
	now := time.Now().UTC()
	if err := p.db.WithContext(ctx).
		Model(&AuthSession{}).
		Where("refresh_token_hash = ? AND revoked_at IS NULL", tokenHash).
		Updates(map[string]any{"revoked_at": now, "last_used_at": now}).Error; err != nil {
		return fmt.Errorf("failed to revoke postgres session: %w", err)
	}
	return nil
}

// RevokeAll marks every active refresh session for the user as revoked.
func (p *PostgresSessionStore) RevokeAll(ctx context.Context, userID uuid.UUID) error {
	now := time.Now().UTC()
	if err := p.db.WithContext(ctx).
		Model(&AuthSession{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Updates(map[string]any{"revoked_at": now, "last_used_at": now}).Error; err != nil {
		return fmt.Errorf("failed to revoke all postgres sessions: %w", err)
	}
	return nil
}

// MarkRevoked records revoked-token metadata for reuse detection.
func (p *PostgresSessionStore) MarkRevoked(ctx context.Context, tokenHash string, userID uuid.UUID, ttl time.Duration) error {
	now := time.Now().UTC()
	if err := p.db.WithContext(ctx).
		Model(&AuthSession{}).
		Where("refresh_token_hash = ? AND user_id = ? AND revoked_at IS NULL", tokenHash, userID).
		Updates(map[string]any{"revoked_at": now, "last_used_at": now}).Error; err != nil {
		return fmt.Errorf("failed to mark postgres session revoked: %w", err)
	}
	return nil
}

// RevokedUserID returns the owning user for a revoked refresh-token hash.
func (p *PostgresSessionStore) RevokedUserID(ctx context.Context, tokenHash string) (uuid.UUID, bool, error) {
	var row AuthSession
	if err := p.db.WithContext(ctx).
		Select("user_id").
		Where("refresh_token_hash = ? AND revoked_at IS NOT NULL", tokenHash).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.Nil, false, nil
		}
		return uuid.Nil, false, fmt.Errorf("failed to check revoked postgres session: %w", err)
	}
	return row.UserID, true, nil
}
