package rooms

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/games"
	"github.com/raven/geoguess/backend/internal/session"
)

type Store interface {
	CreateRoomBundle(ctx context.Context, bundle CreateRoomBundle) error
	GetRoomByCode(ctx context.Context, code string) (*Room, error)
	JoinRoom(ctx context.Context, roomID uuid.UUID, identity ownerIdentity, displayName string, now time.Time) (*JoinRoomBundle, error)
	ListParticipants(ctx context.Context, roomID uuid.UUID) ([]Participant, error)
	UpdateSettings(ctx context.Context, roomID uuid.UUID, req UpdateRoomSettingsRequest, now time.Time) (*Room, error)
	SetPlayerStatus(ctx context.Context, roomID, playerID uuid.UUID, status string, leftAt *time.Time) (*Room, error)
	StartRoom(ctx context.Context, roomID uuid.UUID, now time.Time) (*Room, error)
}

type Coordinator interface {
	IncrementVersion(ctx context.Context, roomCode string) (int64, error)
	GetVersion(ctx context.Context, roomCode string) (int64, error)
	GetReadyPlayerIDs(ctx context.Context, roomCode string) ([]uuid.UUID, error)
	SetReady(ctx context.Context, roomCode string, playerID uuid.UUID, ready bool) error
	ClearReady(ctx context.Context, roomCode string) error
	ClaimCommand(ctx context.Context, roomCode, command, idempotencyKey string, ttl time.Duration) (bool, error)
	SetReconnectWindow(ctx context.Context, roomCode string, playerID uuid.UUID, lastVersion int64, ttl time.Duration) error
	GetPresence(ctx context.Context, roomCode string, playerID uuid.UUID) (string, error)
	SetPresence(ctx context.Context, roomCode string, playerID uuid.UUID, status string, ttl time.Duration) error
	StoreSnapshot(ctx context.Context, roomCode string, snapshot any, ttl time.Duration) error
	Publish(ctx context.Context, roomCode string, event any) error
}

type GameService interface {
	StartPrivateRoomGame(ctx context.Context, gameID uuid.UUID) (*games.MultiplayerStart, error)
	GetPrivateRoomRoundState(ctx context.Context, gameID uuid.UUID) (*games.MultiplayerRoundState, error)
}

type Service struct {
	repo         Store
	coordinator  Coordinator
	games        GameService
	clock        clockFunc
	logger       *slog.Logger
	metrics      MetricsRecorder
	presenceTTL  time.Duration
	reconnectTTL time.Duration
}

type clockFunc func() time.Time

func NewService(repo Store, coordinator Coordinator, logger *slog.Logger, metrics MetricsRecorder) *Service {
	return NewServiceWithGames(repo, coordinator, nil, logger, metrics)
}

func NewServiceWithGames(repo Store, coordinator Coordinator, gameService GameService, logger *slog.Logger, metrics MetricsRecorder) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	if metrics == nil {
		metrics = NoopMetrics{}
	}
	return &Service{
		repo:         repo,
		coordinator:  coordinator,
		games:        gameService,
		clock:        time.Now,
		logger:       logger,
		metrics:      metrics,
		presenceTTL:  30 * time.Second,
		reconnectTTL: 30 * time.Second,
	}
}

