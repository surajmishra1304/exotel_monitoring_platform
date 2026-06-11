# Exotel Monitoring Platform — Frontend Requirements

**Version:** 1.0  
**Status:** Draft  
**Aligned with backend API version:** 1.0

---

## 1. Executive Summary

The frontend is a single-page React dashboard that consumes all backend APIs of the Exotel Monitoring Platform. It provides operations teams, support engineers, and management with centralised visibility into Exophone health, call analytics, active alerts, and monitoring transaction history.

All data is served from the backend's Redis + snapshot cache layer, so the UI must not implement its own heavy data-fetching logic — lightweight polling and on-demand queries are sufficient.

---

## 2. Tech Stack

| Layer | Library / Tool | Reason |
|---|---|---|
| Framework | React 18 + Vite | Fast HMR, modern bundler |
| Language | TypeScript | Type safety across API contracts |
| Router | React Router v6 | Declarative routing |
| State | TanStack Query v5 | Server-state, cache, polling, refetch |
| UI Components | Ant Design 5 | Rich table/form/alert components |
| Charts | Recharts | Composable React charts |
| HTTP Client | Axios | Interceptors, typed responses |
| Styling | Tailwind CSS + Ant Design tokens | Utility-first + consistent theming |
| Linting | ESLint + Prettier | Code quality |
| Build | Vite | Production bundles |

---

## 3. Project Structure

```
exotel-monitoring-frontend/
├── public/
├── src/
│   ├── api/                  ← Axios client + one file per API group
│   │   ├── client.ts         ← Base Axios instance, base URL, interceptors
│   │   ├── dashboard.ts      ← /api/v1/dashboard/*
│   │   ├── exophones.ts      ← /api/v1/accounts/*/exophones, /exophones/*
│   │   ├── alerts.ts         ← /api/v1/alerts/*
│   │   ├── transactions.ts   ← /api/v1/transactions
│   │   └── health.ts         ← /health
│   │
│   ├── types/                ← TypeScript interfaces matching backend response shapes
│   │   ├── account.ts
│   │   ├── exophone.ts
│   │   ├── alert.ts
│   │   ├── transaction.ts
│   │   ├── metric.ts
│   │   └── snapshot.ts
│   │
│   ├── hooks/                ← TanStack Query hooks, one per API call
│   │   ├── useDashboardSummary.ts
│   │   ├── useExophoneMetrics.ts
│   │   ├── useExophoneHealth.ts
│   │   ├── useActiveAlerts.ts
│   │   ├── useTransactions.ts
│   │   └── useSystemHealth.ts
│   │
│   ├── pages/                ← One folder per route
│   │   ├── Dashboard/
│   │   ├── Exophones/
│   │   ├── ExophoneDetail/
│   │   ├── Alerts/
│   │   ├── Transactions/
│   │   └── SystemHealth/
│   │
│   ├── components/           ← Shared, reusable UI components
│   │   ├── layout/
│   │   │   ├── AppShell.tsx  ← Sidebar + top nav wrapper
│   │   │   └── PageHeader.tsx
│   │   ├── charts/
│   │   │   ├── CallStatusDonut.tsx
│   │   │   ├── StreamUtilizationBar.tsx
│   │   │   ├── AvailabilityLine.tsx
│   │   │   └── LatencyTrend.tsx
│   │   ├── cards/
│   │   │   ├── KpiCard.tsx
│   │   │   ├── HeartbeatBadge.tsx
│   │   │   └── AlertBadge.tsx
│   │   └── common/
│   │       ├── StatusTag.tsx
│   │       ├── SeverityTag.tsx
│   │       ├── LoadingSpinner.tsx
│   │       ├── ErrorBoundary.tsx
│   │       └── EmptyState.tsx
│   │
│   ├── utils/
│   │   ├── formatters.ts     ← Date, duration, percent formatters
│   │   └── constants.ts      ← Priority labels, severity colours
│   │
│   ├── App.tsx
│   ├── main.tsx
│   └── vite-env.d.ts
│
├── .env.local                ← VITE_API_BASE_URL=http://localhost:8080
├── package.json
├── tsconfig.json
├── tailwind.config.ts
└── vite.config.ts
```

---

## 4. Environment Configuration

```env
# .env.local (development)
VITE_API_BASE_URL=http://localhost:8080

# .env.production
VITE_API_BASE_URL=https://your-monitoring-service.internal
```

