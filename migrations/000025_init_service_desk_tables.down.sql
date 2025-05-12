BEGIN;

DROP TABLE service_desk_requests;
DROP TABLE service_desk_request_comments;
DROP INDEX idx_service_desk_request_comments_request_id;
DROP INDEX idx_service_desk_requests_created_by_id;
DROP INDEX idx_service_desk_requests_assigned_to_id;

COMMIT;
