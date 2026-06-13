-- Fix duplicate stock rows caused by the UNIQUE (item_category_id, location_id,
-- origin_id, origin_suffix) constraint treating NULLs as distinct. When origin_suffix
-- (or origin_id) is NULL, the ON CONFLICT clauses in the stocks repository never fire,
-- so adding/transferring the same item creates a new row instead of summing quantities.
--
-- This migration merges existing duplicates, then recreates the constraint with
-- NULLS NOT DISTINCT (PostgreSQL 15+) so NULL origins are treated as equal.

-- 1. Re-point any transfer records referencing a surplus (to-be-deleted) duplicate
--    stock row to the surviving (lowest-id) row, so the FK on
--    non_serialized_transfers.stock_id does not block the delete below.
UPDATE non_serialized_transfers nst
SET stock_id = keep.keep_id
FROM (
    SELECT id,
           MIN(id) OVER (
               PARTITION BY item_category_id, location_id, origin_id, origin_suffix
           ) AS keep_id
    FROM non_serialized_items
) keep
WHERE nst.stock_id = keep.id
  AND keep.id <> keep.keep_id;

-- 2. Sum every duplicate group's quantity into its surviving (lowest-id) row.
UPDATE non_serialized_items target
SET quantity = agg.total_qty
FROM (
    SELECT MIN(id) AS keep_id, SUM(quantity) AS total_qty, COUNT(*) AS cnt
    FROM non_serialized_items
    GROUP BY item_category_id, location_id, origin_id, origin_suffix
) agg
WHERE target.id = agg.keep_id
  AND agg.cnt > 1;

-- 3. Delete the now-redundant surplus duplicate rows.
DELETE FROM non_serialized_items nsi
USING (
    SELECT id,
           MIN(id) OVER (
               PARTITION BY item_category_id, location_id, origin_id, origin_suffix
           ) AS keep_id
    FROM non_serialized_items
) keep
WHERE nsi.id = keep.id
  AND keep.id <> keep.keep_id;

-- 4. Recreate the unique constraint so NULL origin_id/origin_suffix are treated as equal.
ALTER TABLE non_serialized_items DROP CONSTRAINT non_serialized_items_unique_constraint;
ALTER TABLE non_serialized_items
    ADD CONSTRAINT non_serialized_items_unique_constraint
    UNIQUE NULLS NOT DISTINCT (item_category_id, location_id, origin_id, origin_suffix);
