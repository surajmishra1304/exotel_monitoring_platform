# Exotel Monitoring Platform — Complete Product Guide

> For teams onboarding to this platform for the first time. No prior technical knowledge assumed.
> Version 2.0 — Updated June 2026

---

## What Is This Platform?

Your business uses **Exotel** — a cloud telephony provider — to handle phone calls with customers. You have virtual phone numbers (called **Exophones**) through which those calls flow.

This platform is a **health and performance monitoring dashboard** built on top of Exotel. It continuously checks whether your phone numbers are working, tracks call success rates, measures how fast Exotel's systems respond, and alerts you when something goes wrong — before your customers notice.

Think of it like a 24×7 control room for your phone operations.

---

## Key Terminology

### Organisation (Account)
A single Exotel account registered under one set of credentials (API Key + API Token). At Cashify/Reglobe, this is the **"Reglobe"** account. One organisation can own many Exophones. Credentials are stored **AES-256-GCM encrypted** — no plaintext API keys anywhere in the database.

### Exophone
A virtual phone number provided by Exotel (e.g., `07948224931`). Customers call or receive calls from this number. Each Exophone is individually monitored by this platform. You can have 50+ Exophones under one organisation.

> **Validation:** When adding Exophones to an organisation, the platform calls Exotel's API live to verify each number actually belongs to that account. Numbers that don't belong are rejected before any DB write.

### Monitoring Job
An automated background task that runs on a schedule. For each Exophone, three job types run repeatedly:

| Job Type | What It Checks | Default Frequency |
|---|---|---|
| **HEARTBEAT** | Is Exotel's service up? Are calls affected? | Per priority config |
| **CALLS** | How many calls happened? Success vs failure? | Per priority config |
| **STREAMS** | How many active calls right now? Capacity used? | Per priority config |

### Priority (P0 → P3)
Controls how often jobs run and which ones get processed first when the system is busy. **Frequencies are fully configurable** from the Priority Config tab — not hardcoded.

| Priority | Default Frequency | Use Case |
|---|---|---|
| **P0** | 15 minutes | Mission-critical numbers (main customer helpline) |
| **P1** | 30 minutes | Important but not critical numbers |
| **P2** | 60 minutes | Standard numbers |
| **P3** | 60 minutes | Low-priority / test numbers |

P0 jobs are always processed before P1, P1 before P2, and so on.

### Priority Configuration Table
A dedicated DB table (`priority_config`) that stores the frequency in minutes for each priority level. This is the **single source of truth** for monitoring intervals — changing a value here immediately affects all new job scheduling for that priority.

Accessible from the frontend via the **Priority Config** tab on the Exophones page. Each priority row shows:
- Priority level (P0–P3)
- Description
- Check every N minutes (editable inline)
- Last updated timestamp + who updated it

**Limits:** Frequency must be between **1 and 1440 minutes** (1 minute to once per day). Values outside this range are rejected with HTTP 400.

### `is_cron_applicable` Flag
A per-Exophone on/off switch that controls whether that number is included in the scheduler. **Default is OFF (0).** You must explicitly enable it for each Exophone you want monitored. Toggle it from the **Cron Active** column in the Exophones table. This lets you monitor only the numbers you care about without deleting the others.

### Health Snapshot
A point-in-time summary of an Exophone's health, stored after every job run. Contains:
- **Heartbeat Status** — OK / DEGRADED / OUTAGE / UNKNOWN
- **Availability %** — percentage of time the number was reachable
- **Last API Latency** — how long Exotel took to respond (milliseconds)
- **Active Streams** — how many calls are live right now

### Call Metrics Snapshot
A daily summary per Exophone containing:
- Total calls received/made
- Connected calls (answered)
- Failed calls (dropped, unanswered)
- Busy calls
- Average call duration
- Success rate (%)

### Transaction
Every single job run is recorded as a transaction. It logs: when it started, how long it took, whether it succeeded or failed, how many retries it needed, and the raw response from Exotel. Useful for debugging.

### Alert
An automated notification triggered when something is wrong:

| Alert Type | Triggered When |
|---|---|
| HEARTBEAT_FAILURE | Exotel's service reports degradation or outage |
| API_FAILURE | Exotel's API returned an error |
| HIGH_LATENCY | API response took longer than 3 seconds |
| EXOPHONE_DOWN | A specific number stopped responding |
| RETRY_EXHAUSTED | A job failed 3 times in a row |
| VENDOR_THROTTLING | Exotel rate-limited our requests |
| CACHE_FAILURE | Redis (our speed layer) went down |

