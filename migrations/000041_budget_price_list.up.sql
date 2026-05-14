-- Dynamic supplier price list: one row per (item_name, supplier) pair.
-- Adding a new supplier requires no schema change — just INSERT with the new supplier name.
CREATE TABLE equipment_request_price_list (
    id         SERIAL PRIMARY KEY,
    item_name  VARCHAR(255) NOT NULL,
    supplier   VARCHAR(255) NOT NULL,
    unit_price NUMERIC(10, 2) NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (item_name, supplier)
);

CREATE INDEX idx_price_list_item_name ON equipment_request_price_list (item_name);
CREATE INDEX idx_price_list_supplier  ON equipment_request_price_list (supplier);

COMMENT ON TABLE equipment_request_price_list IS 'Per-supplier unit prices for budget calculator (Zapotrzebowanie techniczne). Add any number of suppliers without schema changes.';

-- Default app settings for budget / price list integration
INSERT INTO app_settings (key, value, description) VALUES
    ('equipment_request.sheet_id', '16BytrbWmyWeBGnlSIDZn1Lnb5rdspoQu_rpc5m5Vtbc', 'Google Sheets spreadsheet ID for equipment requests')
    ON CONFLICT (key) DO NOTHING;

INSERT INTO app_settings (key, value, description) VALUES
    ('equipment_request.sheet_name', 'Zamówienia', 'Sheet name for order data (Zamówienia tab)')
    ON CONFLICT (key) DO NOTHING;

INSERT INTO app_settings (key, value, description) VALUES
    ('equipment_request.cennik_sheet_name', 'Cennik', 'Sheet name for price list (Cennik tab)')
    ON CONFLICT (key) DO NOTHING;
