BEGIN;
ALTER TABLE serialized_transfers ADD COLUMN origin VARCHAR(128);
ALTER TABLE item_category ADD COLUMN category_type VARCHAR(32) NOT NULL DEFAULT 'asset';
ALTER TABLE items DROP COLUMN accessories;
COMMIT;