# Frontend Guide: Zapotrzebowanie techniczne (Budget Calculator)

## Overview

The budget view aggregates all equipment requests (quests) by item type, applies supplier prices,
and shows cost estimates per supplier. The supplier list is **fully dynamic** — adding Probis,
Netland, Oki-event, or any future supplier requires only a price entry in the DB, not a code change.

Data source: `equipment_request_items` + `equipment_request_quests` (synced from Google Sheets).
Prices live in `equipment_request_price_list` (one row per item × supplier).

**All endpoints require admin role. Non-admins receive `403 Forbidden`.**

---

## API Endpoints

Base path: `/api/equipment-requests`

| Method   | Path                | Role  | Description                                              |
|----------|---------------------|-------|----------------------------------------------------------|
| `GET`    | `/budget`           | admin | Full budget summary with dynamic supplier breakdown      |
| `GET`    | `/budget/persons`   | admin | Distinct budget owners (for person filter dropdown)      |
| `GET`    | `/suppliers`        | admin | Known supplier names (for dynamic column headers)        |
| `GET`    | `/prices`           | admin | All (item, supplier, unit_price) rows                    |
| `PUT`    | `/prices`           | admin | Create or update one (item, supplier) price              |
| `DELETE` | `/prices`           | admin | Delete one (item, supplier) price (query params)         |
| `POST`   | `/prices/sync`      | admin | Sync prices from the Cennik Google Sheet                 |

### Query params for `GET /budget`

| Param          | Type    | Default | Description                                           |
|----------------|---------|---------|-------------------------------------------------------|
| `budget_owner` | string  | `""`    | Filter by person. Empty = all.                        |
| `vat`          | boolean | `false` | `true` → multiply all prices × 1.23 (gross, z VAT)   |

### Query params for `DELETE /prices`

| Param       | Required | Description          |
|-------------|----------|----------------------|
| `item_name` | yes      | e.g. `Laptop`        |
| `supplier`  | yes      | e.g. `Probis`        |

---

## TypeScript Types

```ts
export interface PriceListItem {
  id: number
  item_name: string
  supplier: string       // e.g. "Probis", "Netland", "Oki-event"
  unit_price: number
  updated_at: string     // ISO date
}

export interface UpsertPriceRequest {
  item_name: string
  supplier: string
  unit_price: number
}

export interface SupplierPrice {
  supplier: string
  unit_price: number
  total: number
}

export interface BudgetItem {
  item_name: string
  quantity: number
  prices: SupplierPrice[]  // one entry per supplier; empty array = no price from any supplier
}

export interface SupplierTotal {
  supplier: string
  total: number
}

export interface BudgetSummary {
  total_positions: number
  total_quantity: number
  supplier_totals: SupplierTotal[]  // sorted by supplier name
  unpriced_count: number
  items: BudgetItem[]
}
```

---

## Step-by-Step Frontend Flow

### 1. Load suppliers and persons on page load (parallel)

```ts
const [suppliersRes, personsRes] = await Promise.all([
  fetch('/api/equipment-requests/suppliers', { headers: { Authorization: `Bearer ${token}` } }),
  fetch('/api/equipment-requests/budget/persons', { headers: { Authorization: `Bearer ${token}` } }),
])
const { suppliers }: { suppliers: string[] } = await suppliersRes.json()
// suppliers = ["Netland", "Oki-event", "Probis"]  — sorted, use as column headers

const { persons }: { persons: string[] } = await personsRes.json()
// persons = ["Czesław Dolata", "Anna Kowalska", ...]
```

### 2. Fetch budget summary

Triggered on: page load, person filter change, VAT toggle.

```ts
const params = new URLSearchParams()
if (selectedPerson) params.set('budget_owner', selectedPerson)
if (vatEnabled) params.set('vat', 'true')

const res = await fetch(`/api/equipment-requests/budget?${params}`, {
  headers: { Authorization: `Bearer ${token}` },
})
const summary: BudgetSummary = await res.json()
```

