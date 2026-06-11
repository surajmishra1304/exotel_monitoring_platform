# Exotel Monitoring Platform QA Test Cases

**Prepared by:** Senior QA Engineer  
**Scope:** Backend APIs, frontend dashboard, and error handling across operational monitoring workflows.

---

## 1. Test Case Matrix

| ID | Area | Objective | Priority | Automation |
|---|---|---|---|---|
| TC-BK-001 | Backend Health | Validate health endpoint under healthy dependencies | P0 | Automated |
| TC-BK-002 | Backend Dashboard | Validate cache-first summary flow | P0 | Automated |
| TC-BK-003 | Backend Dashboard Fallback | Validate snapshot fallback when cache misses | P0 | Automated |
| TC-BK-004 | Backend Alerts | Validate alert persistence and dispatch flow | P0 | Automated |
| TC-BK-005 | Backend Transactions | Validate transaction retrieval and ordering | P1 | Automated |
| TC-BK-006 | Backend Error Handling | Validate invalid exophone ID handling | P0 | Automated |
| TC-BK-007 | Backend Retry/Alerting | Validate retry exhaustion and failure escalation | P0 | Automated |
| TC-BK-008 | Backend Scheduler | Validate worker pool and snapshot refresh behavior | P1 | Semi-automated |
| TC-FR-001 | Frontend Dashboard | Validate KPI rendering and summary data | P0 | Automated |
| TC-FR-002 | Frontend Exophones | Validate exophone list and detail rendering | P0 | Automated |
| TC-FR-003 | Frontend Alerts | Validate active alerts rendering and status | P0 | Automated |
| TC-FR-004 | Frontend Transactions | Validate transaction history rendering | P1 | Automated |
| TC-FR-005 | Frontend Error Handling | Validate loading, empty, and failure states | P0 | Automated |
| TC-FR-006 | Frontend System Health | Validate system health page status behavior | P1 | Automated |

---

## 2. Backend Test Cases

### TC-BK-001: Health endpoint shows healthy status when MySQL and Redis are available
- **Type:** API integration test
- **Preconditions:** MySQL and Redis running; backend service started.
- **Steps:**
  1. Send `GET /health`.
  2. Observe HTTP response.
  3. Validate response schema.
- **Expected Result:**
  - HTTP status `200`.
  - Body contains `status: UP`.
  - `mysql: CONNECTED`.
  - `redis: CONNECTED`.
- **Senior QA Notes:** This is the earliest health gate for the full platform. Any regression here blocks release.

### TC-BK-002: Dashboard summary uses Redis cache when available
- **Type:** Cache integration test
- **Preconditions:** Redis contains dashboard summary data for the current key.
- **Steps:**
  1. Seed Redis with a known dashboard summary payload.
  2. Send `GET /api/v1/dashboard/summary`.
  3. Verify response source and payload.
- **Expected Result:**
  - HTTP status `200`.
  - Response contains `source: cache`.
  - Returned data matches seeded Redis payload.
  - No fallback DB query is required for response generation.
- **Senior QA Notes:** Cache correctness is critical because dashboards are read-heavy and cache-first behavior is part of the product design.

### TC-BK-003: Dashboard summary falls back to snapshot table when cache is empty
- **Type:** Database fallback test
- **Preconditions:** Redis has no dashboard key; MySQL contains account dashboard snapshots.
- **Steps:**
  1. Clear dashboard cache key.
  2. Ensure snapshot data exists in MySQL.
  3. Send `GET /api/v1/dashboard/summary`.
- **Expected Result:**
  - HTTP status `200`.
  - Response contains `source: snapshot`.
  - Returned account rows match snapshot table values.
  - Cache is warmed for subsequent reads.
- **Senior QA Notes:** This verifies the backend remains useful even when Redis is cold.

### TC-BK-004: Alert creation persists and dispatches Slack notification
- **Type:** Alert processing test
- **Preconditions:** Slack webhook enabled or mocked endpoint available; database available.
- **Steps:**
  1. Post an alert payload to `POST /api/v1/alerts`.
  2. Validate alert record saved in database.
  3. Validate dispatch log created.
