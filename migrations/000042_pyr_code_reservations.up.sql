CREATE TABLE pyr_code_reservations (
    id          SERIAL PRIMARY KEY,
    pyr_code    VARCHAR NOT NULL UNIQUE,
    category_id INT NOT NULL REFERENCES item_category(id),
    reserved_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    claimed_at  TIMESTAMPTZ,
    item_id     INT REFERENCES items(id)
);

CREATE INDEX pyr_code_reservations_unclaimed_idx
    ON pyr_code_reservations(pyr_code) WHERE claimed_at IS NULL;
