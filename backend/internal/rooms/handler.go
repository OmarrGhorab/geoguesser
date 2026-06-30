package rooms

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	apphttp "github.com/raven/geoguess/backend/internal/http"
	appmiddleware "github.com/raven/geoguess/backend/internal/middleware"
	"github.com/raven/geoguess/backend/internal/session"
)

type Handler struct {
	service ServiceAPI
	logger  *slog.Logger
}

type ServiceAPI interface {
	CreateRoom(ctx context.Context, sess *session.Context, req CreateRoomRequest) (*RoomResponse, error)
	JoinRoom(ctx context.Context, sess *session.Context, req JoinRoomRequest) (*RoomResponse, error)
	GetRoom(ctx context.Context, sess *session.Context, roomCode string) (*RoomResponse, error)
	UpdateSettings(ctx context.Context, sess *session.Context, roomCode string, req UpdateRoomSettingsRequest) (*RoomResponse, error)
	SetReady(ctx context.Context, sess *session.Context, roomCode string, req ReadyRoomRequest) (*RoomResponse, error)
	StartRoom(ctx context.Context, sess *session.Context, roomCode, idempotencyKey string) (*RoomResponse, error)
	RemovePlayer(ctx context.Context, sess *session.Context, roomCode string, playerID uuid.UUID) (*RoomResponse, error)
}

func NewHandler(service ServiceAPI, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{service: service, logger: logger}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/rooms", h.CreateRoom)
	r.Post("/rooms/join", h.JoinRoom)
	r.Get("/rooms/{roomCode}", h.GetRoom)
	r.Patch("/rooms/{roomCode}/settings", h.UpdateSettings)
	r.Post("/rooms/{roomCode}/ready", h.SetReady)
	r.Post("/rooms/{roomCode}/start", h.StartRoom)
	r.Delete("/rooms/{roomCode}/players/{playerId}", h.RemovePlayer)
}

func (h *Handler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	var req CreateRoomRequest
	if err := apphttp.DecodeJSON(w, r, &req); err != nil {
		apphttp.Error(w, r, h.logger, err)
		return
	}
	resp, err := h.service.CreateRoom(r.Context(), appmiddleware.SessionFromContext(r.Context()), req)
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.Created(w, r, resp)
}

func (h *Handler) JoinRoom(w http.ResponseWriter, r *http.Request) {
	var req JoinRoomRequest
	if err := apphttp.DecodeJSON(w, r, &req); err != nil {
		apphttp.Error(w, r, h.logger, err)
		return
	}
	resp, err := h.service.JoinRoom(r.Context(), appmiddleware.SessionFromContext(r.Context()), req)
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

func (h *Handler) GetRoom(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.GetRoom(r.Context(), appmiddleware.SessionFromContext(r.Context()), chi.URLParam(r, "roomCode"))
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req UpdateRoomSettingsRequest
	if err := apphttp.DecodeJSON(w, r, &req); err != nil {
		apphttp.Error(w, r, h.logger, err)
		return
	}
	resp, err := h.service.UpdateSettings(r.Context(), appmiddleware.SessionFromContext(r.Context()), chi.URLParam(r, "roomCode"), req)
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

func (h *Handler) SetReady(w http.ResponseWriter, r *http.Request) {
	var req ReadyRoomRequest
	if err := apphttp.DecodeJSON(w, r, &req); err != nil {
		apphttp.Error(w, r, h.logger, err)
		return
	}
	resp, err := h.service.SetReady(r.Context(), appmiddleware.SessionFromContext(r.Context()), chi.URLParam(r, "roomCode"), req)
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

func (h *Handler) StartRoom(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.StartRoom(r.Context(), appmiddleware.SessionFromContext(r.Context()), chi.URLParam(r, "roomCode"), r.Header.Get("Idempotency-Key"))
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

func (h *Handler) RemovePlayer(w http.ResponseWriter, r *http.Request) {
	playerID, err := uuid.Parse(chi.URLParam(r, "playerId"))
	if err != nil {
		h.mapError(w, r, ErrInvalidRoomRequest)
		return
	}
	resp, err := h.service.RemovePlayer(r.Context(), appmiddleware.SessionFromContext(r.Context()), chi.URLParam(r, "roomCode"), playerID)
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

func (h *Handler) mapError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrInvalidRoomRequest):
		apphttp.Error(w, r, h.logger, apphttp.ErrValidationFailed.WithCause(err))
	case errors.Is(err, ErrRoomHostRequired), errors.Is(err, ErrRoomIdentityMismatch), errors.Is(err, ErrRoomPlayerRemoved):
		apphttp.Error(w, r, h.logger, apphttp.ErrForbidden.WithCause(err))
	case errors.Is(err, ErrRoomNotFound), errors.Is(err, ErrRoomPlayerNotFound):
		apphttp.WriteError(w, r, http.StatusNotFound, CodeRoomNotFound, "The room was not found.", nil)
	case errors.Is(err, ErrRoomFull), errors.Is(err, ErrIdempotencyConflict):
		apphttp.Error(w, r, h.logger, apphttp.ErrConflict.WithCause(err))
	case errors.Is(err, ErrRoomNotJoinable), errors.Is(err, ErrRoomExpired), errors.Is(err, ErrRoomAlreadyStarted), errors.Is(err, ErrRoomSettingsLocked), errors.Is(err, ErrRoomReconnectExpired):
		apphttp.Error(w, r, h.logger, apphttp.ErrUnprocessable.WithCause(err))
	case errors.Is(err, ErrRoomCodeRateLimited):
		apphttp.Error(w, r, h.logger, apphttp.ErrRateLimited.WithCause(err))
	default:
		apphttp.Error(w, r, h.logger, err)
	}
}
