package models

import "time"

type CallMetricsSnapshot struct {
	ID                  uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	SnapshotDate        time.Time `gorm:"column:snapshot_date;type:date;uniqueIndex:uk_snapshot_date_exophone" json:"snapshot_date"`
	AccountID           uint64    `gorm:"column:account_id;not null;index" json:"account_id"`
	ExophoneID          uint64    `gorm:"column:exophone_id;not null;uniqueIndex:uk_snapshot_date_exophone" json:"exophone_id"`
	TotalCalls          int       `gorm:"column:total_calls;default:0" json:"total_calls"`
	// Leg 1 (A-leg): agent/customer → VN (IVR routing leg)
	Leg1Total           int       `gorm:"column:leg1_total;default:0" json:"leg1_total"`
	Leg1Drops           int       `gorm:"column:leg1_drops;default:0" json:"leg1_drops"`           // Leg1 duration <= DropThresholdSec
	// Leg 2 (B-leg): VN → agent/customer (conversation leg)
	Leg2Total           int       `gorm:"column:leg2_total;default:0" json:"leg2_total"`
	Leg2Drops           int       `gorm:"column:leg2_drops;default:0" json:"leg2_drops"`           // Leg2 duration <= DropThresholdSec (real drops)
	// Status breakdown across all legs
	ConnectedCalls      int       `gorm:"column:connected_calls;default:0" json:"connected_calls"` // status=completed
	DroppedCalls        int       `gorm:"column:dropped_calls;default:0" json:"dropped_calls"`     // Leg2 completed AND duration <= DropThresholdSec
	FailedCalls         int       `gorm:"column:failed_calls;default:0" json:"failed_calls"`       // status=failed
	NoAnswerCalls       int       `gorm:"column:no_answer_calls;default:0" json:"no_answer_calls"` // status=no-answer
	BusyCalls           int       `gorm:"column:busy_calls;default:0" json:"busy_calls"`           // status=busy
	CanceledCalls       int       `gorm:"column:canceled_calls;default:0" json:"canceled_calls"`   // status=canceled
	OtherCalls          int       `gorm:"column:other_calls;default:0" json:"other_calls"`         // any unrecognised status
	AvgDurationSec      float64   `gorm:"column:avg_duration_sec;default:0" json:"avg_duration_sec"`
	AnswerRate          float64   `gorm:"column:answer_rate;default:0" json:"answer_rate"`         // connected/total * 100
	DropRate            float64   `gorm:"column:drop_rate;default:0" json:"drop_rate"`             // Leg2 drops / Leg2 total * 100
	Leg1DropRate        float64   `gorm:"column:leg1_drop_rate;default:0" json:"leg1_drop_rate"`   // Leg1 drops / Leg1 total * 100
	SuccessRate         float64   `gorm:"column:success_rate;default:0" json:"success_rate"`       // alias for answer_rate for compat
	HourlyDistribution  string    `gorm:"column:hourly_distribution;type:json" json:"hourly_distribution"` // 24-element JSON array
	PeakHour            int       `gorm:"column:peak_hour;default:0" json:"peak_hour"`             // 0-23, hour with most calls
	CreatedAt           time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (CallMetricsSnapshot) TableName() string { return "call_metrics_snapshot" }

type ExophoneHealthSnapshot struct {
	ID                  uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ExophoneID          uint64     `gorm:"column:exophone_id;not null;uniqueIndex" json:"exophone_id"`
	AccountID           uint64     `gorm:"column:account_id;not null;index" json:"account_id"`
	HeartbeatStatus     string     `gorm:"column:heartbeat_status;default:UNKNOWN" json:"heartbeat_status"`
	AvailabilityPercent float64    `gorm:"column:availability_percent;default:0" json:"availability_percent"`
	LastAPILatencyMs    *int64     `gorm:"column:last_api_latency_ms" json:"last_api_latency_ms"`
	ActiveStreams        int        `gorm:"column:active_streams;default:0" json:"active_streams"`
	LastCheckedAt       *time.Time `gorm:"column:last_checked_at" json:"last_checked_at"`
	CreatedAt           time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (ExophoneHealthSnapshot) TableName() string { return "exophone_health_snapshot" }

type StreamUtilizationSnapshot struct {
	ID                 uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	SnapshotHour       time.Time `gorm:"column:snapshot_hour;uniqueIndex:uk_sus_hour_exophone" json:"snapshot_hour"`
	AccountID          uint64    `gorm:"column:account_id;not null;index" json:"account_id"`
	ExophoneID         uint64    `gorm:"column:exophone_id;not null;uniqueIndex:uk_sus_hour_exophone" json:"exophone_id"`
	AvgActiveStreams    float64   `gorm:"column:avg_active_streams;default:0" json:"avg_active_streams"`
	MaxActiveStreams    int       `gorm:"column:max_active_streams;default:0" json:"max_active_streams"`
	UtilizationPercent float64   `gorm:"column:utilization_percent;default:0" json:"utilization_percent"`
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (StreamUtilizationSnapshot) TableName() string { return "stream_utilization_snapshot" }

type AccountDashboardSnapshot struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	SnapshotDate    time.Time `gorm:"column:snapshot_date;type:date;uniqueIndex:uk_ads_date_account" json:"snapshot_date"`
	AccountID       uint64    `gorm:"column:account_id;not null;uniqueIndex:uk_ads_date_account" json:"account_id"`
	TotalExophones  int       `gorm:"column:total_exophones;default:0" json:"total_exophones"`
	ActiveExophones int       `gorm:"column:active_exophones;default:0" json:"active_exophones"`
	TotalCalls      int       `gorm:"column:total_calls;default:0" json:"total_calls"`
	FailedCalls     int       `gorm:"column:failed_calls;default:0" json:"failed_calls"`
	SuccessRate     float64   `gorm:"column:success_rate;default:0" json:"success_rate"`
	AvgLatencyMs    *int64    `gorm:"column:avg_latency_ms" json:"avg_latency_ms"`
	AlertCount      int       `gorm:"column:alert_count;default:0" json:"alert_count"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (AccountDashboardSnapshot) TableName() string { return "account_dashboard_snapshot" }
