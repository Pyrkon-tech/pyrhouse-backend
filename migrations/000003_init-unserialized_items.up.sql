BEGIN;
    ALTER TABLE items
    DROP COLUMN IF EXISTS item_type;

    CREATE TABLE non_serialized_items (
        id SERIAL PRIMARY KEY,
        item_category_id INT NOT NULL, -- Type of item (e.g., extension cord)
        location_id INT NOT NULL, -- Location
        quantity INT NOT NULL, -- Available stock
        FOREIGN KEY (location_id) REFERENCES locations (id),
        UNIQUE (item_category_id, location_id) -- Ensures unique stock record per item type and location
    );

    CREATE TABLE transfers (
        id SERIAL PRIMARY KEY,
        from_location_id INT NOT NULL,
        to_location_id INT NOT NULL,
        status VARCHAR(50) DEFAULT 'pending',
        transfer_date TIMESTAMP DEFAULT NOW(),
        receiver VARCHAR(255)
    );

    CREATE TABLE serialized_transfers (
        id SERIAL PRIMARY KEY,
        transfer_id INT REFERENCES transfers(id) ON DELETE CASCADE,
        item_id INT NOT NULL
    );
    CREATE TABLE non_serialized_transfers (
        id SERIAL PRIMARY KEY,
        transfer_id INT REFERENCES transfers(id) ON DELETE CASCADE,
        item_category_id INT NOT NULL,
        quantity INT NOT NULL
    );

COMMIT;