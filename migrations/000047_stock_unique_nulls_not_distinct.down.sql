-- Revert to the default NULLS DISTINCT unique constraint. The duplicate-row merge
-- performed by the up migration is not reversible.
ALTER TABLE non_serialized_items DROP CONSTRAINT non_serialized_items_unique_constraint;
ALTER TABLE non_serialized_items
    ADD CONSTRAINT non_serialized_items_unique_constraint
    UNIQUE (item_category_id, location_id, origin_id, origin_suffix);