Alerts have three states: **OPEN** → **ACKNOWLEDGED** → **RESOLVED**

### Cache (Redis)
A speed layer sitting in front of the database. When the frontend asks for a health snapshot, it first checks Redis (response in ~1ms). Only if Redis doesn't have it does it query the database (~200ms). Keeps the dashboard fast even with 50+ Exophones.

### Cron / Scheduler
The internal engine that runs monitoring jobs on a schedule. It polls every 60 seconds, picks up all jobs that are due (based on their `next_run_at` timestamp), and dispatches them to a pool of 20 parallel workers. Only Exophones where `is_cron_applicable = 1` are included.

### Worker Pool
20 background workers that execute monitoring jobs in parallel. When 150 jobs are due at once (50 Exophones × 3 job types), all 20 workers start immediately and process the queue concurrently — the full batch completes in roughly the time it takes one job to run.

### Cluster
The Exotel data center region where your account is hosted. Common values: `in1` (India 1 — default), `in2`, `us1`, `ap1`. Must match your Exotel account's region or API calls will fail.

---

## How the System Works — End to End

```
1. You create an Organisation via the frontend
   └── Platform validates each Exophone number against Exotel's API live
       (rejects any number that doesn't belong to this account)
   └── Reads frequency_minutes for each priority from priority_config table
   └── Creates account + exophones + 3 monitoring jobs per exophone
   └── API credentials (Key + Token) are AES-256-GCM encrypted before DB storage

2. You configure Priority Config (optional)
   └── Go to Exophones → Priority Config tab
   └── Click edit on any priority row
   └── Set the frequency in minutes (1–1440)
   └── All new exophones created with that priority will use this frequency

3. You enable is_cron_applicable for the numbers you want monitored
   └── Toggle the Cron Active switch in the Exophones table

4. Every 60 seconds, the Scheduler checks:
   "Which jobs are due right now, for exophones with cron enabled?"
   └── Fetches due jobs ordered by priority (P0 first)
   └── Dispatches them to the 20-worker pool

5. Each worker runs its job:
   HEARTBEAT → calls Exotel heartbeat API → stores service status
   CALLS     → calls Exotel Calls API → stores call counts + metrics
   STREAMS   → calls Exotel Streams API → stores active stream count

6. After each job:
   └── Transaction logged (success/failure/latency)
   └── Health snapshot updated in DB + Redis cache
   └── Call metrics snapshot updated (daily aggregation)
   └── If failure: Alert triggered and dispatched (Slack/Email/Webhook)
   └── Job's next_run_at advanced by frequency_minutes (from priority_config)

7. Frontend reads snapshots (cache-first) to display the dashboard
```

---

## Priority Configuration in Detail

### How to Change Monitoring Frequency

1. Go to **Exophones** page → **Priority Config** tab
2. Click the pencil (✏) icon next to any priority row
3. Type the new frequency in minutes (1–1440)
4. Click **Save** — the change takes effect for the next scheduler cycle

### What Happens When You Change a Frequency

| Scenario | Effect |
|---|---|
| Change P2 from 60 min to 30 min | All *future* jobs for P2 exophones will schedule at 30-minute intervals after their next run |
| Change P0 from 15 min to 5 min | P0 jobs will check every 5 minutes after next cycle |
| Set any priority to 1440 min | That priority is checked at most once per day |

> **Note:** Changing the priority config does not retroactively reschedule in-progress jobs. The new frequency applies from the next `next_run_at` advance.

### Overriding Frequency Per Exophone

You can also set a **custom frequency for a single exophone** without changing the global priority config:

1. Click the priority badge (e.g., **P2**) next to any exophone in the table
2. A popover opens with a priority dropdown + frequency input
3. The frequency input pre-fills with the current priority config default
4. Override the value and click **Save**

This sets a custom `frequency_minutes` on all 3 monitoring jobs for that specific exophone only — it does not affect the priority config table or other exophones.

---

## Frontend Pages at a Glance

