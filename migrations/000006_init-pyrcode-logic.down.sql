BEGIN;
ALTER TABLE items DROP COLUMN pyr_code;
ALTER TABLE item_category DROP COLUMN pyr_id;
ALTER TABLE item_category DROP COLUMN accessories;
COMMIT;