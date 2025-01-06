BEGIN;
ALTER TABLE non_serialized_items DROP COLUMN status;
ALTER TABLE locations DROP details status;
COMMIT;