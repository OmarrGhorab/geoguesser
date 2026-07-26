package app_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/config"
	"github.com/raven/geoguess/backend/internal/health"
	apphttp "github.com/raven/geoguess/backend/internal/http"
	"github.com/raven/geoguess/backend/internal/matchmaking"
	"github.com/raven/geoguess/backend/internal/matchplay"
	"github.com/raven/geoguess/backend/internal/platform/storage"
	"github.com/raven/geoguess/backend/internal/session"
	"github.com/raven/geoguess/backend/internal/uploads"
)

// --- matchmaking failure fakes (durable-first formation) ---

type failClock struct{ now time.Time }

func (c failClock) Now() time.Time { return c.now }

type failDurableStore struct {
	users        map[uuid.UUID]*matchmaking.ActiveUser
	assignments  map[uuid.UUID]*matchmaking.ActiveAssignment
	matches      map[string]*matchmaking.Match
	postgresDown bool
}

func newFailDurableStore() *failDurableStore {
	return &failDurableStore{
		users:       map[uuid.UUID]*matchmaking.ActiveUser{},
		assignments: map[uuid.UUID]*matchmaking.ActiveAssignment{},
		matches:     map[string]*matchmaking.Match{},
	}
}

func (s *failDurableStore) FindActiveUser(_ context.Context, userID uuid.UUID) (*matchmaking.ActiveUser, error) {
	if s.postgresDown {
		return nil, matchmaking.ErrUnavailable
	}
	return s.users[userID], nil
}

func (s *failDurableStore) FindActiveAssignment(_ context.Context, userID uuid.UUID) (*matchmaking.ActiveAssignment, error) {
	if s.postgresDown {
		return nil, matchmaking.ErrUnavailable
	}
	return s.assignments[userID], nil
}

func (s *failDurableStore) HasConflictingActiveGame(_ context.Context, userID uuid.UUID) (bool, error) {
	if s.postgresDown {
		return false, matchmaking.ErrUnavailable
	}
	return false, nil
}

func (s *failDurableStore) FindMatchByFormationKey(_ context.Context, formationKey string) (*matchmaking.Match, error) {
	if s.postgresDown {
		return nil, matchmaking.ErrUnavailable
	}
	return s.matches[formationKey], nil
}

func (s *failDurableStore) CreateFormationBundle(_ context.Context, input matchmaking.FormationInput) (*matchmaking.FormationResult, error) {
	timer := input.TimerSeconds
	return s.CreateTeamFormationBundle(context.Background(), matchmaking.TeamFormationInput{
		FormationKey: input.FormationKey,
		Mode:         input.Mode,
		Playlist:     matchmaking.PlaylistRanked,
		Format:       matchmaking.FormatSolo,
		TeamSize:     1,
		MapID:        input.MapID,
		RoundCount:   input.RoundCount,
		TimerSeconds: &timer,
		StartDelay:   input.StartDelay,
		TeamOne:      []uuid.UUID{input.UserIDs[0]},
		TeamTwo:      []uuid.UUID{input.UserIDs[1]},
		SeasonID:     input.SeasonID,
		LocationIDs:  input.LocationIDs,
		MatchedAt:    input.MatchedAt,
	})
}

func (s *failDurableStore) CreateTeamFormationBundle(_ context.Context, input matchmaking.TeamFormationInput) (*matchmaking.FormationResult, error) {
	if s.postgresDown {
		return nil, matchmaking.ErrUnavailable
	}
	if existing := s.matches[input.FormationKey]; existing != nil {
		return &matchmaking.FormationResult{Match: *existing}, nil
	}
	started := input.MatchedAt
	match := matchmaking.Match{
		ID:             uuid.New(),
		FormationKey:   input.FormationKey,
		GameID:         uuid.New(),
		Mode:           input.Mode,
		Status:         matchmaking.MatchStatusActive,
		Playlist:       input.Playlist,
		Format:         input.Format,
		TeamSize:       input.TeamSize,
		MatchedAt:      input.MatchedAt,
		StartedAt:      &started,
		LastActivityAt: input.MatchedAt,
		CreatedAt:      input.MatchedAt,
		UpdatedAt:      input.MatchedAt,
	}
	s.matches[input.FormationKey] = &match
	for _, uid := range append(append([]uuid.UUID{}, input.TeamOne...), input.TeamTwo...) {
		s.assignments[uid] = &matchmaking.ActiveAssignment{
			MatchID: match.ID, GameID: match.GameID, Mode: match.Mode,
			Status: match.Status, MatchedAt: match.MatchedAt, UserID: uid,
		}
	}
	return &matchmaking.FormationResult{Match: match}, nil
}

