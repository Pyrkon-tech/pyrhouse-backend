ALTER TABLE serialized_transfers DROP COLUMN origin;
ALTER TABLE non_serialized_items ADD COLUMN origin VARCHAR(128);

