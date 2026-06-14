-- Make equipment_request_items.quantity optional. The source Google Sheet sometimes has
-- a requested item with the quantity cell left blank; previously such rows were dropped on
-- sync. A NULL quantity now means "not specified yet" — the item is imported and surfaced
-- for the dispatcher to fill in before a transfer can be created.
ALTER TABLE equipment_request_items ALTER COLUMN quantity DROP NOT NULL;

ALTER TABLE equipment_request_items DROP CONSTRAINT equipment_request_items_quantity_check;
ALTER TABLE equipment_request_items
    ADD CONSTRAINT equipment_request_items_quantity_check CHECK (quantity IS NULL OR quantity > 0);
