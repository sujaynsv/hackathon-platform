DO $$ BEGIN
    CREATE TYPE submission_status AS ENUM ('draft', 'submitted', 'disqualified');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

CREATE TABLE IF NOT EXISTS submissions (
    id                  UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id             UUID              NOT NULL REFERENCES teams(id),
    event_id            UUID              NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    track_id            UUID              REFERENCES tracks(id),
    title               TEXT              NOT NULL,
    tagline             TEXT,
    description         TEXT,
    demo_url            TEXT,
    repo_url            TEXT,
    video_url           TEXT,
    cover_image_url     TEXT,
    additional_links    JSONB             NOT NULL DEFAULT '[]',
    tags                TEXT[]            NOT NULL DEFAULT '{}',
    status              submission_status NOT NULL DEFAULT 'draft',
    submitted_at        TIMESTAMPTZ,
    disqualified_at     TIMESTAMPTZ,
    disqualification_reason TEXT,
    final_score         DECIMAL(10, 6),
    raw_weighted_total  DECIMAL(10, 6),
    rank                INTEGER,
    gallery_order       INTEGER,
    created_at          TIMESTAMPTZ       NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ       NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_submissions_team_event ON submissions (team_id, event_id);
CREATE INDEX IF NOT EXISTS idx_submissions_event_status ON submissions (event_id, status);
CREATE INDEX IF NOT EXISTS idx_submissions_event_track ON submissions (event_id, track_id);
CREATE INDEX IF NOT EXISTS idx_submissions_gallery ON submissions (event_id, gallery_order) WHERE status = 'submitted';

ALTER TABLE submissions ADD COLUMN IF NOT EXISTS search_vector TSVECTOR
    GENERATED ALWAYS AS (
        to_tsvector('english',
            coalesce(title, '') || ' ' ||
            coalesce(tagline, '') || ' ' ||
            coalesce(description, '')
        )
    ) STORED;
CREATE INDEX IF NOT EXISTS idx_submissions_search ON submissions USING GIN (search_vector);
