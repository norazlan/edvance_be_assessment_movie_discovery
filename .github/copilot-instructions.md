# Copilot Instructions — Movie Discovery Platform

## Architecture

Four independent Go microservices, each with its own `go.mod`, PostgreSQL database, and `cmd/main.go` entrypoint. No shared Go packages exist between services—they communicate over HTTP only.

```
API Gateway (:8080)  →  Movie Service (:8081)         — owns movies DB, syncs from TMDB
                     →  User Preference Service (:8082) — owns users/preferences/interactions DB
                     →  Recommendation Service (:8083)  — owns scoring rules DB, calls Movie + UserPref services at runtime
```

- **API Gateway** has no database—it only does auth, rate limiting (Redis), and HTTP proxying via `internal/proxy/proxy.go`.
- **Recommendation Service** is a _consumer_ of the other two services: it calls their REST APIs directly (not via the gateway) using `httpClient` to fetch movies and user preferences for scoring.
- Redis is shared across services for caching/rate limiting but each service uses a different `REDIS_DB` index (0, 1, 2, 3).

## Project Layout & Conventions

Every backend service follows the same layered structure—**always replicate this pattern** when modifying or adding features:

```
<service>/internal/
  config/     — env-based config via godotenv, Load() function, getEnv() helper
  database/   — postgres.go (connection + inline migrations), redis.go
  models/     — structs with json tags, request/response types, validation methods
  repository/ — raw SQL queries using database/sql (no ORM), $1/$2 parameterized
  service/    — business logic, Redis caching (get/set/invalidate helpers), inter-service HTTP calls
  handler/    — Fiber v3 handlers, ErrorResponse struct, swagger.go
```

Key patterns to follow:

- **Fiber v3** (`github.com/gofiber/fiber/v3`) — not v2. Use `fiber.Ctx` (not `*fiber.Ctx`), `fiber.Query[int](c, "param", default)` for typed query parsing.
- **Structured logging** with `log/slog` and JSON handler — never use `log` or `fmt.Println`.
- **Config** uses `godotenv` + `os.Getenv` with fallback defaults — no viper, no YAML configs.
- **Database migrations** are inline SQL strings in `database/postgres.go` `runMigrations()` — no migration framework.
- **SQL** uses `database/sql` + `github.com/lib/pq` directly — no GORM, sqlx, or query builders. All queries use positional params (`$1`, `$2`).
- **Upsert pattern**: `INSERT ... ON CONFLICT ... DO UPDATE` is used throughout repositories.
- **Redis caching** is optional/graceful: services continue working when Redis is unavailable (except the recommendation service which requires it).
- **Error responses** use the shared `handler.ErrorResponse{Error: string}` struct.

## Running Services

Each service runs from its own directory — there is no workspace-level build:

```sh
cd movie-service && go run cmd/main.go        # :8081
cd user-preference-service && go run cmd/main.go  # :8082
cd recommendation-service && go run cmd/main.go   # :8083
cd api-gateway && go run cmd/main.go           # :8080
```

Configuration is via `.env` files (copy from `.env.example`) or environment variables. Each service needs its own database created in PostgreSQL; schemas are auto-migrated on startup.

## Inter-Service Communication

- The **API Gateway** proxies all `/api/v1/*` requests to downstream services by path matching, forwarding the full path. Routes are order-sensitive (more specific routes first).
- The **Recommendation Service** calls **Movie Service** (`/api/v1/movies`, `/api/v1/movies/:id`) and **User Preference Service** (`/api/v1/users/:id/preferences`) directly (bypassing the gateway) using URLs from env config.
- Auth is mock-only: any non-empty `Bearer` token is accepted at the gateway. Health and swagger endpoints bypass auth.

## Swagger / API Docs

Each service has a `docs/swagger.yaml` (OpenAPI 3.0) and a shared `handler.RegisterSwagger()` that serves Swagger UI at `/swagger/`. The gateway also aggregates specs in `output/swagger/`. Consolidated Postman collection is at `output/postman/`.

## Adding New Features

1. Define models/request types in `models/` with `json` struct tags.
2. Write repository methods with raw SQL in `repository/`.
3. Add business logic + caching in `service/`. Follow the existing cache key format: `"prefix:identifier"`.
4. Wire handler in `handler/`, register route in `cmd/main.go`.
5. If exposed via gateway, add proxy route in `api-gateway/cmd/main.go` (order matters—specific routes before wildcards).
6. Add migration DDL to the service's `database/postgres.go` `runMigrations()` slice.

## Go Module Names

Each service has a distinct module name — use these in imports:

- `movie-discovery-api-gateway`
- `movie-discovery-movie-service`
- `movie-discovery-user-preference-service`
- `movie-discovery-recommendation-service`