| Page | What You Do Here |
|---|---|
| **Dashboard** | Overview of all organisations — total calls, failure counts, alert counts |
| **Exophones → Exophones tab** | List all numbers; toggle `Cron Active`; click priority badge to edit priority/frequency |
| **Exophones → Performance tab** | Daily call metrics per number — connected, failed, success rate. Has a **date picker** to view any historical date. |
| **Exophones → Priority Config tab** | View and edit the global monitoring frequency for each priority level (P0–P3) |
| **Exophone Detail** | Deep-dive on one number — health KPIs, call analytics pie chart, related alerts |
| **Alerts** | All open/acknowledged alerts across the org; acknowledge from here |
| **Transactions** | Full log of every job run — useful for debugging a specific failure |
| **System Health** | MySQL and Redis connection status |

---

## API Reference (Key Endpoints)

| Method | Endpoint | Description |
|---|---|---|
| GET | `/health` | System health (MySQL + Redis status) |
| POST | `/api/v1/accounts` | Create organisation + exophones + jobs |
| GET | `/api/v1/dashboard/summary` | All accounts KPI summary |
| GET | `/api/v1/accounts/:id/exophones` | Paginated exophone list (`?page=&limit=`) |
| GET | `/api/v1/accounts/:id/exophones/health` | Health snapshots for all exophones |
| GET | `/api/v1/accounts/:id/performance/today` | Daily call metrics (`?date=YYYY-MM-DD&page=&limit=`) |
| GET | `/api/v1/exophones/:id` | Single exophone record |
| GET | `/api/v1/exophones/:id/metrics` | Health snapshot for one exophone |
| PATCH | `/api/v1/exophones/:id/priority` | Change priority + frequency for one exophone |
| PATCH | `/api/v1/exophones/:id/cron` | Toggle `is_cron_applicable` (0 or 1) |
| GET | `/api/v1/priority-config` | List all 4 priority config rows |
| PATCH | `/api/v1/priority-config/:priority` | Update frequency for P0/P1/P2/P3 |
| GET | `/api/v1/alerts/active` | All open alerts |
| PUT | `/api/v1/alerts/:id/acknowledge` | Acknowledge an alert |
| GET | `/api/v1/transactions` | Job execution history |
| POST | `/exotel/webhook` | Exotel call-event webhook receiver |

---

## What Is Stored and For How Long

| Table | What's In It | Retention |
|---|---|---|
| `accounts` | Organisation credentials (AES-256-GCM encrypted) | Permanent |
| `exophones` | Phone numbers + config + cron flag | Permanent |
| `priority_config` | 4 rows: P0/P1/P2/P3 with frequency_minutes | Permanent (editable) |
| `monitoring_jobs` | Job schedule config | Permanent |
| `job_transactions` | Every job run record | Until manually cleaned |
| `job_responses` | Raw API responses | Until manually cleaned |
| `call_logs` | Individual call records | Until manually cleaned |
| `heartbeat_metrics` | Service status history | Until manually cleaned |
| `stream_metrics` | Stream usage history | Until manually cleaned |
| `exophone_health_snapshot` | Latest health per exophone | Overwritten each run |
| `call_metrics_snapshot` | Daily call totals per exophone | One row per exophone per day |
| `alerts` | All triggered alerts | Until resolved/deleted |

---

## API Cost Analysis

### Do Exotel's Reporting APIs Cost Money Per Call?

**No.** Exotel does not charge per GET request to their reporting/monitoring APIs. The billing model is based on:
- Telephony minutes consumed (actual calls made/received)
- Virtual number monthly rental
- Plan subscription credits

The three endpoints this platform uses — Heartbeat, Calls reporting, and Active Streams — are **included in your plan at no per-call surcharge**.

### Rate Limits

