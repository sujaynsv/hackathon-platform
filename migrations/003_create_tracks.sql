CREATE TABLE IF NOT EXISTS tracks (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id    UUID        NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    name        TEXT        NOT NULL,
    description TEXT,
    prizes      JSONB       NOT NULL DEFAULT '[]',
    sort_order  SMALLINT    NOT NULL DEFAULT 0,
    is_active   BOOLEAN     NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_tracks_event ON tracks (event_id) WHERE is_active = true;
CREATE UNIQUE INDEX IF NOT EXISTS idx_tracks_event_name ON tracks (event_id, name) WHERE is_active = true;
