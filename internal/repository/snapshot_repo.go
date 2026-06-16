package repository

import (
	"encoding/json"
	"time"

	"exotel-monitoring-platform/internal/database"
	"exotel-monitoring-platform/internal/models"
)

// UpsertCallMetricsSnapshot creates or updates today's call metrics snapshot for an exophone.
func UpsertCallMetricsSnapshot(s *models.CallMetricsSnapshot) error {
	return database.DB.Exec(`
		INSERT INTO call_metrics_snapshot
			(snapshot_date, account_id, exophone_id, total_calls,
			 leg1_total, leg1_drops,
			 leg2_total, leg2_drops,
			 connected_calls, dropped_calls,
			 failed_calls, no_answer_calls, busy_calls, canceled_calls, other_calls,
			 avg_duration_sec, answer_rate, drop_rate, leg1_drop_rate, success_rate,
			 hourly_distribution, peak_hour,
			 leg1_no_answer, leg1_busy, leg1_failed,
			 leg2_no_answer, leg2_busy, leg2_failed, leg2_canceled,
			 leg_breakdown,
			 created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
		ON DUPLICATE KEY UPDATE
			total_calls         = VALUES(total_calls),
			leg1_total          = VALUES(leg1_total),
			leg1_drops          = VALUES(leg1_drops),
			leg2_total          = VALUES(leg2_total),
			leg2_drops          = VALUES(leg2_drops),
			connected_calls     = VALUES(connected_calls),
			dropped_calls       = VALUES(dropped_calls),
			failed_calls        = VALUES(failed_calls),
			no_answer_calls     = VALUES(no_answer_calls),
			busy_calls          = VALUES(busy_calls),
			canceled_calls      = VALUES(canceled_calls),
			other_calls         = VALUES(other_calls),
			avg_duration_sec    = VALUES(avg_duration_sec),
			answer_rate         = VALUES(answer_rate),
			drop_rate           = VALUES(drop_rate),
			leg1_drop_rate      = VALUES(leg1_drop_rate),
			success_rate        = VALUES(success_rate),
			hourly_distribution = VALUES(hourly_distribution),
			peak_hour           = VALUES(peak_hour),
			leg1_no_answer      = VALUES(leg1_no_answer),
			leg1_busy           = VALUES(leg1_busy),
			leg1_failed         = VALUES(leg1_failed),
			leg2_no_answer      = VALUES(leg2_no_answer),
			leg2_busy           = VALUES(leg2_busy),
			leg2_failed         = VALUES(leg2_failed),
			leg2_canceled       = VALUES(leg2_canceled),
			leg_breakdown       = VALUES(leg_breakdown),
			updated_at          = NOW()`,
		s.SnapshotDate, s.AccountID, s.ExophoneID, s.TotalCalls,
		s.Leg1Total, s.Leg1Drops,
		s.Leg2Total, s.Leg2Drops,
		s.ConnectedCalls, s.DroppedCalls,
		s.FailedCalls, s.NoAnswerCalls, s.BusyCalls, s.CanceledCalls, s.OtherCalls,
		s.AvgDurationSec, s.AnswerRate, s.DropRate, s.Leg1DropRate, s.SuccessRate,
		s.HourlyDistribution, s.PeakHour,
		s.Leg1NoAnswer, s.Leg1Busy, s.Leg1Failed,
		s.Leg2NoAnswer, s.Leg2Busy, s.Leg2Failed, s.Leg2Canceled,
		s.LegBreakdown,
	).Error
}

// hourlyRow is used to scan per-hour call counts from call_logs.
type hourlyRow struct {
	Hour  int `gorm:"column:h"`
	Count int `gorm:"column:c"`
}

// RecomputeCallMetricsFromLogs rebuilds today's call_metrics_snapshot by aggregating
// all call_logs for today — not just the current batch. This ensures the snapshot is
// always the accurate day-total regardless of how many windows have run.
func RecomputeCallMetricsFromLogs(accountID, exophoneID uint64) error {
	return RecomputeCallMetricsForDate(accountID, exophoneID, "")
}

