-- ============================================================
-- Exotel Monitoring Platform — Complete Database Schema
-- ============================================================

CREATE DATABASE IF NOT EXISTS exotel_monitoring;
USE exotel_monitoring;

-- ============================================================
-- MASTER TABLES
-- ============================================================

CREATE TABLE IF NOT EXISTS accounts (
    id          BIGINT PRIMARY KEY AUTO_INCREMENT,
    account_name VARCHAR(255) NOT NULL,
    sid          VARCHAR(255) NOT NULL UNIQUE,
    api_key      VARCHAR(512) NOT NULL COMMENT 'AES-encrypted',
    api_token    VARCHAR(512) NOT NULL COMMENT 'AES-encrypted',
    subdomain    VARCHAR(255) NOT NULL,
    cluster      VARCHAR(100) NOT NULL DEFAULT 'in1',
    config_json  JSON,
    is_active    TINYINT NOT NULL DEFAULT 1,
    created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by   VARCHAR(100) NOT NULL DEFAULT 'SYSTEM',
    updated_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    updated_by   VARCHAR(100) NOT NULL DEFAULT 'SYSTEM',
    is_deleted   TINYINT NOT NULL DEFAULT 0,
    -- No separate idx_accounts_sid: the UNIQUE constraint on sid already creates an implicit index
    INDEX idx_accounts_active (is_active)
);

CREATE TABLE IF NOT EXISTS exophones (
    id              BIGINT PRIMARY KEY AUTO_INCREMENT,
    account_id      BIGINT NOT NULL,
    exophone_number VARCHAR(20) NOT NULL,
    status          VARCHAR(50) NOT NULL DEFAULT 'active',
    priority        VARCHAR(10) NOT NULL DEFAULT 'P1' COMMENT 'P0/P1/P2/P3',
    monitoring_flag      TINYINT NOT NULL DEFAULT 1,
    is_cron_applicable   TINYINT NOT NULL DEFAULT 0,
    cache_ttl            INT NOT NULL DEFAULT 900 COMMENT 'TTL in seconds',
    metadata_json   JSON,
    last_synced_at  TIMESTAMP NULL,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by      VARCHAR(100) NOT NULL DEFAULT 'SYSTEM',
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    updated_by      VARCHAR(100) NOT NULL DEFAULT 'SYSTEM',
    is_deleted      TINYINT NOT NULL DEFAULT 0,
    -- idx_exophones_account dropped: covered by idx_exophones_account_deleted (account_id, is_deleted)
    INDEX idx_exophones_account_deleted (account_id, is_deleted),
    INDEX idx_exophones_number (exophone_number),
    INDEX idx_exophones_priority (priority),
    INDEX idx_exophones_status (status),
    INDEX idx_exophones_monitoring (monitoring_flag, is_deleted),
    INDEX idx_exophones_cron (is_cron_applicable, is_deleted),
    UNIQUE KEY uk_account_number (account_id, exophone_number)
);

CREATE TABLE IF NOT EXISTS monitoring_jobs (
    id                BIGINT PRIMARY KEY AUTO_INCREMENT,
    exophone_id       BIGINT NOT NULL,
    account_id        BIGINT NOT NULL,
    job_name          VARCHAR(255) NOT NULL,
    job_type          VARCHAR(100) NOT NULL DEFAULT 'HEARTBEAT' COMMENT 'HEARTBEAT/CALLS/STREAMS/INVENTORY_SYNC/SNAPSHOT/CLEANUP',
    priority          VARCHAR(10) NOT NULL DEFAULT 'P1',
    cron_expression   VARCHAR(255),
    frequency_minutes INT NOT NULL DEFAULT 30,
    timeout_seconds   INT NOT NULL DEFAULT 30,
    retry_policy      JSON COMMENT '{"max_retries":3,"base_delay_ms":1000,"max_delay_ms":30000}',
    next_run_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_run_at       TIMESTAMP NULL,
    last_status       VARCHAR(50) DEFAULT NULL,
    is_active         TINYINT NOT NULL DEFAULT 1,
    created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by        VARCHAR(100) NOT NULL DEFAULT 'SYSTEM',
    updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    updated_by        VARCHAR(100) NOT NULL DEFAULT 'SYSTEM',
    is_deleted        TINYINT NOT NULL DEFAULT 0,
    -- idx_jobs_next_run dropped: covered by idx_jobs_scheduler (next_run_at, is_active, is_deleted)
    INDEX idx_jobs_scheduler (next_run_at, is_active, is_deleted),
    INDEX idx_jobs_priority (priority),
    INDEX idx_jobs_exophone (exophone_id)
);

