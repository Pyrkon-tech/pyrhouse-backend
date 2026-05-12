-- Index for equipment_request_quests.transfer_id (joined on every GET /schedule/on-duty and GET /dispatch/volunteers)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_equipment_request_quests_transfer_id
    ON equipment_request_quests (transfer_id);

-- FK: serialized_transfers.item_id → items(id)
ALTER TABLE serialized_transfers
    ADD CONSTRAINT fk_serialized_transfers_item
    FOREIGN KEY (item_id) REFERENCES items(id) NOT VALID;

ALTER TABLE serialized_transfers VALIDATE CONSTRAINT fk_serialized_transfers_item;

-- FK: transfers.from_location_id / to_location_id → locations(id)
ALTER TABLE transfers
    ADD CONSTRAINT fk_transfers_from_location
    FOREIGN KEY (from_location_id) REFERENCES locations(id) NOT VALID;

ALTER TABLE transfers VALIDATE CONSTRAINT fk_transfers_from_location;

ALTER TABLE transfers
    ADD CONSTRAINT fk_transfers_to_location
    FOREIGN KEY (to_location_id) REFERENCES locations(id) NOT VALID;

ALTER TABLE transfers VALIDATE CONSTRAINT fk_transfers_to_location;

-- FK: non_serialized_transfers.stock_id → non_serialized_items(id)
ALTER TABLE non_serialized_transfers
    ADD CONSTRAINT fk_non_serialized_transfers_stock
    FOREIGN KEY (stock_id) REFERENCES non_serialized_items(id) NOT VALID;

ALTER TABLE non_serialized_transfers VALIDATE CONSTRAINT fk_non_serialized_transfers_stock;

-- FK: service_desk user references → users(id)
ALTER TABLE service_desk_requests
    ADD CONSTRAINT fk_service_desk_requests_created_by
    FOREIGN KEY (created_by_id) REFERENCES users(id) NOT VALID;

ALTER TABLE service_desk_requests VALIDATE CONSTRAINT fk_service_desk_requests_created_by;

ALTER TABLE service_desk_requests
    ADD CONSTRAINT fk_service_desk_requests_assigned_to
    FOREIGN KEY (assigned_to_id) REFERENCES users(id) NOT VALID;

ALTER TABLE service_desk_requests VALIDATE CONSTRAINT fk_service_desk_requests_assigned_to;

ALTER TABLE service_desk_request_comments
    ADD CONSTRAINT fk_service_desk_comments_user
    FOREIGN KEY (user_id) REFERENCES users(id) NOT VALID;

ALTER TABLE service_desk_request_comments VALIDATE CONSTRAINT fk_service_desk_comments_user;
