package repository

import (
	"time"

	"exotel-monitoring-platform/internal/database"
	"exotel-monitoring-platform/internal/models"
)

// GetDueJobs fetches all active jobs whose next_run_at is <= now,
// only for exophones where is_cron_applicable = 1,
// ordered by priority (P0 first) then by next_run_at.
func GetDueJobs() ([]models.MonitoringJob, error) {
	var jobs []models.MonitoringJob
	result := database.DB.
		Joins("JOIN exophones ON exophones.id = monitoring_jobs.exophone_id AND exophones.is_cron_applicable = 1 AND exophones.is_deleted = 0").
		Where("monitoring_jobs.next_run_at <= ? AND monitoring_jobs.is_active = 1 AND monitoring_jobs.is_deleted = 0", time.Now()).
		Order("FIELD(monitoring_jobs.priority,'P0','P1','P2','P3'), monitoring_jobs.next_run_at ASC").
		Find(&jobs)
	return jobs, result.Error
}

// AdvanceNextRun updates next_run_at for a job based on its frequency_minutes.
func AdvanceNextRun(jobID uint64, freqMinutes int, status string) error {
	next := time.Now().Add(time.Duration(freqMinutes) * time.Minute)
	return database.DB.Model(&models.MonitoringJob{}).
		Where("id = ?", jobID).
		Updates(map[string]interface{}{
			"next_run_at": next,
			"last_run_at": time.Now(),
			"last_status": status,
			"updated_by":  "SYSTEM_SCHEDULER",
		}).Error
}

// GetJobByID retrieves a single monitoring job.
func GetJobByID(id uint64) (*models.MonitoringJob, error) {
	var j models.MonitoringJob
	result := database.DB.Where("id = ?", id).First(&j)
	return &j, result.Error
}

// GetCallsJobForExophone returns the CALLS monitoring job for an exophone.
func GetCallsJobForExophone(exophoneID uint64) (*models.MonitoringJob, error) {
	var j models.MonitoringJob
	result := database.DB.
		Where("exophone_id = ? AND job_type = 'CALLS'", exophoneID).
		First(&j)
	return &j, result.Error
}
