# Backend API — Stan implementacji vs. specyfikacja

> Dokument dla frontendu. Opisuje co jest zaimplementowane zgodnie ze specyfikacją, a co jeszcze nie. Aktualizowany przy każdym etapie prac.
>
> **Spec referencyjna:** `docs/SCHEDULE_API.md`
> **Ostatnia aktualizacja:** 2026-05-05 — wszystkie etapy 1–4 zaimplementowane

---

## Legenda

| Symbol | Znaczenie |
|--------|-----------|
| ✅ | Zaimplementowane zgodnie ze specyfikacją |
| ⚠️ | Zaimplementowane, ale z odchyleniami — szczegóły poniżej |
| ❌ | Brak implementacji |

---

## Endpointy

### GET /schedule
**Status: ✅ Zgodny**

Zwraca pełny `ScheduleDetail` z `version`, `slots[].volunteers[]` w formacie `{ id: assignment_id, volunteer_id, nickname }`.

---

### POST /schedule
**Status: ✅ Zgodny**

Tworzy harmonogram i auto-generuje festiwalowe sloty godzinowe z `festival_start`/`festival_end`. Poprzedni harmonogram jest archiwizowany.

---

### PATCH /schedule/status
**Status: ✅ Zgodny**

Przyjmuje `{ "status": "published" | "draft" }`. Publish wymaga roli `admin` i braku błędów walidacyjnych. Zwraca `ScheduleDetail`.

**Uwaga:** Endpoint wymaga roli `admin` zarówno do publikacji jak i do cofnięcia do draftu.

---

### PUT /schedule/draft
**Status: ✅ Zgodny**

Bulk-save z optimistic locking. Wysyłaj `version` z ostatniego response — jeśli wersja nieaktualna, zwracany jest 409 `version_conflict`. Festival sloty są chronione — nie zostaną usunięte nawet jeśli ich nie ma w body.

---

### POST /schedule/validate
**Status: ✅ Zgodny**

---

### POST /schedule/slots
**Status: ✅ Zgodny**

`capacity: 0` jest dozwolone (oznacza "auto").

---

### PATCH /schedule/slots/:id
**Status: ✅ Zgodny**

Zmiana `type` slotu festiwalowego zwraca 422.

---

### DELETE /schedule/slots/:id
**Status: ✅ Zgodny**

Festival sloty zwracają 403. Montaż/demontaż — kaskadowe usunięcie assignmentów, 204.

---

### POST /schedule/assignments
**Status: ✅ Zgodny**

Duplicate zwraca 409 `already_assigned`. Zwraca `{ id, slot_id, volunteer_id, nickname }`.

> **Uwaga:** `Idempotency-Key` header jest przyjmowany ale ignorowany (nie implementujemy cache — edge case retry jest nieistotny przy jednym edytorze).

---

### DELETE /schedule/assignments/:id
**Status: ✅ Zgodny**

Idempotent — jeśli assignment nie istnieje, zwraca 204 (nie 404). Response: 204 bez body.

---

### POST /schedule/assignments/move
**Status: ✅ Zgodny**

Atomowe przeniesienie. Zwraca `{ deleted_assignment_id, created_assignment: { id, slot_id, volunteer_id, nickname } }`.

---

### POST /schedule/assignments/swap
**Status: ✅ Zgodny**

Zwraca `{ assignment_a: { id, slot_id, volunteer_id, nickname }, assignment_b: { ... } }`.

---

### POST /schedule/volunteers
**Status: ✅ Zgodny**

Response 200 `{ imported, updated, skipped }`. Wolontariusze o tym samym nicku są liczeni jako `updated` (nie duplikowane).

---

### POST /schedule/volunteers/import-sheet
**Status: ✅ Zgodny**

Response 200 `{ imported, updated, skipped, errors }`. Błędy parsowania pojedynczych wierszy nie przerywają importu.

---

### GET /schedule/volunteers
**Status: ✅ Zgodny**

---

### PATCH /schedule/volunteers/:id
**Status: ✅ Zgodny**

---

### POST /schedule/generate
**Status: ✅ Zgodny**

Wymaga roli `admin`. Czyści tylko assignmenty (nie sloty), uruchamia solver, zwraca `ScheduleDetail`.

---

### GET /schedule/export/csv
**Status: ✅ Zgodny**

Ścieżka `/schedule/export/csv`. Response z nagłówkami `Content-Type: text/csv` i `Content-Disposition`.

---

### POST /schedule/export/sheets
**Status: ⚠️ Niekompletny response**

Działa, ale response nie zawiera `sheet_url`:
```json
{ "rows_written": 142 }
```
Spec:
```json
{ "rows_written": 142, "sheet_url": "https://docs.google.com/..." }
```

---

## Format błędów

**Status: ✅ Zgodny**

Wszystkie błędy zwracane w formacie:
```json
{ "error": "machine_slug", "message": "Czytelny komunikat", "details": {} }
```
`details` jest pomijane jeśli nie ma dodatkowych informacji.

---

## Optimistic Locking (`version`)

**Status: ✅ Zaimplementowane**

Pole `version` istnieje w modelu i bazie danych (migracja `000037`). Przy `PUT /schedule/draft`:
- Jeśli nie wysyłasz `version` (lub `version: 0`) — locking jest pomijany (tryb kompatybilności)
- Jeśli wysyłasz `version` > 0 — sprawdzany jest match, konflikt zwraca 409 `version_conflict`

---

## Daty

**Status: ✅ Zgodny dla datetime**

Backend akceptuje i zwraca daty w ISO 8601 z offsetem (`+02:00`). `event_start`/`event_end` są zwracane jako datetime (z godziną `00:00:00`), nie jako pure date — frontend powinien to obsłużyć przez `.split('T')[0]` jeśli potrzebuje samej daty.

---

## Pozostałe odchylenia

### Status "active" vs "draft"
Wewnętrznie aktywny harmonogram ma `status: "active"` w bazie danych. Kod `GET /schedule` zwraca to bezpośrednio — frontend otrzyma `"status": "active"` zamiast `"draft"`. Spec mówi `'draft' | 'published'`. To istniejące zachowanie nie zostało zmienione, żeby nie popsuć danych.

**Tymczasowe:** frontend powinien traktować `"active"` jako `"draft"`.

---

## Migracje do uruchomienia

Po `git pull` uruchom:
```
migrate -path migrations -database $DATABASE_URL up
```
Nowa migracja: `000037_schedule_version` — dodaje kolumnę `version INT NOT NULL DEFAULT 1` do tabeli `schedules`.
