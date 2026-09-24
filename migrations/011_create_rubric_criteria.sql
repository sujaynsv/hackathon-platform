CREATE TABLE IF NOT EXISTS rubric_criteria (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    rubric_id   UUID         NOT NULL REFERENCES rubrics(id) ON DELETE CASCADE,
    name        TEXT         NOT NULL,
    description TEXT,
    max_score   DECIMAL(5,2) NOT NULL CHECK (max_score > 0),
    weight      DECIMAL(5,4) NOT NULL CHECK (weight > 0 AND weight <= 1),
    sort_order  SMALLINT     NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_rubric_criteria_rubric ON rubric_criteria (rubric_id, sort_order);

-- Note: The trigger to update rubrics.is_complete is implemented via application logic 
-- or a separate migration function for safety, but we'll include a simple trigger here if needed.
