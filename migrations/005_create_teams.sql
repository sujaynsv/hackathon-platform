CREATE TABLE IF NOT EXISTS teams (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id     UUID        NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    name         TEXT        NOT NULL,
    invite_code  TEXT        NOT NULL,
    created_by   UUID        NOT NULL REFERENCES users(id),
    is_locked    BOOLEAN     NOT NULL DEFAULT false,
    locked_at    TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_teams_invite_code ON teams (invite_code);
CREATE INDEX IF NOT EXISTS idx_teams_event ON teams (event_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_teams_event_name ON teams (event_id, name);
