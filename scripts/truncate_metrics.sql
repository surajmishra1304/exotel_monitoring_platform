-- truncate_metrics.sql
-- Clears all reporting/transaction data so a clean backfill can be run.
-- Core config tables (accounts, exophones, monitoring_jobs, priority_config,
-- feature_flags, settings) are NOT touched.
--
-- Run with:
--   mysql -u <user> -p exotel_monitoring < scripts/truncate_metrics.sql

SET FOREIGN_KEY_CHECKS = 0;

TRUNCATE TABLE call_logs;
TRUNCATE TABLE job_responses;
TRUNCATE TABLE job_transactions;
TRUNCATE TABLE call_metrics_snapshot;
TRUNCATE TABLE account_dashboard_snapshot;
TRUNCATE TABLE metrics;
TRUNCATE TABLE heartbeat_metrics;
TRUNCATE TABLE stream_metrics;
TRUNCATE TABLE exophone_health_snapshot;
TRUNCATE TABLE stream_utilization_snapshot;
TRUNCATE TABLE alerts;

SET FOREIGN_KEY_CHECKS = 1;

-- Reset monitoring job cursors so the next scheduler run uses the
-- frequency_minutes fallback instead of a stale last_run_at window.
UPDATE monitoring_jobs SET last_run_at = NULL;

SELECT 'truncate complete' AS status;
