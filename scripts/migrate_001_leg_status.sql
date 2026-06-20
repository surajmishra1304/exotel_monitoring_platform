-- ============================================================
-- Migration 001 — Leg status, conversation duration, and all
--                 leg-level breakdown columns
-- Idempotent: uses ADD COLUMN IF NOT EXISTS (MySQL 8.0+)
-- Apply once on existing databases before deploying this build.
-- Safe to re-run — existing columns are silently skipped.
-- ============================================================

USE exotel_monitoring;

-- ── call_logs: leg classification + conversation duration ──────────────────
ALTER TABLE call_logs
  ADD COLUMN IF NOT EXISTS parent_call_sid      VARCHAR(255) NOT NULL DEFAULT '' AFTER call_sid,
  ADD COLUMN IF NOT EXISTS leg_number           INT          NOT NULL DEFAULT 0  AFTER parent_call_sid,
  ADD COLUMN IF NOT EXISTS conversation_duration INT         NOT NULL DEFAULT 0  AFTER duration_sec,
  ADD COLUMN IF NOT EXISTS leg1_status          VARCHAR(50)  NULL                AFTER recording_url,
  ADD COLUMN IF NOT EXISTS leg2_status          VARCHAR(50)  NULL                AFTER leg1_status;

-- ── exophones: scheduler opt-in flag + per-exophone skip_call_logs ─────────
ALTER TABLE exophones
  ADD COLUMN IF NOT EXISTS is_cron_applicable   TINYINT NOT NULL DEFAULT 0 AFTER monitoring_flag,
  ADD COLUMN IF NOT EXISTS skip_call_logs       TINYINT NOT NULL DEFAULT 0 AFTER is_cron_applicable;

-- ── call_metrics_snapshot: leg totals, drop counts, drop rates, ────────────
--    per-leg status breakdown (all required by UpsertCallMetricsSnapshot)
ALTER TABLE call_metrics_snapshot
  ADD COLUMN IF NOT EXISTS leg1_total      INT          NOT NULL DEFAULT 0    AFTER total_calls,
  ADD COLUMN IF NOT EXISTS leg1_drops      INT          NOT NULL DEFAULT 0    AFTER leg1_total,
  ADD COLUMN IF NOT EXISTS leg2_total      INT          NOT NULL DEFAULT 0    AFTER leg1_drops,
  ADD COLUMN IF NOT EXISTS leg2_drops      INT          NOT NULL DEFAULT 0    AFTER leg2_total,
  ADD COLUMN IF NOT EXISTS leg1_drop_rate  DECIMAL(5,2) NOT NULL DEFAULT 0.00 AFTER leg2_drops,
  ADD COLUMN IF NOT EXISTS leg1_no_answer  INT          NOT NULL DEFAULT 0    AFTER peak_hour,
  ADD COLUMN IF NOT EXISTS leg1_busy       INT          NOT NULL DEFAULT 0    AFTER leg1_no_answer,
  ADD COLUMN IF NOT EXISTS leg1_failed     INT          NOT NULL DEFAULT 0    AFTER leg1_busy,
  ADD COLUMN IF NOT EXISTS leg2_no_answer  INT          NOT NULL DEFAULT 0    AFTER leg1_failed,
  ADD COLUMN IF NOT EXISTS leg2_busy       INT          NOT NULL DEFAULT 0    AFTER leg2_no_answer,
  ADD COLUMN IF NOT EXISTS leg2_failed     INT          NOT NULL DEFAULT 0    AFTER leg2_busy,
  ADD COLUMN IF NOT EXISTS leg2_canceled   INT          NOT NULL DEFAULT 0    AFTER leg2_failed,
  ADD COLUMN IF NOT EXISTS leg_breakdown   JSON         NULL                  AFTER leg2_canceled;

SELECT 'migration 001 complete' AS status;
