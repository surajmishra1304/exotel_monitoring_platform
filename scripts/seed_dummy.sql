-- ============================================================
-- Dummy seed data for local testing
-- Uses the crypto key from configs/config.yaml
-- Run: mysql -u root -proot exotel_monitoring < scripts/seed_dummy.sql
-- ============================================================

USE exotel_monitoring;

-- ============================================================
-- ACCOUNTS  (api_key / api_token are AES-256-GCM encrypted)
-- Replace with real Exotel credentials for live API testing
-- ============================================================

INSERT INTO accounts (id, account_name, sid, api_key, api_token, subdomain, cluster, is_active, created_by) VALUES
(1, 'cash ',  'cash',
 'hlWC5Q8rbC+vsvlaLyyWy2+SXQiUdXvtGkMiZiQV9ArwFIQiddScADLMNmLPomLk5fDVqQ==',
 'oMXSOJphAL0ArxHGYGJNm4NNuHOEAetONf/GNG5+wV9uugwoRQ5IQR2XPrmNJjMc9wtd+/uB',
 'cashify', 'in1', 1, 'SYSTEM'),
(2, 'Demo Corp',      'DEMO_CORP_002',
 'FdixF3rLiB/kUS0EnqAYWwgyvfUtXgDDKGNcR7xiVsAxY6vCAsC1ylc4JVKRVoH76A==',
 'JUzdi19Ok+DGjWTW3EJYBsL4Zhpwekido9kOJMWRkDc2nr9zZmCCn//bya3kThBFQN54',
 'democorp', 'in2', 1, 'SYSTEM')
ON DUPLICATE KEY UPDATE updated_at = NOW();

-- ============================================================
-- EXOPHONES
-- ============================================================

INSERT INTO exophones (id, account_id, exophone_number, status, priority, monitoring_flag, cache_ttl, created_by) VALUES
(1, 1, '+911140000001', 'active', 'P0', 1, 900,  'SYSTEM'),
(2, 1, '+911140000002', 'active', 'P1', 1, 1800, 'SYSTEM'),
(3, 1, '+911140000003', 'active', 'P1', 1, 1800, 'SYSTEM'),
(4, 1, '+911140000004', 'active', 'P2', 1, 3600, 'SYSTEM'),
(5, 2, '+912240000001', 'active', 'P0', 1, 900,  'SYSTEM'),
(6, 2, '+912240000002', 'active', 'P2', 1, 3600, 'SYSTEM')
ON DUPLICATE KEY UPDATE updated_at = NOW();

-- ============================================================
-- MONITORING JOBS
-- ============================================================

INSERT INTO monitoring_jobs (exophone_id, account_id, job_name, job_type, priority, frequency_minutes, timeout_seconds, retry_policy, next_run_at, last_run_at, last_status, created_by) VALUES
(1, 1, 'HB-P0-911140000001',     'HEARTBEAT', 'P0', 15, 30, '{"max_retries":3,"base_delay_ms":1000,"max_delay_ms":30000}', NOW(), NOW() - INTERVAL 15 MINUTE, 'SUCCESS', 'SYSTEM'),
(1, 1, 'CALLS-P0-911140000001',  'CALLS',     'P0', 15, 60, '{"max_retries":3,"base_delay_ms":1000,"max_delay_ms":30000}', NOW(), NOW() - INTERVAL 15 MINUTE, 'SUCCESS', 'SYSTEM'),
(1, 1, 'STRM-P0-911140000001',   'STREAMS',   'P0', 15, 30, '{"max_retries":3,"base_delay_ms":1000,"max_delay_ms":30000}', NOW(), NOW() - INTERVAL 15 MINUTE, 'SUCCESS', 'SYSTEM'),
(2, 1, 'HB-P1-911140000002',     'HEARTBEAT', 'P1', 30, 30, '{"max_retries":3,"base_delay_ms":1000,"max_delay_ms":30000}', NOW(), NOW() - INTERVAL 30 MINUTE, 'SUCCESS', 'SYSTEM'),
(2, 1, 'CALLS-P1-911140000002',  'CALLS',     'P1', 30, 60, '{"max_retries":3,"base_delay_ms":1000,"max_delay_ms":30000}', NOW(), NOW() - INTERVAL 30 MINUTE, 'FAILED',  'SYSTEM'),
(3, 1, 'HB-P1-911140000003',     'HEARTBEAT', 'P1', 30, 30, '{"max_retries":3,"base_delay_ms":1000,"max_delay_ms":30000}', NOW(), NOW() - INTERVAL 30 MINUTE, 'SUCCESS', 'SYSTEM'),
(4, 1, 'HB-P2-911140000004',     'HEARTBEAT', 'P2', 60, 30, '{"max_retries":3,"base_delay_ms":1000,"max_delay_ms":30000}', NOW(), NOW() - INTERVAL 60 MINUTE, 'SUCCESS', 'SYSTEM'),
(5, 2, 'HB-P0-912240000001',     'HEARTBEAT', 'P0', 15, 30, '{"max_retries":3,"base_delay_ms":1000,"max_delay_ms":30000}', NOW(), NOW() - INTERVAL 15 MINUTE, 'SUCCESS', 'SYSTEM'),
(5, 2, 'CALLS-P0-912240000001',  'CALLS',     'P0', 15, 60, '{"max_retries":3,"base_delay_ms":1000,"max_delay_ms":30000}', NOW(), NOW() - INTERVAL 15 MINUTE, 'SUCCESS', 'SYSTEM'),
(6, 2, 'HB-P2-912240000002',     'HEARTBEAT', 'P2', 60, 30, '{"max_retries":3,"base_delay_ms":1000,"max_delay_ms":30000}', NOW(), NOW() - INTERVAL 60 MINUTE, 'SUCCESS', 'SYSTEM');

