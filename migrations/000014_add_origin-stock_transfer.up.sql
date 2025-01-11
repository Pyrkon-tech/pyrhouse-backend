BEGIN;
ALTER TABLE non_serialized_transfers ADD COLUMN origin VARCHAR(128);
COMMIT;