- **Expected Result:**
  - HTTP status `201` or success response.
  - Stored alert has correct type, severity, and status.
  - Dispatch log indicates sent or failed depending on environment; failure path must be logged.
- **Senior QA Notes:** Alerting is an operational safety function, so persistence and dispatch must be both validated.

### TC-BK-005: Transactions endpoint returns ordered history for monitoring execution
- **Type:** API query test
- **Preconditions:** Multiple job transactions exist in database.
- **Steps:**
  1. Send `GET /api/v1/transactions`.
  2. Validate result ordering and fields.
- **Expected Result:**
  - HTTP status `200`.
  - Returned transaction list includes `transaction_id`, status, job type, latency, retries, timestamps.
  - Most recent transactions appear first.
- **Senior QA Notes:** Transaction history is a key audit trail for operations teams.

### TC-BK-006: Invalid exophone ID returns a client error
- **Type:** API validation test
- **Preconditions:** None.
- **Steps:**
  1. Send `GET /api/v1/dashboard/exophones/invalid/calls`.
  2. Review response.
- **Expected Result:**
  - HTTP status `400`.
  - Error message clearly indicates invalid exophone ID.
- **Senior QA Notes:** Invalid input must fail fast and predictably.

### TC-BK-007: Retry exhaustion escalates into critical alerting
- **Type:** Failure and alert escalation test
- **Preconditions:** Monitoring job configured with retry policy; upstream API unreachable.
- **Steps:**
  1. Execute a heartbeat or calls job against an unavailable upstream API.
  2. Allow retries to exhaust.
  3. Check transaction status and alert records.
- **Expected Result:**
  - Transaction ends in failed state.
  - Retry count equals configured max retries.
  - Critical alert for retry exhaustion is created.
  - Dispatch attempt is logged.
- **Senior QA Notes:** This validates the most serious operational failure path.

### TC-BK-008: Scheduler refresh updates snapshots without overload
- **Type:** Scheduler regression test
- **Preconditions:** Active accounts and exophones seeded; worker pool initialized.
- **Steps:**
  1. Start scheduler.
  2. Allow scheduled poll and snapshot refresh cycles to run.
  3. Monitor worker concurrency and snapshot tables.
- **Expected Result:**
  - Polling jobs execute without exceeding configured concurrency.
  - Snapshot refresh updates aggregate state.
  - No deadlock or repeated failed scheduling.
- **Senior QA Notes:** Scheduler correctness protects system stability under load.

---

## 3. Frontend Test Cases

### TC-FR-001: Dashboard loads KPI summary and renders key widgets
- **Type:** UI rendering test
- **Preconditions:** API returns valid dashboard summary.
- **Steps:**
  1. Open dashboard route.
  2. Wait for data fetch completion.
  3. Validate KPI cards and charts render.
- **Expected Result:**
  - Dashboard loads successfully.
  - Summary cards show totals and percentages.
  - Charts render without layout errors.
- **Senior QA Notes:** The dashboard is the primary operational view; rendering must be trustworthy.

### TC-FR-002: Exophones page renders account-specific exophone data
- **Type:** UI list test
- **Preconditions:** Backend returns exophone list and health data.
- **Steps:**
  1. Open exophones page.
  2. Validate table rows, status tags, and counts.
- **Expected Result:**
  - Exophone rows display expected values.
  - Status badges reflect health.
  - Navigation to exophone detail works.
- **Senior QA Notes:** Exophone data is the central operational asset.

### TC-FR-003: Alerts page shows active alerts and severity state
- **Type:** UI alert test
- **Preconditions:** Backend returns active alerts.
- **Steps:**
  1. Open alerts page.
  2. Validate alert rows, severity styling, and update states.
- **Expected Result:**
  - Alerts render with correct severity labels and timestamps.
  - No rendering errors on row expansion or filtering.
- **Senior QA Notes:** Alert visibility is critical for incident response.

