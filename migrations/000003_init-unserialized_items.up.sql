BEGIN;
    CREATE TABLE non_serialized_items (
        id SERIAL PRIMARY KEY,
        item_category_id INT NOT NULL, -- Type of item (e.g., extension cord)
        location_id INT NOT NULL, -- Location
        quantity INT NOT NULL, -- Available stock
        UNIQUE (item_category_id, location_id) -- Ensures unique stock record per item type and location
    );
COMMIT;