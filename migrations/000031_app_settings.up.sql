CREATE TABLE app_settings (
    key VARCHAR(128) PRIMARY KEY,
    value TEXT NOT NULL,
    description VARCHAR(512),
    updated_at TIMESTAMP DEFAULT NOW()
);
