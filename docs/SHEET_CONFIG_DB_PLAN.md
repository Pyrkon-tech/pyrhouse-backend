# Plan: Przeniesienie konfiguracji Google Sheets do bazy danych

## Stan obecny

`EQUIPMENT_REQUEST_SHEET_ID` i `EQUIPMENT_REQUEST_SHEET_NAME` sa w `.env` i czytane przez `config.Load()`:

```
EQUIPMENT_REQUEST_SHEET_ID=16BytrbWmyWeBGnlSIDZn1Lnb5rdspoQu_rpc5m5Vtbc
EQUIPMENT_REQUEST_SHEET_NAME=Zamówienia
```

Uzycie w kodzie:
- `internal/config/config.go` → `EquipmentRequestConfig.SheetID`, `.SheetName`
- `internal/di/container.go` → przekazywane do `equipment_requests.NewService(..., sheetID, sheetName, ...)`
- `internal/equipment_requests/service.go` → `s.sheetID`, `s.sheetName` uzywane w `FetchSheet()`

### Problem

Zmiana arkusza (np. nowy event, inny formularz) wymaga edycji `.env` + restart serwera. Admin powinien moc to zmienic z frontu bez redeployu.

---

## Architektura rozwiazania

### Nowa tabela `app_settings`

Generyczna tabela key-value na ustawienia aplikacji (nie tylko sheet — moze sluzyc do innych konfigow w przyszlosci):

```sql
CREATE TABLE app_settings (
    key VARCHAR(128) PRIMARY KEY,
    value TEXT NOT NULL,
    description VARCHAR(512),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### Seed data

```sql
INSERT INTO app_settings (key, value, description) VALUES
  ('equipment_request.sheet_id', '16BytrbWmyWeBGnlSIDZn1Lnb5rdspoQu_rpc5m5Vtbc', 'Google Sheets document ID for equipment requests'),
  ('equipment_request.sheet_name', 'Zamówienia', 'Sheet tab name within the document');
```

### Logika odczytu — DB-first, env-fallback

Serwis equipment_requests nie przechowuje `sheetID`/`sheetName` jako pola statyczne. Zamiast tego odpytuje DB przy kazdym sync:

```go
func (s *Service) getSheetConfig(ctx context.Context) (sheetID, sheetName string, err error) {
    sheetID, _ = s.settingsRepo.Get(ctx, "equipment_request.sheet_id")
    sheetName, _ = s.settingsRepo.Get(ctx, "equipment_request.sheet_name")

    // Fallback na env jesli DB puste (backward compat na czas migracji)
    if sheetID == "" {
        sheetID = s.fallbackSheetID
    }
    if sheetName == "" {
        sheetName = s.fallbackSheetName
    }

    if sheetID == "" {
        return "", "", fmt.Errorf("equipment_request.sheet_id not configured")
    }
    return sheetID, sheetName, nil
}
```

---

## Plan implementacji

### Krok 1: Migracja `000031_app_settings`

**Plik:** `migrations/000031_app_settings.up.sql`

- CREATE TABLE `app_settings`
- Seed z obecnymi wartosciami z `.env`

**Down:** DROP TABLE `app_settings`

### Krok 2: Modul `internal/settings/`

Prosty modul — bez handler/service rozdzialu (za maly scope):

**`repository.go`:**
```go
type Repository struct { repo *repository.Repository }

func (r *Repository) Get(ctx context.Context, key string) (string, error)
func (r *Repository) Set(ctx context.Context, key, value string) error
func (r *Repository) GetAll(ctx context.Context) ([]Setting, error)
func (r *Repository) GetByPrefix(ctx context.Context, prefix string) ([]Setting, error)
```

**`handler.go`:**

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/settings` | admin | Lista wszystkich ustawien |
| GET | `/settings/:key` | admin | Pobierz wartosc |
| PUT | `/settings/:key` | admin | Ustaw wartosc |

### Krok 3: Integracja z equipment_requests

**`internal/equipment_requests/service.go`:**

- Dodaj `settingsRepo *settings.Repository` do Service struct
- Zamien statyczne `s.sheetID` / `s.sheetName` na dynamiczny odczyt z `getSheetConfig()`
- Zachowaj fallback na wartosci z env (przekazane w konstruktorze)

Zmiana dotyczy 2 miejsc:
- `SyncFromSheet()` (linia ~118): `s.sheetReader.FetchSheet(s.sheetID, s.sheetName)` → dynamiczny odczyt
- `ResyncQuest()` (linia ~287): to samo

### Krok 4: DI + Routing

**`internal/di/container.go`:**
```go
settingsRepo := settings.NewRepository(repo)
settingsHandler := settings.NewHandler(settingsRepo)
```

Przekaz `settingsRepo` do `equipment_requests.NewService(...)`.

**`internal/routing/routes.go`:**
```go
container.SettingsHandler.RegisterRoutes(protectedRoutes)
```

### Krok 5: OpenAPI

Dodaj 3 endpointy settings do `docs/openapi.yaml`.

---

## Frontend

### Panel administracyjny

Nowy widok (admin only):
- Formularz z `Sheet ID` i `Sheet Name`
- Dane z `GET /settings?prefix=equipment_request`
- Zapis przez `PUT /settings/equipment_request.sheet_id`
- Opcjonalnie: po zmianie automatyczny trigger `POST /equipment-requests/sync`

---

## Backward compatibility

1. `.env` wartosci sa nadal czytane i przekazywane jako fallback do Service
2. Jesli `app_settings` jest pusta — zachowanie identyczne jak dzis
3. Jesli admin ustawi wartosc w DB — ta wartosc ma priorytet nad `.env`
4. Env vars `EQUIPMENT_REQUEST_SHEET_ID` / `EQUIPMENT_REQUEST_SHEET_NAME` mozna usunac po potwierdzeniu ze DB dziala

---

## Co NIE wchodzi w zakres

- `EQUIPMENT_REQUEST_SYNC_ENABLED`, `SYNC_INTERVAL`, `FUZZY_THRESHOLD` — zostaja w `.env` (to ustawienia infrastrukturalne, nie biznesowe)
- Hot-reload interwalu sync — wymagalby zmian w Scheduler, nie jest potrzebny

---

## Kolejnosc wdrazania

```
1. Migracja DB (000031)            — tabela + seed
2. Modul settings/                 — repository + handler
3. DI + Routing                    — wiring
4. Integracja equipment_requests   — dynamiczny odczyt sheetID/sheetName
5. OpenAPI                         — 3 nowe endpointy
6. Testy
```

## Estymacja zlozonosci

| Krok | Pliki | Zlozonosc |
|------|-------|-----------|
| Migracja | 2 | Niska |
| Modul settings/ | 2 (handler + repo) | Niska |
| DI + Routing | 2 | Niska |
| Integracja | 1 (service.go) | Niska |
| OpenAPI | 1 | Niska |
| Testy | 1-2 | Niska |
