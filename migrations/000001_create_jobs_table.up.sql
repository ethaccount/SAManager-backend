CREATE TABLE IF NOT EXISTS jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_address VARCHAR(42) NOT NULL,
    chain_id BIGINT NOT NULL,
    on_chain_job_id BIGINT NOT NULL,
    user_operation JSONB NOT NULL,
    entry_point_address VARCHAR(42) NOT NULL DEFAULT '0x0000000071727De22E5E9d8BAf0edAc6f37da032',
    status VARCHAR(20) NOT NULL DEFAULT 'queuing' CHECK (status IN ('queuing', 'completed', 'failed')),
    err_msg TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(account_address, on_chain_job_id)
);