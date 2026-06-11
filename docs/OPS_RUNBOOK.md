# Exotel Monitoring Platform — Ops Runbook

> Audience: Ops, Support, and On-Call engineers. No coding knowledge assumed.

---

## 1. What this platform does

We monitor every Exotel virtual number (Exophone) owned by Reglobe. Every 10–60 minutes the system checks:

- Is the Exophone alive? (Heartbeat)
- How many calls went through it? (Call Analytics)
- How busy is the voice stream? (Stream Utilization)

Results are stored in a dashboard and alerts fire when something breaks.

---

## 2. Where to access it

| Service | URL |
|---|---|
| Dashboard (UI) | http://localhost:5173 |
| Backend API | http://localhost:8080 |
| API Health | http://localhost:8080/health |

---

## 3. Jargon glossary

### Platform terms

| Term | Plain-English meaning |
|---|---|
| **Exophone** | A virtual phone number provided by Exotel (e.g. 07948224931). Calls to this number get routed by Exotel. We monitor each one independently. |
| **Exophone ID** | Our internal numeric ID for an Exophone. Not the phone number itself — the number `12` means the 12th Exophone we registered in the system. |
| **Account SID** | Exotel's unique identifier for a business account. Acts like a username. Example: `reglobe`. |
| **Subdomain** | The company-specific URL prefix Exotel assigns. For Reglobe it's `reglobe`, making API calls go to `reglobe.exotel.com`. |
| **API Key / Token** | Credentials used to authenticate our system with Exotel's API. Stored AES-256 encrypted in the database — never stored in plain text. |
| **Cluster** | Exotel's data-centre region. `in1` = India. Other values: `sg1` (Singapore), `us1` (US). |
| **Priority (P0/P1/P2/P3)** | How often we check an Exophone: P0=every 10 min (most critical), P1=every 30 min, P2/P3=every 60 min. |
| **Monitoring Flag** | `1` = actively monitored, `0` = paused. |
| **is_cron_applicable** | Whether the scheduler should run jobs for this Exophone. `1` = yes, `0` = excluded. |

### Call Analytics terms

| Term | Plain-English meaning |
|---|---|
| **Total Calls** | All calls that came through the Exophone in the selected time period. |
| **Connected** | Calls that were answered by a person (`status=completed`). |
| **Missed (No Answer)** | Caller rang but nobody picked up (`status=no-answer`). |
| **Failed** | Call dropped due to a network or system error (`status=failed`). Not the same as no-answer. |
| **Busy** | Callee was already on a call; caller got a busy tone (`status=busy`). |
| **Answer Rate** | `Connected ÷ Total × 100`. The percentage of calls that actually reached a person. A drop in Answer Rate is the first signal of a problem. |
| **Avg Duration** | Average length of connected calls, shown as minutes and seconds (e.g. `2m 14s`). Only counts completed calls. |
| **Peak Hour** | The hour of the day (0–23, IST) with the highest call volume. `18` means 6 PM had the most calls. |
| **Hourly Distribution** | A 24-bar chart showing how many calls happened in each hour of the day. Helps identify busy periods vs dead hours. |
| **Snapshot** | A pre-computed daily summary stored in the database. The dashboard serves snapshots (fast) rather than re-reading every call log on each page load. |

### Health & Heartbeat terms

| Term | Plain-English meaning |
|---|---|
| **Heartbeat** | A periodic check that verifies Exotel's service is reachable and the Exophone is still registered. |
| **Heartbeat Status** | `OK` = everything working; `DEGRADED` = partial issues (some services affected); `OUTAGE` = Exotel is down or the number is unreachable; `UNKNOWN` = we haven't checked yet. |
| **Availability %** | What percentage of heartbeat checks returned `OK` in the rolling window. `100%` = no issues detected. |
| **API Latency (ms)** | How long the last Exotel API call took to respond, in milliseconds. Under 1000ms is healthy. Over 3000ms is a warning. |
| **Active Streams** | How many live voice calls are passing through the Exophone right now. |
| **Stream Utilization %** | `Active Streams ÷ Max Allowed × 100`. At 80%+ the Exophone is near capacity and new calls may get dropped. |

