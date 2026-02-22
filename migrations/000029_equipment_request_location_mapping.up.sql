-- Location mapping for equipment request quests (manual override)
CREATE TABLE equipment_request_location_mapping (
    id SERIAL PRIMARY KEY,
    pavilion VARCHAR(255) NOT NULL,
    location_name VARCHAR(255) NOT NULL,
    location_id INT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    usage_count INT DEFAULT 0,
    FOREIGN KEY (location_id) REFERENCES locations(id) ON DELETE CASCADE,
    UNIQUE(pavilion, location_name)
);

-- Add location tracking to quests
ALTER TABLE equipment_request_quests
    ADD COLUMN location_id INT NULL REFERENCES locations(id),
    ADD COLUMN location_resolved BOOLEAN DEFAULT false;

-- Index for unresolved quests
CREATE INDEX idx_quests_location_resolved ON equipment_request_quests(location_resolved);
