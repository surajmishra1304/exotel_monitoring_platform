package repository

import (
	"fmt"

	"exotel-monitoring-platform/internal/database"
	"exotel-monitoring-platform/internal/models"
)

// GetAllPriorityConfigs returns all four priority config rows ordered by priority.
func GetAllPriorityConfigs() ([]models.PriorityConfig, error) {
	var list []models.PriorityConfig
	result := database.DB.Order("priority ASC").Find(&list)
	return list, result.Error
}

// GetPriorityConfigMap returns a map[priority]frequency_minutes for fast lookup.
func GetPriorityConfigMap() (map[string]int, error) {
	configs, err := GetAllPriorityConfigs()
	if err != nil {
		return nil, err
	}
	m := make(map[string]int, len(configs))
	for _, c := range configs {
		m[c.Priority] = c.FrequencyMinutes
	}
	return m, nil
}

// GetPriorityConfigByPriority returns the config row for a single priority level.
func GetPriorityConfigByPriority(priority string) (*models.PriorityConfig, error) {
	var cfg models.PriorityConfig
	result := database.DB.Where("priority = ?", priority).First(&cfg)
	return &cfg, result.Error
}

// UpdatePriorityConfig updates the frequency_minutes for a given priority level.
func UpdatePriorityConfig(priority string, freqMinutes int, updatedBy string) error {
	return database.DB.Model(&models.PriorityConfig{}).
		Where("priority = ?", priority).
		Updates(map[string]interface{}{
			"frequency_minutes": freqMinutes,
			"updated_by":        updatedBy,
		}).Error
}

// SaveConfigChangeLog appends one audit record to config_change_logs.
// oldFreq and newFreq are the before/after frequency_minutes values.
func SaveConfigChangeLog(entityType, entityKey string, entityID uint64, oldFreq, newFreq int, changedBy, reason string) error {
	return database.DB.Create(&models.ConfigChangeLog{
		EntityType:   entityType,
		EntityID:     entityID,
		OldValue:     fmt.Sprintf(`{"frequency_minutes":%d,"priority":"%s"}`, oldFreq, entityKey),
		NewValue:     fmt.Sprintf(`{"frequency_minutes":%d,"priority":"%s"}`, newFreq, entityKey),
		ChangedBy:    changedBy,
		ChangeReason: reason,
	}).Error
}