-- ============================================================
-- JOB TRANSACTIONS (last 48 hours)
-- ============================================================

INSERT INTO job_transactions (transaction_id, job_id, account_id, exophone_id, job_type, status, api_latency_ms, cache_hit, retry_count, started_at, completed_at, created_by, updated_by) VALUES
('TXN-001', 1, 1, 1, 'HEARTBEAT', 'SUCCESS', 210,  0, 0, NOW() - INTERVAL 15  MINUTE, NOW() - INTERVAL 15  MINUTE + INTERVAL 1 SECOND, 'SYSTEM_SCHEDULER', 'SYSTEM_SCHEDULER'),
('TXN-002', 1, 1, 1, 'HEARTBEAT', 'SUCCESS', 198,  1, 0, NOW() - INTERVAL 30  MINUTE, NOW() - INTERVAL 30  MINUTE + INTERVAL 1 SECOND, 'SYSTEM_SCHEDULER', 'SYSTEM_SCHEDULER'),
('TXN-003', 1, 1, 1, 'HEARTBEAT', 'SUCCESS', 305,  0, 0, NOW() - INTERVAL 45  MINUTE, NOW() - INTERVAL 45  MINUTE + INTERVAL 1 SECOND, 'SYSTEM_SCHEDULER', 'SYSTEM_SCHEDULER'),
('TXN-004', 2, 1, 1, 'CALLS',     'SUCCESS', 450,  0, 0, NOW() - INTERVAL 15  MINUTE, NOW() - INTERVAL 15  MINUTE + INTERVAL 2 SECOND, 'SYSTEM_SCHEDULER', 'SYSTEM_SCHEDULER'),
('TXN-005', 3, 1, 1, 'STREAMS',   'SUCCESS', 180,  0, 0, NOW() - INTERVAL 15  MINUTE, NOW() - INTERVAL 15  MINUTE + INTERVAL 1 SECOND, 'SYSTEM_SCHEDULER', 'SYSTEM_SCHEDULER'),
('TXN-006', 4, 1, 2, 'HEARTBEAT', 'SUCCESS', 230,  0, 0, NOW() - INTERVAL 30  MINUTE, NOW() - INTERVAL 30  MINUTE + INTERVAL 1 SECOND, 'SYSTEM_SCHEDULER', 'SYSTEM_SCHEDULER'),
('TXN-007', 5, 1, 2, 'CALLS',     'FAILED',  NULL, 0, 3, NOW() - INTERVAL 30  MINUTE, NOW() - INTERVAL 30  MINUTE + INTERVAL 5 SECOND, 'SYSTEM_SCHEDULER', 'SYSTEM_SCHEDULER'),
('TXN-008', 6, 1, 3, 'HEARTBEAT', 'SUCCESS', 190,  0, 0, NOW() - INTERVAL 30  MINUTE, NOW() - INTERVAL 30  MINUTE + INTERVAL 1 SECOND, 'SYSTEM_SCHEDULER', 'SYSTEM_SCHEDULER'),
('TXN-009', 7, 1, 4, 'HEARTBEAT', 'SUCCESS', 260,  0, 0, NOW() - INTERVAL 60  MINUTE, NOW() - INTERVAL 60  MINUTE + INTERVAL 1 SECOND, 'SYSTEM_SCHEDULER', 'SYSTEM_SCHEDULER'),
('TXN-010', 8, 2, 5, 'HEARTBEAT', 'SUCCESS', 320,  0, 0, NOW() - INTERVAL 15  MINUTE, NOW() - INTERVAL 15  MINUTE + INTERVAL 1 SECOND, 'SYSTEM_SCHEDULER', 'SYSTEM_SCHEDULER'),
('TXN-011', 9, 2, 5, 'CALLS',     'SUCCESS', 510,  0, 0, NOW() - INTERVAL 15  MINUTE, NOW() - INTERVAL 15  MINUTE + INTERVAL 2 SECOND, 'SYSTEM_SCHEDULER', 'SYSTEM_SCHEDULER'),
('TXN-012', 1, 1, 1, 'HEARTBEAT', 'SUCCESS', 3500, 0, 0, NOW() - INTERVAL 2   HOUR,   NOW() - INTERVAL 2   HOUR   + INTERVAL 1 SECOND, 'SYSTEM_SCHEDULER', 'SYSTEM_SCHEDULER'),
('TXN-013', 1, 1, 1, 'HEARTBEAT', 'FAILED',  NULL, 0, 3, NOW() - INTERVAL 6   HOUR,   NOW() - INTERVAL 6   HOUR   + INTERVAL 10 SECOND,'SYSTEM_SCHEDULER', 'SYSTEM_SCHEDULER'),
('TXN-014', 8, 2, 5, 'HEARTBEAT', 'SUCCESS', 290,  0, 0, NOW() - INTERVAL 1   DAY,    NOW() - INTERVAL 1   DAY    + INTERVAL 1 SECOND, 'SYSTEM_SCHEDULER', 'SYSTEM_SCHEDULER');

