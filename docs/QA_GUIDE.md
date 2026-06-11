# Exotel Monitoring Platform QA Guide

**Author:** Senior QA Engineer  
**Audience:** QA, Development, Release Management  
**Objective:** Define a complete quality strategy, risk-based test coverage, and execution approach for the Exotel Monitoring Platform.

---

## 1. Product Overview

The Exotel Monitoring Platform is a production-style monitoring system that collects telephony and operational health data from Exotel, processes it through scheduled jobs, stores it in MySQL, caches dashboards in Redis, triggers alerts, and exposes monitoring data through backend APIs and a React-based operational dashboard.

### Core business functions
- Monitor Exophone health and availability.
- Collect call, heartbeat, and stream metrics.
- Persist raw monitoring transactions and generated snapshots.
- Trigger alerts on failures, latency breaches, throttling, and degraded health.
- Serve dashboard summaries and drill-down analytics to operations teams.

### System layers
1. **Frontend:** React + Vite + TypeScript dashboard for operations users.
2. **Backend API:** Go + Gin REST APIs.
3. **Data layer:** MySQL for persistent storage, Redis for caching and high-speed reads.
4. **Scheduling/processing:** worker pool, cron-driven monitoring jobs, snapshot refresh, cleanup tasks.
5. **Alerting:** Slack integrated alert dispatch and persistence.

---

## 2. Quality Goals

### Primary goals
- Ensure accurate monitoring data ingestion, processing, and presentation.
- Validate system stability under normal operations, API failures, and partial infrastructure issues.
- Confirm that alerting works correctly and that operational dashboards are trustworthy.
- Ensure graceful degradation when Redis, MySQL, or upstream Exotel APIs fail.

### Non-functional quality targets
- Backend APIs return correct status codes and consistent payloads.
- Frontend handles loading, empty states, network failures, and stale data safely.
- Alerts are persisted and dispatched without breaking monitoring execution.
- Scheduler and worker pool maintain bounded resource usage.
- Recovery paths from failure conditions are predictable and observable.

---

## 3. Quality Strategy

### Test levels
| Level | Focus | Examples |
|---|---|---|
| Unit | Logic correctness inside isolated functions | retry handling, metric calculation, formatting, encryption helpers |
| Component/Hook | UI rendering and data behavior | loading states, error states, cached data workflows |
| API/Integration | Backend REST behavior with persistence and cache interactions | health endpoint, dashboard summary, alerts, webhooks |
| End-to-end | Cross-layer workflows | dashboard refresh, alert-triggered failure scenarios |
| Regression | Repeated validation around known failure paths | degraded service, fallback logic, scheduler stability |

### Risk-based priorities
1. External Exotel API instability and throttling.
2. MySQL and Redis connectivity failures.
3. Incorrect alert generation or missed alerts.
4. Incorrect dashboard metrics and stale data.
5. Frontend failures in error states, loading states, or empty states.
6. Scheduler overrun or worker saturation.

---

## 4. Key Business Workflows to Validate

### Backend workflows
- Infrastructure health endpoint returns service status.
- Dashboard summary serves cached data first and falls back to snapshot data.
- Exophone analytics and transaction history return meaningful data.
- Monitoring jobs create transactions, call upstream APIs, store responses, refresh snapshots, and advance execution.
- Alerts are persisted and dispatched to Slack on relevant failures.
- Cleanup tasks retain retention policies.

### Frontend workflows
- Dashboard loads KPI summary and renders counts.
- Exophone pages show health and metrics.
- Alerts page displays active alerts, severity, and status.
- Transactions page shows execution history.
- System Health page reflects backend service status.
- Error and empty states are user-friendly and actionable.

---

## 5. Environment and Test Data Strategy

### Recommended environments
| Environment | Purpose |
|---|---|
| Local development | Developer validation |
| QA integration | API/UI testing with DB and Redis |
| Staging | Pre-release regression validation |
| Production-like | Final release verification |

### Data setup
- Seed MySQL with accounts, exophones, monitoring jobs, and transaction data.
- Seed Redis with expected cache keys for dashboard behavior verification.
- Use controlled stubbed upstream API responses for deterministic failures.
- Validate secrets and encryption handling with non-production keys.

### Infrastructure dependencies
- MySQL 8
- Redis 7
- Docker Compose for local stack
- Backend service on port 8080
- Frontend development server

---

## 6. Backend QA Coverage

### API contract validation
- Required fields exist in successful responses.
- Unsupported paths return correct error objects.
- Invalid path parameters return 400/404 appropriately.
- Response payload shape remains stable across release changes.

