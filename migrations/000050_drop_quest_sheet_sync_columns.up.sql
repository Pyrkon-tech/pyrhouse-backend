-- Leftovers of the Google Sheets sync (removed in 000049): sync timestamp, source spreadsheet rows
-- and the fuzzy item-name → category match metadata. An item either has a category_id or it does not.

ALTER TABLE equipment_request_quests DROP COLUMN IF EXISTS last_synced_at;

ALTER TABLE equipment_request_items
    DROP COLUMN IF EXISTS source_row_number,
    DROP COLUMN IF EXISTS category_match_confidence,
    DROP COLUMN IF EXISTS category_match_type;
