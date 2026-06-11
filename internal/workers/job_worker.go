package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"exotel-monitoring-platform/internal/alerts"
	"exotel-monitoring-platform/internal/cache"
	"exotel-monitoring-platform/internal/config"
	"exotel-monitoring-platform/internal/exotel"
	"exotel-monitoring-platform/internal/feature"
	"exotel-monitoring-platform/internal/logger"
	"exotel-monitoring-platform/internal/metrics"
	"exotel-monitoring-platform/internal/models"
	"exotel-monitoring-platform/internal/repository"
	"exotel-monitoring-platform/internal/snapshots"
	"exotel-monitoring-platform/internal/utils"
	"go.uber.org/zap"
)

// ExecuteJob runs the full monitoring lifecycle for a single job:
//  1. Create transaction record
//  2. Fetch account credentials (with decryption)
//  3. Call the appropriate Exotel API (with retry)
//  4. Store raw response
//  5. Generate metrics
//  6. Evaluate alert rules
//  7. Refresh snapshots & advance next_run_at
func ExecuteJob(job models.MonitoringJob) {
	txnID := fmt.Sprintf("TXN-%s-%d", time.Now().Format("20060102-150405"), job.ID)
	log := logger.Log.With(
		zap.String("transaction_id", txnID),
		zap.Uint64("job_id", job.ID),
		zap.String("job_type", job.JobType),
		zap.Uint64("exophone_id", job.ExophoneID),
	)
	log.Info("job started")

	// ── 1. Create transaction record ──────────────────────────────────
	txn := &models.JobTransaction{
		TransactionID: txnID,
		JobID:         job.ID,
		AccountID:     job.AccountID,
		ExophoneID:    job.ExophoneID,
		JobType:       job.JobType,
		Status:        models.TxnStatusRunning,
		StartedAt:     time.Now(),
		CreatedBy:     "SYSTEM_SCHEDULER",
		UpdatedBy:     "SYSTEM_SCHEDULER",
	}
	if err := repository.CreateTransaction(txn); err != nil {
		log.Error("failed to create transaction", zap.Error(err))
		return
	}

	// ── 2. Fetch and decrypt account credentials ───────────────────────
	account, err := repository.GetAccountByID(job.AccountID)
	if err != nil {
		log.Error("account not found", zap.Error(err))
		_ = repository.UpdateTransactionStatus(txnID, models.TxnStatusFailed, nil, 0, err.Error())
		_ = repository.AdvanceNextRun(job.ID, job.FrequencyMinute, models.TxnStatusFailed)
		return
	}

	secretKey := config.App.Crypto.SecretKey
	apiKey, err := utils.Decrypt(account.APIKey, secretKey)
	if err != nil {
		log.Warn("api_key decryption failed, using raw value", zap.Error(err))
		apiKey = account.APIKey // fallback for dev environments
	}
	apiToken, err := utils.Decrypt(account.APIToken, secretKey)
	if err != nil {
		log.Warn("api_token decryption failed, using raw value", zap.Error(err))
		apiToken = account.APIToken
	}

	// ── 3. Parse retry policy from DB ─────────────────────────────────
	retryCfg := utils.DefaultRetryConfig
	if job.RetryPolicy != "" {
		var rp models.RetryPolicy
		if err := json.Unmarshal([]byte(job.RetryPolicy), &rp); err == nil {
			retryCfg = utils.RetryConfig{
				MaxRetries:  rp.MaxRetries,
				BaseDelayMs: rp.BaseDelayMs,
				MaxDelayMs:  rp.MaxDelayMs,
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(job.TimeoutSeconds)*time.Second)
	defer cancel()

	// ── 4. Execute the API call with retry ────────────────────────────
	var (
		finalStatus  = models.TxnStatusSuccess
		finalErrMsg  string
		latencyMs    int64
		retryCount   int
		httpStatus   int
		rawBody      string
	)

	switch job.JobType {
	case "HEARTBEAT":
		err = executeHeartbeat(ctx, txnID, job, account, apiKey, apiToken, retryCfg,
			&latencyMs, &retryCount, &httpStatus, &rawBody, &finalStatus, &finalErrMsg, log)

	case "CALLS":
		err = executeCalls(ctx, txnID, job, account, apiKey, apiToken, retryCfg,
			&latencyMs, &retryCount, &httpStatus, &rawBody, &finalStatus, &finalErrMsg, log)

	case "STREAMS":
		err = executeStreams(ctx, txnID, job, account, apiKey, apiToken, retryCfg,
			&latencyMs, &retryCount, &httpStatus, &rawBody, &finalStatus, &finalErrMsg, log)

	default:
		log.Warn("unknown job type, skipping", zap.String("job_type", job.JobType))
		finalStatus = models.TxnStatusFailed
		finalErrMsg = "unknown job type: " + job.JobType
	}

	// ── 5. Update transaction ──────────────────────────────────────────
	_ = repository.UpdateTransactionStatus(txnID, finalStatus, &latencyMs, retryCount, finalErrMsg)

	// ── 6. Refresh snapshots ───────────────────────────────────────────
	go snapshots.RefreshExophoneHealth(job.ExophoneID, job.AccountID, txnID)

	// ── 7. Advance next_run_at ─────────────────────────────────────────
	_ = repository.AdvanceNextRun(job.ID, job.FrequencyMinute, finalStatus)

	log.Info("job completed",
		zap.String("status", finalStatus),
		zap.Int64("latency_ms", latencyMs),
		zap.Int("retries", retryCount),
	)
}

// ─── Heartbeat execution ──────────────────────────────────────────────────────

func executeHeartbeat(ctx context.Context, txnID string, job models.MonitoringJob,
	account *models.Account, apiKey, apiToken string, retryCfg utils.RetryConfig,
	latencyMs *int64, retryCount *int, httpStatus *int, rawBody *string,
	status *string, errMsg *string, log *zap.Logger) error {

	// Resolve the exophone number for VN-level heartbeat.
	exophoneNumber := ""
	if ep, epErr := repository.GetExophoneByID(job.ExophoneID); epErr == nil {
		exophoneNumber = ep.ExophoneNumber
	}

	var hbResult *exotel.FetchHeartbeatResult

	err := utils.DoWithRetry(ctx, "heartbeat", retryCfg, func() (bool, error) {
		result, err := exotel.FetchHeartbeat(account.SID, account.Subdomain, apiKey, apiToken, exophoneNumber)
		hbResult = result
		if result != nil {
			*latencyMs = result.LatencyMs
			*httpStatus = result.HTTPStatus
			*rawBody = result.RawBody
		}
		if err != nil {
			*retryCount++
			return true, err
		}
		if result.HTTPStatus == 429 {
			alerts.TriggerAlert(models.Alert{
				TransactionID: txnID,
				ExophoneID:    job.ExophoneID,
				AccountID:     job.AccountID,
				AlertType:     models.AlertTypeVendorThrottling,
				Severity:      models.SeverityWarning,
				Message:       fmt.Sprintf("Exotel API rate-limited (429) for account %d", job.AccountID),
				Channel:       models.ChannelSlack,
			})
			return false, fmt.Errorf("rate limited (HTTP 429) — will retry at next scheduled run")
		}
		if result.HTTPStatus >= 500 {
			*retryCount++
			return true, fmt.Errorf("server error HTTP %d", result.HTTPStatus)
		}
		return false, nil
	})

	// Save raw response
	_ = repository.SaveJobResponse(&models.JobResponse{
		TransactionID: txnID,
		ResponseType:  "HEARTBEAT",
		HTTPStatus:    *httpStatus,
		RawResponse:   *rawBody,
	})

	if err != nil {
		if *retryCount >= retryCfg.MaxRetries {
			alerts.TriggerAlert(models.Alert{
				TransactionID: txnID,
				ExophoneID:    job.ExophoneID,
				AccountID:     job.AccountID,
				AlertType:     models.AlertTypeRetryExhausted,
				Severity:      models.SeverityCritical,
				Message:       fmt.Sprintf("Max retries exhausted for HEARTBEAT job %d: %v", job.ID, err),
				Channel:       models.ChannelSlack,
			})
		} else {
			alerts.TriggerAlert(models.Alert{
				TransactionID: txnID,
				ExophoneID:    job.ExophoneID,
				AccountID:     job.AccountID,
				AlertType:     models.AlertTypeAPIFailure,
				Severity:      models.SeverityWarning,
				Message:       fmt.Sprintf("Heartbeat API failure for account %d: %v", job.AccountID, err),
				Channel:       models.ChannelSlack,
			})
		}
		*status = models.TxnStatusFailed
		*errMsg = err.Error()
		return err
	}

	// Check high latency
	if *latencyMs > int64(config.App.Alerts.HighLatencyThresholdMs) {
		alerts.TriggerAlert(models.Alert{
			TransactionID: txnID,
			ExophoneID:    job.ExophoneID,
			AccountID:     job.AccountID,
			AlertType:     models.AlertTypeHighLatency,
			Severity:      models.SeverityWarning,
			Message:       fmt.Sprintf("High API latency: %dms (threshold: %dms)", *latencyMs, config.App.Alerts.HighLatencyThresholdMs),
			Channel:       models.ChannelSlack,
		})
	}

	// Persist heartbeat metric + alert on non-OK status
	if hbResult != nil && hbResult.Response != nil {
		hb := hbResult.Response

		// Determine account-level status from affected VN counts.
		statusType := models.HeartbeatOK
		if len(hb.IncomingAffected) > 0 || len(hb.OutgoingAffected) > 0 {
			statusType = models.HeartbeatDegraded
		}
		if hb.StatusType == "CRITICAL" {
			statusType = models.HeartbeatOutage
		}

		// Fire VN-level alert when this specific exophone is degraded or missing.
		if statusType != models.HeartbeatOK && exophoneNumber != "" {
			alerts.TriggerAlert(models.Alert{
				TransactionID: txnID,
				ExophoneID:    job.ExophoneID,
				AccountID:     job.AccountID,
				AlertType:     models.AlertTypeHeartbeatFailure,
				Severity:      models.SeverityCritical,
				Message: fmt.Sprintf(
					"VN %s heartbeat %s: incoming=%v outgoing=%v",
					exophoneNumber, hb.StatusType,
					hb.IncomingAffected, hb.OutgoingAffected,
				),
				Channel: models.ChannelSlack,
			})
		}

		// Fire VN-level alert if this specific exophone is affected.
		if exophoneNumber != "" {
			incDown, outDown := hb.AffectsVN(exophoneNumber)
			if incDown {
				alerts.TriggerAlert(models.Alert{
					TransactionID: txnID,
					ExophoneID:    job.ExophoneID,
					AccountID:     job.AccountID,
					AlertType:     models.AlertTypeExophoneDown,
					Severity:      models.SeverityCritical,
					Message:       fmt.Sprintf("VN %s: incoming calls degraded (in Exotel affected list)", exophoneNumber),
					Channel:       models.ChannelSlack,
				})
			}
			if outDown {
				alerts.TriggerAlert(models.Alert{
					TransactionID: txnID,
					ExophoneID:    job.ExophoneID,
					AccountID:     job.AccountID,
					AlertType:     models.AlertTypeExophoneDown,
					Severity:      models.SeverityCritical,
					Message:       fmt.Sprintf("VN %s: outgoing calls degraded (in Exotel affected list)", exophoneNumber),
					Channel:       models.ChannelSlack,
				})
			}
		}

		_ = repository.SaveHeartbeatMetric(&models.HeartbeatMetric{
			TransactionID:    txnID,
			AccountID:        job.AccountID,
			StatusType:       statusType,
			IncomingAffected: len(hb.IncomingAffected),
			OutgoingAffected: len(hb.OutgoingAffected),
			ResponseTimeMs:   latencyMs,
			RawStatus:        hb.StatusType,
			CreatedBy:        "SYSTEM_METRICS",
		})

		availVal := 100.0
		if statusType != models.HeartbeatOK {
			availVal = 0.0
		}
		_ = repository.SaveMetric(&models.Metric{
			TransactionID:   txnID,
			ExophoneID:      job.ExophoneID,
			AccountID:       job.AccountID,
			MetricName:      "availability",
			MetricValue:     availVal,
			MetricUnit:      "percent",
			MetricTimestamp: time.Now(),
			AggregationType: "INSTANT",
			CreatedBy:       "SYSTEM_METRICS",
		})
	}

	return nil
}

// ─── Calls execution ──────────────────────────────────────────────────────────

func executeCalls(ctx context.Context, txnID string, job models.MonitoringJob,
	account *models.Account, apiKey, apiToken string, retryCfg utils.RetryConfig,
	latencyMs *int64, retryCount *int, httpStatus *int, rawBody *string,
	status *string, errMsg *string, log *zap.Logger) error {

	// Fetch calls for the last frequency_minutes window with 1-minute overlap so
	// boundary calls are never missed. Upsert deduplication handles any overlap.
	to := time.Now()
	from := to.Add(-time.Duration(job.FrequencyMinute+1) * time.Minute)

	// Resolve exophone — number for API filtering, SkipCallLogs for per-exophone write control.
	exophoneNumber := ""
	exophoneSkipLogs := false
	if ep, err := repository.GetExophoneByID(job.ExophoneID); err == nil {
		exophoneNumber = ep.ExophoneNumber
		exophoneSkipLogs = ep.SkipCallLogs == 1
	}

	var callsResult *exotel.FetchCallsResult

	err := utils.DoWithRetry(ctx, "calls", retryCfg, func() (bool, error) {
		result, err := exotel.FetchCalls(account.SID, account.Subdomain, apiKey, apiToken, exophoneNumber, from, to)
		callsResult = result
		if result != nil {
			*latencyMs = result.LatencyMs
			*httpStatus = result.HTTPStatus
			*rawBody = result.RawBody
		}
		if err != nil {
			*retryCount++
			return true, err
		}
		if result.HTTPStatus == 401 || result.HTTPStatus == 403 {
			return false, fmt.Errorf("auth failure HTTP %d — check API credentials", result.HTTPStatus)
		}
		if result.HTTPStatus == 429 {
			// Do not retry — hammering a rate-limited endpoint burns the budget.
			// The scheduler will re-run at the next frequency interval.
			return false, fmt.Errorf("rate limited (HTTP 429) — will retry at next scheduled run")
		}
		if result.HTTPStatus >= 500 {
			*retryCount++
			return true, fmt.Errorf("server error HTTP %d", result.HTTPStatus)
		}
		return false, nil
	})

	// Save each page of the Exotel response as a separate job_response row so that
	// reprocess can reconstruct every call record even if the job was mid-pagination
	// when it was interrupted. One row per page; deduplication by Sid happens at
	// reprocess time.
	if callsResult != nil && len(callsResult.PageBodies) > 0 {
		for _, pageRaw := range callsResult.PageBodies {
			_ = repository.SaveJobResponse(&models.JobResponse{
				TransactionID: txnID,
				ResponseType:  "CALLS",
				HTTPStatus:    *httpStatus,
				RawResponse:   pageRaw,
			})
		}
	} else {
		// Fallback: save whatever raw body we have (e.g. error response or empty result).
		_ = repository.SaveJobResponse(&models.JobResponse{
			TransactionID: txnID,
			ResponseType:  "CALLS",
			HTTPStatus:    *httpStatus,
			RawResponse:   *rawBody,
		})
	}

	if err != nil {
		handleRetryExhausted(txnID, job, err, *retryCount, retryCfg.MaxRetries, "CALLS")
		*status = models.TxnStatusFailed
		*errMsg = err.Error()
		return err
	}

	if *latencyMs > int64(config.App.Alerts.HighLatencyThresholdMs) {
		alerts.TriggerAlert(models.Alert{
			TransactionID: txnID,
			ExophoneID:    job.ExophoneID,
			AccountID:     job.AccountID,
			AlertType:     models.AlertTypeHighLatency,
			Severity:      models.SeverityWarning,
			Message:       fmt.Sprintf("Calls API high latency: %dms", *latencyMs),
			Channel:       models.ChannelSlack,
		})
	}

	if callsResult != nil && callsResult.Response != nil {
		records := callsResult.Response.Result
		log.Info("calls fetched for exophone",
			zap.String("exophone_number", exophoneNumber),
			zap.Int("count", len(records)),
			zap.Int("pages", callsResult.PagesFetched),
		)
		callLogs, mets, summary := metrics.ProcessCallRecords(txnID, job, exophoneNumber, records)
		// Skip writing call_logs if either the global flag is on OR this exophone has skip_call_logs=1.
		if !feature.SkipCallLogsWrite() && !exophoneSkipLogs {
			_ = repository.SaveCallLogs(callLogs)
		}
		_ = repository.SaveMetrics(mets)

		// Alert on high Leg-2 drop rate (customer-side drops).
		if summary.Leg2Total > 0 && summary.DropRate >= metrics.DropRateWarningPct {
			severity := models.SeverityWarning
			if summary.DropRate >= metrics.DropRateCriticalPct {
				severity = models.SeverityCritical
			}
			alerts.TriggerAlert(models.Alert{
				TransactionID: txnID,
				ExophoneID:    job.ExophoneID,
				AccountID:     job.AccountID,
				AlertType:     models.AlertTypeCallDrop,
				Severity:      severity,
				Message: fmt.Sprintf(
					"VN %s — Leg-2 drop rate %.1f%% (%d agent no-answers / %d calls, conversation_duration=0)",
					exophoneNumber, summary.DropRate, summary.DroppedLeg2, summary.Total,
				),
				Channel: models.ChannelSlack,
			})
		}

		// Alert on high Leg-1 drop rate (agent-side / routing failures).
		if summary.Leg1Total > 0 && summary.Leg1DropRate >= metrics.DropRateWarningPct {
			severity := models.SeverityWarning
			if summary.Leg1DropRate >= metrics.DropRateCriticalPct {
				severity = models.SeverityCritical
			}
			alerts.TriggerAlert(models.Alert{
				TransactionID: txnID,
				ExophoneID:    job.ExophoneID,
				AccountID:     job.AccountID,
				AlertType:     models.AlertTypeCallDrop,
				Severity:      severity,
				Message: fmt.Sprintf(
					"VN %s — Leg-1 (IVR abandon) drop rate %.1f%% (%d customer IVR drops / %d calls, conversation_duration=0)",
					exophoneNumber, summary.Leg1DropRate, summary.DroppedLeg1, summary.Total,
				),
				Channel: models.ChannelSlack,
			})
		}

		// Update snapshots: optimised path skips call_logs queries entirely.
		_ = snapshots.AccumulateCallMetrics(job.AccountID, job.ExophoneID, summary)
		if feature.SkipCallLogsWrite() {
			_ = repository.AccumulateAccountDashboardFromSnapshots(job.AccountID)
		} else {
			_ = repository.RecomputeAccountDashboardSnapshot(job.AccountID)
		}
		// Invalidate cached dashboard so next request reads fresh snapshot from DB.
		_ = cache.Delete(ctx, cache.MetricsKey(job.ExophoneID))
	}

	return nil
}

// ─── Streams execution ────────────────────────────────────────────────────────

func executeStreams(ctx context.Context, txnID string, job models.MonitoringJob,
	account *models.Account, apiKey, apiToken string, retryCfg utils.RetryConfig,
	latencyMs *int64, retryCount *int, httpStatus *int, rawBody *string,
	status *string, errMsg *string, log *zap.Logger) error {

	var streamsResult *exotel.FetchStreamsResult

	err := utils.DoWithRetry(ctx, "streams", retryCfg, func() (bool, error) {
		result, err := exotel.FetchActiveStreams(account.SID, account.Subdomain, apiKey, apiToken)
		streamsResult = result
		if result != nil {
			*latencyMs = result.LatencyMs
			*httpStatus = result.HTTPStatus
			*rawBody = result.RawBody
		}
		if err != nil {
			*retryCount++
			return true, err
		}
		if result.HTTPStatus == 429 {
			return false, fmt.Errorf("rate limited (HTTP 429) — will retry at next scheduled run")
		}
		if result.HTTPStatus >= 500 {
			*retryCount++
			return true, fmt.Errorf("server error HTTP %d", result.HTTPStatus)
		}
		return false, nil
	})

	_ = repository.SaveJobResponse(&models.JobResponse{
		TransactionID: txnID,
		ResponseType:  "STREAMS",
		HTTPStatus:    *httpStatus,
		RawResponse:   *rawBody,
	})

	if err != nil {
		handleRetryExhausted(txnID, job, err, *retryCount, retryCfg.MaxRetries, "STREAMS")
		*status = models.TxnStatusFailed
		*errMsg = err.Error()
		return err
	}

	if streamsResult != nil && streamsResult.Response != nil {
		sr := streamsResult.Response
		utilPct := 0.0
		if sr.MaxAllowedStreams > 0 {
			utilPct = float64(sr.ActiveStreams) / float64(sr.MaxAllowedStreams) * 100
		}

		_ = repository.SaveStreamMetric(&models.StreamMetric{
			TransactionID:      txnID,
			AccountID:          job.AccountID,
			ExophoneID:         job.ExophoneID,
			ActiveStreams:       sr.ActiveStreams,
			MaxAllowedStreams:   sr.MaxAllowedStreams,
			UtilizationPercent: utilPct,
			CreatedBy:          "SYSTEM_METRICS",
		})

		// Upsert hourly stream utilization snapshot (aggregates per-hour)
		_ = snapshots.UpdateStreamUtilizationSnapshot(job.AccountID, job.ExophoneID, sr.ActiveStreams, sr.MaxAllowedStreams, utilPct)

		// High utilization alert (> 90%)
		if utilPct > 90 {
			alerts.TriggerAlert(models.Alert{
				TransactionID: txnID,
				ExophoneID:    job.ExophoneID,
				AccountID:     job.AccountID,
				AlertType:     models.AlertTypeExophoneDown,
				Severity:      models.SeverityWarning,
				Message:       fmt.Sprintf("High stream utilization: %.1f%% (%d/%d active)", utilPct, sr.ActiveStreams, sr.MaxAllowedStreams),
				Channel:       models.ChannelSlack,
			})
		}
	}

	return nil
}

// handleRetryExhausted fires a RETRY_EXHAUSTED alert when max retries are hit.
func handleRetryExhausted(txnID string, job models.MonitoringJob, err error, retryCount, maxRetries int, jobType string) {
	if retryCount >= maxRetries {
		alerts.TriggerAlert(models.Alert{
			TransactionID: txnID,
			ExophoneID:    job.ExophoneID,
			AccountID:     job.AccountID,
			AlertType:     models.AlertTypeRetryExhausted,
			Severity:      models.SeverityCritical,
			Message:       fmt.Sprintf("Max retries exhausted for %s job %d: %v", jobType, job.ID, err),
			Channel:       models.ChannelSlack,
		})
	} else {
		alerts.TriggerAlert(models.Alert{
			TransactionID: txnID,
			ExophoneID:    job.ExophoneID,
			AccountID:     job.AccountID,
			AlertType:     models.AlertTypeAPIFailure,
			Severity:      models.SeverityWarning,
			Message:       fmt.Sprintf("%s API failure for exophone %d: %v", jobType, job.ExophoneID, err),
			Channel:       models.ChannelSlack,
		})
	}
}
