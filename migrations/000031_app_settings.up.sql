CREATE TABLE app_settings (
    key VARCHAR(128) PRIMARY KEY,
    value TEXT NOT NULL,
    description VARCHAR(512),
    updated_at TIMESTAMP DEFAULT NOW()
);

INSERT INTO app_settings (key, value, description) VALUES
  ('equipment_request.sheet_id', '16BytrbWmyWeBGnlSIDZn1Lnb5rdspoQu_rpc5m5Vtbc', 'Google Sheets document ID for equipment requests'),
  ('equipment_request.sheet_name', 'Zamówienia', 'Sheet tab name within the document');
