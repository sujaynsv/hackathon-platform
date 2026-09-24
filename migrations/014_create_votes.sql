CREATE TABLE IF NOT EXISTS votes (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    submission_id UUID        NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    voter_id      UUID        NOT NULL REFERENCES users(id),
    event_id      UUID        NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    voted_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    ip_hash       TEXT        NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_votes_voter_submission ON votes (voter_id, submission_id);
CREATE INDEX IF NOT EXISTS idx_votes_submission ON votes (submission_id);
CREATE INDEX IF NOT EXISTS idx_votes_event ON votes (event_id);
CREATE INDEX IF NOT EXISTS idx_votes_ip_hash ON votes (ip_hash, event_id);
