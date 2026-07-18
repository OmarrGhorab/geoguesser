package rooms

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/games"
	"github.com/raven/geoguess/backend/internal/session"
)

func TestServiceCreateJoinAndReloadRoom(t *testing.T) {
	store := newMemoryStore()
	coord := newMemoryCoordinator()
	service := NewService(store, coord, nil, nil)
	service.clock = func() time.Time { return time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC) }

	hostGuest := "guest-host"
	created, err := service.CreateRoom(context.Background(), &session.Context{Kind: session.KindGuest, GuestID: &hostGuest}, CreateRoomRequest{
		MapID:      uuid.New(),
		Visibility: VisibilityPrivate,
		RoundCount: 5,
		MaxPlayers: 2,
	})
	if err != nil {
		t.Fatalf("CreateRoom failed: %v", err)
	}
	if len(created.Room.Code) != 6 || created.Room.HostPlayerID == nil || len(created.Room.Players) != 1 {
		t.Fatalf("created room = %+v", created.Room)
	}

	playerGuest := "guest-player"
	joined, err := service.JoinRoom(context.Background(), &session.Context{Kind: session.KindGuest, GuestID: &playerGuest}, JoinRoomRequest{Code: created.Room.Code, DisplayName: ptr("Raven")})
	if err != nil {
		t.Fatalf("JoinRoom failed: %v", err)
	}
	if len(joined.Room.Players) != 2 || joined.Room.Version < created.Room.Version {
		t.Fatalf("joined room = %+v", joined.Room)
	}

	rejoined, err := service.JoinRoom(context.Background(), &session.Context{Kind: session.KindGuest, GuestID: &playerGuest}, JoinRoomRequest{Code: created.Room.Code})
	if err != nil {
		t.Fatalf("rejoin failed: %v", err)
	}
	if len(rejoined.Room.Players) != 2 {
		t.Fatalf("rejoin duplicated player: %+v", rejoined.Room.Players)
	}
}

func TestServiceRejectsInvalidCreateRoom(t *testing.T) {
	service := NewService(newMemoryStore(), nil, nil, nil)
	guest := "guest"
	_, err := service.CreateRoom(context.Background(), &session.Context{Kind: session.KindGuest, GuestID: &guest}, CreateRoomRequest{Visibility: VisibilityPrivate, RoundCount: 5, MaxPlayers: 2})
	if err != ErrInvalidRoomRequest {
		t.Fatalf("err = %v, want ErrInvalidRoomRequest", err)
	}
}

func TestServiceCreateRoomAppliesPartyLobbyDefaults(t *testing.T) {
	service := NewService(newMemoryStore(), nil, nil, nil)
	service.clock = func() time.Time { return time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC) }
	guest := "party-host"
	created, err := service.CreateRoom(context.Background(), &session.Context{Kind: session.KindGuest, GuestID: &guest}, CreateRoomRequest{
		MapID: uuid.New(), Visibility: VisibilityPrivate,
	})
	if err != nil {
		t.Fatalf("CreateRoom: %v", err)
	}
	if created.Room.Mode != games.GameModePartyLobby || created.Room.RoundCount != 5 || created.Room.MaxPlayers != 50 {
		t.Fatalf("defaults = %+v", created.Room)
	}
	if created.Room.TimerSeconds == nil || *created.Room.TimerSeconds != 180 {
		t.Fatalf("timer = %v, want 180", created.Room.TimerSeconds)
	}
}

