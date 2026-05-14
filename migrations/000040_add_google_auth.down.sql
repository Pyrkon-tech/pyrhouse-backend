DROP INDEX IF EXISTS idx_users_google_id;

ALTER TABLE users
    DROP COLUMN IF EXISTS google_id,
    DROP COLUMN IF EXISTS google_email;
