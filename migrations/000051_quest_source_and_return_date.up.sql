-- Where a quest came from: 'sheet' (historic Google Sheets import) or 'shop' (confirmed organizer shop order).
-- No default after the backfill, so every insert has to say where the quest came from.
ALTER TABLE equipment_request_quests ADD COLUMN source VARCHAR(16) NOT NULL DEFAULT 'sheet';
ALTER TABLE equipment_request_quests ALTER COLUMN source DROP DEFAULT;
ALTER TABLE equipment_request_quests
    ADD CONSTRAINT equipment_request_quests_source_check CHECK (source IN ('sheet', 'shop'));

-- When the equipment comes back (shop orders: the return window's day or the exact date given).
ALTER TABLE equipment_request_quests ADD COLUMN return_date DATE NULL;
