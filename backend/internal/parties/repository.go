package parties

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository persists parties, members, and invites.
type Repository struct {
	db *gorm.DB
}

// NewRepository returns a parties repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

type activeUserRow struct {
	ID uuid.UUID `gorm:"column:id"`
}

// FindActiveUser returns the user id when status is active, otherwise nil.
func (r *Repository) FindActiveUser(ctx context.Context, userID uuid.UUID) (*uuid.UUID, error) {
	var row activeUserRow
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

// HasActiveMatchAssignment reports whether the user has a non-terminal match assignment.
func (r *Repository) HasActiveMatchAssignment(ctx context.Context, userID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(1)
		FROM match_players mp
		JOIN matches m ON m.id = mp.match_id
		WHERE mp.user_id = ?
		  AND mp.status IN ('assigned', 'active')
		  AND m.status IN ('matched', 'active')
	`, userID).Scan(&count).Error
	if err != nil {
		return false, fmt.Errorf("has active match assignment: %w", err)
	}
	return count > 0, nil
}

// CreateParty creates a forming party with the leader as the first active member.
func (r *Repository) CreateParty(ctx context.Context, leaderID uuid.UUID, format string, now time.Time) (*PartySnapshot, error) {
	capacity, ok := CapacityForFormat(format)
	if !ok {
		return nil, ErrUnsupportedFormat
	}
	now = now.UTC()

	var snap *PartySnapshot
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := r.lockUsersSorted(tx, leaderID); err != nil {
			return err
		}
		active, err := r.findActiveUserTx(tx, leaderID)
		if err != nil {
			return err
		}
		if active == nil {
			return ErrUnauthorized
		}
		existing, err := r.findActivePartyIDForUserTx(tx, leaderID)
		if err != nil {
			return err
		}
		if existing != nil {
			return ErrActivePartyConflict
		}
		busy, err := r.hasActiveMatchTx(tx, leaderID)
		if err != nil {
			return err
		}
		if busy {
			return ErrActiveQueueOrMatch
		}

		party := Party{
			ID:           uuid.New(),
			Format:       format,
			Capacity:     int16(capacity),
			LeaderUserID: leaderID,
			Status:       StatusForming,
			Version:      0,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := tx.Create(&party).Error; err != nil {
			return fmt.Errorf("create party: %w", err)
		}
		member := PartyMember{
			PartyID:  party.ID,
			UserID:   leaderID,
			Status:   MemberStatusActive,
			Ready:    false,
			JoinedAt: now,
		}
		if err := tx.Create(&member).Error; err != nil {
			if isUniqueViolation(err) {
				return ErrActivePartyConflict
			}
			return fmt.Errorf("create party member: %w", err)
		}
		loaded, err := r.loadSnapshotTx(tx, party.ID, true)
		if err != nil {
			return err
		}
		snap = loaded
		return nil
	})
	if err != nil {
		return nil, err
	}
	return snap, nil
}

// GetActivePartyForUser returns the caller's active party snapshot, or nil.
func (r *Repository) GetActivePartyForUser(ctx context.Context, userID uuid.UUID) (*PartySnapshot, error) {
	var row struct {
		PartyID uuid.UUID `gorm:"column:party_id"`
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT party_id FROM party_members
		WHERE user_id = ? AND status = 'active'
		LIMIT 1
	`, userID).Scan(&row).Error
	if err != nil {
		return nil, fmt.Errorf("get active party for user: %w", err)
	}
	if row.PartyID == uuid.Nil {
		return nil, nil
	}
	return r.LoadSnapshot(ctx, row.PartyID)
}

// LoadSnapshot loads a versioned party with active members and public profiles.
func (r *Repository) LoadSnapshot(ctx context.Context, partyID uuid.UUID) (*PartySnapshot, error) {
	return r.loadSnapshotTx(r.db.WithContext(ctx), partyID, false)
}

