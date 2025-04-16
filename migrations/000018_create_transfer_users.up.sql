BEGIN;

CREATE TABLE transfer_users (
    id SERIAL PRIMARY KEY,
    transfer_id INTEGER NOT NULL REFERENCES transfers(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    UNIQUE (transfer_id, user_id)
);

-- Dodajemy indeks dla szybszego wyszukiwania
CREATE INDEX idx_transfer_users_transfer_id ON transfer_users(transfer_id);
CREATE INDEX idx_transfer_users_user_id ON transfer_users(user_id);

COMMIT; 