### Functional coverage
- Health endpoint returns `UP` when both MySQL and Redis respond.
- Health endpoint returns degraded status when one dependency fails.
- Dashboard summary returns cached data when Redis contains values.
- Dashboard summary falls back to snapshot data when cache misses occur.
- Exophone metrics endpoint handles missing snapshots gracefully.
- Transactions endpoint returns ordered data with filtering behavior.
- Alerts endpoint returns active alerts, supports creation and acknowledgement.
- Webhook endpoint accepts Exotel payloads and processes them without crashing.

### Business logic coverage
- Heartbeat retry behavior uses configured retries.
- High latency triggers warnings.
- 429 throttling is detected and recorded.
- Retry exhaustion escalates into critical alerting.
- Worker pool maintains concurrency limits.
- Snapshot refresh updates dashboard-serving tables.

### Error handling coverage
- MySQL connection failure should degrade health response.
- Redis ping failure should degrade health response.
- Invalid exophone ID should return 400.
- Missing snapshot should return 404 or empty body based on endpoint behavior.
- Upstream API errors should be persisted and alerted, not silently ignored.
- Alert dispatch failures must be logged and stored in dispatch logs.

---

## 7. Frontend QA Coverage

### UI and routing
- Navigation between Dashboard, Exophones, Alerts, Transactions, and System Health.
- Sidebar/header layout loads consistently.
- Page titles and route-level empty states behave correctly.

### Data-loading states
- Loading spinner or skeleton while fetching.
- Error panel for failed API requests.
- Empty state when no data is available.

### Functional UI validation
- KPI cards render totals, rates, and count metrics.
- List and table views show correct columns and row values.
- Charts load correctly and show meaningful labels.
- Severity badges and status tags are color-coded correctly.
- Alert action workflows, such as acknowledgement, are reflected in UI.

### Frontend error handling
- API failures display user-friendly messages.
- Failed requests do not leave the page blank.
- Retry or refresh actions recover from temporary failures.
- Invalid route handling gives a graceful fallback.

### Performance/UX checks
- Rapid refreshes do not break rendering.
- Large datasets remain scrollable and usable.
- Polling or refetch behavior does not duplicate rows or flicker unexpectedly.

---

## 8. Error Handling Standards

### Required error handling principles
1. Every API call must have deterministic outcomes.
2. A user-facing failure must never result in a blank crash page.
3. System failures must be recorded in logs and persisted where possible.
4. Failures must surface severity and context.
5. Degraded behavior must still preserve operational visibility.

### Backend error handling expectations
- Responses must use structured JSON error messages.
- Failures in external dependencies should be logged with correlation identifiers.
- Retry and alert escalations must respect configured thresholds.
- Internal failures must not leave partially updated transactions untracked.

### Frontend error handling expectations
- All API failures must be represented in UI state.
- Error banners must contain actionable guidance.
- Failed network requests must not destroy existing good data.
- Loading states must recover after failures.

---

## 9. Automation Strategy

### Backend automation
- Use Go testing for handlers, utilities, repository behavior, and retry logic.
- Favor integration tests for API routes, DB interactions, and cache fallback paths.
- Validate log and alert side effects through real repository interactions where possible.

### Frontend automation
- Use React testing tooling for routing, async data handling, and rendering.
- Cover API success, loading, empty, and error conditions.
- Prefer real user behavior interactions instead of mock-only assertions.

### Test execution gates
- Smoke suite on every commit.
- API/core regression suite in QA environment.
- UI regression suite before release.
- Manual-focused validation for alert delivery and operational dashboards.

---

## 10. Defect Severity and Escalation

| Severity | Meaning | Examples |
|---|---|---|
| Critical | Release-blocking production risk | incorrect alerting, monitoring data loss, system outage |
| High | Major functionality broken | dashboard summary wrong, API endpoint failing |
| Medium | Partial feature issue | UI rendering issue, stale data | 
| Low | Cosmetic or minor UX defect | minor alignment, label mismatch |

---

## 11. Suggested Release Checklist

- [ ] Backend API regression suite passes.
- [ ] Frontend UI regression suite passes.
- [ ] Health endpoint and dependency checks pass.
- [ ] Alerting triggers verified.
- [ ] Dashboard fallback and cache behavior verified.
- [ ] Error handling scenarios exercised.
- [ ] Release notes and known risks documented.

---

## 12. Senior QA Observations

The platform is operationally critical. The most important test risk is not a simple UI bug; it is an integrity issue where monitoring data becomes misleading. QA must treat every failure as a business-impact issue, especially where alerting, snapshots, and dashboard metrics are involved. The system must never appear healthy when dependencies are degraded, and it must never silently fail to record or alert on operational issues.