### Job / Scheduler terms

| Term | Plain-English meaning |
|---|---|
| **Job** | A scheduled task for one Exophone. Types: `HEARTBEAT`, `CALLS`, `STREAMS`, `SNAPSHOT`, `CLEANUP`. |
| **Transaction** | One execution of a job. Each run gets a unique Transaction ID (e.g. `TXN-20260607-221554-34`). |
| **Transaction Status** | `RUNNING` = in progress; `SUCCESS` = completed without error; `FAILED` = errored out; `RETRYING` = being retried after a failure; `TIMEOUT` = took too long. |
| **Retry Policy** | When a job fails we retry with exponential backoff. Default: up to 3 retries, starting at 1 second, capping at 30 seconds. |
| **Worker Pool** | 20 background threads that execute jobs concurrently. If all 20 are busy, new jobs queue up. |
| **Poll Interval** | How often the scheduler checks for jobs that are due. Default: 60 seconds. |
| **Next Run At** | Timestamp of the next scheduled execution for a job. |

### Alert terms

| Term | Plain-English meaning |
|---|---|
| **Alert** | A notification created when something crosses a threshold. |
| **Severity** | `CRITICAL` = immediate action needed; `WARNING` = monitor closely; `INFO` = informational only. |
| **Alert Status** | `OPEN` = unacknowledged; `ACKNOWLEDGED` = someone has seen it and is looking into it; `RESOLVED` = fixed. |
| **Alert Types** | See table below. |
| **Channel** | Where the alert is sent: `SLACK`, `EMAIL`, or `WEBHOOK`. |

#### Alert type reference

| Alert Type | What it means | What to do |
|---|---|---|
| `HEARTBEAT_FAILURE` | Exotel's heartbeat API returned `CRITICAL` or `OUTAGE` | Check Exotel's status page; escalate to Exotel support if it persists > 15 min |
| `RETRY_EXHAUSTED` | A monitoring job failed 3 times in a row | Check the error message in the alert — usually a network issue or Exotel API down |
| `HIGH_LATENCY` | API response time > 3000ms | Check your network connectivity to Exotel; may be a temporary spike |
| `API_FAILURE` | Exotel API returned an HTTP error (4xx/5xx) | Check credentials are valid; check Exotel's status page |
| `EXOPHONE_DOWN` | Exophone not found on the Exotel account | Verify the number is still provisioned in Exotel console |
| `VENDOR_THROTTLING` | Exotel returned HTTP 429 (too many requests) | Reduce monitoring frequency (lower priority) or contact Exotel to raise rate limits |
| `CACHE_FAILURE` | Redis is unavailable | Check if the Redis service is running |

---

## 4. Dashboard pages explained

### Home / Summary
Shows all accounts at a glance:
- Total and active Exophones
- Today's total calls and failed calls
- Overall success rate
- Number of open alerts

### Exophones List
All Exophones for an account. Click any row to open the detail view.

### Exophone Detail (the main monitoring page)
Three sections:

**KPI cards (top row)**
- Availability % — Is this number alive?
- API Latency — How fast is Exotel responding?
- Active Streams — How many calls right now?

**Call Analytics**
- Date picker (top right of the card) — pick any past date to view historical data. Defaults to today.
- Pie chart — visual breakdown of call outcomes
- Stats — exact counts per outcome + Answer Rate + Avg Duration + Peak Hour
- Hourly Distribution bar chart — blue bar = the peak hour; green = all other hours; a bar touching zero for several hours usually means the Exophone was idle or down

**Health Snapshot**
Last-known state: heartbeat status, availability, latency, streams, and when it was last checked.

**Active Alerts**
Any unresolved alerts for this specific Exophone. Click `Ack` once you've acknowledged an alert.

