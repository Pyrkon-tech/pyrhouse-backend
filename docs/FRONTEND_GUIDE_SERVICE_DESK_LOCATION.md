# Service Desk — Location Field Update

## What changed

The `POST /service-desk/requests` endpoint now accepts **two mutually exclusive ways** to specify location:

| Field | Type | When to use |
|---|---|---|
| `location` | `string \| null` | User typed a custom location not from the list |
| `location_id` | `integer \| null` | User selected a known location via autocomplete |

**You must never send both at the same time** — the API returns `400 Bad Request` if you do.

---

## Autocomplete data source

Use the existing endpoint to populate the autocomplete list:

```
GET /locations
```

Response shape (already used elsewhere in the app):
```json
[
  { "id": 1, "name": "Magazyn Techniczny", "details": "...", "pavilion": "A" },
  { "id": 3, "name": "Sala A", "details": null, "pavilion": null },
  ...
]
```

No authentication required for `GET /locations`.

---

## Request payload examples

### User selected from autocomplete (location_id)
```json
{
  "title": "Zepsuta drukarka",
  "description": "Drukarka nie drukuje",
  "type": "hardware_issue",
  "priority": "medium",
  "created_by": "Jan Kowalski",
  "location_id": 3
}
```

### User typed a custom value (location)
```json
{
  "title": "Zepsuta drukarka",
  "description": "Drukarka nie drukuje",
  "type": "hardware_issue",
  "priority": "medium",
  "created_by": "Jan Kowalski",
  "location": "Pokój obok sali B"
}
```

### No location provided — both fields omitted
```json
{
  "title": "Zepsuta drukarka",
  "description": "Drukarka nie drukuje",
  "type": "hardware_issue",
  "priority": "medium",
  "created_by": "Jan Kowalski"
}
```

---

## Response shape

Both `location` and `location_id` are returned in every request response:

```json
{
  "id": 42,
  "title": "Zepsuta drukarka",
  "location": "Sala A",
  "location_id": 3,
  ...
}
```

- When `location_id` is set → `location` contains the **name from the database** (resolved automatically, always up-to-date)
- When only free text was used → `location_id` is `null`, `location` contains the typed string

---

## Suggested UI behaviour

```
[ Location field ]
  - Input text → triggers GET /locations?... or filters local cached list
  - User picks suggestion  → store only location_id, clear location string
  - User types & blurs without picking → store only location string, clear location_id
  - Field empty → send neither field (or send both as null — API treats null as absent)
```

Tip: cache the locations list on app load — it changes rarely and has no auth requirement.

---

## Error responses

| Status | Reason |
|---|---|
| `400` | Both `location` and `location_id` provided at the same time |
| `400` | Invalid JSON / missing required fields |
| `429` | Rate limit exceeded (unauthenticated requests only) |
