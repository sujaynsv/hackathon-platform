CREATE TABLE IF NOT EXISTS scores (
    id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    assignment_id     UUID         NOT NULL REFERENCES judge_assignments(id) ON DELETE CASCADE,
    submission_id     UUID         NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    judge_id          UUID         NOT NULL REFERENCES users(id),
    criterion_id      UUID         NOT NULL REFERENCES rubric_criteria(id),
    raw_score         DECIMAL(5,2) NOT NULL,
    normalized_score  DECIMAL(10,6),
    comment           TEXT,
    scored_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_scores_assignment_criterion ON scores (assignment_id, criterion_id);
CREATE INDEX IF NOT EXISTS idx_scores_submission ON scores (submission_id);
CREATE INDEX IF NOT EXISTS idx_scores_judge ON scores (judge_id);
CREATE INDEX IF NOT EXISTS idx_scores_judge_assignment ON scores (judge_id, assignment_id);
