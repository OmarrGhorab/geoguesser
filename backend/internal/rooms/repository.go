package rooms

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/games"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

type CreateRoomBundle struct {
	Room   *Room
	Game   *games.Game
	Player *games.GamePlayer
}

type JoinRoomBundle struct {
	Room   *Room
	Player *games.GamePlayer
	Joined bool
}

func (r *Repository) CreateRoomBundle(ctx context.Context, bundle CreateRoomBundle) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(bundle.Game).Error; err != nil {
			return fmt.Errorf("create room game: %w", err)
		}
		bundle.Player.GameID = bundle.Game.ID
		if err := tx.Create(bundle.Player).Error; err != nil {
			return fmt.Errorf("create room host player: %w", err)
		}
		bundle.Room.GameID = &bundle.Game.ID
		if err := tx.Create(bundle.Room).Error; err != nil {
			return fmt.Errorf("create room: %w", err)
		}
		membership := RoomPlayer{
			RoomID:       bundle.Room.ID,
			GamePlayerID: bundle.Player.ID,
			Status:       ParticipantStatusJoined,
		}
		if err := tx.Create(&membership).Error; err != nil {
			return fmt.Errorf("create room host membership: %w", err)
		}
		return nil
	})
}

func (r *Repository) JoinRoom(ctx context.Context, roomID uuid.UUID, identity ownerIdentity, displayName string, now time.Time) (*JoinRoomBundle, error) {
	out := &JoinRoomBundle{}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var room Room
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&room, "id = ?", roomID).Error; err != nil {
			return err
		}
		if room.GameID == nil {
			return ErrRoomNotJoinable
		}
		out.Room = &room

		var existing games.GamePlayer
		query := tx.Where("game_id = ?", *room.GameID)
		if identity.userID != nil {
			query = query.Where("user_id = ?", *identity.userID)
		} else {
			query = query.Where("guest_identity_hash = ?", *identity.guestHash)
		}
		err := query.First(&existing).Error
		if err == nil {
			var membership RoomPlayer
			if err := tx.First(&membership, "room_id = ? AND game_player_id = ?", room.ID, existing.ID).Error; err != nil {
				return err
			}
			if membership.Status == ParticipantStatusKicked {
				return ErrRoomPlayerRemoved
			}
			if err := tx.Model(&RoomPlayer{}).Where("room_id = ? AND game_player_id = ?", room.ID, existing.ID).Updates(map[string]any{
				"status":    ParticipantStatusJoined,
				"left_at":   nil,
				"joined_at": membership.JoinedAt,
			}).Error; err != nil {
				return err
			}
			out.Player = &existing
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var count int64
		if err := tx.Model(&RoomPlayer{}).Where("room_id = ? AND status IN ?", room.ID, []string{ParticipantStatusJoined, ParticipantStatusDisconnected}).Count(&count).Error; err != nil {
			return err
		}
		if count >= int64(room.MaxPlayers) {
			return ErrRoomFull
		}

		player := &games.GamePlayer{
			GameID:            *room.GameID,
			UserID:            identity.userID,
			GuestIdentityHash: identity.guestHash,
			DisplayName:       displayName,
			Role:              games.PlayerRolePlayer,
			Status:            games.PlayerStatusActive,
			JoinedAt:          now,
		}
		if err := tx.Create(player).Error; err != nil {
			return err
		}
		membership := &RoomPlayer{
			RoomID:       room.ID,
			GamePlayerID: player.ID,
			Status:       ParticipantStatusJoined,
			JoinedAt:     now,
		}
		if err := tx.Create(membership).Error; err != nil {
			return err
		}
		out.Player = player
		out.Joined = true
		return nil
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRoomNotFound
	}
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) GetRoomByCode(ctx context.Context, code string) (*Room, error) {
	var room Room
	if err := r.db.WithContext(ctx).Where("code = ?", code).First(&room).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get room by code: %w", err)
	}
	return &room, nil
}

func (r *Repository) ListParticipants(ctx context.Context, roomID uuid.UUID) ([]Participant, error) {
	var rows []Participant
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			rp.room_id,
			rp.game_player_id,
			rp.status,
			rp.joined_at,
			rp.left_at,
			gp.user_id,
			gp.guest_identity_hash,
			gp.display_name,
			gp.role,
			gp.status AS game_status,
			gp.total_score
		FROM room_players rp
		JOIN game_players gp ON gp.id = rp.game_player_id
		WHERE rp.room_id = ?
		ORDER BY rp.joined_at ASC, rp.game_player_id ASC
	`, roomID).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list room participants: %w", err)
	}
	return rows, nil
}

func (r *Repository) UpdateSettings(ctx context.Context, roomID uuid.UUID, req UpdateRoomSettingsRequest, now time.Time) (*Room, error) {
	var room Room
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&room, "id = ?", roomID).Error; err != nil {
			return err
		}
		if !CanUpdateSettings(room.Status) {
			return ErrRoomSettingsLocked
		}
		updates := map[string]any{"updated_at": now}
		if req.RoundCount != nil {
			updates["round_count"] = *req.RoundCount
		}
		if req.TimerSeconds != nil {
			updates["timer_seconds"] = *req.TimerSeconds
		}
		if req.MaxPlayers != nil {
			updates["max_players"] = *req.MaxPlayers
		}
		if err := tx.Model(&Room{}).Where("id = ?", roomID).Updates(updates).Error; err != nil {
			return err
		}
		if room.GameID != nil {
			gameUpdates := map[string]any{"updated_at": now}
			if req.MapID != nil {
				gameUpdates["map_id"] = *req.MapID
			}
			if req.RoundCount != nil {
				gameUpdates["round_count"] = *req.RoundCount
			}
			if req.TimerSeconds != nil {
				gameUpdates["timer_seconds"] = *req.TimerSeconds
			}
			if err := tx.Model(&games.Game{}).Where("id = ?", *room.GameID).Updates(gameUpdates).Error; err != nil {
				return err
			}
		}
		return tx.First(&room, "id = ?", roomID).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRoomNotFound
	}
	if err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *Repository) SetPlayerStatus(ctx context.Context, roomID, playerID uuid.UUID, status string, leftAt *time.Time) (*Room, error) {
	var room Room
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&room, "id = ?", roomID).Error; err != nil {
			return err
		}
		result := tx.Model(&RoomPlayer{}).Where("room_id = ? AND game_player_id = ?", roomID, playerID).Updates(map[string]any{
			"status":  status,
			"left_at": leftAt,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrRoomPlayerNotFound
		}
		return nil
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRoomNotFound
	}
	if err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *Repository) StartRoom(ctx context.Context, roomID uuid.UUID, now time.Time) (*Room, error) {
	var room Room
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&room, "id = ?", roomID).Error; err != nil {
			return err
		}
		if !CanStart(room.Status) {
			return ErrRoomAlreadyStarted
		}
		if err := tx.Model(&Room{}).Where("id = ?", roomID).Updates(map[string]any{
			"status":     StatusActive,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}
		if room.GameID != nil {
			if err := tx.Model(&games.Game{}).Where("id = ?", *room.GameID).Updates(map[string]any{
				"status":     games.GameStatusActive,
				"started_at": now,
				"updated_at": now,
			}).Error; err != nil {
				return err
			}
		}
		return tx.First(&room, "id = ?", roomID).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRoomNotFound
	}
	if err != nil {
		return nil, err
	}
	return &room, nil
}
