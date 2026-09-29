-- Anonymises a restored copy of the database so it can become fixtures: every person-shaped value is replaced,
-- deterministically (the same original gets the same stand-in, so one recipient stays one person across transfers),
-- credentials and links to real documents are removed, and the noisiest logs are trimmed.
-- Run by postgres/build-fixtures.sh against a throwaway database. Every account's password becomes "pyrhouse".
CREATE EXTENSION IF NOT EXISTS pgcrypto;
BEGIN;

CREATE TEMP TABLE fake_names AS
SELECT ARRAY['Anna','Jan','Katarzyna','Piotr','Magdalena','Tomasz','Agnieszka','Paweł','Monika','Michał','Ewa','Krzysztof',
             'Joanna','Marcin','Aleksandra','Łukasz','Natalia','Kamil','Zofia','Bartosz','Julia','Wojciech','Maria','Adam'] AS first,
       ARRAY['Nowak','Wiśniewski','Wójcik','Kamiński','Lewandowski','Zieliński','Szymański','Woźniak','Dąbrowski','Kozłowski',
             'Jankowski','Mazur','Kwiatkowski','Krawczyk','Piotrowski','Grabowski','Nowakowski','Pawłowski','Michalski','Król'] AS last,
       ARRAY['Poznań','Warszawa','Kraków','Wrocław','Gdańsk','Łódź','Szczecin','Lublin','Bydgoszcz','Katowice'] AS city;

-- A stand-in name for any original text: the same text always maps to the same name.
CREATE FUNCTION pg_temp.fake_name(original text) RETURNS text LANGUAGE sql AS $$
  SELECT CASE WHEN original IS NULL THEN NULL ELSE
    f.first[1 + (('x' || substr(md5(original), 1, 6))::bit(24)::int % array_length(f.first, 1))] || ' ' ||
    f.last[1 + (('x' || substr(md5(original), 7, 6))::bit(24)::int % array_length(f.last, 1))] END
  FROM fake_names f $$;

-- People. Usernames go through a temporary value first: an original may already look like another row's new one.
UPDATE users SET username = 'anon-tmp-' || id;
UPDATE users SET
  username = CASE WHEN role = 'admin' AND id = (SELECT min(id) FROM users WHERE role = 'admin' AND active) THEN 'admin' ELSE 'user' || id END,
  fullname = pg_temp.fake_name('user:' || id),
  password_hash = crypt('pyrhouse', gen_salt('bf', 10)),
  discord_id = CASE WHEN discord_id IS NULL THEN NULL ELSE (100000000000000000 + id)::text END,
  discord_username = CASE WHEN discord_username IS NULL THEN NULL ELSE 'discord_user' || id END,
  avatar_url = NULL,
  google_id = CASE WHEN google_id IS NULL THEN NULL ELSE 'google-' || id END,
  google_email = CASE WHEN google_email IS NULL THEN NULL ELSE 'user' || id || '@example.com' END;

UPDATE schedule_volunteers v SET
  nickname = coalesce((SELECT u.username FROM users u WHERE u.id = v.user_id), 'wolontariusz' || v.id),
  city = CASE WHEN city IS NULL THEN NULL ELSE (SELECT f.city[1 + v.id % array_length(f.city, 1)] FROM fake_names f) END,
  notes = NULL,
  -- Despite the name, the Discord handle that confirmed.
  discord_confirmed = CASE WHEN coalesce(discord_confirmed, '') = '' THEN discord_confirmed ELSE 'discord_user' || coalesce(v.user_id, v.id) END;

UPDATE transfers SET
  receiver = pg_temp.fake_name(receiver),
  -- Deliveries land around the festival grounds (MTP Poznań), not where someone's phone was.
  delivery_latitude = CASE WHEN delivery_latitude IS NULL THEN NULL ELSE 52.3960 + (id % 40) * 0.0001 END,
  delivery_longitude = CASE WHEN delivery_longitude IS NULL THEN NULL ELSE 16.8990 + (id % 40) * 0.0001 END;

UPDATE equipment_request_quests SET
  recipient = pg_temp.fake_name(recipient),
  -- The sync's row key: pavilion|location|recipient (a name or an e-mail)|date|time — the recipient part is replaced.
  quest_key = split_part(quest_key, '|', 1) || '|' || split_part(quest_key, '|', 2) || '|' || pg_temp.fake_name(split_part(quest_key, '|', 3))
    || '|' || split_part(quest_key, '|', 4) || '|' || split_part(quest_key, '|', 5),
  budget_owner = CASE WHEN budget_owner IS NULL THEN NULL ELSE 'Budżet ' || upper(substr(md5(budget_owner), 1, 4)) END;
UPDATE equipment_request_items SET
  budget_owner = CASE WHEN budget_owner IS NULL THEN NULL ELSE 'Budżet ' || upper(substr(md5(budget_owner), 1, 4)) END,
  notes = CASE WHEN notes IS NULL THEN NULL ELSE 'Uwagi do pozycji ' || id END;

-- Free text people wrote, which may name anyone
UPDATE releases SET notes = CASE WHEN notes IS NULL THEN NULL ELSE 'Notatka do wydania ' || reference END;
UPDATE service_desk_requests SET
  title = 'Zgłoszenie #' || id || ' (' || type || ')',
  description = 'Opis zgłoszenia testowego #' || id || '.',
  created_by = pg_temp.fake_name(created_by);
UPDATE service_desk_request_comments SET comment = 'Komentarz testowy #' || id || '.';
UPDATE audit_logs SET data = '{}'::jsonb;
DELETE FROM audit_logs WHERE id NOT IN (SELECT id FROM audit_logs ORDER BY created_at DESC LIMIT 200);

-- Links to real documents and what syncing them logged
UPDATE app_settings SET value = 'fixture-sheet-id' WHERE key LIKE '%sheet_id';
-- The sync log is gone since migration 000049; older dumps still carry it (they are anonymised before migrating).
DO $$
BEGIN
  IF to_regclass('equipment_request_sync_log') IS NOT NULL THEN
    UPDATE equipment_request_sync_log SET sheet_id = 'fixture-sheet-id', errors = NULL;
    DELETE FROM equipment_request_sync_log WHERE id NOT IN (SELECT id FROM equipment_request_sync_log ORDER BY synced_at DESC LIMIT 50);
  END IF;
END $$;

COMMIT;
DROP EXTENSION pgcrypto;
