package handlers

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"exotel-monitoring-platform/internal/cache"
	"exotel-monitoring-platform/internal/logger"
	"exotel-monitoring-platform/internal/repository"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func parsePage(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return page, limit
}

// GetExophoneMetrics returns the health snapshot for a single exophone.
// Serve order: Redis → exophone_health_snapshot → 404 with specific DB error.
func GetExophoneMetrics(c *gin.Context) {
	ctx := context.Background()
	reqID, _ := c.Get("request_id")

	exophoneID := parseUint(c.Param("id"))
	if exophoneID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid exophone id: must be a positive integer"})
		return
	}

	snapKey := cache.HealthSnapKey(exophoneID)
	var snap any
	if hit, cacheErr := cache.Get(ctx, snapKey, &snap); hit {
		c.JSON(http.StatusOK, gin.H{"source": "cache", "data": snap})
		return
	} else if cacheErr != nil {
		logger.Log.Warn("exophone metrics: cache read failed",
			zap.Uint64("exophone_id", exophoneID),
			zap.String("request_id", fmt.Sprintf("%v", reqID)),
			zap.Error(cacheErr),
		)
	}

	s, err := repository.GetExophoneHealthSnapshot(exophoneID)
	if err != nil {
		logger.Log.Error("exophone metrics: snapshot not found",
			zap.Uint64("exophone_id", exophoneID),
			zap.String("request_id", fmt.Sprintf("%v", reqID)),
			zap.Error(err),
		)
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("health snapshot not found for exophone %d: %s", exophoneID, err.Error()),
		})
		return
	}

	if setErr := cache.Set(ctx, snapKey, s, cache.TTLHealthSnap); setErr != nil {
		logger.Log.Warn("exophone metrics: failed to warm cache",
			zap.Uint64("exophone_id", exophoneID),
			zap.Error(setErr),
		)
	}
	c.JSON(http.StatusOK, gin.H{"source": "snapshot", "data": s})
}

// ListExophones returns paginated exophones for an account ordered by priority.
// Query params: page (default 1), limit (default 20, max 100).
func ListExophones(c *gin.Context) {
	reqID, _ := c.Get("request_id")

	accountID := parseUint(c.Param("account_id"))
	if accountID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id: must be a positive integer"})
		return
	}

	page, limit := parsePage(c)

	list, total, err := repository.GetExophonesByAccount(accountID, page, limit)
	if err != nil {
		logger.Log.Error("list exophones: DB query failed",
			zap.Uint64("account_id", accountID),
			zap.String("request_id", fmt.Sprintf("%v", reqID)),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to fetch exophones for account %d: %s", accountID, err.Error()),
		})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	c.JSON(http.StatusOK, gin.H{
		"data":        list,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
	})
}

// TodayPerformance returns today's call metrics snapshots for an account with pagination.
// GET /api/v1/accounts/:account_id/performance/today
func TodayPerformance(c *gin.Context) {
	reqID, _ := c.Get("request_id")

	accountID := parseUint(c.Param("account_id"))
	if accountID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id: must be a positive integer"})
		return
	}

	page, limit := parsePage(c)

	// ?date=YYYY-MM-DD; defaults to today when omitted or "today"
	dateParam := c.DefaultQuery("date", "")
	if dateParam == "today" {
		dateParam = ""
	}
	if dateParam != "" {
		if _, err := time.Parse("2006-01-02", dateParam); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format, use YYYY-MM-DD"})
			return
		}
	}
	resolvedDate := dateParam
	if resolvedDate == "" {
		resolvedDate = time.Now().Format("2006-01-02")
	}

	snaps, total, err := repository.GetCallMetricsByDate(accountID, dateParam, page, limit)
	if err != nil {
		logger.Log.Error("performance: DB query failed",
			zap.Uint64("account_id", accountID),
			zap.String("date", resolvedDate),
			zap.String("request_id", fmt.Sprintf("%v", reqID)),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to fetch performance for account %d on %s: %s", accountID, resolvedDate, err.Error()),
		})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	c.JSON(http.StatusOK, gin.H{
		"data":        snaps,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
		"date":        resolvedDate,
	})
}

// GetExophoneHealthList returns health snapshots for all exophones of an account.
func GetExophoneHealthList(c *gin.Context) {
	reqID, _ := c.Get("request_id")

	accountID := parseUint(c.Param("account_id"))
	if accountID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id: must be a positive integer"})
		return
	}

	snaps, err := repository.GetAllExophoneHealthSnapshots(accountID)
	if err != nil {
		logger.Log.Error("exophone health list: DB query failed",
			zap.Uint64("account_id", accountID),
			zap.String("request_id", fmt.Sprintf("%v", reqID)),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to fetch health snapshots for account %d: %s", accountID, err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": snaps})
}

