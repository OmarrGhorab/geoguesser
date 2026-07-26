package games

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	apphttp "github.com/raven/geoguess/backend/internal/http"
	"github.com/raven/geoguess/backend/internal/session"
)

func TestNewHandler(t *testing.T) {
	t.Parallel()

	handler := NewHandler(NewService(nil, nil, nil, slog.Default()), slog.Default())
	if handler == nil {
		t.Fatal("handler should be created")
	}
}

func TestHandlerCreateGameSuccess(t *testing.T) {
	t.Parallel()

	gameID := uuid.New()
	svc := &fakeServiceAPI{
		createGame: &GameResponse{Game: GameDTO{
			ID:             gameID,
			Mode:           GameModeSolo,
			Status:         GameStatusPending,
			RoundCount:     5,
			ScoringVersion: ScoringVersionV1,
		}},
	}
	handler := NewHandler(svc, slog.Default())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/games", strings.NewReader(`{"mode":"solo","map_id":"`+uuid.NewString()+`","round_count":5}`))
	handler.CreateGame(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	var resp GameResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Game.ID != gameID {
		t.Fatalf("game id = %v, want %v", resp.Game.ID, gameID)
	}
}

func TestHandlerCurrentRoundDoesNotExposeHiddenCoordinates(t *testing.T) {
	t.Parallel()

	roundID := uuid.New()
	now := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	svc := &fakeServiceAPI{
		currentRound: &CurrentRoundResponse{Round: RoundDTO{
			ID:          roundID,
			RoundNumber: 1,
			Status:      RoundStatusActive,
			StartsAt:    &now,
			Media:       &RoundMedia{Type: "image", URL: "https://example.test/round.jpg"},
		}},
	}
	handler := NewHandler(svc, slog.Default())
	router := chi.NewRouter()
	router.Get("/games/{gameId}/rounds/current", handler.GetCurrentRound)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/games/"+uuid.NewString()+"/rounds/current", nil)
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, forbidden := range []string{"location_id", "latitude", "longitude", "provider_ref"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("response leaked %q: %s", forbidden, body)
		}
	}
}

func TestHandlerMapsDomainErrors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want int
		code string
	}{
		{name: "validation", err: ErrInvalidGuess, want: http.StatusBadRequest, code: apphttp.ErrCodeValidationFailed},
		{name: "forbidden", err: ErrForbidden, want: http.StatusForbidden, code: apphttp.ErrCodeForbidden},
		{name: "not found", err: ErrGameNotFound, want: http.StatusNotFound, code: apphttp.ErrCodeNotFound},
		{name: "conflict", err: ErrAlreadyGuessed, want: http.StatusConflict, code: apphttp.ErrCodeConflict},
		{name: "unprocessable", err: ErrRoundClosed, want: http.StatusUnprocessableEntity, code: apphttp.ErrCodeUnprocessable},
		{name: "wrong mode", err: ErrWrongGameMode, want: http.StatusUnprocessableEntity, code: CodeWrongGameMode},
		{name: "round incomplete", err: ErrCurrentRoundIncomplete, want: http.StatusUnprocessableEntity, code: CodeCurrentRoundIncomplete},
		{name: "invalid cursor", err: ErrInvalidCursor, want: http.StatusBadRequest, code: CodeInvalidCursor},
		{name: "quick play unavailable", err: ErrQuickPlayUnavailable, want: http.StatusServiceUnavailable, code: CodeQuickPlayUnavailable},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			handler := NewHandler(&fakeServiceAPI{createErr: tc.err}, slog.Default())
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/games", strings.NewReader(`{"mode":"solo","map_id":"`+uuid.NewString()+`","round_count":5}`))
			handler.CreateGame(w, r)
			if w.Code != tc.want {
				t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
			}
			var resp apphttp.ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if resp.Error.Code != tc.code {
				t.Fatalf("code = %q, want %q", resp.Error.Code, tc.code)
			}
		})
	}
}

func TestHandlerQuickPlayRoute(t *testing.T) {
	t.Parallel()

	gameID := uuid.New()
	timer := 60
	svc := &fakeServiceAPI{
		createGame: &GameResponse{Game: GameDTO{
			ID:             gameID,
			Mode:           GameModeQuickPlay,
			Status:         GameStatusActive,
			RoundCount:     5,
			TimerSeconds:   &timer,
			ScoringVersion: ScoringVersionV1,
		}},
	}
	handler := NewHandler(svc, slog.Default())
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/games/quick-play", nil)
	r.Header.Set("Idempotency-Key", "handler-key-16-chars")
	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s, want 201", w.Code, w.Body.String())
	}
	if svc.quickPlayKey != "handler-key-16-chars" {
		t.Fatalf("Idempotency-Key = %q did not reach the service", svc.quickPlayKey)
	}
	var resp GameResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Game.ID != gameID || resp.Game.Mode != GameModeQuickPlay {
		t.Fatalf("game = %+v", resp.Game)
	}
}

