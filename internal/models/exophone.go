package models

import "time"

type Exophone struct {
	ID             uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	AccountID      uint64     `gorm:"column:account_id;not null;index" json:"account_id"`
	ExophoneNumber string     `gorm:"column:exophone_number;not null" json:"exophone_number"`
	Status         string     `gorm:"column:status;default:active" json:"status"`
	Priority       string     `gorm:"column:priority;default:P1" json:"priority"`
	MonitoringFlag     int        `gorm:"column:monitoring_flag;default:1" json:"monitoring_flag"`
	IsCronApplicable   int        `gorm:"column:is_cron_applicable;default:0" json:"is_cron_applicable"`
	CacheTTL       int        `gorm:"column:cache_ttl;default:900" json:"cache_ttl"`
	MetadataJSON   string     `gorm:"column:metadata_json;type:json" json:"metadata_json,omitempty"`
	LastSyncedAt   *time.Time `gorm:"column:last_synced_at" json:"last_synced_at"`
	IsDeleted      int        `gorm:"column:is_deleted;default:0" json:"-"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	CreatedBy      string     `gorm:"column:created_by;default:SYSTEM" json:"created_by"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	UpdatedBy      string     `gorm:"column:updated_by;default:SYSTEM" json:"updated_by"`
}

func (Exophone) TableName() string { return "exophones" }
