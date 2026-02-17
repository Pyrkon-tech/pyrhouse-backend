# Equipment Requests - Frontend Implementation Specification

## 📋 Overview

System do zarządzania zamówieniami sprzętu zintegrowany z Google Sheets. Backend automatycznie synchronizuje dane z formularza Google → arkusz kalkulacyjny → baza danych PostgreSQL.

**Stack Backend:**
- Go 1.23 + Gin + PostgreSQL
- Auto-sync co 5-15 minut (konfigurowalny)
- Fuzzy matching kategorii (Levenshtein distance)
- Quest aggregation (grupowanie pozycji według lokalizacji/odbiorcy/daty)

**Frontend do zaimplementowania:**
- React/Vue/Angular (do wyboru)
- TypeScript (zalecane)
- UI do przeglądania i zarządzania questami

---

## 🎯 Funkcjonalności do Implementacji

### Must-Have (Priority 1)
1. ✅ **Lista questów** z filtrowaniem i paginacją
2. ✅ **Szczegóły questa** z listą pozycji
3. ✅ **Zmiana statusu** questa (pending → in_progress → completed)
4. ✅ **Manual sync trigger** (przycisk "Synchronizuj teraz")
5. ✅ **Status ostatniej synchronizacji**

### Nice-to-Have (Priority 2)
6. ⭐ **Dashboard ze statystykami** (ile pending, in_progress, completed)
7. ⭐ **Category mapping management** (ręczne dopasowanie nazw → kategorie)
8. ⭐ **Export do CSV/Excel**
9. ⭐ **Search/filtering** po odbiorcach, lokalizacjach

### Future (Priority 3)
10. 🚀 **Real-time updates** (WebSocket/SSE gdy backend będzie wspierał)
11. 🚀 **Automatic transfer creation** z questa
12. 🚀 **Budget tracking**

---

## 📡 Backend API Reference

**Base URL:** `http://localhost:8080/api`

### Authentication
Wszystkie endpointy wymagają JWT token w header:
```
Authorization: Bearer <your-jwt-token>
```

### Endpoints

#### 1. GET `/equipment-requests/quests`
Pobierz listę questów z opcjonalnym filtrowaniem i paginacją.

**Query Parameters:**
- `status` (optional): `pending` | `in_progress` | `completed` | `cancelled`
- `limit` (optional): max 500, default 100
- `offset` (optional): default 0

**Request Example:**
```bash
GET /api/equipment-requests/quests?status=pending&limit=20&offset=0
Authorization: Bearer <token>
```

**Response 200 OK:**
```json
{
  "count": 15,
  "limit": 20,
  "offset": 0,
  "quests": [
    {
      "id": "quest-f6c39c6c14716069",
      "destination": {
        "pavilion": "PCC",
        "location": "Maskarada"
      },
      "recipient": "Jan Kowalski",
      "delivery_date": "2025-06-13",
      "pickup_time": "17-18",
      "budget_owner": "Anna Nowak",
      "items": [
        {
          "name": "Laptop",
          "quantity": 2,
          "category_id": 123,
          "category_match": "exact",
          "category_match_confidence": 1.0,
          "budget_owner": "Anna Nowak",
          "notes": "Musi mieć dobrą baterię"
        }
      ],
      "status": "pending",
      "source_rows": [115, 116],
      "last_synced": "2026-02-17T12:30:00Z"
    }
  ]
}
```

---

#### 2. GET `/equipment-requests/quests/:id`
Pobierz szczegóły pojedynczego questa.

**URL Parameters:**
- `id`: Quest ID (np. `quest-f6c39c6c14716069`)

**Request Example:**
```bash
GET /api/equipment-requests/quests/quest-f6c39c6c14716069
Authorization: Bearer <token>
```

**Response 200 OK:**
```json
{
  "id": "quest-f6c39c6c14716069",
  "destination": {
    "pavilion": "PCC",
    "location": "Maskarada"
  },
  "recipient": "Jan Kowalski",
  "delivery_date": "2025-06-13",
  "pickup_time": "17-18",
  "budget_owner": "Anna Nowak",
  "items": [
    {
      "name": "Laptop Dell",
      "quantity": 2,
      "category_id": 123,
      "category_match": "fuzzy",
      "category_match_confidence": 0.85,
      "notes": "Specjalne wymagania"
    },
    {
      "name": "Mysz bezprzewodowa",
      "quantity": 2,
      "category_id": 456,
      "category_match": "exact",
      "category_match_confidence": 1.0
    }
  ],
  "status": "pending",
  "source_rows": [115, 116],
  "last_synced": "2026-02-17T12:30:00Z"
}
```

