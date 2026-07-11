package friends

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository persists friendship edges and public profile joins.
type Repository struct {
	db *gorm.DB
}

// NewRepository returns a friendships repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

type activeUser struct {
	ID uuid.UUID `gorm:"column:id"`
}

// FindActiveUser returns the user when status is active, otherwise nil.
func (r *Repository) FindActiveUser(ctx context.Context, userID uuid.UUID) (*uuid.UUID, error) {
	var row activeUser
	err := r.db.WithContext(ctx).Raw(`
		SELECT id FROM users WHERE id = ? AND status = 'active'
	`, userID).Scan(&row).Error
	if err != nil {
		return nil, fmt.Errorf("find active user: %w", err)
	}
	if row.ID == uuid.Nil {
		return nil, nil
	}
	return &row.ID, nil
}

// lockUsersInOrder locks the two user rows in sorted UUID order to prevent deadlocks.
func (r *Repository) lockUsersInOrder(tx *gorm.DB, a, b uuid.UUID) error {
	first, second := a, b
	if bytesLess(second, first) {
		first, second = second, first
	}
	for _, id := range []uuid.UUID{first, second} {
		var user activeUser
		if err := tx.Raw(`SELECT id FROM users WHERE id = ? FOR UPDATE`, id).Scan(&user).Error; err != nil {
			return fmt.Errorf("lock user: %w", err)
		}
		if user.ID == uuid.Nil {
			return ErrTargetNotFound
		}
	}
	return nil
}

func bytesLess(a, b uuid.UUID) bool {
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

// GetByPair returns the friendship for a normalized pair, if any.
func (r *Repository) GetByPair(ctx context.Context, userA, userB uuid.UUID) (*Friendship, error) {
	var f Friendship
	err := r.db.WithContext(ctx).
		Where("user_a_id = ? AND user_b_id = ?", userA, userB).
		First(&f).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get friendship by pair: %w", err)
	}
	return &f, nil
}

// GetByID returns a friendship by id.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Friendship, error) {
	var f Friendship
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&f).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get friendship by id: %w", err)
	}
	return &f, nil
}

// CreateRequest creates a pending friendship or returns a domain conflict/not-found error.
func (r *Repository) CreateRequest(ctx context.Context, requester, target uuid.UUID) (*Friendship, error) {
	userA, userB, err := NormalizePair(requester, target)
	if err != nil {
		return nil, err
	}

	var created *Friendship
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := r.lockUsersInOrder(tx, requester, target); err != nil {
			// Distinguish inactive/missing target vs missing requester later in service.
			return err
		}

		// Ensure both are active (lock only checked existence; re-check status).
		activeReq, err := r.findActiveUserTx(tx, requester)
		if err != nil {
			return err
		}
		if activeReq == nil {
			return ErrUnauthorized
		}
		activeTarget, err := r.findActiveUserTx(tx, target)
		if err != nil {
			return err
		}
		if activeTarget == nil {
			return ErrTargetNotFound
		}

		existing, err := r.getByPairTx(tx, userA, userB, true)
		if err != nil {
			return err
		}
		if existing != nil {
			switch existing.Status {
			case StatusBlocked:
				return ErrTargetNotFound // privacy-safe
			case StatusAccepted:
				return ErrAlreadyFriends
			case StatusPending:
				return ErrAlreadyPending
			default:
				return ErrConflict
			}
		}

		now := time.Now().UTC()
		f := Friendship{
			ID:                uuid.New(),
			UserAID:           userA,
			UserBID:           userB,
			RequestedByUserID: requester,
			Status:            StatusPending,
			CreatedAt:         now,
			UpdatedAt:         now,
		}
		if err := tx.Create(&f).Error; err != nil {
			return fmt.Errorf("create friend request: %w", err)
		}
		created = &f
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

// AcceptRequest accepts a pending request when the acceptor is the non-requester participant.
func (r *Repository) AcceptRequest(ctx context.Context, requestID, acceptor uuid.UUID) (*Friendship, *PublicProfile, error) {
	var accepted *Friendship
	var other PublicProfile
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var f Friendship
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", requestID).
			First(&f).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("lock friendship: %w", err)
		}
		if f.Status != StatusPending {
			return ErrNotFound
		}
		if acceptor != f.UserAID && acceptor != f.UserBID {
			return ErrNotFound
		}
		if f.RequestedByUserID == acceptor {
			return ErrNotFound // only recipient may accept
		}

		if err := r.lockUsersInOrder(tx, f.UserAID, f.UserBID); err != nil {
			return err
		}

		now := time.Now().UTC()
		f.Status = StatusAccepted
		f.AcceptedAt = &now
		f.BlockedByUserID = nil
		f.UpdatedAt = now
		if err := tx.Model(&Friendship{}).Where("id = ? AND status = ?", f.ID, StatusPending).Updates(map[string]any{
			"status":             StatusAccepted,
			"accepted_at":        now,
			"blocked_by_user_id": nil,
			"updated_at":         now,
		}).Error; err != nil {
			return fmt.Errorf("accept friendship: %w", err)
		}

		otherID, err := f.OtherUserID(acceptor)
		if err != nil {
			return err
		}
		profile, err := r.loadPublicProfileTx(tx, otherID)
		if err != nil {
			return err
		}
		if profile == nil {
			return ErrTargetNotFound
		}
		accepted = &f
		other = *profile
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return accepted, &other, nil
}

