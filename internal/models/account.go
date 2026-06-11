package models

import "time"

type Account struct {
	ID          uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	AccountName string     `gorm:"column:account_name;not null" json:"account_name"`
	SID         string     `gorm:"column:sid;uniqueIndex;not null" json:"sid"`
	APIKey      string     `gorm:"column:api_key;not null" json:"-"` // encrypted at rest
	APIToken    string     `gorm:"column:api_token;not null" json:"-"` // encrypted at rest
	Subdomain   string     `gorm:"column:subdomain;not null" json:"subdomain"`
	Cluster     string     `gorm:"column:cluster;not null;default:in1" json:"cluster"`
	ConfigJSON  string     `gorm:"column:config_json;type:json" json:"config_json,omitempty"`
	IsActive    int        `gorm:"column:is_active;default:1" json:"is_active"`
	IsDeleted   int        `gorm:"column:is_deleted;default:0" json:"-"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	CreatedBy   string     `gorm:"column:created_by;default:SYSTEM" json:"created_by"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	UpdatedBy   string     `gorm:"column:updated_by;default:SYSTEM" json:"updated_by"`
}

func (Account) TableName() string { return "accounts" }
