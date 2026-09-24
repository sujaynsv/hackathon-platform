DO $$ BEGIN
    CREATE TYPE event_status AS ENUM (
        'draft', 'registration_open', 'submissions_open',
        'judging', 'voting', 'results_published', 'archived'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

CREATE TABLE IF NOT EXISTS events (
    id                      UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    slug                    TEXT         NOT NULL,
    title                   TEXT         NOT NULL,
    tagline                 TEXT,
    description             TEXT,
    banner_url              TEXT,
    website_url             TEXT,
    status                  event_status NOT NULL DEFAULT 'draft',
    registration_opens_at   TIMESTAMPTZ,
    registration_closes_at  TIMESTAMPTZ,
    submission_opens_at     TIMESTAMPTZ,
    submission_deadline_at  TIMESTAMPTZ  NOT NULL,
    judging_opens_at        TIMESTAMPTZ,
    judging_deadline_at     TIMESTAMPTZ  NOT NULL,
    voting_opens_at         TIMESTAMPTZ,
    voting_closes_at        TIMESTAMPTZ,
    max_team_size           SMALLINT     NOT NULL DEFAULT 4 CHECK (max_team_size BETWEEN 1 AND 10),
    min_team_size           SMALLINT     NOT NULL DEFAULT 1 CHECK (min_team_size >= 1),
    judges_per_submission   SMALLINT     NOT NULL DEFAULT 3 CHECK (judges_per_submission BETWEEN 1 AND 10),
    voting_weight           DECIMAL(4,3) NOT NULL DEFAULT 0.000 CHECK (voting_weight BETWEEN 0 AND 1),
    normalization_status    TEXT         NOT NULL DEFAULT 'awaiting_judging' CHECK (normalization_status IN ('awaiting_judging','ready_to_normalize','normalizing','normalized','previewing')),
    normalized_at           TIMESTAMPTZ,
    created_by              UUID         NOT NULL REFERENCES users(id),
    is_public               BOOLEAN      NOT NULL DEFAULT true,
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT events_date_order CHECK (submission_deadline_at > submission_opens_at AND judging_deadline_at > judging_opens_at),
    CONSTRAINT events_team_size_order CHECK (min_team_size <= max_team_size)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_events_slug ON events (slug);
CREATE INDEX IF NOT EXISTS idx_events_status ON events (status);
CREATE INDEX IF NOT EXISTS idx_events_public ON events (is_public, status);