// DeclineRequest deletes a pending request when the caller is the recipient.
func (r *Repository) DeclineRequest(ctx context.Context, requestID, actor uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var f Friendship
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", requestID).
			First(&f).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("lock friendship: %w", err)
		}
		if f.Status != StatusPending {
			return ErrNotFound
		}
		if actor != f.UserAID && actor != f.UserBID {
			return ErrNotFound
		}
		if f.RequestedByUserID == actor {
			return ErrNotFound
		}
		if err := tx.Delete(&Friendship{}, "id = ?", f.ID).Error; err != nil {
			return fmt.Errorf("decline friendship: %w", err)
		}
		return nil
	})
}

// RemoveFriendship deletes an accepted friendship when the actor is a participant.
// Missing or non-accepted edges are treated as successful no-ops for idempotency.
func (r *Repository) RemoveFriendship(ctx context.Context, actor, other uuid.UUID) error {
	userA, userB, err := NormalizePair(actor, other)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("user_a_id = ? AND user_b_id = ? AND status = ?", userA, userB, StatusAccepted).
			Delete(&Friendship{})
		if res.Error != nil {
			return fmt.Errorf("remove friendship: %w", res.Error)
		}
		return nil
	})
}

// BlockUser replaces any pair relationship with a block owned by blocker.
func (r *Repository) BlockUser(ctx context.Context, blocker, target uuid.UUID) error {
	userA, userB, err := NormalizePair(blocker, target)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := r.lockUsersInOrder(tx, blocker, target); err != nil {
			// Target missing: still allow block? Product says privacy-safe; blocking
			// a missing user is a no-op success to avoid enumeration via block vs not-found.
			if errors.Is(err, ErrTargetNotFound) {
				activeBlocker, findErr := r.findActiveUserTx(tx, blocker)
				if findErr != nil {
					return findErr
				}
				if activeBlocker == nil {
					return ErrUnauthorized
				}
				// If target truly missing, idempotent success.
				activeTarget, findErr := r.findActiveUserTx(tx, target)
				if findErr != nil {
					return findErr
				}
				if activeTarget == nil {
					return nil
				}
			} else {
				return err
			}
		}

		activeTarget, err := r.findActiveUserTx(tx, target)
		if err != nil {
			return err
		}
		if activeTarget == nil {
			return nil // privacy-safe no-op
		}
		activeBlocker, err := r.findActiveUserTx(tx, blocker)
		if err != nil {
			return err
		}
		if activeBlocker == nil {
			return ErrUnauthorized
		}

		existing, err := r.getByPairTx(tx, userA, userB, true)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		if existing != nil {
			if existing.Status == StatusBlocked && existing.BlockedByUserID != nil && *existing.BlockedByUserID == blocker {
				return nil // same-blocker idempotent
			}
			if existing.Status == StatusBlocked && existing.BlockedByUserID != nil && *existing.BlockedByUserID != blocker {
				// Other party already blocked; keep their ownership (privacy).
				return nil
			}
			return tx.Model(&Friendship{}).Where("id = ?", existing.ID).Updates(map[string]any{
				"status":             StatusBlocked,
				"blocked_by_user_id": blocker,
				"accepted_at":        nil,
				"updated_at":         now,
			}).Error
		}

		f := Friendship{
			ID:                uuid.New(),
			UserAID:           userA,
			UserBID:           userB,
			RequestedByUserID: blocker,
			Status:            StatusBlocked,
			BlockedByUserID:   &blocker,
			CreatedAt:         now,
			UpdatedAt:         now,
		}
		return tx.Create(&f).Error
	})
}