// RecomputeCallMetricsForDate rebuilds the call_metrics_snapshot for a specific date
// (YYYY-MM-DD). Empty dateStr defaults to today.
// All Exotel call statuses are aggregated; dropped_calls counts completed calls with
// duration_sec <= 5 which indicates a call drop.
func RecomputeCallMetricsForDate(accountID, exophoneID uint64, dateStr string) error {
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	// Step 1: aggregate all status + per-leg drop counts from call_logs.
	// Leg 1 = call_to matches the VN (trigger/app-based call TO the VN).
	// Leg 2 = call_to does not match the VN (bridged conversation leg).
	// Drops on Leg 2 = customer-side drop (didn't pick up / call fell within DropThresholdSec).
	type agg struct {
		Total        int     `gorm:"column:total"`
		Leg1Total    int     `gorm:"column:leg1_total"`
		Leg1Drops    int     `gorm:"column:leg1_drops"`
		Leg2Total    int     `gorm:"column:leg2_total"`
		Leg2Drops    int     `gorm:"column:leg2_drops"`
		Connected    int     `gorm:"column:connected"`
		Failed       int     `gorm:"column:failed"`
		NoAnswer     int     `gorm:"column:no_answer"`
		Busy         int     `gorm:"column:busy"`
		Canceled     int     `gorm:"column:canceled"`
		Other        int     `gorm:"column:other"`
		AvgDur       float64 `gorm:"column:avg_dur"`
		Leg1NoAnswer int     `gorm:"column:leg1_no_answer"`
		Leg1Busy     int     `gorm:"column:leg1_busy"`
		Leg1Failed   int     `gorm:"column:leg1_failed"`
		Leg2NoAnswer int     `gorm:"column:leg2_no_answer"`
		Leg2Busy     int     `gorm:"column:leg2_busy"`
		Leg2Failed   int     `gorm:"column:leg2_failed"`
		Leg2Canceled int     `gorm:"column:leg2_canceled"`
	}
	var a agg
	err := database.DB.Raw(`
		SELECT
			COUNT(*) AS total,
			SUM(leg_number = 1) AS leg1_total,
			-- Leg1 drop: IVR inbound where customer abandoned before agent routing (leg2_status is null/empty)
			-- OR Panel mode leg1 where agent didn't answer (leg2_status is null = Leg2 never created)
			SUM(status = 'completed' AND conversation_duration = 0
			    AND (leg2_status IS NULL OR leg2_status = '')) AS leg1_drops,
			SUM(leg_number = 2) AS leg2_total,
			-- Leg2 drop: agent routing was attempted (leg2_status populated) but no conversation happened
			-- OR Panel mode leg2 where customer didn't answer
			SUM(status = 'completed' AND conversation_duration = 0
			    AND leg2_status IS NOT NULL AND leg2_status != '') AS leg2_drops,
			-- connected = ConversationDuration > 0: both parties were actually bridged
			SUM(status = 'completed' AND conversation_duration > 0) AS connected,
			SUM(status = 'failed') AS failed,
			SUM(status = 'no-answer') AS no_answer,
			SUM(status = 'busy') AS busy,
			SUM(status = 'canceled') AS canceled,
			SUM(status NOT IN ('completed','failed','no-answer','busy','canceled')) AS other,
			COALESCE(AVG(duration_sec), 0) AS avg_dur,
			SUM(leg_number = 1 AND status = 'no-answer') AS leg1_no_answer,
			SUM(leg_number = 1 AND status = 'busy')      AS leg1_busy,
			SUM(leg_number = 1 AND status = 'failed')    AS leg1_failed,
			SUM(leg_number = 2 AND status = 'no-answer') AS leg2_no_answer,
			SUM(leg_number = 2 AND status = 'busy')      AS leg2_busy,
			SUM(leg_number = 2 AND status = 'failed')    AS leg2_failed,
			SUM(leg_number = 2 AND status = 'canceled')  AS leg2_canceled
		FROM call_logs
		WHERE exophone_id = ? AND DATE(start_time) = ?`,
		exophoneID, dateStr,
	).Scan(&a).Error
	if err != nil {
		return err
	}

	// Step 2: compute hourly distribution.
	var hourlyRows []hourlyRow
	_ = database.DB.Raw(`
		SELECT HOUR(start_time) AS h, COUNT(*) AS c
		FROM call_logs
		WHERE exophone_id = ? AND DATE(start_time) = ?
		GROUP BY HOUR(start_time)`,
		exophoneID, dateStr,
	).Scan(&hourlyRows)

	var dist [24]int
	peakHour, peakCount := 0, 0
	for _, r := range hourlyRows {
		if r.Hour >= 0 && r.Hour < 24 {
			dist[r.Hour] = r.Count
			if r.Count > peakCount {
				peakCount = r.Count
				peakHour = r.Hour
			}
		}
	}
	distJSON, _ := json.Marshal(dist[:])

	// Step 3: compute rates and dropped count.
	// Panel-based calling (Leg 2 records exist): denominator = Leg2Total; dropped = Leg2Drops.
	// IVR/app-based calling (all inbound, leg_number=0): denominator = Total; dropped = Total - Connected.
	answerRate, dropRate, leg1DropRate := 0.0, 0.0, 0.0
	droppedCalls := 0
	denominator := a.Total
	if a.Leg2Total > 0 {
		// Panel mode: measure against Leg2 records (customer-side)
		denominator = a.Leg2Total
		droppedCalls = a.Leg1Drops + a.Leg2Drops
		dropRate = float64(a.Leg2Drops) / float64(a.Leg2Total) * 100
	} else if a.Total > 0 {
		// IVR mode:
		//   drop_rate    = Leg2 drops / Total (agent no-answer rate — the actionable metric)
		//   leg1_drop_rate = Leg1 drops / Total (IVR abandon rate — customer gave up early)
		//   dropped_calls  = Leg1 + Leg2 (total calls that didn't connect)
		droppedCalls = a.Leg1Drops + a.Leg2Drops
		dropRate = float64(a.Leg2Drops) / float64(a.Total) * 100
	}
	if denominator > 0 {
		answerRate = float64(a.Connected) / float64(denominator) * 100
	}
	if a.Leg1Total > 0 {
		leg1DropRate = float64(a.Leg1Drops) / float64(a.Leg1Total) * 100
	} else if a.Total > 0 {
		// Fallback for IVR-mode records inserted before leg_number classification was added.
		leg1DropRate = float64(a.Leg1Drops) / float64(a.Total) * 100
	}

	snapshotDate, _ := time.Parse("2006-01-02", dateStr)

	// Step 4: upsert snapshot.
	return database.DB.Exec(`
		INSERT INTO call_metrics_snapshot
			(snapshot_date, account_id, exophone_id, total_calls,
			 leg1_total, leg1_drops, leg2_total, leg2_drops,
			 connected_calls, dropped_calls,
			 failed_calls, no_answer_calls, busy_calls, canceled_calls, other_calls,
			 avg_duration_sec, answer_rate, drop_rate, leg1_drop_rate, success_rate,
			 hourly_distribution, peak_hour,
			 leg1_no_answer, leg1_busy, leg1_failed,
			 leg2_no_answer, leg2_busy, leg2_failed, leg2_canceled,
			 created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
		ON DUPLICATE KEY UPDATE
			total_calls         = VALUES(total_calls),
			leg1_total          = VALUES(leg1_total),
			leg1_drops          = VALUES(leg1_drops),
			leg2_total          = VALUES(leg2_total),
			leg2_drops          = VALUES(leg2_drops),
			connected_calls     = VALUES(connected_calls),
			dropped_calls       = VALUES(dropped_calls),
			failed_calls        = VALUES(failed_calls),
			no_answer_calls     = VALUES(no_answer_calls),
			busy_calls          = VALUES(busy_calls),
			canceled_calls      = VALUES(canceled_calls),
			other_calls         = VALUES(other_calls),
			avg_duration_sec    = VALUES(avg_duration_sec),
			answer_rate         = VALUES(answer_rate),
			drop_rate           = VALUES(drop_rate),
			leg1_drop_rate      = VALUES(leg1_drop_rate),
			success_rate        = VALUES(success_rate),
			hourly_distribution = VALUES(hourly_distribution),
			peak_hour           = VALUES(peak_hour),
			leg1_no_answer      = VALUES(leg1_no_answer),
			leg1_busy           = VALUES(leg1_busy),
			leg1_failed         = VALUES(leg1_failed),
			leg2_no_answer      = VALUES(leg2_no_answer),
			leg2_busy           = VALUES(leg2_busy),
			leg2_failed         = VALUES(leg2_failed),
			leg2_canceled       = VALUES(leg2_canceled),
			updated_at          = NOW()`,
		snapshotDate, accountID, exophoneID, a.Total,
		a.Leg1Total, a.Leg1Drops, a.Leg2Total, a.Leg2Drops,
		a.Connected, droppedCalls,
		a.Failed, a.NoAnswer, a.Busy, a.Canceled, a.Other,
		a.AvgDur, answerRate, dropRate, leg1DropRate, answerRate,
		string(distJSON), peakHour,
		a.Leg1NoAnswer, a.Leg1Busy, a.Leg1Failed,
		a.Leg2NoAnswer, a.Leg2Busy, a.Leg2Failed, a.Leg2Canceled,
	).Error
}

