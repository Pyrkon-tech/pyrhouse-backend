BEGIN;

ALTER TABLE service_desk_requests
    ADD COLUMN location_id INT REFERENCES locations(id) ON DELETE SET NULL;

ALTER TABLE service_desk_requests
    ADD CONSTRAINT chk_service_desk_location_exclusive
        CHECK (location_id IS NULL OR location IS NULL);

COMMIT;
