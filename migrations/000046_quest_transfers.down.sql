-- Restore single transfer_id column (takes the most recent transfer per quest)
ALTER TABLE equipment_request_quests
    ADD COLUMN transfer_id INTEGER REFERENCES transfers(id);

UPDATE equipment_request_quests q
SET transfer_id = (
    SELECT qt.transfer_id
    FROM quest_transfers qt
    WHERE qt.quest_id = q.quest_id
    ORDER BY qt.created_at DESC
    LIMIT 1
);

DROP TABLE IF EXISTS quest_transfers;
