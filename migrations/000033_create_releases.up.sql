BEGIN;

CREATE TABLE releases (
    id SERIAL PRIMARY KEY,
    reference VARCHAR(20) NOT NULL UNIQUE,
    origin_id INT NOT NULL REFERENCES origins(id),
    notes TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    created_by INT NOT NULL REFERENCES users(id),
    completed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE release_assets (
    id SERIAL PRIMARY KEY,
    release_id INT NOT NULL REFERENCES releases(id) ON DELETE CASCADE,
    item_id INT NOT NULL,
    pyr_code VARCHAR(50),
    item_serial VARCHAR(255),
    category_name VARCHAR(255),
    origin_label VARCHAR(255),
    location_name VARCHAR(255),
    UNIQUE(release_id, item_id)
);

CREATE TABLE release_stocks (
    id SERIAL PRIMARY KEY,
    release_id INT NOT NULL REFERENCES releases(id) ON DELETE CASCADE,
    stock_id INT NOT NULL,
    item_category_id INT NOT NULL,
    category_name VARCHAR(255),
    quantity INT NOT NULL,
    origin_label VARCHAR(255),
    location_name VARCHAR(255),
    UNIQUE(release_id, stock_id)
);

CREATE INDEX idx_releases_status ON releases(status);
CREATE INDEX idx_releases_origin_id ON releases(origin_id);
CREATE INDEX idx_release_assets_release_id ON release_assets(release_id);
CREATE INDEX idx_release_stocks_release_id ON release_stocks(release_id);

COMMIT;
