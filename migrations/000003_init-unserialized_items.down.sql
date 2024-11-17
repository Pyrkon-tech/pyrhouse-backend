BEGIN;
    DROP TABLE non_serialized_items;
    DROP TABLE transfers;
    DROP TABLE serialized_transfers;
    DROP TABLE non_serialized_transfers;
COMMIT;