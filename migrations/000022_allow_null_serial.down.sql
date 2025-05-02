BEGIN;
-- Usuń indeks częściowy
DROP INDEX IF EXISTS items_item_serial_unique;

-- Przywróć oryginalne ograniczenie
ALTER TABLE items ADD CONSTRAINT items_item_serial_key UNIQUE (item_serial);

-- Przywróć NOT NULL
ALTER TABLE items ALTER COLUMN item_serial SET NOT NULL;
COMMIT; 