BEGIN;

DROP INDEX IF EXISTS idx_non_serialized_transfers_transfer_id;
DROP INDEX IF EXISTS idx_transfers_id;
DROP INDEX IF EXISTS idx_item_category_id;
DROP INDEX IF EXISTS idx_locations_id;
DROP INDEX IF EXISTS idx_items_id;

COMMIT; 