package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"exotel-monitoring-platform/internal/logger"
	"exotel-monitoring-platform/internal/repository"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GetTransactions returns job transactions with optional filters.
// Query params: exophone_id, status, job_type, from (RFC3339), to (RFC3339), page, limit.
func GetTransactions(c *gin.Context) {
	reqID, _ := c.Get("request_id")

	exophoneID, _ := strconv.ParseUint(c.Query("exophone_id"), 10, 64)
	status := c.Query("status")
	jobType := c.Query("job_type")

	var from, to time.Time
	if s := c.Query("from"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("invalid 'from' timestamp (expected RFC3339): %s", err.Error()),
			})
			return
		}
		from = t
	}
	if s := c.Query("to"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("invalid 'to' timestamp (expected RFC3339): %s", err.Error()),
			})
			return
		}
		to = t
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	list, total, err := repository.GetTransactions(exophoneID, status, jobType, from, to, page, limit)
	if err != nil {
		logger.Log.Error("get transactions: DB query failed",
			zap.Uint64("exophone_id", exophoneID),
			zap.String("status", status),
			zap.String("job_type", jobType),
			zap.String("request_id", fmt.Sprintf("%v", reqID)),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to fetch transactions: %s", err.Error()),
		})
		return
	}

	totalPages := 1
	if limit > 0 {
		totalPages = (int(total) + limit - 1) / limit
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        list,
		"count":       len(list),
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
	})
}
