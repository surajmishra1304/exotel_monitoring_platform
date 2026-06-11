package models

import "time"

// Heartbeat status constants.
const (
	HeartbeatOK      = "OK"
	HeartbeatDegraded = "DEGRADED"
	HeartbeatOutage  = "OUTAGE"
)

type HeartbeatMetric struct {
	ID               uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	TransactionID    string    `gorm:"column:transaction_id;not null;index" json:"transaction_id"`
	AccountID        uint64    `gorm:"column:account_id;not null;index" json:"account_id"`
	StatusType       string    `gorm:"column:status_type;not null" json:"status_type"`
	IncomingAffected int       `gorm:"column:incoming_affected;default:0" json:"incoming_affected"`
	OutgoingAffected int       `gorm:"column:outgoing_affected;default:0" json:"outgoing_affected"`
	ResponseTimeMs   *int64    `gorm:"column:response_time_ms" json:"response_time_ms"`
	RawStatus        string    `gorm:"column:raw_status" json:"raw_status,omitempty"`
	CreatedAt        time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	CreatedBy        string    `gorm:"column:created_by;default:SYSTEM_METRICS" json:"created_by"`
}

func (HeartbeatMetric) TableName() string { return "heartbeat_metrics" }

type StreamMetric struct {
	ID                 uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	TransactionID      string    `gorm:"column:transaction_id;not null;index" json:"transaction_id"`
	AccountID          uint64    `gorm:"column:account_id;not null;index" json:"account_id"`
	ExophoneID         uint64    `gorm:"column:exophone_id;not null;index" json:"exophone_id"`
	ActiveStreams       int       `gorm:"column:active_streams;default:0" json:"active_streams"`
	MaxAllowedStreams   int       `gorm:"column:max_allowed_streams;default:0" json:"max_allowed_streams"`
	UtilizationPercent float64   `gorm:"column:utilization_percent;default:0" json:"utilization_percent"`
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	CreatedBy          string    `gorm:"column:created_by;default:SYSTEM_METRICS" json:"created_by"`
}

func (StreamMetric) TableName() string { return "stream_metrics" }

type Metric struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	TransactionID   string    `gorm:"column:transaction_id;not null;index" json:"transaction_id"`
	ExophoneID      uint64    `gorm:"column:exophone_id;not null;index" json:"exophone_id"`
	AccountID       uint64    `gorm:"column:account_id;not null" json:"account_id"`
	MetricName      string    `gorm:"column:metric_name;not null;index" json:"metric_name"`
	MetricValue     float64   `gorm:"column:metric_value;not null" json:"metric_value"`
	MetricUnit      string    `gorm:"column:metric_unit" json:"metric_unit,omitempty"`
	MetricTimestamp time.Time `gorm:"column:metric_timestamp;index" json:"metric_timestamp"`
	AggregationType string    `gorm:"column:aggregation_type;default:INSTANT" json:"aggregation_type"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	CreatedBy       string    `gorm:"column:created_by;default:SYSTEM_METRICS" json:"created_by"`
}

func (Metric) TableName() string { return "metrics" }
