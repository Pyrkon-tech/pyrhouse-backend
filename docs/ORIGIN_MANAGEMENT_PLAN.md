# Plan: Origin Management — przeniesienie do bazy danych

## Stan obecny

Origin (pochodzenie) jest **hardcoded** w `internal/metadata/origin.go` jako `const`:

```
druga-era | probis | netland | targowe | dj-sound | oki-event | personal-{x} | other-{x}
```

Walidacja: `metadata.NewOrigin(value)` — whitelist + normalizacja (lowercase, trim, space→hyphen).

Uzycie:
- `items.origin` — VARCHAR(128), kazdy asset ma origin
- `non_serialized_items.origin` — VARCHAR(128), UNIQUE constraint z `(item_category_id, location_id, origin)`
- `non_serialized_transfers.origin` — VARCHAR(128)
- Walidacja w handlerach: `assets/handler.go` (create, bulk, emergency)
- Brak walidacji w stock handler — origin przyjmowany as-is

### Problemy

1. Dodanie nowego origin wymaga zmiany kodu Go + deploy
2. Brak CRUD — admin nie moze zarzadzac originami z frontu
3. Niespojnosc — Asset uzywa `metadata.Origin` (typed), Stock uzywa `string`
4. Brak walidacji origin w stock endpoints
5. `personal-{x}` i `other-{x}` to specjalne formaty — trzeba zachowac

---

## Architektura rozwiazania

> **Uwaga:** Baza bedzie czyszczona przed nastepnym eventem, wiec robimy pelna migracje danych — zamiana VARCHAR `origin` na `origin_id INT FK`.

### Nowa tabela `origins`

```sql
CREATE TABLE origins (
    id SERIAL PRIMARY KEY,
    slug VARCHAR(128) NOT NULL UNIQUE,        -- "druga-era", "probis", etc.
    label VARCHAR(255) NOT NULL,              -- "Druga Era", "Probis", etc.
    allow_suffix BOOLEAN DEFAULT false,       -- true for "personal", "other"
    active BOOLEAN DEFAULT true,              -- soft-delete
    sort_order INT DEFAULT 0,                 -- display ordering
    created_at TIMESTAMP DEFAULT NOW()
);
```

**`allow_suffix = true`** — oznacza ze origin akceptuje format `slug-{suffix}` (np. `personal-hagrid`, `other-sponsor`). Frontend wyswietla dodatkowe pole tekstowe. Suffix jest przechowywany w osobnej kolumnie `origin_suffix` obok `origin_id`.

**`active = false`** — origin nie jest dostepny do wyboru, ale istniejace rekordy z nim pozostaja.

### Seed data (migracja)

```sql
INSERT INTO origins (slug, label, allow_suffix, sort_order) VALUES
  ('druga-era',  'Druga Era',  false, 1),
  ('probis',     'Probis',     false, 2),
  ('netland',    'Netland',    false, 3),
  ('targowe',    'Targowe',    false, 4),
  ('dj-sound',   'DJ Sound',   false, 5),
  ('oki-event',  'Oki Event',  false, 6),
  ('personal',   'Personal',   true,  7),
  ('other',      'Other',      true,  8);
```

### Migracja kolumn `origin` → `origin_id` + `origin_suffix`

Dotyczy 3 tabel:

| Tabela | Stara kolumna | Nowe kolumny |
|--------|---------------|-------------|
| `items` | `origin VARCHAR(128)` | `origin_id INT FK`, `origin_suffix VARCHAR(128)` |
| `non_serialized_items` | `origin VARCHAR(128)` | `origin_id INT FK`, `origin_suffix VARCHAR(128)` |
| `non_serialized_transfers` | `origin VARCHAR(128)` | `origin_id INT FK`, `origin_suffix VARCHAR(128)` |

**`origin_suffix`** — przechowuje suffix dla originow z `allow_suffix=true` (np. dla `personal-hagrid` → `origin_id` wskazuje na `personal`, `origin_suffix = "hagrid"`). Dla zwyklych originow `origin_suffix` jest NULL.