**Example response (vat: false, all persons):**
```json
{
  "total_positions": 17,
  "total_quantity": 582,
  "supplier_totals": [
    { "supplier": "Netland",   "total": 21410 },
    { "supplier": "Oki-event", "total": 28500 },
    { "supplier": "Probis",    "total": 35860 }
  ],
  "unpriced_count": 4,
  "items": [
    {
      "item_name": "Drukarka A3",
      "quantity": 3,
      "prices": [
        { "supplier": "Netland",   "unit_price": 200, "total": 600  },
        { "supplier": "Oki-event", "unit_price": 250, "total": 750  },
        { "supplier": "Probis",    "unit_price": 300, "total": 900  }
      ]
    },
    {
      "item_name": "Przedłużacz",
      "quantity": 371,
      "prices": []
    }
  ]
}
```

---

## UI Breakdown

### Page header

```
Zapotrzebowanie techniczne
Wybierz osobę odpowiedzialną za budżet — zobaczysz jej zamówienia i wyceny od dostawców.
```

### Controls bar

```
[Dropdown: — Wszyscy — ▾]   [Zamówienia | Porównanie dostawców]   [☐ Ceny brutto (z 23% VAT)]
```

### Summary cards (always visible)

Build one card per supplier from `summary.supplier_totals`, plus the fixed POZYCJE / BEZ WYCENY cards:

```
┌─────────────┐  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────┐
│ POZYCJE     │  │ NETLAND (EST.)   │  │ OKI-EVENT (EST.) │  │ PROBIS (EST.)    │  │ BEZ WYCENY  │
│ 17          │  │ 21 410 zł        │  │ 28 500 zł        │  │ 35 860 zł        │  │ 4            │
│ 582 szt.    │  │ netto            │  │ netto            │  │ netto            │  │ pozycji      │
└─────────────┘  └──────────────────┘  └──────────────────┘  └──────────────────┘  └──────────────┘
```

```ts
// Build supplier total cards dynamically
summary.supplier_totals.map(st => (
  <SummaryCard title={`${st.supplier.toUpperCase()} (EST.)`} value={st.total} unit="netto" />
))
```

---

## Tab: Zamówienia (Order Table)

Columns are generated dynamically from the `suppliers` list loaded at startup:

```ts
// Column definition (build once on load)
const supplierColumns = suppliers.flatMap(supplier => [
  { key: `${supplier}_unit`,  label: `${supplier} / SZT` },
  { key: `${supplier}_total`, label: `${supplier} RAZEM` },
])

// Row rendering
function getSupplierCell(item: BudgetItem, supplier: string): { unit: number | null, total: number | null } {
  const p = item.prices.find(p => p.supplier === supplier)
  return p ? { unit: p.unit_price, total: p.total } : { unit: null, total: null }
}
```

```
PRZEDMIOT     ILOŚĆ  NETLAND/SZT  NETLAND RAZEM  OKI-EVENT/SZT  OKI-EVENT RAZEM  PROBIS/SZT  PROBIS RAZEM
──────────────────────────────────────────────────────────────────────────────────────────────────────────
Drukarka A3     3      200 zł        600 zł          250 zł          750 zł          300 zł       900 zł
Przedłużacz   371      brak            —              brak              —              brak           —
──────────────────────────────────────────────────────────────────────────────────────────────────────────
SUMA                              21 410 zł                        28 500 zł                   35 860 zł
```

**Rendering rules:**
- `prices` empty or no entry for supplier → show `"brak"` in unit column, `"—"` in total column
- Footer SUMA row: per-supplier total from `summary.supplier_totals`
- On mobile: chip toggle to show one supplier at a time

---

## Tab: Porównanie dostawców (Supplier Comparison)

