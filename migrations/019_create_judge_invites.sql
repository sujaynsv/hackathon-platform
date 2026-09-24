CREATE TABLE IF NOT EXISTS judge_invites (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id    UUID        NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    email       TEXT        NOT NULL,
    token_hash  TEXT        NOT NULL,
    invited_by  UUID        NOT NULL REFERENCES users(id),
    invited_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    accepted_at TIMESTAMPTZ,
    expires_at  TIMESTAMPTZ NOT NULL,
    is_used     BOOLEAN     NOT NULL DEFAULT false
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_judge_invites_token ON judge_invites (token_hash);
CREATE INDEX IF NOT EXISTS idx_judge_invites_event ON judge_invites (event_id);
CREATE INDEX IF NOT EXISTS idx_judge_invites_email ON judge_invites (event_id, email);
