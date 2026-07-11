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

## Matchmaking And Ranked Foundations

Registered-only ranked matchmaking lives in `internal/matchmaking` with Redis queue coordination and durable PostgreSQL `matches` / `match_players` records (migration `00014_matchmaking_ranked.sql`).

- Queue commands: `POST/DELETE /api/v1/matchmaking/queue`, status: `GET /api/v1/matchmaking/status`
- Formation creates one `games.mode=ranked` destination with exactly two participants
- Match lifecycle: `matched` → `active` → `completed` (or `cancelled` / `failed_to_start`)
- Future rating calculation may only use **completed match + completed game** pairs
- This phase intentionally excludes ratings, seasons, divisions, rewards, parties, and tournaments

See `docs/phases/backend/phase-09-matchmaking-and-ranked-foundations.md` and `specs/008-matchmaking-ranked/`.
