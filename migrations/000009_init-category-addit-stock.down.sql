BEGIN;
ALTER TABLE serialized_transfers DROP COLUMN origin;
ALTER TABLE item_category DROP COLUMN category_type;
ALTER TABLE items ADD COLUMN accessories JSONB;
COMMIT;