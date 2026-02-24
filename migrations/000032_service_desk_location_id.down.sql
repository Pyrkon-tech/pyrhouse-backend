BEGIN;

ALTER TABLE service_desk_requests DROP CONSTRAINT IF EXISTS chk_service_desk_location_exclusive;
ALTER TABLE service_desk_requests DROP COLUMN IF EXISTS location_id;

COMMIT;