**UNIQUE constraint na `non_serialized_items`** zmienia sie z `(item_category_id, location_id, origin)` na `(item_category_id, location_id, origin_id, origin_suffix)`.

### Walidacja — nowa logika

Zamiast hardcoded whitelist w `metadata/origin.go`, walidacja odpytuje DB:

```
input: "druga-era"     → slug "druga-era" istnieje, allow_suffix=false → origin_id=1, suffix=NULL
input: "personal-jan"  → slug "personal" istnieje, allow_suffix=true   → origin_id=7, suffix="jan"
input: "xyz"           → slug "xyz" nie istnieje                       → INVALID
```

Na wyjsciu API origin jest nadal zwracany jako string (slug + ewentualny suffix) dla backward compatibility z frontendem.

---

## Plan implementacji

### Krok 1: Migracja `000030_origins_table`

**Plik:** `migrations/000030_origins_table.up.sql`

```sql
-- 1. Tabela origins + seed
CREATE TABLE origins (
    id SERIAL PRIMARY KEY,
    slug VARCHAR(128) NOT NULL UNIQUE,
    label VARCHAR(255) NOT NULL,
    allow_suffix BOOLEAN DEFAULT false,
    active BOOLEAN DEFAULT true,
    sort_order INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);

INSERT INTO origins (slug, label, allow_suffix, sort_order) VALUES
  ('druga-era',  'Druga Era',  false, 1),
  ('probis',     'Probis',     false, 2),
  ('netland',    'Netland',    false, 3),
  ('targowe',    'Targowe',    false, 4),
  ('dj-sound',   'DJ Sound',   false, 5),
  ('oki-event',  'Oki Event',  false, 6),
  ('personal',   'Personal',   true,  7),
  ('other',      'Other',      true,  8);

-- 2. items: origin VARCHAR → origin_id FK + origin_suffix
ALTER TABLE items ADD COLUMN origin_id INT REFERENCES origins(id);
ALTER TABLE items ADD COLUMN origin_suffix VARCHAR(128);
-- Migracja danych (exact slug match)
UPDATE items SET origin_id = o.id
  FROM origins o WHERE items.origin = o.slug;
-- Migracja danych (suffix match: "personal-jan" → origin_id=personal, suffix="jan")
UPDATE items SET
  origin_id = o.id,
  origin_suffix = SUBSTRING(items.origin FROM LENGTH(o.slug) + 2)
  FROM origins o
  WHERE items.origin_id IS NULL
    AND o.allow_suffix = true
    AND items.origin LIKE o.slug || '-%';
ALTER TABLE items DROP COLUMN origin;

-- 3. non_serialized_items: origin VARCHAR → origin_id FK + origin_suffix
ALTER TABLE non_serialized_items DROP CONSTRAINT IF EXISTS non_serialized_items_unique_constraint;
ALTER TABLE non_serialized_items ADD COLUMN origin_id INT REFERENCES origins(id);
ALTER TABLE non_serialized_items ADD COLUMN origin_suffix VARCHAR(128);
UPDATE non_serialized_items SET origin_id = o.id
  FROM origins o WHERE non_serialized_items.origin = o.slug;
UPDATE non_serialized_items SET
  origin_id = o.id,
  origin_suffix = SUBSTRING(non_serialized_items.origin FROM LENGTH(o.slug) + 2)
  FROM origins o
  WHERE non_serialized_items.origin_id IS NULL
    AND o.allow_suffix = true
    AND non_serialized_items.origin LIKE o.slug || '-%';
ALTER TABLE non_serialized_items DROP COLUMN origin;
ALTER TABLE non_serialized_items ADD CONSTRAINT non_serialized_items_unique_constraint
  UNIQUE (item_category_id, location_id, origin_id, origin_suffix);

-- 4. non_serialized_transfers: origin VARCHAR → origin_id FK + origin_suffix
ALTER TABLE non_serialized_transfers ADD COLUMN origin_id INT REFERENCES origins(id);
ALTER TABLE non_serialized_transfers ADD COLUMN origin_suffix VARCHAR(128);
UPDATE non_serialized_transfers SET origin_id = o.id
  FROM origins o WHERE non_serialized_transfers.origin = o.slug;
UPDATE non_serialized_transfers SET
  origin_id = o.id,
  origin_suffix = SUBSTRING(non_serialized_transfers.origin FROM LENGTH(o.slug) + 2)
  FROM origins o
  WHERE non_serialized_transfers.origin_id IS NULL
    AND o.allow_suffix = true
    AND non_serialized_transfers.origin LIKE o.slug || '-%';
ALTER TABLE non_serialized_transfers DROP COLUMN origin;
```