// CreateInvite inserts a pending invite after validating capacity and uniqueness.
func (r *Repository) CreateInvite(
	ctx context.Context,
	partyID, leaderID, inviteeID uuid.UUID,
	expiresAt, now time.Time,
) (*InviteSnapshot, error) {
	now = now.UTC()
	expiresAt = expiresAt.UTC()

	var out *InviteSnapshot
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := r.lockUsersSorted(tx, leaderID, inviteeID); err != nil {
			return err
		}
		party, err := r.lockPartyTx(tx, partyID)
		if err != nil {
			return err
		}
		if party.Status != StatusForming {
			return ErrPartyLocked
		}
		if party.LeaderUserID != leaderID {
			return ErrLeaderRequired
		}

		activeCount, err := r.countActiveMembersTx(tx, partyID)
		if err != nil {
			return err
		}
		if activeCount >= int(party.Capacity) {
			return ErrPartyFull
		}

		// Already a member?
		var existingMember PartyMember
		err = tx.Where("party_id = ? AND user_id = ? AND status = ?", partyID, inviteeID, MemberStatusActive).
			First(&existingMember).Error
		if err == nil {
			return ErrTargetBusy
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("check existing member: %w", err)
		}

		// Invitee already has another active party?
		otherParty, err := r.findActivePartyIDForUserTx(tx, inviteeID)
		if err != nil {
			return err
		}
		if otherParty != nil {
			return ErrTargetBusy
		}
		busy, err := r.hasActiveMatchTx(tx, inviteeID)
		if err != nil {
			return err
		}
		if busy {
			return ErrTargetBusy
		}

		// Pending invite uniqueness.
		var pendingCount int64
		if err := tx.Model(&PartyInvite{}).
			Where("party_id = ? AND invitee_user_id = ? AND status = ?", partyID, inviteeID, InviteStatusPending).
			Count(&pendingCount).Error; err != nil {
			return fmt.Errorf("count pending invites: %w", err)
		}
		if pendingCount > 0 {
			return ErrAlreadyInvited
		}

		invite := PartyInvite{
			ID:            uuid.New(),
			PartyID:       partyID,
			InviterUserID: leaderID,
			InviteeUserID: inviteeID,
			Status:        InviteStatusPending,
			ExpiresAt:     expiresAt,
			CreatedAt:     now,
		}
		if err := tx.Create(&invite).Error; err != nil {
			if isUniqueViolation(err) {
				return ErrAlreadyInvited
			}
			return fmt.Errorf("create invite: %w", err)
		}
		profile, err := r.loadPublicProfileTx(tx, leaderID)
		if err != nil {
			return err
		}
		if profile == nil {
			return ErrUnauthorized
		}
		out = &InviteSnapshot{
			Invite:   invite,
			Inviter:  *profile,
			Format:   party.Format,
			Capacity: party.Capacity,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ListPendingInvitesForUser returns non-expired pending invites for the invitee.
func (r *Repository) ListPendingInvitesForUser(ctx context.Context, inviteeID uuid.UUID, now time.Time) ([]InviteSnapshot, error) {
	now = now.UTC()
	type row struct {
		PartyInvite
		Format   string `gorm:"column:format"`
		Capacity int16  `gorm:"column:capacity"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		SELECT pi.*, p.format, p.capacity
		FROM party_invites pi
		JOIN parties p ON p.id = pi.party_id
		WHERE pi.invitee_user_id = ?
		  AND pi.status = 'pending'
		  AND pi.expires_at > ?
		  AND p.status = 'forming'
		ORDER BY pi.created_at DESC, pi.id DESC
	`, inviteeID, now).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list pending invites: %w", err)
	}

	// Expire any stale pending invites for this user (best-effort, non-fatal).
	_ = r.db.WithContext(ctx).Exec(`
		UPDATE party_invites
		SET status = 'expired', responded_at = ?
		WHERE invitee_user_id = ? AND status = 'pending' AND expires_at <= ?
	`, now, inviteeID, now)

	out := make([]InviteSnapshot, 0, len(rows))
	for _, row := range rows {
		profile, err := r.loadPublicProfileTx(r.db.WithContext(ctx), row.InviterUserID)
		if err != nil {
			return nil, err
		}
		inviter := PublicProfile{UserID: row.InviterUserID, DisplayName: ""}
		if profile != nil {
			inviter = *profile
		}
		out = append(out, InviteSnapshot{
			Invite:   row.PartyInvite,
			Inviter:  inviter,
			Format:   row.Format,
			Capacity: row.Capacity,
		})
	}
	return out, nil
}

// AcceptInvite atomically accepts a pending invite into a forming party.
func (r *Repository) AcceptInvite(ctx context.Context, inviteID, inviteeID uuid.UUID, now time.Time) (*PartySnapshot, error) {
	now = now.UTC()
	var snap *PartySnapshot
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Preview invite without lock to discover party/users for ordered locking.
		var preview PartyInvite
		if err := tx.Where("id = ?", inviteID).First(&preview).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInviteNotFound
			}
			return fmt.Errorf("load invite: %w", err)
		}
		if preview.InviteeUserID != inviteeID {
			return ErrInviteNotFound
		}

		// Load current active members to lock all involved users in UUID order.
		memberIDs, err := r.listActiveMemberIDsTx(tx, preview.PartyID)
		if err != nil {
			return err
		}
		lockIDs := append([]uuid.UUID{}, memberIDs...)
		lockIDs = append(lockIDs, inviteeID, preview.InviterUserID)
		if err := r.lockUsersSorted(tx, lockIDs...); err != nil {
			return err
		}

		// Lock invite.
		var invite PartyInvite
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", inviteID).
			First(&invite).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInviteNotFound
			}
			return fmt.Errorf("lock invite: %w", err)
		}
		if invite.InviteeUserID != inviteeID {
			return ErrInviteNotFound
		}
		if invite.Status == InviteStatusAccepted {
			// Idempotent replay: already accepted by this invitee.
			loaded, err := r.loadSnapshotTx(tx, invite.PartyID, true)
			if err != nil {
				return err
			}
			// Ensure invitee is still an active member of the party.
			for _, m := range loaded.Members {
				if m.Member.UserID == inviteeID && m.Member.Status == MemberStatusActive {
					snap = loaded
					return nil
				}
			}
			return ErrInviteNotFound
		}
		if invite.Status != InviteStatusPending {
			return ErrInviteNotFound
		}
		if !invite.ExpiresAt.After(now) {
			_ = tx.Model(&PartyInvite{}).Where("id = ?", invite.ID).Updates(map[string]any{
				"status":       InviteStatusExpired,
				"responded_at": now,
			}).Error
			return ErrInviteNotFound
		}

		party, err := r.lockPartyTx(tx, invite.PartyID)
		if err != nil {
			return err
		}
		if party.Status != StatusForming {
			return ErrPartyLocked
		}

		activeInvitee, err := r.findActiveUserTx(tx, inviteeID)
		if err != nil {
			return err
		}
		if activeInvitee == nil {
			return ErrUnauthorized
		}

		// Capacity recheck under lock.
		activeCount, err := r.countActiveMembersTx(tx, party.ID)
		if err != nil {
			return err
		}
		if activeCount >= int(party.Capacity) {
			return ErrPartyFull
		}

		// One active party per user.
		other, err := r.findActivePartyIDForUserTx(tx, inviteeID)
		if err != nil {
			return err
		}
		if other != nil {
			return ErrTargetBusy
		}
		busy, err := r.hasActiveMatchTx(tx, inviteeID)
		if err != nil {
			return err
		}
		if busy {
			return ErrTargetBusy
		}

		// Upsert membership: rejoin if previously left/kicked from same party, else insert.
		var existing PartyMember
		err = tx.Where("party_id = ? AND user_id = ?", party.ID, inviteeID).First(&existing).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("load membership: %w", err)
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			member := PartyMember{
				PartyID:  party.ID,
				UserID:   inviteeID,
				Status:   MemberStatusActive,
				Ready:    false,
				JoinedAt: now,
			}
			if err := tx.Create(&member).Error; err != nil {
				if isUniqueViolation(err) {
					return ErrTargetBusy
				}
				return fmt.Errorf("insert member: %w", err)
			}
		} else {
			if existing.Status == MemberStatusActive {
				return ErrTargetBusy
			}
			if err := tx.Model(&PartyMember{}).
				Where("party_id = ? AND user_id = ?", party.ID, inviteeID).
				Updates(map[string]any{
					"status":    MemberStatusActive,
					"ready":     false,
					"joined_at": now,
					"left_at":   nil,
				}).Error; err != nil {
				if isUniqueViolation(err) {
					return ErrTargetBusy
				}
				return fmt.Errorf("reactivate member: %w", err)
			}
		}

		// Clear readiness for all remaining active members (roster change).
		if err := tx.Model(&PartyMember{}).
			Where("party_id = ? AND status = ?", party.ID, MemberStatusActive).
			Update("ready", false).Error; err != nil {
			return fmt.Errorf("clear readiness: %w", err)
		}

		if err := tx.Model(&PartyInvite{}).Where("id = ? AND status = ?", invite.ID, InviteStatusPending).
			Updates(map[string]any{
				"status":       InviteStatusAccepted,
				"responded_at": now,
			}).Error; err != nil {
			return fmt.Errorf("accept invite: %w", err)
		}

		// Bump version on membership transition.
		if err := r.bumpVersionTx(tx, party.ID, now); err != nil {
			return err
		}

		loaded, err := r.loadSnapshotTx(tx, party.ID, true)
		if err != nil {
			return err
		}
		snap = loaded
		return nil
	})
	if err != nil {
		return nil, err
	}
	return snap, nil
}

