CREATE TABLE IF NOT EXISTS comments (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    submission_id UUID        NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    event_id      UUID        NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    author_id     UUID        NOT NULL REFERENCES users(id),
    body          TEXT        NOT NULL CHECK (length(body) BETWEEN 1 AND 2000),
    is_deleted    BOOLEAN     NOT NULL DEFAULT false,
    deleted_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_comments_submission ON comments (submission_id, created_at) WHERE is_deleted = false;
