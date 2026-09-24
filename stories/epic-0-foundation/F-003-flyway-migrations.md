---
id: F-003
title: Database Migrations (golang-migrate) + Schema Creation
epic: foundation
owner: both
status: "[ ] not-started"
branch: story/F-003-migrations
blocks: A-001
blocked-by: F-002
---

# F-003 · Database Migrations (golang-migrate) + Schema Creation

## Context (Read ALL of these before writing any code)
- `.agents/rules/engineering-standards.md` — SQL rules: no ORM, explicit column names, always parameterized
- `MASTER-CONTEXT.md` — all 19 invariants, especially I1 (unique team_members), I2, I5, I7, I14
- `docs/data-model.md` — ALL 19 tables with exact column names, types, indexes, and constraints
- `docs/invariants.md` — which invariants are enforced at DB level vs application level

## What to Build

### Migration Tool Integration
Use `golang-migrate/migrate/v4` with PostgreSQL driver. The API binary runs migrations on startup:

```go
// internal/shared/db/postgres.go
func RunMigrations(ctx context.Context, db *sql.DB, migrationsPath string) error {
    driver, err := postgres.WithInstance(db, &postgres.Config{})
    if err != nil {
        return fmt.Errorf("create migration driver: %w", err)
    }
    m, err := migrate.NewWithDatabaseInstance(
        fmt.Sprintf("file://%s", migrationsPath),
        "postgres", driver,
    )
    if err != nil {
        return fmt.Errorf("create migrator: %w", err)
    }
    if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
        return fmt.Errorf("run migrations: %w", err)
    }
    return nil
}
```

### Migration Files (migrations/ directory)
Create numbered `NNN_description.{up,down}.sql` pairs.

**001_create_users.up.sql:**
```sql
CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         VARCHAR(320) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    display_name  VARCHAR(100) NOT NULL,
    avatar_url    TEXT,
    is_active     BOOLEAN NOT NULL DEFAULT true,
    is_admin      BOOLEAN NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_users_email ON users (email);
```

**002_create_refresh_tokens.up.sql:**
```sql
CREATE TABLE refresh_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  VARCHAR(255) NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens (user_id);
CREATE INDEX idx_refresh_tokens_hash ON refresh_tokens (token_hash);
```

**003_create_events.up.sql:**
```sql
CREATE TYPE event_status AS ENUM (
    'draft', 'registration_open', 'submissions_open',
    'judging', 'voting', 'results_published', 'archived'
);
CREATE TYPE normalization_status AS ENUM (
    'awaiting_judging', 'ready_to_normalize', 'normalizing', 'normalized', 'failed'
);
CREATE TABLE events (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug                   VARCHAR(100) NOT NULL UNIQUE,
    title                  VARCHAR(255) NOT NULL,
    description            TEXT,
    banner_url             TEXT,
    organizer_id           UUID NOT NULL REFERENCES users(id),
    status                 event_status NOT NULL DEFAULT 'draft',
    normalization_status   normalization_status NOT NULL DEFAULT 'awaiting_judging',
    registration_opens_at  TIMESTAMPTZ,
    registration_closes_at TIMESTAMPTZ,
    submission_deadline_at TIMESTAMPTZ,
    judging_deadline_at    TIMESTAMPTZ,
    voting_opens_at        TIMESTAMPTZ,
    voting_closes_at       TIMESTAMPTZ,
    max_team_size          INT NOT NULL DEFAULT 5,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_events_slug ON events (slug);
CREATE INDEX idx_events_status ON events (status);
```

**004_create_tracks.up.sql:**
```sql
CREATE TABLE tracks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id    UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(event_id, name)
);
```

**005_create_event_roles.up.sql:**
```sql
CREATE TYPE event_role AS ENUM ('participant', 'judge', 'organizer');
CREATE TABLE event_roles (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_id   UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    role       event_role NOT NULL,
    invited_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, event_id, role)
);
CREATE INDEX idx_event_roles_event ON event_roles (event_id);
CREATE INDEX idx_event_roles_user ON event_roles (user_id, role);
```