**Response 404 Not Found:**
```json
{
  "error": "Quest not found",
  "details": "no quest found with id quest-xyz"
}
```

---

#### 3. PATCH `/equipment-requests/quests/:id/status`
Zmień status questa.

**URL Parameters:**
- `id`: Quest ID

**Request Body:**
```json
{
  "status": "in_progress"
}
```

**Valid Statuses:**
- `pending` - Oczekujące
- `in_progress` - W trakcie realizacji
- `completed` - Zrealizowane
- `cancelled` - Anulowane

**Request Example:**
```bash
PATCH /api/equipment-requests/quests/quest-f6c39c6c14716069/status
Authorization: Bearer <token>
Content-Type: application/json

{
  "status": "in_progress"
}
```

**Response 200 OK:**
```json
{
  "message": "Quest status updated successfully",
  "status": "in_progress"
}
```

**Response 400 Bad Request:**
```json
{
  "error": "Invalid status",
  "details": "Status must be one of: pending, in_progress, completed, cancelled"
}
```

---

#### 4. POST `/equipment-requests/sync`
Ręcznie wywołaj synchronizację z Google Sheets.

**Request Example:**
```bash
POST /api/equipment-requests/sync
Authorization: Bearer <token>
```

**Response 200 OK:**
```json
{
  "message": "Sync completed successfully",
  "stats": {
    "quests_created": 5,
    "quests_updated": 3,
    "quests_unchanged": 12,
    "items_added": 8,
    "items_removed": 2
  },
  "quests": [
    {
      "id": "quest-abc123",
      "destination": {...},
      "status": "pending"
    }
  ]
}
```

**Response 500 Internal Server Error:**
```json
{
  "error": "Failed to sync equipment requests",
  "details": "unable to fetch spreadsheet data: ..."
}
```

---

#### 5. GET `/equipment-requests/sync-log`
Pobierz informacje o ostatniej synchronizacji.

**Request Example:**
```bash
GET /api/equipment-requests/sync-log
Authorization: Bearer <token>
```

**Response 200 OK:**
```json
{
  "id": 42,
  "synced_at": "2026-02-17T14:30:00Z",
  "rows_processed": 20,
  "quests_created": 5,
  "quests_updated": 3,
  "quests_unchanged": 12,
  "items_added": 8,
  "items_removed": 2,
  "success": true,
  "duration_ms": 2350,
  "sheet_id": "16BytrbWmyWeBGnlSIDZn1Lnb5rdspoQu_rpc5m5Vtbc",
  "errors": ""
}
```

**Response 404 Not Found:**
```json
{
  "error": "No sync log found",
  "details": "no sync operations recorded yet"
}
```

---

#### 6. POST `/equipment-requests/category-mapping`
Stwórz ręczne mapowanie nazwy z formularza → kategoria.

**Request Body:**
```json
{
  "form_item_name": "Laptop Dell",
  "category_id": 123,
  "created_by": 7
}
```

**Request Example:**
```bash
POST /api/equipment-requests/category-mapping
Authorization: Bearer <token>
Content-Type: application/json

{
  "form_item_name": "Laptop Dell XPS",
  "category_id": 123
}
```

**Response 201 Created:**
```json
{
  "message": "Category mapping created successfully",
  "mapping": {
    "id": 15,
    "form_item_name": "Laptop Dell XPS",
    "category_id": 123,
    "created_by": null,
    "created_at": "2026-02-17T15:00:00Z"
  }
}
```

---

## 📊 TypeScript Type Definitions

```typescript
// Quest Types
export interface Quest {
  id: string;
  destination: Destination;
  recipient: string;
  delivery_date: string; // ISO date format: "2025-06-13"
  pickup_time?: string;
  budget_owner: string;
  items: QuestItem[];
  status: QuestStatus;
  source_rows: number[];
  last_synced: string; // ISO datetime: "2026-02-17T12:30:00Z"
}

export interface Destination {
  pavilion: string;
  location: string;
}

export interface QuestItem {
  name: string;
  quantity: number;
  category_id?: number;
  category_match: CategoryMatchType;
  category_match_confidence?: number; // 0.0 - 1.0
  budget_owner?: string;
  notes?: string;
}

export type QuestStatus = 'pending' | 'in_progress' | 'completed' | 'cancelled';
export type CategoryMatchType = 'exact' | 'fuzzy' | 'manual' | 'none';

// API Response Types
export interface QuestsListResponse {
  count: number;
  limit: number;
  offset: number;
  quests: Quest[];
}

export interface SyncResponse {
  message: string;
  stats: SyncStats;
  quests: Quest[];
}

export interface SyncStats {
  quests_created: number;
  quests_updated: number;
  quests_unchanged: number;
  items_added: number;
  items_removed: number;
}

export interface SyncLog {
  id: number;
  synced_at: string;
  rows_processed: number;
  quests_created: number;
  quests_updated: number;
  quests_unchanged: number;
  items_added: number;
  items_removed: number;
  success: boolean;
  duration_ms: number;
  sheet_id: string;
  errors: string;
}

export interface CategoryMapping {
  id: number;
  form_item_name: string;
  category_id: number;
  created_by?: number;
  created_at: string;
}

export interface StatusUpdateRequest {
  status: QuestStatus;
}

export interface CategoryMappingRequest {
  form_item_name: string;
  category_id: number;
  created_by?: number;
}

// Error Response
export interface ApiError {
  error: string;
  details?: string;
}
```

