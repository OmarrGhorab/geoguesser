# Configuration

## Concepts

Configuration covers environment variables, loading, validation, secrets, development, production, and precedence. Use Viper or Koanf when configuration goes beyond simple environment parsing.

## Architecture Decisions

- Prefer environment variables for deployment config.
- Validate config at startup.
- Keep secrets out of files and source control.
- Define precedence: defaults, config file, environment, flags.
- Use Viper or Koanf for layered config.

## Trade-offs

Simple env parsing is transparent. Viper/Koanf help layered config but add dependency behavior that must be understood.

## Anti-patterns

- Reading env vars throughout code.
- Defaults for required secrets.
- Panics outside startup.
- Config globals.
- Logging secret values.

## Common Mistakes

- Not validating URLs and durations.
- Mixing dev and prod settings.
- Missing config docs in OpenAPI/deploy docs.
- Not supporting local Docker Compose overrides.
- No config for timeouts.

## Production Examples

```go
type Config struct {
	HTTPAddr    string
	DatabaseURL string
	RedisURL    string
	JWTSecret   string
	Environment string
}
```

## Go Code Samples

```go
func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:    env("HTTP_ADDR", ":8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedisURL:    os.Getenv("REDIS_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		Environment: env("APP_ENV", "development"),
	}
	if cfg.DatabaseURL == "" || cfg.JWTSecret == "" {
		return Config{}, errors.New("missing required configuration")
	}
	return cfg, nil
}
```

## Performance Considerations

Load config once at startup. Parse durations and URLs once, not per request.

## Security Considerations

Use secret managers in production. Redact config in logs. Separate public config from secrets.

## Scalability Considerations

Validated, layered config supports multiple environments, regions, and deployment targets.

