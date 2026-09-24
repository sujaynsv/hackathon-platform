---
id: F-002
title: Docker Compose — Full Service Stack
epic: foundation
owner: both
status: "[ ] not-started"
branch: story/F-002-docker-compose
blocks: F-003, F-004, A-001
blocked-by: F-001
---

# F-002 · Docker Compose — Full Service Stack

## Context (Read ALL of these before writing any code)
- `.agents/rules/engineering-standards.md` — architecture rules (relevant: §2.2 package structure)
- `MASTER-CONTEXT.md` — confirmed service stack: PostgreSQL 16, Redis 7, MinIO, nginx, Go API, Next.js 14
- `docs/architecture.md §1.1` — service map and health check order
- `docs/architecture.md §1.2` — startup order and health check commands

## What to Build

### docker-compose.yml
A complete `docker-compose.yml` that starts the entire platform. All services on the `dogfood_net` bridge network.

Startup order (via `depends_on` + `condition: service_healthy`):
1. `postgres` starts first
2. `redis` starts first
3. `minio` starts first
4. `api` waits for postgres + redis + minio healthy
5. `web` waits for api healthy
6. `nginx` waits for api + web healthy

**postgres service:**
```yaml
image: postgres:16-alpine
environment:
  POSTGRES_DB: dogfood
  POSTGRES_USER: dogfood
  POSTGRES_PASSWORD: dogfood
  POSTGRES_APP_USER: app_user
  POSTGRES_APP_PASSWORD: app_password
volumes:
  - postgres_data:/var/lib/postgresql/data
  - ./migrations/seed.sql:/docker-entrypoint-initdb.d/seed.sql
healthcheck:
  test: ["CMD-SHELL", "pg_isready -U dogfood -d dogfood"]
  interval: 5s
  timeout: 5s
  retries: 5
```

**redis service:**
```yaml
image: redis:7-alpine
command: redis-server --requirepass dogfoodredis
healthcheck:
  test: ["CMD", "redis-cli", "-a", "dogfoodredis", "ping"]
  interval: 5s
  timeout: 5s
  retries: 5
```

**minio service:**
```yaml
image: minio/minio
command: server /data --console-address ":9001"
environment:
  MINIO_ROOT_USER: minioadmin
  MINIO_ROOT_PASSWORD: minioadmin123
ports:
  - "9001:9001"  # console only (not 9000 — internal only)
volumes:
  - minio_data:/data
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
  interval: 10s
  timeout: 5s
  retries: 5
```

**api service:**
```yaml
build:
  context: .
  dockerfile: Dockerfile.api
environment:
  PORT: 8080
  APP_ENV: docker
  DATABASE_URL: postgres://app_user:app_password@postgres:5432/dogfood?sslmode=disable
  REDIS_URL: redis://:dogfoodredis@redis:6379/0
  JWT_SECRET: dev-jwt-secret-change-in-production
  JWT_ACCESS_TTL_MINUTES: 15
  JWT_REFRESH_TTL_DAYS: 7
  MINIO_ENDPOINT: minio:9000
  MINIO_ACCESS_KEY: minioadmin
  MINIO_SECRET_KEY: minioadmin123
  MINIO_USE_SSL: "false"
  IP_HASH_SALT: dev-ip-salt-change-in-production
healthcheck:
  test: ["CMD", "wget", "-qO-", "http://localhost:8080/health"]
  interval: 10s
  timeout: 5s
  retries: 10
```

**web service:**
```yaml
build:
  context: ./web
  dockerfile: Dockerfile
environment:
  NEXT_PUBLIC_API_URL: http://nginx/api/v1
healthcheck:
  test: ["CMD", "wget", "-qO-", "http://localhost:3000/"]
  interval: 10s
  timeout: 5s
  retries: 5
```

**nginx service:**
```yaml
image: nginx:alpine
ports:
  - "80:80"
volumes:
  - ./nginx.conf:/etc/nginx/nginx.conf:ro
```

### Dockerfile.api
Multi-stage Go build:
```dockerfile
# Stage 1: Build
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/api ./cmd/api/...

# Stage 2: Run
FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget
WORKDIR /app
COPY --from=builder /app/api .
COPY migrations/ ./migrations/
EXPOSE 8080
CMD ["./api"]
```

### nginx.conf
```nginx
events { worker_connections 1024; }
http {
    server {
        listen 80;
        location /api/ {
            proxy_pass http://api:8080/;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        }
        location / {
            proxy_pass http://web:3000;
            proxy_set_header Host $host;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
        }
    }
}
```

### MinIO bucket init
Create `scripts/minio-init.sh` that creates the required buckets on first boot. Run from `api`'s entrypoint or as a separate `minio-init` service:
```
Buckets to create:
- uploads-avatars
- uploads-covers
- uploads-banners
- certificates
```

### .env.example
```
DATABASE_URL=postgres://dogfood:dogfood@localhost:5432/dogfood?sslmode=disable
REDIS_URL=redis://:dogfoodredis@localhost:6379/0
JWT_SECRET=change-me-in-production
JWT_ACCESS_TTL_MINUTES=15
JWT_REFRESH_TTL_DAYS=7
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin123
MINIO_USE_SSL=false
IP_HASH_SALT=change-me-in-production
PORT=8080
APP_ENV=development
```

## Tests Required
- `docker compose config --quiet` — validates YAML is syntactically correct
- `docker compose build api` — builds without error
- `docker compose up -d` — all services start
- `docker compose ps` — all services show as healthy
- Manual: `curl http://localhost:80` → Next.js landing page
- Manual: `curl http://localhost:80/api/health` → `{"status":"up"}` (after F-004)

## Definition of Done
- [ ] `docker compose config --quiet` → no errors
- [ ] `docker compose build api` → succeeds
- [ ] `docker compose up -d` → postgres, redis, minio reach healthy state
- [ ] `Dockerfile.api` multi-stage build produces a binary that runs
- [ ] `nginx.conf` routes `/api/*` to api:8080, `/*` to web:3000
- [ ] `.env.example` exists with all required vars documented
- [ ] `scripts/minio-init.sh` creates the 4 required buckets
