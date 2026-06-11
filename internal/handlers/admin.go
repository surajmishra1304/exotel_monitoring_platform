package handlers

import (
	"net/http"

	"exotel-monitoring-platform/internal/database"
	"github.com/gin-gonic/gin"
)

// TruncateMetrics drops all reporting/transaction data and resets monitoring
// job cursors so the next run starts fresh.
//
// POST /api/v1/admin/truncate-metrics
//
// Core config tables (accounts, exophones, monitoring_jobs, priority_config,
// feature_flags, settings) are NOT touched.
func TruncateMetrics(c *gin.Context) {
	tables := []string{
		"call_logs",
		"job_responses",
		"job_transactions",
		"call_metrics_snapshot",
		"account_dashboard_snapshot",
		"metrics",
		"heartbeat_metrics",
		"stream_metrics",
		"exophone_health_snapshot",
		"stream_utilization_snapshot",
		"alerts",
	}

	db := database.DB
	db.Exec("SET FOREIGN_KEY_CHECKS = 0")
	for _, t := range tables {
		db.Exec("TRUNCATE TABLE " + t)
	}
	db.Exec("SET FOREIGN_KEY_CHECKS = 1")

	// Reset last_run_at so the next monitoring job run uses frequency_minutes
	// fallback instead of a stale cursor.
	db.Exec("UPDATE monitoring_jobs SET last_run_at = NULL")

	c.JSON(http.StatusOK, gin.H{
		"truncated":        tables,
		"last_run_at_reset": true,
		"message":          "all reporting tables cleared; monitoring cursors reset",
	})
}
