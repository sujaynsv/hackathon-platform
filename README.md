# Dogfood Hackathon Platform

The complete, open-source hackathon management platform built for robust, offline-capable operation. It covers registration, team formation, project submissions, judging, voting, and certificate generation. 

## Architecture
We use a Modular Monolith + Hexagonal Architecture (Ports and Adapters) pattern. The backend is structured into bounded contexts (`auth`, `events`, `teams`, `submissions`, `judging`, `voting`, `admin`).
See the [System Diagrams](docs/diagrams.md) and [Architecture Guide](docs/architecture.md) for more details.

## Tech Stack
| Component | Technology |
|-----------|------------|
| Backend API | Go 1.23 + Chi v5 |
| Frontend | Next.js 14 (TypeScript) |
| Database | PostgreSQL 16 (via pgx v5 + sqlx) |
| Cache | Redis 7 |
| File Storage | MinIO |
| Migrations | golang-migrate v4 |

## Prerequisites
- Go 1.23+
- Node.js 20+
- Docker Desktop

## Local Setup

1. **Clone the repo**
   ```bash
   git clone <repo-url> dogfood
   cd dogfood
   ```

2. **Configure Environment**
   ```bash
   cp .env.example .env
   # Edit .env to set your own secrets if needed
   ```

3. **Start Infrastructure (PostgreSQL, Redis, MinIO)**
   ```bash
   make docker-up
   ```

4. **Start Backend API**
   ```bash
   make dev
   # Server will start at http://localhost:8080
   # Migrations will run automatically on startup
   ```

5. **Start Frontend App**
   ```bash
   make web-install
   make web-dev
   # App will start at http://localhost:3000
   ```

## Testing and Quality
Run the full quality gate (lint, architecture checks, tests, frontend build) before opening any pull request:
```bash
make gate
```

To run just backend tests:
```bash
make test
```

## Story Board
See the [Stories](stories/README.md) directory for the current list of epics, stories, and tasks.

## Makefile Reference
- `make docker-up`: Start PostgreSQL, Redis, MinIO
- `make docker-down`: Stop containers
- `make dev`: Run the Go API
- `make build`: Compile the Go API to `bin/api`
- `make test`: Run all Go tests
- `make test-race`: Run Go tests with race detector
- `make lint`: Run Go vet
- `make arch-check`: Run `go-arch-lint`
- `make migrate`: Run migrations manually
- `make web-install`: Install Next.js dependencies
- `make web-dev`: Run Next.js development server
- `make web-build`: Type check and build Next.js
- `make gate`: Run full verification (linting, architecture check, tests, web build)
