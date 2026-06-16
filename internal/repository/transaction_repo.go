package repository

import (
	"fmt"
	"strings"
	"time"

	"exotel-monitoring-platform/internal/database"
	"exotel-monitoring-platform/internal/models"
)

// CreateTransaction inserts a new job transaction record.
func CreateTransaction(t *models.JobTransaction) error {
	return database.DB.Create(t).Error
}

// UpdateTransactionStatus updates status, latency, retry count and completion time.
func UpdateTransactionStatus(txnID string, status string, latencyMs *int64, retryCount int, errMsg string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":       status,
		"retry_count":  retryCount,
		"completed_at": now,
		"updated_by":   "SYSTEM_SCHEDULER",
	}
	if latencyMs != nil {
		updates["api_latency_ms"] = *latencyMs
	}
	if errMsg != "" {
		updates["error_message"] = errMsg
	}
	return database.DB.Model(&models.JobTransaction{}).
		Where("transaction_id = ?", txnID).
		Updates(updates).Error
}

// SaveJobResponse stores the raw + parsed response for a transaction.
func SaveJobResponse(r *models.JobResponse) error {
	return database.DB.Create(r).Error
}

// SaveCallLogs bulk-inserts call log records using INSERT IGNORE to skip duplicates.
// The UNIQUE key uk_cl_exophone_callsid(exophone_id, call_sid) makes this idempotent —
// overlapping 15-minute fetch windows re-submit the same call_sids harmlessly.
func SaveCallLogs(logs []models.CallLog) error {
	if len(logs) == 0 {
		return nil
	}
	const batchSize = 500
	for i := 0; i < len(logs); i += batchSize {
		end := i + batchSize
		if end > len(logs) {
			end = len(logs)
		}
		batch := logs[i:end]
		if err := database.DB.Exec(buildIgnoreInsert(batch)).Error; err != nil {
			return err
		}
	}
	return nil
}

// mysqlEscape escapes a string value for safe interpolation into a MySQL single-quoted literal.
// Doubles single quotes and escapes backslashes per the MySQL string literal rules.
func mysqlEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `''`)
	return s
}

// buildIgnoreInsert constructs a single INSERT IGNORE statement for a batch of CallLog rows.
// leg1_status and leg2_status are persisted so that RecomputeCallMetricsForDate and the
// reprocess fallback path can correctly classify Leg1 vs Leg2 drops via the leg2_status field.
func buildIgnoreInsert(batch []models.CallLog) string {
	// Use raw SQL so we can specify INSERT IGNORE — GORM Clauses(OnConflict{DoNothing:true})
	// generates INSERT INTO ... ON DUPLICATE KEY UPDATE which still touches rows; IGNORE is cheaper.
	q := "INSERT IGNORE INTO call_logs " +
		"(transaction_id,account_id,exophone_id,call_sid,parent_call_sid,leg_number," +
		"call_from,call_to,status,direction," +
		"duration_sec,conversation_duration,start_time,end_time,recording_url," +
		"leg1_status,leg2_status,created_at) VALUES "
	now := "NOW()"
	for i, cl := range batch {
		if i > 0 {
			q += ","
		}
		startTime := "NULL"
		if cl.StartTime != nil {
			startTime = "'" + cl.StartTime.Format("2006-01-02 15:04:05") + "'"
		}
		endTime := "NULL"
		if cl.EndTime != nil {
			endTime = "'" + cl.EndTime.Format("2006-01-02 15:04:05") + "'"
		}
		q += fmt.Sprintf("('%s',%d,%d,'%s','%s',%d,'%s','%s','%s','%s',%d,%d,%s,%s,'%s','%s','%s',%s)",
			mysqlEscape(cl.TransactionID), cl.AccountID, cl.ExophoneID, mysqlEscape(cl.CallSID),
			mysqlEscape(cl.ParentCallSID), cl.LegNumber,
			mysqlEscape(cl.CallFrom), mysqlEscape(cl.CallTo),
			mysqlEscape(cl.Status), mysqlEscape(cl.Direction),
			cl.DurationSec, cl.ConversationDuration,
			startTime, endTime, mysqlEscape(cl.RecordingURL),
			mysqlEscape(cl.Leg1Status), mysqlEscape(cl.Leg2Status), now,
		)
	}
	return q
}

// SaveCallFlowEvent stores a single webhook event.
func SaveCallFlowEvent(e *models.CallFlowEvent) error {
	return database.DB.Create(e).Error
}

// GetTransactions returns transactions with optional filters and pagination.
// Returns (records, totalCount, error).
func GetTransactions(exophoneID uint64, status, jobType string, from, to time.Time, page, limit int) ([]models.JobTransaction, int64, error) {
	q := database.DB.Model(&models.JobTransaction{})
	if exophoneID > 0 {
		q = q.Where("exophone_id = ?", exophoneID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if jobType != "" {
		q = q.Where("job_type = ?", jobType)
	}
	if !from.IsZero() {
		q = q.Where("started_at >= ?", from)
	}
	if !to.IsZero() {
		q = q.Where("started_at <= ?", to)
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	if page < 1 {
		page = 1
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []models.JobTransaction
	result := q.Order("started_at DESC").
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&list)
	return list, total, result.Error
}

// CleanupOldTransactions deletes transactions older than retentionDays.
func CleanupOldTransactions(retentionDays int) error {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	return database.DB.
		Where("created_at < ?", cutoff).
		Delete(&models.JobTransaction{}).Error
}

// CleanupOldResponses deletes job responses older than retentionDays.
func CleanupOldResponses(retentionDays int) error {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	return database.DB.
		Where("created_at < ?", cutoff).
		Delete(&models.JobResponse{}).Error
}
