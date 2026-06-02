-- Replace the single transfer_id column on equipment_request_quests with a
-- dedicated junction table so one quest can be fulfilled by multiple transfers.

CREATE TABLE quest_transfers (
    id          SERIAL      PRIMARY KEY,
    quest_id    VARCHAR(100) NOT NULL REFERENCES equipment_request_quests(quest_id) ON DELETE CASCADE,
    transfer_id INTEGER     NOT NULL REFERENCES transfers(id) ON DELETE CASCADE,
    created_at  TIMESTAMP   NOT NULL DEFAULT NOW(),
    UNIQUE (transfer_id)   -- each transfer belongs to at most one quest
);

CREATE INDEX idx_quest_transfers_quest_id    ON quest_transfers(quest_id);
CREATE INDEX idx_quest_transfers_transfer_id ON quest_transfers(transfer_id);

-- Migrate existing links
INSERT INTO quest_transfers (quest_id, transfer_id, created_at)
SELECT quest_id, transfer_id, NOW()
FROM equipment_request_quests
WHERE transfer_id IS NOT NULL;

-- Drop the old column
ALTER TABLE equipment_request_quests DROP COLUMN transfer_id;
