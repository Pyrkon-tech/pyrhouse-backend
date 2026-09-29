-- Decision 2026-09-29: the shop does not collect phone numbers (contacts are available elsewhere),
-- which also keeps personal data out of shop orders.
ALTER TABLE shop_orders DROP COLUMN IF EXISTS contact_phone;
