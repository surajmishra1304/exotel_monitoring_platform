package models

import "time"

type PriorityConfig struct {
	ID               uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Priority         string    `gorm:"column:priority;uniqueIndex;not null" json:"priority"`
	FrequencyMinutes int       `gorm:"column:frequency_minutes;not null;default:60" json:"frequency_minutes"`
	Description      string    `gorm:"column:description" json:"description"`
	UpdatedAt        time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	UpdatedBy        string    `gorm:"column:updated_by;default:SYSTEM" json:"updated_by"`
}

func (PriorityConfig) TableName() string { return "priority_config" }

// ConfigChangeLog records every mutation to configuration entities (priority_config, accounts, etc.).
type ConfigChangeLog struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	EntityType   string    `gorm:"column:entity_type;not null" json:"entity_type"`
	EntityID     uint64    `gorm:"column:entity_id;not null" json:"entity_id"`
	OldValue     string    `gorm:"column:old_value;type:json" json:"old_value"`
	NewValue     string    `gorm:"column:new_value;type:json" json:"new_value"`
	ChangedBy    string    `gorm:"column:changed_by;not null" json:"changed_by"`
	ChangedAt    time.Time `gorm:"column:changed_at;autoCreateTime" json:"changed_at"`
	ChangeReason string    `gorm:"column:change_reason" json:"change_reason"`
}

func (ConfigChangeLog) TableName() string { return "config_change_logs" }
