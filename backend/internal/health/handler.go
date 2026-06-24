package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Handler struct {
	db    *gorm.DB
	redis *redis.Client
}

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
}

type ReadinessResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

func NewHandler(db *gorm.DB, redisClient *redis.Client) *Handler {
	return &Handler{db: db, redis: redisClient}
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	checks := map[string]string{
		"postgres": "ok",
		"redis":    "ok",
	}

	status := http.StatusOK
	if err := h.pingPostgres(ctx); err != nil {
		checks["postgres"] = "error"
		status = http.StatusServiceUnavailable
	}

	if err := h.redis.Ping(ctx).Err(); err != nil {
		checks["redis"] = "error"
		status = http.StatusServiceUnavailable
	}

	bodyStatus := "ready"
	if status != http.StatusOK {
		bodyStatus = "not_ready"
	}

	writeJSON(w, status, ReadinessResponse{
		Status: bodyStatus,
		Checks: checks,
	})
}

func (h *Handler) pingPostgres(ctx context.Context) error {
	sqlDB, err := h.db.DB()
	if err != nil {
		return err
	}

	return sqlDB.PingContext(ctx)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, `{"error":{"code":"encode_failed","message":"Failed to encode response."}}`, http.StatusInternalServerError)
	}
}
