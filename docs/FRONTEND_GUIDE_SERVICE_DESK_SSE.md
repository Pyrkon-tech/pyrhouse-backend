# Service Desk — Server-Sent Events (SSE)

## Endpoint

```
GET /service-desk/stream
Authorization: Bearer <jwt>
```

Requires a valid JWT (same token used for all authenticated requests).
The connection stays open indefinitely — the server pushes events as they happen.

---

## Connecting

```ts
const es = new EventSource('/service-desk/stream', {
  // EventSource doesn't support custom headers natively.
  // Use a polyfill that does, or pass the token as a query param
  // if the backend supports it (it does not yet — use a polyfill).
});

es.addEventListener('service_desk_update', (e) => {
  const event = JSON.parse(e.data);
  handleServiceDeskEvent(event);
});

es.onerror = () => {
  // The browser auto-reconnects — no manual retry needed.
};
```

> **Recommended polyfill:** [`@microsoft/fetch-event-source`](https://github.com/Azure/fetch-event-source)
> — supports custom `Authorization` header and gives full control over reconnect logic.

```ts
import { fetchEventSource } from '@microsoft/fetch-event-source';

fetchEventSource('/service-desk/stream', {
  headers: { Authorization: `Bearer ${token}` },
  onmessage(e) {
    if (e.event === 'service_desk_update') {
      handleServiceDeskEvent(JSON.parse(e.data));
    }
  },
});
```

---

## Event types

All events arrive as `event: service_desk_update`.
Discriminate on the `type` field in `data`.

### `request_created`

A new service desk request was submitted.

```json
{
  "type": "request_created",
  "request_id": 42,
  "request_type": "hardware_issue"
}
```

**Suggested action:** append to or refresh the request list.

---

### `request_updated`

Status, priority, or assignment of an existing request changed.

```json
{
  "type": "request_updated",
  "request_id": 42,
  "field": "status",
  "value": "in_progress"
}
```

Possible `field` values:

| field | value |
|---|---|
| `status` | `new` \| `in_progress` \| `waiting` \| `resolved` \| `closed` |
| `priority` | `low` \| `medium` \| `high` \| `urgent` |
| `assigned_to` | *(empty string — re-fetch the request to get the user object)* |

**Suggested action:** update the matching request in local state, or re-fetch `GET /service-desk/requests/:id`.

---

### `comment_added`

A new comment was added to a request.

```json
{
  "type": "comment_added",
  "request_id": 42
}
```

**Suggested action:** if the user is currently viewing that request, re-fetch `GET /service-desk/requests/:id/comments`.

---

## Handler pattern (TypeScript)

```ts
type ServiceDeskEventType = 'request_created' | 'request_updated' | 'comment_added';

interface ServiceDeskSSEEvent {
  type: ServiceDeskEventType;
  request_id: number;
  request_type?: string; // request_created only
  field?: string;        // request_updated only
  value?: string;        // request_updated only
}

function handleServiceDeskEvent(event: ServiceDeskSSEEvent) {
  switch (event.type) {
    case 'request_created':
      // refresh list or append
      break;
    case 'request_updated':
      // patch local state by request_id
      break;
    case 'comment_added':
      // re-fetch comments if viewing that request
      break;
  }
}
```

---

## Notes

- The SSE connection does **not** send a keepalive ping — rely on the browser's native reconnect or the polyfill's retry logic.
- Slow clients are silently skipped (the event buffer is 10 items). If the tab is in the background for a long time, a re-fetch on reconnect is safer.
- Priority changes are broadcast but `value` is not currently set for `assigned_to` — re-fetch the full request to get the user object.
