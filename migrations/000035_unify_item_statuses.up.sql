BEGIN;

-- Unify all item statuses to available/in_transit/unavailable
-- located, in_stock, completed, cancelled all mean "asset is at its location, ready"
UPDATE items SET status = 'available' WHERE status IN ('located', 'in_stock', 'completed', 'cancelled');

COMMIT;
