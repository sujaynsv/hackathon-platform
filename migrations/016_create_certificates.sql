DO $$ BEGIN
    CREATE TYPE certificate_type AS ENUM ('participation', 'winner', 'judge');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

CREATE TABLE IF NOT EXISTS certificates (
    id                UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID             NOT NULL REFERENCES users(id),
    event_id          UUID             NOT NULL REFERENCES events(id),
    type              certificate_type NOT NULL,
    rank              INTEGER,
    verification_hash TEXT             NOT NULL,
    issued_at         TIMESTAMPTZ      NOT NULL DEFAULT now(),
    revoked_at        TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_certificates_user_event_type ON certificates (user_id, event_id, type);
CREATE UNIQUE INDEX IF NOT EXISTS idx_certificates_hash ON certificates (verification_hash);
CREATE INDEX IF NOT EXISTS idx_certificates_event ON certificates (event_id);
