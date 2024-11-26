BEGIN;
ALTER TABLE item_category DROP COLUMN accessories;
ALTER TABLE items ADD COLUMN accessories JSONB;
COMMIT;