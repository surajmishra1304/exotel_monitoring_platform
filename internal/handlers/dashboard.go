package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"exotel-monitoring-platform/internal/cache"
	"exotel-monitoring-platform/internal/database"
	"exotel-monitoring-platform/internal/logger"
	"exotel-monitoring-platform/internal/models"
	"exotel-monitoring-platform/internal/repository"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// DashboardSummary returns the global account-level KPI summary.
// Serve order: Redis → snapshot table → 503 on DB failure.
func DashboardSummary(c *gin.Context) {
	ctx := context.Background()
	reqID, _ := c.Get("request_id")

	// 1. Try Redis cache.
	var summary any
	if hit, cacheErr := cache.Get(ctx, cache.KeyDashboardSum, &summary); hit {
		c.JSON(http.StatusOK, gin.H{"source": "cache", "data": summary})
		return
	} else if cacheErr != nil {
		// Cache error is non-fatal — log and fall through to DB.
		logger.Log.Warn("dashboard cache read failed, falling back to DB",
			zap.String("request_id", fmt.Sprintf("%v", reqID)),
			zap.Error(cacheErr),
		)
	}

	// 2. Fall back to snapshot table.
	accounts, err := repository.GetActiveAccounts()
	if err != nil {
		logger.Log.Error("dashboard: failed to fetch active accounts",
			zap.String("request_id", fmt.Sprintf("%v", reqID)),
			zap.Error(err),
		)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": fmt.Sprintf("database unavailable: %s", err.Error()),
		})
		return
	}

	if len(accounts) == 0 {
		c.JSON(http.StatusOK, gin.H{"source": "snapshot", "data": []any{}})
		return
	}

	today := time.Now().Truncate(24 * time.Hour)

	// Single batch query instead of one query per account.
	snapsByAccountID, _ := repository.GetAllAccountDashboardSnapshots(today)

	var results []gin.H

	for _, acc := range accounts {
		row := gin.H{
			"account_id":       acc.ID,
			"account_name":     acc.AccountName,
			"total_exophones":  0,
			"active_exophones": 0,
			"total_calls":      0,
			"failed_calls":     0,
			"success_rate":     0,
			"avg_latency_ms":   nil,
			"alert_count":      0,
			"snapshot_date":    today.Format("2006-01-02"),
		}

		if snap, ok := snapsByAccountID[acc.ID]; ok {
			row["total_exophones"] = snap.TotalExophones
			row["active_exophones"] = snap.ActiveExophones
			row["total_calls"] = snap.TotalCalls
			row["failed_calls"] = snap.FailedCalls
			row["success_rate"] = snap.SuccessRate
			row["avg_latency_ms"] = snap.AvgLatencyMs
			row["alert_count"] = snap.AlertCount
			row["snapshot_date"] = snap.SnapshotDate
		} else {
			// Snapshot not yet generated — count live from exophones table.
			var exophoneCount, activeCount int64
			database.DB.Model(&models.Exophone{}).
				Where("account_id = ? AND is_deleted = 0", acc.ID).
				Count(&exophoneCount)
			database.DB.Model(&models.Exophone{}).
				Where("account_id = ? AND status = 'active' AND is_deleted = 0", acc.ID).
				Count(&activeCount)
			row["total_exophones"] = exophoneCount
			row["active_exophones"] = activeCount
		}

		results = append(results, row)
	}

	// Warm cache; log but don't fail on cache write error.
	if setErr := cache.Set(ctx, cache.KeyDashboardSum, results, cache.TTLDashboard); setErr != nil {
		logger.Log.Warn("dashboard: failed to warm cache",
			zap.String("request_id", fmt.Sprintf("%v", reqID)),
			zap.Error(setErr),
		)
	}

	c.JSON(http.StatusOK, gin.H{"source": "snapshot", "data": results})
}

