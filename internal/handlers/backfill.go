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

// ── Bulk backfill state ───────────────────────────────────────────────────────

type bulkBackfillState struct {
	mu      sync.RWMutex
	Total   int
	Done    int
	Failed  int
	Running bool
	Results []bulkResult
}

type bulkResult struct {
	ExophoneID     uint64  `json:"exophone_id"`
	ExophoneNumber string  `json:"exophone_number"`
	Date           string  `json:"date"`
	Status         string  `json:"status"` // "done" | "error"
	TotalCalls     int     `json:"total_calls,omitempty"`
	AnswerRate     float64 `json:"answer_rate_pct,omitempty"`
	PagesHit       int     `json:"pages_hit,omitempty"`
	Error          string  `json:"error,omitempty"`
}

var bulk = &bulkBackfillState{}

// backfillJob tracks the progress of an async backfill run.
type backfillJob struct {
	ExophoneID uint64
	Date       string
	Status     string // "running" | "done" | "error"
	Error      string
	TotalCalls int
	AnswerRate float64
	PagesHit   int
	FinishedAt *time.Time
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
// POST /api/v1/exophones/:id/backfill?date=YYYY-MM-DD&skip_call_logs=1
//
// skip_call_logs query param overrides the per-exophone setting for this run only.
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

	// skip_call_logs query param overrides the exophone's persisted setting for this run.
	skipLogs := ep.SkipCallLogs == 1
	if q := c.Query("skip_call_logs"); q == "1" || q == "true" {
		skipLogs = true
	} else if q == "0" || q == "false" {
		skipLogs = false
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

		if !skipLogs && len(callLogs) > 0 {
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

// BulkBackfill kicks off backfill for every monitored exophone across the
// requested dates. Runs one (exophone, date) pair at a time to avoid
// hammering Exotel's rate limits. Returns 202 immediately; poll
// GET /api/v1/backfill/bulk/status for progress.
//
// POST /api/v1/backfill/bulk
//
//	{
//	  "dates":          ["2026-06-09","2026-06-10","2026-06-11"],
//	  "skip_call_logs": true   // overrides per-exophone flag for all runs
//	}
func BulkBackfill(c *gin.Context) {
	var body struct {
		Dates        []string `json:"dates" binding:"required"`
		SkipCallLogs bool     `json:"skip_call_logs"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.Dates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dates array required"})
		return
	}
	for _, d := range body.Dates {
		if _, err := time.Parse("2006-01-02", d); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date: " + d})
			return
		}
	}

	bulk.mu.Lock()
	if bulk.Running {
		bulk.mu.Unlock()
		c.JSON(http.StatusConflict, gin.H{"error": "bulk backfill already running"})
		return
	}
	bulk.Running = true
	bulk.Done = 0
	bulk.Failed = 0
	bulk.Results = nil
	bulk.mu.Unlock()

	exophones, err := repository.GetMonitoredExophones()
	if err != nil || len(exophones) == 0 {
		bulk.mu.Lock()
		bulk.Running = false
		bulk.mu.Unlock()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load exophones"})
		return
	}

	secretKey := config.App.Crypto.SecretKey

	// Total work items = exophones × dates.
	total := len(exophones) * len(body.Dates)
	bulk.mu.Lock()
	bulk.Total = total
	bulk.mu.Unlock()

	go func() {
		ist := time.FixedZone("IST", 5*60*60+30*60)

		// Worker pool: 3 concurrent exophone fetches to balance speed vs rate limits.
		// Each worker processes one (exophone, date) pair at a time.
		type task struct {
			ep       models.Exophone
			dateStr  string
			skipLogs bool
		}

		taskCh := make(chan task, total)

		// Pre-decrypt credentials per account to avoid repeated DB lookups.
		type creds struct{ apiKey, apiToken string }
		credCache := map[uint64]creds{}
		for _, ep := range exophones {
			if _, ok := credCache[ep.AccountID]; ok {
				continue
			}
			acc, err := repository.GetAccountByID(ep.AccountID)
			if err != nil {
				continue
			}
			k, err := utils.Decrypt(acc.APIKey, secretKey)
			if err != nil {
				k = acc.APIKey
			}
			t2, err := utils.Decrypt(acc.APIToken, secretKey)
			if err != nil {
				t2 = acc.APIToken
			}
			credCache[ep.AccountID] = creds{k, t2}
		}

		// Enqueue all tasks.
		for _, ep := range exophones {
			sl := body.SkipCallLogs || ep.SkipCallLogs == 1
			for _, d := range body.Dates {
				taskCh <- task{ep: ep, dateStr: d, skipLogs: sl}
			}
		}
		close(taskCh)

		var wg sync.WaitGroup
		const concurrency = 3
		for w := 0; w < concurrency; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for t := range taskCh {
					ep := t.ep
					dateStr := t.dateStr

					acc, err := repository.GetAccountByID(ep.AccountID)
					if err != nil {
						bulk.mu.Lock()
						bulk.Failed++
						bulk.Done++
						bulk.Results = append(bulk.Results, bulkResult{
							ExophoneID: ep.ID, ExophoneNumber: ep.ExophoneNumber,
							Date: dateStr, Status: "error", Error: "account not found",
						})
						bulk.mu.Unlock()
						continue
					}
					cr := credCache[ep.AccountID]

					snapDate, _ := time.Parse("2006-01-02", dateStr)
					dayStart := time.Date(snapDate.Year(), snapDate.Month(), snapDate.Day(), 0, 0, 0, 0, ist)
					dayEnd := time.Date(snapDate.Year(), snapDate.Month(), snapDate.Day(), 23, 59, 59, 0, ist)

					result, fetchErr := exotel.FetchCalls(
						acc.SID, acc.Subdomain, cr.apiKey, cr.apiToken,
						ep.ExophoneNumber, dayStart, dayEnd,
					)

					if fetchErr != nil || result == nil || !isHTTPSuccess(result.HTTPStatus) {
						httpCode := 0
						if result != nil {
							httpCode = result.HTTPStatus
						}
						errMsg := fmt.Sprintf("HTTP %d: %v", httpCode, fetchErr)
						// Back off on rate-limit so other workers also slow down.
						if httpCode == 429 {
							time.Sleep(10 * time.Second)
						}
						bulk.mu.Lock()
						bulk.Failed++
						bulk.Done++
						bulk.Results = append(bulk.Results, bulkResult{
							ExophoneID: ep.ID, ExophoneNumber: ep.ExophoneNumber,
							Date: dateStr, Status: "error", Error: errMsg,
						})
						bulk.mu.Unlock()
						continue
					}

					callsJob, jobErr := repository.GetCallsJobForExophone(ep.ID)
					var jobID uint64
					if jobErr == nil {
						jobID = callsJob.ID
					}

					txnID := fmt.Sprintf("BACKFILL-%d-%s", ep.ID, dateStr)
					now := time.Now()
					latMs := result.LatencyMs
					_ = repository.CreateTransaction(&models.JobTransaction{
						TransactionID: txnID, JobID: jobID,
						AccountID: acc.ID, ExophoneID: ep.ID,
						JobType: "BACKFILL", Status: "SUCCESS",
						StartedAt: now, CompletedAt: &now,
						APILatencyMs: &latMs, UpdatedBy: "BACKFILL",
					})
					for _, pageRaw := range result.PageBodies {
						_ = repository.SaveJobResponse(&models.JobResponse{
							TransactionID: txnID, ResponseType: "CALLS",
							HTTPStatus: result.HTTPStatus, RawResponse: pageRaw,
						})
					}

					var allRecords []exotel.CallRecord
					if result.Response != nil {
						allRecords = result.Response.Result
					}

					fakeJob := models.MonitoringJob{AccountID: acc.ID, ExophoneID: ep.ID}
					callLogs, _, summary := metrics.ProcessCallRecords(txnID, fakeJob, ep.ExophoneNumber, allRecords)

					if !t.skipLogs && len(callLogs) > 0 {
						_ = repository.SaveCallLogs(callLogs)
					}

					snap := &models.CallMetricsSnapshot{
						SnapshotDate: snapDate, AccountID: acc.ID, ExophoneID: ep.ID,
						TotalCalls: summary.Total, Leg1Total: summary.Leg1Total,
						Leg1Drops: summary.DroppedLeg1, Leg2Total: summary.Leg2Total,
						Leg2Drops: summary.DroppedLeg2, ConnectedCalls: summary.Connected,
						DroppedCalls: summary.DroppedLeg1 + summary.DroppedLeg2,
						FailedCalls:  summary.Failed, NoAnswerCalls: summary.NoAnswer,
						BusyCalls: summary.Busy, CanceledCalls: summary.Canceled,
						OtherCalls: summary.Other, AvgDurationSec: summary.AvgDurationSec,
						AnswerRate: summary.AnswerRate, DropRate: summary.DropRate,
						Leg1DropRate: summary.Leg1DropRate, SuccessRate: summary.SuccessRate,
						HourlyDistribution: metrics.HourlyDistributionJSON(summary.HourlyDistribution),
						PeakHour:           summary.PeakHour,
						Leg1NoAnswer:       summary.Leg1NoAnswer, Leg1Busy: summary.Leg1Busy,
						Leg1Failed: summary.Leg1Failed, Leg2NoAnswer: summary.Leg2NoAnswer,
						Leg2Busy: summary.Leg2Busy, Leg2Failed: summary.Leg2Failed,
						Leg2Canceled: summary.Leg2Canceled,
					}
					_ = repository.UpsertCallMetricsSnapshot(snap)
					_ = repository.AccumulateAccountDashboardFromSnapshots(acc.ID)

					bulk.mu.Lock()
					bulk.Done++
					bulk.Results = append(bulk.Results, bulkResult{
						ExophoneID: ep.ID, ExophoneNumber: ep.ExophoneNumber,
						Date: dateStr, Status: "done",
						TotalCalls: summary.Total, AnswerRate: summary.AnswerRate,
						PagesHit: result.PagesFetched,
					})
					bulk.mu.Unlock()
				}
			}()
		}

		wg.Wait()
		bulk.mu.Lock()
		bulk.Running = false
		bulk.mu.Unlock()
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"status":     "accepted",
		"total_jobs": total,
		"dates":      body.Dates,
		"exophones":  len(exophones),
		"poll_url":   "/api/v1/backfill/bulk/status",
	})
}

// BulkBackfillStatus returns current progress of the running or last bulk backfill.
//
// GET /api/v1/backfill/bulk/status
func BulkBackfillStatus(c *gin.Context) {
	bulk.mu.RLock()
	defer bulk.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{
		"running": bulk.Running,
		"total":   bulk.Total,
		"done":    bulk.Done,
		"failed":  bulk.Failed,
		"results": bulk.Results,
	})
}