-- ============================================================
-- HEARTBEAT METRICS
-- ============================================================

INSERT INTO heartbeat_metrics (transaction_id, account_id, status_type, incoming_affected, outgoing_affected, response_time_ms, raw_status, created_by) VALUES
('TXN-001', 1, 'OK',      0, 0,  210,  'OK',      'SYSTEM_METRICS'),
('TXN-002', 1, 'OK',      0, 0,  198,  'OK',      'SYSTEM_METRICS'),
('TXN-003', 1, 'OK',      0, 0,  305,  'OK',      'SYSTEM_METRICS'),
('TXN-006', 1, 'OK',      0, 0,  230,  'OK',      'SYSTEM_METRICS'),
('TXN-008', 1, 'OK',      0, 0,  190,  'OK',      'SYSTEM_METRICS'),
('TXN-009', 1, 'OK',      0, 0,  260,  'OK',      'SYSTEM_METRICS'),
('TXN-010', 2, 'OK',      0, 0,  320,  'OK',      'SYSTEM_METRICS'),
('TXN-012', 1, 'DEGRADED',1, 0,  3500, 'DEGRADED','SYSTEM_METRICS'),
('TXN-013', 1, 'OUTAGE',  1, 1,  NULL, 'OUTAGE',  'SYSTEM_METRICS'),
('TXN-014', 2, 'OK',      0, 0,  290,  'OK',      'SYSTEM_METRICS');

-- ============================================================
-- STREAM METRICS
-- ============================================================

INSERT INTO stream_metrics (transaction_id, account_id, exophone_id, active_streams, max_allowed_streams, utilization_percent, created_by) VALUES
('TXN-005', 1, 1, 12, 50, 24.00, 'SYSTEM_METRICS'),
('TXN-005', 1, 1, 8,  50, 16.00, 'SYSTEM_METRICS'),
('TXN-011', 2, 5, 45, 50, 90.00, 'SYSTEM_METRICS');

