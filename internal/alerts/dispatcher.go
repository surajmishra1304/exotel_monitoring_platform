package alerts

import (
	"time"

	"exotel-monitoring-platform/internal/logger"
	"exotel-monitoring-platform/internal/models"
	"exotel-monitoring-platform/internal/repository"
	"go.uber.org/zap"
)

// TriggerAlert persists an alert and dispatches it to the configured channel.
// It is designed to be called fire-and-forget (safe to call from goroutines).
func TriggerAlert(a models.Alert) {
	a.TriggeredAt = time.Now()
	a.AlertStatus = models.AlertStatusOpen
	a.CreatedBy = "SYSTEM_ALERT_ENGINE"
	a.UpdatedBy = "SYSTEM_ALERT_ENGINE"

	// Persist to DB
	if err := repository.CreateAlert(&a); err != nil {
		logger.Log.Error("failed to persist alert",
			zap.String("alert_type", a.AlertType),
			zap.Error(err),
		)
		return
	}

	log := logger.Log.With(
		zap.Uint64("alert_id", a.ID),
		zap.String("alert_type", a.AlertType),
		zap.String("severity", a.Severity),
	)

	// Dispatch to channel
	var (
		httpStatus int
		dispatchErr error
	)

	switch a.Channel {
	case models.ChannelSlack:
		httpStatus, dispatchErr = SendSlackAlert(a.AlertType, a.Severity, a.Message, a.TransactionID)
	default:
		log.Warn("unsupported alert channel", zap.String("channel", a.Channel))
		return
	}

	dispatchStatus := "SENT"
	errMsg := ""
	if dispatchErr != nil {
		dispatchStatus = "FAILED"
		errMsg = dispatchErr.Error()
		log.Error("alert dispatch failed", zap.Error(dispatchErr))
	} else {
		log.Info("alert dispatched", zap.Int("http_status", httpStatus))
	}

	// Log dispatch result
	_ = repository.CreateDispatchLog(&models.AlertDispatchLog{
		AlertID:      a.ID,
		Channel:      a.Channel,
		Status:       dispatchStatus,
		ResponseCode: httpStatus,
		ErrorMessage: errMsg,
	})
}