---

## 5. Axios Base Client

**File:** `src/api/client.ts`

```typescript
import axios from 'axios';

const client = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL,
  timeout: 15000,
  headers: { 'Content-Type': 'application/json' },
});

// Request interceptor — attach X-User header for audit trail
client.interceptors.request.use((config) => {
  const user = localStorage.getItem('ops_user') ?? 'OPS_CONSOLE';
  config.headers['X-User'] = user;
  return config;
});

// Response interceptor — normalise errors
client.interceptors.response.use(
  (res) => res,
  (err) => {
    const message = err.response?.data?.error ?? err.message;
    return Promise.reject(new Error(message));
  }
);

export default client;
```

---

## 6. TypeScript Types

**File:** `src/types/snapshot.ts`

```typescript
export interface CallMetricsSnapshot {
  id: number;
  snapshot_date: string;
  account_id: number;
  exophone_id: number;
  total_calls: number;
  failed_calls: number;
  connected_calls: number;
  busy_calls: number;
  avg_duration_sec: number;
  success_rate: number;
}

export interface ExophoneHealthSnapshot {
  id: number;
  exophone_id: number;
  account_id: number;
  heartbeat_status: 'OK' | 'DEGRADED' | 'OUTAGE' | 'UNKNOWN';
  availability_percent: number;
  last_api_latency_ms: number | null;
  active_streams: number;
  last_checked_at: string | null;
}

export interface AccountDashboardSnapshot {
  account_id: number;
  account_name: string;
  total_exophones: number;
  active_exophones: number;
  total_calls: number;
  failed_calls: number;
  success_rate: number;
  avg_latency_ms: number | null;
  alert_count: number;
  snapshot_date: string;
}
```

**File:** `src/types/alert.ts`

```typescript
export type AlertType =
  | 'API_FAILURE'
  | 'HEARTBEAT_FAILURE'
  | 'HIGH_LATENCY'
  | 'RETRY_EXHAUSTED'
  | 'VENDOR_THROTTLING'
  | 'EXOPHONE_DOWN'
  | 'CACHE_FAILURE';

export type Severity = 'CRITICAL' | 'WARNING' | 'INFO';
export type AlertStatus = 'OPEN' | 'ACKNOWLEDGED' | 'RESOLVED';
export type AlertChannel = 'SLACK' | 'EMAIL' | 'WEBHOOK';

export interface Alert {
  id: number;
  transaction_id: string;
  exophone_id: number;
  account_id: number;
  alert_type: AlertType;
  severity: Severity;
  alert_status: AlertStatus;
  message: string;
  channel: AlertChannel;
  triggered_at: string;
  acknowledged_at: string | null;
}
```

**File:** `src/types/transaction.ts`

```typescript
export type TxnStatus = 'RUNNING' | 'SUCCESS' | 'FAILED' | 'RETRYING' | 'TIMEOUT';

export interface JobTransaction {
  id: number;
  transaction_id: string;
  job_id: number;
  account_id: number;
  exophone_id: number;
  job_type: 'HEARTBEAT' | 'CALLS' | 'STREAMS';
  status: TxnStatus;
  api_latency_ms: number | null;
  cache_hit: number;
  retry_count: number;
  error_message: string;
  started_at: string;
  completed_at: string | null;
}
```

---

## 7. API Layer

### 7.1 dashboard.ts

```typescript
// GET /api/v1/dashboard/summary
export const getDashboardSummary = () =>
  client.get<{ source: string; data: AccountDashboardSnapshot[] }>('/api/v1/dashboard/summary');

// GET /api/v1/dashboard/exophones/:id/calls
export const getExophoneCallAnalytics = (exophoneId: number) =>
  client.get<{ source: string; data: CallMetricsSnapshot }>(`/api/v1/dashboard/exophones/${exophoneId}/calls`);
```

### 7.2 exophones.ts

```typescript
// GET /api/v1/accounts/:account_id/exophones
export const getExophones = (accountId: number) =>
  client.get<{ data: Exophone[] }>(`/api/v1/accounts/${accountId}/exophones`);

// GET /api/v1/accounts/:account_id/exophones/health
export const getExophoneHealthList = (accountId: number) =>
  client.get<{ data: ExophoneHealthSnapshot[] }>(`/api/v1/accounts/${accountId}/exophones/health`);

// GET /api/v1/exophones/:id/metrics
export const getExophoneMetrics = (id: number) =>
  client.get<{ source: string; data: ExophoneHealthSnapshot }>(`/api/v1/exophones/${id}/metrics`);
```

