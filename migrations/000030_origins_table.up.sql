-- 1. Create origins table + seed
CREATE TABLE origins (
    id SERIAL PRIMARY KEY,
    slug VARCHAR(128) NOT NULL UNIQUE,
    label VARCHAR(255) NOT NULL,
    allow_suffix BOOLEAN DEFAULT false,
    active BOOLEAN DEFAULT true,
    sort_order INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);

INSERT INTO origins (slug, label, allow_suffix, sort_order) VALUES
  ('druga-era',  'Druga Era',  false, 1),
  ('probis',     'Probis',     false, 2),
  ('netland',    'Netland',    false, 3),
  ('targowe',    'Targowe',    false, 4),
  ('dj-sound',   'DJ Sound',   false, 5),
  ('oki-event',  'Oki Event',  false, 6),
  ('personal',   'Personal',   true,  7),
  ('other',      'Other',      true,  8);

-- 2. items: origin VARCHAR → origin_id FK + origin_suffix
ALTER TABLE items ADD COLUMN origin_id INT REFERENCES origins(id);
ALTER TABLE items ADD COLUMN origin_suffix VARCHAR(128);
-- Migrate data: exact slug match
UPDATE items SET origin_id = o.id
  FROM origins o WHERE items.origin = o.slug;
-- Migrate data: suffix match (e.g. "personal-jan" → origin_id=personal, suffix="jan")
UPDATE items SET
  origin_id = o.id,
  origin_suffix = SUBSTRING(items.origin FROM LENGTH(o.slug) + 2)
  FROM origins o
  WHERE items.origin_id IS NULL
    AND o.allow_suffix = true
    AND items.origin LIKE o.slug || '-%';
ALTER TABLE items DROP COLUMN origin;

-- 3. non_serialized_items: origin VARCHAR → origin_id FK + origin_suffix
ALTER TABLE non_serialized_items DROP CONSTRAINT IF EXISTS non_serialized_items_unique_constraint;
ALTER TABLE non_serialized_items ADD COLUMN origin_id INT REFERENCES origins(id);
ALTER TABLE non_serialized_items ADD COLUMN origin_suffix VARCHAR(128);
UPDATE non_serialized_items SET origin_id = o.id
  FROM origins o WHERE non_serialized_items.origin = o.slug;
UPDATE non_serialized_items SET
  origin_id = o.id,
  origin_suffix = SUBSTRING(non_serialized_items.origin FROM LENGTH(o.slug) + 2)
  FROM origins o
  WHERE non_serialized_items.origin_id IS NULL
    AND o.allow_suffix = true
    AND non_serialized_items.origin LIKE o.slug || '-%';
ALTER TABLE non_serialized_items DROP COLUMN origin;
ALTER TABLE non_serialized_items ADD CONSTRAINT non_serialized_items_unique_constraint
  UNIQUE (item_category_id, location_id, origin_id, origin_suffix);

-- 4. non_serialized_transfers: origin VARCHAR → origin_id FK + origin_suffix
ALTER TABLE non_serialized_transfers ADD COLUMN origin_id INT REFERENCES origins(id);
ALTER TABLE non_serialized_transfers ADD COLUMN origin_suffix VARCHAR(128);
UPDATE non_serialized_transfers SET origin_id = o.id
  FROM origins o WHERE non_serialized_transfers.origin = o.slug;
UPDATE non_serialized_transfers SET
  origin_id = o.id,
  origin_suffix = SUBSTRING(non_serialized_transfers.origin FROM LENGTH(o.slug) + 2)
  FROM origins o
  WHERE non_serialized_transfers.origin_id IS NULL
    AND o.allow_suffix = true
    AND non_serialized_transfers.origin LIKE o.slug || '-%';
ALTER TABLE non_serialized_transfers DROP COLUMN origin;