func TestServicePublishesPartyLobbyGameOutcomeAndCompletesRoom(t *testing.T) {
	store := newMemoryStore()
	coordinator := newMemoryCoordinator()
	service := NewService(store, coordinator, nil, nil)
	guest := "outcome-host"
	created, err := service.CreateRoom(context.Background(), &session.Context{Kind: session.KindGuest, GuestID: &guest}, CreateRoomRequest{
		MapID: uuid.New(), Visibility: VisibilityPrivate,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	room := store.rooms[created.Room.Code]
	room.Status = StatusActive
	if err := service.PublishMultiplayerOutcome(context.Background(), *room.GameID, games.MultiplayerGuessOutcome{RoundCompleted: true, GameCompleted: true}); err != nil {
		t.Fatalf("publish outcome: %v", err)
	}
	if room.Status != StatusCompleted {
		t.Fatalf("room status = %s", room.Status)
	}
}

func TestServiceStartRoomIsDurableAndNaturallyIdempotent(t *testing.T) {
	store := newMemoryStore()
	coordinator := newMemoryCoordinator()
	gameService := &memoryGameService{store: store}
	service := NewServiceWithGames(store, coordinator, gameService, nil, nil)
	service.clock = func() time.Time { return time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC) }
	host := "start-host"
	created, err := service.CreateRoom(context.Background(), &session.Context{Kind: session.KindGuest, GuestID: &host}, CreateRoomRequest{
		MapID: uuid.New(), Visibility: VisibilityPrivate, MaxPlayers: 2,
	})
	if err != nil {
		t.Fatalf("create room: %v", err)
	}
	guest := "start-guest"
	if _, err := service.JoinRoom(context.Background(), &session.Context{Kind: session.KindGuest, GuestID: &guest}, JoinRoomRequest{Code: created.Room.Code}); err != nil {
		t.Fatalf("join room: %v", err)
	}
	key := "room-start-retry-key"
	started, err := service.StartRoom(context.Background(), &session.Context{Kind: session.KindGuest, GuestID: &host}, created.Room.Code, key)
	if err != nil || started.Room.Status != StatusActive {
		t.Fatalf("start room=%+v err=%v", started, err)
	}
	replay, err := service.StartRoom(context.Background(), &session.Context{Kind: session.KindGuest, GuestID: &host}, created.Room.Code, key)
	if err != nil || replay.Room.Status != StatusActive || gameService.starts != 1 {
		t.Fatalf("start replay=%+v starts=%d err=%v", replay, gameService.starts, err)
	}
	if len(coordinator.commands) != 0 {
		t.Fatalf("start must not consume Redis command claims: %+v", coordinator.commands)
	}
}

func TestServiceGetRoomRequiresParticipantAndReturnsCurrentPlayer(t *testing.T) {
	store := newMemoryStore()
	service := NewService(store, newMemoryCoordinator(), nil, nil)
	service.clock = func() time.Time { return time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC) }

	hostGuest := "guest-host"
	created, err := service.CreateRoom(context.Background(), &session.Context{Kind: session.KindGuest, GuestID: &hostGuest}, CreateRoomRequest{
		MapID:      uuid.New(),
		Visibility: VisibilityPrivate,
		RoundCount: 5,
		MaxPlayers: 2,
	})
	if err != nil {
		t.Fatalf("CreateRoom failed: %v", err)
	}

	outsiderGuest := "guest-outsider"
	if _, err := service.GetRoom(context.Background(), &session.Context{Kind: session.KindGuest, GuestID: &outsiderGuest}, created.Room.Code); err != ErrRoomPlayerNotFound {
		t.Fatalf("outsider GetRoom err = %v, want ErrRoomPlayerNotFound", err)
	}

	loaded, err := service.GetRoom(context.Background(), &session.Context{Kind: session.KindGuest, GuestID: &hostGuest}, created.Room.Code)
	if err != nil {
		t.Fatalf("host GetRoom failed: %v", err)
	}
	if loaded.Room.CurrentPlayerID == nil || loaded.Room.HostPlayerID == nil || *loaded.Room.CurrentPlayerID != *loaded.Room.HostPlayerID {
		t.Fatalf("current_player_id=%v host_player_id=%v", loaded.Room.CurrentPlayerID, loaded.Room.HostPlayerID)
	}
}

type memoryStore struct {
	rooms          map[string]*Room
	participants   map[uuid.UUID][]Participant
	playersByGuest map[string]uuid.UUID
}

