package workers

import (
	"exotel-monitoring-platform/internal/logger"
	"exotel-monitoring-platform/internal/repository"
	"go.uber.org/zap"
)

// RunDueJobs fetches all jobs that are due and dispatches them
// through the bounded worker pool.
func RunDueJobs() {
	jobs, err := repository.GetDueJobs()
	if err != nil {
		logger.Log.Error("failed to fetch due jobs", zap.Error(err))
		return
	}

	if len(jobs) == 0 {
		return
	}

	logger.Log.Info("dispatching due jobs", zap.Int("count", len(jobs)))

	for _, job := range jobs {
		j := job // capture loop variable
		DefaultPool.Submit(func() {
			ExecuteJob(j)
		})
	}

	// Do NOT block here — scheduler runs every minute and must return promptly.
	// Jobs continue running in the background via the pool.
}
