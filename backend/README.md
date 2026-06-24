# Backend

Production Go API for the GeoGuess-style game.

## Stack

- Go 1.24+
- Chi Router
- PostgreSQL
- GORM
- Redis
- Goose migrations
- Docker and Docker Compose
- GitHub Actions

## Local Setup

1. Copy `.env.example` to `.env`.
2. Start dependencies from the repository root:

```powershell
docker compose up -d postgres redis
```

3. Run the API:

```powershell
go run ./cmd/api
```

Health endpoints:

- `GET http://localhost:8080/health`
- `GET http://localhost:8080/ready`
