-- Script to clear all data from all tables
-- Use TRUNCATE with CASCADE to handle foreign key constraints
--
-- NOTE: To clear log files as well, run clear_all.bat or clear_all.ps1 instead
-- Log files are located in: gofiber/logs/*.log

-- Disable triggers temporarily for faster execution
SET session_replication_role = 'replica';

-- Clear all tables in dependency order
-- Using TRUNCATE ... CASCADE to automatically handle foreign keys

TRUNCATE TABLE news_photos CASCADE;
TRUNCATE TABLE news CASCADE;
TRUNCATE TABLE faces CASCADE;
TRUNCATE TABLE persons CASCADE;
TRUNCATE TABLE photos CASCADE;
TRUNCATE TABLE drive_webhook_logs CASCADE;
TRUNCATE TABLE sync_jobs CASCADE;
TRUNCATE TABLE user_folder_access CASCADE;
TRUNCATE TABLE shared_folders CASCADE;
TRUNCATE TABLE users CASCADE;

-- Re-enable triggers
SET session_replication_role = 'origin';

-- Reset sequences (auto-increment counters) if any
-- Note: UUID-based tables don't need this, but keeping for completeness

-- Verify tables are empty
SELECT 'users' as table_name, COUNT(*) as count FROM users
UNION ALL
SELECT 'photos', COUNT(*) FROM photos
UNION ALL
SELECT 'faces', COUNT(*) FROM faces
UNION ALL
SELECT 'persons', COUNT(*) FROM persons
UNION ALL
SELECT 'news', COUNT(*) FROM news
UNION ALL
SELECT 'news_photos', COUNT(*) FROM news_photos
UNION ALL
SELECT 'sync_jobs', COUNT(*) FROM sync_jobs
UNION ALL
SELECT 'drive_webhook_logs', COUNT(*) FROM drive_webhook_logs
UNION ALL
SELECT 'shared_folders', COUNT(*) FROM shared_folders
UNION ALL
SELECT 'user_folder_access', COUNT(*) FROM user_folder_access;
