package alerts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"exotel-monitoring-platform/internal/config"
	"exotel-monitoring-platform/internal/logger"
	"go.uber.org/zap"
)

type slackPayload struct {
	Text        string            `json:"text"`
	Attachments []slackAttachment `json:"attachments,omitempty"`
}

type slackAttachment struct {
	Color  string       `json:"color"`
	Fields []slackField `json:"fields"`
	Footer string       `json:"footer"`
	Ts     int64        `json:"ts"`
}

type slackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// severityColor maps alert severity to Slack attachment colour.
func severityColor(severity string) string {
	switch severity {
	case "CRITICAL":
		return "danger" // red
	case "WARNING":
		return "warning" // yellow
	default:
		return "good" // green
	}
}

// SendSlackAlert dispatches an alert to the configured Slack webhook.
// Returns the HTTP status code and any error.
func SendSlackAlert(alertType, severity, message, txnID string) (int, error) {
	cfg := config.App.Slack
	if !cfg.Enabled || cfg.WebhookURL == "" {
		logger.Log.Debug("slack alerting disabled or webhook not configured")
		return 0, nil
	}

	payload := slackPayload{
		Text: fmt.Sprintf("*[%s]* %s", severity, alertType),
		Attachments: []slackAttachment{
			{
				Color: severityColor(severity),
				Fields: []slackField{
					{Title: "Alert Type", Value: alertType, Short: true},
					{Title: "Severity", Value: severity, Short: true},
					{Title: "Message", Value: message, Short: false},
					{Title: "Transaction ID", Value: txnID, Short: true},
				},
				Footer: "Exotel Monitoring Platform",
				Ts:     time.Now().Unix(),
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("slack marshal error: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(cfg.WebhookURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		logger.Log.Error("slack dispatch failed", zap.Error(err))
		return 0, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}