---

## 🎨 UI/UX Requirements

### 1. Quest List View

**Layout:**
```
┌─────────────────────────────────────────────────────────────┐
│ Equipment Requests                    [Synchronize Now] [+] │
├─────────────────────────────────────────────────────────────┤
│ Filters: [All ▼] [Status: All ▼] [Search: ________]         │
├─────────────────────────────────────────────────────────────┤
│ Last sync: 2 minutes ago (5 created, 3 updated, 12 unchanged)│
├─────────────────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ 📦 PCC - Maskarada                    [PENDING]         │ │
│ │ Recipient: Jan Kowalski                                 │ │
│ │ Delivery: 2025-06-13 | Pickup: 17-18                   │ │
│ │ Items: Laptop (2), Mouse (2)                           │ │
│ │ Budget: Anna Nowak                                      │ │
│ └─────────────────────────────────────────────────────────┘ │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ 📦 Pawilon 5 - POW               [IN_PROGRESS]         │ │
│ │ Recipient: Anna Nowak                                   │ │
│ │ Delivery: 2025-06-14                                    │ │
│ │ Items: Monitor (1), Keyboard (1), Mouse (1)            │ │
│ └─────────────────────────────────────────────────────────┘ │
│                                                             │
│                  [< Previous] Page 1 of 3 [Next >]          │
└─────────────────────────────────────────────────────────────┘
```

**Features:**
- ✅ Card-based layout dla questów
- ✅ Color-coded status badges:
  - `pending`: 🟡 Yellow
  - `in_progress`: 🔵 Blue
  - `completed`: 🟢 Green
  - `cancelled`: 🔴 Red
- ✅ Filter by status dropdown
- ✅ Search by recipient/location
- ✅ Pagination controls
- ✅ Last sync info + stats
- ✅ "Synchronize Now" button with loading state

---

### 2. Quest Detail View

**Layout:**
```
┌─────────────────────────────────────────────────────────────┐
│ [← Back to List]     Quest #quest-f6c39c6c14716069          │
├─────────────────────────────────────────────────────────────┤
│ Status: [PENDING ▼]                    [Change Status]      │
├─────────────────────────────────────────────────────────────┤
│ 📍 Destination                                              │
│   Pavilion: PCC                                             │
│   Location: Maskarada                                       │
│                                                             │
│ 👤 Recipient                                                │
│   Jan Kowalski                                              │
│                                                             │
│ 📅 Delivery Details                                         │
│   Date: 2025-06-13                                          │
│   Pickup Time: 17-18                                        │
│                                                             │
│ 💰 Budget Owner                                             │
│   Anna Nowak                                                │
├─────────────────────────────────────────────────────────────┤
│ 📦 Items (2)                                                │
├─────────────────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ Laptop Dell                                      Qty: 2 │ │
│ │ Category: 💻 Electronics (fuzzy match, 85%)            │ │
│ │ Budget: Anna Nowak                                      │ │
│ │ Notes: Musi mieć dobrą baterię                         │ │
│ └─────────────────────────────────────────────────────────┘ │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ Mysz bezprzewodowa                               Qty: 2 │ │
│ │ Category: 🖱️ Accessories (exact match, 100%)          │ │
│ └─────────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────┤
│ 🔍 Metadata                                                 │
│   Source Rows: 115, 116                                     │
│   Last Synced: 2026-02-17 12:30:00                         │
└─────────────────────────────────────────────────────────────┘
```

**Features:**
- ✅ Status dropdown with inline update
- ✅ Structured info display
- ✅ Items list with:
  - Category match indicator (exact/fuzzy/manual/none)
  - Confidence percentage for fuzzy matches
  - Color coding based on match quality