type memoryGameService struct {
	store  *memoryStore
	starts int
}

func (s *memoryGameService) StartPrivateRoomGame(_ context.Context, gameID uuid.UUID) (*games.MultiplayerStart, error) {
	for _, room := range s.store.rooms {
		if room.GameID != nil && *room.GameID == gameID {
			if room.Status != StatusLobby {
				return nil, games.ErrInvalidTransition
			}
			room.Status = StatusActive
			s.starts++
			return &games.MultiplayerStart{}, nil
		}
	}
	return nil, games.ErrGameNotFound
}

func (*memoryGameService) GetPrivateRoomRoundState(context.Context, uuid.UUID) (*games.MultiplayerRoundState, error) {
	return nil, nil
}

func newMemoryStore() *memoryStore {
	return &memoryStore{rooms: map[string]*Room{}, participants: map[uuid.UUID][]Participant{}, playersByGuest: map[string]uuid.UUID{}}
}

func (s *memoryStore) CreateRoomBundle(_ context.Context, bundle CreateRoomBundle) error {
	bundle.Game.ID = uuid.New()
	bundle.Room.ID = uuid.New()
	bundle.Room.GameID = &bundle.Game.ID
	bundle.Player.ID = uuid.New()
	bundle.Player.GameID = bundle.Game.ID
	s.rooms[bundle.Room.Code] = bundle.Room
	s.participants[bundle.Room.ID] = []Participant{{
		RoomPlayer:        RoomPlayer{RoomID: bundle.Room.ID, GamePlayerID: bundle.Player.ID, Status: ParticipantStatusJoined, JoinedAt: bundle.Player.JoinedAt},
		UserID:            bundle.Player.UserID,
		GuestIdentityHash: bundle.Player.GuestIdentityHash,
		DisplayName:       bundle.Player.DisplayName,
		Role:              bundle.Player.Role,
		GameStatus:        bundle.Player.Status,
	}}
	if bundle.Player.GuestIdentityHash != nil {
		s.playersByGuest[*bundle.Player.GuestIdentityHash] = bundle.Player.ID
	}
	return nil
}

func (s *memoryStore) GetRoomByCode(_ context.Context, code string) (*Room, error) {
	return s.rooms[code], nil
}

func (s *memoryStore) JoinRoom(_ context.Context, roomID uuid.UUID, identity ownerIdentity, displayName string, now time.Time) (*JoinRoomBundle, error) {
	for _, room := range s.rooms {
		if room.ID != roomID {
			continue
		}
		if identity.guestHash != nil {
			if id, ok := s.playersByGuest[*identity.guestHash]; ok {
				return &JoinRoomBundle{Room: room, Player: &games.GamePlayer{ID: id}}, nil
			}
		}
		id := uuid.New()
		s.participants[roomID] = append(s.participants[roomID], Participant{
			RoomPlayer:        RoomPlayer{RoomID: roomID, GamePlayerID: id, Status: ParticipantStatusJoined, JoinedAt: now},
			GuestIdentityHash: identity.guestHash,
			DisplayName:       displayName,
			Role:              games.PlayerRolePlayer,
			GameStatus:        games.PlayerStatusActive,
		})
		if identity.guestHash != nil {
			s.playersByGuest[*identity.guestHash] = id
		}
		return &JoinRoomBundle{Room: room, Player: &games.GamePlayer{ID: id}, Joined: true}, nil
	}
	return nil, ErrRoomNotFound
}

func (s *memoryStore) ListParticipants(_ context.Context, roomID uuid.UUID) ([]Participant, error) {
	return s.participants[roomID], nil
}