-- ============================================================
-- TRANSACTION TABLES
-- ============================================================

CREATE TABLE IF NOT EXISTS job_transactions (
    id             BIGINT PRIMARY KEY AUTO_INCREMENT,
    transaction_id VARCHAR(64) NOT NULL UNIQUE,
    job_id         BIGINT NOT NULL,
    account_id     BIGINT NOT NULL,
    exophone_id    BIGINT NOT NULL,
    job_type       VARCHAR(100) NOT NULL,
    status         VARCHAR(50) NOT NULL DEFAULT 'RUNNING' COMMENT 'RUNNING/SUCCESS/FAILED/RETRYING/TIMEOUT',
    api_latency_ms BIGINT DEFAULT NULL,
    cache_hit      TINYINT NOT NULL DEFAULT 0,
    retry_count    INT NOT NULL DEFAULT 0,
    error_message  TEXT,
    started_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at   TIMESTAMP NULL,
    created_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by     VARCHAR(100) NOT NULL DEFAULT 'SYSTEM_SCHEDULER',
    updated_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    updated_by     VARCHAR(100) NOT NULL DEFAULT 'SYSTEM_SCHEDULER',
    INDEX idx_txn_id (transaction_id),
    INDEX idx_txn_job (job_id),
    INDEX idx_txn_exophone (exophone_id),
    INDEX idx_txn_status (status),
    INDEX idx_txn_started (started_at),
    -- Composite index for reprocess queries: WHERE exophone_id=? AND DATE(started_at)=?
    INDEX idx_txn_exophone_started (exophone_id, started_at)
);

CREATE TABLE IF NOT EXISTS job_responses (
    id              BIGINT PRIMARY KEY AUTO_INCREMENT,
    transaction_id  VARCHAR(64) NOT NULL,
    response_type   VARCHAR(100) NOT NULL COMMENT 'HEARTBEAT/CALLS/STREAMS',
    http_status     INT,
    raw_response    MEDIUMTEXT,
    parsed_response JSON,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_jr_txn (transaction_id),
    INDEX idx_jr_type (response_type)
);

CREATE TABLE IF NOT EXISTS call_logs (
    id                    BIGINT PRIMARY KEY AUTO_INCREMENT,
    transaction_id        VARCHAR(64) NOT NULL,
    account_id            BIGINT NOT NULL,
    exophone_id           BIGINT NOT NULL,
    call_sid              VARCHAR(255),
    parent_call_sid       VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'non-empty = Leg-2 call bridged from Leg-1',
    leg_number            INT NOT NULL DEFAULT 0 COMMENT '1=A-leg (agent/IVR), 2=B-leg (customer)',
    call_from             VARCHAR(20),
    call_to               VARCHAR(20),
    status                VARCHAR(50) COMMENT 'completed/failed/busy/no-answer',
    direction             VARCHAR(20) COMMENT 'inbound/outbound',
    duration_sec          INT DEFAULT 0,
    conversation_duration INT DEFAULT 0 COMMENT '0=agent never answered; >0=actual talk time in seconds',
    start_time            TIMESTAMP NULL,
    end_time              TIMESTAMP NULL,
    recording_url         VARCHAR(512),
    leg1_status           VARCHAR(50) NULL COMMENT 'Exotel Details.Leg1Status (requires details=true)',
    leg2_status           VARCHAR(50) NULL COMMENT 'empty=no agent routing (Leg1 drop); canceled=agent no-answer (Leg2 drop)',
    created_at            TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    -- uk_cl_exophone_callsid prevents duplicate ingestion from overlapping 15-min fetch windows
    UNIQUE KEY uk_cl_exophone_callsid (exophone_id, call_sid),
    INDEX idx_cl_account (account_id),
    -- idx_cl_exophone dropped: covered by uk_cl_exophone_callsid and idx_cl_exophone_time
    INDEX idx_cl_exophone_time (exophone_id, start_time),   -- snapshot recompute: WHERE exophone_id=? AND DATE(start_time)=?
    INDEX idx_cl_exophone_status (exophone_id, status),     -- status breakdown per exophone
    INDEX idx_cl_status (status),                           -- account-wide status queries
    INDEX idx_cl_start_time (start_time)                    -- time-range queries across all exophones
);