// DeclineInvite idempotently declines a pending invite for the invitee.
func (r *Repository) DeclineInvite(ctx context.Context, inviteID, inviteeID uuid.UUID, now time.Time) error {
	now = now.UTC()
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var invite PartyInvite
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", inviteID).
			First(&invite).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInviteNotFound
			}
			return fmt.Errorf("lock invite: %w", err)
		}
		if invite.InviteeUserID != inviteeID {
			return ErrInviteNotFound
		}
		switch invite.Status {
		case InviteStatusDeclined:
			return nil // idempotent
		case InviteStatusPending:
			// fall through
		default:
			// Already accepted/expired/revoked — privacy-safe not found for non-pending.
			if invite.Status == InviteStatusExpired {
				return nil // treat expired decline as no-op success for idempotency of "I'm not joining"
			}
			return ErrInviteNotFound
		}
		if !invite.ExpiresAt.After(now) {
			_ = tx.Model(&PartyInvite{}).Where("id = ?", invite.ID).Updates(map[string]any{
				"status":       InviteStatusExpired,
				"responded_at": now,
			}).Error
			return nil
		}
		return tx.Model(&PartyInvite{}).Where("id = ? AND status = ?", invite.ID, InviteStatusPending).
			Updates(map[string]any{
				"status":       InviteStatusDeclined,
				"responded_at": now,
			}).Error
	})
}

