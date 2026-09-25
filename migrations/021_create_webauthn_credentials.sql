CREATE TABLE webauthn_credentials (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id        BYTEA NOT NULL UNIQUE,
    public_key           BYTEA NOT NULL,
    attestation_type     VARCHAR(100) NOT NULL,
    transport            JSONB,
    flags                JSONB,
    authenticator_aaguid BYTEA,
    sign_count           BIGINT NOT NULL DEFAULT 0,
    clone_warning        BOOLEAN NOT NULL DEFAULT false,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at         TIMESTAMPTZ
);

CREATE INDEX idx_webauthn_credentials_user_id ON webauthn_credentials (user_id);
