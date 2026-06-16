# Integration Test Report

## Test Run: 2026-06-17

### Build Status
```
go build ./...  → PASS (clean, no errors)
```

### Unit Test Results
```
internal/exotel    → PASS
internal/metrics   → PASS (13 tests)
internal/snapshots → PASS
```

---

## Test Scenarios

### 1. IVR Mode — Status Classification
**Input**: 6 IVR inbound records (completed×2 with ConvDur>0, no-answer×2, failed×1, busy×1)
**Expected**: Total=6, Connected=2, NoAnswer=2, Failed=1, Busy=1, Leg2Total=0
**Actual**: PASS
**Notes**: All records need Direction="inbound" for correct IVR classification; records without Direction default to Leg2 and trigger panel mode correction.

### 2. IVR Mode — Answer Rate
**Input**: 4 records (completed×3 with ConvDur>0, no-answer×1 — all inbound)
**Expected**: AnswerRate=75.00, SuccessRate=AnswerRate
**Actual**: PASS

### 3. Leg Classification — All 5 Rules
**Input**: 6 records testing each classification rule
**Expected**: Rule1 (ParentSid→Leg2), Rule2 (inbound→Leg1), Rule3 (outbound+no-parent→Leg1), Rule4 (To=VN→Leg1), Rule5 (unknown→Leg2)
**Actual**: PASS

### 4. ConversationDuration Gate
**Input**: completed records — one with ConvDur=0, one with ConvDur=30
**Expected**: ConvDur=0 → drop, not Connected; ConvDur>0 → Connected
**Actual**: PASS

### 5. IVR Drop Classification (Leg2Status split)
**Input**: 2 completed+ConvDur=0 records — one with Leg2Status="" (IVR abandon = Leg1 drop), one with Leg2Status="canceled" (agent routing attempted = Leg2 drop)
**Expected**: DroppedLeg1=1, DroppedLeg2=1
**Actual**: PASS

### 6. Panel Mode — Connected Dedup
**Input**: Leg1 + Leg2 CDR pair both with ConvDur>0 (simulates one bridged panel call)
**Expected**: Connected=1 (not 2), Leg2Total=1
**Actual**: PASS

### 7. Panel Mode — Partial Connect
**Input**: Leg1 ConvDur=45 (agent answered), Leg2 ConvDur=0 (customer no-answer)
**Expected**: Connected=0 (Leg1 connected subtracted), DroppedLeg2=1
**Actual**: PASS

### 8. Cursor Pagination
**Input**: Mock Exotel API returning 2 pages with NextPageUri cursor
**Expected**: Both pages accumulated, all records returned
**Actual**: PASS

### 9. Time Parsing
**Input**: Various Exotel timestamp formats (DB, RFC3339, RFC2822)
**Expected**: All parse correctly to IST timezone
**Actual**: PASS

---

## Known Gaps (not unit-tested, verified by code review)

| Scenario | Status |
|---|---|
| Redis dedup across overlap window | Code-reviewed ✓ |
| Backfill race condition | Code-reviewed ✓ |
| Cache invalidation after jobs | Code-reviewed ✓ |
| Per-exophone skip_call_logs dashboard path | Code-reviewed ✓ |
| Frontend limit=100 for Alerts/Transactions dropdowns | Code-reviewed ✓ |
| parsePage clamping to 1000 | Code-reviewed ✓ |

---

## Pre-Deploy Checklist

- [x] `go build ./...` clean
- [x] All unit tests pass
- [ ] `scripts/migrate_001_leg_status.sql` applied to prod DB
- [ ] Redis available and `call_dedup:*` keys not conflicting
- [ ] Frontend build passes (`npm run build` in exotel-monitoring-frontend/)