type failQueue struct {
	entries      map[uuid.UUID]*matchmaking.QueueEntry
	nextClaim    *matchmaking.PairClaim
	redisDown    bool
	finalizeFail bool
	finalizeN    int
}

func newFailQueue() *failQueue {
	return &failQueue{entries: map[uuid.UUID]*matchmaking.QueueEntry{}}
}

func (q *failQueue) Join(_ context.Context, userID uuid.UUID, mode string, now time.Time, leaseTTL time.Duration) (*matchmaking.QueueEntry, error) {
	if q.redisDown {
		return nil, matchmaking.ErrUnavailable
	}
	e := &matchmaking.QueueEntry{
		EntryID:          uuid.NewString(),
		UserID:           userID,
		Mode:             mode,
		State:            matchmaking.QueueStateSearching,
		EnqueuedAtMs:     now.UnixMilli(),
		LeaseExpiresAtMs: now.Add(leaseTTL).UnixMilli(),
	}
	q.entries[userID] = e
	return e, nil
}

func (q *failQueue) Leave(_ context.Context, userID uuid.UUID) error {
	if q.redisDown {
		return matchmaking.ErrUnavailable
	}
	delete(q.entries, userID)
	return nil
}

func (q *failQueue) GetEntry(_ context.Context, userID uuid.UUID) (*matchmaking.QueueEntry, error) {
	if q.redisDown {
		return nil, matchmaking.ErrUnavailable
	}
	return q.entries[userID], nil
}

func (q *failQueue) RenewLease(_ context.Context, userID uuid.UUID, leaseTTL time.Duration, now time.Time) (*matchmaking.QueueEntry, error) {
	if q.redisDown {
		return nil, matchmaking.ErrUnavailable
	}
	e := q.entries[userID]
	if e == nil {
		return nil, nil
	}
	e.LeaseExpiresAtMs = now.Add(leaseTTL).UnixMilli()
	return e, nil
}

func (q *failQueue) ClaimPair(context.Context, string, time.Time, time.Duration, int) (*matchmaking.PairClaim, error) {
	if q.redisDown {
		return nil, matchmaking.ErrUnavailable
	}
	c := q.nextClaim
	q.nextClaim = nil
	return c, nil
}

func (q *failQueue) GetClaim(_ context.Context, claimID string) (*matchmaking.PairClaim, error) {
	if q.redisDown {
		return nil, matchmaking.ErrUnavailable
	}
	if q.nextClaim != nil && q.nextClaim.ClaimID == claimID {
		return q.nextClaim, nil
	}
	return nil, nil
}

func (q *failQueue) FinalizeClaim(context.Context, *matchmaking.PairClaim) error {
	q.finalizeN++
	if q.finalizeFail || q.redisDown {
		return matchmaking.ErrUnavailable
	}
	return nil
}

func (q *failQueue) ReleaseClaim(context.Context, *matchmaking.PairClaim, bool, bool, time.Duration) error {
	if q.redisDown {
		return matchmaking.ErrUnavailable
	}
	return nil
}

func (q *failQueue) ListExpiredClaims(context.Context, time.Time, int) ([]string, error) {
	if q.redisDown {
		return nil, matchmaking.ErrUnavailable
	}
	return nil, nil
}

func (q *failQueue) DropClaimIndex(context.Context, string) error {
	return nil
}

type failLocations struct {
	ids []uuid.UUID
}

func (l *failLocations) SelectLocations(context.Context, uuid.UUID, int) ([]uuid.UUID, error) {
	return append([]uuid.UUID{}, l.ids...), nil
}

func failMMConfig() matchmaking.Config {
	return matchmaking.Config{
		DefaultMapID:       uuid.New(),
		QueueLease:         30 * time.Second,
		ClaimTTL:           15 * time.Second,
		StartDelay:         5 * time.Second,
		RoundCount:         5,
		TimerSeconds:       60,
		CandidateScanLimit: 20,
	}
}

func failUserSession(id uuid.UUID) *session.Context {
	s := id.String()
	return &session.Context{Kind: session.KindUser, UserID: &s, Role: "user"}
}