---

## 5. Normal vs abnormal readings

| Metric | Healthy | Warning | Critical |
|---|---|---|---|
| Heartbeat Status | OK | DEGRADED | OUTAGE |
| Availability % | ≥ 99% | 90–98% | < 90% |
| Answer Rate | ≥ 85% | 70–84% | < 70% |
| API Latency | < 1000ms | 1000–3000ms | > 3000ms |
| Stream Utilization | < 70% | 70–85% | > 85% |
| Open CRITICAL Alerts | 0 | — | ≥ 1 |

---

## 6. Common ops scenarios

### "The dashboard shows no data for today"
Normal if it's early morning — the first snapshot generates after the first monitoring job runs (~10–60 min depending on priority). If still empty past 11 AM, check if the backend process is running (`http://localhost:8080/health`).

### "Answer Rate dropped suddenly"
1. Open the Exophone detail page
2. Look at the Hourly Distribution chart — which hour did calls drop?
3. Cross-reference with Exotel status page for that time window
4. Check if `failed_calls` or `no_answer_calls` spiked (different root causes)

### "All jobs are RETRY_EXHAUSTED"
Likely a connectivity issue to Exotel's API from the server. Check:
1. `http://localhost:8080/health` — is MySQL/Redis still connected?
2. Can the server reach `{subdomain}.exotel.com`?
3. Are the API credentials still valid?

### "Heartbeat is UNKNOWN"
The system hasn't run a heartbeat check yet, or the last check was too long ago. If it stays UNKNOWN for more than 30 minutes, the scheduler may have stopped. Restart the backend process.

### "Stream Utilization is at 90%+"
The Exophone is near capacity. New inbound calls may start getting rejected. Immediately:
1. Acknowledge the alert
2. Contact Exotel to increase the concurrent call limit for that number
3. If possible, load-balance incoming calls across another Exophone

### "There are hundreds of OPEN alerts"
This usually means a batch of jobs all failed at the same time (e.g. during a network outage). Once the root cause is fixed:
1. Verify the health endpoint is clean
2. Bulk-acknowledge alerts via the dashboard
3. The next job cycle will automatically create fresh data

---

## 7. Key database tables (for escalations)

| Table | What's in it |
|---|---|
| `accounts` | Exotel account credentials (encrypted) and subdomain |
| `exophones` | Every virtual number we monitor |
| `monitoring_jobs` | The schedule — what to check, when, how often |
| `job_transactions` | History of every job execution |
| `call_logs` | Raw call records fetched from Exotel |
| `call_metrics_snapshot` | Pre-computed daily call stats per Exophone (what the dashboard shows) |
| `exophone_health_snapshot` | Latest heartbeat/availability/stream reading per Exophone |
| `alerts` | All alerts ever raised |
| `heartbeat_metrics` | Raw heartbeat check results over time |

---

## 8. Startup / restart procedure

```bash
# 1. Start the backend (from the project root)
./exotel-monitor-bin &

# 2. Start the frontend (from exotel-monitoring-frontend/)
npm run dev &

# 3. Verify both are up
curl http://localhost:8080/health   # should return {"status":"UP","mysql":"CONNECTED","redis":"CONNECTED"}
open http://localhost:5173          # should load the dashboard
```

If either service fails to start, check that MySQL is running on port 3306 and Redis on port 6379.

---

## 9. What data is and isn't real-time

| Data | Freshness |
|---|---|
| Heartbeat status | Updated every 10–60 min (depends on Exophone priority) |
| Active streams | Updated every 10–60 min |
| Call counts (today) | Accumulated every 10–60 min, recalculated from full day's logs each cycle |
| Hourly distribution | Recomputed each cycle from all call logs for that day |
| Historical snapshots | Fixed — represent the final count for that calendar date |
| Alerts | Created in real-time as jobs run |
| Dashboard summary | Cached in Redis for 5 minutes; refreshes automatically |