// UnblockUser removes a block only when the actor is the blocker. Other cases no-op.
func (r *Repository) UnblockUser(ctx context.Context, actor, other uuid.UUID) error {
	userA, userB, err := NormalizePair(actor, other)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where(
			"user_a_id = ? AND user_b_id = ? AND status = ? AND blocked_by_user_id = ?",
			userA, userB, StatusBlocked, actor,
		).Delete(&Friendship{})
		if res.Error != nil {
			return fmt.Errorf("unblock user: %w", res.Error)
		}
		return nil
	})
}

// ListIncomingRequests returns pending requests where viewer is the recipient.
func (r *Repository) ListIncomingRequests(ctx context.Context, viewer uuid.UUID, limit int, cursor string) (*Page, error) {
	return r.listRelationships(ctx, listParams{
		viewer: viewer,
		limit:  limit,
		cursor: cursor,
		where:  `f.status = 'pending' AND f.requested_by_user_id <> ? AND (f.user_a_id = ? OR f.user_b_id = ?)`,
		args:   []any{viewer, viewer, viewer},
	})
}

// ListOutgoingRequests returns pending requests created by the viewer.
func (r *Repository) ListOutgoingRequests(ctx context.Context, viewer uuid.UUID, limit int, cursor string) (*Page, error) {
	return r.listRelationships(ctx, listParams{
		viewer: viewer,
		limit:  limit,
		cursor: cursor,
		where:  `f.status = 'pending' AND f.requested_by_user_id = ?`,
		args:   []any{viewer},
	})
}

// ListAcceptedFriends returns accepted friends for the viewer.
func (r *Repository) ListAcceptedFriends(ctx context.Context, viewer uuid.UUID, limit int, cursor string) (*Page, error) {
	return r.listRelationships(ctx, listParams{
		viewer: viewer,
		limit:  limit,
		cursor: cursor,
		where:  `f.status = 'accepted' AND (f.user_a_id = ? OR f.user_b_id = ?)`,
		args:   []any{viewer, viewer},
	})
}

// ListBlockedUsers returns users blocked by the viewer.
func (r *Repository) ListBlockedUsers(ctx context.Context, viewer uuid.UUID, limit int, cursor string) (*Page, error) {
	return r.listRelationships(ctx, listParams{
		viewer: viewer,
		limit:  limit,
		cursor: cursor,
		where:  `f.status = 'blocked' AND f.blocked_by_user_id = ?`,
		args:   []any{viewer},
	})
}

type listParams struct {
	viewer uuid.UUID
	limit  int
	cursor string
	where  string
	args   []any
}

type listScanRow struct {
	ID                uuid.UUID  `gorm:"column:id"`
	UserAID           uuid.UUID  `gorm:"column:user_a_id"`
	UserBID           uuid.UUID  `gorm:"column:user_b_id"`
	RequestedByUserID uuid.UUID  `gorm:"column:requested_by_user_id"`
	Status            string     `gorm:"column:status"`
	BlockedByUserID   *uuid.UUID `gorm:"column:blocked_by_user_id"`
	AcceptedAt        *time.Time `gorm:"column:accepted_at"`
	CreatedAt         time.Time  `gorm:"column:created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at"`
	OtherUserID       uuid.UUID  `gorm:"column:other_user_id"`
	DisplayName       string     `gorm:"column:display_name"`
	AvatarURL         *string    `gorm:"column:avatar_url"`
	CountryCode       *string    `gorm:"column:country_code"`
}

