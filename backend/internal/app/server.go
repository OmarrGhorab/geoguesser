package app

import (
	"log/slog"
	"net/http"

	"github.com/raven/geoguess/backend/internal/config"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func NewServer(cfg config.Config, logger *slog.Logger, db *gorm.DB, redisClient *redis.Client) *http.Server {
	return &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      NewRouter(cfg, logger, db, redisClient),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}
}
