package scheduler

import (
	"time"

	"exotel-monitoring-platform/internal/models"
	"exotel-monitoring-platform/internal/repository"
)

// AccountDashboardSnapshotInput holds values for an account-level snapshot upsert.
type AccountDashboardSnapshotInput struct {
	SnapshotDate    time.Time
	AccountID       uint64
	TotalExophones  int
	ActiveExophones int
	TotalCalls      int
	FailedCalls     int
	SuccessRate     float64
	AlertCount      int
}

// UpsertAccountDashboardSnapshot is a thin adapter so scheduler.go doesn't import repository directly.
func UpsertAccountDashboardSnapshot(in *AccountDashboardSnapshotInput) error {
	return repository.UpsertAccountDashboardSnapshot(&models.AccountDashboardSnapshot{
		SnapshotDate:    in.SnapshotDate,
		AccountID:       in.AccountID,
		TotalExophones:  in.TotalExophones,
		ActiveExophones: in.ActiveExophones,
		TotalCalls:      in.TotalCalls,
		FailedCalls:     in.FailedCalls,
		SuccessRate:     in.SuccessRate,
		AlertCount:      in.AlertCount,
	})
}
