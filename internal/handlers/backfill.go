package handlers

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"exotel-monitoring-platform/internal/config"
	"exotel-monitoring-platform/internal/exotel"
	"exotel-monitoring-platform/internal/metrics"
	"exotel-monitoring-platform/internal/models"
	"exotel-monitoring-platform/internal/repository"
	"exotel-monitoring-platform/internal/utils"
	"github.com/gin-gonic/gin"
)

// backfillJob tracks the progress of an async backfill run.
type backfillJob struct {
	ExophoneID  uint64
	Date        string
	Status      string // "running" | "done" | "error"
	Error       string
	TotalCalls  int
	AnswerRate  float64
	PagesHit    int
	FinishedAt  *time.Time
}

var (
	backfillMu   sync.RWMutex
	backfillJobs = map[string]*backfillJob{} // key: "exophoneID-date"
)

func backfillKey(exophoneID uint64, date string) string {
	return fmt.Sprintf("%d-%s", exophoneID, date)
}

// BackfillExophoneSnapshot kicks off an async full-day fetch from the Exotel API
// for one exophone + date. Returns 202 immediately; poll GET /backfill/status.
//
// POST /api/v1/exophones/:id/backfill?date=YYYY-MM-DD
func BackfillExophoneSnapshot(c *gin.Context) {
	exophoneID := parseUint(c.Param("id"))
	if exophoneID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid exophone id"})
		return
	}

	dateStr := c.DefaultQuery("date", "")
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}
	snapDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date, use YYYY-MM-DD"})
		return
	}

	ep, err := repository.GetExophoneByID(exophoneID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("exophone %d not found", exophoneID)})
		return
	}
	acc, err := repository.GetAccountByID(ep.AccountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "account not found: " + err.Error()})
		return
	}

	key := backfillKey(exophoneID, dateStr)
	backfillMu.Lock()
	if existing, ok := backfillJobs[key]; ok && existing.Status == "running" {
		backfillMu.Unlock()
		c.JSON(http.StatusConflict, gin.H{"error": "backfill already running for this exophone+date", "key": key})
		return
	}
	job := &backfillJob{ExophoneID: exophoneID, Date: dateStr, Status: "running"}
	backfillJobs[key] = job
	backfillMu.Unlock()

	// Decrypt credentials once before handing off to the goroutine.
	secretKey := config.App.Crypto.SecretKey
	apiKey, err := utils.Decrypt(acc.APIKey, secretKey)
	if err != nil {
		apiKey = acc.APIKey
	}
	apiToken, err := utils.Decrypt(acc.APIToken, secretKey)
	if err != nil {
		apiToken = acc.APIToken
	}

	go func() {
		ist := time.FixedZone("IST", 5*60*60+30*60)
		dayStart := time.Date(snapDate.Year(), snapDate.Month(), snapDate.Day(), 0, 0, 0, 0, ist)
		dayEnd := time.Date(snapDate.Year(), snapDate.Month(), snapDate.Day(), 23, 59, 59, 0, ist)

		result, fetchErr := exotel.FetchCalls(
			acc.SID, acc.Subdomain, apiKey, apiToken,
			ep.ExophoneNumber, dayStart, dayEnd,
		)

		fail := func(msg string) {
			t := time.Now()
			backfillMu.Lock()
			job.Status = "error"
			job.Error = msg
			job.FinishedAt = &t
			backfillMu.Unlock()
		}

		if fetchErr != nil || result == nil || !isHTTPSuccess(result.HTTPStatus) {
			status := 0
			if result != nil {
				status = result.HTTPStatus
			}
			fail(fmt.Sprintf("Exotel API error HTTP %d: %v", status, fetchErr))
			return
		}

		// Find CALLS job for transaction FK.
		callsJob, jobErr := repository.GetCallsJobForExophone(exophoneID)
		var jobID uint64
		if jobErr == nil {
			jobID = callsJob.ID
		}

		txnID := fmt.Sprintf("BACKFILL-%d-%s", exophoneID, dateStr)
		now := time.Now()
		latMs := result.LatencyMs
		_ = repository.CreateTransaction(&models.JobTransaction{
			TransactionID: txnID,
			JobID:         jobID,
			AccountID:     acc.ID,
			ExophoneID:    exophoneID,
			JobType:       "BACKFILL",
			Status:        "SUCCESS",
			StartedAt:     now,
			CompletedAt:   &now,
			APILatencyMs:  &latMs,
			UpdatedBy:     "BACKFILL",
		})

		for _, pageRaw := range result.PageBodies {
			_ = repository.SaveJobResponse(&models.JobResponse{
				TransactionID: txnID,
				ResponseType:  "CALLS",
				HTTPStatus:    result.HTTPStatus,
				RawResponse:   pageRaw,
			})
		}

		var allRecords []exotel.CallRecord
		if result.Response != nil {
			allRecords = result.Response.Result
		}

		fakeJob := models.MonitoringJob{AccountID: acc.ID, ExophoneID: exophoneID}
		callLogs, _, summary := metrics.ProcessCallRecords(txnID, fakeJob, ep.ExophoneNumber, allRecords)

		if ep.SkipCallLogs == 0 && len(callLogs) > 0 {
			_ = repository.SaveCallLogs(callLogs)
		}

		snap := &models.CallMetricsSnapshot{
			SnapshotDate:       snapDate,
			AccountID:          acc.ID,
			ExophoneID:         exophoneID,
			TotalCalls:         summary.Total,
			Leg1Total:          summary.Leg1Total,
			Leg1Drops:          summary.DroppedLeg1,
			Leg2Total:          summary.Leg2Total,
			Leg2Drops:          summary.DroppedLeg2,
			ConnectedCalls:     summary.Connected,
			DroppedCalls:       summary.DroppedLeg1 + summary.DroppedLeg2,
			FailedCalls:        summary.Failed,
			NoAnswerCalls:      summary.NoAnswer,
			BusyCalls:          summary.Busy,
			CanceledCalls:      summary.Canceled,
			OtherCalls:         summary.Other,
			AvgDurationSec:     summary.AvgDurationSec,
			AnswerRate:         summary.AnswerRate,
			DropRate:           summary.DropRate,
			Leg1DropRate:       summary.Leg1DropRate,
			SuccessRate:        summary.SuccessRate,
			HourlyDistribution: metrics.HourlyDistributionJSON(summary.HourlyDistribution),
			PeakHour:           summary.PeakHour,
			Leg1NoAnswer:       summary.Leg1NoAnswer,
			Leg1Busy:           summary.Leg1Busy,
			Leg1Failed:         summary.Leg1Failed,
			Leg2NoAnswer:       summary.Leg2NoAnswer,
			Leg2Busy:           summary.Leg2Busy,
			Leg2Failed:         summary.Leg2Failed,
			Leg2Canceled:       summary.Leg2Canceled,
		}
		if saveErr := repository.UpsertCallMetricsSnapshot(snap); saveErr != nil {
			fail("snapshot save failed: " + saveErr.Error())
			return
		}

		_ = repository.AccumulateAccountDashboardFromSnapshots(acc.ID)

		t := time.Now()
		backfillMu.Lock()
		job.Status = "done"
		job.TotalCalls = summary.Total
		job.AnswerRate = summary.AnswerRate
		job.PagesHit = result.PagesFetched
		job.FinishedAt = &t
		backfillMu.Unlock()
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"status":          "accepted",
		"exophone_id":     exophoneID,
		"exophone_number": ep.ExophoneNumber,
		"date":            dateStr,
		"poll_url":        fmt.Sprintf("/api/v1/exophones/%d/backfill/status?date=%s", exophoneID, dateStr),
	})
}

// BackfillStatus returns the current state of an async backfill job.
//
// GET /api/v1/exophones/:id/backfill/status?date=YYYY-MM-DD
func BackfillStatus(c *gin.Context) {
	exophoneID := parseUint(c.Param("id"))
	dateStr := c.DefaultQuery("date", time.Now().Format("2006-01-02"))
	key := backfillKey(exophoneID, dateStr)

	backfillMu.RLock()
	job, ok := backfillJobs[key]
	backfillMu.RUnlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "no backfill job found for this exophone+date"})
		return
	}

	backfillMu.RLock()
	resp := gin.H{
		"exophone_id": exophoneID,
		"date":        dateStr,
		"status":      job.Status,
	}
	if job.Error != "" {
		resp["error"] = job.Error
	}
	if job.Status == "done" {
		resp["total_calls"] = job.TotalCalls
		resp["answer_rate_pct"] = fmt.Sprintf("%.2f", job.AnswerRate)
		resp["pages_fetched"] = job.PagesHit
		resp["finished_at"] = job.FinishedAt
	}
	backfillMu.RUnlock()

	c.JSON(http.StatusOK, resp)
}

func isHTTPSuccess(code int) bool {
	return code >= 200 && code < 300
}