func (s *memoryStore) UpdateSettings(_ context.Context, roomID uuid.UUID, req UpdateRoomSettingsRequest, now time.Time) (*Room, error) {
	for _, room := range s.rooms {
		if room.ID != roomID {
			continue
		}
		if !CanUpdateSettings(room.Status) {
			return nil, ErrRoomSettingsLocked
		}
		if req.RoundCount != nil {
			room.RoundCount = *req.RoundCount
		}
		if req.TimerSeconds != nil {
			room.TimerSeconds = req.TimerSeconds
		}
		if req.MaxPlayers != nil {
			room.MaxPlayers = *req.MaxPlayers
		}
		room.UpdatedAt = now
		return room, nil
	}
	return nil, ErrRoomNotFound
}

func (s *memoryStore) SetPlayerStatus(_ context.Context, roomID, playerID uuid.UUID, status string, leftAt *time.Time) (*Room, error) {
	for _, room := range s.rooms {
		if room.ID != roomID {
			continue
		}
		for i := range s.participants[roomID] {
			if s.participants[roomID][i].GamePlayerID == playerID {
				s.participants[roomID][i].Status = status
				s.participants[roomID][i].LeftAt = leftAt
				return room, nil
			}
		}
		return nil, ErrRoomPlayerNotFound
	}
	return nil, ErrRoomNotFound
}

func (s *memoryStore) StartRoom(_ context.Context, roomID uuid.UUID, now time.Time) (*Room, error) {
	for _, room := range s.rooms {
		if room.ID == roomID {
			if !CanStart(room.Status) {
				return nil, ErrRoomAlreadyStarted
			}
			room.Status = StatusActive
			room.UpdatedAt = now
			return room, nil
		}
	}
	return nil, ErrRoomNotFound
}

func (s *memoryStore) ApplyGameOutcome(_ context.Context, gameID uuid.UUID, completed bool, now time.Time) (*Room, error) {
	for _, room := range s.rooms {
		if room.GameID != nil && *room.GameID == gameID {
			if completed {
				room.Status = StatusCompleted
			}
			room.UpdatedAt = now
			return room, nil
		}
	}
	return nil, nil
}

type memoryCoordinator struct {
	version  int64
	ready    map[uuid.UUID]bool
	commands map[string]bool
}

func newMemoryCoordinator() *memoryCoordinator {
	return &memoryCoordinator{ready: map[uuid.UUID]bool{}, commands: map[string]bool{}}
}
func (c *memoryCoordinator) IncrementVersion(context.Context, string) (int64, error) {
	c.version++
	return c.version, nil
}
func (c *memoryCoordinator) GetVersion(context.Context, string) (int64, error) { return c.version, nil }
func (c *memoryCoordinator) GetReadyPlayerIDs(context.Context, string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(c.ready))
	for id, ready := range c.ready {
		if ready {
			ids = append(ids, id)
		}
	}
	return ids, nil
}
func (c *memoryCoordinator) SetReady(_ context.Context, _ string, playerID uuid.UUID, ready bool) error {
	c.ready[playerID] = ready
	return nil
}
func (c *memoryCoordinator) ClearReady(context.Context, string) error {
	c.ready = map[uuid.UUID]bool{}
	return nil
}
func (c *memoryCoordinator) ClaimCommand(_ context.Context, roomCode, command, idempotencyKey string, _ time.Duration) (bool, error) {
	key := roomCode + ":" + command + ":" + idempotencyKey
	if c.commands[key] {
		return false, nil
	}
	c.commands[key] = true
	return true, nil
}
func (c *memoryCoordinator) SetReconnectWindow(context.Context, string, uuid.UUID, int64, time.Duration) error {
	return nil
}
func (c *memoryCoordinator) GetPresence(context.Context, string, uuid.UUID) (string, error) {
	return PresenceConnected, nil
}
func (c *memoryCoordinator) SetPresence(context.Context, string, uuid.UUID, string, time.Duration) error {
	return nil
}
func (c *memoryCoordinator) StoreSnapshot(context.Context, string, any, time.Duration) error {
	return nil
}
func (c *memoryCoordinator) Publish(context.Context, string, any) error { return nil }

func ptr(value string) *string { return &value }