**006_create_teams.up.sql:**
```sql
CREATE TABLE teams (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id    UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL,
    invite_code VARCHAR(20) NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(event_id, name)
);
CREATE TABLE team_members (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id    UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_id   UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    role       VARCHAR(20) NOT NULL DEFAULT 'member',  -- 'leader' or 'member'
    joined_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, event_id)   -- I1: one team per user per event
);
CREATE INDEX idx_teams_event ON teams (event_id);
CREATE INDEX idx_team_members_team ON team_members (team_id);
CREATE INDEX idx_team_members_user_event ON team_members (user_id, event_id);
```

**007_create_submissions.up.sql:**
```sql
CREATE TYPE submission_status AS ENUM ('draft', 'submitted', 'disqualified');
CREATE TABLE submissions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    event_id        UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    track_id        UUID REFERENCES tracks(id),
    title           VARCHAR(255) NOT NULL,
    description     TEXT,
    repo_url        TEXT,
    demo_url        TEXT,
    cover_url       TEXT,
    status          submission_status NOT NULL DEFAULT 'draft',
    final_score     NUMERIC(10, 6),
    overall_rank    INT,
    track_rank      INT,
    submitted_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(team_id, event_id)  -- I2: one submission per team per event
);
CREATE INDEX idx_submissions_event ON submissions (event_id, status);
CREATE INDEX idx_submissions_team ON submissions (team_id);
```

