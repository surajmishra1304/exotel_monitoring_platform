package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"exotel-monitoring-platform/internal/alerts"
	"exotel-monitoring-platform/internal/exotel"
	"exotel-monitoring-platform/internal/logger"
	"exotel-monitoring-platform/internal/models"
	"exotel-monitoring-platform/internal/repository"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ExotelWebhook handles inbound Passthru Applet / StatusCallback events from Exotel.
func ExotelWebhook(c *gin.Context) {
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	raw, _ := json.Marshal(payload)

	event := &models.CallFlowEvent{
		CallSID:    stringFromMap(payload, "CallSid"),
		FlowID:     stringFromMap(payload, "FlowId"),
		CallFrom:   stringFromMap(payload, "From"),
		CallTo:     stringFromMap(payload, "To"),
		CallStatus: stringFromMap(payload, "CallStatus"),
		RawPayload: string(raw),
	}

	if err := repository.SaveCallFlowEvent(event); err != nil {
		logger.Log.Error("failed to save webhook event", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save event"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "received"})
}

// ExotelHeartbeat receives the push-based heartbeat from Exotel's Notification Service.
// Configure this URL in Exotel Dashboard → Notifications Settings → Heartbeat URL.
// Exotel POSTs application/json and expects HTTP 200 back within 15s; retries twice on failure.
//
// Payload (Exotel docs: https://developer.exotel.com/docs/heartbeat/webhook-format):
//
//	{
//	  "timestamp":         "2018-08-22T15:19:23Z",
//	  "status_type":       "WARNING",          // OK | WARNING | CRITICAL | PAYLOAD_TOO_LARGE
//	  "incoming_affected": ["VN_SID_1", ...],  // VN SIDs with incoming disruptions
//	  "outgoing_affected": [],
//	  "data": {
//	    "VN_SID_1": {
//	      "connectivity": {
//	        "incoming_call": { "status": "major_outage", "last_check_time": "...", "alternate_exophone": null },
//	        "outgoing_call": { "status": "active",       "last_check_time": "...", "alternate_exophone": null }
//	      }
//	    }
//	  }
//	}
func ExotelHeartbeat(c *gin.Context) {
	var hb exotel.HeartbeatResponse
	if err := c.ShouldBindJSON(&hb); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid heartbeat payload"})
		return
	}

	log := logger.Log.With(
		zap.String("status_type", hb.StatusType),
		zap.String("timestamp", hb.Timestamp),
		zap.Strings("incoming_affected", hb.IncomingAffected),
		zap.Strings("outgoing_affected", hb.OutgoingAffected),
	)
	log.Info("exotel heartbeat received")

	// Map to our internal status constants.
	statusType := models.HeartbeatOK
	if len(hb.IncomingAffected) > 0 || len(hb.OutgoingAffected) > 0 {
		statusType = models.HeartbeatDegraded
	}
	if hb.StatusType == "CRITICAL" {
		statusType = models.HeartbeatOutage
	}

	_ = repository.SaveHeartbeatMetric(&models.HeartbeatMetric{
		TransactionID:    "WEBHOOK",
		AccountID:        0,
		StatusType:       statusType,
		IncomingAffected: len(hb.IncomingAffected),
		OutgoingAffected: len(hb.OutgoingAffected),
		RawStatus:        hb.StatusType,
		CreatedBy:        "EXOTEL_WEBHOOK",
	})

	if statusType != models.HeartbeatOK {
		// Per-VN alerts (with real account_id) are fired inside fireVNAlerts.
		// We log the account-level summary here rather than inserting a DB row
		// with account_id=0 (which would fail the FK constraint).
		logger.Log.Warn("exotel heartbeat degraded",
			zap.String("status_type", hb.StatusType),
			zap.Strings("incoming_affected", hb.IncomingAffected),
			zap.Strings("outgoing_affected", hb.OutgoingAffected),
		)
		fireVNAlerts(&hb, statusType, "WEBHOOK")
	}

	// Exotel requires 200 OK — any other status triggers a retry.
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// fireVNAlerts looks up each affected VN SID in the DB and:
//  1. Updates the exophone_health_snapshot with real connectivity status from hb.Data
//  2. Fires per-direction alerts (incoming / outgoing) with alternate exophone suggestions
func fireVNAlerts(hb *exotel.HeartbeatResponse, statusType string, txnID string) {
	// Merge incoming_affected and outgoing_affected into a single set.
	type vnFlags struct{ inc, out bool }
	affected := map[string]vnFlags{}
	for _, sid := range hb.IncomingAffected {
		f := affected[sid]
		f.inc = true
		affected[sid] = f
	}
	for _, sid := range hb.OutgoingAffected {
		f := affected[sid]
		f.out = true
		affected[sid] = f
	}

	for vnSID, flags := range affected {
		// Look up the VN detail from hb.Data for richer context.
		vnData, hasData := hb.Data[vnSID]

		exophones, _, _ := repository.GetExophonesByNumber(vnSID)
		for _, ep := range exophones {
			// Determine the worst-case heartbeat status for this VN.
			hbStatus := models.HeartbeatDegraded
			if statusType == models.HeartbeatOutage {
				hbStatus = models.HeartbeatOutage
			}

			// Update the health snapshot using the per-VN connectivity data.
			snap := &models.ExophoneHealthSnapshot{
				ExophoneID:          ep.ID,
				AccountID:           ep.AccountID,
				HeartbeatStatus:     hbStatus,
				AvailabilityPercent: 0,
			}
			if hasData {
				incStatus := vnData.Connectivity.IncomingCall.Status
				outStatus := vnData.Connectivity.OutgoingCall.Status
				if incStatus == "active" && outStatus == "active" {
					snap.HeartbeatStatus = models.HeartbeatOK
					snap.AvailabilityPercent = 100
				} else if incStatus == "active" || outStatus == "active" {
					snap.HeartbeatStatus = models.HeartbeatDegraded
					snap.AvailabilityPercent = 50
				}
			}
			_ = repository.UpsertExophoneHealthSnapshot(snap)

			// Build per-direction alert messages, including alternate VN if provided.
			if flags.inc {
				msg := fmt.Sprintf("VN %s — incoming calls degraded (Exotel heartbeat: %s)", vnSID, hb.StatusType)
				if hasData {
					if alt := vnData.Connectivity.IncomingCall.AlternateExophone; alt != nil && *alt != "" {
						msg += fmt.Sprintf("; suggested fallback: %s", *alt)
					}
				}
				alerts.TriggerAlert(models.Alert{
					TransactionID: txnID,
					ExophoneID:    ep.ID,
					AccountID:     ep.AccountID,
					AlertType:     models.AlertTypeExophoneDown,
					Severity:      models.SeverityCritical,
					Message:       msg,
					Channel:       models.ChannelSlack,
				})
			}
			if flags.out {
				msg := fmt.Sprintf("VN %s — outgoing calls degraded (Exotel heartbeat: %s)", vnSID, hb.StatusType)
				if hasData {
					if alt := vnData.Connectivity.OutgoingCall.AlternateExophone; alt != nil && *alt != "" {
						msg += fmt.Sprintf("; suggested fallback: %s", *alt)
					}
				}
				alerts.TriggerAlert(models.Alert{
					TransactionID: txnID,
					ExophoneID:    ep.ID,
					AccountID:     ep.AccountID,
					AlertType:     models.AlertTypeExophoneDown,
					Severity:      models.SeverityCritical,
					Message:       msg,
					Channel:       models.ChannelSlack,
				})
			}
		}
	}
}

func stringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
