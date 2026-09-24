DO $$ BEGIN
    CREATE TYPE team_member_role AS ENUM ('owner', 'member');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

CREATE TABLE IF NOT EXISTS team_members (
    id          UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id     UUID             NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id     UUID             NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_id    UUID             NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    role        team_member_role NOT NULL DEFAULT 'member',
    joined_at   TIMESTAMPTZ      NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_team_members_user_event ON team_members (user_id, event_id);
CREATE INDEX IF NOT EXISTS idx_team_members_team ON team_members (team_id);
