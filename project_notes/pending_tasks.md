# Pending Tasks

## DONE (this session)
- [x] Fix syntax error `SetBasicok toAuth` → `SetBasicAuth` in internal/exotel/calls.go:66
- [x] Redis Sid dedup (call_dedup key) — cache/dedup.go
- [x] filterNewRecords in job_worker.go (overlap window fix)
- [x] Panel mode Connected double-count fix in engine.go
- [x] Cache invalidation after RefreshExophoneHealth in snapshots/generator.go
- [x] Cache invalidation after backfill and reprocess in handlers
- [x] dashboard:summary cache never invalidated → fixed in job_worker.go
- [x] per-exophone skip_call_logs account dashboard path → fixed in job_worker.go
- [x] Fix stale tests + add 5 new tests in engine_test.go
- [x] Raise parsePage limit cap from 100 → 1000 in handlers/exophones.go
- [x] Add GetAllExophonesByAccount to exophone_repo.go
- [x] Fix scheduler.go hourly refresh — replace hardcoded 1000 with GetAllExophonesByAccount
- [x] Fix parsePage clamp logic (was resetting to 20 instead of clamping to max)
- [x] Frontend Alerts.tsx — useExophones with limit=100
- [x] Frontend Transactions.tsx — useExophones with limit=100

## PRE-DEPLOY (manual action needed)
- [ ] Apply scripts/migrate_001_leg_status.sql to production DB (adds 12 columns across 3 tables — idempotent)
- [ ] Confirm Redis is available with sufficient memory for call_dedup keys
- [ ] Run scripts/truncate_metrics.sql to clear stale data before backfill

## FUTURE (not blocking deploy)
- [ ] call_logs optimisation: skip row-per-call DB writes (flag: scheduler.skip_call_logs_write) — design pending approval
- [ ] UpsertAccountDashboardSnapshot: currently only persists exophone counts (not calls) — intentional for now; call data handled by AccumulateAccountDashboardFromSnapshots