func (s *Service) CreateRoom(ctx context.Context, sess *session.Context, req CreateRoomRequest) (*RoomResponse, error) {
	owner, err := ownerFromSession(sess)
	if err != nil {
		return nil, err
	}
	if err := validateCreateRoomRequest(req); err != nil {
		return nil, err
	}
	now := s.clock().UTC()
	code, err := s.generateUniqueCode(ctx)
	if err != nil {
		return nil, err
	}

	game := &games.Game{
		Mode:            games.GameModePrivateRoom,
		Status:          games.GameStatusPending,
		MapID:           req.MapID,
		CreatedByUserID: owner.userID,
		RoundCount:      req.RoundCount,
		TimerSeconds:    req.TimerSeconds,
		ScoringVersion:  games.ScoringVersionV1,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	player := &games.GamePlayer{
		UserID:            owner.userID,
		GuestIdentityHash: owner.guestHash,
		DisplayName:       displayNameFromRequest(req.DisplayName, owner.displayName),
		Role:              games.PlayerRoleHost,
		Status:            games.PlayerStatusActive,
		JoinedAt:          now,
	}
	room := &Room{
		Code:         code,
		Visibility:   VisibilityPrivate,
		Status:       StatusLobby,
		HostUserID:   owner.userID,
		MaxPlayers:   req.MaxPlayers,
		RoundCount:   req.RoundCount,
		TimerSeconds: req.TimerSeconds,
		ExpiresAt:    now.Add(24 * time.Hour),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.repo.CreateRoomBundle(ctx, CreateRoomBundle{Room: room, Game: game, Player: player}); err != nil {
		return nil, err
	}
	return s.loadAndPublish(ctx, *room, EventRoomSnapshot)
}

func (s *Service) JoinRoom(ctx context.Context, sess *session.Context, req JoinRoomRequest) (*RoomResponse, error) {
	owner, err := ownerFromSession(sess)
	if err != nil {
		return nil, err
	}
	code := normalizeRoomCode(req.Code)
	if code == "" {
		return nil, ErrRoomNotFound
	}
	room, err := s.repo.GetRoomByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, ErrRoomNotFound
	}
	if room.ExpiresAt.Before(s.clock()) {
		return nil, ErrRoomExpired
	}
	if !CanJoin(room.Status) {
		return nil, ErrRoomNotJoinable
	}
	joined, err := s.repo.JoinRoom(ctx, room.ID, owner, displayNameFromRequest(req.DisplayName, owner.displayName), s.clock().UTC())
	if err != nil {
		return nil, err
	}
	if s.coordinator != nil && joined.Player != nil {
		_ = s.coordinator.SetPresence(ctx, room.Code, joined.Player.ID, PresenceConnected, s.presenceTTL)
	}
	return s.loadAndPublish(ctx, *joined.Room, EventRoomPlayerJoined)
}

func (s *Service) GetRoom(ctx context.Context, sess *session.Context, roomCode string) (*RoomResponse, error) {
	owner, err := ownerFromSession(sess)
	if err != nil {
		return nil, err
	}
	room, participants, err := s.authorizedRoom(ctx, sess, roomCode, false)
	if err != nil {
		return nil, err
	}
	participant, ok := participantForSession(participants, owner)
	if !ok {
		return nil, ErrRoomPlayerNotFound
	}
	dto, err := s.buildRoomDTO(ctx, *room, participants, &participant.GamePlayerID)
	if err != nil {
		return nil, err
	}
	return &RoomResponse{Room: dto}, nil
}

func (s *Service) TouchPresence(ctx context.Context, sess *session.Context, roomCode string) (*RoomResponse, error) {
	room, participants, err := s.authorizedRoom(ctx, sess, roomCode, false)
	if err != nil {
		return nil, err
	}
	owner := mustOwner(sess)
	participant, ok := participantForSession(participants, owner)
	if !ok {
		return nil, ErrRoomPlayerNotFound
	}
	if s.coordinator != nil {
		if err := s.coordinator.SetPresence(ctx, room.Code, participant.GamePlayerID, PresenceConnected, s.presenceTTL); err != nil {
			return nil, err
		}
	}
	dto, err := s.buildRoomDTO(ctx, *room, participants, &participant.GamePlayerID)
	if err != nil {
		return nil, err
	}
	return &RoomResponse{Room: dto}, nil
}

func (s *Service) MarkDisconnected(ctx context.Context, sess *session.Context, roomCode string, lastVersion int64) error {
	room, participants, err := s.authorizedRoom(ctx, sess, roomCode, false)
	if err != nil {
		return err
	}
	participant, ok := participantForSession(participants, mustOwner(sess))
	if !ok {
		return ErrRoomPlayerNotFound
	}
	if s.coordinator != nil {
		if err := s.coordinator.SetPresence(ctx, room.Code, participant.GamePlayerID, PresenceDisconnected, s.reconnectTTL); err != nil {
			return err
		}
		if err := s.coordinator.SetReconnectWindow(ctx, room.Code, participant.GamePlayerID, lastVersion, s.reconnectTTL); err != nil {
			return err
		}
		_ = s.coordinator.Publish(ctx, room.Code, map[string]any{"type": EventRoomPlayerDisconnected, "player_id": participant.GamePlayerID})
	}
	return nil
}

func (s *Service) UpdateSettings(ctx context.Context, sess *session.Context, roomCode string, req UpdateRoomSettingsRequest) (*RoomResponse, error) {
	room, participants, err := s.authorizedRoom(ctx, sess, roomCode, true)
	if err != nil {
		return nil, err
	}
	if !CanUpdateSettings(room.Status) {
		return nil, ErrRoomSettingsLocked
	}
	if err := validateSettingsUpdate(req, len(activeParticipants(participants))); err != nil {
		return nil, err
	}
	updated, err := s.repo.UpdateSettings(ctx, room.ID, req, s.clock().UTC())
	if err != nil {
		return nil, err
	}
	if s.coordinator != nil {
		if err := s.coordinator.ClearReady(ctx, updated.Code); err != nil {
			return nil, err
		}
	}
	resp, err := s.loadAndPublish(ctx, *updated, EventRoomSettingsUpdated)
	if err != nil {
		return nil, err
	}
	if s.coordinator != nil {
		_ = s.coordinator.Publish(ctx, updated.Code, map[string]any{"type": EventRoomReadyReset, "room": resp.Room})
	}
	return resp, nil
}

func (s *Service) SetReady(ctx context.Context, sess *session.Context, roomCode string, req ReadyRoomRequest) (*RoomResponse, error) {
	room, participants, err := s.authorizedRoom(ctx, sess, roomCode, false)
	if err != nil {
		return nil, err
	}
	if !CanStart(room.Status) {
		return nil, ErrRoomAlreadyStarted
	}
	player, ok := participantForSession(participants, mustOwner(sess))
	if !ok {
		return nil, ErrRoomPlayerNotFound
	}
	if s.coordinator != nil {
		if err := s.coordinator.SetReady(ctx, room.Code, player.GamePlayerID, req.Ready); err != nil {
			return nil, err
		}
	}
	return s.loadAndPublish(ctx, *room, EventRoomReadyUpdated)
}

func (s *Service) RemovePlayer(ctx context.Context, sess *session.Context, roomCode string, playerID uuid.UUID) (*RoomResponse, error) {
	room, participants, err := s.authorizedRoom(ctx, sess, roomCode, true)
	if err != nil {
		return nil, err
	}
	if playerID == uuid.Nil {
		return nil, ErrInvalidRoomRequest
	}
	host := hostParticipant(participants)
	if host != nil && host.GamePlayerID == playerID {
		return nil, ErrInvalidRoomRequest
	}
	if _, ok := participantByID(participants, playerID); !ok {
		return nil, ErrRoomPlayerNotFound
	}
	now := s.clock().UTC()
	updated, err := s.repo.SetPlayerStatus(ctx, room.ID, playerID, ParticipantStatusKicked, &now)
	if err != nil {
		return nil, err
	}
	if s.coordinator != nil {
		_ = s.coordinator.SetReady(ctx, updated.Code, playerID, false)
		_ = s.coordinator.SetPresence(ctx, updated.Code, playerID, PresenceDisconnected, s.presenceTTL)
	}
	return s.loadAndPublish(ctx, *updated, EventRoomPlayerRemoved)
}

func (s *Service) StartRoom(ctx context.Context, sess *session.Context, roomCode, idempotencyKey string) (*RoomResponse, error) {
	room, participants, err := s.authorizedRoom(ctx, sess, roomCode, true)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return nil, ErrIdempotencyConflict
	}
	if s.coordinator != nil {
		claimed, err := s.coordinator.ClaimCommand(ctx, room.Code, "start", strings.TrimSpace(idempotencyKey), 24*time.Hour)
		if err != nil {
			return nil, err
		}
		if !claimed {
			return nil, ErrIdempotencyConflict
		}
	}
	if !CanStart(room.Status) {
		return nil, ErrRoomAlreadyStarted
	}
	if len(activeParticipants(participants)) < 2 {
		return nil, ErrInvalidRoomRequest
	}
	if room.GameID == nil {
		return nil, ErrRoomNotJoinable
	}
	if s.games != nil {
		if _, err := s.games.StartPrivateRoomGame(ctx, *room.GameID); err != nil {
			return nil, err
		}
	}
	updated, err := s.repo.StartRoom(ctx, room.ID, s.clock().UTC())
	if err != nil {
		return nil, err
	}
	if s.coordinator != nil {
		if err := s.coordinator.ClearReady(ctx, updated.Code); err != nil {
			return nil, err
		}
	}
	return s.loadAndPublish(ctx, *updated, EventRoomStarted)
}

