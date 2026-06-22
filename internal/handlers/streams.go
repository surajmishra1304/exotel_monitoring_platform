package handlers

import (
	"fmt"
	"net/http"
	"time"

	"exotel-monitoring-platform/internal/config"
	"exotel-monitoring-platform/internal/exotel"
	"exotel-monitoring-platform/internal/models"
	"exotel-monitoring-platform/internal/repository"
	"exotel-monitoring-platform/internal/snapshots"
	"exotel-monitoring-platform/internal/utils"
	"github.com/gin-gonic/gin"
)

// GetLiveActiveStreams fetches the current active stream count for an exophone
// directly from the Exotel API and persists the reading as a stream_metric row.
//
// GET /api/v1/exophones/:id/streams/live
func GetLiveActiveStreams(c *gin.Context) {
	exophoneID := parseUint(c.Param("id"))
	if exophoneID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid exophone id"})
		return
	}

	exophone, err := repository.GetExophoneByID(exophoneID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("exophone %d not found", exophoneID)})
		return
	}

	account, err := repository.GetAccountByID(exophone.AccountID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}

	secretKey := config.App.Crypto.SecretKey
	apiKey, err := utils.Decrypt(account.APIKey, secretKey)
	if err != nil {
		apiKey = account.APIKey
	}
	apiToken, err := utils.Decrypt(account.APIToken, secretKey)
	if err != nil {
		apiToken = account.APIToken
	}

	result, err := exotel.FetchActiveStreams(account.SID, account.Subdomain, apiKey, apiToken)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error":       "exotel streams API call failed",
			"detail":      err.Error(),
			"http_status": result.HTTPStatus,
			"latency_ms":  result.LatencyMs,
		})
		return
	}

	if !isSuccessHTTP(result.HTTPStatus) {
		c.JSON(http.StatusBadGateway, gin.H{
			"error":       fmt.Sprintf("exotel returned HTTP %d", result.HTTPStatus),
			"raw":         result.RawBody,
			"http_status": result.HTTPStatus,
			"latency_ms":  result.LatencyMs,
		})
		return
	}

	active := result.Response.ActiveStreams()
	maxAllowed := result.Response.MaxAllowedStreams()
	utilPct := 0.0
	if maxAllowed > 0 {
		utilPct = float64(active) / float64(maxAllowed) * 100
	}

	txnID := fmt.Sprintf("STREAMS-LIVE-%d-%d", exophoneID, time.Now().UnixMilli())

	_ = repository.SaveStreamMetric(&models.StreamMetric{
		TransactionID:      txnID,
		AccountID:          exophone.AccountID,
		ExophoneID:         exophoneID,
		ActiveStreams:       active,
		MaxAllowedStreams:   maxAllowed,
		UtilizationPercent: utilPct,
		CreatedBy:          "SYSTEM_LIVE",
	})

	_ = snapshots.UpdateStreamUtilizationSnapshot(exophone.AccountID, exophoneID, active, maxAllowed, utilPct)

	c.JSON(http.StatusOK, gin.H{
		"exophone_id":         exophoneID,
		"exophone_number":     exophone.ExophoneNumber,
		"active_streams":      active,
		"max_allowed_streams": maxAllowed,
		"utilization_pct":     utilPct,
		"latency_ms":          result.LatencyMs,
		"raw":                 result.Response,
	})
}

func isSuccessHTTP(code int) bool {
	return code >= 200 && code <= 299
}