func (r *Repository) listRelationships(ctx context.Context, p listParams) (*Page, error) {
	parsed, err := decodeListCursor(p.cursor)
	if err != nil {
		return nil, ErrInvalidCursor
	}

	query := `
		SELECT
			f.id,
			f.user_a_id,
			f.user_b_id,
			f.requested_by_user_id,
			f.status,
			f.blocked_by_user_id,
			f.accepted_at,
			f.created_at,
			f.updated_at,
			other_u.id AS other_user_id,
			COALESCE(p.display_name, '') AS display_name,
			p.avatar_url,
			p.country_code
		FROM friendships f
		JOIN users other_u ON other_u.id = CASE
			WHEN f.user_a_id = ? THEN f.user_b_id
			ELSE f.user_a_id
		END
		LEFT JOIN user_profiles p ON p.user_id = other_u.id
		WHERE ` + p.where + `
		  AND other_u.status = 'active'
	`
	args := append([]any{p.viewer}, p.args...)
	if parsed != nil {
		query += ` AND (f.created_at, f.id) < (?, ?)`
		args = append(args, parsed.CreatedAt, parsed.ID)
	}
	query += ` ORDER BY f.created_at DESC, f.id DESC LIMIT ?`
	args = append(args, p.limit+1)

	var rows []listScanRow
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("list relationships: %w", err)
	}

	hasNext := len(rows) > p.limit
	if hasNext {
		rows = rows[:p.limit]
	}

	items := make([]RelationshipRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, RelationshipRow{
			Friendship: Friendship{
				ID:                row.ID,
				UserAID:           row.UserAID,
				UserBID:           row.UserBID,
				RequestedByUserID: row.RequestedByUserID,
				Status:            row.Status,
				BlockedByUserID:   row.BlockedByUserID,
				AcceptedAt:        row.AcceptedAt,
				CreatedAt:         row.CreatedAt,
				UpdatedAt:         row.UpdatedAt,
			},
			Other: PublicProfile{
				UserID:      row.OtherUserID,
				DisplayName: row.DisplayName,
				AvatarURL:   row.AvatarURL,
				CountryCode: row.CountryCode,
			},
		})
	}

	page := &Page{Items: items, Limit: p.limit}
	if hasNext && len(items) > 0 {
		last := items[len(items)-1]
		c := encodeListCursor(last.Friendship.CreatedAt, last.Friendship.ID)
		page.NextCursor = &c
	}
	return page, nil
}

func (r *Repository) findActiveUserTx(tx *gorm.DB, userID uuid.UUID) (*uuid.UUID, error) {
	var row activeUser
	if err := tx.Raw(`SELECT id FROM users WHERE id = ? AND status = 'active'`, userID).Scan(&row).Error; err != nil {
		return nil, fmt.Errorf("find active user: %w", err)
	}
	if row.ID == uuid.Nil {
		return nil, nil
	}
	return &row.ID, nil
}

func (r *Repository) getByPairTx(tx *gorm.DB, userA, userB uuid.UUID, forUpdate bool) (*Friendship, error) {
	q := tx.Where("user_a_id = ? AND user_b_id = ?", userA, userB)
	if forUpdate {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var f Friendship
	err := q.First(&f).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get friendship by pair: %w", err)
	}
	return &f, nil
}

func (r *Repository) loadPublicProfileTx(tx *gorm.DB, userID uuid.UUID) (*PublicProfile, error) {
	var row struct {
		UserID      uuid.UUID `gorm:"column:user_id"`
		DisplayName string    `gorm:"column:display_name"`
		AvatarURL   *string   `gorm:"column:avatar_url"`
		CountryCode *string   `gorm:"column:country_code"`
	}
	err := tx.Raw(`
		SELECT u.id AS user_id, COALESCE(p.display_name, '') AS display_name, p.avatar_url, p.country_code
		FROM users u
		LEFT JOIN user_profiles p ON p.user_id = u.id
		WHERE u.id = ? AND u.status = 'active'
	`, userID).Scan(&row).Error
	if err != nil {
		return nil, fmt.Errorf("load public profile: %w", err)
	}
	if row.UserID == uuid.Nil {
		return nil, nil
	}
	return &PublicProfile{
		UserID:      row.UserID,
		DisplayName: row.DisplayName,
		AvatarURL:   row.AvatarURL,
		CountryCode: row.CountryCode,
	}, nil
}

// LoadPublicProfile loads a public profile for an active user.
func (r *Repository) LoadPublicProfile(ctx context.Context, userID uuid.UUID) (*PublicProfile, error) {
	return r.loadPublicProfileTx(r.db.WithContext(ctx), userID)
}
