-- ============================================================
-- Migration 001 — Leg status & conversation duration columns
-- Idempotent: uses ADD COLUMN IF NOT EXISTS (MySQL 8.0+)
-- Apply once on existing databases.
-- ============================================================

USE exotel_monitoring;

-- call_logs: add leg classification columns
ALTER TABLE call_logs
  ADD COLUMN IF NOT EXISTS parent_call_sid  VARCHAR(255) NOT NULL DEFAULT '' AFTER call_sid,
  ADD COLUMN IF NOT EXISTS leg_number       INT          NOT NULL DEFAULT 0  AFTER parent_call_sid,
  ADD COLUMN IF NOT EXISTS leg1_status      VARCHAR(50)  NULL                AFTER recording_url,
  ADD COLUMN IF NOT EXISTS leg2_status      VARCHAR(50)  NULL                AFTER leg1_status;

-- call_metrics_snapshot: add leg-level breakdown columns
ALTER TABLE call_metrics_snapshot
  ADD COLUMN IF NOT EXISTS leg1_total     INT          NOT NULL DEFAULT 0    AFTER total_calls,
  ADD COLUMN IF NOT EXISTS leg1_drops     INT          NOT NULL DEFAULT 0    AFTER leg1_total,
  ADD COLUMN IF NOT EXISTS leg2_total     INT          NOT NULL DEFAULT 0    AFTER leg1_drops,
  ADD COLUMN IF NOT EXISTS leg2_drops     INT          NOT NULL DEFAULT 0    AFTER leg2_total,
  ADD COLUMN IF NOT EXISTS leg1_drop_rate DECIMAL(5,2) NOT NULL DEFAULT 0.00 AFTER leg2_drops;

-- exophones: add cron-applicable flag (scheduler opt-in)
ALTER TABLE exophones
  ADD COLUMN IF NOT EXISTS is_cron_applicable TINYINT NOT NULL DEFAULT 0 AFTER monitoring_flag;