### 7.3 alerts.ts

```typescript
// GET /api/v1/alerts/active
export const getActiveAlerts = () =>
  client.get<{ source: string; data: Alert[] }>('/api/v1/alerts/active');

// POST /api/v1/alerts
export const createAlert = (payload: Omit<Alert, 'id' | 'triggered_at' | 'acknowledged_at'>) =>
  client.post<{ status: string }>('/api/v1/alerts', payload);

// PUT /api/v1/alerts/:id/acknowledge
export const acknowledgeAlert = (alertId: number) =>
  client.put<{ status: string }>(`/api/v1/alerts/${alertId}/acknowledge`);
```

### 7.4 transactions.ts

```typescript
export interface TransactionFilters {
  exophone_id?: number;
  status?: TxnStatus;
  from?: string;   // ISO 8601
  to?: string;
  limit?: number;
}

// GET /api/v1/transactions
export const getTransactions = (filters: TransactionFilters = {}) =>
  client.get<{ data: JobTransaction[]; count: number }>('/api/v1/transactions', { params: filters });
```

### 7.5 health.ts

```typescript
export interface HealthResponse {
  status: 'UP' | 'DEGRADED';
  mysql: 'CONNECTED' | 'DISCONNECTED';
  redis: 'CONNECTED' | 'DISCONNECTED';
}

// GET /health
export const getSystemHealth = () =>
  client.get<HealthResponse>('/health');
```

---

## 8. TanStack Query Hooks

```typescript
// src/hooks/useDashboardSummary.ts
export const useDashboardSummary = () =>
  useQuery({
    queryKey: ['dashboard', 'summary'],
    queryFn: () => getDashboardSummary().then(r => r.data.data),
    refetchInterval: 60_000,   // refresh every 60 s
    staleTime: 30_000,
  });

// src/hooks/useActiveAlerts.ts
export const useActiveAlerts = () =>
  useQuery({
    queryKey: ['alerts', 'active'],
    queryFn: () => getActiveAlerts().then(r => r.data.data),
    refetchInterval: 30_000,   // alerts poll every 30 s
    staleTime: 10_000,
  });

// src/hooks/useTransactions.ts
export const useTransactions = (filters: TransactionFilters) =>
  useQuery({
    queryKey: ['transactions', filters],
    queryFn: () => getTransactions(filters).then(r => r.data),
    keepPreviousData: true,
  });

// src/hooks/useAcknowledgeAlert.ts
export const useAcknowledgeAlert = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (alertId: number) => acknowledgeAlert(alertId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['alerts', 'active'] }),
  });
};

// src/hooks/useSystemHealth.ts
export const useSystemHealth = () =>
  useQuery({
    queryKey: ['system', 'health'],
    queryFn: () => getSystemHealth().then(r => r.data),
    refetchInterval: 15_000,
  });
```

---

## 9. Pages & Routes

### Route Map

```
/                        → redirect to /dashboard
/dashboard               → Dashboard (global KPI summary)
/exophones               → Exophone List & Health Grid
/exophones/:id           → Exophone Detail (metrics + call analytics)
/alerts                  → Alerts Center
/transactions            → Transaction Explorer
/health                  → System Health
```

---

## 10. Page Specifications

---

### 10.1 Dashboard — `/dashboard`

**Purpose:** Global, at-a-glance view of all accounts. First screen ops team lands on.

**APIs consumed:**
- `GET /api/v1/dashboard/summary` → KPI cards, account table
- `GET /api/v1/alerts/active` → Alert count badge in top nav
- `GET /health` → Infrastructure status banner

**Layout:**

