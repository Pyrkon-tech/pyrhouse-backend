-- Dodanie kolumny points do tabeli users
ALTER TABLE users ADD COLUMN points INTEGER NOT NULL DEFAULT 0; 