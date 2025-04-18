ALTER TABLE transfers
ADD COLUMN delivery_latitude DECIMAL(10, 8),
ADD COLUMN delivery_longitude DECIMAL(11, 8),
ADD COLUMN delivery_timestamp TIMESTAMP; 