// RecomputeAccountDashboardSnapshot rebuilds today's account-level snapshot by aggregating
// all call_logs and exophone data for the account.
func RecomputeAccountDashboardSnapshot(accountID uint64) error {
	return database.DB.Exec(`
		INSERT INTO account_dashboard_snapshot
			(snapshot_date, account_id, total_exophones, active_exophones,
			 total_calls, failed_calls, success_rate, created_at, updated_at)
		SELECT
			DATE(NOW()), ?,
			(SELECT COUNT(*) FROM exophones WHERE account_id = ? AND is_deleted = 0),
			(SELECT COUNT(*) FROM exophones WHERE account_id = ? AND is_deleted = 0 AND status = 'active'),
			COALESCE(SUM(t.total), 0),
			COALESCE(SUM(t.failed) + SUM(t.no_answer), 0),
			COALESCE(ROUND(SUM(t.connected) * 100.0 / NULLIF(SUM(t.total), 0), 2), 0),
			NOW(), NOW()
		FROM (
			SELECT
				COUNT(*) as total,
				SUM(status = 'completed' AND conversation_duration > 0) as connected,
				SUM(status = 'failed') as failed,
				SUM(status = 'no-answer') as no_answer
			FROM call_logs
			WHERE account_id = ? AND DATE(start_time) = DATE(NOW())
		) t
		ON DUPLICATE KEY UPDATE
			total_exophones  = VALUES(total_exophones),
			active_exophones = VALUES(active_exophones),
			total_calls      = VALUES(total_calls),
			failed_calls     = VALUES(failed_calls),
			success_rate     = VALUES(success_rate),
			updated_at       = NOW()`,
		accountID, accountID, accountID, accountID,
	).Error
}

