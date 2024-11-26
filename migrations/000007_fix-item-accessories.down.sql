BEGIN;
ALTER TABLE item_category ADD COLUMN accessories JSONB;
ALTER TABLE items DROP COLUMN accessories;

COMMIT;