```
┌─────────────────────────────────────────────────────────┐
│  NAVBAR: "Exotel Monitor"    [Alerts 3🔴]  [System: UP] │
├──────────┬──────────────────────────────────────────────┤
│          │  Page: Dashboard                              │
│ SIDEBAR  ├──────────────────────────────────────────────┤
│          │  KPI CARDS ROW                                │
│ Dashboard│  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐        │
│ Exophones│  │Total │ │Failed│ │Succ% │ │Active│        │
│ Alerts   │  │Calls │ │Calls │ │Rate  │ │Alerts│        │
│ TXNs     │  └──────┘ └──────┘ └──────┘ └──────┘        │
│ Health   │                                               │
│          │  ACCOUNTS TABLE                               │
│          │  ┌──────────────────────────────────────────┐ │
│          │  │Account│Exophones│Calls│Failed│SuccRate│  │ │
│          │  │──────────────────────────────────────────│ │
│          │  │Demo   │ 5 / 5  │ 120 │  3   │ 97.5%  │  │ │
│          │  └──────────────────────────────────────────┘ │
└──────────┴──────────────────────────────────────────────┘
```

**Components:**
- `KpiCard` × 4 — Total Calls, Failed Calls, Global Success Rate, Open Alert Count
- `AccountSummaryTable` — one row per account from `/dashboard/summary`, clickable rows navigate to `/exophones?account_id=X`
- `AlertCountBadge` in navbar from `/alerts/active`
- `SystemHealthBanner` — green/yellow/red strip at top when degraded, from `/health`

**Polling:** Dashboard summary refreshes every 60 s automatically via TanStack Query `refetchInterval`.

**Data source label:** Show `source: "cache"` or `source: "snapshot"` as a small footnote (useful for ops debugging).

---

### 10.2 Exophone List — `/exophones`

**Purpose:** Inventory view of all Exophones for a selected account, with live health grid.

**APIs consumed:**
- `GET /api/v1/accounts/:account_id/exophones` → inventory list
- `GET /api/v1/accounts/:account_id/exophones/health` → health snapshot grid

**Filters (URL query params):**
- `account_id` — pre-selected account (defaults to first account from summary)
- `status` — active / inactive / all
- `priority` — P0 / P1 / P2 / P3 / all

**Layout:**

```
┌───────────────────────────────────────────────────────────┐
│  Exophones                    [Account ▾] [Status ▾] [P▾]  │
├───────────────────────────────────────────────────────────┤
│  HEALTH GRID (card per exophone)                          │
│  ┌────────────────┐  ┌────────────────┐                   │
│  │ +911234567890  │  │ +919876543210  │                   │
│  │ ● OK    P0     │  │ ⚠ DEGRADED P1  │                   │
│  │ Avail: 100%    │  │ Avail: 50%     │                   │
│  │ Streams: 3/10  │  │ Latency: 4.2s  │                   │
│  │ [View Detail]  │  │ [View Detail]  │                   │
│  └────────────────┘  └────────────────┘                   │
├───────────────────────────────────────────────────────────┤
│  EXOPHONE TABLE                                           │
│  Number | Status | Priority | Last Synced | Monitoring    │
│  ────────────────────────────────────────────────────────  │
│  +91123… │ active │ P0      │ 2m ago      │ ✓             │
└───────────────────────────────────────────────────────────┘
```

**Components:**
- `AccountSelector` — dropdown populated from dashboard summary data (no extra API call)
- `ExophoneHealthCard` × N — one card per exophone showing heartbeat badge, availability %, active streams, last latency
- `HeartbeatBadge` — colour-coded: OK = green, DEGRADED = amber, OUTAGE = red, UNKNOWN = grey
- `ExophoneTable` — sortable by priority, last_synced_at, status; each row links to `/exophones/:id`

**Polling:** Health grid refreshes every 30 s.

---

### 10.3 Exophone Detail — `/exophones/:id`

**Purpose:** Deep-dive into a single Exophone's health snapshot, call metrics, and chart history.

**APIs consumed:**
- `GET /api/v1/exophones/:id/metrics` → health snapshot KPI cards
- `GET /api/v1/dashboard/exophones/:id/calls` → call analytics + donut chart
- `GET /api/v1/alerts/active` → filtered to this exophone_id, shown in sidebar panel

**Layout:**

