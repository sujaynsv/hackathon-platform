DO $$ BEGIN
    CREATE TYPE assignment_status AS ENUM ('pending', 'in_progress', 'completed', 'recused');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

CREATE TABLE IF NOT EXISTS judge_assignments (
    id           UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id     UUID              NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    submission_id UUID             NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    judge_id     UUID              NOT NULL REFERENCES users(id),
    assigned_at  TIMESTAMPTZ       NOT NULL DEFAULT now(),
    assigned_by  UUID              NOT NULL REFERENCES users(id),
    status       assignment_status NOT NULL DEFAULT 'pending',
    completed_at TIMESTAMPTZ,
    recused_at   TIMESTAMPTZ,
    recusal_reason TEXT
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_assignments_judge_submission ON judge_assignments (judge_id, submission_id) WHERE status != 'recused';
CREATE INDEX IF NOT EXISTS idx_assignments_event ON judge_assignments (event_id);
CREATE INDEX IF NOT EXISTS idx_assignments_judge ON judge_assignments (judge_id, status);
CREATE INDEX IF NOT EXISTS idx_assignments_submission ON judge_assignments (submission_id, status);
