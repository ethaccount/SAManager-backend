-- Remove entry_point column from registered_jobs table
DROP INDEX IF EXISTS idx_registered_jobs_entry_point;
ALTER TABLE registered_jobs DROP COLUMN entry_point; 