// AccumulateAccountDashboardFromSnapshots rebuilds today's account-level snapshot by
// aggregating call_metrics_snapshot rows — never reads call_logs directly.
func AccumulateAccountDashboardFromSnapshots(accountID uint64) error {
	return database.DB.Exec(`
		INSERT INTO account_dashboard_snapshot
			(snapshot_date, account_id, total_exophones, active_exophones,
			 total_calls, failed_calls, success_rate, created_at, updated_at)
		SELECT
			DATE(NOW()), ?,
			(SELECT COUNT(*) FROM exophones WHERE account_id = ? AND is_deleted = 0),
			(SELECT COUNT(*) FROM exophones WHERE account_id = ? AND is_deleted = 0 AND status = 'active'),
			COALESCE(SUM(cms.total_calls), 0),
			COALESCE(SUM(cms.failed_calls + cms.no_answer_calls), 0),
			COALESCE(ROUND(SUM(cms.connected_calls) * 100.0 / NULLIF(SUM(cms.total_calls), 0), 2), 0),
			NOW(), NOW()
		FROM call_metrics_snapshot cms
		WHERE cms.account_id = ? AND cms.snapshot_date = DATE(NOW())
		ON DUPLICATE KEY UPDATE
			total_exophones  = VALUES(total_exophones),
			active_exophones = VALUES(active_exophones),
			total_calls      = VALUES(total_calls),
			failed_calls     = VALUES(failed_calls),
			success_rate     = VALUES(success_rate),
			updated_at       = NOW()`,
		accountID, accountID, accountID, accountID,
	).Error
}

