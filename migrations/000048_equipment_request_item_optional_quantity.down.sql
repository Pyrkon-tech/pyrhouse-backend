-- Restore NOT NULL quantity. Backfill any unspecified quantities to 1 first so the
-- constraint can be re-applied.
UPDATE equipment_request_items SET quantity = 1 WHERE quantity IS NULL;

ALTER TABLE equipment_request_items DROP CONSTRAINT equipment_request_items_quantity_check;
ALTER TABLE equipment_request_items
    ADD CONSTRAINT equipment_request_items_quantity_check CHECK (quantity > 0);

ALTER TABLE equipment_request_items ALTER COLUMN quantity SET NOT NULL;
