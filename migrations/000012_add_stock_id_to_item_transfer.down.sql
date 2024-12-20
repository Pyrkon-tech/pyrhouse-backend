BEGIN;
ALTER TABLE non_serialized_transfers DROP COLUMN stock_id;
COMMIT;