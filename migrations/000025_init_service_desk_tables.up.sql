BEGIN;

    CREATE TABLE service_desk_requests (
        id SERIAL PRIMARY KEY,
        title VARCHAR(255) NOT NULL,
        description TEXT NOT NULL,
        type VARCHAR(255) NOT NULL,
        status VARCHAR(128) NOT NULL,
        created_by VARCHAR(128) NOT NULL,
        created_by_id INT,
        assigned_to_id INT,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        priority VARCHAR(128) NOT NULL,
        location VARCHAR(128)
    );

    CREATE TABLE service_desk_request_comments (
        id SERIAL PRIMARY KEY,
        request_id INT NOT NULL,
        comment TEXT NOT NULL,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        user_id INT NOT NULL
    );

    CREATE INDEX IF NOT EXISTS idx_service_desk_request_comments_request_id ON service_desk_request_comments (request_id);
    CREATE INDEX IF NOT EXISTS idx_service_desk_requests_created_by_id ON service_desk_requests (created_by_id);
    CREATE INDEX IF NOT EXISTS idx_service_desk_requests_assigned_to_id ON service_desk_requests (assigned_to_id);

COMMIT;