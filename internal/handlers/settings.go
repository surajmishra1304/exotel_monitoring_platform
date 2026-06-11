package handlers

import (
	"net/http"

	"exotel-monitoring-platform/internal/feature"
	"github.com/gin-gonic/gin"
)

// GetSettings returns current runtime feature flags.
// GET /api/v1/settings
func GetSettings(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"skip_call_logs_write": feature.SkipCallLogsWrite(),
	})
}

// UpdateSettings toggles runtime feature flags.
// PATCH /api/v1/settings
func UpdateSettings(c *gin.Context) {
	var body struct {
		SkipCallLogsWrite *bool `json:"skip_call_logs_write"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	if body.SkipCallLogsWrite != nil {
		feature.SetSkipCallLogsWrite(*body.SkipCallLogsWrite)
	}
	c.JSON(http.StatusOK, gin.H{
		"skip_call_logs_write": feature.SkipCallLogsWrite(),
	})
}
