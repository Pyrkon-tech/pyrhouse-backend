BEGIN;
-- Usuń istniejące ograniczenie jeśli istnieje
ALTER TABLE items DROP CONSTRAINT IF EXISTS items_item_serial_key;

-- Dodaj nowe ograniczenie, które pozwala na wartości NULL
CREATE UNIQUE INDEX items_item_serial_unique ON items (item_serial) WHERE item_serial IS NOT NULL;

-- Zmień kolumnę na nullable
ALTER TABLE items ALTER COLUMN item_serial DROP NOT NULL;
COMMIT; 