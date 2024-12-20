BEGIN;
ALTER TABLE non_serialized_transfers ALTER COLUMN item_category_id DROP NOT NULL;

ALTER TABLE non_serialized_transfers ADD COLUMN IF NOT EXISTS stock_id INT;
COMMIT;