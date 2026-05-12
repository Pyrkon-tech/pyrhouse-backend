-- equipment_request_quests.location_id is nullable — set to NULL on location delete
ALTER TABLE equipment_request_quests
    DROP CONSTRAINT IF EXISTS equipment_request_quests_location_id_fkey;

ALTER TABLE equipment_request_quests
    ADD CONSTRAINT equipment_request_quests_location_id_fkey
    FOREIGN KEY (location_id) REFERENCES locations(id) ON DELETE SET NULL;