func (s *Service) loadAndPublish(ctx context.Context, room Room, eventType string) (*RoomResponse, error) {
	participants, err := s.repo.ListParticipants(ctx, room.ID)
	if err != nil {
		return nil, err
	}
	if s.coordinator != nil {
		if _, err := s.coordinator.IncrementVersion(ctx, room.Code); err != nil {
			return nil, err
		}
		for _, participant := range participants {
			if participant.Status == ParticipantStatusJoined {
				if err := s.coordinator.SetPresence(ctx, room.Code, participant.GamePlayerID, PresenceConnected, s.presenceTTL); err != nil {
					return nil, err
				}
			}
		}
	}
	dto, err := s.buildRoomDTO(ctx, room, participants, nil)
	if err != nil {
		return nil, err
	}
	resp := &RoomResponse{Room: dto}
	if s.coordinator != nil {
		_ = s.coordinator.StoreSnapshot(ctx, room.Code, resp, 24*time.Hour)
		_ = s.coordinator.Publish(ctx, room.Code, map[string]any{"type": eventType, "room": dto})
	}
	return resp, nil
}

func (s *Service) buildRoomDTO(ctx context.Context, room Room, participants []Participant, currentPlayerID *uuid.UUID) (RoomDTO, error) {
	version := int64(0)
	readyIDs := []uuid.UUID{}
	ready := map[uuid.UUID]bool{}
	if s.coordinator != nil {
		var err error
		version, err = s.coordinator.GetVersion(ctx, room.Code)
		if err != nil {
			return RoomDTO{}, err
		}
		readyIDs, err = s.coordinator.GetReadyPlayerIDs(ctx, room.Code)
		if err != nil {
			return RoomDTO{}, err
		}
		for _, id := range readyIDs {
			ready[id] = true
		}
	}

	players := make([]RoomPlayerDTO, 0, len(participants))
	var hostPlayerID *uuid.UUID
	for _, p := range participants {
		if p.Role == PlayerRoleHost {
			id := p.GamePlayerID
			hostPlayerID = &id
		}
		presence := PresenceDisconnected
		if IsActiveParticipant(p.Status) {
			presence = PresenceConnected
		}
		if s.coordinator != nil {
			resolved, err := s.coordinator.GetPresence(ctx, room.Code, p.GamePlayerID)
			if err != nil {
				return RoomDTO{}, err
			}
			if resolved != "" {
				presence = resolved
			}
		}
		players = append(players, RoomPlayerDTO{
			ID:               p.GamePlayerID,
			UserID:           p.UserID,
			DisplayName:      p.DisplayName,
			Role:             p.Role,
			MembershipStatus: p.Status,
			PresenceStatus:   presence,
			IsReady:          ready[p.GamePlayerID],
			TotalScore:       p.TotalScore,
			JoinedAt:         p.JoinedAt,
			LeftAt:           p.LeftAt,
		})
	}
	var currentRound *RoomCurrentRoundDTO
	var guessProgress *RoomGuessProgress
	if s.games != nil && room.GameID != nil && (room.Status == StatusActive || room.Status == StatusCompleted) {
		state, err := s.games.GetPrivateRoomRoundState(ctx, *room.GameID)
		if err != nil {
			return RoomDTO{}, err
		}
		if state != nil {
			currentRound = &RoomCurrentRoundDTO{
				ID:          state.RoundID,
				RoundNumber: state.RoundNumber,
				Status:      state.Status,
				StartsAt:    state.StartsAt,
				EndsAt:      state.EndsAt,
				Media: &games.RoundMedia{
					Type:        state.Provider,
					URL:         state.MediaURL,
					Attribution: state.Attribution,
				},
				Revealed: state.Status == games.RoundStatusCompleted || room.Status == StatusCompleted,
			}
			guessProgress = &RoomGuessProgress{
				SubmittedCount:     state.SubmittedCount,
				EligibleCount:      state.EligibleCount,
				SubmittedPlayerIDs: state.SubmittedPlayerIDs,
			}
		}
	}

	return RoomDTO{
		ID:              room.ID,
		Code:            room.Code,
		Visibility:      room.Visibility,
		Status:          room.Status,
		GameID:          room.GameID,
		HostPlayerID:    hostPlayerID,
		CurrentPlayerID: currentPlayerID,
		Version:         version,
		MaxPlayers:      room.MaxPlayers,
		RoundCount:      room.RoundCount,
		TimerSeconds:    room.TimerSeconds,
		ExpiresAt:       room.ExpiresAt,
		Players:         players,
		ReadyPlayerIDs:  readyIDs,
		CurrentRound:    currentRound,
		GuessProgress:   guessProgress,
	}, nil
}

func normalizeRoomCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

type ownerIdentity struct {
	userID      *uuid.UUID
	guestHash   *string
	displayName string
}

func ownerFromSession(sess *session.Context) (ownerIdentity, error) {
	if sess == nil {
		return ownerIdentity{}, ErrRoomPlayerNotFound
	}
	if sess.IsRegistered() {
		id, err := uuid.Parse(*sess.UserID)
		if err != nil {
			return ownerIdentity{}, ErrRoomPlayerNotFound
		}
		return ownerIdentity{userID: &id, displayName: "Player"}, nil
	}
	if sess.IsGuest() {
		guest := *sess.GuestID
		return ownerIdentity{guestHash: &guest, displayName: "Guest"}, nil
	}
	return ownerIdentity{}, ErrRoomPlayerNotFound
}

func validateCreateRoomRequest(req CreateRoomRequest) error {
	if req.MapID == uuid.Nil || req.Visibility != VisibilityPrivate || req.RoundCount < 1 || req.RoundCount > 10 || req.MaxPlayers < 2 || req.MaxPlayers > 50 {
		return ErrInvalidRoomRequest
	}
	if req.TimerSeconds != nil && (*req.TimerSeconds < 10 || *req.TimerSeconds > 600) {
		return ErrInvalidRoomRequest
	}
	return nil
}

