BEGIN;

-- Sprawdź czy indeksy już istnieją przed utworzeniem
DO $$
BEGIN
    -- Indeks dla non_serialized_transfers
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes 
        WHERE indexname = 'idx_non_serialized_transfers_transfer_id'
    ) THEN
        CREATE INDEX idx_non_serialized_transfers_transfer_id 
        ON non_serialized_transfers(transfer_id);
    END IF;

    -- Indeks dla transfers
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes 
        WHERE indexname = 'idx_transfers_id'
    ) THEN
        CREATE INDEX idx_transfers_id 
        ON transfers(id);
    END IF;

    -- Indeks dla item_category
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes 
        WHERE indexname = 'idx_item_category_id'
    ) THEN
        CREATE INDEX idx_item_category_id 
        ON item_category(id);
    END IF;

    -- Indeks dla locations
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes 
        WHERE indexname = 'idx_locations_id'
    ) THEN
        CREATE INDEX idx_locations_id 
        ON locations(id);
    END IF;

    -- Indeks dla items
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes 
        WHERE indexname = 'idx_items_id'
    ) THEN
        CREATE INDEX idx_items_id 
        ON items(id);
    END IF;
END $$;

COMMIT; 