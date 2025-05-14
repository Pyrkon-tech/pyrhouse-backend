BEGIN;

    ALTER TABLE users ADD COLUMN active BOOLEAN NOT NULL DEFAULT true;
    UPDATE users SET active = true WHERE active IS NULL; 

COMMIT;