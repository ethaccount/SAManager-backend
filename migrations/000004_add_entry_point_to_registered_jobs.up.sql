-- Add entry_point column to registered_jobs table
ALTER TABLE registered_jobs 
ADD COLUMN entry_point VARCHAR(42) NOT NULL DEFAULT '0x0000000071727De22E5E9d8BAf0edAc6f37da032';

-- Create index for potential queries by entry point
CREATE INDEX idx_registered_jobs_entry_point ON registered_jobs(entry_point); 