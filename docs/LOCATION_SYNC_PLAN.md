# Plan: Location Synchronization for Equipment Requests

## Problem

Formularz Google Sheets wysyla dane w formacie:

| Pawilon | Miejsce |
|---------|---------|
| 6 | Magazyn Techniczny |
| WTC | Biuro Akredytacji |
| 10 | HQ |

Tabela `locations` w bazie zawiera kolumny: `id`, `name`, `pavilion` (nullable), `details` (nullable).

Obecna metoda `ResolveLocationByPavilionAndName` robi proste `ILIKE` na `pavilion` i `name` — dziala tylko gdy dane w formularzu dokladnie pasuja do bazy. Brak mechanizmu obslugi sytuacji, gdy lokalizacja nie zostanie znaleziona.

## Cele

1. **Inteligentne laczenie** — formularz podaje `pavilion=6`, baza moze miec `pavilion="6"` lub `pavilion="Pawilon 6"` — system powinien to rozwiazac.
2. **Fallback na manualne mapowanie** — gdy lokalizacja nie pasuje, zapis do tabeli mapowania + mozliwosc uzupelnienia przez usera.
3. **CRUD dla mapowania lokalizacji** — endpoint do zarzadzania mapowaniem.

---

## Architektura rozwiazania

### 1. Tabela `equipment_request_location_mapping`

Analogicznie do istniejacego `equipment_request_category_mapping` — manualny override mapowania.

```sql
-- Migration 000029_equipment_request_location_mapping.up.sql
CREATE TABLE equipment_request_location_mapping (
    id SERIAL PRIMARY KEY,
    pavilion VARCHAR(255) NOT NULL,       -- wartosc z formularza (np. "6")
    location_name VARCHAR(255) NOT NULL,  -- wartosc z formularza (np. "Magazyn Techniczny")
    location_id INT NOT NULL,             -- FK do locations.id
    created_at TIMESTAMP DEFAULT NOW(),
    usage_count INT DEFAULT 0,
    FOREIGN KEY (location_id) REFERENCES locations(id) ON DELETE CASCADE,
    UNIQUE(pavilion, location_name)       -- jeden mapping per kombinacja
);
```

### 2. Strategia resolwowania lokalizacji (4-poziomowa, priorytetowa)

Analogicznie do category matching — wielopoziomowy fallback:

```
1. Manual mapping     → equipment_request_location_mapping (confidence: 1.0)
2. Exact match        → pavilion = "6" AND name ILIKE "Magazyn Techniczny"
3. Normalized match   → strip "Pawilon " prefix, try numeric pavilion matching
4. No match           → zwroc nil, zapisz do unresolved
```

#### Logika normalizacji pavilion:
- Formularz: `"6"` → szukaj `pavilion = '6'` OR `pavilion ILIKE '%6'` (np. "Pawilon 6")
- Formularz: `"WTC"` → szukaj `pavilion ILIKE 'WTC'`
- Formularz: `"Pawilon 6"` → wyciagnij `"6"`, szukaj `pavilion = '6'` OR `pavilion ILIKE 'Pawilon 6'`

### 3. Tracking nierozwiazanych lokalizacji

Nowe pole w quest: `location_resolved bool` + `location_id *int`.

Podczas synca:
- Jesli lokalizacja rozwiazana → `location_resolved = true`, `location_id = X`
- Jesli nie → `location_resolved = false`, `location_id = null`

Frontend moze filtrowac questy z `location_resolved = false` i pokazywac je do recznego uzupelnienia.

---

## Plan implementacji

### Krok 1: Migracja bazy danych (000029)

**Plik:** `migrations/000029_equipment_request_location_mapping.up.sql`

- Tabela `equipment_request_location_mapping` (jak wyzej)
- Dodanie kolumn do `equipment_request_quests`:
  - `location_id INT NULL REFERENCES locations(id)`
  - `location_resolved BOOLEAN DEFAULT false`

**Down migration:** usun kolumny + tabele.

### Krok 2: Repository — nowe metody

**Plik:** `internal/equipment_requests/repository.go`

Nowe metody w `QuestRepositoryInterface`:

```go
// Location mapping CRUD
GetLocationMapping(ctx context.Context, pavilion, locationName string) (*int, error)
CreateLocationMapping(ctx context.Context, mapping *LocationMapping) error
ListLocationMappings(ctx context.Context) ([]LocationMapping, error)
DeleteLocationMapping(ctx context.Context, id int) error
IncrementLocationMappingUsage(ctx context.Context, pavilion, locationName string) error

// Enhanced location resolution
ResolveLocationMultiStrategy(pavilion, name string) (*int, string, error)  // returns: locationID, matchType, error

// Unresolved tracking
UpdateQuestLocationResolution(ctx context.Context, questID string, locationID *int, resolved bool) error
ListUnresolvedLocationQuests(ctx context.Context) ([]Quest, error)
```

### Krok 3: Model — nowy struct

**Plik:** `internal/equipment_requests/models.go`

```go
type LocationMapping struct {
    ID           int       `json:"id" db:"id"`
    Pavilion     string    `json:"pavilion" db:"pavilion"`
    LocationName string    `json:"location_name" db:"location_name"`
    LocationID   int       `json:"location_id" db:"location_id"`
    CreatedAt    time.Time `json:"created_at" db:"created_at"`
    UsageCount   int       `json:"usage_count" db:"usage_count"`
}

type LocationResolution struct {
    LocationID *int   `json:"location_id,omitempty"`
    MatchType  string `json:"match_type"`  // manual, exact, normalized, none
    Pavilion   string `json:"pavilion"`
    Location   string `json:"location"`
}
```