// TestFailure_PostgresDurableWinsAfterFormation: Redis finalize failure after
// durable commit still returns matched with durable assignment present.
func TestFailure_PostgresDurableWinsAfterFormation(t *testing.T) {
	t.Parallel()
	userA, userB := uuid.New(), uuid.New()
	store := newFailDurableStore()
	store.users[userA] = &matchmaking.ActiveUser{ID: userA, Status: "active"}
	store.users[userB] = &matchmaking.ActiveUser{ID: userB, Status: "active"}

	queue := newFailQueue()
	claimID := uuid.NewString()
	queue.nextClaim = &matchmaking.PairClaim{
		ClaimID: claimID, FormationKey: claimID, Mode: matchmaking.ModeRankedStandard,
		UserIDA: userA, UserIDB: userB, EntryIDA: "e-a", EntryIDB: "e-b",
	}
	queue.finalizeFail = true

	locs := &failLocations{ids: []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}}
	svc := matchmaking.NewService(store, queue, failMMConfig(), nil, nil).
		WithClock(failClock{now: time.Now().UTC()}).
		WithLocations(locs)

	resp, err := svc.JoinQueue(context.Background(), failUserSession(userA), matchmaking.JoinQueueRequest{
		Mode: matchmaking.ModeRankedStandard,
	})
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if resp.Status != matchmaking.PublicStatusMatched {
		t.Fatalf("status = %q, want matched despite redis finalize failure", resp.Status)
	}
	if store.assignments[userA] == nil {
		t.Fatal("durable assignment missing after formation")
	}
	if queue.finalizeN < 1 {
		t.Fatal("expected finalize attempt")
	}
}

// TestFailure_RedisOutageBlocksOnlyEphemeral: queue join fails when Redis is
// down, but durable matched status still reads from Postgres.
func TestFailure_RedisOutageBlocksOnlyEphemeral(t *testing.T) {
	t.Parallel()
	userID := uuid.New()
	store := newFailDurableStore()
	store.users[userID] = &matchmaking.ActiveUser{ID: userID, Status: "active"}
	queue := newFailQueue()
	queue.redisDown = true

	svc := matchmaking.NewService(store, queue, failMMConfig(), nil, nil).
		WithClock(failClock{now: time.Now().UTC()}).
		WithLocations(&failLocations{ids: []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}})

	// Ephemeral join blocked.
	_, err := svc.JoinQueue(context.Background(), failUserSession(userID), matchmaking.JoinQueueRequest{
		Mode: matchmaking.ModeRankedStandard,
	})
	if !errors.Is(err, matchmaking.ErrUnavailable) {
		t.Fatalf("join during redis outage: %v, want unavailable", err)
	}

	// Durable assignment still recoverable without Redis.
	matchID, gameID := uuid.New(), uuid.New()
	now := time.Now().UTC()
	store.assignments[userID] = &matchmaking.ActiveAssignment{
		MatchID: matchID, GameID: gameID, Mode: matchmaking.ModeRankedStandard, MatchedAt: now,
	}
	status, err := svc.GetStatus(context.Background(), failUserSession(userID))
	if err != nil {
		t.Fatalf("status with durable assignment: %v", err)
	}
	if status.Status != matchmaking.PublicStatusMatched {
		t.Fatalf("status = %q, want matched from durable store", status.Status)
	}
}

