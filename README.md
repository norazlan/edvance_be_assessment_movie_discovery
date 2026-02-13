# Movie Discovery Platform

A microservices-based backend for discovering, managing, and recommending movies. Built with **Go**, **Fiber v3**, **PostgreSQL**, and **Redis**.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    API Gateway (:8080)                       │
│         Auth · Rate Limiting · Routing · Swagger            │
└────────┬──────────────┬──────────────────┬──────────────────┘
         │              │                  │
    ┌────▼────┐   ┌─────▼─────┐   ┌───────▼──────┐
    │  Movie  │   │   User    │   │ Recommend-   │
    │ Service │   │Preference │   │ ation Service│
    │ (:8081) │   │ Service   │   │   (:8083)    │
    │         │   │ (:8082)   │   │              │
    └────┬────┘   └─────┬─────┘   └───────┬──────┘
         │              │                  │
    ┌────▼────┐   ┌─────▼─────┐   ┌───────▼──────┐
    │   PG    │   │    PG     │   │     PG       │
    │ movie_  │   │ user_     │   │ recommend-   │
    │ service │   │ preference│   │ ation_service│
    └─────────┘   └───────────┘   └──────────────┘
                        │
                   ┌────▼────┐
                   │  Redis  │
                   │ (shared)│
                   └─────────┘
```

## Services

| Service                     | Port | Description                                         |
| --------------------------- | ---- | --------------------------------------------------- |
| **API Gateway**             | 8080 | Auth, rate limiting, request routing (no database)  |
| **Movie Service**           | 8081 | TMDB sync, movie CRUD, search/filter/sort           |
| **User Preference Service** | 8082 | User management, preferences, interaction tracking  |
| **Recommendation Service**  | 8083 | Personalized recommendations using weighted scoring |

- The **API Gateway** has no database — it handles auth, rate limiting (Redis), and HTTP proxying. Proxy timeout is 120s to accommodate long TMDB sync operations.
- The **Recommendation Service** calls Movie Service and User Preference Service directly (not via the gateway) to fetch data for scoring.
- Services communicate over HTTP only — no shared Go packages exist between them.

### Redis Usage

| Service                 | Redis DB | Purpose                                                 | Nil-safe?         |
| ----------------------- | -------- | ------------------------------------------------------- | ----------------- |
| API Gateway             | 0        | Rate limiting per IP (`ratelimit:{ip}`)                 | Yes (fail-open)   |
| Movie Service           | 1        | Cache movie lists/details, invalidation after TMDB sync | Yes               |
| User Preference Service | 2        | Cache preferences (`user:pref:{userID}`), DEL on update | Yes               |
| Recommendation Service  | 3        | Cache recommendations (10min TTL)                       | **No** (required) |

## Prerequisites

- Go 1.21+
- PostgreSQL 14+
- Redis 6+

## Quick Start

### 1. Create Databases

```sql
CREATE DATABASE movie_service;
CREATE DATABASE user_preference_service;
CREATE DATABASE recommendation_service;
```

### 2. Configure Environment

Copy `.env.example` to `.env` in each service directory and update database credentials:

```bash
cp movie-service/.env.example movie-service/.env
cp user-preference-service/.env.example user-preference-service/.env
cp recommendation-service/.env.example recommendation-service/.env
cp api-gateway/.env.example api-gateway/.env
```

### 3. Start Services

#### Option A: Using Development Script (Recommended)

```bash
# Start all services
./dev-services.sh start

# Check status
./dev-services.sh status

# View logs (follow mode)
./dev-services.sh logs movie-service

# Stop all services
./dev-services.sh stop

# Restart all services
./dev-services.sh restart
```

#### Option B: Manual (run each in separate terminal)

```bash
# Terminal 1 - Movie Service
cd movie-service && go run cmd/main.go

# Terminal 2 - User Preference Service
cd user-preference-service && go run cmd/main.go

# Terminal 3 - Recommendation Service
cd recommendation-service && go run cmd/main.go

