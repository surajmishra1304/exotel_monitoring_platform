package scheduler

import (
	"fmt"
	"time"

	"exotel-monitoring-platform/internal/logger"
	"exotel-monitoring-platform/internal/repository"
	"exotel-monitoring-platform/internal/workers"
	"exotel-monitoring-platform/internal/config"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

var cronRunner *cron.Cron

// Start initialises the scheduler and registers all recurring jobs.
func Start() {
	cronRunner = cron.New(cron.WithSeconds())

	pollSec := config.App.Scheduler.PollIntervalSeconds
	if pollSec <= 0 {
		pollSec = 60
	}

	// Primary: poll DB for due monitoring jobs every N seconds.
	cronRunner.AddFunc(fmt.Sprintf("@every %ds", pollSec), func() {
		workers.RunDueJobs()
	})

	// Hourly snapshot refresh across all accounts.
	cronRunner.AddFunc("0 0 * * * *", func() {
		runHourlySnapshotRefresh()
	})

	// Daily cleanup at 2 AM UTC.
	cronRunner.AddFunc("0 0 2 * * *", func() {
		runDailyCleanup()
	})

	cronRunner.Start()
	logger.Log.Info("scheduler started",
		zap.Int("poll_interval_seconds", pollSec),
	)
}

// Stop gracefully drains the scheduler (waits for in-flight jobs).
func Stop() {
	if cronRunner != nil {
		ctx := cronRunner.Stop()
		<-ctx.Done()
		logger.Log.Info("scheduler stopped")
	}
}

func runHourlySnapshotRefresh() {
	logger.Log.Info("running hourly snapshot refresh")
	// Account-level dashboard snapshots are refreshed here.
	accounts, err := repository.GetActiveAccounts()
	if err != nil {
		logger.Log.Error("snapshot refresh: failed to fetch accounts", zap.Error(err))
		return
	}
	for _, a := range accounts {
		today := time.Now().Truncate(24 * time.Hour)
		exophones, _ := repository.GetAllExophonesByAccount(a.ID)

		var totalCalls, failedCalls, activeEx int
		for _, ex := range exophones {
			if ex.Status == "active" {
				activeEx++
			}
			snap, err := repository.GetCallMetricsSnapshot(ex.ID, today)
			if err == nil {
				totalCalls += snap.TotalCalls
				failedCalls += snap.FailedCalls
			}
		}

		successRate := 0.0
		if totalCalls > 0 {
			successRate = float64(totalCalls-failedCalls) / float64(totalCalls) * 100
		}

		activeAlerts, _ := repository.GetActiveAlerts()
		alertCount := 0
		for _, al := range activeAlerts {
			if al.AccountID == a.ID {
				alertCount++
			}
		}

		_ = UpsertAccountDashboardSnapshot(&AccountDashboardSnapshotInput{
			SnapshotDate:    today,
			AccountID:       a.ID,
			TotalExophones:  len(exophones),
			ActiveExophones: activeEx,
			TotalCalls:      totalCalls,
			FailedCalls:     failedCalls,
			SuccessRate:     successRate,
			AlertCount:      alertCount,
		})
	}
}

func runDailyCleanup() {
	logger.Log.Info("running daily cleanup")
	if err := repository.CleanupOldTransactions(90); err != nil {
		logger.Log.Error("cleanup transactions failed", zap.Error(err))
	}
	if err := repository.CleanupOldResponses(30); err != nil {
		logger.Log.Error("cleanup responses failed", zap.Error(err))
	}
	logger.Log.Info("daily cleanup complete")
}