```
┌──────────────────────────────────────────────────────────────┐
│  ← Back   Exophone: +911234567890   [P0]  [● OK]            │
├─────────────────────────────────┬────────────────────────────┤
│  KPI CARDS                      │  ACTIVE ALERTS             │
│  ┌──────┐┌──────┐┌──────┐      │  ┌──────────────────────┐  │
│  │Avail.││Latency││Streams│     │  │ ⚠ HIGH_LATENCY       │  │
│  │100%  ││220ms  ││ 5/20  │     │  │   triggered 5m ago   │  │
│  └──────┘└──────┘└──────┘      │  │  [Acknowledge]        │  │
│                                 │  └──────────────────────┘  │
│  CALL ANALYTICS (today)         │                            │
│  ┌────────────────────────────┐ │                            │
│  │ DONUT CHART                │ │                            │
│  │  Completed: 82%            │ │                            │
│  │  Failed:    12%            │ │                            │
│  │  Busy:       6%            │ │                            │
│  └────────────────────────────┘ │                            │
│                                 │                            │
│  CALL STATS TABLE               │                            │
│  Total | Connected | Failed | Avg Duration | Success Rate   │
│  120   │ 98        │ 15     │ 121s         │ 81.6%          │
└─────────────────────────────────┴────────────────────────────┘
```

**Components:**
- `KpiCard` × 3 — Availability %, API Latency ms, Active Streams (active / max)
- `CallStatusDonut` — Recharts PieChart with three segments: completed / failed / busy
- `CallStatsTable` — single-row summary from snapshot data
- `ActiveAlertPanel` — filtered list of `alerts/active` for this exophone, each with Acknowledge button
- `HeartbeatBadge` — in page header

**Actions:**
- Acknowledge Alert → `PUT /api/v1/alerts/:id/acknowledge`, invalidates alerts cache

---

### 10.4 Alerts Center — `/alerts`

**Purpose:** Operations alert triage center. All open alerts with severity filtering and bulk acknowledge.

**APIs consumed:**
- `GET /api/v1/alerts/active` → alert list
- `PUT /api/v1/alerts/:id/acknowledge` → acknowledge action

**Layout:**

```
┌──────────────────────────────────────────────────────────────┐
│  Alerts Center                [Severity ▾] [Type ▾] [Refresh]│
├──────────────────────────────────────────────────────────────┤
│  SUMMARY CHIPS                                               │
│  [🔴 CRITICAL: 2]  [🟡 WARNING: 5]  [🔵 INFO: 1]           │
├──────────────────────────────────────────────────────────────┤
│  ALERTS TABLE                                                │
│  Severity │ Type              │ Exophone    │ Message │ When  │ Action │
│  ─────────────────────────────────────────────────────────── │
│  CRITICAL │ HEARTBEAT_FAILURE │ +91123…     │ Non-OK  │ 3m    │ [Ack]  │
│  WARNING  │ HIGH_LATENCY      │ +91987…     │ 4.2s    │ 10m   │ [Ack]  │
└──────────────────────────────────────────────────────────────┘
```

**Components:**
- `SeverityChip` × 3 — count pills for CRITICAL / WARNING / INFO, act as filters
- `AlertTypeFilter` — dropdown: All / API_FAILURE / HEARTBEAT_FAILURE / HIGH_LATENCY / RETRY_EXHAUSTED / VENDOR_THROTTLING / EXOPHONE_DOWN / CACHE_FAILURE
- `AlertsTable` — sortable by severity then triggered_at, with Acknowledge button per row
- `SeverityTag` — colour-coded tag component
- `AlertBadge` — inline severity indicator

**Filtering (client-side):** Filter by severity and alert_type from the already-fetched active alerts list. No extra API call needed.

**Actions:**
- Single Acknowledge → `PUT /api/v1/alerts/:id/acknowledge`
- After acknowledge: row disappears from table (refetch), show success toast

**Polling:** Refetch every 30 s.

---

### 10.5 Transaction Explorer — `/transactions`

**Purpose:** Full audit and debugging view of all monitoring job executions.

**APIs consumed:**
- `GET /api/v1/transactions` with query params: `exophone_id`, `status`, `from`, `to`, `limit`

**Layout:**

```
┌──────────────────────────────────────────────────────────────┐
│  Transactions     [Exophone ▾] [Status ▾] [From──To] [Search]│
├──────────────────────────────────────────────────────────────┤
│  TRANSACTIONS TABLE                                          │
│  TXN ID          │ Type      │ Status  │ Exophone │Latency│Retries│ Started │
│  ───────────────────────────────────────────────────────────  │
│  TXN-20260527-.. │ HEARTBEAT │ SUCCESS │ +91123… │ 220ms │  0   │ 2m ago  │
│  TXN-20260527-.. │ CALLS     │ FAILED  │ +91987… │  —    │  3   │ 5m ago  │
├──────────────────────────────────────────────────────────────┤
│  [Selected row expanded]                                     │
│  Transaction ID: TXN-20260527-100001                         │
│  Job ID: 3 | Account: Demo | Exophone: +919876543210         │
│  Status: FAILED | Error: "server error HTTP 503"             │
│  Cache Hit: No | Retry Count: 3                              │
└──────────────────────────────────────────────────────────────┘
```