// UpsertExophoneHealthSnapshot creates or updates the health snapshot for an exophone.
func UpsertExophoneHealthSnapshot(s *models.ExophoneHealthSnapshot) error {
	return database.DB.Exec(`
		INSERT INTO exophone_health_snapshot
			(exophone_id, account_id, heartbeat_status, availability_percent,
			 last_api_latency_ms, active_streams, last_checked_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW(), NOW())
		ON DUPLICATE KEY UPDATE
			heartbeat_status      = VALUES(heartbeat_status),
			availability_percent  = VALUES(availability_percent),
			last_api_latency_ms   = VALUES(last_api_latency_ms),
			active_streams        = VALUES(active_streams),
			last_checked_at       = NOW(),
			updated_at            = NOW()`,
		s.ExophoneID, s.AccountID, s.HeartbeatStatus, s.AvailabilityPercent,
		s.LastAPILatencyMs, s.ActiveStreams,
	).Error
}

// UpsertStreamUtilizationSnapshot creates or updates the hourly stream snapshot.
func UpsertStreamUtilizationSnapshot(s *models.StreamUtilizationSnapshot) error {
	return database.DB.Exec(`
		INSERT INTO stream_utilization_snapshot
			(snapshot_hour, account_id, exophone_id, avg_active_streams,
			 max_active_streams, utilization_percent, created_at)
		VALUES (?, ?, ?, ?, ?, ?, NOW())
		ON DUPLICATE KEY UPDATE
			avg_active_streams    = VALUES(avg_active_streams),
			max_active_streams    = VALUES(max_active_streams),
			utilization_percent   = VALUES(utilization_percent)`,
		s.SnapshotHour, s.AccountID, s.ExophoneID,
		s.AvgActiveStreams, s.MaxActiveStreams, s.UtilizationPercent,
	).Error
}

// UpsertAccountDashboardSnapshot creates or updates the account daily snapshot.
func UpsertAccountDashboardSnapshot(s *models.AccountDashboardSnapshot) error {
	return database.DB.Exec(`
		INSERT INTO account_dashboard_snapshot
			(snapshot_date, account_id, total_exophones, active_exophones, created_at, updated_at)
		VALUES (?, ?, ?, ?, NOW(), NOW())
		ON DUPLICATE KEY UPDATE
			total_exophones  = VALUES(total_exophones),
			active_exophones = VALUES(active_exophones),
			updated_at       = NOW()`,
		s.SnapshotDate, s.AccountID, s.TotalExophones, s.ActiveExophones,
	).Error
}

// GetCallMetricsSnapshot returns the snapshot for a given exophone and date.
func GetCallMetricsSnapshot(exophoneID uint64, date time.Time) (*models.CallMetricsSnapshot, error) {
	var s models.CallMetricsSnapshot
	result := database.DB.
		Where("exophone_id = ? AND snapshot_date = ?", exophoneID, date.Format("2006-01-02")).
		First(&s)
	return &s, result.Error
}