-- ============================================================
-- CALL LOGS  (last 2 days, varied statuses)
-- ============================================================

INSERT INTO call_logs (transaction_id, account_id, exophone_id, call_sid, call_from, call_to, status, direction, duration_sec, conversation_duration, start_time, end_time) VALUES
('TXN-004', 1, 1, 'CS001', '+919876500001', '+911140000001', 'completed', 'inbound',  125, 120, NOW() - INTERVAL 15 MINUTE - INTERVAL 125 SECOND, NOW() - INTERVAL 15 MINUTE),
('TXN-004', 1, 1, 'CS002', '+919876500002', '+911140000001', 'completed', 'inbound',  87,  82,  NOW() - INTERVAL 30 MINUTE - INTERVAL 87  SECOND, NOW() - INTERVAL 30 MINUTE),
('TXN-004', 1, 1, 'CS003', '+919876500003', '+911140000001', 'failed',    'outbound', 0,   0,   NOW() - INTERVAL 45 MINUTE,                         NOW() - INTERVAL 45 MINUTE),
('TXN-004', 1, 1, 'CS004', '+919876500004', '+911140000001', 'busy',      'inbound',  0,   0,   NOW() - INTERVAL 60 MINUTE,                         NOW() - INTERVAL 60 MINUTE),
('TXN-004', 1, 1, 'CS005', '+919876500005', '+911140000001', 'completed', 'inbound',  200, 195, NOW() - INTERVAL 90 MINUTE - INTERVAL 200 SECOND, NOW() - INTERVAL 90 MINUTE),
('TXN-004', 1, 1, 'CS006', '+919876500006', '+911140000001', 'completed', 'outbound', 310, 305, NOW() - INTERVAL 2  HOUR   - INTERVAL 310 SECOND, NOW() - INTERVAL 2  HOUR),
('TXN-004', 1, 1, 'CS007', '+919876500007', '+911140000001', 'no-answer', 'inbound',  0,   0,   NOW() - INTERVAL 3  HOUR,                           NOW() - INTERVAL 3  HOUR),
('TXN-004', 1, 1, 'CS008', '+919876500008', '+911140000001', 'completed', 'inbound',  55,  50,  NOW() - INTERVAL 4  HOUR   - INTERVAL 55  SECOND, NOW() - INTERVAL 4  HOUR),
('TXN-004', 1, 1, 'CS009', '+919876500009', '+911140000001', 'completed', 'inbound',  420, 415, NOW() - INTERVAL 5  HOUR   - INTERVAL 420 SECOND, NOW() - INTERVAL 5  HOUR),
('TXN-004', 1, 1, 'CS010', '+919876500010', '+911140000001', 'failed',    'outbound', 0,   0,   NOW() - INTERVAL 6  HOUR,                           NOW() - INTERVAL 6  HOUR),
('TXN-011', 2, 5, 'CS011', '+918887770001', '+912240000001', 'completed', 'inbound',  98,  93,  NOW() - INTERVAL 20 MINUTE - INTERVAL 98  SECOND, NOW() - INTERVAL 20 MINUTE),
('TXN-011', 2, 5, 'CS012', '+918887770002', '+912240000001', 'completed', 'inbound',  145, 140, NOW() - INTERVAL 40 MINUTE - INTERVAL 145 SECOND, NOW() - INTERVAL 40 MINUTE),
('TXN-011', 2, 5, 'CS013', '+918887770003', '+912240000001', 'failed',    'inbound',  0,   0,   NOW() - INTERVAL 1  HOUR,                           NOW() - INTERVAL 1  HOUR);

-- ============================================================
-- METRICS (availability, latency, etc.)
-- ============================================================