**Down:** `migrations/000030_origins_table.down.sql`

```sql
-- Restore VARCHAR origin columns
ALTER TABLE items ADD COLUMN origin VARCHAR(128);
UPDATE items SET origin = CASE
  WHEN origin_suffix IS NOT NULL THEN o.slug || '-' || origin_suffix
  ELSE o.slug END
  FROM origins o WHERE items.origin_id = o.id;
ALTER TABLE items DROP COLUMN origin_id, DROP COLUMN origin_suffix;

ALTER TABLE non_serialized_items DROP CONSTRAINT IF EXISTS non_serialized_items_unique_constraint;
ALTER TABLE non_serialized_items ADD COLUMN origin VARCHAR(128);
UPDATE non_serialized_items SET origin = CASE
  WHEN origin_suffix IS NOT NULL THEN o.slug || '-' || origin_suffix
  ELSE o.slug END
  FROM origins o WHERE non_serialized_items.origin_id = o.id;
ALTER TABLE non_serialized_items DROP COLUMN origin_id, DROP COLUMN origin_suffix;
ALTER TABLE non_serialized_items ADD CONSTRAINT non_serialized_items_unique_constraint
  UNIQUE (item_category_id, location_id, origin);

ALTER TABLE non_serialized_transfers ADD COLUMN origin VARCHAR(128);
UPDATE non_serialized_transfers SET origin = CASE
  WHEN origin_suffix IS NOT NULL THEN o.slug || '-' || origin_suffix
  ELSE o.slug END
  FROM origins o WHERE non_serialized_transfers.origin_id = o.id;
ALTER TABLE non_serialized_transfers DROP COLUMN origin_id, DROP COLUMN origin_suffix;

DROP TABLE origins;
```

### Krok 2: Modul `internal/origins/`

Nowy modul wg wzorca Handler → Service → Repository:

**`repository.go`:**
```go
type Origin struct {
    ID          int       `json:"id" db:"id"`
    Slug        string    `json:"slug" db:"slug"`
    Label       string    `json:"label" db:"label"`
    AllowSuffix bool      `json:"allow_suffix" db:"allow_suffix"`
    Active      bool      `json:"active" db:"active"`
    SortOrder   int       `json:"sort_order" db:"sort_order"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// Metody:
GetAll(ctx) ([]Origin, error)                    // SELECT ... WHERE active = true ORDER BY sort_order
GetBySlug(ctx, slug string) (*Origin, error)     // dla walidacji
Create(ctx, origin *Origin) error
Update(ctx, id int, req UpdateRequest) error
Deactivate(ctx, id int) error                    // soft-delete: active = false
```

**`service.go`:**
```go
// OriginResolution — wynik walidacji, gotowy do zapisu w DB
type OriginResolution struct {
    OriginID     int
    OriginSuffix *string  // nil dla zwyklych originow, "jan" dla "personal-jan"
    DisplayValue string   // "druga-era" lub "personal-jan" — do zwracania w API response
}