CREATE TABLE IF NOT EXISTS call_flow_events (
    id           BIGINT PRIMARY KEY AUTO_INCREMENT,
    call_sid     VARCHAR(255),
    flow_id      VARCHAR(255),
    exophone_id  BIGINT,
    call_from    VARCHAR(20),
    call_to      VARCHAR(20),
    call_status  VARCHAR(50),
    raw_payload  JSON,
    created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_cfe_call_sid (call_sid),
    INDEX idx_cfe_exophone (exophone_id)
);

-- ============================================================
-- METRICS TABLES
-- ============================================================

CREATE TABLE IF NOT EXISTS heartbeat_metrics (
    id                 BIGINT PRIMARY KEY AUTO_INCREMENT,
    transaction_id     VARCHAR(64) NOT NULL,
    account_id         BIGINT NOT NULL,
    status_type        VARCHAR(50) NOT NULL COMMENT 'OK/DEGRADED/OUTAGE',
    incoming_affected  TINYINT NOT NULL DEFAULT 0,
    outgoing_affected  TINYINT NOT NULL DEFAULT 0,
    response_time_ms   BIGINT DEFAULT NULL,
    raw_status         VARCHAR(255),
    created_at         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by         VARCHAR(100) NOT NULL DEFAULT 'SYSTEM_METRICS',
    INDEX idx_hb_account (account_id),
    INDEX idx_hb_status (status_type),
    INDEX idx_hb_created (created_at)
);

CREATE TABLE IF NOT EXISTS stream_metrics (
    id                   BIGINT PRIMARY KEY AUTO_INCREMENT,
    transaction_id       VARCHAR(64) NOT NULL,
    account_id           BIGINT NOT NULL,
    exophone_id          BIGINT NOT NULL,
    active_streams       INT NOT NULL DEFAULT 0,
    max_allowed_streams  INT NOT NULL DEFAULT 0,
    utilization_percent  DECIMAL(5,2) DEFAULT 0.00,
    created_at           TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by           VARCHAR(100) NOT NULL DEFAULT 'SYSTEM_METRICS',
    INDEX idx_sm_account (account_id),
    INDEX idx_sm_exophone (exophone_id),
    INDEX idx_sm_created (created_at)
);

CREATE TABLE IF NOT EXISTS metrics (
    id                 BIGINT PRIMARY KEY AUTO_INCREMENT,
    transaction_id     VARCHAR(64) NOT NULL,
    exophone_id        BIGINT NOT NULL,
    account_id         BIGINT NOT NULL,
    metric_name        VARCHAR(100) NOT NULL COMMENT 'availability/failed_calls/connected_calls/avg_duration/api_latency/cache_hit_ratio',
    metric_value       DECIMAL(15,4) NOT NULL,
    metric_unit        VARCHAR(50) DEFAULT NULL COMMENT 'percent/ms/count/seconds',
    metric_timestamp   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    aggregation_type   VARCHAR(50) DEFAULT 'INSTANT' COMMENT 'INSTANT/AVG/SUM/MAX',
    created_at         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by         VARCHAR(100) NOT NULL DEFAULT 'SYSTEM_METRICS',
    INDEX idx_m_exophone (exophone_id),
    INDEX idx_m_name (metric_name),
    INDEX idx_m_timestamp (metric_timestamp)
);

-- ============================================================
-- SNAPSHOT TABLES (dashboard-serving, never cleaned)
-- ============================================================