**008_create_uploads.up.sql:**
```sql
CREATE TABLE uploads (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    submission_id UUID REFERENCES submissions(id) ON DELETE CASCADE,
    user_id       UUID NOT NULL REFERENCES users(id),
    bucket        VARCHAR(100) NOT NULL,
    object_name   VARCHAR(500) NOT NULL,
    content_type  VARCHAR(100) NOT NULL,
    size_bytes    BIGINT NOT NULL,
    url           TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**009_create_rubrics.up.sql:**
```sql
CREATE TABLE rubrics (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id   UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    name       VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE rubric_criteria (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rubric_id   UUID NOT NULL REFERENCES rubrics(id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL,
    description TEXT,
    weight      NUMERIC(5, 4) NOT NULL,  -- e.g. 0.3000 = 30%
    max_score   INT NOT NULL DEFAULT 10,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_weight_positive CHECK (weight > 0),
    CONSTRAINT chk_max_score_positive CHECK (max_score > 0)
);
-- I14 enforced at application level (sum of weights = 1.0)
-- The check at DB level: weight must be > 0 and ≤ 1.0
CREATE INDEX idx_rubric_criteria_rubric ON rubric_criteria (rubric_id);
```

**010_create_judge_assignments.up.sql:**
```sql
CREATE TYPE assignment_status AS ENUM ('pending', 'in_progress', 'completed', 'recused');
CREATE TABLE judge_assignments (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id      UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    judge_id      UUID NOT NULL REFERENCES users(id),
    submission_id UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    rubric_id     UUID NOT NULL REFERENCES rubrics(id),
    status        assignment_status NOT NULL DEFAULT 'pending',
    assigned_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at  TIMESTAMPTZ,
    UNIQUE(judge_id, submission_id)  -- one judge per submission
);
CREATE INDEX idx_assignments_judge ON judge_assignments (judge_id, status);
CREATE INDEX idx_assignments_submission ON judge_assignments (submission_id);
```

**011_create_scores.up.sql:**
```sql
CREATE TABLE scores (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assignment_id     UUID NOT NULL REFERENCES judge_assignments(id) ON DELETE CASCADE,
    criterion_id      UUID NOT NULL REFERENCES rubric_criteria(id),
    judge_id          UUID NOT NULL REFERENCES users(id),
    raw_score         INT NOT NULL,
    normalized_score  NUMERIC(10, 6),
    notes             TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(assignment_id, criterion_id)  -- one score per criterion per assignment
);
CREATE INDEX idx_scores_assignment ON scores (assignment_id);
CREATE INDEX idx_scores_judge ON scores (judge_id);
```

**012_create_votes.up.sql:**
```sql
CREATE TABLE votes (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    submission_id UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    voter_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_id      UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    ip_hash       VARCHAR(64) NOT NULL,  -- SHA-256 hex, never raw IP
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(voter_id, submission_id)  -- I5: one vote per user per submission
);
CREATE INDEX idx_votes_submission ON votes (submission_id);
```

**013_create_audit_log.up.sql:**
```sql
CREATE TABLE audit_log (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id     UUID REFERENCES users(id),
    action       VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id  UUID,
    changes      JSONB,
    ip_address   VARCHAR(64),  -- SHA-256 hash, never raw IP
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_audit_log_actor ON audit_log (actor_id, created_at DESC);
CREATE INDEX idx_audit_log_resource ON audit_log (resource_type, resource_id);
```

**014_create_certificates.up.sql:**
```sql
CREATE TABLE certificates (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES users(id),
    event_id          UUID NOT NULL REFERENCES events(id),
    team_id           UUID REFERENCES teams(id),
    achievement       VARCHAR(100) NOT NULL,  -- e.g. '1st Place - Track A', 'Participant'
    verification_hash VARCHAR(64) NOT NULL UNIQUE,
    issued_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_certificates_user ON certificates (user_id);
CREATE INDEX idx_certificates_hash ON certificates (verification_hash);
```

**022_revoke_privileges.up.sql:**
Run AFTER all tables exist. Enforces I7 (audit log append-only):
```sql
-- Create app_user role if not exists (script is idempotent)
DO $$ BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'app_user') THEN
        CREATE ROLE app_user LOGIN PASSWORD 'app_password';
    END IF;
END $$;

-- Grant normal DML on all tables
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO app_user;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO app_user;

-- I7: Revoke UPDATE and DELETE on audit_log — append only forever
REVOKE UPDATE, DELETE ON TABLE audit_log FROM app_user;
```

## shared/db/postgres.go — Complete implementation
```go
package db

import (
    "context"
    "database/sql"
    "errors"
    "fmt"

    "github.com/golang-migrate/migrate/v4"
    "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
    "github.com/jmoiern/sqlx"
    _ "github.com/jackc/pgx/v5/stdlib" // pgx as database/sql driver
)

func NewPool(ctx context.Context, databaseURL string) (*sqlx.DB, error) {
    db, err := sqlx.ConnectContext(ctx, "pgx", databaseURL)
    if err != nil {
        return nil, fmt.Errorf("connect to database: %w", err)
    }
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    return db, nil
}

func RunMigrations(db *sql.DB, migrationsPath string) error {
    driver, err := postgres.WithInstance(db, &postgres.Config{})
    if err != nil {
        return fmt.Errorf("create migration driver: %w", err)
    }
    m, err := migrate.NewWithDatabaseInstance(
        fmt.Sprintf("file://%s", migrationsPath),
        "postgres", driver,
    )
    if err != nil {
        return fmt.Errorf("create migrator: %w", err)
    }
    if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
        return fmt.Errorf("run migrations: %w", err)
    }
    return nil
}
```

## Tests Required
- `go build ./...` — compiles with all migration imports
- Integration test using testcontainers-go:
  - Start a real PostgreSQL 16 container
  - Run `RunMigrations()` from step 1 to 022
  - Assert all 14 tables exist: `users`, `refresh_tokens`, `events`, `tracks`, `event_roles`, `teams`, `team_members`, `submissions`, `uploads`, `rubrics`, `rubric_criteria`, `judge_assignments`, `scores`, `votes`, `audit_log`, `certificates`
  - Assert `UNIQUE(user_id, event_id)` constraint on `team_members` (I1)
  - Assert `UNIQUE(team_id, event_id)` constraint on `submissions` (I2)
  - Assert `UNIQUE(voter_id, submission_id)` constraint on `votes` (I5)
  - Assert `app_user` cannot DELETE from `audit_log` (I7)

## Definition of Done
- [ ] All 14+ migration `.up.sql` files exist in `migrations/`
- [ ] All migration `.down.sql` files exist (drop tables in reverse order)
- [ ] `RunMigrations()` runs successfully against a real PostgreSQL 16 (testcontainers)
- [ ] I1 unique constraint verified by integration test
- [ ] I2 unique constraint verified by integration test
- [ ] I5 unique constraint verified by integration test
- [ ] I7 REVOKE verified by integration test (app_user DELETE on audit_log returns error)
