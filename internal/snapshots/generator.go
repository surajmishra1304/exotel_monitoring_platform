package snapshots

import (
	"encoding/json"
	"time"

	"exotel-monitoring-platform/internal/logger"
	"exotel-monitoring-platform/internal/metrics"
	"exotel-monitoring-platform/internal/models"
	"exotel-monitoring-platform/internal/repository"
	"go.uber.org/zap"
)

// RefreshExophoneHealth updates the exophone_health_snapshot for a given exophone
// after any monitoring job completes. It aggregates the latest heartbeat + stream data.
func RefreshExophoneHealth(exophoneID, accountID uint64, txnID string) {
	log := logger.Log.With(
		zap.Uint64("exophone_id", exophoneID),
		zap.String("transaction_id", txnID),
	)

	snap := &models.ExophoneHealthSnapshot{
		ExophoneID:      exophoneID,
		AccountID:       accountID,
		HeartbeatStatus: "UNKNOWN",
	}
	now := time.Now()
	snap.LastCheckedAt = &now

	// Pull latest heartbeat
	hb, err := repository.GetLatestHeartbeat(accountID)
	if err == nil {
		snap.HeartbeatStatus = hb.StatusType
		snap.AvailabilityPercent = heartbeatToAvailability(hb.StatusType)
		if hb.ResponseTimeMs != nil {
			snap.LastAPILatencyMs = hb.ResponseTimeMs
		}
	}

	// Pull latest stream metric
	sm, err := repository.GetLatestStreamMetric(exophoneID)
	if err == nil {
		snap.ActiveStreams = sm.ActiveStreams
	}

	if err := repository.UpsertExophoneHealthSnapshot(snap); err != nil {
		log.Error("failed to upsert health snapshot", zap.Error(err))
		return
	}

	log.Debug("exophone health snapshot refreshed")
}

// AccumulateCallMetrics incrementally adds a batch summary into today's
// call_metrics_snapshot without touching call_logs.
//
// Each CALLS job run covers a 15-min window. Instead of re-aggregating all raw
// rows for the day, we read the existing snapshot, add the new batch's counts,
// recalculate derived rates, and upsert — O(1) DB work per run regardless of
// how many calls have been processed today.
func AccumulateCallMetrics(accountID, exophoneID uint64, s metrics.CallSummary) error {
	if s.Total == 0 {
		return nil
	}

	// Use IST (UTC+5:30) to determine the current date.
	// time.Truncate operates on absolute UTC, so Truncate(24h) would give UTC midnight
	// which is 5:30 AM IST — meaning calls between midnight and 5:30 AM IST would land
	// on the wrong snapshot date on an IST-deployed server.
	ist := time.FixedZone("IST", 5*60*60+30*60)
	nowIST := time.Now().In(ist)
	today := time.Date(nowIST.Year(), nowIST.Month(), nowIST.Day(), 0, 0, 0, 0, time.UTC)

	// Fetch existing snapshot so we can accumulate on top of it.
	existing, err := repository.GetCallMetricsSnapshot(exophoneID, today)

	var snap models.CallMetricsSnapshot
	if err == nil && existing != nil {
		snap = *existing
	} else {
		snap = models.CallMetricsSnapshot{
			ExophoneID:   exophoneID,
			AccountID:    accountID,
			SnapshotDate: today,
		}
	}

	// Accumulate raw counts.
	snap.TotalCalls += s.Total
	snap.Leg1Total += s.Leg1Total
	snap.Leg1Drops += s.DroppedLeg1
	snap.Leg2Total += s.Leg2Total
	snap.Leg2Drops += s.DroppedLeg2
	snap.ConnectedCalls += s.Connected
	// DroppedCalls must always equal Leg1Drops + Leg2Drops.
	// Never accumulate it independently — if old rows had DroppedCalls set before
	// Leg1/Leg2 tracking was added, independent accumulation would cause them to diverge.
	snap.DroppedCalls = snap.Leg1Drops + snap.Leg2Drops
	snap.FailedCalls += s.Failed
	snap.NoAnswerCalls += s.NoAnswer
	snap.BusyCalls += s.Busy
	snap.CanceledCalls += s.Canceled
	snap.OtherCalls += s.Other
	snap.Leg1NoAnswer += s.Leg1NoAnswer
	snap.Leg1Busy += s.Leg1Busy
	snap.Leg1Failed += s.Leg1Failed
	snap.Leg2NoAnswer += s.Leg2NoAnswer
	snap.Leg2Busy += s.Leg2Busy
	snap.Leg2Failed += s.Leg2Failed
	snap.Leg2Canceled += s.Leg2Canceled

	// Weighted-average duration: reconstruct existing total from avg * count.
	existingTotalDur := snap.AvgDurationSec * float64(snap.TotalCalls-s.Total)
	newTotalDur := existingTotalDur + float64(s.TotalDurationSec)
	if snap.TotalCalls > 0 {
		snap.AvgDurationSec = newTotalDur / float64(snap.TotalCalls)
	}

	// Accumulate hourly distribution.
	var dist [24]int
	if snap.HourlyDistribution != "" {
		_ = json.Unmarshal([]byte(snap.HourlyDistribution), &dist)
	}
	for h, v := range s.HourlyDistribution {
		dist[h] += v
	}
	distJSON, _ := json.Marshal(dist[:])
	snap.HourlyDistribution = string(distJSON)

	// Recompute peak hour.
	peak, peakCount := 0, 0
	for h, v := range dist {
		if v > peakCount {
			peakCount = v
			peak = h
		}
	}
	snap.PeakHour = peak

	// Recompute derived rates from accumulated totals.
	denom := snap.TotalCalls
	if snap.Leg2Total > 0 {
		denom = snap.Leg2Total
		snap.DropRate = float64(snap.Leg2Drops) / float64(snap.Leg2Total) * 100
	} else if snap.TotalCalls > 0 {
		snap.DropRate = float64(snap.Leg2Drops) / float64(snap.TotalCalls) * 100
	}
	if denom > 0 {
		snap.AnswerRate = float64(snap.ConnectedCalls) / float64(denom) * 100
	}
	snap.SuccessRate = snap.AnswerRate
	if snap.Leg1Total > 0 {
		snap.Leg1DropRate = float64(snap.Leg1Drops) / float64(snap.Leg1Total) * 100
	}

	return repository.UpsertCallMetricsSnapshot(&snap)
}

// UpdateStreamUtilizationSnapshot upserts the current-hour stream utilization snapshot.
func UpdateStreamUtilizationSnapshot(accountID, exophoneID uint64, active, maxAllowed int, utilPct float64) error {
	// Truncate to current hour
	now := time.Now()
	hour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())

	snap := &models.StreamUtilizationSnapshot{
		SnapshotHour:       hour,
		AccountID:          accountID,
		ExophoneID:         exophoneID,
		AvgActiveStreams:    float64(active),
		MaxActiveStreams:    active,
		UtilizationPercent: utilPct,
	}

	return repository.UpsertStreamUtilizationSnapshot(snap)
}

// heartbeatToAvailability converts a heartbeat status string to an availability percentage.
func heartbeatToAvailability(status string) float64 {
	switch status {
	case models.HeartbeatOK:
		return 100.0
	case models.HeartbeatDegraded:
		return 50.0
	case models.HeartbeatOutage:
		return 0.0
	default:
		return 0.0
	}
}
