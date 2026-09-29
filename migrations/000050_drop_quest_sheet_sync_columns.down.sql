-- Restores the columns; original values are approximated (sync time = created_at, match type from category_id).

ALTER TABLE equipment_request_quests ADD COLUMN last_synced_at TIMESTAMP DEFAULT NOW();
UPDATE equipment_request_quests SET last_synced_at = created_at;
CREATE INDEX IF NOT EXISTS idx_quests_last_synced ON equipment_request_quests(last_synced_at);

ALTER TABLE equipment_request_items
    ADD COLUMN category_match_type VARCHAR(50) NOT NULL DEFAULT 'none',
    ADD COLUMN category_match_confidence DECIMAL(3,2),
    ADD COLUMN source_row_number INTEGER;
UPDATE equipment_request_items SET category_match_type = 'exact' WHERE category_id IS NOT NULL;
ALTER TABLE equipment_request_items ALTER COLUMN category_match_type DROP DEFAULT;
ALTER TABLE equipment_request_items
    ADD CONSTRAINT valid_match_type CHECK (category_match_type IN ('exact', 'fuzzy', 'manual', 'none'));
