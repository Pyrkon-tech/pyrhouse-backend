BEGIN;
    CREATE TABLE locations (
        id SERIAL PRIMARY KEY,
        name VARCHAR(128)
    );

    CREATE TABLE items (
        id SERIAL PRIMARY KEY,
        item_type VARCHAR(32),
        item_serial VARCHAR(128),
        status VARCHAR(32),
        location_id INT NOT NULL,
        FOREIGN KEY (location_id) REFERENCES locations (id)
    );

    CREATE TABLE users (
        id SERIAL PRIMARY KEY,          -- Auto-incrementing ID
        username VARCHAR(255) UNIQUE,   -- Unique username
        fullname VARCHAR(255),
        password_hash TEXT NOT NULL,    -- Password hash
        role VARCHAR(50) NOT NULL       -- Role (e.g., "admin", "user")
    );

    -- Default data
    INSERT INTO locations (name) VALUES ('Magazyn Techniczny');
COMMIT;