func displayNameFromRequest(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	trimmed := strings.TrimSpace(*value)
	if len(trimmed) < 2 || len(trimmed) > 32 {
		return fallback
	}
	return trimmed
}

func (s *Service) authorizedRoom(ctx context.Context, sess *session.Context, roomCode string, requireHost bool) (*Room, []Participant, error) {
	owner, err := ownerFromSession(sess)
	if err != nil {
		return nil, nil, err
	}
	code := normalizeRoomCode(roomCode)
	if code == "" {
		return nil, nil, ErrRoomNotFound
	}
	room, err := s.repo.GetRoomByCode(ctx, code)
	if err != nil {
		return nil, nil, err
	}
	if room == nil {
		return nil, nil, ErrRoomNotFound
	}
	participants, err := s.repo.ListParticipants(ctx, room.ID)
	if err != nil {
		return nil, nil, err
	}
	participant, ok := participantForSession(participants, owner)
	if !ok || !IsActiveParticipant(participant.Status) {
		return nil, nil, ErrRoomPlayerNotFound
	}
	if requireHost && participant.Role != PlayerRoleHost {
		return nil, nil, ErrRoomHostRequired
	}
	return room, participants, nil
}

func participantForSession(participants []Participant, owner ownerIdentity) (Participant, bool) {
	for _, participant := range participants {
		if owner.userID != nil && participant.UserID != nil && *participant.UserID == *owner.userID {
			return participant, true
		}
		if owner.guestHash != nil && participant.GuestIdentityHash != nil && *participant.GuestIdentityHash == *owner.guestHash {
			return participant, true
		}
	}
	return Participant{}, false
}