// GetLatestCallMetricsSnapshot returns the most recent snapshot for an exophone
// regardless of date. Used as a fallback when today's snapshot hasn't been generated yet.
func GetLatestCallMetricsSnapshot(exophoneID uint64) (*models.CallMetricsSnapshot, error) {
	var s models.CallMetricsSnapshot
	result := database.DB.
		Where("exophone_id = ?", exophoneID).
		Order("snapshot_date DESC").
		First(&s)
	return &s, result.Error
}

// GetAccountDashboardSnapshot returns today's account-level snapshot.
func GetAccountDashboardSnapshot(accountID uint64, date time.Time) (*models.AccountDashboardSnapshot, error) {
	var s models.AccountDashboardSnapshot
	result := database.DB.
		Where("account_id = ? AND snapshot_date = ?", accountID, date.Format("2006-01-02")).
		First(&s)
	return &s, result.Error
}

// GetExophoneHealthSnapshot returns the health snapshot for a given exophone.
func GetExophoneHealthSnapshot(exophoneID uint64) (*models.ExophoneHealthSnapshot, error) {
	var s models.ExophoneHealthSnapshot
	result := database.DB.
		Where("exophone_id = ?", exophoneID).
		First(&s)
	return &s, result.Error
}

// GetAllExophoneHealthSnapshots returns health snapshots for all exophones of an account.
func GetAllExophoneHealthSnapshots(accountID uint64) ([]models.ExophoneHealthSnapshot, error) {
	var list []models.ExophoneHealthSnapshot
	result := database.DB.
		Where("account_id = ?", accountID).
		Find(&list)
	return list, result.Error
}

// GetCallMetricsByDate returns call metrics snapshots for an account on a given date with pagination.
// dateStr must be in "YYYY-MM-DD" format; empty string defaults to today.
func GetCallMetricsByDate(accountID uint64, dateStr string, page, limit int) ([]models.CallMetricsSnapshot, int64, error) {
	var list []models.CallMetricsSnapshot
	var total int64

	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	q := database.DB.Model(&models.CallMetricsSnapshot{}).
		Where("account_id = ? AND snapshot_date = ?", accountID, dateStr)

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	result := q.Order("exophone_id ASC").
		Offset(offset).
		Limit(limit).
		Find(&list)

	return list, total, result.Error
}

// CallMetricsExportRow holds snapshot data joined with the exophone number for CSV export.
type CallMetricsExportRow struct {
	ExophoneNumber string    `gorm:"column:exophone_number"`
	ExophoneID     uint64    `gorm:"column:exophone_id"`
	SnapshotDate   time.Time `gorm:"column:snapshot_date"`
	TotalCalls     int       `gorm:"column:total_calls"`
	ConnectedCalls int       `gorm:"column:connected_calls"`
	Leg1Total      int       `gorm:"column:leg1_total"`
	Leg1Drops      int       `gorm:"column:leg1_drops"`
	Leg1DropRate   float64   `gorm:"column:leg1_drop_rate"`
	Leg2Total      int       `gorm:"column:leg2_total"`
	Leg2Drops      int       `gorm:"column:leg2_drops"`
	DropRate       float64   `gorm:"column:drop_rate"`
	FailedCalls    int       `gorm:"column:failed_calls"`
	NoAnswerCalls  int       `gorm:"column:no_answer_calls"`
	BusyCalls      int       `gorm:"column:busy_calls"`
	CanceledCalls  int       `gorm:"column:canceled_calls"`
	AnswerRate     float64   `gorm:"column:answer_rate"`
	AvgDurationSec float64   `gorm:"column:avg_duration_sec"`
	PeakHour       int       `gorm:"column:peak_hour"`
	Leg1NoAnswer   int       `gorm:"column:leg1_no_answer"`
	Leg1Busy       int       `gorm:"column:leg1_busy"`
	Leg1Failed     int       `gorm:"column:leg1_failed"`
	Leg2NoAnswer   int       `gorm:"column:leg2_no_answer"`
	Leg2Busy       int       `gorm:"column:leg2_busy"`
	Leg2Failed     int       `gorm:"column:leg2_failed"`
	Leg2Canceled   int       `gorm:"column:leg2_canceled"`
}