// ResolveOrigin waliduje i parsuje origin string na origin_id + suffix
// Zastepuje metadata.NewOrigin()
func (s *Service) ResolveOrigin(ctx context.Context, value string) (*OriginResolution, error) {
    normalized := normalizeOrigin(value)  // lowercase, trim, space→hyphen

    // 1. Exact match — slug istnieje i active
    if origin, _ := s.repo.GetBySlug(ctx, normalized); origin != nil && origin.Active {
        return &OriginResolution{OriginID: origin.ID, DisplayValue: normalized}, nil
    }

    // 2. Suffix match — "personal-jan" → szukaj slug "personal" z allow_suffix=true
    if idx := strings.Index(normalized, "-"); idx > 0 {
        base := normalized[:idx]
        suffix := normalized[idx+1:]
        if origin, _ := s.repo.GetBySlug(ctx, base); origin != nil && origin.AllowSuffix && origin.Active {
            return &OriginResolution{
                OriginID: origin.ID, OriginSuffix: &suffix, DisplayValue: normalized,
            }, nil
        }
    }

    return nil, fmt.Errorf("invalid origin: %s", value)
}
```

**`handler.go`:**

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/origins` | JWT | Lista aktywnych originow (populowanie dropdown na FE) |
| GET | `/origins/all` | moderator | Lista wszystkich (wlacznie z inactive) — do panelu admin |
| POST | `/origins` | admin | Dodaj nowy origin |
| PATCH | `/origins/:id` | admin | Edytuj (label, sort_order, allow_suffix, active) |
| DELETE | `/origins/:id` | admin | Soft-delete (active = false) |

### Krok 3: Integracja z istniejacymi handlerami

**Zmiana modeli Go** — we wszystkich modelach `Origin string` zamieniane na:
```go
OriginID     *int    `json:"-" db:"origin_id"`
OriginSuffix *string `json:"-" db:"origin_suffix"`
Origin       string  `json:"origin"` // computed: slug + optional suffix — nie zapisywane do DB
```

API nadal zwraca `"origin": "personal-jan"` (string) — wartosc jest skladana z JOIN na `origins.slug` + `origin_suffix`.

**`internal/inventory/assets/handler.go`:**

Zamienic `metadata.NewOrigin(req.Origin)` na `originService.ResolveOrigin(ctx, req.Origin)`.
Wynik `OriginResolution` daje `origin_id` i `origin_suffix` do zapisu.

Dotyczy metod:
- `CreateAsset` (linia ~80)
- `getRequestDefaults` (linia ~245)

**`internal/inventory/assets/repository.go`:**

Zmiana zapisu — zamiast `"origin": req.Origin` teraz:
```go
"origin_id":     resolution.OriginID,
"origin_suffix": resolution.OriginSuffix,
```

Zmiana odczytu — SELECT z JOIN na origins:
```go
goqu.L("CASE WHEN i.origin_suffix IS NOT NULL THEN o.slug || '-' || i.origin_suffix ELSE o.slug END").As("origin")
```

**`internal/inventory/stocks/service.go`:**

Dodac walidacje origin (obecnie brak!):
```go
func (s *StockService) CreateStock(ctx, req) {
    if req.Origin != "" {
        resolution, err := s.originService.ResolveOrigin(ctx, req.Origin)
        // zapisz resolution.OriginID + resolution.OriginSuffix
    }
}
```

**`internal/inventory/stocks/repository.go`:**

Analogiczna zmiana jak w assets — zapis `origin_id` + `origin_suffix`, odczyt z JOIN.

**UNIQUE constraint zmiana:** `PersistStockItem()` uzywa ON CONFLICT — trzeba zaktualizowac z `(item_category_id, location_id, origin)` na `(item_category_id, location_id, origin_id, origin_suffix)`.

**`internal/inventory/transfers/repository.go`:**

`InsertStockItemsTransferRecord()` — zamiast subselect na `origin`:
```go
"origin_id":     goqu.L("(SELECT origin_id FROM non_serialized_items WHERE id = ?)", stockItem.ID),
"origin_suffix": goqu.L("(SELECT origin_suffix FROM non_serialized_items WHERE id = ?)", stockItem.ID),
```

### Krok 4: DI + Routing

