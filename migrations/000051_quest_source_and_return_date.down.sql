ALTER TABLE equipment_request_quests
    DROP COLUMN IF EXISTS return_date,
    DROP COLUMN IF EXISTS source;
