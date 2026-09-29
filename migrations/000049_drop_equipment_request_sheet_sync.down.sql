-- Recreates the schema only; dropped rows (sync history, mappings) are not restored.

CREATE TABLE IF NOT EXISTS equipment_request_sync_log (
    id SERIAL PRIMARY KEY,
    synced_at TIMESTAMP DEFAULT NOW(),
    rows_processed INTEGER NOT NULL,
    quests_created INTEGER NOT NULL DEFAULT 0,
    quests_updated INTEGER NOT NULL DEFAULT 0,
    quests_unchanged INTEGER NOT NULL DEFAULT 0,
    items_added INTEGER NOT NULL DEFAULT 0,
    items_removed INTEGER NOT NULL DEFAULT 0,
    errors TEXT,
    success BOOLEAN DEFAULT true,
    duration_ms INTEGER,
    sheet_id VARCHAR(255) NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sync_log_synced_at ON equipment_request_sync_log(synced_at DESC);
COMMENT ON TABLE equipment_request_sync_log IS 'History of sync operations from Google Sheets';

CREATE TABLE IF NOT EXISTS equipment_request_category_mapping (
    id SERIAL PRIMARY KEY,
    form_item_name VARCHAR(255) UNIQUE NOT NULL,
    category_id INTEGER REFERENCES item_category(id) ON DELETE CASCADE,
    created_by INTEGER REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    last_used_at TIMESTAMP,
    use_count INTEGER DEFAULT 0
);
COMMENT ON TABLE equipment_request_category_mapping IS 'Manual category mapping overrides for fuzzy matching';

CREATE TABLE IF NOT EXISTS equipment_request_location_mapping (
    id SERIAL PRIMARY KEY,
    pavilion VARCHAR(255) NOT NULL,
    location_name VARCHAR(255) NOT NULL,
    location_id INT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    usage_count INT DEFAULT 0,
    FOREIGN KEY (location_id) REFERENCES locations(id) ON DELETE CASCADE,
    UNIQUE(pavilion, location_name)
);

INSERT INTO app_settings (key, value, description) VALUES
    ('equipment_request.sheet_id', '16BytrbWmyWeBGnlSIDZn1Lnb5rdspoQu_rpc5m5Vtbc', 'Google Sheets spreadsheet ID for equipment requests'),
    ('equipment_request.sheet_name', 'Zamówienia', 'Sheet name for order data (Zamówienia tab)'),
    ('equipment_request.cennik_sheet_name', 'Cennik', 'Sheet name for price list (Cennik tab)')
    ON CONFLICT (key) DO NOTHING;
