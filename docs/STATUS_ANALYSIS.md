# Analiza statusów sprzętu — problemy i plan naprawy

## 1. Stan obecny

### Zdefiniowane statusy (`internal/metadata/status.go`)

| Status | Opis | Używany? |
|--------|------|----------|
| `in_stock` | Deprecated — nigdzie nie ustawiany automatycznie | Tylko w `CanRemoveAsset()` jako warunek |
| `in_transit` | W trakcie transferu | Tak — ustawiany przy `InitTransfer()` |
| `located` | Po dostarczeniu / po anulowaniu transferu | Tak — ustawiany przy `ConfirmTransfer()` i `CancelTransfer()` |
| `completed` | ? | Nigdy nie ustawiany na item — tylko na transfer |
| `available` | Domyślny status nowego asseta | Tak — domyślny przy tworzeniu + przy usunięciu z transferu |
| `unavailable` | ? | Nigdzie nie ustawiany |
| `cancelled` | ? | Nigdzie nie ustawiany na item — tylko na transfer |

### Zduplikowane stałe w `internal/models/asset.go`

```go
AssetStatusInStock   = "in_stock"
AssetStatusInTransit = "in_transit"
AssetStatusDelivered = "delivered"  // ← nie istnieje w metadata/status.go!
```

Te stałe nie są używane w logice — to martwy kod, ale `delivered` to potencjalny bug w przyszłości.

---

## 2. Aktualny cykl życia asseta

```
Tworzenie → status = "available", location = 1 (Magazyn Techniczny)
    │
    ▼
InitTransfer() → status = "in_transit", location = destination
    │
    ├── ConfirmTransfer() → status = "located"
    │
    ├── CancelTransfer() → status = "located", location = oryginalna
    │
    └── RemoveAssetFromTransfer() → status = "available", location = wskazana
```

### Problemy

#### Problem 1: `located` vs `available` — brak jasnej semantyki
- **Nowy asset** dostaje `available`
- **Po dostarczeniu** dostaje `located`
- **Po anulowaniu** też dostaje `located` (wraca skąd przyszedł)
- **Po usunięciu z transferu** dostaje `available`

Pytanie: czym się różni `available` od `located`? W praktyce oba oznaczają "asset jest na miejscu, nie jest w trakcie transportu". Nie ma logiki biznesowej, która traktuje je inaczej (poza `CanRemoveAsset` i `releases/suggest`).

#### Problem 2: Powrót do magazynu (location_id=1) nie resetuje statusu
Transfer z lokalizacji eventowej **do** Magazynu Technicznego (ID=1) skutkuje:
- `ConfirmTransfer()` → status = `located`

Ale asset stworzony bezpośrednio w magazynie ma status = `available`. Więc ten sam asset w tym samym miejscu ma **różny status** w zależności od historii.

To powoduje niespójność: `CanRemoveAsset()` wymaga `in_stock` LUB `available` — asset ze statusem `located` w magazynie **nie może być usunięty** mimo że jest fizycznie w magazynie.

#### Problem 3: `in_stock` jest deprecated ale wciąż wymagany
`CanRemoveAsset()` sprawdza:
```go
"items.status": goqu.Op{"in": []string{"in_stock", "available"}}
```
Ale żaden flow nie ustawia `in_stock`. Sprzęt importowany dawno temu może mieć ten status w bazie, ale nowy sprzęt nigdy go nie dostanie.

#### Problem 4: `completed`, `unavailable`, `cancelled` — statusy widma
Zdefiniowane w `metadata/status.go`, ale **nigdy nie ustawiane na itemach**:
- `completed` — używany tylko jako status transferu
- `cancelled` — używany tylko jako status transferu
- `unavailable` — nigdzie nie używany

Te statusy mieszają koncepty: statusy **transferu** vs statusy **asseta** dzielą ten sam enum.

#### Problem 5: `releases/suggest` filtruje po `available` i `located`
```go
goqu.I("i.status").In([]string{"available", "located"})
```
To jedyne miejsce gdzie oba statusy są traktowane jako "gotowe do operacji". Potwierdza to, że oba oznaczają to samo.

---

## 3. Plan naprawy

### Cel: uproszczenie do 3 statusów asseta