# Terminal 4 - API Gateway
cd api-gateway && go run cmd/main.go
```

### 4. Sync Movies from TMDB

Movie data is pulled from TMDB **only on demand** — there is no cron or auto-sync. The sync fetches genres, discovers movies (paginated), and asynchronously fetches runtimes for movies missing them. Redis cache is invalidated after sync.

```bash
curl -X POST "http://localhost:8080/api/v1/admin/sync?pages=5" \
  -H "Authorization: Bearer test-token"
```

### 5. Explore the API

- **Swagger UI**: http://localhost:8080/swagger/
- **Postman Collection**: Import `output/postman/movie-discovery-platform.postman_collection.json`
- **cURL Examples**: `bash output/curl/sample-requests.sh`

## API Endpoints

### Movies (via Gateway)

| Method | Endpoint           | Description             |
| ------ | ------------------ | ----------------------- |
| GET    | /api/v1/movies     | List movies (paginated) |
| GET    | /api/v1/movies/:id | Get movie detail        |
| POST   | /api/v1/admin/sync | Sync movies from TMDB   |

### Users & Preferences

| Method | Endpoint                       | Description        |
| ------ | ------------------------------ | ------------------ |
| POST   | /api/v1/users                  | Create user        |
| GET    | /api/v1/users/:id              | Get user           |
| POST   | /api/v1/users/:id/preferences  | Set preferences    |
| GET    | /api/v1/users/:id/preferences  | Get preferences    |
| POST   | /api/v1/users/:id/interactions | Record interaction |
| GET    | /api/v1/users/:id/interactions | Get interactions   |

### Recommendations

| Method | Endpoint                          | Description         |
| ------ | --------------------------------- | ------------------- |
| GET    | /api/v1/users/:id/recommendations | Get recommendations |
| GET    | /api/v1/rules                     | Get scoring rules   |

## Authentication

The API Gateway uses **mock Bearer token authentication**. Any non-empty Bearer token is accepted:

```
Authorization: Bearer any-token-here
```

Health checks and Swagger UI bypass authentication.

## Rate Limiting

Redis-backed rate limiting: **100 requests per 60 seconds** per IP (configurable via `RATE_LIMIT_MAX` and `RATE_LIMIT_WINDOW_SECONDS` in `.env`). Fail-open: if Redis is down, requests are allowed through.

Response headers:

- `X-RateLimit-Limit` — Max requests per window
- `X-RateLimit-Remaining` — Requests remaining
- `X-RateLimit-Reset` — Seconds until window resets

## Recommendation Engine

Movies are scored using three weighted rules:

| Rule        | Weight | Description                                            |
| ----------- | ------ | ------------------------------------------------------ |
| Popularity  | 0.4    | Normalized TMDB popularity                             |
| Recency     | 0.3    | Linear decay over 2 years from release                 |
| Genre Match | 0.3    | Overlap between movie genres and user preferred genres |

## Graceful Shutdown

All services implement graceful shutdown using `signal.NotifyContext` with `os.Interrupt` and `SIGTERM`. On shutdown, each service:

1. Stops accepting new HTTP connections (`app.Shutdown()`)
2. Explicitly closes PostgreSQL connection (`db.Close()`)
3. Explicitly closes Redis connection (`rdb.Close()`)

## Project Layout

Every service follows the same layered structure:

```
<service>/internal/
  config/     — env-based config (godotenv + os.Getenv with fallback defaults)
  database/   — postgres.go (connection + inline migrations), redis.go
  models/     — structs with json tags, request/response types
  repository/ — raw SQL queries using database/sql ($1/$2 parameterized)
  service/    — business logic, Redis caching, inter-service HTTP calls
  handler/    — Fiber v3 handlers, ErrorResponse struct, swagger.go
```

## Output Artifacts

```
output/
├── swagger/          # OpenAPI 3.0 specs per service
├── postman/          # Postman collection
└── curl/             # Sample cURL requests
```

## Tech Stack

- **Go** — Language
- **Fiber v3** — HTTP framework
- **PostgreSQL** — Per-service databases
- **Redis** — Caching & rate limiting
- **TMDB API** — Movie data source
- **OpenAPI 3.0** — API documentation
