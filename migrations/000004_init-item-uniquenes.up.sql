BEGIN; 
    ALTER TABLE items
    ADD UNIQUE (item_serial);
COMMIT;