CREATE TABLE IF NOT EXISTS call_metrics_snapshot (
    id                   BIGINT PRIMARY KEY AUTO_INCREMENT,
    snapshot_date        DATE NOT NULL,
    account_id           BIGINT NOT NULL,
    exophone_id          BIGINT NOT NULL,
    total_calls          INT NOT NULL DEFAULT 0,
    leg1_total           INT NOT NULL DEFAULT 0 COMMENT 'A-leg count',
    leg1_drops           INT NOT NULL DEFAULT 0 COMMENT 'completed + conv_dur=0 + leg2_status=empty (customer IVR abandon)',
    leg2_total           INT NOT NULL DEFAULT 0 COMMENT 'B-leg count (panel mode only)',
    leg2_drops           INT NOT NULL DEFAULT 0 COMMENT 'completed + conv_dur=0 + leg2_status!=empty (agent no-answer)',
    connected_calls      INT NOT NULL DEFAULT 0 COMMENT 'conversation_duration > 0 (both parties bridged)',
    dropped_calls        INT NOT NULL DEFAULT 0 COMMENT 'leg1_drops + leg2_drops',
    failed_calls         INT NOT NULL DEFAULT 0 COMMENT 'status=failed (network failure)',
    no_answer_calls      INT NOT NULL DEFAULT 0 COMMENT 'status=no-answer (missed)',
    busy_calls           INT NOT NULL DEFAULT 0 COMMENT 'status=busy',
    canceled_calls       INT NOT NULL DEFAULT 0 COMMENT 'status=canceled',
    other_calls          INT NOT NULL DEFAULT 0 COMMENT 'any unrecognised Exotel status',
    avg_duration_sec     DECIMAL(10,2) DEFAULT 0.00,
    answer_rate          DECIMAL(5,2) DEFAULT 0.00 COMMENT 'connected/total * 100',
    drop_rate            DECIMAL(5,2) DEFAULT 0.00 COMMENT 'leg2_drops/total * 100 (agent no-answer rate)',
    leg1_drop_rate       DECIMAL(5,2) DEFAULT 0.00 COMMENT 'leg1_drops/total * 100 (IVR abandon rate)',
    hourly_distribution  JSON COMMENT '24-element array [h0, h1, ..., h23] with call counts per hour',
    peak_hour            TINYINT NOT NULL DEFAULT 0 COMMENT '0-23 hour with highest call volume',
    success_rate         DECIMAL(5,2) DEFAULT 0.00 COMMENT 'alias for answer_rate; kept for API backward compatibility',
    created_at           TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at           TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_snapshot_date_exophone (snapshot_date, exophone_id),
    INDEX idx_cms_account (account_id),
    INDEX idx_cms_date (snapshot_date),
    -- Composite index for account+date lookups (dashboard aggregation, export queries)
    INDEX idx_cms_account_date (account_id, snapshot_date)
);

CREATE TABLE IF NOT EXISTS exophone_health_snapshot (
    id                   BIGINT PRIMARY KEY AUTO_INCREMENT,
    exophone_id          BIGINT NOT NULL,
    account_id           BIGINT NOT NULL,
    heartbeat_status     VARCHAR(50) NOT NULL DEFAULT 'UNKNOWN',
    availability_percent DECIMAL(5,2) DEFAULT 0.00,
    last_api_latency_ms  BIGINT DEFAULT NULL,
    active_streams       INT DEFAULT 0,
    last_checked_at      TIMESTAMP NULL,
    created_at           TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at           TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_ehs_exophone (exophone_id),
    INDEX idx_ehs_account (account_id),
    INDEX idx_ehs_status (heartbeat_status)
);

CREATE TABLE IF NOT EXISTS stream_utilization_snapshot (
    id                  BIGINT PRIMARY KEY AUTO_INCREMENT,
    snapshot_hour       DATETIME NOT NULL,
    account_id          BIGINT NOT NULL,
    exophone_id         BIGINT NOT NULL,
    avg_active_streams  DECIMAL(10,2) DEFAULT 0.00,
    max_active_streams  INT DEFAULT 0,
    utilization_percent DECIMAL(5,2) DEFAULT 0.00,
    created_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_sus_hour_exophone (snapshot_hour, exophone_id),
    INDEX idx_sus_account (account_id)
);

CREATE TABLE IF NOT EXISTS account_dashboard_snapshot (
    id                BIGINT PRIMARY KEY AUTO_INCREMENT,
    snapshot_date     DATE NOT NULL,
    account_id        BIGINT NOT NULL,
    total_exophones   INT DEFAULT 0,
    active_exophones  INT DEFAULT 0,
    total_calls       INT DEFAULT 0,
    failed_calls      INT DEFAULT 0,
    success_rate      DECIMAL(5,2) DEFAULT 0.00,
    avg_latency_ms    BIGINT DEFAULT NULL,
    alert_count       INT DEFAULT 0,
    created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_ads_date_account (snapshot_date, account_id)
);

-- ============================================================
-- ALERTING TABLES
-- ============================================================

CREATE TABLE IF NOT EXISTS alerts (
    id              BIGINT PRIMARY KEY AUTO_INCREMENT,
    transaction_id  VARCHAR(64),
    exophone_id     BIGINT,
    account_id      BIGINT,
    alert_type      VARCHAR(100) NOT NULL COMMENT 'API_FAILURE/HEARTBEAT_FAILURE/HIGH_LATENCY/RETRY_EXHAUSTED/VENDOR_THROTTLING/EXOPHONE_DOWN/CACHE_FAILURE',
    severity        VARCHAR(50) NOT NULL DEFAULT 'WARNING' COMMENT 'CRITICAL/WARNING/INFO',
    alert_status    VARCHAR(50) NOT NULL DEFAULT 'OPEN' COMMENT 'OPEN/ACKNOWLEDGED/RESOLVED',
    message         TEXT NOT NULL,
    channel         VARCHAR(100) NOT NULL COMMENT 'SLACK/EMAIL/WEBHOOK',
    triggered_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    acknowledged_at TIMESTAMP NULL,
    resolved_at     TIMESTAMP NULL,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by      VARCHAR(100) NOT NULL DEFAULT 'SYSTEM_ALERT_ENGINE',
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    updated_by      VARCHAR(100) NOT NULL DEFAULT 'SYSTEM_ALERT_ENGINE',
    INDEX idx_alerts_exophone (exophone_id),
    INDEX idx_alerts_status (alert_status),
    INDEX idx_alerts_severity (severity),
    INDEX idx_alerts_triggered (triggered_at)
);