Rozszerzenie `Quest`:
```go
LocationID       *int  `json:"location_id,omitempty" db:"location_id"`
LocationResolved bool  `json:"location_resolved" db:"location_resolved"`
```

### Krok 4: Service — nowa logika resolwowania

**Plik:** `internal/equipment_requests/service.go`

Zrefaktoryzowac `ResolveQuestLocation()` na wielopoziomowy matching:

```go
func (s *Service) ResolveQuestLocation(quest *Quest) (*int, string, error) {
    pav := quest.Destination.Pavilion
    loc := quest.Destination.Location

    // 1. Manual mapping (highest priority)
    if id, err := s.questRepo.GetLocationMapping(ctx, pav, loc); err == nil && id != nil {
        s.questRepo.IncrementLocationMappingUsage(ctx, pav, loc)
        return id, "manual", nil
    }

    // 2. Exact match — pavilion + name
    if id, err := s.questRepo.ResolveLocationByPavilionAndName(pav, loc); err == nil && id != nil {
        return id, "exact", nil
    }

    // 3. Normalized match — strip "Pawilon " prefix, try variations
    normalized := normalizePavilion(pav)
    if normalized != pav {
        if id, err := s.questRepo.ResolveLocationByPavilionAndName(normalized, loc); err == nil && id != nil {
            return id, "normalized", nil
        }
    }

    // 4. Fallback: search by name only (if unique)
    // Only use if exactly one location matches by name
    if id, err := s.questRepo.ResolveLocationByNameOnly(loc); err == nil && id != nil {
        return id, "name_only", nil
    }

    return nil, "none", nil
}

func normalizePavilion(pav string) string {
    // "Pawilon 6" → "6"
    // "pawilon 10" → "10"
    stripped := strings.TrimPrefix(strings.ToLower(pav), "pawilon ")
    if stripped != strings.ToLower(pav) {
        return stripped
    }
    return pav
}
```

Integracja z `SyncQuestsToDatabase`:
- Po upsert questa, probuj rozwiazac lokalizacje
- Zapisz wynik do `location_id` + `location_resolved`

### Krok 5: Handler — nowe endpointy

**Plik:** `internal/equipment_requests/handler.go`

| Method | Path | Description |
|--------|------|-------------|
| GET | `/equipment-requests/location-mappings` | Lista mapowania lokalizacji |
| POST | `/equipment-requests/location-mappings` | Dodaj manualne mapowanie |
| DELETE | `/equipment-requests/location-mappings/:id` | Usun mapowanie |
| GET | `/equipment-requests/quests/unresolved-locations` | Questy bez rozwiazanej lokalizacji |
| PATCH | `/equipment-requests/quests/:id/location` | Reczne przypisanie lokalizacji do questa |

#### POST `/location-mappings` — request body:
```json
{
  "pavilion": "6",
  "location_name": "Magazyn Techniczny",
  "location_id": 3
}
```

#### PATCH `/quests/:id/location` — request body:
```json
{
  "location_id": 3,
  "save_mapping": true  // opcjonalnie: zapisz jako mapping na przyszlosc
}
```
Gdy `save_mapping = true`, system automatycznie tworzy wpis w `equipment_request_location_mapping` z pavilion+location z questa.

### Krok 6: Routing + DI

- `internal/routing/routes.go` — rejestracja nowych endpointow
- `internal/di/container.go` — bez zmian (te same serwisy)

### Krok 7: Aktualizacja OpenAPI

**Plik:** `docs/openapi.yaml`

Dodac:
- Tag `equipment-request-locations` (jesli potrzebny)
- 5 nowych endpointow z request/response schemas
- Rozszerzenie Quest schema o `location_id` i `location_resolved`

### Krok 8: Testy

- Unit testy dla `normalizePavilion()`
- Unit testy dla wielopoziomowego matching w `ResolveQuestLocation()`
- Testy handler (mock repository) dla nowych endpointow
- Table-driven tests zgodnie z konwencja projektu

---

## Wplyw na istniejacy kod

| Plik | Zmiana |
|------|--------|
| `equipment_requests/repository.go` | Nowe metody + interface |
| `equipment_requests/service.go` | Refaktor `ResolveQuestLocation()`, integracja z sync |
| `equipment_requests/handler.go` | 5 nowych endpointow |
| `equipment_requests/models.go` | `LocationMapping` struct, rozszerzenie `Quest` |
| `routing/routes.go` | Nowe trasy |
| `docs/openapi.yaml` | Nowe endpointy |
| `migrations/` | 000029 — nowa tabela + kolumny |

## Kompatybilnosc wsteczna

- Istniejace questy otrzymaja `location_resolved = false` (default)
- `ResolveQuestLocation()` zwraca dodatkowy `matchType` string — obecni callers (`CreateTransferFromQuest`, `PreviewTransferFromQuest`) wymagaja drobnej aktualizacji sygnatury
- Brak breaking changes w API — nowe pola sa opcjonalne w response

## Kolejnosc wdrazania

1. Migracja DB
2. Models + Repository
3. Service (resolving logic)
4. Handler + Routing
5. OpenAPI
6. Testy