// DashboardCalls returns call analytics for a specific exophone.
// Optional ?date=YYYY-MM-DD query parameter selects a historical date (defaults to today).
//
// When today's snapshot hasn't been generated yet (first sync of the day is pending),
// the handler falls back to the most recent available snapshot and marks the response
// with "stale":true and "data_as_of":<date> so the UI can show a staleness badge.
//
// Serve order: Redis (today only) → exact-date snapshot → latest fallback snapshot → 404.
func DashboardCalls(c *gin.Context) {
	ctx := database.Ctx
	reqID, _ := c.Get("request_id")

	exophoneID := parseUint(c.Param("id"))
	if exophoneID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid exophone id: must be a positive integer"})
		return
	}

	// Optional date picker — defaults to today when omitted.
	dateParam := c.DefaultQuery("date", "")
	if dateParam != "" && dateParam != "today" {
		if _, err := time.Parse("2006-01-02", dateParam); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format, use YYYY-MM-DD"})
			return
		}
	}
	isToday := dateParam == "" || dateParam == "today"
	resolvedDate := time.Now().Format("2006-01-02")
	if !isToday {
		resolvedDate = dateParam
	}

	// Cache hit — only for today's data.
	if isToday {
		cacheKey := cache.MetricsKey(exophoneID)
		var cached any
		if hit, cacheErr := cache.Get(ctx, cacheKey, &cached); hit {
			c.JSON(http.StatusOK, gin.H{
				"source":     "cache",
				"data":       cached,
				"date":       resolvedDate,
				"stale":      false,
				"data_as_of": resolvedDate,
			})
			return
		} else if cacheErr != nil {
			logger.Log.Warn("dashboard calls: cache read failed",
				zap.Uint64("exophone_id", exophoneID),
				zap.String("request_id", fmt.Sprintf("%v", reqID)),
				zap.Error(cacheErr),
			)
		}
	}

	snapshotDate, _ := time.Parse("2006-01-02", resolvedDate)
	snap, err := repository.GetCallMetricsSnapshot(exophoneID, snapshotDate)
	if err == nil {
		// Exact match found — warm cache for today and return.
		if isToday {
			cacheKey := cache.MetricsKey(exophoneID)
			if setErr := cache.Set(ctx, cacheKey, snap, cache.TTLMetrics); setErr != nil {
				logger.Log.Warn("dashboard calls: failed to warm cache",
					zap.Uint64("exophone_id", exophoneID),
					zap.Error(setErr),
				)
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"source":     "snapshot",
			"data":       snap,
			"date":       resolvedDate,
			"stale":      false,
			"data_as_of": resolvedDate,
		})
		return
	}

	// Snapshot not found for the requested date.
	// If this is a historical date request, 404 is the correct answer.
	if !isToday {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("no call snapshot available for exophone %d on %s", exophoneID, resolvedDate),
		})
		return
	}

	// For today: fall back to the most recent snapshot (from any past date).
	// This happens during the window between midnight and the first job run of the day.
	latest, latestErr := repository.GetLatestCallMetricsSnapshot(exophoneID)
	if latestErr != nil {
		// No data at all yet for this exophone.
		logger.Log.Info("dashboard calls: no snapshot found for exophone",
			zap.Uint64("exophone_id", exophoneID),
			zap.String("request_id", fmt.Sprintf("%v", reqID)),
		)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   fmt.Sprintf("no call data available yet for exophone %d — first sync pending", exophoneID),
			"stale":   false,
			"pending": true,
		})
		return
	}

	dataAsOf := latest.SnapshotDate.Format("2006-01-02")
	logger.Log.Info("dashboard calls: serving stale fallback snapshot",
		zap.Uint64("exophone_id", exophoneID),
		zap.String("data_as_of", dataAsOf),
		zap.String("request_id", fmt.Sprintf("%v", reqID)),
	)
	c.JSON(http.StatusOK, gin.H{
		"source":     "snapshot",
		"data":       latest,
		"date":       resolvedDate,
		"stale":      true,
		"data_as_of": dataAsOf,
		"message":    fmt.Sprintf("Showing data from %s — today's data will appear after the next sync", dataAsOf),
	})
}
