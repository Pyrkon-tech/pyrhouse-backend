-- The organizer shop replaces the Google Sheets order form (docs/shop/PLAN.md, D11).
-- Drop the sheet-sync machinery: sync history, fuzzy-match overrides and sheet config.
-- Quests imported from the sheet stay as history; equipment_request_price_list stays (budget page, D13).

DROP TABLE IF EXISTS equipment_request_sync_log;
DROP TABLE IF EXISTS equipment_request_category_mapping;
DROP TABLE IF EXISTS equipment_request_location_mapping;

DELETE FROM app_settings WHERE key IN (
    'equipment_request.sheet_id',
    'equipment_request.sheet_name',
    'equipment_request.cennik_sheet_name'
);
