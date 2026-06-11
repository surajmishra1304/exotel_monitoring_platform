package models

import "time"

// Alert type constants.
const (
	AlertTypeAPIFailure        = "API_FAILURE"
	AlertTypeHeartbeatFailure  = "HEARTBEAT_FAILURE"
	AlertTypeHighLatency       = "HIGH_LATENCY"
	AlertTypeRetryExhausted    = "RETRY_EXHAUSTED"
	AlertTypeVendorThrottling  = "VENDOR_THROTTLING"
	AlertTypeExophoneDown      = "EXOPHONE_DOWN"
	AlertTypeCacheFailure      = "CACHE_FAILURE"
	AlertTypeCallDrop          = "CALL_DROP"
)

// Alert severity constants.
const (
	SeverityCritical = "CRITICAL"
	SeverityWarning  = "WARNING"
	SeverityInfo     = "INFO"
)

// Alert status constants.
const (
	AlertStatusOpen         = "OPEN"
	AlertStatusAcknowledged = "ACKNOWLEDGED"
	AlertStatusResolved     = "RESOLVED"
)

// Alert channel constants.
const (
	ChannelSlack   = "SLACK"
	ChannelEmail   = "EMAIL"
	ChannelWebhook = "WEBHOOK"
)

type Alert struct {
	ID             uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TransactionID  string     `gorm:"column:transaction_id;index" json:"transaction_id,omitempty"`
	ExophoneID     uint64     `gorm:"column:exophone_id;index" json:"exophone_id"`
	AccountID      uint64     `gorm:"column:account_id" json:"account_id"`
	AlertType      string     `gorm:"column:alert_type;not null" json:"alert_type"`
	Severity       string     `gorm:"column:severity;default:WARNING" json:"severity"`
	AlertStatus    string     `gorm:"column:alert_status;default:OPEN;index" json:"alert_status"`
	Message        string     `gorm:"column:message;type:text;not null" json:"message"`
	Channel        string     `gorm:"column:channel;not null" json:"channel"`
	TriggeredAt    time.Time  `gorm:"column:triggered_at;index" json:"triggered_at"`
	AcknowledgedAt *time.Time `gorm:"column:acknowledged_at" json:"acknowledged_at"`
	ResolvedAt     *time.Time `gorm:"column:resolved_at" json:"resolved_at"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	CreatedBy      string     `gorm:"column:created_by;default:SYSTEM_ALERT_ENGINE" json:"created_by"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	UpdatedBy      string     `gorm:"column:updated_by;default:SYSTEM_ALERT_ENGINE" json:"updated_by"`
}

func (Alert) TableName() string { return "alerts" }

type AlertDispatchLog struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	AlertID      uint64    `gorm:"column:alert_id;not null;index" json:"alert_id"`
	Channel      string    `gorm:"column:channel;not null" json:"channel"`
	Status       string    `gorm:"column:status;not null" json:"status"`
	ResponseCode int       `gorm:"column:response_code" json:"response_code"`
	ErrorMessage string    `gorm:"column:error_message;type:text" json:"error_message,omitempty"`
	DispatchedAt time.Time `gorm:"column:dispatched_at;autoCreateTime" json:"dispatched_at"`
}

func (AlertDispatchLog) TableName() string { return "alert_dispatch_logs" }