// SetReadiness updates the caller's ready flag while the party is forming.
func (r *Repository) SetReadiness(ctx context.Context, partyID, userID uuid.UUID, ready bool, now time.Time) (*PartySnapshot, error) {
	now = now.UTC()
	var snap *PartySnapshot
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := r.lockUsersSorted(tx, userID); err != nil {
			return err
		}
		party, err := r.lockPartyTx(tx, partyID)
		if err != nil {
			return err
		}
		if party.Status != StatusForming {
			return ErrPartyLocked
		}
		var member PartyMember
		if err := tx.Where("party_id = ? AND user_id = ? AND status = ?", partyID, userID, MemberStatusActive).
			First(&member).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("load member: %w", err)
		}
		if member.Ready == ready {
			loaded, err := r.loadSnapshotTx(tx, partyID, true)
			if err != nil {
				return err
			}
			snap = loaded
			return nil
		}
		if err := tx.Model(&PartyMember{}).
			Where("party_id = ? AND user_id = ? AND status = ?", partyID, userID, MemberStatusActive).
			Update("ready", ready).Error; err != nil {
			return fmt.Errorf("set ready: %w", err)
		}
		if err := r.bumpVersionTx(tx, partyID, now); err != nil {
			return err
		}
		loaded, err := r.loadSnapshotTx(tx, partyID, true)
		if err != nil {
			return err
		}
		snap = loaded
		return nil
	})
	if err != nil {
		return nil, err
	}
	return snap, nil
}

