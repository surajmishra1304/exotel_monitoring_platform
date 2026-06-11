package repository

import (
	"exotel-monitoring-platform/internal/database"
	"exotel-monitoring-platform/internal/models"
	"gorm.io/gorm"
)

// CreateAlert inserts a new alert.
func CreateAlert(a *models.Alert) error {
	return database.DB.Create(a).Error
}

// CreateDispatchLog records the result of an alert dispatch attempt.
func CreateDispatchLog(log *models.AlertDispatchLog) error {
	return database.DB.Create(log).Error
}

// GetActiveAlerts returns all OPEN alerts ordered by severity then triggered_at.
func GetActiveAlerts() ([]models.Alert, error) {
	var list []models.Alert
	result := database.DB.
		Where("alert_status = ?", models.AlertStatusOpen).
		Order("FIELD(severity,'CRITICAL','WARNING','INFO'), triggered_at DESC").
		Find(&list)
	return list, result.Error
}

// GetAlertsByExophone returns alerts for a specific exophone.
func GetAlertsByExophone(exophoneID uint64, limit int) ([]models.Alert, error) {
	if limit <= 0 {
		limit = 50
	}
	var list []models.Alert
	result := database.DB.
		Where("exophone_id = ?", exophoneID).
		Order("triggered_at DESC").
		Limit(limit).
		Find(&list)
	return list, result.Error
}

// AcknowledgeAlert marks an alert as acknowledged.
// Returns gorm.ErrRecordNotFound if no alert with that ID exists.
func AcknowledgeAlert(alertID uint64, by string) error {
	result := database.DB.Model(&models.Alert{}).
		Where("id = ?", alertID).
		Updates(map[string]interface{}{
			"alert_status": models.AlertStatusAcknowledged,
			"updated_by":   by,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
