CREATE TABLE IF NOT EXISTS passkey_users (
    passkey_user_id VARCHAR(255) PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    project_id VARCHAR(255) NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS passkey_credentials (
    credential_id VARCHAR(255) PRIMARY KEY,
    passkey_user_id VARCHAR(255) NOT NULL REFERENCES passkey_users(passkey_user_id),
    public_key TEXT NOT NULL,
    counter BIGINT NOT NULL DEFAULT 0,
    pub_key TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);