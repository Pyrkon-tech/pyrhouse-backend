# Database Purge Guide — Pre-Edition Reset

Instrukcja czyszczenia bazy przed nową edycją Pyrkonu. Wykonaj kroki w podanej kolejności.

---

## Co zostaje (nie ruszamy)

| Tabela | Powód |
|---|---|
| `users` | konta użytkowników zostają |
| `locations` | lokalizacje skonfigurowane ręcznie |
| `item_category` | kategorie sprzętu |
| `origins` | źródła — opcjonalnie możliwy reset (patrz niżej) |
| `app_settings` | konfiguracja aplikacji |
| `schedules`, `schedule_volunteers`, `schedule_slots`, `schedule_assignments` | grafiki już w trakcie pracy na nową edycję |
| `equipment_request_*` | sync z Google Sheets odświeży dane automatycznie |

> **Equipment requests:** tabele `equipment_request_quests` i `equipment_request_items` pozostają.
> Link `transfer_id` zostaje wyzerowany (krok 1), bo stare transfery znikają.
> Po uruchomieniu synca dane zostaną zaktualizowane ze Sheets — **nie wykonuj transferów ani nie zmieniaj statusów** przed resetem.

---

## Krok 1 — odepnij equipment requests od transferów

#### Sprawdź czy w ogóle są `not null` wcześniej 
```sql
UPDATE equipment_request_quests
SET transfer_id = NULL
WHERE transfer_id IS NOT NULL;
```

---

## Krok 2 — właściwy purge

Kolejność respektuje foreign key constraints. `RESTART IDENTITY` resetuje sekwencje — ID zaczną od 1.

```sql
BEGIN;

-- (już wykonane wyżej, można powtórzyć dla pewności)
UPDATE equipment_request_quests SET transfer_id = NULL WHERE transfer_id IS NOT NULL;

TRUNCATE TABLE
    service_desk_request_comments,
    service_desk_requests,
    pyr_code_reservations,
    release_assets,
    release_stocks,
    serialized_transfers,
    non_serialized_transfers,
    transfer_users,
    transfers,
    releases,
    items,
    non_serialized_items,
    audit_logs
RESTART IDENTITY CASCADE;

COMMIT;
```

---

## Co zostało wyczyszczone i jakie sekwencje reset

| Tabela | Sekwencja | Efekt |
|---|---|---|
| `items` | `items_id_seq → 1` | pyr kody zaczną od `PYR-XX1` (kod = `PYR-` + kategoria + `items.id`) |
| `non_serialized_items` | `non_serialized_items_id_seq → 1` | stany magazynowe wyzerowane |
| `transfers` | `transfers_id_seq → 1` | historia transferów usunięta |
| `serialized_transfers` | `serialized_transfers_id_seq → 1` | |
| `non_serialized_transfers` | `non_serialized_transfers_id_seq → 1` | |
| `transfer_users` | `transfer_users_id_seq → 1` | |
| `releases` | `releases_id_seq → 1` | wszystkie wydania usunięte |
| `release_assets` | `release_assets_id_seq → 1` | |
| `release_stocks` | `release_stocks_id_seq → 1` | |
| `pyr_code_reservations` | `pyr_code_reservations_id_seq → 1` | rezerwacje kodów wyczyszczone |
| `service_desk_requests` | `service_desk_requests_id_seq → 1` | zgłoszenia service desk usunięte |
| `service_desk_request_comments` | `service_desk_request_comments_id_seq → 1` | |
| `audit_logs` | `audit_logs_id_seq → 1` | logi akcji usunięte |

---

## Opcjonalnie — reset origins

Jeśli chcesz wyczyścić też źródła sprzętu (`origins`), odpal **po kroku 2**:

```sql
-- Tylko jeśli items i releases już wyczyszczone (są do nich FK)
TRUNCATE TABLE origins RESTART IDENTITY CASCADE;
```

> Uwaga: `origins` jest referencjonowane przez `items` i `releases`. Po truncate tych tabel w kroku 2 ten krok jest bezpieczny.

---

## Weryfikacja po wykonaniu

```sql
-- Sprawdź czy tabele puste
SELECT 'items' AS t, COUNT(*) FROM items
UNION ALL SELECT 'non_serialized_items', COUNT(*) FROM non_serialized_items
UNION ALL SELECT 'transfers', COUNT(*) FROM transfers
UNION ALL SELECT 'releases', COUNT(*) FROM releases
UNION ALL SELECT 'service_desk_requests', COUNT(*) FROM service_desk_requests
UNION ALL SELECT 'pyr_code_reservations', COUNT(*) FROM pyr_code_reservations
UNION ALL SELECT 'audit_logs', COUNT(*) FROM audit_logs;

-- Sprawdź sekwencje (powinny zaczynać od 1)
SELECT sequencename, last_value
FROM pg_sequences
WHERE sequencename IN (
    'items_id_seq',
    'transfers_id_seq',
    'releases_id_seq',
    'service_desk_requests_id_seq'
);

-- Sprawdź że equipment requests bez transferów
SELECT COUNT(*) AS quest_with_transfer FROM equipment_request_quests WHERE transfer_id IS NOT NULL;
```

---

## Przed wykonaniem

Upewnij się, że masz aktualny dump bazy:

```bash
PGPASSWORD=<hasło> pg_dump -h <host> -p <port> -U postgres pyrhouse > pyrhouse-$(date +%Y%m%d_%H%M%S).sql
```

Pliki `pyrhouse-*.sql` są w `.gitignore`.
