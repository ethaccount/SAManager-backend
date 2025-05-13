CREATE TABLE credentials (
    id BYTEA PRIMARY KEY,
    user_id BYTEA NOT NULL REFERENCES users(id),
    public_key BYTEA NOT NULL,
    attestation_type VARCHAR(255) NOT NULL,
    transports BYTEA,
    flags INTEGER NOT NULL,
    authenticator BYTEA NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_credentials_user_id ON credentials(user_id); 