- ✅ Metadata section (debug info)

---

### 3. Dashboard (Optional - Priority 2)

**Layout:**
```
┌─────────────────────────────────────────────────────────────┐
│ Equipment Requests Dashboard                                │
├─────────────────────────────────────────────────────────────┤
│ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐      │
│ │  PENDING │ │   IN     │ │COMPLETED │ │CANCELLED │      │
│ │    15    │ │ PROGRESS │ │    45    │ │    3     │      │
│ │          │ │    8     │ │          │ │          │      │
│ └──────────┘ └──────────┘ └──────────┘ └──────────┘      │
├─────────────────────────────────────────────────────────────┤
│ 📊 Last 7 Days Activity                                     │
│ [Chart showing quests created/completed over time]          │
├─────────────────────────────────────────────────────────────┤
│ 🔥 Most Requested Items                                     │
│  1. Laptop (24 requests)                                    │
│  2. Mouse (18 requests)                                     │
│  3. Monitor (12 requests)                                   │
└─────────────────────────────────────────────────────────────┘
```

---

## 🔧 Implementation Guide

### Step 1: API Client Setup

```typescript
// src/api/equipmentRequests.ts
import axios from 'axios';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080/api';

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Add JWT token to requests
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('auth_token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

export const equipmentRequestsAPI = {
  // Get quests list
  getQuests: async (params?: {
    status?: QuestStatus;
    limit?: number;
    offset?: number;
  }): Promise<QuestsListResponse> => {
    const response = await api.get('/equipment-requests/quests', { params });
    return response.data;
  },

  // Get single quest
  getQuest: async (id: string): Promise<Quest> => {
    const response = await api.get(`/equipment-requests/quests/${id}`);
    return response.data;
  },

  // Update quest status
  updateQuestStatus: async (
    id: string,
    status: QuestStatus
  ): Promise<{ message: string; status: string }> => {
    const response = await api.patch(
      `/equipment-requests/quests/${id}/status`,
      { status }
    );
    return response.data;
  },

  // Trigger manual sync
  triggerSync: async (): Promise<SyncResponse> => {
    const response = await api.post('/equipment-requests/sync');
    return response.data;
  },

  // Get sync log
  getSyncLog: async (): Promise<SyncLog> => {
    const response = await api.get('/equipment-requests/sync-log');
    return response.data;
  },

  // Create category mapping
  createCategoryMapping: async (
    mapping: CategoryMappingRequest
  ): Promise<{ message: string; mapping: CategoryMapping }> => {
    const response = await api.post(
      '/equipment-requests/category-mapping',
      mapping
    );
    return response.data;
  },
};
```

---

### Step 2: React Components Structure (Example)

```
src/
├── features/
│   └── equipment-requests/
│       ├── components/
│       │   ├── QuestList.tsx
│       │   ├── QuestCard.tsx
│       │   ├── QuestDetail.tsx
│       │   ├── QuestFilters.tsx
│       │   ├── SyncButton.tsx
│       │   ├── SyncStatus.tsx
│       │   └── StatusBadge.tsx
│       ├── hooks/
│       │   ├── useQuests.ts
│       │   ├── useQuestDetail.ts
│       │   └── useSync.ts
│       └── types.ts
├── api/
│   └── equipmentRequests.ts
└── App.tsx
```

---

### Step 3: Example React Hook

```typescript
// src/features/equipment-requests/hooks/useQuests.ts
import { useState, useEffect } from 'react';
import { equipmentRequestsAPI } from '@/api/equipmentRequests';
import type { Quest, QuestStatus } from '../types';

export const useQuests = () => {
  const [quests, setQuests] = useState<Quest[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [filters, setFilters] = useState({
    status: undefined as QuestStatus | undefined,
    limit: 20,
    offset: 0,
  });

  const fetchQuests = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await equipmentRequestsAPI.getQuests(filters);
      setQuests(response.quests);
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to fetch quests');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchQuests();
  }, [filters]);

  const updateStatus = async (id: string, status: QuestStatus) => {
    try {
      await equipmentRequestsAPI.updateQuestStatus(id, status);
      // Refresh list
      await fetchQuests();
    } catch (err: any) {
      throw new Error(err.response?.data?.error || 'Failed to update status');
    }
  };

  return {
    quests,
    loading,
    error,
    filters,
    setFilters,
    updateStatus,
    refresh: fetchQuests,
  };
};
```

---

### Step 4: Example Component

