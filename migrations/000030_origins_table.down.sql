-- Restore VARCHAR origin columns
ALTER TABLE items ADD COLUMN origin VARCHAR(128);
UPDATE items SET origin = CASE
  WHEN origin_suffix IS NOT NULL THEN o.slug || '-' || origin_suffix
  ELSE o.slug END
  FROM origins o WHERE items.origin_id = o.id;
ALTER TABLE items DROP COLUMN origin_id, DROP COLUMN origin_suffix;

ALTER TABLE non_serialized_items DROP CONSTRAINT IF EXISTS non_serialized_items_unique_constraint;
ALTER TABLE non_serialized_items ADD COLUMN origin VARCHAR(128);
UPDATE non_serialized_items SET origin = CASE
  WHEN origin_suffix IS NOT NULL THEN o.slug || '-' || origin_suffix
  ELSE o.slug END
  FROM origins o WHERE non_serialized_items.origin_id = o.id;
ALTER TABLE non_serialized_items DROP COLUMN origin_id, DROP COLUMN origin_suffix;
ALTER TABLE non_serialized_items ADD CONSTRAINT non_serialized_items_unique_constraint
  UNIQUE (item_category_id, location_id, origin);

ALTER TABLE non_serialized_transfers ADD COLUMN origin VARCHAR(128);
UPDATE non_serialized_transfers SET origin = CASE
  WHEN origin_suffix IS NOT NULL THEN o.slug || '-' || origin_suffix
  ELSE o.slug END
  FROM origins o WHERE non_serialized_transfers.origin_id = o.id;
ALTER TABLE non_serialized_transfers DROP COLUMN origin_id, DROP COLUMN origin_suffix;

DROP TABLE origins;
