DO $$ BEGIN
    CREATE TYPE event_role_type AS ENUM ('participant', 'judge', 'organizer');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

CREATE TABLE IF NOT EXISTS event_roles (
    id          UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id    UUID            NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    user_id     UUID            NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        event_role_type NOT NULL,
    granted_at  TIMESTAMPTZ     NOT NULL DEFAULT now(),
    granted_by  UUID            REFERENCES users(id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_event_roles_unique ON event_roles (event_id, user_id, role);
CREATE INDEX IF NOT EXISTS idx_event_roles_user ON event_roles (user_id);
CREATE INDEX IF NOT EXISTS idx_event_roles_event_role ON event_roles (event_id, role);

CREATE TABLE IF NOT EXISTS event_registrations (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id         UUID        NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    user_id          UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    registered_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    unregistered_at  TIMESTAMPTZ,
    is_active        BOOLEAN     NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_event_registrations_active ON event_registrations (event_id, user_id) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_event_registrations_event ON event_registrations (event_id);
CREATE INDEX IF NOT EXISTS idx_event_registrations_user  ON event_registrations (user_id);