| Status | Znaczenie |
|--------|-----------|
| `available` | Asset jest na swoim miejscu, gotowy do użycia/transferu/wydania |
| `in_transit` | Asset jest w trakcie transferu |
| `unavailable` | Asset jest tymczasowo niedostępny (np. uszkodzony, w serwisie — na przyszłość) |

### Krok 1: Migracja danych w bazie

Nowa migracja SQL:

```sql
BEGIN;

-- Zunifikuj located → available (to samo znaczenie)
UPDATE items SET status = 'available' WHERE status = 'located';

-- Zunifikuj in_stock → available (deprecated)
UPDATE items SET status = 'available' WHERE status = 'in_stock';

-- Zunifikuj completed → available (nie powinien istnieć na itemach, ale na wszelki wypadek)
UPDATE items SET status = 'available' WHERE status = 'completed';

-- Zunifikuj cancelled → available (nie powinien istnieć na itemach, ale na wszelki wypadek)
UPDATE items SET status = 'available' WHERE status = 'cancelled';

COMMIT;
```

### Krok 2: Napraw `ConfirmTransfer()` — status po dostarczeniu

**Plik:** `internal/inventory/transfers/service.go:351`

Zmiana:
```go
// BYŁO:
s.ar.UpdateItemStatus(assetIDs, metadata.StatusLocated, tx)

// MA BYĆ:
s.ar.UpdateItemStatus(assetIDs, metadata.StatusAvailable, tx)
```

Po dostarczeniu asset jest `available` — bez względu na lokalizację.

### Krok 3: Napraw `CancelTransfer()` — status po anulowaniu

**Plik:** `internal/inventory/transfers/service.go:415`

Zmiana:
```go
// BYŁO:
s.ar.UpdateAssetStatusAndLocation(tx, assetID, transfer.FromLocation.ID, metadata.StatusLocated)

// MA BYĆ:
s.ar.UpdateAssetStatusAndLocation(tx, assetID, transfer.FromLocation.ID, metadata.StatusAvailable)
```

### Krok 4: Uprostij `CanRemoveAsset()`

**Plik:** `internal/inventory/assets/repository.go:151`

Zmiana:
```go
// BYŁO:
"items.status": goqu.Op{"in": []string{string(metadata.StatusInStock), string(metadata.StatusAvailable)}}

// MA BYĆ:
"items.status": string(metadata.StatusAvailable)
```

### Krok 5: Uprostij `releases/repository.go` — suggest i walidacja

**Plik:** `internal/releases/repository.go:36`
```go
// BYŁO:
goqu.I("i.status").In([]string{"available", "located"})

// MA BYĆ:
goqu.Ex{"i.status": "available"}
```

**Plik:** `internal/releases/repository.go:371`
```go
// BYŁO:
goqu.I("status").In([]string{"available", "located"})

// MA BYĆ:
goqu.Ex{"status": "available"}
```

### Krok 6: Wyczyść `metadata/status.go`

```go
const (
    StatusAvailable   Status = "available"    // Na miejscu, gotowy
    StatusInTransit   Status = "in_transit"   // W trakcie transferu
    StatusUnavailable Status = "unavailable"  // Niedostępny (przyszłe użycie)
)
```

Usunięte: `in_stock` (deprecated), `located` (= available), `completed` (status transferu), `cancelled` (status transferu).

### Krok 7: Usuń martwy kod z `internal/models/asset.go`

Usuń:
```go
const (
    AssetStatusInStock   string = "in_stock"
    AssetStatusInTransit string = "in_transit"
    AssetStatusDelivered string = "delivered"
)
```

Te stałe nie są nigdzie używane i kolidują z `metadata/status.go`.

### Krok 8: Walidacja `isValid()` — zaktualizuj

```go
func (s Status) isValid() bool {
    switch s {
    case StatusAvailable, StatusInTransit, StatusUnavailable:
        return true
    default:
        return false
    }
}
```

---

## 4. Statusy transferów — osobny byt

Transfery mają własne statusy w tabeli `transfers` i `non_serialized_transfers`:

