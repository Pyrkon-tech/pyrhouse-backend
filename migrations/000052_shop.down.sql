DELETE FROM app_settings WHERE key LIKE 'shop.%';

ALTER TABLE equipment_request_quests DROP COLUMN IF EXISTS shop_order_id;

DROP TABLE IF EXISTS shop_order_events;
DROP TABLE IF EXISTS shop_order_items;
DROP TABLE IF EXISTS shop_orders;
DROP TABLE IF EXISTS shop_delivery_windows;
DROP TABLE IF EXISTS shop_products;
DROP TABLE IF EXISTS shop_invites;
DROP TABLE IF EXISTS shop_accounts;
