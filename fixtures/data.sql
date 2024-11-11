BEGIN;
    -- Default data
    INSERT INTO locations (name) VALUES ('Magazyn Techniczny');
    INSERT INTO items (item_type, item_serial, status, location_id) VALUES ('LAPTOP', 'XXKSA03', 'IN_STOCK', 1)
COMMIT;