CREATE TABLE IF NOT EXISTS alert_dispatch_logs (
    id             BIGINT PRIMARY KEY AUTO_INCREMENT,
    alert_id       BIGINT NOT NULL,
    channel        VARCHAR(100) NOT NULL,
    status         VARCHAR(50) NOT NULL COMMENT 'SENT/FAILED/PENDING',
    response_code  INT,
    error_message  TEXT,
    dispatched_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_adl_alert (alert_id),
    INDEX idx_adl_status (status)
);

CREATE TABLE IF NOT EXISTS config_change_logs (
    id            BIGINT PRIMARY KEY AUTO_INCREMENT,
    entity_type   VARCHAR(100) NOT NULL,
    entity_id     BIGINT NOT NULL,
    old_value     JSON,
    new_value     JSON,
    changed_by    VARCHAR(100) NOT NULL,
    changed_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    change_reason VARCHAR(255),
    INDEX idx_ccl_entity (entity_type, entity_id),
    INDEX idx_ccl_changed_at (changed_at)
);

-- ============================================================
-- CONFIGURATION TABLES
-- ============================================================

CREATE TABLE IF NOT EXISTS priority_config (
    id                BIGINT PRIMARY KEY AUTO_INCREMENT,
    priority          VARCHAR(10) NOT NULL COMMENT 'P0/P1/P2/P3',
    frequency_minutes INT NOT NULL DEFAULT 60 COMMENT 'How often to poll each exophone at this priority',
    description       VARCHAR(255),
    updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    updated_by        VARCHAR(100) NOT NULL DEFAULT 'SYSTEM',
    UNIQUE KEY uk_priority (priority)
);

-- Default priority polling frequencies
INSERT INTO priority_config (priority, frequency_minutes, description, updated_by) VALUES
    ('P0', 15,  'Critical VNs — polled every 15 minutes',  'SYSTEM'),
    ('P1', 30,  'High-priority VNs — polled every 30 min', 'SYSTEM'),
    ('P2', 60,  'Standard VNs — polled every hour',        'SYSTEM'),
    ('P3', 120, 'Low-priority VNs — polled every 2 hours', 'SYSTEM')
ON DUPLICATE KEY UPDATE updated_at = NOW();

-- ============================================================
-- SEED DATA — Example Account + Exophone + Jobs
-- ============================================================

INSERT INTO accounts (account_name, sid, api_key, api_token, subdomain, cluster, created_by)
VALUES ('Demo Account', 'DEMO_SID_001', 'ENCRYPTED_KEY', 'ENCRYPTED_TOKEN', 'demo', 'in1', 'SYSTEM')
ON DUPLICATE KEY UPDATE updated_at = NOW();

INSERT INTO exophones (account_id, exophone_number, status, priority, monitoring_flag, cache_ttl, created_by)
VALUES (1, '+911234567890', 'active', 'P0', 1, 900, 'SYSTEM')
ON DUPLICATE KEY UPDATE updated_at = NOW();

INSERT INTO monitoring_jobs (exophone_id, account_id, job_name, job_type, priority, frequency_minutes, timeout_seconds, retry_policy, next_run_at, created_by)
VALUES
  (1, 1, 'Heartbeat Monitor P0', 'HEARTBEAT', 'P0', 15, 30, '{"max_retries":3,"base_delay_ms":1000,"max_delay_ms":30000}', NOW(), 'SYSTEM'),
  (1, 1, 'Calls Sync P0',        'CALLS',     'P0', 15, 60, '{"max_retries":3,"base_delay_ms":1000,"max_delay_ms":30000}', NOW(), 'SYSTEM'),
  (1, 1, 'Active Streams P0',    'STREAMS',   'P0', 15, 30, '{"max_retries":3,"base_delay_ms":1000,"max_delay_ms":30000}', NOW(), 'SYSTEM');