**`internal/di/container.go`:**
```go
OriginRepository *origins.Repository
OriginService    *origins.Service
OriginHandler    *origins.Handler
```

**`internal/routing/routes.go`:**
```go
container.OriginHandler.RegisterRoutes(protectedRoutes)
```

### Krok 5: Usuniecie `metadata/origin.go`

Skoro robimy pelna migracje danych, `metadata/origin.go` mozna **usunac calkowicie**:
1. Usun plik `internal/metadata/origin.go`
2. Usun wszystkie importy `metadata.Origin` i `metadata.NewOrigin()`
3. Cala walidacja przechodzi przez `originService.ResolveOrigin()`

### Krok 6: OpenAPI

Zaktualizowac `docs/openapi.yaml`:
- Nowy tag `origins`
- 5 endpointow z request/response schemas
- Zmienic origin w asset/stock schemas z `enum` na `string` z opisem "validated against origins table"

---

## Frontend — co sie zmienia

### Dropdown origin

Zamiast hardcoded listy, FE pobiera:
```
GET /origins → [{ slug: "druga-era", label: "Druga Era", allow_suffix: false }, ...]
```

Gdy user wybierze origin z `allow_suffix: true` (np. "Personal") → FE wyswietla dodatkowe pole na suffix, a wysyla `"personal-jan"`.

### Panel administracyjny

Nowy widok (admin only):
- Lista originow (z `/origins/all`)
- Dodawanie nowych
- Edycja label/sort_order
- Dezaktywacja (soft-delete)

---

## Wplyw na istniejace dane

**Pelna migracja danych.** Baza bedzie czyszczona przed eventem, wiec:

1. Migracja konwertuje istniejace `origin` VARCHAR na `origin_id` FK + `origin_suffix`
2. Stare kolumny `origin` sa usuwane
3. Dane sa migrowane automatycznie w SQL (exact match + suffix match)
4. `metadata/origin.go` jest usuwany — zero legacy kodu

**Korzysci pelnej migracji:**
- Referential integrity — FK zapewnia ze origin_id zawsze wskazuje na istniejacy rekord
- Brak duplikacji — jeden origin slug, jedna wartosc w tabeli
- Prostsza walidacja — wystarczy sprawdzic czy origin_id istnieje
- Lepsza wydajnosc — JOIN na INT zamiast porownywania stringow

**Ryzyka:**
- Jesli w danych sa origin stringi niezmapowane do zadnego seed origin → `origin_id` pozostanie NULL po migracji. Migracja powinna to obsluzyc (log warning lub domyslny origin).

---

## Kolejnosc wdrazania

```
1. Migracja DB (000030)           — tabela + seed + konwersja kolumn we wszystkich tabelach
2. Repository + Service + Handler — CRUD + ResolveOrigin
3. DI + Routing                   — wiring
4. Integracja assets              — handler + repository (origin_id/origin_suffix zamiast origin string)
5. Integracja stocks              — service + repository (dodanie walidacji + zmiana kolumn)
6. Integracja transfers           — repository (zmiana subselect na origin_id/origin_suffix)
7. Usuniecie metadata/origin.go   — cleanup legacy kodu
8. OpenAPI                        — aktualizacja
9. Testy                          — unit + handler tests
```

## Estymacja zlozonosci

| Krok | Pliki | Zlozonosc |
|------|-------|-----------|
| Migracja | 2 (up + down) | Srednia (migracja danych w 3 tabelach) |
| Modul origins/ | 3 (handler, service, repo) | Srednia |
| DI + Routing | 2 | Niska |
| Integracja assets | 2 (handler + repo) | Srednia (zmiana modeli + queries) |
| Integracja stocks | 2 (service + repo) | Srednia (dodanie walidacji + zmiana queries) |
| Integracja transfers | 1 (repo) | Niska |
| Cleanup metadata | 1 + usuwanie importow | Niska |
| OpenAPI | 1 | Srednia |
| Testy | 2-3 | Srednia |
