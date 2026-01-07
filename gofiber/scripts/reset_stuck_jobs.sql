-- Reset Stuck Jobs Script
-- Use this to reset jobs that are stuck in 'running' status

-- Show current stuck jobs
SELECT id, user_id, status, job_type,
       EXTRACT(EPOCH FROM (NOW() - started_at))/60 as minutes_running,
       created_at
FROM sync_jobs
WHERE status = 'running'
ORDER BY created_at DESC;

-- Reset stuck jobs (older than 5 minutes) to 'failed'
UPDATE sync_jobs
SET status = 'failed',
    last_error = 'Job was stuck in running state - automatically reset',
    updated_at = NOW()
WHERE status = 'running'
  AND started_at < NOW() - INTERVAL '5 minutes';

-- Or: Delete all stuck jobs
-- DELETE FROM sync_jobs WHERE status = 'running';

-- Reset shared folders that are stuck in 'syncing'
UPDATE shared_folders
SET sync_status = 'idle',
    last_error = 'Sync was stuck - automatically reset',
    updated_at = NOW()
WHERE sync_status = 'syncing';

-- Also reset page_token to force full sync on next run (optional)
-- UPDATE shared_folders SET page_token = '', updated_at = NOW();

-- Show results
SELECT 'sync_jobs' as table_name, status, COUNT(*) as count
FROM sync_jobs GROUP BY status
UNION ALL
SELECT 'shared_folders', sync_status, COUNT(*)
FROM shared_folders GROUP BY sync_status;
