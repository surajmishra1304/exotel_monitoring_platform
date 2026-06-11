package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"exotel-monitoring-platform/internal/alerts"
	"exotel-monitoring-platform/internal/cache"
	"exotel-monitoring-platform/internal/logger"
	"exotel-monitoring-platform/internal/models"
	"exotel-monitoring-platform/internal/repository"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// GetActiveAlerts returns all OPEN alerts.
// Serve order: Redis cache → DB, failing loud on DB error.
func GetActiveAlerts(c *gin.Context) {
	ctx := context.Background()
	reqID, _ := c.Get("request_id")

	var list any
	if hit, cacheErr := cache.Get(ctx, cache.KeyAlertsActive, &list); hit {
		c.JSON(http.StatusOK, gin.H{"source": "cache", "data": list})
		return
	} else if cacheErr != nil {
		logger.Log.Warn("get active alerts: cache read failed",
			zap.String("request_id", fmt.Sprintf("%v", reqID)),
			zap.Error(cacheErr),
		)
	}

	dbList, err := repository.GetActiveAlerts()
	if err != nil {
		logger.Log.Error("get active alerts: DB query failed",
			zap.String("request_id", fmt.Sprintf("%v", reqID)),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to fetch active alerts: %s", err.Error()),
		})
		return
	}

	if setErr := cache.Set(ctx, cache.KeyAlertsActive, dbList, cache.TTLAlerts); setErr != nil {
		logger.Log.Warn("get active alerts: failed to warm cache",
			zap.String("request_id", fmt.Sprintf("%v", reqID)),
			zap.Error(setErr),
		)
	}
	c.JSON(http.StatusOK, gin.H{"source": "db", "data": dbList})
}

// CreateAlert is the internal API to programmatically fire an alert.
func CreateAlert(c *gin.Context) {
	reqID, _ := c.Get("request_id")

	var req struct {
		TransactionID string `json:"transaction_id"`
		ExophoneID    uint64 `json:"exophone_id" binding:"required"`
		AccountID     uint64 `json:"account_id" binding:"required"`
		AlertType     string `json:"alert_type" binding:"required"`
		Severity      string `json:"severity" binding:"required"`
		Message       string `json:"message" binding:"required"`
		Channel       string `json:"channel" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Log.Warn("create alert: invalid request body",
			zap.String("request_id", fmt.Sprintf("%v", reqID)),
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid alert payload: %s", err.Error()),
		})
		return
	}

	alerts.TriggerAlert(models.Alert{
		TransactionID: req.TransactionID,
		ExophoneID:    req.ExophoneID,
		AccountID:     req.AccountID,
		AlertType:     req.AlertType,
		Severity:      req.Severity,
		Message:       req.Message,
		Channel:       req.Channel,
		TriggeredAt:   time.Now(),
	})

	c.JSON(http.StatusCreated, gin.H{"status": "QUEUED"})
}

// AcknowledgeAlert marks a specific alert as acknowledged.
func AcknowledgeAlert(c *gin.Context) {
	reqID, _ := c.Get("request_id")

	alertID := parseUint(c.Param("id"))
	if alertID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert id: must be a positive integer"})
		return
	}

	by := c.GetHeader("X-User")
	if by == "" {
		by = "API_USER"
	}

	if err := repository.AcknowledgeAlert(alertID, by); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("alert %d not found", alertID)})
			return
		}
		logger.Log.Error("acknowledge alert: DB update failed",
			zap.Uint64("alert_id", alertID),
			zap.String("request_id", fmt.Sprintf("%v", reqID)),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to acknowledge alert %d: %s", alertID, err.Error()),
		})
		return
	}

	// Invalidate active alerts cache so next read reflects the acknowledgement.
	if delErr := cache.Delete(context.Background(), cache.KeyAlertsActive); delErr != nil {
		logger.Log.Warn("acknowledge alert: failed to invalidate alerts cache",
			zap.Uint64("alert_id", alertID),
			zap.Error(delErr),
		)
	}

	c.JSON(http.StatusOK, gin.H{"status": "ACKNOWLEDGED"})
}
