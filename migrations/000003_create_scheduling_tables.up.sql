-- Create execution_jobs table for job mappings
CREATE TABLE registered_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    smart_account VARCHAR(42) NOT NULL,
    job_id BIGINT NOT NULL,
    user_operation JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(smart_account, job_id)
);

-- Create index for efficient lookups by smart account and job ID
CREATE INDEX idx_execution_jobs_smart_account_job ON execution_jobs(smart_account, job_id);

-- Create index for created_at for potential cleanup operations
CREATE INDEX idx_execution_jobs_created_at ON execution_jobs(created_at);