// UpdatePriority changes the priority and monitoring frequency for a single exophone.
// PATCH /api/v1/exophones/:id/priority
func UpdatePriority(c *gin.Context) {
	reqID, _ := c.Get("request_id")

	exophoneID := parseUint(c.Param("id"))
	if exophoneID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid exophone id"})
		return
	}

	var body struct {
		Priority        string `json:"priority"         binding:"required"`
		FrequencyMinutes *int  `json:"frequency_minutes"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "priority (P0/P1/P2/P3) is required"})
		return
	}

	// Load configured defaults from priority_config table; fall back if unavailable.
	configMap, _ := repository.GetPriorityConfigMap()
	if len(configMap) == 0 {
		configMap = map[string]int{"P0": 15, "P1": 30, "P2": 60, "P3": 60}
	}
	defaultFreq, ok := configMap[body.Priority]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "priority must be one of P0, P1, P2, P3"})
		return
	}

	freqMinutes := defaultFreq
	if body.FrequencyMinutes != nil {
		if *body.FrequencyMinutes < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "frequency_minutes must be >= 1"})
			return
		}
		freqMinutes = *body.FrequencyMinutes
	}

	if err := repository.UpdatePriority(exophoneID, body.Priority, freqMinutes); err != nil {
		logger.Log.Error("update priority: DB update failed",
			zap.Uint64("exophone_id", exophoneID),
			zap.String("request_id", fmt.Sprintf("%v", reqID)),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"exophone_id":       exophoneID,
		"priority":          body.Priority,
		"frequency_minutes": freqMinutes,
		"message":           "priority updated",
	})
}

// ToggleCron sets is_cron_applicable for a single exophone.
// PATCH /api/v1/exophones/:id/cron
func ToggleCron(c *gin.Context) {
	reqID, _ := c.Get("request_id")

	exophoneID := parseUint(c.Param("id"))
	if exophoneID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid exophone id"})
		return
	}

	// Accept both integer (0/1) and boolean (true/false) forms.
	var raw struct {
		IsCronApplicable json.RawMessage `json:"is_cron_applicable" binding:"required"`
	}
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "is_cron_applicable (0/1 or true/false) is required"})
		return
	}
	s := string(raw.IsCronApplicable)
	var cronVal int
	switch s {
	case "1", "true":
		cronVal = 1
	case "0", "false":
		cronVal = 0
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "is_cron_applicable must be 0, 1, true, or false"})
		return
	}

	if err := repository.SetCronApplicable(exophoneID, cronVal); err != nil {
		logger.Log.Error("toggle cron: DB update failed",
			zap.Uint64("exophone_id", exophoneID),
			zap.String("request_id", fmt.Sprintf("%v", reqID)),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"exophone_id":        exophoneID,
		"is_cron_applicable": cronVal,
		"message":           "cron flag updated",
	})
}

// ExportPerformanceCSV streams exophone call-metric snapshots for an account as CSV.
//
// GET /api/v1/accounts/:account_id/performance/export?date=YYYY-MM-DD&exophone_id=N
// date defaults to today; exophone_id is optional (omit for all exophones).
func ExportPerformanceCSV(c *gin.Context) {
	reqID, _ := c.Get("request_id")

	accountID := parseUint(c.Param("account_id"))
	if accountID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
		return
	}

	dateParam := c.DefaultQuery("date", "")
	if dateParam == "today" {
		dateParam = ""
	}
	if dateParam != "" {
		if _, err := time.Parse("2006-01-02", dateParam); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format, use YYYY-MM-DD"})
			return
		}
	}
	resolvedDate := dateParam
	if resolvedDate == "" {
		resolvedDate = time.Now().Format("2006-01-02")
	}

	exophoneID := parseUint(c.DefaultQuery("exophone_id", "0"))

	rows, err := repository.GetCallMetricsForExport(accountID, dateParam, exophoneID)
	if err != nil {
		logger.Log.Error("export performance: DB query failed",
			zap.Uint64("account_id", accountID),
			zap.String("date", resolvedDate),
			zap.String("request_id", fmt.Sprintf("%v", reqID)),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	filename := fmt.Sprintf("exophone-report-%s.csv", resolvedDate)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Cache-Control", "no-cache")

	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{
		"exophone_number", "exophone_id", "date",
		"total_calls", "connected_calls", "answer_rate_%",
		"leg1_total", "leg1_drops", "leg1_drop_rate_%",
		"leg2_total", "leg2_drops", "drop_rate_%",
		"failed_calls", "no_answer_calls", "busy_calls", "canceled_calls",
		"avg_duration_sec", "peak_hour",
	})

	for _, s := range rows {
		_ = w.Write([]string{
			s.ExophoneNumber,
			strconv.FormatUint(s.ExophoneID, 10),
			s.SnapshotDate.Format("2006-01-02"),
			strconv.Itoa(s.TotalCalls),
			strconv.Itoa(s.ConnectedCalls),
			fmt.Sprintf("%.2f", s.AnswerRate),
			strconv.Itoa(s.Leg1Total),
			strconv.Itoa(s.Leg1Drops),
			fmt.Sprintf("%.2f", s.Leg1DropRate),
			strconv.Itoa(s.Leg2Total),
			strconv.Itoa(s.Leg2Drops),
			fmt.Sprintf("%.2f", s.DropRate),
			strconv.Itoa(s.FailedCalls),
			strconv.Itoa(s.NoAnswerCalls),
			strconv.Itoa(s.BusyCalls),
			strconv.Itoa(s.CanceledCalls),
			fmt.Sprintf("%.1f", s.AvgDurationSec),
			strconv.Itoa(s.PeakHour),
		})
	}
	w.Flush()
}

// parseUint safely parses a URL param to uint64.
// Returns 0 for any invalid or non-positive input.
func parseUint(s string) uint64 {
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
}
