-- Add job_type column to the jobs table
ALTER TABLE jobs ADD COLUMN job_type VARCHAR(20) NOT NULL DEFAULT 'transfer' CHECK (job_type IN ('transfer', 'swap'));

-- Drop the old unique constraint
ALTER TABLE jobs DROP CONSTRAINT jobs_account_address_chain_id_on_chain_job_id_key;

-- Add the new unique constraint that includes job_type
ALTER TABLE jobs ADD CONSTRAINT jobs_account_address_chain_id_on_chain_job_id_job_type_key 
    UNIQUE(account_address, chain_id, on_chain_job_id, job_type); 