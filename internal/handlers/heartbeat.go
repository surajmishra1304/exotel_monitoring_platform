package handlers

import (
	"net/http"
	"strconv"
	"time"

	"exotel-monitoring-platform/internal/config"
	"exotel-monitoring-platform/internal/exotel"
	"exotel-monitoring-platform/internal/models"
	"exotel-monitoring-platform/internal/repository"
	"exotel-monitoring-platform/internal/utils"
	"github.com/gin-gonic/gin"
)

// CheckExophoneHeartbeat manually triggers a heartbeat check for a single exophone.
// It calls the Exotel v2 API (with v1 fallback) and updates the health snapshot immediately.
//
// POST /api/v1/exophones/:id/heartbeat/check
func CheckExophoneHeartbeat(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid exophone id"})
		return
	}

	ep, err := repository.GetExophoneByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "exophone not found"})
		return
	}

	account, err := repository.GetAccountByID(ep.AccountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "account not found"})
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

	start := time.Now()
	result, fetchErr := exotel.FetchHeartbeat(account.SID, account.Subdomain, apiKey, apiToken, ep.ExophoneNumber)
	elapsed := time.Since(start).Milliseconds()

	if fetchErr != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"exophone_id":     id,
			"exophone_number": ep.ExophoneNumber,
			"error":           fetchErr.Error(),
			"latency_ms":      elapsed,
		})
		return
	}

	// Update the health snapshot with the fresh result.
	hbStatus := models.HeartbeatOK
	availPct := 100.0
	if result.Response != nil {
		switch result.Response.StatusType {
		case "WARNING":
			hbStatus = models.HeartbeatDegraded
			availPct = 50.0
		case "CRITICAL":
			hbStatus = models.HeartbeatOutage
			availPct = 0.0
		}

		// Use per-VN connectivity data from the v2 response if available.
		if vnData, ok := result.Response.Data[ep.ExophoneNumber]; ok {
			incStatus := vnData.Connectivity.IncomingCall.Status
			outStatus := vnData.Connectivity.OutgoingCall.Status
			switch {
			case incStatus == "major_outage" && outStatus == "major_outage":
				hbStatus = models.HeartbeatOutage
				availPct = 0.0
			case incStatus == "major_outage" || outStatus == "major_outage":
				hbStatus = models.HeartbeatDegraded
				availPct = 50.0
			default:
				hbStatus = models.HeartbeatOK
				availPct = 100.0
			}
		}
	}

	latMs := result.LatencyMs
	_ = repository.UpsertExophoneHealthSnapshot(&models.ExophoneHealthSnapshot{
		ExophoneID:          ep.ID,
		AccountID:           ep.AccountID,
		HeartbeatStatus:     hbStatus,
		AvailabilityPercent: availPct,
		LastAPILatencyMs:    &latMs,
	})

	// Build a clean response.
	resp := gin.H{
		"exophone_id":      id,
		"exophone_number":  ep.ExophoneNumber,
		"http_status":      result.HTTPStatus,
		"latency_ms":       result.LatencyMs,
		"heartbeat_status": hbStatus,
		"availability_pct": availPct,
	}

	if result.Response != nil {
		resp["status_type"] = result.Response.StatusType
		resp["incoming_affected"] = result.Response.IncomingAffected
		resp["outgoing_affected"] = result.Response.OutgoingAffected

		if vnData, ok := result.Response.Data[ep.ExophoneNumber]; ok {
			resp["connectivity"] = gin.H{
				"incoming_call": gin.H{
					"status":          vnData.Connectivity.IncomingCall.Status,
					"last_check_time": vnData.Connectivity.IncomingCall.LastCheckTime,
				},
				"outgoing_call": gin.H{
					"status":          vnData.Connectivity.OutgoingCall.Status,
					"last_check_time": vnData.Connectivity.OutgoingCall.LastCheckTime,
				},
			}
		}
	}

	c.JSON(http.StatusOK, resp)
}