### TC-FR-004: Transactions page shows monitoring execution history
- **Type:** UI transactional history test
- **Preconditions:** Mocked backend returns transaction data.
- **Steps:**
  1. Open transactions page.
  2. Validate list entries and status badge mapping.
- **Expected Result:**
  - Transaction rows display statuses, durations, and timestamps.
  - Failed and retried states are clearly shown.
- **Senior QA Notes:** This page provides execution auditability.

### TC-FR-005: Frontend shows graceful fallback on API failure
- **Type:** UI failure-state test
- **Preconditions:** Backend returns error for dashboard API.
- **Steps:**
  1. Simulate network failure or 500 response.
  2. Open dashboard page.
- **Expected Result:**
  - Loading ends without blank page.
  - Error message is user-friendly and actionable.
  - Retry or refresh path remains available.
- **Senior QA Notes:** Users must understand failure conditions and recover without confusion.

### TC-FR-006: System Health page reflects dependency status
- **Type:** UI dependency status test
- **Preconditions:** Backend health endpoint returns degraded state.
- **Steps:**
  1. Open system health page.
  2. Validate displayed connectivity status.
- **Expected Result:**
  - System health reflects degraded status.
  - Users can distinguish backend, MySQL, and Redis conditions.
- **Senior QA Notes:** Operational dashboards must not hide infrastructure degradation.

---

## 4. Error Handling Test Cases

### TC-ERR-001: Backend returns degraded status when Redis fails
- **Type:** Infrastructure failure test
- **Preconditions:** Redis unavailable; MySQL available.
- **Steps:**
  1. Stop Redis.
  2. Call `/health`.
- **Expected Result:**
  - HTTP status `503`.
  - Response shows `redis: DISCONNECTED`.
  - Overall status is degraded.
- **Senior QA Notes:** Degraded health is better than false `UP` status.

### TC-ERR-002: Backend returns degraded status when MySQL fails
- **Type:** Infrastructure failure test
- **Preconditions:** MySQL unavailable; Redis available.
- **Steps:**
  1. Stop MySQL.
  2. Call `/health`.
- **Expected Result:**
  - HTTP status `503`.
  - Response shows `mysql: DISCONNECTED`.
  - Overall status is degraded.
- **Senior QA Notes:** This prevents false healthy reporting.

### TC-ERR-003: Frontend handles server error without crashing
- **Type:** UI resilience test
- **Preconditions:** Dashboard API returns 500 or broken payload.
- **Steps:**
  1. Open dashboard page under failure conditions.
- **Expected Result:**
  - Error panel shown.
  - Existing page structure remains intact.
  - No JavaScript crash or blank page.
- **Senior QA Notes:** UI resilience is a core operational requirement.

### TC-ERR-004: Missing snapshot data is handled gracefully
- **Type:** Data absence test
- **Preconditions:** No snapshot exists for requested exophone.
- **Steps:**
  1. Request exophone metrics for absent data.
  2. Open relevant UI page.
- **Expected Result:**
  - Backend returns a controlled error or empty result.
  - UI shows empty/no data state instead of rendering invalid content.
- **Senior QA Notes:** Missing telemetry must be visible, not misleading.

### TC-ERR-005: Alert dispatch failure is recorded and does not halt execution
- **Type:** Alert dispatch resilience test
- **Preconditions:** Slack endpoint unreachable.
- **Steps:**
  1. Trigger alert.
  2. Observe alert persistence and dispatch log.
- **Expected Result:**
  - Alert record is created.
  - Dispatch failure logged.
  - Monitoring execution continues.
- **Senior QA Notes:** Alerting must fail safely.

---

## 5. Suggested Execution Order

1. Run backend health and dashboard fallback tests.
2. Run alerting and retry exhaustion scenarios.
3. Run frontend dashboard and alert rendering tests.
4. Run degraded infrastructure and missing data scenarios.
5. Execute smoke regression against the full stack.

---

## 6. QA Sign-Off Criteria

Release is ready only when:
- backend API and cache fallback tests pass,
- frontend loading/error states are validated,
- degraded dependency handling is verified,
- alert persistence and dispatch behavior is stable,
- no critical or high severity defects remain open.
