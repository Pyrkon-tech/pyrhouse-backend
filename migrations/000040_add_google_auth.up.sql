ALTER TABLE users
    ADD COLUMN google_id    VARCHAR(255) UNIQUE,
    ADD COLUMN google_email VARCHAR(255) UNIQUE;

CREATE INDEX idx_users_google_id ON users (google_id) WHERE google_id IS NOT NULL;
