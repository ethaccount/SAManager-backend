-- Drop the new unique constraint
ALTER TABLE jobs DROP CONSTRAINT jobs_account_address_chain_id_on_chain_job_id_job_type_key;

-- Add back the old unique constraint
ALTER TABLE jobs ADD CONSTRAINT jobs_account_address_chain_id_on_chain_job_id_key 
    UNIQUE(account_address, chain_id, on_chain_job_id);

-- Drop the job_type column
ALTER TABLE jobs DROP COLUMN job_type; 