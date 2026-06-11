package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"exotel-monitoring-platform/internal/exotel"
	"exotel-monitoring-platform/internal/metrics"
	"exotel-monitoring-platform/internal/models"
	"exotel-monitoring-platform/internal/repository"
	"github.com/gin-gonic/gin"
)

// ReprocessExophoneSnapshot rebuilds the call_metrics_snapshot for a specific
// exophone and date entirely from the raw job_responses stored in the DB.
//
// POST /api/v1/exophones/:id/snapshot/reprocess?date=YYYY-MM-DD
//
// Use this when skip_call_logs_write is ON and the snapshot has drifted
// (e.g. due to a mid-day code deploy, accumulation bug, or clock skew).
// All job_responses for the exophone on that date are parsed, call records
// are deduplicated by Sid across all pages/batches, and the snapshot is
// overwritten with a fresh computation.
func ReprocessExophoneSnapshot(c *gin.Context) {
	exophoneID := parseUint(c.Param("id"))
	if exophoneID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid exophone id"})
		return
	}

	dateStr := c.DefaultQuery("date", "")
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", dateStr); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date, use YYYY-MM-DD"})
		return
	}

	exophone, err := repository.GetExophoneByID(exophoneID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("exophone %d not found", exophoneID)})
		return
	}

	responses, err := repository.GetJobResponsesForReprocess(exophoneID, dateStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error fetching responses: " + err.Error()})
		return
	}

	// Parse every response page and deduplicate call records by Sid across all pages/batches.
	seen := make(map[string]struct{})
	var allRecords []exotel.CallRecord
	skipped := 0
	source := "job_responses"

	for _, resp := range responses {
		var cr exotel.CallsResponse
		if err := json.Unmarshal([]byte(resp.RawResponse), &cr); err != nil {
			skipped++
			continue
		}
		for _, rec := range cr.Result {
			if _, dup := seen[rec.Sid]; !dup {
				seen[rec.Sid] = struct{}{}
				allRecords = append(allRecords, rec)
			}
		}
	}

	// Fallback: if no job_responses exist (or none parsed), rebuild from call_logs.
	// call_logs are always present when skip_call_logs_write is OFF for this exophone.
	if len(allRecords) == 0 {
		logs, lerr := repository.GetCallLogsForReprocess(exophoneID, dateStr)
		if lerr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error fetching call_logs: " + lerr.Error()})
			return
		}
		if len(logs) == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"error":             fmt.Sprintf("no data found for exophone %d on %s in job_responses or call_logs", exophoneID, dateStr),
				"responses_checked": len(responses),
				"responses_skipped": skipped,
			})
			return
		}
		// Convert call_logs rows to exotel.CallRecord so ProcessCallRecords can handle them.
		for _, cl := range logs {
			allRecords = append(allRecords, callLogToRecord(cl))
		}
		source = "call_logs"
		skipped = 0
	}

	// Build a minimal job context — only AccountID and ExophoneID are used by ProcessCallRecords.
	fakeJob := models.MonitoringJob{
		AccountID:  exophone.AccountID,
		ExophoneID: exophoneID,
	}

	// Recompute the full summary from scratch.
	callLogs, _, summary := metrics.ProcessCallRecords("REPROCESS", fakeJob, exophone.ExophoneNumber, allRecords)

	// Write call_logs when skip_call_logs is OFF so the DB reflects the reprocessed data.
	if exophone.SkipCallLogs == 0 && len(callLogs) > 0 {
		_ = repository.SaveCallLogs(callLogs)
	}

	// Build a clean snapshot (not accumulated — full overwrite).
	snapDate, _ := time.Parse("2006-01-02", dateStr)
	snap := &models.CallMetricsSnapshot{
		SnapshotDate:       snapDate,
		AccountID:          exophone.AccountID,
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

	if err := repository.UpsertCallMetricsSnapshot(snap); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save snapshot: " + err.Error()})
		return
	}

	// Refresh the account dashboard snapshot to reflect the corrected exophone data.
	_ = repository.AccumulateAccountDashboardFromSnapshots(exophone.AccountID)

	c.JSON(http.StatusOK, gin.H{
		"source":               source,
		"exophone_id":          exophoneID,
		"exophone_number":      exophone.ExophoneNumber,
		"date":                 dateStr,
		"responses_processed":  len(responses) - skipped,
		"responses_skipped":    skipped,
		"unique_call_records":  len(allRecords),
		"total_calls":          summary.Total,
		"connected_calls":      summary.Connected,
		"failed_calls":         summary.Failed,
		"no_answer_calls":      summary.NoAnswer,
		"busy_calls":           summary.Busy,
		"dropped_leg1":         summary.DroppedLeg1,
		"dropped_leg2":         summary.DroppedLeg2,
		"leg1_no_answer":       summary.Leg1NoAnswer,
		"leg1_busy":            summary.Leg1Busy,
		"leg2_no_answer":       summary.Leg2NoAnswer,
		"leg2_busy":            summary.Leg2Busy,
		"leg2_canceled":        summary.Leg2Canceled,
		"answer_rate_pct":      fmt.Sprintf("%.2f", summary.AnswerRate),
		"drop_rate_pct":        fmt.Sprintf("%.2f", summary.DropRate),
		"leg1_drop_rate_pct":   fmt.Sprintf("%.2f", summary.Leg1DropRate),
	})
}

// callLogToRecord converts a stored CallLog row back into an exotel.CallRecord so
// ProcessCallRecords can recompute the full summary from call_logs as a fallback
// when job_responses are unavailable.
func callLogToRecord(cl models.CallLog) exotel.CallRecord {
	rec := exotel.CallRecord{
		Sid:           cl.CallSID,
		ParentCallSid: cl.ParentCallSID,
		From:          cl.CallFrom,
		To:            cl.CallTo,
		Status:        cl.Status,
		Direction:     cl.Direction,
		Duration:      cl.DurationSec,
		Details: exotel.CallDetails{
			ConversationDuration: cl.ConversationDuration,
			Leg1Status:           cl.Leg1Status,
			Leg2Status:           cl.Leg2Status,
		},
	}
	if cl.StartTime != nil {
		rec.StartTime = cl.StartTime.Format("2006-01-02 15:04:05")
	}
	return rec
}
