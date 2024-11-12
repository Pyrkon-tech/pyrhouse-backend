BEGIN;
    DROP TABLE IF EXISTS item_category

    ALTER TABLE items
    DROP COLUMN item_category_id;
    DROP CONSTRAINT item_category_id
COMMIT;