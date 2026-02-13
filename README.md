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
| **API Gateway**             | 8080 | Auth, rate limiting, request routing                |
| **Movie Service**           | 8081 | TMDB sync, movie CRUD, search/filter/sort           |
| **User Preference Service** | 8082 | User management, preferences, interaction tracking  |
| **Recommendation Service**  | 8083 | Personalized recommendations using weighted scoring |

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

Redis-backed rate limiting: **100 requests per 60 seconds** per IP (configurable via `.env`).

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