INSERT INTO metrics (transaction_id, exophone_id, account_id, metric_name, metric_value, metric_unit, aggregation_type, created_by) VALUES
('TXN-001', 1, 1, 'availability',    100.00, 'percent', 'INSTANT', 'SYSTEM_METRICS'),
('TXN-001', 1, 1, 'api_latency',     210.00, 'ms',      'INSTANT', 'SYSTEM_METRICS'),
('TXN-002', 1, 1, 'availability',    100.00, 'percent', 'INSTANT', 'SYSTEM_METRICS'),
('TXN-002', 1, 1, 'api_latency',     198.00, 'ms',      'INSTANT', 'SYSTEM_METRICS'),
('TXN-004', 1, 1, 'connected_calls', 7.00,   'count',   'SUM',     'SYSTEM_METRICS'),
('TXN-004', 1, 1, 'failed_calls',    2.00,   'count',   'SUM',     'SYSTEM_METRICS'),
('TXN-004', 1, 1, 'avg_duration',    171.00, 'seconds', 'AVG',     'SYSTEM_METRICS'),
('TXN-005', 1, 1, 'api_latency',     180.00, 'ms',      'INSTANT', 'SYSTEM_METRICS'),
('TXN-006', 2, 1, 'availability',    100.00, 'percent', 'INSTANT', 'SYSTEM_METRICS'),
('TXN-007', 2, 1, 'failed_calls',    1.00,   'count',   'SUM',     'SYSTEM_METRICS'),
('TXN-010', 5, 2, 'availability',    100.00, 'percent', 'INSTANT', 'SYSTEM_METRICS'),
('TXN-010', 5, 2, 'api_latency',     320.00, 'ms',      'INSTANT', 'SYSTEM_METRICS'),
('TXN-011', 5, 2, 'connected_calls', 2.00,   'count',   'SUM',     'SYSTEM_METRICS'),
('TXN-011', 5, 2, 'failed_calls',    1.00,   'count',   'SUM',     'SYSTEM_METRICS'),
('TXN-012', 1, 1, 'api_latency',     3500.00,'ms',      'INSTANT', 'SYSTEM_METRICS'),
('TXN-013', 1, 1, 'availability',    0.00,   'percent', 'INSTANT', 'SYSTEM_METRICS');

-- ============================================================
-- SNAPSHOT TABLES
-- ============================================================

INSERT INTO exophone_health_snapshot (exophone_id, account_id, heartbeat_status, availability_percent, last_api_latency_ms, active_streams, last_checked_at) VALUES
(1, 1, 'OK',      99.50, 210,  12, NOW() - INTERVAL 15 MINUTE),
(2, 1, 'OK',      98.00, 230,  0,  NOW() - INTERVAL 30 MINUTE),
(3, 1, 'OK',      100.00,190,  0,  NOW() - INTERVAL 30 MINUTE),
(4, 1, 'OK',      100.00,260,  0,  NOW() - INTERVAL 60 MINUTE),
(5, 2, 'OK',      100.00,320,  45, NOW() - INTERVAL 15 MINUTE),
(6, 2, 'UNKNOWN', 0.00,  NULL, 0,  NULL)
ON DUPLICATE KEY UPDATE
  heartbeat_status     = VALUES(heartbeat_status),
  availability_percent = VALUES(availability_percent),
  last_api_latency_ms  = VALUES(last_api_latency_ms),
  active_streams       = VALUES(active_streams),
  last_checked_at      = VALUES(last_checked_at),
  updated_at           = NOW();

INSERT INTO call_metrics_snapshot (snapshot_date, account_id, exophone_id, total_calls, failed_calls, connected_calls, busy_calls, avg_duration_sec, success_rate) VALUES
(CURDATE(),              1, 1, 10, 2, 7, 1, 171.00, 70.00),
(CURDATE() - INTERVAL 1 DAY, 1, 1, 18, 3, 14, 1, 185.00, 77.78),
(CURDATE() - INTERVAL 2 DAY, 1, 1, 22, 1, 20, 1, 160.00, 90.91),
(CURDATE(),              2, 5,  3, 1, 2, 0,  121.50, 66.67),
(CURDATE() - INTERVAL 1 DAY, 2, 5,  8, 2, 6, 0,  135.00, 75.00)
ON DUPLICATE KEY UPDATE
  total_calls     = VALUES(total_calls),
  failed_calls    = VALUES(failed_calls),
  connected_calls = VALUES(connected_calls),
  busy_calls      = VALUES(busy_calls),
  avg_duration_sec= VALUES(avg_duration_sec),
  success_rate    = VALUES(success_rate),
  updated_at      = NOW();

