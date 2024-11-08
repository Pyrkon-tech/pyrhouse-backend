CREATE TABLE albums (
	id SERIAL PRIMARY KEY,
    title VARCHAR(255),
    artist VARCHAR(255),
    price DECIMAL(10, 2)
);

CREATE TABLE locations (
    id SERIAL PRIMARY KEY,
    name VARCHAR(128)
);

CREATE TABLE items (
	id SERIAL PRIMARY KEY,
    item_type VARCHAR(32),
    item_serial VARCHAR(128),
    status VARCHAR(32),
    location_id INT NOT NULL,
    FOREIGN KEY (location_id) REFERENCES locations (id)
);

-- Default data
INSERT INTO locations (name) VALUES ('Magazyn Techniczny');
