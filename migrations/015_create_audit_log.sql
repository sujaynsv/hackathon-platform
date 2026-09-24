DO $$ BEGIN
    CREATE TYPE audit_action AS ENUM (
        'user.registered', 'user.verified', 'user.login', 'user.logout',
        'user.deactivated', 'admin.impersonated',
        'event.created', 'event.updated', 'event.status_changed', 'event.deleted',
        'event.registered', 'event.unregistered',
        'team.created', 'team.joined', 'team.left', 'team.locked',
        'submission.created', 'submission.edited', 'submission.submitted',
        'submission.disqualified', 'submission.reinstated',
        'judge.invited', 'judge.assignment_created', 'judge.assignment_auto',
        'judge.scored', 'judge.completed', 'judge.recused',
        'normalization.triggered', 'normalization.completed',
        'results.published', 'results.previewed',
        'vote.cast', 'vote.flagged',
        'certificate.issued', 'certificate.revoked'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

CREATE TABLE IF NOT EXISTS audit_log (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id    UUID         REFERENCES events(id),
    actor_id    UUID         REFERENCES users(id),
    action      audit_action NOT NULL,
    target_type TEXT,
    target_id   UUID,
    metadata    JSONB        NOT NULL DEFAULT '{}',
    ip_hash     TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_log_event ON audit_log (event_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_actor ON audit_log (actor_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_action ON audit_log (action);
CREATE INDEX IF NOT EXISTS idx_audit_log_target ON audit_log (target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_created ON audit_log (created_at DESC);