INSERT INTO stream_utilization_snapshot (snapshot_hour, account_id, exophone_id, avg_active_streams, max_active_streams, utilization_percent) VALUES
(DATE_FORMAT(NOW() - INTERVAL 1 HOUR, '%Y-%m-%d %H:00:00'), 1, 1, 10.50, 14, 28.00),
(DATE_FORMAT(NOW() - INTERVAL 2 HOUR, '%Y-%m-%d %H:00:00'), 1, 1, 8.00,  12, 24.00),
(DATE_FORMAT(NOW() - INTERVAL 1 HOUR, '%Y-%m-%d %H:00:00'), 2, 5, 42.00, 47, 94.00),
(DATE_FORMAT(NOW() - INTERVAL 2 HOUR, '%Y-%m-%d %H:00:00'), 2, 5, 38.00, 45, 90.00)
ON DUPLICATE KEY UPDATE
  avg_active_streams  = VALUES(avg_active_streams),
  max_active_streams  = VALUES(max_active_streams),
  utilization_percent = VALUES(utilization_percent);

INSERT INTO account_dashboard_snapshot (snapshot_date, account_id, total_exophones, active_exophones, total_calls, failed_calls, success_rate, avg_latency_ms, alert_count) VALUES
(CURDATE(),              1, 4, 4, 10, 2, 70.00, 248, 2),
(CURDATE() - INTERVAL 1 DAY, 1, 4, 4, 18, 3, 77.78, 215, 1),
(CURDATE(),              2, 2, 2,  3, 1, 66.67, 320, 1)
ON DUPLICATE KEY UPDATE
  total_calls     = VALUES(total_calls),
  failed_calls    = VALUES(failed_calls),
  success_rate    = VALUES(success_rate),
  avg_latency_ms  = VALUES(avg_latency_ms),
  alert_count     = VALUES(alert_count),
  updated_at      = NOW();

-- ============================================================
-- ALERTS
-- ============================================================

INSERT INTO alerts (transaction_id, exophone_id, account_id, alert_type, severity, alert_status, message, channel, triggered_at) VALUES
('TXN-013', 1, 1, 'HEARTBEAT_FAILURE', 'CRITICAL', 'OPEN',
 'Exophone +911140000001 heartbeat failed after 3 retries. Outage detected.',
 'SLACK', NOW() - INTERVAL 6 HOUR),
('TXN-007', 2, 1, 'API_FAILURE', 'WARNING', 'ACKNOWLEDGED',
 'Calls API failed for exophone +911140000002. Retry exhausted.',
 'SLACK', NOW() - INTERVAL 30 MINUTE),
('TXN-012', 1, 1, 'HIGH_LATENCY', 'WARNING', 'OPEN',
 'Exophone +911140000001 API latency 3500ms exceeds threshold of 3000ms.',
 'SLACK', NOW() - INTERVAL 2 HOUR),
('TXN-005', 5, 2, 'VENDOR_THROTTLING', 'CRITICAL', 'OPEN',
 'Exophone +912240000001 stream utilisation at 90% (45/50 active streams).',
 'SLACK', NOW() - INTERVAL 15 MINUTE),
(NULL, 6, 2, 'EXOPHONE_DOWN', 'CRITICAL', 'OPEN',
 'Exophone +912240000002 has not been checked in over 1 hour.',
 'SLACK', NOW() - INTERVAL 90 MINUTE);

UPDATE alerts SET acknowledged_at = NOW() - INTERVAL 20 MINUTE, alert_status = 'ACKNOWLEDGED', updated_by = 'ops-team'
WHERE transaction_id = 'TXN-007';

-- ============================================================
-- ALERT DISPATCH LOGS
-- ============================================================

INSERT INTO alert_dispatch_logs (alert_id, channel, status, response_code, dispatched_at) VALUES
(1, 'SLACK', 'SENT',   200, NOW() - INTERVAL 6 HOUR),
(2, 'SLACK', 'SENT',   200, NOW() - INTERVAL 30 MINUTE),
(3, 'SLACK', 'SENT',   200, NOW() - INTERVAL 2 HOUR),
(4, 'SLACK', 'SENT',   200, NOW() - INTERVAL 15 MINUTE),
(5, 'SLACK', 'FAILED', 500, NOW() - INTERVAL 90 MINUTE);