// GetCallMetricsForExport returns snapshot rows joined with exophone numbers for CSV export.
// Pass exophoneID=0 to return all exophones for the account.
func GetCallMetricsForExport(accountID uint64, dateStr string, exophoneID uint64) ([]CallMetricsExportRow, error) {
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	rawSQL := `
		SELECT e.exophone_number, cms.exophone_id, cms.snapshot_date,
		       cms.total_calls, cms.connected_calls,
		       cms.leg1_total, cms.leg1_drops, cms.leg1_drop_rate,
		       cms.leg2_total, cms.leg2_drops, cms.drop_rate,
		       cms.failed_calls, cms.no_answer_calls, cms.busy_calls, cms.canceled_calls,
		       cms.answer_rate, cms.avg_duration_sec, cms.peak_hour,
		       cms.leg1_no_answer, cms.leg1_busy, cms.leg1_failed,
		       cms.leg2_no_answer, cms.leg2_busy, cms.leg2_failed, cms.leg2_canceled
		FROM call_metrics_snapshot cms
		JOIN exophones e ON e.id = cms.exophone_id
		WHERE cms.account_id = ? AND cms.snapshot_date = ?`

	args := []interface{}{accountID, dateStr}
	if exophoneID != 0 {
		rawSQL += " AND cms.exophone_id = ?"
		args = append(args, exophoneID)
	}
	rawSQL += " ORDER BY e.exophone_number ASC"

	var rows []CallMetricsExportRow
	result := database.DB.Raw(rawSQL, args...).Scan(&rows)
	return rows, result.Error
}

// GetJobResponsesForReprocess returns all successful CALLS job_responses for a
// given exophone on a given date. Rows are ordered oldest-first so pages that
// were fetched earlier in the day come before later ones.
// GetCallLogsForReprocess returns all call_logs for an exophone on a given date.
// Used as a fallback in the reprocess endpoint when job_responses are unavailable
// (e.g. exophone had skip_call_logs_write=false so raw pages were not saved).
func GetCallLogsForReprocess(exophoneID uint64, dateStr string) ([]models.CallLog, error) {
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}
	var logs []models.CallLog
	result := database.DB.
		Where("exophone_id = ? AND DATE(start_time) = ?", exophoneID, dateStr).
		Order("start_time ASC").
		Find(&logs)
	return logs, result.Error
}

func GetJobResponsesForReprocess(exophoneID uint64, dateStr string) ([]models.JobResponse, error) {
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}
	var responses []models.JobResponse
	// Match on either:
	//   (a) live jobs where the job ran on that date (DATE(jt.started_at) = dateStr), or
	//   (b) backfill jobs whose transaction_id encodes the data date (BACKFILL-{id}-{date}).
	result := database.DB.Raw(`
		SELECT jr.id, jr.transaction_id, jr.response_type, jr.http_status,
		       jr.raw_response, jr.parsed_response, jr.created_at
		FROM job_responses jr
		JOIN job_transactions jt ON jt.transaction_id = jr.transaction_id
		WHERE jt.exophone_id = ?
		  AND jr.response_type = 'CALLS'
		  AND jr.http_status BETWEEN 200 AND 299
		  AND (DATE(jt.started_at) = ? OR jt.transaction_id = CONCAT('BACKFILL-', ?, '-', ?))
		ORDER BY jt.started_at ASC`,
		exophoneID, dateStr, exophoneID, dateStr,
	).Scan(&responses)
	return responses, result.Error
}