// LeaveParty removes the caller from a forming party, transferring leadership when needed.
func (r *Repository) LeaveParty(ctx context.Context, partyID, userID uuid.UUID, now time.Time) error {
	now = now.UTC()
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := r.lockUsersSorted(tx, userID); err != nil {
			return err
		}
		party, err := r.lockPartyTx(tx, partyID)
		if err != nil {
			return err
		}
		if party.Status == StatusClosed {
			return ErrNotFound
		}
		if party.Status != StatusForming {
			return ErrPartyLocked
		}
		var member PartyMember
		if err := tx.Where("party_id = ? AND user_id = ? AND status = ?", partyID, userID, MemberStatusActive).
			First(&member).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil // already left — idempotent
			}
			return fmt.Errorf("load member: %w", err)
		}

		if err := tx.Model(&PartyMember{}).
			Where("party_id = ? AND user_id = ? AND status = ?", partyID, userID, MemberStatusActive).
			Updates(map[string]any{
				"status":  MemberStatusLeft,
				"ready":   false,
				"left_at": now,
			}).Error; err != nil {
			return fmt.Errorf("leave member: %w", err)
		}

		// Clear readiness of remaining members (roster change).
		if err := tx.Model(&PartyMember{}).
			Where("party_id = ? AND status = ?", partyID, MemberStatusActive).
			Update("ready", false).Error; err != nil {
			return fmt.Errorf("clear readiness: %w", err)
		}

		remaining, err := r.listActiveMembersTx(tx, partyID)
		if err != nil {
			return err
		}
		if len(remaining) == 0 {
			return r.closePartyTx(tx, partyID, now)
		}

		if userID == party.LeaderUserID {
			// Transfer to earliest joined_at, then lowest user_id.
			sort.Slice(remaining, func(i, j int) bool {
				if !remaining[i].JoinedAt.Equal(remaining[j].JoinedAt) {
					return remaining[i].JoinedAt.Before(remaining[j].JoinedAt)
				}
				return bytes.Compare(remaining[i].UserID[:], remaining[j].UserID[:]) < 0
			})
			if err := tx.Model(&Party{}).Where("id = ?", partyID).Updates(map[string]any{
				"leader_user_id": remaining[0].UserID,
				"updated_at":     now,
			}).Error; err != nil {
				return fmt.Errorf("transfer leadership: %w", err)
			}
		}

		return r.bumpVersionTx(tx, partyID, now)
	})
}