```typescript
// src/features/equipment-requests/components/QuestList.tsx
import React from 'react';
import { useQuests } from '../hooks/useQuests';
import { QuestCard } from './QuestCard';
import { QuestFilters } from './QuestFilters';
import { SyncButton } from './SyncButton';

export const QuestList: React.FC = () => {
  const { quests, loading, error, filters, setFilters, refresh } = useQuests();

  if (loading) return <div>Loading...</div>;
  if (error) return <div>Error: {error}</div>;

  return (
    <div className="quest-list">
      <header>
        <h1>Equipment Requests</h1>
        <SyncButton onSync={refresh} />
      </header>

      <QuestFilters
        currentFilters={filters}
        onChange={setFilters}
      />

      <div className="quest-grid">
        {quests.map((quest) => (
          <QuestCard key={quest.id} quest={quest} />
        ))}
      </div>

      {/* Pagination */}
      <div className="pagination">
        <button
          disabled={filters.offset === 0}
          onClick={() => setFilters({ ...filters, offset: filters.offset - filters.limit })}
        >
          Previous
        </button>
        <button
          onClick={() => setFilters({ ...filters, offset: filters.offset + filters.limit })}
        >
          Next
        </button>
      </div>
    </div>
  );
};
```

---

## 🎨 Design System Recommendations

### Colors (Status-based)
```css
:root {
  --status-pending: #FCD34D;      /* Yellow 300 */
  --status-in-progress: #60A5FA;  /* Blue 400 */
  --status-completed: #34D399;    /* Green 400 */
  --status-cancelled: #F87171;    /* Red 400 */

  --match-exact: #10B981;         /* Green 500 */
  --match-fuzzy: #F59E0B;         /* Amber 500 */
  --match-manual: #8B5CF6;        /* Purple 500 */
  --match-none: #6B7280;          /* Gray 500 */
}
```

### Typography
- Headers: Inter/Roboto Bold
- Body: Inter/Roboto Regular
- Monospace (IDs): JetBrains Mono / Fira Code

### Icons (Recommended: Heroicons / Lucide)
- 📦 Box: Quest card
- 📍 Map Pin: Location
- 👤 User: Recipient
- 📅 Calendar: Date
- 💰 Currency: Budget
- 🔄 Refresh: Sync button
- ✅ Check: Completed
- ⏸️ Pause: In Progress
- ⏳ Clock: Pending

---

## ✅ Testing Checklist

### Manual Testing
- [ ] Lista questów ładuje się poprawnie
- [ ] Filtrowanie po statusie działa
- [ ] Paginacja działa (next/previous)
- [ ] Kliknięcie w quest otwiera szczegóły
- [ ] Zmiana statusu questa działa i odświeża listę
- [ ] Manual sync button wywołuje synchronizację
- [ ] Sync status pokazuje ostatnią synchronizację
- [ ] Error handling - nieprawidłowy quest ID
- [ ] Error handling - błąd sieci
- [ ] Loading states działają

### Edge Cases
- [ ] Pusta lista questów (brak danych)
- [ ] Quest bez kategorii (category_match = "none")
- [ ] Quest z fuzzy match (pokazuje confidence)
- [ ] Quest z bardzo długimi nazwami pozycji
- [ ] Dużo pozycji w queście (>10)
- [ ] Token wygasł (401 Unauthorized)

---

## 🚀 Deployment Notes

### Environment Variables
```bash
# .env.production
REACT_APP_API_URL=https://api.yourcompany.com/api
REACT_APP_AUTO_REFRESH_INTERVAL=300000  # 5 minutes
```

### CORS Configuration
Backend musi mieć skonfigurowane CORS dla frontendu:
```go
// Sprawdź .env na backendzie:
CORS_ALLOWED_ORIGINS=http://localhost:3000,https://yourapp.com
```

---

## 📚 Resources

- **OpenAPI Spec:** `/docs/openapi.yaml`
- **Backend Repo:** [link]
- **Design Mockups:** [Figma/link]
- **API Testing:** Use Postman collection (export z openapi.yaml)

---

## 🐛 Known Issues & Limitations

1. **Auto-refresh:** Frontend nie ma real-time updates - użyj polling lub manual refresh
2. **Large datasets:** Paginacja max 500 questów per page
3. **Category matching:** Confidence score nie zawsze odzwierciedla jakość - może być false positive
4. **Sync timing:** Auto-sync jest asynchroniczny - może trwać kilka sekund

---

## 📞 Support

Pytania? Problemy?
- Backend docs: `README.md`, `AGENTS.md`
- API spec: `docs/openapi.yaml`
- Create issue w repo

---

**Good luck! 🚀**