| Status | Znaczenie | Zarządza |
|--------|-----------|----------|
| `in_transit` | Transfer w trakcie | `InitTransfer()` |
| `completed` | Transfer dostarczony | `ConfirmTransfer()` |
| `cancelled` | Transfer anulowany | `CancelTransfer()` |

**Te statusy są poprawne i nie wymagają zmian.** Problem polegał na tym, że `completed` i `cancelled` były zdefiniowane w `metadata/status.go` jako statusy **assetów**, co sugerowało że asset może mieć status `completed` — a to nie ma sensu.

### Rozdzielenie

Po naprawie:
- `metadata/status.go` — zawiera **tylko** statusy assetów (`available`, `in_transit`, `unavailable`)
- Statusy transferów — to zwykłe stringi w tabeli `transfers`, nie potrzebują enuma (są zarządzane tylko w 3 metodach serwisu)

### Czy potrzebna maszyna stanów?

**Nie.** Przy 2 aktywnych stanach asseta (`available` ↔ `in_transit`) i 3 stanach transferu nie ma niejednoznacznych przejść. Cały flow jest liniowy:

```
Asset:     available ──→ in_transit ──→ available
                              │
                              └── (cancel) ──→ available

Transfer:  in_transit ──→ completed
                     └──→ cancelled
```

Formalna state machine (z interfejsami, transition maps, event handlers) miałaby sens gdyby:
- Było 5+ stanów z nieoczywistymi przejściami
- Różne moduły mogły zmieniać status niezależnie
- Potrzebne były hooki/eventy na przejścia

Tutaj wystarczy że 3 metody w `TransferService` kontrolują przejścia — to i tak jest de facto maszyna stanów, tylko bez abstrakcji.

---

## 5. Kwestia Magazynu Technicznego (location_id=1)

### Czy powinien istnieć?

**Tak.** Location ID=1 pełni ważną rolę:
- Domyślna lokalizacja dla nowego sprzętu
- Punkt wyjścia dla transferów
- `CanRemoveAsset()` wymaga `location_id = 1` (tylko z magazynu można usuwać)
- `DecreaseStockItemsQuantity()` nie kasuje stocków z location_id=1 (zabezpieczenie)

### Czy powoduje problem ze statusem?

**Nie sam w sobie.** Problem polega na tym, że status nie jest powiązany z lokalizacją. Po naprawie:
- Asset w dowolnej lokalizacji po dostarczeniu = `available`
- Asset w magazynie po utworzeniu = `available`
- Spójność zapewniona

---

## 6. Podsumowanie zmian

| Plik | Zmiana |
|------|--------|
| `migrations/000035_*.sql` | Migracja: located/in_stock/completed/cancelled → available |
| `internal/metadata/status.go` | 3 statusy zamiast 7 |
| `internal/models/asset.go` | Usunięcie martwych stałych |
| `internal/inventory/transfers/service.go` | `ConfirmTransfer` i `CancelTransfer`: located → available |
| `internal/inventory/assets/repository.go` | `CanRemoveAsset`: uproszczenie warunku |
| `internal/releases/repository.go` | `SuggestAssets` i `ValidateAssetsForRelease`: uproszczenie |

### Ryzyka
- **Dane historyczne:** Audit log zawiera stare statusy — to OK, log jest immutable
- **Frontend:** Jeśli frontend wyświetla `located` jako osobny status, trzeba zaktualizować (ale teraz wszystko będzie `available` lub `in_transit`)
- **Istniejące assety w bazie z `in_stock`:** migracja je skonwertuje

---

## 7. Mapowanie statusów na frontend

Backend zwraca 2 statusy asseta + lokalizację. Frontend powinien mapować je na czytelne labele:

| `status` | `location_id` | Label na froncie | Kolor/badge |
|----------|---------------|------------------|-------------|
| `in_transit` | (dowolny) | "W transporcie" | żółty |
| `available` | `= 1` | "Na stanie" | zielony |
| `available` | `!= 1` | nazwa lokalizacji, np. "Hala B2" | niebieski |
| `unavailable` | (dowolny) | "Niedostępny" | czerwony |

Nie ma potrzeby dodawania osobnych statusów `in_use` / `deployed` — lokalizacja mówi nam czy sprzęt jest w magazynie czy w terenie. Dodanie takiego statusu duplikowałoby informację z `location_id`.