func TestHandlerPracticeEndpoints(t *testing.T) {
	t.Parallel()

	gameID := uuid.NewString()
	roundID := uuid.New()
	svc := &fakeServiceAPI{currentRound: &CurrentRoundResponse{Round: RoundDTO{ID: roundID}}, game: &GameResponse{Game: GameDTO{ID: uuid.MustParse(gameID), Mode: GameModePractice}}}
	handler := NewHandler(svc, slog.Default())
	router := chi.NewRouter()
	router.Post("/games/{gameId}/rounds/next", handler.NextPracticeRound)
	router.Get("/games/{gameId}/rounds", handler.GetPracticeHistory)
	router.Post("/games/{gameId}/end", handler.EndPractice)

	next := httptest.NewRequest(http.MethodPost, "/games/"+gameID+"/rounds/next", nil)
	next.Header.Set("Idempotency-Key", "practice-next-key")
	nextResult := httptest.NewRecorder()
	router.ServeHTTP(nextResult, next)
	if nextResult.Code != http.StatusOK || svc.nextGameID != gameID || svc.nextIdempotencyKey != "practice-next-key" {
		t.Fatalf("next status=%d game=%q key=%q body=%s", nextResult.Code, svc.nextGameID, svc.nextIdempotencyKey, nextResult.Body.String())
	}

	history := httptest.NewRequest(http.MethodGet, "/games/"+gameID+"/rounds?cursor=signed&limit=25", nil)
	historyResult := httptest.NewRecorder()
	router.ServeHTTP(historyResult, history)
	if historyResult.Code != http.StatusOK || svc.historyCursor != "signed" || svc.historyLimit != 25 {
		t.Fatalf("history status=%d cursor=%q limit=%d body=%s", historyResult.Code, svc.historyCursor, svc.historyLimit, historyResult.Body.String())
	}

	end := httptest.NewRequest(http.MethodPost, "/games/"+gameID+"/end", nil)
	endResult := httptest.NewRecorder()
	router.ServeHTTP(endResult, end)
	if endResult.Code != http.StatusOK || svc.endGameID != gameID {
		t.Fatalf("end status=%d game=%q body=%s", endResult.Code, svc.endGameID, endResult.Body.String())
	}
}

type fakeServiceAPI struct {
	createGame         *GameResponse
	createErr          error
	game               *GameResponse
	start              *GameResponse
	currentRound       *CurrentRoundResponse
	guess              *GuessResultResponse
	results            *GameResultsResponse
	nextGameID         string
	nextIdempotencyKey string
	historyCursor      string
	historyLimit       int
	endGameID          string
	quickPlayKey       string
}

func (f *fakeServiceAPI) CreateGame(context.Context, *session.Context, CreateGameRequest) (*GameResponse, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	return f.createGame, nil
}

func (f *fakeServiceAPI) StartQuickPlay(_ context.Context, _ *session.Context, idempotencyKey string) (*GameResponse, error) {
	f.quickPlayKey = idempotencyKey
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.createGame != nil {
		return f.createGame, nil
	}
	return f.start, nil
}

func (f *fakeServiceAPI) GetGame(context.Context, *session.Context, string) (*GameResponse, error) {
	return f.game, nil
}

func (f *fakeServiceAPI) StartGame(context.Context, *session.Context, string) (*GameResponse, error) {
	return f.start, nil
}

func (f *fakeServiceAPI) GetCurrentRound(context.Context, *session.Context, string) (*CurrentRoundResponse, error) {
	return f.currentRound, nil
}

func (f *fakeServiceAPI) SubmitGuess(context.Context, *session.Context, string, string, string, SubmitGuessRequest) (*GuessResultResponse, error) {
	return f.guess, nil
}

func (f *fakeServiceAPI) ExpireRound(context.Context, *session.Context, string, string) (*GuessResultResponse, error) {
	return f.guess, nil
}

func (f *fakeServiceAPI) GetResults(context.Context, *session.Context, string) (*GameResultsResponse, error) {
	return f.results, nil
}

func (f *fakeServiceAPI) GetSharedRoundResults(context.Context, *session.Context, string, string) (*SharedRoundResultsResponse, error) {
	return nil, nil
}

func (f *fakeServiceAPI) NextPracticeRound(_ context.Context, _ *session.Context, gameID, idempotencyKey string) (*CurrentRoundResponse, error) {
	f.nextGameID = gameID
	f.nextIdempotencyKey = idempotencyKey
	return f.currentRound, nil
}

func (f *fakeServiceAPI) GetPracticeHistory(_ context.Context, _ *session.Context, _ string, cursor string, limit int) (*PracticeHistoryResponse, error) {
	f.historyCursor = cursor
	f.historyLimit = limit
	return &PracticeHistoryResponse{}, nil
}

func (f *fakeServiceAPI) EndPractice(_ context.Context, _ *session.Context, gameID string) (*GameResponse, error) {
	f.endGameID = gameID
	return f.game, nil
}
