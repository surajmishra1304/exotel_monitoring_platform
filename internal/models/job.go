package models

import "time"

// RetryPolicy is embedded as JSON in monitoring_jobs.retry_policy.
type RetryPolicy struct {
	MaxRetries   int `json:"max_retries"`
	BaseDelayMs  int `json:"base_delay_ms"`
	MaxDelayMs   int `json:"max_delay_ms"`
}

type MonitoringJob struct {
	ID              uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ExophoneID      uint64     `gorm:"column:exophone_id;not null;index" json:"exophone_id"`
	AccountID       uint64     `gorm:"column:account_id;not null" json:"account_id"`
	JobName         string     `gorm:"column:job_name;not null" json:"job_name"`
	JobType         string     `gorm:"column:job_type;not null;default:HEARTBEAT" json:"job_type"`
	Priority        string     `gorm:"column:priority;default:P1" json:"priority"`
	CronExpression  string     `gorm:"column:cron_expression" json:"cron_expression,omitempty"`
	FrequencyMinute int        `gorm:"column:frequency_minutes;default:30" json:"frequency_minutes"`
	TimeoutSeconds  int        `gorm:"column:timeout_seconds;default:30" json:"timeout_seconds"`
	RetryPolicy     string     `gorm:"column:retry_policy;type:json" json:"retry_policy"`
	NextRunAt       time.Time  `gorm:"column:next_run_at" json:"next_run_at"`
	LastRunAt       *time.Time `gorm:"column:last_run_at" json:"last_run_at"`
	LastStatus      string     `gorm:"column:last_status" json:"last_status,omitempty"`
	IsActive        int        `gorm:"column:is_active;default:1" json:"is_active"`
	IsDeleted       int        `gorm:"column:is_deleted;default:0" json:"-"`
	CreatedAt       time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	CreatedBy       string     `gorm:"column:created_by;default:SYSTEM" json:"created_by"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	UpdatedBy       string     `gorm:"column:updated_by;default:SYSTEM" json:"updated_by"`
}

func (MonitoringJob) TableName() string { return "monitoring_jobs" }