Exotel enforces a global limit of **200 API requests per minute** across all Voice APIs. Breaching it returns HTTP 429 (the platform's retry logic handles this).

**Usage calculation at 50 Exophones on P2 (default 60-min frequency):**

| Metric | Calculation | Value |
|---|---|---|
| Jobs per cycle | 50 exophones × 3 job types | 150 jobs |
| Cycles per hour | 60 min ÷ 60 min frequency | 1 cycle |
| API calls per hour | 150 × 1 | **150 calls/hr** |
| API calls per day | 150 × 24 | **3,600 calls/day** |
| API calls per month | 3,600 × 30 | **~108,000 calls/month** |
| Peak burst rate | 150 jobs over ~30 sec | **~5 calls/sec = ~300/min** |

> The peak rate of ~300 calls/min can briefly exceed the 200/min rate limit. For sustained high-frequency use (P0 on 50+ exophones), request a higher rate limit from your Exotel account manager.

**Impact of changing Priority Config:**

| Config | Calls/hr | Calls/day |
|---|---|---|
| All 50 on P3, 60 min (default) | 150 | 3,600 |
| All 50 on P0, 15 min | 600 | 14,400 |
| All 50 on P0, 5 min | 1,800 | 43,200 ⚠️ rate-limit risk |
| Mixed: 10 P0 (15min) + 40 P2 (60min) | 150 | 3,600 |

---

## Infrastructure Cost to Run This Platform

| Component | Spec | Estimated Monthly Cost |
|---|---|---|
| MySQL database | AWS RDS t3.micro | ₹800–₹2,500 |
| Redis cache | ElastiCache t3.micro | ₹500–₹1,500 |
| Go backend | 1 vCPU, 512MB RAM | ₹300–₹800 |
| Frontend hosting | Static build, Nginx/CDN | ₹0–₹500 |
| **Total** | | **₹1,600–₹5,300 / month** |

Local dev/test: ₹0.

---

## Security Notes

| Area | Status |
|---|---|
| API credentials at rest | AES-256-GCM encrypted — never stored plaintext |
| API credentials in responses | Hidden (`json:"-"` tag) — never returned by any endpoint |
| CORS | Explicit allowlist — unknown origins receive 403 |
| Webhook signature | HMAC-SHA256 verified when `EXOTEL_WEBHOOK_SECRET` env var is set |
| SQL injection | Parameterised queries via GORM ORM — injection attempts return 400 |
| Integer overflow in IDs | Capped at INT32 max (2,147,483,647) — oversized IDs return 400 |
| Duplicate SID | Returns 409 Conflict, no internal DB details exposed |
| **Missing: Authentication** | ⚠️ All API endpoints are currently open — add JWT/API-key middleware before production |
| **Missing: IDOR protection** | ⚠️ Account ID is trusted from URL — must validate ownership after auth is added |

---

## Quick-Start Checklist for a New User

1. Open the frontend at `http://localhost:5173`
2. Go to **Exophones** → click **New Organisation**
3. Enter your Exotel Account SID, API Key, API Token, and add phone numbers with their priority
4. The platform validates each number against Exotel live — only real numbers are accepted
5. *(Optional)* Go to **Priority Config** tab and adjust monitoring frequencies if needed
6. In the **Exophones** table, flip the **Cron Active** toggle ON for the numbers you want monitored
7. Within 60 seconds the scheduler fires its first cycle — check **Transactions** to see jobs running
8. Health snapshots appear on the **Exophone Detail** page after the first cycle completes
9. Set up Slack alerts by updating `slack.webhook_url` in `configs/config.yaml` and setting `enabled: true`
10. For production: set `EXOTEL_WEBHOOK_SECRET=<your-secret>` in the server environment

---

## Contact & Escalation

| Issue | Where to Look First |
|---|---|
| All jobs failing | System Health page → check MySQL/Redis status |
| One number's jobs failing | Transactions page → filter by exophone → read error message |
| Dashboard showing stale data | Redis may be down; check System Health |
| Monitoring too frequent / too slow | Exophones → Priority Config tab → adjust frequency |
| Rate limit errors (429) | Reduce P0/P1 exophones or increase their frequency_minutes in Priority Config |
| Credential errors (401) | API Key/Token may have rotated; re-create the organisation |
| Cron not running for a number | Check `is_cron_applicable` flag — toggle Cron Active in Exophones table |

---

## QA & Security Test Results (v2 — June 2026)

**39 / 39 tests passed.** 5 previously reported CVEs confirmed fixed. 1 medium finding fixed immediately.

| Category | Tests | Pass |
|---|---|---|
| Functional | 12 | 12 ✓ |
| Priority Config CRUD | 10 | 10 ✓ |
| Edge Cases | 14 | 14 ✓ |
| VAPT Regression (prior CVEs) | 5 | 5 ✓ |
| VAPT New Surface (priority config) | 8 | 8 ✓ (after VN04 fix) |

**Known open items before production go-live:**
1. Add authentication middleware (JWT or API key) — currently all routes are open
2. Add IDOR ownership checks once auth is in place
3. Enable webhook signature verification via `EXOTEL_WEBHOOK_SECRET` env var

---

*Generated for Cashify/Reglobe internal use. Platform version: 2.0. Last updated: June 2026.*