**Components:**
- `TransactionFilters` — Exophone selector (from health list), Status dropdown (ALL / RUNNING / SUCCESS / FAILED / RETRYING / TIMEOUT), DateRangePicker (from/to), Limit selector
- `TransactionTable` — paginated, sortable by started_at; expandable rows showing full error detail
- `StatusTag` — colour-coded: SUCCESS=green, FAILED=red, RUNNING=blue, RETRYING=amber, TIMEOUT=grey
- `LatencyCell` — shows latency in ms, red highlight if > 3000ms

**Filtering:** Server-side — all filters passed as query params to `GET /api/v1/transactions`.

**No auto-polling** — this is a manual query/audit view. User hits "Search" or changes filters.

---

### 10.6 System Health — `/health`

**Purpose:** Infrastructure dependency status. Quick check for ops and SRE.

**APIs consumed:**
- `GET /health`

**Layout:**

```
┌─────────────────────────────────────────────┐
│  System Health                   Last checked: just now │
├─────────────────────────────────────────────┤
│                                             │
│   Overall Status:  ● UP                    │
│                                             │
│   ┌─────────────────────┐                  │
│   │ MySQL    ● CONNECTED │                  │
│   └─────────────────────┘                  │
│   ┌─────────────────────┐                  │
│   │ Redis    ● CONNECTED │                  │
│   └─────────────────────┘                  │
│                                             │
│   Auto-refreshes every 15 seconds          │
└─────────────────────────────────────────────┘
```

**Components:**
- `OverallStatusBadge` — large UP / DEGRADED indicator
- `DependencyStatusCard` × 2 — MySQL, Redis with green/red status dots
- `LastCheckedLabel` — relative timestamp ("just now", "15s ago")

**Polling:** 15 s.

---

## 11. Shared Components Spec

### KpiCard

```typescript
interface KpiCardProps {
  title: string;
  value: string | number;
  unit?: string;           // 'calls', 'ms', '%'
  trend?: 'up' | 'down' | 'neutral';
  highlight?: 'success' | 'warning' | 'danger';
  loading?: boolean;
}
```

### StatusTag

| Status | Colour |
|--------|--------|
| SUCCESS | Green |
| RUNNING | Blue |
| FAILED | Red |
| RETRYING | Orange |
| TIMEOUT | Grey |

### SeverityTag

| Severity | Colour |
|----------|--------|
| CRITICAL | Red (filled) |
| WARNING | Orange (filled) |
| INFO | Blue (outline) |

### HeartbeatBadge

| Status | Dot colour | Label |
|--------|-----------|-------|
| OK | Green | OK |
| DEGRADED | Amber | DEGRADED |
| OUTAGE | Red | OUTAGE |
| UNKNOWN | Grey | UNKNOWN |

### CallStatusDonut

- Recharts `PieChart` with `Pie` + `Tooltip` + `Legend`
- Segments: `connected_calls` (green), `failed_calls` (red), `busy_calls` (amber)
- Centre label: success rate %
- Data source: `CallMetricsSnapshot`

---

## 12. Navigation & Layout

### Sidebar items

| Icon | Label | Route |
|------|-------|-------|
| 📊 | Dashboard | /dashboard |
| 📞 | Exophones | /exophones |
| 🔔 | Alerts | /alerts |
| 🔍 | Transactions | /transactions |
| ❤️ | System Health | /health |

### Top Navbar

- Platform name: **Exotel Monitor**
- Right side: `AlertCountBadge` (red dot with count from `/alerts/active`), System status chip from `/health`
- The alert count badge links to `/alerts`

### Breadcrumbs

- `/exophones/:id` → Exophones / +91XXXXXXXXXX
- `/transactions?exophone_id=X` → Transactions / Exophone X

---

## 13. Filtering & Sorting Requirements

| Page | Filter Type | Implementation |
|------|------------|----------------|
| Exophone List | Account, Status, Priority | Client-side on fetched data |
| Alerts | Severity, Alert Type | Client-side on fetched data |
| Transactions | Exophone, Status, Date Range, Limit | Server-side query params |
| Dashboard | (none — global view) | — |

