DROP INDEX IF EXISTS idx_quests_location_resolved;

ALTER TABLE equipment_request_quests
    DROP COLUMN IF EXISTS location_id,
    DROP COLUMN IF EXISTS location_resolved;

DROP TABLE IF EXISTS equipment_request_location_mapping;