// KickMember removes another member; caller must be leader (enforced by service or here).
func (r *Repository) KickMember(ctx context.Context, partyID, leaderID, targetID uuid.UUID, now time.Time) error {
	now = now.UTC()
	if leaderID == targetID {
		return ErrInvalidRequest
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := r.lockUsersSorted(tx, leaderID, targetID); err != nil {
			return err
		}
		party, err := r.lockPartyTx(tx, partyID)
		if err != nil {
			return err
		}
		if party.Status != StatusForming {
			return ErrPartyLocked
		}
		if party.LeaderUserID != leaderID {
			return ErrLeaderRequired
		}
		var member PartyMember
		if err := tx.Where("party_id = ? AND user_id = ? AND status = ?", partyID, targetID, MemberStatusActive).
			First(&member).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil // already gone — idempotent
			}
			return fmt.Errorf("load target member: %w", err)
		}
		if err := tx.Model(&PartyMember{}).
			Where("party_id = ? AND user_id = ? AND status = ?", partyID, targetID, MemberStatusActive).
			Updates(map[string]any{
				"status":  MemberStatusKicked,
				"ready":   false,
				"left_at": now,
			}).Error; err != nil {
			return fmt.Errorf("kick member: %w", err)
		}
		if err := tx.Model(&PartyMember{}).
			Where("party_id = ? AND status = ?", partyID, MemberStatusActive).
			Update("ready", false).Error; err != nil {
			return fmt.Errorf("clear readiness: %w", err)
		}
		// Revoke pending invites for kicked user on this party.
		_ = tx.Model(&PartyInvite{}).
			Where("party_id = ? AND invitee_user_id = ? AND status = ?", partyID, targetID, InviteStatusPending).
			Updates(map[string]any{
				"status":       InviteStatusRevoked,
				"responded_at": now,
			}).Error
		return r.bumpVersionTx(tx, partyID, now)
	})
}

// DisbandParty closes a forming party, marks members left, and revokes invites.
func (r *Repository) DisbandParty(ctx context.Context, partyID, leaderID uuid.UUID, now time.Time) error {
	now = now.UTC()
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := r.lockUsersSorted(tx, leaderID); err != nil {
			return err
		}
		party, err := r.lockPartyTx(tx, partyID)
		if err != nil {
			return err
		}
		if party.Status == StatusClosed {
			// Idempotent when caller was the leader of the closed party.
			if party.LeaderUserID != leaderID {
				return ErrNotFound
			}
			return nil
		}
		if party.Status != StatusForming {
			return ErrPartyLocked
		}
		if party.LeaderUserID != leaderID {
			return ErrLeaderRequired
		}
		if err := tx.Model(&PartyMember{}).
			Where("party_id = ? AND status = ?", partyID, MemberStatusActive).
			Updates(map[string]any{
				"status":  MemberStatusLeft,
				"ready":   false,
				"left_at": now,
			}).Error; err != nil {
			return fmt.Errorf("disband members: %w", err)
		}
		if err := tx.Model(&PartyInvite{}).
			Where("party_id = ? AND status = ?", partyID, InviteStatusPending).
			Updates(map[string]any{
				"status":       InviteStatusRevoked,
				"responded_at": now,
			}).Error; err != nil {
			return fmt.Errorf("revoke invites: %w", err)
		}
		return r.closePartyTx(tx, partyID, now)
	})
}

// GetInvite loads an invite by id without locking.
func (r *Repository) GetInvite(ctx context.Context, inviteID uuid.UUID) (*PartyInvite, error) {
	var invite PartyInvite
	err := r.db.WithContext(ctx).Where("id = ?", inviteID).First(&invite).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get invite: %w", err)
	}
	return &invite, nil
}