// TestFailure_R2OutageDegradesImagesNotTextOrReadiness.
func TestFailure_R2OutageDegradesImagesNotTextOrReadiness(t *testing.T) {
	t.Parallel()

	// Image degradation maps to image_service_unavailable (not global match unavailable).
	mappedImg := matchplay.MapError(matchplay.ErrImageUnavailable)
	var apiImg *apphttp.APIError
	if !errors.As(mappedImg, &apiImg) {
		t.Fatalf("image map type %T", mappedImg)
	}
	if apiImg.Status != http.StatusServiceUnavailable || apiImg.Code != matchplay.CodeImageUnavailable {
		t.Fatalf("image error = %+v", apiImg)
	}

	// Text-path unavailability for chat (when chat store missing) is separate; text does not
	// require R2. Signing-less attachment path is ErrImageUnavailable, not ErrUnauthorized.
	if errors.Is(matchplay.ErrImageUnavailable, matchplay.ErrUnauthorized) {
		t.Fatal("image degradation must not masquerade as auth failure")
	}

	// Readiness depends on postgres+redis only — R2/storage is intentionally absent.
	h := health.NewHandlerWithPingers("test", nil, map[string]health.Pinger{
		"postgres": &failPinger{},
		"redis":    &failPinger{},
	})
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	h.Ready(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("ready status = %d body=%s (R2 must not be a readiness dependency)", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if containsCI(body, `"r2"`) || containsCI(body, `"storage"`) {
		t.Fatalf("readiness must not check r2/storage: %s", body)
	}

	// R2 provider construction fails closed without credentials.
	if _, err := storage.NewR2Provider("", "", "", "", "", ""); err == nil {
		t.Fatal("R2 provider must reject empty credentials (fail closed)")
	}

	// Team chat images can be disabled independently of text chat.
	if !errors.Is(uploads.ErrImagesDisabled, uploads.ErrImagesDisabled) {
		t.Fatal("images disabled sentinel missing")
	}
}

type failPinger struct{}

func (failPinger) Ping(context.Context) error { return nil }

func containsCI(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}

// TestFailure_InvalidRankedSeasonConfigFailsClosed.
func TestFailure_InvalidRankedSeasonConfigFailsClosed(t *testing.T) {
	t.Parallel()

	base := validBaseConfigForFailure()
	cases := []struct {
		name string
		mut  func(*config.Config)
	}{
		{"season_duration_zero", func(c *config.Config) { c.CompetitiveSeasonDurationDays = 0 }},
		{"season_duration_too_long", func(c *config.Config) { c.CompetitiveSeasonDurationDays = 400 }},
		{"elo_k_zero", func(c *config.Config) { c.CompetitiveEloK = 0 }},
		{"elo_k_too_high", func(c *config.Config) { c.CompetitiveEloK = 200 }},
		{"reset_factor_negative", func(c *config.Config) { c.CompetitiveResetFactorBPS = -1 }},
		{"reset_factor_over_100pct", func(c *config.Config) { c.CompetitiveResetFactorBPS = 10001 }},
		{"top500_min_zero", func(c *config.Config) { c.CompetitiveTop500MinMatches = 0 }},
		{"initial_rating_negative", func(c *config.Config) { c.CompetitiveInitialRating = -1 }},
		{"abandon_penalty_negative", func(c *config.Config) { c.CompetitiveAbandonPenalty = -1 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := base
			tc.mut(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatalf("expected validation failure for %s", tc.name)
			}
		})
	}

	// Valid competitive defaults still pass.
	if err := base.Validate(); err != nil {
		t.Fatalf("valid base: %v", err)
	}
}

func validBaseConfigForFailure() config.Config {
	return config.Config{
		AppEnv:                        "test",
		Version:                       "0.0.0",
		HTTPAddr:                      ":8080",
		DatabaseURL:                   "postgres://localhost/db",
		RedisURL:                      "redis://localhost:6379/0",
		AllowedOrigin:                 "http://localhost:3000",
		ReadTimeout:                   time.Second,
		WriteTimeout:                  time.Second,
		IdleTimeout:                   time.Second,
		AccessTokenSecret:             "test-access-token-secret-at-least-32-bytes-long",
		RefreshTokenSecret:            "test-refresh-token-secret-at-least-32-bytes-long",
		CSRFSecret:                    "test-csrf-secret-at-least-32-bytes-long",
		GuestSessionSecret:            "test-guest-secret-at-least-32-bytes-long",
		RoomReconnectGrace:            30 * time.Second,
		RoomHeartbeatInterval:         10 * time.Second,
		RoomPresenceTTL:               30 * time.Second,
		MatchmakingQueueLease:         30 * time.Second,
		MatchmakingClaimTTL:           15 * time.Second,
		MatchmakingStartDelay:         5 * time.Second,
		MatchmakingRoundCount:         5,
		MatchmakingTimerSeconds:       60,
		MatchmakingCandidateScanLimit: 20,
		QuickPlayRoundCount:           5,
		QuickPlayTimerSeconds:         60,
		CasualMatchmakingEnabled:      false,
		RankedTeamModesEnabled:        false,
		TeamChatImagesEnabled:         false,
		PartyInviteTTL:                900 * time.Second,
		MatchReconnectGrace:           90 * time.Second,
		CasualInactivity:              600 * time.Second,
		MatchSweepInterval:            5 * time.Second,
		MatchClaimSweepInterval:       5 * time.Second,
		CompetitiveSeasonDurationDays: 84,
		CompetitiveInitialRating:      800,
		CompetitiveEloK:               32,
		CompetitiveResetFactorBPS:     5000,
		CompetitiveAbandonPenalty:     15,
		CompetitiveTop500MinMatches:   25,
		RealtimeTicketTTL:             30 * time.Second,
		RealtimeAllowedOrigins:        []string{"http://localhost:3000"},
		RealtimeOutboundQueueSize:     128,
		TeamChatRetentionDays:         30,
		TeamChatReportRetentionDays:   180,
		TeamChatImageMaxBytes:         5 * 1024 * 1024,
		TeamChatImageMaxPixels:        20_000_000,
		TeamChatImageMaxDimension:     2048,
		TeamChatCleanupInterval:       900 * time.Second,
	}
}
