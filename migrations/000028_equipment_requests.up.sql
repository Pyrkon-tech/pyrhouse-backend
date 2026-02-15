-- Equipment Request Quest Management Tables

-- Main quests table
CREATE TABLE equipment_request_quests (
    id SERIAL PRIMARY KEY,
    quest_key VARCHAR(255) UNIQUE NOT NULL,  -- MD5 hash for deduplication
    quest_id VARCHAR(100) UNIQUE NOT NULL,   -- Human-readable ID (quest-abc123)

    -- Destination
    destination_pavilion VARCHAR(100) NOT NULL,
    destination_location VARCHAR(100) NOT NULL,

    -- Recipient & delivery
    recipient VARCHAR(255) NOT NULL,
    delivery_date DATE NOT NULL,
    pickup_time VARCHAR(50),  -- Flexible format: "17-18", "jutro 15", etc.

    -- Metadata
    budget_owner VARCHAR(255),
    status VARCHAR(50) DEFAULT 'pending' NOT NULL,
    transfer_id INTEGER REFERENCES transfers(id),  -- Link to created transfer (nullable)

    -- Tracking
    last_synced_at TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP,

    -- Constraints
    CONSTRAINT valid_status CHECK (status IN ('pending', 'in_progress', 'completed', 'cancelled'))
);

-- Quest items (many-to-one with quests)
CREATE TABLE equipment_request_items (
    id SERIAL PRIMARY KEY,
    quest_id INTEGER REFERENCES equipment_request_quests(id) ON DELETE CASCADE,

    -- Item details
    item_name VARCHAR(255) NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),

    -- Category matching
    category_id INTEGER REFERENCES item_category(id),  -- Nullable if no match
    category_match_type VARCHAR(50) NOT NULL,
    category_match_confidence DECIMAL(3,2),  -- 0.00 to 1.00

    -- Metadata
    budget_owner VARCHAR(255),  -- Per-item budget owner (can differ from quest)
    notes TEXT,
    source_row_number INTEGER,  -- Track back to spreadsheet row

    created_at TIMESTAMP DEFAULT NOW(),

    -- Constraints
    CONSTRAINT valid_match_type CHECK (category_match_type IN ('exact', 'fuzzy', 'manual', 'none'))
);

-- Sync history log
CREATE TABLE equipment_request_sync_log (
    id SERIAL PRIMARY KEY,
    synced_at TIMESTAMP DEFAULT NOW(),

    -- Stats
    rows_processed INTEGER NOT NULL,
    quests_created INTEGER NOT NULL DEFAULT 0,
    quests_updated INTEGER NOT NULL DEFAULT 0,
    quests_unchanged INTEGER NOT NULL DEFAULT 0,
    items_added INTEGER NOT NULL DEFAULT 0,
    items_removed INTEGER NOT NULL DEFAULT 0,

    -- Error tracking
    errors TEXT,  -- JSON array of errors if any
    success BOOLEAN DEFAULT true,

    -- Performance
    duration_ms INTEGER,  -- Sync duration in milliseconds
    sheet_id VARCHAR(255) NOT NULL
);

-- Manual category mapping overrides
CREATE TABLE equipment_request_category_mapping (
    id SERIAL PRIMARY KEY,
    form_item_name VARCHAR(255) UNIQUE NOT NULL,  -- Exact match from form
    category_id INTEGER REFERENCES item_category(id) ON DELETE CASCADE,

    -- Metadata
    created_by INTEGER REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    last_used_at TIMESTAMP,
    use_count INTEGER DEFAULT 0
);

-- Indexes for performance
CREATE INDEX idx_quests_status ON equipment_request_quests(status);
CREATE INDEX idx_quests_delivery_date ON equipment_request_quests(delivery_date);
CREATE INDEX idx_quests_last_synced ON equipment_request_quests(last_synced_at);
CREATE INDEX idx_quests_quest_key ON equipment_request_quests(quest_key);
CREATE INDEX idx_items_quest_id ON equipment_request_items(quest_id);
CREATE INDEX idx_items_category_id ON equipment_request_items(category_id);
CREATE INDEX idx_sync_log_synced_at ON equipment_request_sync_log(synced_at DESC);

-- Comments for documentation
COMMENT ON TABLE equipment_request_quests IS 'Aggregated equipment requests from Google Sheets form';
COMMENT ON TABLE equipment_request_items IS 'Individual items within an equipment request quest';
COMMENT ON TABLE equipment_request_sync_log IS 'History of sync operations from Google Sheets';
COMMENT ON TABLE equipment_request_category_mapping IS 'Manual category mapping overrides for fuzzy matching';

COMMENT ON COLUMN equipment_request_quests.quest_key IS 'MD5 hash of pavilion|location|recipient|delivery_date|pickup_time for deduplication';
COMMENT ON COLUMN equipment_request_quests.quest_id IS 'Human-readable quest ID (quest-{hash})';
COMMENT ON COLUMN equipment_request_items.category_match_type IS 'How category was matched: exact, fuzzy, manual, or none';
COMMENT ON COLUMN equipment_request_items.category_match_confidence IS 'Matching confidence for fuzzy matches (0.00-1.00)';