// RestoreAfterTerminalMatch moves parties linked to a finished match from
// in_match back to forming so members can requeue. Idempotent: parties that
// are not in_match (or already cleared) are left unchanged.
func (r *Repository) RestoreAfterTerminalMatch(ctx context.Context, matchID uuid.UUID, partyIDs []uuid.UUID) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("restore after terminal match: repository unavailable")
	}
	if matchID == uuid.Nil && len(partyIDs) == 0 {
		return nil
	}

	now := time.Now().UTC()
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Collect unique party IDs from the explicit list and any row still
		// pointing at this match via active_match_id.
		seen := make(map[uuid.UUID]struct{}, len(partyIDs)+2)
		var ids []uuid.UUID
		for _, id := range partyIDs {
			if id == uuid.Nil {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
		if matchID != uuid.Nil {
			var linked []uuid.UUID
			if err := tx.Model(&Party{}).
				Where("active_match_id = ?", matchID).
				Pluck("id", &linked).Error; err != nil {
				return fmt.Errorf("list parties by active match: %w", err)
			}
			for _, id := range linked {
				if id == uuid.Nil {
					continue
				}
				if _, ok := seen[id]; ok {
					continue
				}
				seen[id] = struct{}{}
				ids = append(ids, id)
			}
		}

		for _, partyID := range ids {
			// Only restore parties still in_match. Idempotent for forming/closed/queued.
			// When matchID is known, prefer parties still bound to that match (or
			// already cleared active_match_id while stuck in_match).
			q := tx.Exec(`
				UPDATE parties
				SET status = ?,
				    active_match_id = NULL,
				    version = version + 1,
				    updated_at = ?
				WHERE id = ?
				  AND status = ?
				  AND (
				    ?::uuid = '00000000-0000-0000-0000-000000000000'::uuid
				    OR active_match_id IS NULL
				    OR active_match_id = ?
				  )
			`, StatusForming, now, partyID, StatusInMatch, matchID, matchID)
			if q.Error != nil {
				return fmt.Errorf("restore party %s: %w", partyID, q.Error)
			}
			if q.RowsAffected == 0 {
				continue
			}
			// Clear readiness so the party must re-ready before the next queue.
			if err := tx.Model(&PartyMember{}).
				Where("party_id = ? AND status = ?", partyID, MemberStatusActive).
				Update("ready", false).Error; err != nil {
				return fmt.Errorf("clear readiness for party %s: %w", partyID, err)
			}
		}
		return nil
	})
}

// --- internal helpers ---

func (r *Repository) lockPartyTx(tx *gorm.DB, partyID uuid.UUID) (*Party, error) {
	var party Party
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", partyID).
		First(&party).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock party: %w", err)
	}
	return &party, nil
}

