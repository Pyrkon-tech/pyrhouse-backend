-- Usuń istniejące ograniczenie jeśli istnieje
ALTER TABLE non_serialized_items 
DROP CONSTRAINT IF EXISTS non_serialized_items_item_category_id_fkey;

-- Dodaj nowe ograniczenie z ON DELETE RESTRICT
ALTER TABLE non_serialized_items 
ADD CONSTRAINT non_serialized_items_item_category_id_fkey 
FOREIGN KEY (item_category_id) 
REFERENCES item_category(id) 
ON DELETE RESTRICT; 