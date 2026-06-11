package server

import (
	"fmt"
	"net/http"
	"time"

	"exotel-monitoring-platform/internal/config"
	"exotel-monitoring-platform/internal/handlers"
	"exotel-monitoring-platform/internal/middleware"
	"github.com/gin-gonic/gin"
)

// New builds and returns the configured HTTP server.
func New() *http.Server {
	cfg := config.App.Server

	r := gin.New()
	r.Use(middleware.CORS()) // must be first so pre-flight OPTIONS is handled before auth/logging
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(middleware.RequestID())

	// ── Infrastructure ─────────────────────────────────────────────────
	r.GET("/health", handlers.Health)

	// ── Dashboard APIs (cache-first) ────────────────────────────────────
	api := r.Group("/api/v1")
	{
		// Summary dashboard — all accounts
		api.GET("/dashboard/summary", handlers.DashboardSummary)

		// Per-exophone call analytics
		api.GET("/dashboard/exophones/:id/calls", handlers.DashboardCalls)

		// Account / organisation management
		api.POST("/accounts", handlers.CreateAccount)

		// Exophone management
		api.GET("/accounts/:account_id/exophones", handlers.ListExophones)
		api.POST("/accounts/:account_id/exophones", handlers.AddExophone)
		api.POST("/accounts/:account_id/exophones/sync", handlers.SyncExophones)
		api.GET("/accounts/:account_id/exophones/health", handlers.GetExophoneHealthList)
		api.GET("/accounts/:account_id/performance/today", handlers.TodayPerformance)
		api.GET("/accounts/:account_id/performance/export", handlers.ExportPerformanceCSV)
		api.GET("/exophones/:id", handlers.GetExophone)
		api.GET("/exophones/:id/metrics", handlers.GetExophoneMetrics)
		api.PATCH("/exophones/:id/cron", handlers.ToggleCron)
		api.PATCH("/exophones/:id/priority", handlers.UpdatePriority)
		api.PATCH("/exophones/:id/skip-call-logs", handlers.ToggleSkipCallLogs)
		api.POST("/exophones/:id/heartbeat/check", handlers.CheckExophoneHeartbeat)
		api.POST("/exophones/:id/snapshot/reprocess", handlers.ReprocessExophoneSnapshot)
		api.POST("/exophones/:id/backfill", handlers.BackfillExophoneSnapshot)
		api.GET("/exophones/:id/backfill/status", handlers.BackfillStatus)
		api.POST("/backfill/bulk", handlers.BulkBackfill)
		api.GET("/backfill/bulk/status", handlers.BulkBackfillStatus)
		api.POST("/admin/truncate-metrics", handlers.TruncateMetrics)

		// Monitoring transactions
		api.GET("/transactions", handlers.GetTransactions)

		// Alerts
		api.GET("/alerts/active", handlers.GetActiveAlerts)
		api.POST("/alerts", handlers.CreateAlert)
		api.PUT("/alerts/:id/acknowledge", handlers.AcknowledgeAlert)

		// Priority config management
		api.GET("/priority-config", handlers.GetPriorityConfig)
		api.PATCH("/priority-config/:priority", handlers.UpdatePriorityConfigHandler)

		// Runtime settings / feature flags
		api.GET("/settings", handlers.GetSettings)
		api.PATCH("/settings", handlers.UpdateSettings)
	}

	// ── Webhooks (Exotel → Platform) ────────────────────────────────────
	r.POST("/exotel/webhook", handlers.ExotelWebhook)
	// Heartbeat push: configure this URL in Exotel Dashboard → Notifications Settings
	// GET is required for Exotel's URL validation check before it accepts the config.
	r.GET("/exotel/heartbeat", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.POST("/exotel/heartbeat", handlers.ExotelHeartbeat)

	return &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      r,
		ReadTimeout:  time.Duration(cfg.ReadTimeoutSeconds) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeoutSeconds) * time.Second,
		IdleTimeout:  time.Duration(cfg.IdleTimeoutSeconds) * time.Second,
	}
}
