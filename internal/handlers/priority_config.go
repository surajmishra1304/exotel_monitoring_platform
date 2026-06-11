package handlers

import (
	"fmt"
	"net/http"

	"exotel-monitoring-platform/internal/logger"
	"exotel-monitoring-platform/internal/repository"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GetPriorityConfig returns all four priority config rows.
// GET /api/v1/priority-config
func GetPriorityConfig(c *gin.Context) {
	configs, err := repository.GetAllPriorityConfigs()
	if err != nil {
		logger.Log.Error("get priority config: DB query failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch priority config"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": configs})
}

// UpdatePriorityConfigHandler updates frequency_minutes for one priority level.
// PATCH /api/v1/priority-config/:priority
func UpdatePriorityConfigHandler(c *gin.Context) {
	reqID, _ := c.Get("request_id")
	priority := c.Param("priority")

	validPriorities := map[string]bool{"P0": true, "P1": true, "P2": true, "P3": true}
	if !validPriorities[priority] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "priority must be P0, P1, P2, or P3"})
		return
	}

	var body struct {
		FrequencyMinutes int    `json:"frequency_minutes" binding:"required"`
		Description      string `json:"description"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.FrequencyMinutes < 1 || body.FrequencyMinutes > 1440 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "frequency_minutes must be between 1 and 1440"})
		return
	}

	by := c.GetHeader("X-User")
	if by == "" {
		by = "API"
	}

	// Read current value before mutating so we can write a meaningful audit log.
	existing, _ := repository.GetPriorityConfigByPriority(priority)
	oldFreq := 0
	var configID uint64
	if existing != nil {
		oldFreq = existing.FrequencyMinutes
		configID = existing.ID
	}

	if err := repository.UpdatePriorityConfig(priority, body.FrequencyMinutes, by); err != nil {
		logger.Log.Error("update priority config: DB update failed",
			zap.String("priority", priority),
			zap.String("request_id", fmt.Sprintf("%v", reqID)),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update priority config"})
		return
	}

	// Audit log — best-effort, never fails the request.
	_ = repository.SaveConfigChangeLog("priority_config", priority, configID, oldFreq, body.FrequencyMinutes, by, body.Description)

	c.JSON(http.StatusOK, gin.H{
		"priority":          priority,
		"frequency_minutes": body.FrequencyMinutes,
		"message":           "priority config updated",
	})
}
