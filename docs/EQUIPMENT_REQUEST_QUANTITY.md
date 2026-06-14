# Equipment Requests — optional item quantity (frontend guide)

## Why

The source Google Sheet sometimes lists a requested item with the **quantity cell left
blank**. Such rows used to be dropped silently during sync, so the item disappeared from
dispatch. Now the item is imported with an **unknown quantity** and surfaced for a dispatcher
to fill in before a transfer can be created.

Backend representation: `equipment_request_items.quantity` is nullable. In the API a missing
quantity is serialised as `"quantity": null`.

> Scope: this only affects equipment-request items. Real stock quantities
> (`non_serialized_items`, transfer stock lines) are always concrete integers.

## Data states

| State | Condition | Meaning |
|---|---|---|
| known | `quantity` is a positive number | normal |
| **to be determined** | `quantity === null` | sheet left it blank — needs a value before dispatch |

## API contract

### List / get quest — `GET /api/equipment-requests/quests[/:id]`
Any item may have a null quantity:
```jsonc
{
  "id": "quest-…",
  "status": "pending",
  "items": [
    { "name": "Przedłużacz", "quantity": 5,    "category_id": 8 },
    { "name": "Kabel HDMI",   "quantity": null, "category_id": 22 }  // ← to be determined
  ]
}
```
A quest "needs attention" when: `quest.items.some(i => i.quantity === null)`.

### Transfer preview — `GET /api/equipment-requests/quests/:id/transfer-preview?from_location_id=<id>`
Items with no quantity come back in `unresolved_items` with a dedicated reason:
```jsonc
{
  "resolved_items":   [ { "stock_id": 1, "item_name": "Przedłużacz", "quantity": 5, "available": 80 } ],
  "unresolved_items": [
    { "item_name": "Kabel HDMI", "quantity": null, "reason": "quantity not specified in sheet" }
  ]
}
```
Other `reason` values you may also receive: `"no category match for this item"`,
`"no stock found at source location for this category"`.

### Create transfer — `POST /api/equipment-requests/quests/:id/transfer` (role: `dispatcher`)
The dispatcher supplies the concrete quantity inline via `stock_items` (already supported).
Items left auto-resolved keep their sheet quantity; items without one **must** be provided here:
```jsonc
{
  "from_location_id": 1,
  "to_location_id": 2,
  "stock_items": [ { "id": 26, "quantity": 3 } ]   // id = stock item id, quantity >= 1
}
```
Errors: `409` quest is completed/cancelled, `422` could not resolve / no stock, `404` quest not found.

## Frontend handling

1. **Detect** — item: `item.quantity === null`; quest badge: any item null → show ⚠ "brak ilości".
2. **Render** — show an amber chip `ilość: ?` instead of a number, with a tooltip explaining
   the quantity was missing in the sheet.
3. **Gate dispatch (most important)** — in the "Create transfer" modal, render every
   null-quantity item as a **required number input** (`min=1`). Keep the submit button
   **disabled** until each one has a value. Map the inputs into `stock_items: [{ id, quantity }]`.
4. **Fix-at-source path** — offer "Popraw w arkuszu" + a "Synchronizuj teraz"
   (`POST /api/equipment-requests/sync`) action; after sync the quantity is no longer null.
   Note: editing the quantity directly in our DB would be overwritten on the next sync —
   the sheet is the source of truth, so persistent fixes go in the sheet.

## States cheat-sheet

| UI | Trigger |
|---|---|
| normal number | `quantity > 0` |
| amber "ilość: ?" + required input on dispatch | `quantity === null` |
| red "brak kategorii" | preview `unresolved`, reason `no category match for this item` |
| red "brak stocku" | preview `unresolved`, reason `no stock found …` |