---

## 14. Polling Strategy

| Page / Hook | Interval | Rationale |
|---|---|---|
| System Health | 15 s | Fast infra check |
| Active Alerts | 30 s | Ops need near-realtime |
| Exophone Health Grid | 30 s | Monitoring visibility |
| Dashboard Summary | 60 s | Snapshot-based, slow-moving |
| Exophone Metrics | 60 s | Snapshot data |
| Transactions | On-demand only | Audit / debugging view |

All intervals are implemented via TanStack Query's `refetchInterval`. A manual "Refresh" button is provided on every page as a fallback.

---

## 15. Error Handling

### API Error States

All pages must handle three states per data fetch:

1. **Loading** — show `<LoadingSpinner />` or Ant Design Skeleton
2. **Error** — show `<ErrorBoundary />` with the error message and a Retry button (calls `refetch()`)
3. **Empty** — show `<EmptyState />` when the API returns an empty array

### Global Error Toast

The Axios response interceptor fires a notification toast (Ant Design `notification.error`) for any unhandled 4xx/5xx response.

### Network Offline

Use `navigator.onLine` + `window addEventListener('offline')` to show a banner: *"You are offline. Data may be stale."*

---

## 16. Loading States

- KPI cards → Ant Design `<Skeleton.Input active />` placeholders
- Tables → Ant Design `<Table loading={true} />`
- Health cards → Ant Design `<Skeleton active />`
- Charts → show a spinner over the chart container until data arrives

---

## 17. Non-Functional Requirements

| Requirement | Target |
|---|---|
| Initial page load (LCP) | < 2 s on 4G |
| Time to interactive | < 3 s |
| Bundle size (gzipped) | < 500 KB |
| Accessibility | WCAG 2.1 AA |
| Browser support | Chrome 100+, Firefox 100+, Edge 100+ |
| Mobile responsiveness | Tablet (768px+) minimum; sidebar collapses to hamburger |
| API timeout handling | 15 s timeout, show error state |

---

## 18. Development Quick Start

```bash
# 1. Clone and install
npm create vite@latest exotel-monitoring-frontend -- --template react-ts
cd exotel-monitoring-frontend
npm install

# 2. Install dependencies
npm install axios @tanstack/react-query antd recharts react-router-dom tailwindcss

# 3. Set API base URL
echo "VITE_API_BASE_URL=http://localhost:8080" > .env.local

# 4. Start backend (from backend repo)
cd deployments && docker compose up -d
go run cmd/server/main.go

# 5. Start frontend dev server
npm run dev
# → http://localhost:5173
```

---

## 19. API-to-Page Mapping (Complete Reference)

| Backend Endpoint | Method | Page(s) | Hook | Purpose |
|---|---|---|---|---|
| `/health` | GET | System Health, Navbar | `useSystemHealth` | Infrastructure status |
| `/api/v1/dashboard/summary` | GET | Dashboard | `useDashboardSummary` | Account KPI cards + table |
| `/api/v1/dashboard/exophones/:id/calls` | GET | Exophone Detail | `useExophoneCallAnalytics` | Donut chart + call stats |
| `/api/v1/accounts/:id/exophones` | GET | Exophone List | `useExophones` | Inventory table |
| `/api/v1/accounts/:id/exophones/health` | GET | Exophone List | `useExophoneHealthList` | Health card grid |
| `/api/v1/exophones/:id/metrics` | GET | Exophone Detail | `useExophoneMetrics` | KPI cards |
| `/api/v1/alerts/active` | GET | Alerts, Navbar badge, Exophone Detail | `useActiveAlerts` | Alert list |
| `/api/v1/alerts` | POST | (internal / manual fire) | `useCreateAlert` | Manual alert trigger |
| `/api/v1/alerts/:id/acknowledge` | PUT | Alerts, Exophone Detail | `useAcknowledgeAlert` | Acknowledge action |
| `/api/v1/transactions` | GET | Transactions | `useTransactions` | Audit log |

---

## 20. Future Enhancements (Phase 2)

- WebSocket connection for real-time alert push (eliminates polling)
- Date-range picker on Exophone Detail for historical call trends (line chart)
- Exophone priority edit inline in the table
- Bulk alert acknowledge
- Export transactions to CSV
- Dark mode toggle
- Grafana embed iframe for advanced metrics
