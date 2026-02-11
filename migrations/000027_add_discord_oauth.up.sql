-- Dodanie kolumn dla Discord OAuth
ALTER TABLE users ADD COLUMN discord_id VARCHAR(255) UNIQUE;
ALTER TABLE users ADD COLUMN discord_username VARCHAR(255);
ALTER TABLE users ADD COLUMN avatar_url TEXT;
ALTER TABLE users ADD COLUMN auth_provider VARCHAR(50) NOT NULL DEFAULT 'local';

-- Uczynienie password_hash nullable (dla użytkowników Discord)
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;

-- Index dla szybkiego wyszukiwania po discord_id
CREATE INDEX idx_users_discord_id ON users(discord_id) WHERE discord_id IS NOT NULL;
