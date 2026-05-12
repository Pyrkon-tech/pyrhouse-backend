DROP INDEX CONCURRENTLY IF EXISTS idx_equipment_request_quests_transfer_id;

ALTER TABLE service_desk_request_comments DROP CONSTRAINT IF EXISTS fk_service_desk_comments_user;
ALTER TABLE service_desk_requests DROP CONSTRAINT IF EXISTS fk_service_desk_requests_assigned_to;
ALTER TABLE service_desk_requests DROP CONSTRAINT IF EXISTS fk_service_desk_requests_created_by;
ALTER TABLE non_serialized_transfers DROP CONSTRAINT IF EXISTS fk_non_serialized_transfers_stock;
ALTER TABLE transfers DROP CONSTRAINT IF EXISTS fk_transfers_to_location;
ALTER TABLE transfers DROP CONSTRAINT IF EXISTS fk_transfers_from_location;
ALTER TABLE serialized_transfers DROP CONSTRAINT IF EXISTS fk_serialized_transfers_item;
