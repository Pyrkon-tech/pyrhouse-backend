BEGIN;
CREATE TABLE audit_logs (
    id SERIAL PRIMARY KEY,
    resource_id INT NOT NULL,               -- References the resource (e.g., asset_id or stock_id)
    resource_type VARCHAR(50) NOT NULL,     -- Indicates the type of resource (e.g., 'asset', 'stock')
    action VARCHAR(50) NOT NULL,            -- Action performed (e.g., 'transfer', 'create', 'delete')
    data JSONB,                             -- Optional: Stores additional information about the action
    created_at TIMESTAMP DEFAULT NOW(),     -- Timestamp of the action
    user_id INT,                            -- Optional: Tracks which user performed the action
    FOREIGN KEY (user_id) REFERENCES users (id)
);
COMMIT;