func (r *Repository) bumpVersionTx(tx *gorm.DB, partyID uuid.UUID, now time.Time) error {
	res := tx.Exec(`
		UPDATE parties
		SET version = version + 1, updated_at = ?
		WHERE id = ?
	`, now, partyID)
	if res.Error != nil {
		return fmt.Errorf("bump version: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) closePartyTx(tx *gorm.DB, partyID uuid.UUID, now time.Time) error {
	return tx.Model(&Party{}).Where("id = ?", partyID).Updates(map[string]any{
		"status":          StatusClosed,
		"closed_at":       now,
		"updated_at":      now,
		"version":         gorm.Expr("version + 1"),
		"active_match_id": nil,
	}).Error
}

func (r *Repository) countActiveMembersTx(tx *gorm.DB, partyID uuid.UUID) (int, error) {
	var count int64
	if err := tx.Model(&PartyMember{}).
		Where("party_id = ? AND status = ?", partyID, MemberStatusActive).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count active members: %w", err)
	}
	return int(count), nil
}

func (r *Repository) listActiveMemberIDsTx(tx *gorm.DB, partyID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := tx.Model(&PartyMember{}).
		Select("user_id").
		Where("party_id = ? AND status = ?", partyID, MemberStatusActive).
		Order("joined_at ASC, user_id ASC").
		Pluck("user_id", &ids).Error
	if err != nil {
		return nil, fmt.Errorf("list active member ids: %w", err)
	}
	return ids, nil
}

func (r *Repository) listActiveMembersTx(tx *gorm.DB, partyID uuid.UUID) ([]PartyMember, error) {
	var members []PartyMember
	err := tx.Where("party_id = ? AND status = ?", partyID, MemberStatusActive).
		Order("joined_at ASC, user_id ASC").
		Find(&members).Error
	if err != nil {
		return nil, fmt.Errorf("list active members: %w", err)
	}
	return members, nil
}

func (r *Repository) findActivePartyIDForUserTx(tx *gorm.DB, userID uuid.UUID) (*uuid.UUID, error) {
	var row struct {
		PartyID uuid.UUID `gorm:"column:party_id"`
	}
	err := tx.Raw(`
		SELECT party_id FROM party_members
		WHERE user_id = ? AND status = 'active'
		LIMIT 1
	`, userID).Scan(&row).Error
	if err != nil {
		return nil, fmt.Errorf("find active party for user: %w", err)
	}
	if row.PartyID == uuid.Nil {
		return nil, nil
	}
	return &row.PartyID, nil
}

func (r *Repository) hasActiveMatchTx(tx *gorm.DB, userID uuid.UUID) (bool, error) {
	var count int64
	err := tx.Raw(`
		SELECT COUNT(1)
		FROM match_players mp
		JOIN matches m ON m.id = mp.match_id
		WHERE mp.user_id = ?
		  AND mp.status IN ('assigned', 'active')
		  AND m.status IN ('matched', 'active')
	`, userID).Scan(&count).Error
	if err != nil {
		return false, fmt.Errorf("has active match: %w", err)
	}
	return count > 0, nil
}

func (r *Repository) findActiveUserTx(tx *gorm.DB, userID uuid.UUID) (*uuid.UUID, error) {
	var row activeUserRow
	if err := tx.Raw(`SELECT id FROM users WHERE id = ? AND status = 'active'`, userID).Scan(&row).Error; err != nil {
		return nil, fmt.Errorf("find active user tx: %w", err)
	}
	if row.ID == uuid.Nil {
		return nil, nil
	}
	return &row.ID, nil
}

func (r *Repository) lockUsersSorted(tx *gorm.DB, ids ...uuid.UUID) error {
	uniq := make(map[uuid.UUID]struct{}, len(ids))
	ordered := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := uniq[id]; ok {
			continue
		}
		uniq[id] = struct{}{}
		ordered = append(ordered, id)
	}
	sort.Slice(ordered, func(i, j int) bool {
		return bytes.Compare(ordered[i][:], ordered[j][:]) < 0
	})
	for _, id := range ordered {
		var row activeUserRow
		if err := tx.Raw(`SELECT id FROM users WHERE id = ? FOR UPDATE`, id).Scan(&row).Error; err != nil {
			return fmt.Errorf("lock user: %w", err)
		}
		// Missing user is allowed here; callers recheck active status as needed.
	}
	return nil
}

func (r *Repository) loadSnapshotTx(tx *gorm.DB, partyID uuid.UUID, forUpdate bool) (*PartySnapshot, error) {
	var party Party
	q := tx.Where("id = ?", partyID)
	if forUpdate {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := q.First(&party).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("load party: %w", err)
	}

	var members []PartyMember
	if err := tx.Where("party_id = ? AND status = ?", partyID, MemberStatusActive).
		Order("joined_at ASC, user_id ASC").
		Find(&members).Error; err != nil {
		return nil, fmt.Errorf("load members: %w", err)
	}

	out := &PartySnapshot{Party: party, Members: make([]MemberSnapshot, 0, len(members))}
	for _, m := range members {
		profile, err := r.loadPublicProfileTx(tx, m.UserID)
		if err != nil {
			return nil, err
		}
		ps := PublicProfile{UserID: m.UserID, DisplayName: ""}
		if profile != nil {
			ps = *profile
		}
		out.Members = append(out.Members, MemberSnapshot{Member: m, Profile: ps})
	}
	return out, nil
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

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// lib/pq and pgx both include "duplicate key" / SQLSTATE 23505 in the message.
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "23505") ||
		strings.Contains(msg, "unique constraint")
}
