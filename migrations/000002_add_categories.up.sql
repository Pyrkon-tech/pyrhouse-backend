BEGIN;
    CREATE TABLE item_category (
        id SERIAL PRIMARY KEY,
        item_category VARCHAR(128) UNIQUE,
        label VARCHAR(128)
    );
    ALTER TABLE items
    ADD item_category_id INT NOT NULL;
    ALTER TABLE items ADD FOREIGN KEY (item_category_id) REFERENCES item_category (id);


    INSERT INTO item_category (item_category, label) VALUES ('laptop', 'Laptop');
    INSERT INTO item_category (item_category, label) VALUES ('printer', 'Drukarka');
    INSERT INTO item_category (item_category, label) VALUES ('projector', 'Projektor');
    INSERT INTO item_category (item_category, label) VALUES ('tablet', 'Tablet');
COMMIT;