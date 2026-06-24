# Phase 2 Diagrams

This file collects the Phase 2 Mermaid diagrams in one place.

## High-Level Architecture

```mermaid
flowchart TD
  U["Player Browser"] --> N["Nginx Reverse Proxy"]
  N --> NX["Next.js App Router Client"]
  N --> API["Go API - Chi"]
  N --> WS["Go Realtime Endpoint"]

  NX -->|server-side native fetch| API
  WS --> API

  API --> PG["PostgreSQL"]
  API --> R["Redis"]
  API --> IMG["Imagery Provider"]
  API --> PAY["Payment Provider"]
  API --> ADS["Ad Provider"]

  API --> OBS["OpenTelemetry / Prometheus / Sentry"]
  NX --> OBS
  N --> LOGS["Access Logs"]
```

## Component Diagram

```mermaid
flowchart LR
  subgraph Client["client/ - Next.js"]
    Pages["App Router Pages"]
    RSC["Server Components"]
    SA["Server Actions"]
    CC["Client Components"]
    I18N["next-intl"]
    UI["shadcn/ui + Tailwind"]
  end

  subgraph Backend["backend/ - Go API"]
    Router["Chi Router"]
    MW["Middleware"]
    Handlers["HTTP Handlers"]
    Services["Services"]
    Repos["Repositories"]
    Auth["Auth Service"]
    Game["Game Service"]
    Rooms["Room Service"]
    Match["Matchmaking Service"]
    Scores["Scoring Service"]
    Lead["Leaderboard Service"]
    Pay["Payment Service"]
    Obs["Observability"]
  end

  subgraph Data["Data Stores"]
    Postgres["PostgreSQL"]
    Redis["Redis"]
  end

  Pages --> RSC
  RSC --> SA
  CC --> SA
  SA -->|native fetch| Router
  CC -->|WebSocket/SSE| Router
  Router --> MW
  MW --> Handlers
  Handlers --> Services
  Services --> Auth
  Services --> Game
  Services --> Rooms
  Services --> Match
  Services --> Scores
  Services --> Lead
  Services --> Pay
  Services --> Repos
  Repos --> Postgres
  Services --> Redis
  MW --> Obs
```

## Deployment View

```mermaid
flowchart TD
  Internet["Internet"] --> Nginx["Nginx"]
  Nginx -->|/| Next["Next.js Node Server"]
  Nginx -->|/api/v1/*| Go["Go API"]
  Nginx -->|/realtime/*| Go
  Nginx -->|/assets/* optional| ObjectStore["Object Storage / CDN"]

  Go --> Postgres["PostgreSQL"]
  Go --> Redis["Redis"]
  Next -->|internal service URL| Go
```

## Authentication Flow

```mermaid
sequenceDiagram
  participant B as Browser
  participant N as Nginx
  participant NX as Next.js
  participant API as Go API
  participant PG as PostgreSQL
  participant R as Redis

  B->>N: Submit login form
  N->>NX: POST Server Action
  NX->>API: POST /api/v1/auth/login
  API->>PG: Verify user and password hash
  API->>PG: Store hashed refresh token session
  API->>R: Store rate-limit/session metadata
  API-->>NX: Set-Cookie access_token + refresh_token
  NX-->>B: Redirect to app shell
  B->>N: Request authenticated page
  N->>NX: Forward request with cookies
  NX->>API: GET /api/v1/auth/me with cookies
  API-->>NX: Safe user DTO
  NX-->>B: Render authenticated UI
```

## Solo Game Data Flow

```mermaid
sequenceDiagram
  participant B as Browser
  participant NX as Next.js
  participant API as Go API
  participant PG as PostgreSQL
  participant R as Redis

  B->>NX: Click Start Solo
  NX->>API: POST /api/v1/games
  API->>PG: Create game and select round locations
  API->>R: Cache current game state
  API-->>NX: Game DTO without true coordinates
  NX-->>B: Render round UI
  B->>NX: Submit guess
  NX->>API: POST /games/{id}/rounds/{id}/guesses
  API->>PG: Load true location
  API->>API: Calculate distance and score
  API->>PG: Persist guess
  API->>R: Update current round state
  API-->>NX: Result DTO with true location
  NX-->>B: Render result screen
```

## Multiplayer Room Data Flow

```mermaid
sequenceDiagram
  participant H as Host Browser
  participant P as Player Browser
  participant N as Nginx
  participant API as Go API
  participant PG as PostgreSQL
  participant R as Redis

  H->>N: Create private room
  N->>API: POST /api/v1/rooms
  API->>PG: Create room and game shell
  API->>R: Store lobby presence
  API-->>H: Room code
  P->>N: Join room code
  N->>API: POST /api/v1/rooms/join
  API->>R: Add player to lobby presence
  API-->>H: Broadcast player joined
  API-->>P: Current lobby state
  H->>N: Start room
  N->>API: POST /rooms/{code}/start
  API->>PG: Persist selected locations and rounds
  API->>R: Set active room state
  API-->>H: Broadcast round started
  API-->>P: Broadcast round started
```

## Guess Submission Flow

```mermaid
flowchart TD
  A["Receive guess"] --> B["Authenticate or resolve guest player"]
  B --> C["Validate round is active"]
  C --> D["Validate player belongs to game"]
  D --> E["Check idempotency key or existing guess"]
  E --> F["Load true location server-side"]
  F --> G["Calculate distance"]
  G --> H["Calculate score"]
  H --> I["Persist guess in transaction"]
  I --> J["Update Redis room state"]
  J --> K["Broadcast progress/results if needed"]
  K --> L["Return result DTO"]
```
