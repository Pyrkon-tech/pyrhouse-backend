ALTER TABLE non_serialized_transfers
ADD COLUMN status VARCHAR(20) DEFAULT 'in_transit';

-- Aktualizujemy istniejące wpisy
UPDATE non_serialized_transfers nst
SET status = t.status
FROM transfers t
WHERE nst.transfer_id = t.id; 