package handlers

import (
	"fmt"
	"net/http"

	"exotel-monitoring-platform/internal/database"
	"github.com/gin-gonic/gin"
)

// Health returns the operational status of all infrastructure dependencies.
func Health(c *gin.Context) {
	mysqlStatus := "CONNECTED"
	mysqlError := ""
	redisStatus := "CONNECTED"
	redisError := ""

	// Ping MySQL — capture the specific failure reason.
	sqlDB, err := database.DB.DB()
	if err != nil {
		mysqlStatus = "DISCONNECTED"
		mysqlError = fmt.Sprintf("failed to get sql.DB handle: %s", err.Error())
	} else if pingErr := sqlDB.Ping(); pingErr != nil {
		mysqlStatus = "DISCONNECTED"
		mysqlError = fmt.Sprintf("ping failed: %s", pingErr.Error())
	}

	// Ping Redis — capture the specific failure reason.
	if pingErr := database.Redis.Ping(database.Ctx).Err(); pingErr != nil {
		redisStatus = "DISCONNECTED"
		redisError = fmt.Sprintf("ping failed: %s", pingErr.Error())
	}

	overall := "UP"
	httpCode := http.StatusOK
	if mysqlStatus != "CONNECTED" || redisStatus != "CONNECTED" {
		overall = "DEGRADED"
		httpCode = http.StatusServiceUnavailable
	}

	resp := gin.H{
		"status": overall,
		"mysql":  mysqlStatus,
		"redis":  redisStatus,
	}
	if mysqlError != "" {
		resp["mysql_error"] = mysqlError
	}
	if redisError != "" {
		resp["redis_error"] = redisError
	}

	c.JSON(httpCode, resp)
}
