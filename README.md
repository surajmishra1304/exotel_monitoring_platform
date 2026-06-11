# Exotel Monitoring Platform

A production-grade Go service that provides centralised monitoring, alerting, and dashboard analytics for multiple Exotel accounts and Exophones.

---

## Architecture

```
Exotel APIs (Heartbeat / Calls / Streams / Webhooks)
        │
        ▼
Monitoring Scheduler (cron, DB-driven)
        │
        ▼
Bounded Worker Pool (semaphore, configurable size)
        │
     ┌──┴──────────────────┐
     ▼                     ▼
  MySQL               Redis Cache
  (persist)           (hot-serve)
     │                     │
     └──────────┬──────────┘
                ▼
        Snapshot Tables
                │
                ▼
        Dashboard APIs   ←  Alerting (Slack / Email / Webhook)
```

---

## Tech Stack

| Layer     | Technology       |
|-----------|-----------------|
| Language  | Go 1.22+        |
| Router    | Gin             |
| ORM       | GORM            |
| Database  | MySQL 8         |
| Cache     | Redis 7         |
| Scheduler | robfig/cron v3  |
| Logging   | Uber Zap        |
| Config    | Viper           |
| HTTP      | Resty           |
| Container | Docker Compose  |

---

## Quick Start (Local)

### 1. Prerequisites

- Go 1.22+
- Docker & Docker Compose

### 2. Start infrastructure

```bash
cd deployments
docker compose up -d
```

### 3. Install Go dependencies

```bash
go mod tidy
```

### 4. Configure

Edit `configs/config.yaml`:
- Set your Slack webhook URL and enable Slack: `slack.enabled: true`
- Set `crypto.secret_key` to a base64-encoded 32-byte key
- Point `exotel.base_url` and `exotel.heartbeat_url` to your Exotel cluster

### 5. Run

```bash
go run cmd/server/main.go
```

### 6. Verify

```bash
curl http://localhost:8080/health
# {"status":"UP","mysql":"CONNECTED","redis":"CONNECTED"}
```

---

## Key Optimisations Made

| Area | Change |
|------|--------|
| Worker pool | Replaced unbounded goroutines with a **bounded semaphore pool** (configurable size, default 20) |
| Retry logic | **Exponential backoff with full jitter** — different strategy per failure type (429 → delayed, 5xx → exponential, 401 → no retry) |
| Metrics | Complete `ProcessCallRecords` that computes total/failed/connected/busy counts, avg duration, success rate, failure % |
| Snapshots | `RefreshExophoneHealth` + `UpdateCallMetricsSnapshot` wired to run after every job execution |
| Alert coverage | All 7 alert types fully implemented: API_FAILURE, HEARTBEAT_FAILURE, HIGH_LATENCY, RETRY_EXHAUSTED, VENDOR_THROTTLING, EXOPHONE_DOWN, CACHE_FAILURE |
| Dashboard APIs | Cache-first (Redis → snapshot → DB fallback), no raw aggregation queries |
| Secret management | AES-256-GCM encryption/decryption for API keys and tokens |
| Graceful shutdown | SIGINT/SIGTERM → HTTP drain → scheduler drain → worker pool drain |
| Connection pooling | MySQL pool (MaxOpen=100, MaxIdle=20, lifetime=1h) + Redis pool (20) |

---

## API Reference

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /health | Infrastructure health check |
| GET | /api/v1/dashboard/summary | Account-level KPI summary |
| GET | /api/v1/dashboard/exophones/:id/calls | Call analytics for exophone |
| GET | /api/v1/accounts/:account_id/exophones | List exophones for account |
| GET | /api/v1/accounts/:account_id/exophones/health | Health snapshots |
| GET | /api/v1/exophones/:id/metrics | Single exophone health snapshot |
| GET | /api/v1/transactions | Job transaction log (filterable) |
| GET | /api/v1/alerts/active | Active (OPEN) alerts |
| POST | /api/v1/alerts | Fire an alert manually |
| PUT | /api/v1/alerts/:id/acknowledge | Acknowledge an alert |
| POST | /exotel/webhook | Exotel Passthru Applet webhook receiver |

---

## Monitoring Job Types

| Job Type | Description |
|----------|-------------|
| HEARTBEAT | Calls Exotel heartbeat API, stores health metric, triggers alerts on non-OK |
| CALLS | Fetches call records for the last polling window, stores call logs + metrics |
| STREAMS | Fetches active concurrent stream count, stores stream metrics, alerts on >90% utilisation |

---

## Priority Polling

| Priority | Frequency |
|----------|-----------|
| P0 | 15 minutes |
| P1 | 30 minutes |
| P2 | 1 hour |
| P3 | Daily |

---

## Project Structure

```
exotel-monitoring-platform/
├── cmd/server/main.go          ← Entry point with graceful shutdown
├── configs/config.yaml         ← Application configuration
├── deployments/docker-compose.yml
├── Dockerfile                  ← Multi-stage build
├── scripts/init.sql            ← Complete DB schema + seed data
└── internal/
    ├── alerts/                 ← Slack dispatcher + alert evaluator
    ├── cache/                  ← Redis helpers with key constants + TTLs
    ├── config/                 ← Viper config loader
    ├── database/               ← MySQL (with pool) + Redis connections
    ├── exotel/                 ← API clients: heartbeat, calls, streams
    ├── handlers/               ← Gin HTTP handlers
    ├── logger/                 ← Zap logger
    ├── metrics/                ← Metrics calculation engine
    ├── middleware/             ← Request ID injection
    ├── models/                 ← GORM models for all tables
    ├── repository/             ← DB access layer (no business logic)
    ├── scheduler/              ← Cron scheduler + cleanup jobs
    ├── server/                 ← HTTP server + router setup
    ├── snapshots/              ← Snapshot generation & upsert logic
    ├── utils/                  ← Retry (exponential backoff) + AES crypto
    └── workers/                ← Bounded worker pool + job executor
```

---

## Environment Variables

All config values can be overridden via environment variables using `_` as separator:

```
MYSQL_HOST=db.internal
MYSQL_PASSWORD=securepassword
REDIS_HOST=cache.internal
SLACK_WEBHOOK_URL=https://hooks.slack.com/...
CRYPTO_SECRET_KEY=base64encodedkey==
```
