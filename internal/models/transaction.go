package models

import (
	"encoding/json"
	"time"
)

// Job execution status constants.
const (
	TxnStatusRunning  = "RUNNING"
	TxnStatusSuccess  = "SUCCESS"
	TxnStatusFailed   = "FAILED"
	TxnStatusRetrying = "RETRYING"
	TxnStatusTimeout  = "TIMEOUT"
)

type JobTransaction struct {
	ID            uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TransactionID string     `gorm:"column:transaction_id;uniqueIndex;not null" json:"transaction_id"`
	JobID         uint64     `gorm:"column:job_id;not null;index" json:"job_id"`
	AccountID     uint64     `gorm:"column:account_id;not null" json:"account_id"`
	ExophoneID    uint64     `gorm:"column:exophone_id;not null;index" json:"exophone_id"`
	JobType       string     `gorm:"column:job_type;not null" json:"job_type"`
	Status        string     `gorm:"column:status;default:RUNNING" json:"status"`
	APILatencyMs  *int64     `gorm:"column:api_latency_ms" json:"api_latency_ms"`
	CacheHit      int        `gorm:"column:cache_hit;default:0" json:"cache_hit"`
	RetryCount    int        `gorm:"column:retry_count;default:0" json:"retry_count"`
	ErrorMessage  string     `gorm:"column:error_message;type:text" json:"error_message,omitempty"`
	StartedAt     time.Time  `gorm:"column:started_at;autoCreateTime" json:"started_at"`
	CompletedAt   *time.Time `gorm:"column:completed_at" json:"completed_at"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	CreatedBy     string     `gorm:"column:created_by;default:SYSTEM_SCHEDULER" json:"created_by"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	UpdatedBy     string     `gorm:"column:updated_by;default:SYSTEM_SCHEDULER" json:"updated_by"`
}

func (JobTransaction) TableName() string { return "job_transactions" }

type JobResponse struct {
	ID             uint64           `gorm:"primaryKey;autoIncrement" json:"id"`
	TransactionID  string           `gorm:"column:transaction_id;not null;index" json:"transaction_id"`
	ResponseType   string           `gorm:"column:response_type;not null" json:"response_type"`
	HTTPStatus     int              `gorm:"column:http_status" json:"http_status"`
	RawResponse    string           `gorm:"column:raw_response;type:mediumtext" json:"-"`
	ParsedResponse *json.RawMessage `gorm:"column:parsed_response;type:json" json:"parsed_response,omitempty"`
	CreatedAt      time.Time        `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (JobResponse) TableName() string { return "job_responses" }

// LegNumber values for CallLog.
const (
	LegUnknown = 0
	Leg1       = 1 // A-leg: agent/customer → VN (the inbound routing leg)
	Leg2       = 2 // B-leg: VN → agent/customer (the conversation leg)
)

type CallLog struct {
	ID                   uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TransactionID        string     `gorm:"column:transaction_id;not null;index" json:"transaction_id"`
	AccountID            uint64     `gorm:"column:account_id;not null;index" json:"account_id"`
	ExophoneID           uint64     `gorm:"column:exophone_id;not null;index" json:"exophone_id"`
	CallSID              string     `gorm:"column:call_sid" json:"call_sid"`
	ParentCallSID        string     `gorm:"column:parent_call_sid;default:''" json:"parent_call_sid"` // non-empty = Leg 2
	LegNumber            int        `gorm:"column:leg_number;default:0" json:"leg_number"`            // 1=A-leg, 2=B-leg
	CallFrom             string     `gorm:"column:call_from" json:"call_from"`
	CallTo               string     `gorm:"column:call_to" json:"call_to"`
	Status               string     `gorm:"column:status" json:"status"`
	Direction            string     `gorm:"column:direction" json:"direction"`
	DurationSec          int        `gorm:"column:duration_sec;default:0" json:"duration_sec"`
	ConversationDuration int        `gorm:"column:conversation_duration;default:0" json:"conversation_duration"` // actual talk time; 0 = agent never answered
	StartTime            *time.Time `gorm:"column:start_time" json:"start_time"`
	EndTime              *time.Time `gorm:"column:end_time" json:"end_time"`
	RecordingURL         string     `gorm:"column:recording_url" json:"recording_url,omitempty"`
	// Leg1Status/Leg2Status from Exotel Details (requires details=true on Calls API).
	// Leg2Status="" means no second leg was attempted (customer dropped during IVR) → Leg1 drop.
	// Leg2Status="canceled" means agent routing was attempted but failed → Leg2 drop.
	Leg1Status           string     `gorm:"column:leg1_status" json:"leg1_status,omitempty"`
	Leg2Status           string     `gorm:"column:leg2_status" json:"leg2_status,omitempty"`
	CreatedAt            time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (CallLog) TableName() string { return "call_logs" }

type CallFlowEvent struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	CallSID    string    `gorm:"column:call_sid;index" json:"call_sid"`
	FlowID     string    `gorm:"column:flow_id" json:"flow_id"`
	ExophoneID uint64    `gorm:"column:exophone_id;index" json:"exophone_id"`
	CallFrom   string    `gorm:"column:call_from" json:"call_from"`
	CallTo     string    `gorm:"column:call_to" json:"call_to"`
	CallStatus string    `gorm:"column:call_status" json:"call_status"`
	RawPayload string    `gorm:"column:raw_payload;type:json" json:"raw_payload,omitempty"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (CallFlowEvent) TableName() string { return "call_flow_events" }
