package repository

import (
	"time"

	"exotel-monitoring-platform/internal/database"
	"exotel-monitoring-platform/internal/models"
	"exotel-monitoring-platform/internal/utils"
	"gorm.io/gorm"
)

// GetMonitoredExophones returns all active exophones with monitoring enabled.
func GetMonitoredExophones() ([]models.Exophone, error) {
	var list []models.Exophone
	result := database.DB.
		Where("monitoring_flag = 1 AND is_deleted = 0").
		Order("priority ASC").
		Find(&list)
	return list, result.Error
}

// GetExophonesByAccount returns paginated active exophones for a given account, ordered by priority.
func GetExophonesByAccount(accountID uint64, page, limit int) ([]models.Exophone, int64, error) {
	var list []models.Exophone
	var total int64

	q := database.DB.Model(&models.Exophone{}).
		Where("account_id = ? AND is_deleted = 0", accountID)

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	result := q.Order("priority ASC").
		Offset(offset).
		Limit(limit).
		Find(&list)

	return list, total, result.Error
}

// GetExophoneByID retrieves a single exophone.
func GetExophoneByID(id uint64) (*models.Exophone, error) {
	var e models.Exophone
	result := database.DB.
		Where("id = ? AND is_deleted = 0", id).
		First(&e)
	return &e, result.Error
}

// GetExophonesByNumber returns all active exophones matching a phone number.
// The number is normalised before querying so that 07948224931, +917948224931,
// and 917948224931 all resolve to the same DB row.
func GetExophonesByNumber(number string) ([]models.Exophone, int64, error) {
	canonical := utils.NormalizePhone(number)
	var list []models.Exophone
	var total int64
	result := database.DB.Model(&models.Exophone{}).
		Where("exophone_number = ? AND is_deleted = 0", canonical).
		Count(&total).
		Find(&list)
	return list, total, result.Error
}

// UpsertExophone creates or updates an exophone record.
// The exophone number is normalised to the canonical format before saving.
func UpsertExophone(e *models.Exophone) error {
	e.ExophoneNumber = utils.NormalizePhone(e.ExophoneNumber)
	return database.DB.Save(e).Error
}

// UpdatePriority changes the priority on both the exophone and all its monitoring jobs,
// and updates the frequency_minutes on the jobs.
func UpdatePriority(exophoneID uint64, priority string, freqMinutes int) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Exophone{}).
			Where("id = ? AND is_deleted = 0", exophoneID).
			Updates(map[string]interface{}{
				"priority":   priority,
				"updated_by": "API",
			}).Error; err != nil {
			return err
		}
		return tx.Model(&models.MonitoringJob{}).
			Where("exophone_id = ? AND is_deleted = 0", exophoneID).
			Updates(map[string]interface{}{
				"priority":         priority,
				"frequency_minutes": freqMinutes,
				"updated_by":       "API",
			}).Error
	})
}

// SetCronApplicable updates the is_cron_applicable flag for a given exophone
// and syncs is_active on all its monitoring_jobs so the scheduler immediately
// respects the toggle without requiring a manual job-level fix.
func SetCronApplicable(exophoneID uint64, val int) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Exophone{}).
			Where("id = ? AND is_deleted = 0", exophoneID).
			Updates(map[string]interface{}{
				"is_cron_applicable": val,
				"updated_by":         "API",
			}).Error; err != nil {
			return err
		}
		return tx.Model(&models.MonitoringJob{}).
			Where("exophone_id = ? AND is_deleted = 0", exophoneID).
			Updates(map[string]interface{}{
				"is_active":  val,
				"updated_by": "API",
			}).Error
	})
}

// SetSkipCallLogs updates the skip_call_logs flag for a given exophone.
// When val=1 the call_logs table will not be written during monitoring jobs.
func SetSkipCallLogs(exophoneID uint64, val int) error {
	return database.DB.Model(&models.Exophone{}).
		Where("id = ? AND is_deleted = 0", exophoneID).
		Updates(map[string]interface{}{
			"skip_call_logs": val,
			"updated_by":     "API",
		}).Error
}

// UpdateLastSynced stamps the last_synced_at field for a given exophone.
func UpdateLastSynced(exophoneID uint64) error {
	now := time.Now()
	return database.DB.Model(&models.Exophone{}).
		Where("id = ?", exophoneID).
		Updates(map[string]interface{}{
			"last_synced_at": now,
			"updated_by":     "SYSTEM_SYNC",
		}).Error
}