```ts
const sorted = [...summary.supplier_totals].sort((a, b) => a.total - b.total)
const cheapest = sorted[0]
const mostExpensive = sorted[sorted.length - 1]
const savings = mostExpensive.total - cheapest.total
const savingsPct = Math.round((savings / mostExpensive.total) * 100)
```

### Cheaper supplier banner

```
✅ Tańszy dostawca: Netland
Oszczędność: ~14 450 zł netto vs Probis (ok. 40% mniej)
⚠ Szacunek — uwzględnia tylko pozycje z ceną u danego dostawcy.
```

### N-column breakdown (one column per supplier)

```ts
// Only include items where this supplier has a price
summary.supplier_totals.map(st => (
  <SupplierColumn
    supplier={st.supplier}
    grandTotal={st.total}
    items={summary.items
      .map(i => ({ name: i.item_name, qty: i.quantity, price: i.prices.find(p => p.supplier === st.supplier) }))
      .filter(i => i.price != null)}
  />
))
```

---

## Price List Management (Admin UI)

### Load and display

```ts
const res = await fetch('/api/equipment-requests/prices', { headers: { Authorization: `Bearer ${token}` } })
const { prices }: { prices: PriceListItem[] } = await res.json()
// Group by item_name for a matrix display:
// Rows = items, Columns = suppliers
```

### Upsert a price (add new supplier or update existing)

```ts
await fetch('/api/equipment-requests/prices', {
  method: 'PUT',
  headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
  body: JSON.stringify({
    item_name: 'Laptop',
    supplier: 'Oki-event',   // new supplier — no backend change needed
    unit_price: 120,
  } satisfies UpsertPriceRequest),
})
```

### Delete a price entry

```ts
const params = new URLSearchParams({ item_name: 'Laptop', supplier: 'Probis' })
await fetch(`/api/equipment-requests/prices?${params}`, {
  method: 'DELETE',
  headers: { Authorization: `Bearer ${token}` },
})
```

### Sync prices from Cennik sheet

The sheet header row defines supplier columns:

```
Rzeczy    Probis  Netland  Oki-event
Laptop    140     90       120
Tablet    130     130
```

```ts
const res = await fetch('/api/equipment-requests/prices/sync', {
  method: 'POST',
  headers: { Authorization: `Bearer ${token}` },
})
const { message, updated } = await res.json()
// "Synced 36 price entries from Cennik sheet"
```

---

## Adding a New Supplier

No code changes needed on backend or frontend (if the table uses dynamic columns):

1. Add prices to the DB via `PUT /prices` with the new `supplier` name, or add a column to the Cennik sheet and run `POST /prices/sync`
2. The new supplier appears automatically in `GET /suppliers` response
3. Frontend re-fetches suppliers on mount → new column appears in table and comparison view

---

## Authorization

Gate the entire page on `currentUser.role === 'admin'`. On `403`, show "Brak uprawnień" and redirect.

```ts
if (currentUser.role !== 'admin') return <Navigate to="/" />
```

---

## Refresh Strategy

| Event                     | Action                                       |
|---------------------------|----------------------------------------------|
| Person filter / VAT change | Re-fetch `GET /budget`                      |
| After `POST /sync` (quests)| Re-fetch `GET /budget`                      |
| After price upsert/delete  | Re-fetch `GET /budget` + `GET /prices`      |
| After `POST /prices/sync`  | Re-fetch `GET /budget` + `GET /prices` + `GET /suppliers` |

---

## Item Name Matching

The budget SQL matches on `lower(trim(item_name))` — case-insensitive, trims whitespace.
Use the exact item name from `BudgetItem.item_name` when adding prices via `PUT /prices`.

---

## App Setting: Cennik Sheet Name

Configurable without redeploy via `PATCH /api/settings`:
- Key: `equipment_request.cennik_sheet_name` (default: `"Cennik"`)
- Key: `equipment_request.sheet_id` — spreadsheet ID (shared with quest sync)
