# Decisions

## D1 — parsePage max limit: 100 → 1000
**Decision**: Raise backend exophone list cap from 100 to 1000.
**Why**: Frontend Alerts/Transactions pages need all exophones for filter dropdowns. With max=100, accounts with >100 exophones can never see all exophones in one request. 1000 is safe given typical exophone counts per account.
**Alternative rejected**: Keep 100 and add full pagination in frontend dropdowns — too much churn for a dropdown; 1000 covers all realistic prod scenarios.

## D2 — GetAllExophonesByAccount for scheduler
**Decision**: Add a dedicated no-limit repo function for internal processes.
**Why**: `GetExophonesByAccount(page, limit)` is an API-facing function designed for pagination. Internal loops must never silently miss exophones due to a page cap. Separate function makes the intent explicit and prevents accidental LIMIT in scheduler.

## D3 — IVR mode tests must set Direction="inbound" on all records
**Decision**: All IVR-mode test records must have `Direction: "inbound"` even for non-connected outcomes.
**Why**: Records without Direction classify as Leg2 (rule 5: catch-all). If any record is Leg2, Leg2Total > 0, and the panel mode correction fires, subtracting leg1Connected from Connected. IVR mode has no Leg2 records by definition.

## D4 — Panel mode Connected correction denominator
**Decision**: AnswerRate in panel mode uses Leg2Total (not Total) as denominator.
**Why**: In panel mode, Total = Leg1Total + Leg2Total (2 CDRs per call). Using Total would halve the AnswerRate. Leg2Total = unique calls to customer = correct denominator for "did customer answer?".

## D5 — UpsertAccountDashboardSnapshot only saves exophone counts
**Decision**: Leave this as-is. The hourly cron's computed total_calls is dropped by the SQL (intentional).
**Why**: Call data is accurately maintained by `AccumulateAccountDashboardFromSnapshots` (pure SQL, no LIMIT) after every CALLS job. The hourly cron only needs to refresh `total_exophones` and `active_exophones` counts which can change via exophone adds/removes.
