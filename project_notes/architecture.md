# Architecture

## Stack
- **Backend**: Go (Gin, GORM, go-redis/v8, robfig/cron)
- **Frontend**: React + TypeScript (Ant Design, TanStack Query, Recharts)
- **DB**: MySQL (via GORM)
- **Cache**: Redis

## Core Data Flow

```
Exotel API → FetchCalls (cursor-paginated, PageSize=100, up to 500 pages)
           → filterNewRecords (Redis Sid dedup, call_dedup:{id}:{date} SET)
           → metrics.ProcessCallRecords (classifyLeg, accumulate)
           → repository.AccumulateCallMetrics → call_metrics_snapshot
           → repository.AccumulateAccountDashboardFromSnapshots (pure SQL, no limit)
           → cache.Delete(MetricsKey, KeyDashboardSum)
```

## Job Scheduling
- `scheduler.Start()` → cron every N seconds → `workers.RunDueJobs()`
- `GetDueJobs()` — no LIMIT, fetches all due jobs ordered by priority
- `ExecuteJob(job)` — per-job lifecycle: txn → fetch → store raw → process → alert → snapshot → advance
- Hourly cron → `runHourlySnapshotRefresh()` — updates exophone counts only

## Exophone Sync
- `SyncExophones` handler calls `exotel.ListIncomingPhoneNumbers()` (page-based, 50 pages × 100)
- Existing from DB: `GetExophonesByAccount(accountID, 1, 10000)` — high enough

## Repository Layer (no-limit functions)
- `GetMonitoredExophones()` — no LIMIT, all active monitored exophones
- `GetDueJobs()` — no LIMIT
- `AccumulateAccountDashboardFromSnapshots()` — pure SQL, no LIMIT
- `GetAllExophonesByAccount(accountID)` — [ADDED] no LIMIT, used by scheduler

## API Pagination
- Exophones list: `parsePage()` — page/limit, max 1000 (raised from 100)
- TodayPerformance: `parsePage()` — same
- Transactions: max 500

## Cache Keys & TTLs
- `metrics:{id}` — 15 min
- `dashboard:summary` — 15 min
- `health_snap:{id}` — 5 min
- `call_dedup:{id}:{date-IST}` — 25 h

## Leg Classification (per Exotel docs)
1. `ParentCallSid != ""` → Leg2
2. `Direction == "inbound"` → Leg1
3. `Direction == "outbound-api/dial"` + no parent → Leg1
4. `To == VN` → Leg1 (fallback)
5. catch-all → Leg2

## Connected Dedup (Panel mode)
- Both Leg1 and Leg2 may have ConversationDuration > 0
- Post-loop: if Leg2Total > 0, subtract leg1Connected to avoid double-count
