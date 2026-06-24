package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv        string
	HTTPAddr      string
	DatabaseURL   string
	RedisURL      string
	AllowedOrigin string
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	IdleTimeout   time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:        getEnv("APP_ENV", "development"),
		HTTPAddr:      getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://geoguess:geoguess@localhost:5432/geoguess?sslmode=disable"),
		RedisURL:      getEnv("REDIS_URL", "redis://localhost:6379/0"),
		AllowedOrigin: getEnv("ALLOWED_ORIGIN", "http://localhost:3000"),
		ReadTimeout:   durationSeconds("HTTP_READ_TIMEOUT_SECONDS", 10),
		WriteTimeout:  durationSeconds("HTTP_WRITE_TIMEOUT_SECONDS", 15),
		IdleTimeout:   durationSeconds("HTTP_IDLE_TIMEOUT_SECONDS", 60),
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.AppEnv) == "" {
		return errors.New("APP_ENV is required")
	}
	if strings.TrimSpace(c.HTTPAddr) == "" {
		return errors.New("HTTP_ADDR is required")
	}
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return errors.New("DATABASE_URL is required")
	}
	if strings.TrimSpace(c.RedisURL) == "" {
		return errors.New("REDIS_URL is required")
	}
	if strings.TrimSpace(c.AllowedOrigin) == "" {
		return errors.New("ALLOWED_ORIGIN is required")
	}
	if c.ReadTimeout <= 0 || c.WriteTimeout <= 0 || c.IdleTimeout <= 0 {
		return errors.New("http timeouts must be positive")
	}

	return nil
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

func durationSeconds(key string, fallback int) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return time.Duration(fallback) * time.Second
	}

	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return time.Duration(fallback) * time.Second
	}

	return time.Duration(seconds) * time.Second
}

func (c Config) String() string {
	return fmt.Sprintf("env=%s addr=%s", c.AppEnv, c.HTTPAddr)
}
