# Changes Log

## Session 1 (Prior — from context summary)

### internal/cache/dedup.go (NEW)
- Added Redis-based Sid dedup: `AddCallSids()`, `CallDedupKey()`, `TTLCallDedup=25h`
- Key: `call_dedup:{exophone_id}:{date-IST}` SET, pipeline SADD, TTL 25h
- Impact: eliminates overlap-window double-counting in AccumulateCallMetrics

### internal/metrics/engine.go
- Added `leg1Connected` tracker for Panel mode Connected dedup
- Post-loop: if `Leg2Total > 0`, subtract `leg1Connected` from `Connected`
- Standardized `Leg1DropRate` denominator to `Leg1Total` (fallback to Total)
- Impact: Connected no longer double-counted for Panel mode calls

### internal/workers/job_worker.go
- Added `filterNewRecords()` using `cache.AddCallSids` for overlap dedup
- Added `cache.Delete(KeyDashboardSum)` after CALLS job completes
- Fixed per-exophone `skip_call_logs` path to use `AccumulateAccountDashboardFromSnapshots`
- Impact: overlap window calls no longer counted twice; dashboard cache correctly invalidated

### internal/snapshots/generator.go
- Added `cache.Delete(HealthSnapKey(exophoneID))` after `UpsertExophoneHealthSnapshot`
- Impact: health snapshot cache invalidated after every refresh

### internal/handlers/backfill.go
- Added `cache.AddCallSids` after backfill to register Sids in Redis dedup
- Added `cache.Delete(MetricsKey, KeyDashboardSum)` after backfill
- Impact: backfill-today race condition eliminated; cache consistent after backfill

### internal/handlers/reprocess.go
- Added `cache.AddCallSids` after reprocess
- Added `cache.Delete(MetricsKey, KeyDashboardSum)` after reprocess
- Impact: cache consistent after manual reprocess

### internal/metrics/engine_test.go
- Fixed `TestProcessCallRecords_StatusClassification`: added `Direction: "inbound"` + `ConversationDuration`
- Fixed `TestProcessCallRecords_AnswerRate`: added `Direction: "inbound"` + `ConversationDuration`
- Added 5 new tests: `TestClassifyLeg`, `TestProcessCallRecords_ConversationDurationGate`,
  `TestProcessCallRecords_IVRDropClassification`, `TestProcessCallRecords_PanelModeConnectedDedup`,
  `TestProcessCallRecords_PanelModePartialConnect`

---

## Session 2 (Current)

### internal/exotel/calls.go (line 66)
- Fixed syntax error: `SetBasicok toAuth` → `SetBasicAuth`
- Reason: typo/corruption blocked all compilation in internal/exotel package
- Impact: CRITICAL — unblocks all tests

### internal/handlers/exophones.go
- Fixed `parsePage()`: raised limit cap from 100 → 1000; fixed clamp logic (was resetting to 20 for any limit > 100, now clamps to 1000)
- Before: `if limit < 1 || limit > 100 { limit = 20 }`
- After: `if limit < 1 { limit = 20 } else if limit > 1000 { limit = 1000 }`
- Impact: frontend/admin can request up to 1000 exophones per page without silent truncation to 20

### internal/repository/exophone_repo.go
- Added `GetAllExophonesByAccount(accountID uint64)` — no LIMIT, no pagination
- Used by internal processes that must iterate ALL exophones (not API-facing)
- Impact: scheduler can enumerate all exophones without hitting a page cap

### internal/scheduler/scheduler.go
- `runHourlySnapshotRefresh()`: replaced `GetExophonesByAccount(a.ID, 1, 1000)` with `GetAllExophonesByAccount(a.ID)`
- Before: accounts with >1000 exophones had wrong exophone counts in hourly snapshot
- Impact: hourly snapshot accurately reflects all exophones regardless of account size

### exotel-monitoring-frontend/src/pages/Alerts.tsx
- `useExophones(accountIds[0] ?? 0)` → `useExophones(accountIds[0] ?? 0, 1, 100)`
- `useExophones(accountIds[1] ?? 0)` → `useExophones(accountIds[1] ?? 0, 1, 100)`
- Impact: Alerts page filter dropdown shows up to 100 exophones (was 20)

### exotel-monitoring-frontend/src/pages/Transactions.tsx
- Same change as Alerts.tsx
- Impact: Transactions page filter dropdown shows up to 100 exophones (was 20)

### internal/metrics/engine_test.go
- Added `Direction: "inbound"` to IVR-mode records that had no Direction set
- Root cause: records without Direction classify as Leg2 → Leg2Total > 0 → panel mode
  correction fires → Connected subtracted to 0
- Impact: tests correctly model IVR mode; panel mode correction only fires for panel data
