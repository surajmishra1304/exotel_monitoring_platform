package repository

import (
	"exotel-monitoring-platform/internal/database"
	"exotel-monitoring-platform/internal/models"
)

// SaveHeartbeatMetric persists a heartbeat reading.
func SaveHeartbeatMetric(m *models.HeartbeatMetric) error {
	return database.DB.Create(m).Error
}

// SaveStreamMetric persists a stream utilization reading.
func SaveStreamMetric(m *models.StreamMetric) error {
	return database.DB.Create(m).Error
}

// SaveMetric persists a generic aggregated metric.
func SaveMetric(m *models.Metric) error {
	return database.DB.Create(m).Error
}

// SaveMetrics bulk-inserts multiple metrics.
func SaveMetrics(metrics []models.Metric) error {
	if len(metrics) == 0 {
		return nil
	}
	return database.DB.CreateInBatches(metrics, 200).Error
}

// GetLatestHeartbeat returns the most recent heartbeat record for an account.
func GetLatestHeartbeat(accountID uint64) (*models.HeartbeatMetric, error) {
	var m models.HeartbeatMetric
	result := database.DB.
		Where("account_id = ?", accountID).
		Order("created_at DESC").
		First(&m)
	return &m, result.Error
}

// GetLatestHeartbeatForExophone returns the most recent heartbeat record for a specific exophone.
func GetLatestHeartbeatForExophone(exophoneID uint64) (*models.HeartbeatMetric, error) {
	var m models.HeartbeatMetric
	result := database.DB.
		Where("exophone_id = ?", exophoneID).
		Order("created_at DESC").
		First(&m)
	return &m, result.Error
}

// GetLatestStreamMetric returns the most recent stream reading for an exophone.
func GetLatestStreamMetric(exophoneID uint64) (*models.StreamMetric, error) {
	var m models.StreamMetric
	result := database.DB.
		Where("exophone_id = ?", exophoneID).
		Order("created_at DESC").
		First(&m)
	return &m, result.Error
}
