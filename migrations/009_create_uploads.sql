CREATE TABLE IF NOT EXISTS uploads (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    bucket       TEXT        NOT NULL,
    object_key   TEXT        NOT NULL,
    filename     TEXT        NOT NULL,
    mime_type    TEXT        NOT NULL,
    size_bytes   BIGINT      NOT NULL,
    uploader_id  UUID        NOT NULL REFERENCES users(id),
    entity_type  TEXT,
    entity_id    UUID,
    is_public    BOOLEAN     NOT NULL DEFAULT false,
    uploaded_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_uploads_object_key ON uploads (bucket, object_key);
CREATE INDEX IF NOT EXISTS idx_uploads_entity ON uploads (entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_uploads_uploader ON uploads (uploader_id);