func mustOwner(sess *session.Context) ownerIdentity {
	owner, _ := ownerFromSession(sess)
	return owner
}

func hostParticipant(participants []Participant) *Participant {
	for _, participant := range participants {
		if participant.Role == PlayerRoleHost {
			copy := participant
			return &copy
		}
	}
	return nil
}

func participantByID(participants []Participant, playerID uuid.UUID) (Participant, bool) {
	for _, participant := range participants {
		if participant.GamePlayerID == playerID {
			return participant, true
		}
	}
	return Participant{}, false
}

func activeParticipants(participants []Participant) []Participant {
	active := make([]Participant, 0, len(participants))
	for _, participant := range participants {
		if IsActiveParticipant(participant.Status) {
			active = append(active, participant)
		}
	}
	return active
}

func validateSettingsUpdate(req UpdateRoomSettingsRequest, activePlayerCount int) error {
	if req.MapID != nil && *req.MapID == uuid.Nil {
		return ErrInvalidRoomRequest
	}
	if req.RoundCount != nil && (*req.RoundCount < 1 || *req.RoundCount > 10) {
		return ErrInvalidRoomRequest
	}
	if req.MaxPlayers != nil && (*req.MaxPlayers < 2 || *req.MaxPlayers > 50 || *req.MaxPlayers < activePlayerCount) {
		return ErrInvalidRoomRequest
	}
	if req.TimerSeconds != nil && (*req.TimerSeconds < 10 || *req.TimerSeconds > 600) {
		return ErrInvalidRoomRequest
	}
	return nil
}

func (s *Service) generateUniqueCode(ctx context.Context) (string, error) {
	for i := 0; i < 8; i++ {
		code, err := generateRoomCode(6)
		if err != nil {
			return "", err
		}
		existing, err := s.repo.GetRoomByCode(ctx, code)
		if err != nil {
			return "", err
		}
		if existing == nil || IsTerminal(existing.Status) {
			return code, nil
		}
	}
	return "", fmt.Errorf("generate unique room code: %w", ErrRoomNotJoinable)
}

func generateRoomCode(length int) (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	var builder strings.Builder
	builder.Grow(length)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		builder.WriteByte(alphabet[n.Int64()])
	}
	return builder.String(), nil
}

const (
	EventRoomSnapshot           = "room.snapshot"
	EventRoomPlayerJoined       = "room.player_joined"
	EventRoomSettingsUpdated    = "room.settings_updated"
	EventRoomReadyUpdated       = "room.ready_updated"
	EventRoomReadyReset         = "room.ready_reset"
	EventRoomPlayerRemoved      = "room.player_removed"
	EventRoomStarted            = "room.started"
	EventRoomPlayerDisconnected = "room.player_disconnected"
)
