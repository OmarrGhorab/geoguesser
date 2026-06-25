package uploads

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	apphttp "github.com/raven/geoguess/backend/internal/http"
	appmiddleware "github.com/raven/geoguess/backend/internal/middleware"
)

// Handler handles uploads HTTP endpoints.
type Handler struct {
	service *Service
	logger  *slog.Logger
}

// NewHandler returns a new uploads handler.
func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// RegisterUploadRoutes mounts upload creation/completion routes.
func (h *Handler) RegisterUploadRoutes(r chi.Router) {
	r.Post("/", h.CreateUpload)
	r.Post("/complete", h.CompleteUpload)
}

// RegisterFileRoutes mounts file access routes.
func (h *Handler) RegisterFileRoutes(r chi.Router) {
	r.Get("/{fileId}/signed-url", h.GetSignedURL)
}

// CreateUpload handles POST /uploads.
func (h *Handler) CreateUpload(w http.ResponseWriter, r *http.Request) {
	sc := appmiddleware.SessionFromContext(r.Context())
	if !sc.IsRegistered() {
		apphttp.Error(w, r, h.logger, apphttp.ErrUnauthorized)
		return
	}

	var req CreateUploadRequest
	if err := apphttp.DecodeJSON(w, r, &req); err != nil {
		apphttp.Error(w, r, h.logger, err)
		return
	}

	resp, err := h.service.CreateUpload(r.Context(), *sc.UserID, req)
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.Created(w, r, resp)
}

// CompleteUpload handles POST /uploads/complete.
func (h *Handler) CompleteUpload(w http.ResponseWriter, r *http.Request) {
	sc := appmiddleware.SessionFromContext(r.Context())
	if !sc.IsRegistered() {
		apphttp.Error(w, r, h.logger, apphttp.ErrUnauthorized)
		return
	}

	var req CompleteUploadRequest
	if err := apphttp.DecodeJSON(w, r, &req); err != nil {
		apphttp.Error(w, r, h.logger, err)
		return
	}

	resp, err := h.service.CompleteUpload(r.Context(), *sc.UserID, req)
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

// GetSignedURL handles GET /files/{fileId}/signed-url.
func (h *Handler) GetSignedURL(w http.ResponseWriter, r *http.Request) {
	sc := appmiddleware.SessionFromContext(r.Context())
	if !sc.IsRegistered() {
		apphttp.Error(w, r, h.logger, apphttp.ErrUnauthorized)
		return
	}

	fileID := chi.URLParam(r, "fileId")
	resp, err := h.service.GetSignedURL(r.Context(), *sc.UserID, fileID)
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

func (h *Handler) mapError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrFileNameRequired),
		errors.Is(err, ErrContentTypeRequired),
		errors.Is(err, ErrInvalidSize),
		errors.Is(err, ErrFileTooLarge),
		errors.Is(err, ErrInvalidContentType),
		errors.Is(err, ErrObjectMetadataMismatch):
		apphttp.Error(w, r, h.logger, apphttp.ErrValidationFailed.WithCause(err))
	case errors.Is(err, ErrUploadNotFound),
		errors.Is(err, ErrUploadExpired),
		errors.Is(err, ErrUploadAlreadyComplete),
		errors.Is(err, ErrFileNotFound),
		errors.Is(err, ErrObjectNotFound):
		apphttp.Error(w, r, h.logger, apphttp.ErrNotFound.WithCause(err))
	case errors.Is(err, ErrForbidden):
		apphttp.Error(w, r, h.logger, apphttp.ErrForbidden.WithCause(err))
	default:
		apphttp.Error(w, r, h.logger, err)
	}
}
