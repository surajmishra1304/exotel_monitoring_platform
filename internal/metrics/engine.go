package metrics

import (
	"encoding/json"
	"strings"
	"time"

	"exotel-monitoring-platform/internal/exotel"
	"exotel-monitoring-platform/internal/models"
)

// DropRateWarningPct triggers a WARNING alert when Leg-2 drop_rate exceeds this %.
const DropRateWarningPct = 10.0

// DropRateCriticalPct triggers a CRITICAL alert when Leg-2 drop_rate exceeds this %.
const DropRateCriticalPct = 25.0

// classifyLeg returns 1 or 2 for a call record given the monitored VN number.
//
// Two calling modes exist:
//
//   App-based (agent calls VN):
//     Direction="inbound", To=VN → single trigger record, no Leg 2 child.
//
//   Panel-based (Connect Two Numbers API: From=agent, To=CX, CallerId=VN):
//     Leg 1: Direction="outbound-api", To=agent, ParentCallSid="" — VN dials agent first
//     Leg 2: Direction="outbound-api", To=CX,    ParentCallSid=<Leg1 SID> — VN dials CX
//
// Classification rules (priority order):
//  1. ParentCallSid != "" → Leg 2 (explicitly Exotel-linked child leg)
//  2. Direction == "inbound" → Leg 1 (call INTO the VN, app-based trigger)
//  3. Direction starts with "outbound" AND ParentCallSid == "" → Leg 1
//     (panel-based first leg to agent, or app-based outbound trigger; no parent means initiator)
//  4. To == VN (normalised) → Leg 1 (fallback)
//  5. Otherwise → Leg 2
// ClassifyLeg is exported so cmd tools (e.g. refresh-leg-status) can reuse the
// same leg detection logic without duplicating it.
func ClassifyLeg(r exotel.CallRecord, exophoneNumber string) int {
	if r.ParentCallSid != "" {
		return models.Leg2
	}
	dir := strings.ToLower(r.Direction)
	if dir == "inbound" {
		return models.Leg1
	}
	if strings.HasPrefix(dir, "outbound") {
		// No parent: this is the initiating leg (VN→agent for panel-based, or trigger for app-based).
		return models.Leg1
	}
	vnBare := normalise(exophoneNumber)
	toBare := normalise(r.To)
	if toBare == vnBare {
		return models.Leg1
	}
	return models.Leg2
}

func normalise(num string) string {
	num = strings.TrimPrefix(num, "+91")
	num = strings.TrimPrefix(num, "91")
	num = strings.TrimPrefix(num, "0")
	return num
}

// CallSummary holds the aggregated metrics computed from a batch of call records.
// All Exotel status types are tracked. Leg 1 = agent side; Leg 2 = customer side.
//
// Connected means ConversationDuration > 0 (both parties were actually bridged).
// Dropped means status=completed AND ConversationDuration == 0 (agent never answered).
// Leg2Status from the Details block (details=true required) splits the two drop types:
//   - Leg2Status="" → customer abandoned during IVR before agent routing → Leg1 drop
//   - Leg2Status!="" → agent routing attempted but no-answer            → Leg2 drop
type CallSummary struct {
	Total     int
	Leg1Total int // A-leg calls (IVR inbound or agent-initiated)
	Leg2Total int // B-leg calls (to customer, panel-based only)

	Connected   int // ConversationDuration > 0: both parties were bridged
	DroppedLeg1 int // completed + ConvDur=0 + Leg2Status="": customer abandoned in IVR (no agent routing)
	DroppedLeg2 int // completed + ConvDur=0 + Leg2Status!="": agent routing attempted but no-answer

	Failed   int // top-level status=failed
	NoAnswer int // top-level status=no-answer
	Busy     int // top-level status=busy
	Canceled int // top-level status=canceled
	Other    int // any unrecognised top-level status

	// Leg-level status breakdown from Details block (requires details=true).
	// These are always tracked regardless of top-level status.
	Leg1NoAnswer int // Leg1Status = "no-answer": agent did not pick up
	Leg1Busy     int // Leg1Status = "busy": agent line was busy
	Leg1Failed   int // Leg1Status = "failed": agent leg failure
	Leg2NoAnswer int // Leg2Status = "no-answer": customer did not answer (outbound)
	Leg2Busy     int // Leg2Status = "busy": customer line was busy (outbound)
	Leg2Failed   int // Leg2Status = "failed": customer leg network failure
	Leg2Canceled int // Leg2Status = "canceled": customer hung up while waiting for agent

	TotalDurationSec   int64
	AnswerRate         float64 // connected / total * 100
	SuccessRate        float64 // alias for AnswerRate
	DropRate           float64 // DroppedLeg2 / total * 100 — agent no-answer rate (primary actionable metric)
	Leg1DropRate       float64 // DroppedLeg1 / total * 100 — IVR abandon rate
	Leg2DropRate       float64 // DroppedLeg2 / total * 100 — explicit Leg2 drop rate (== DropRate for IVR mode)
	AvgDurationSec     float64
	HourlyDistribution [24]int
	PeakHour           int
}

// ProcessCallRecords converts raw Exotel call records into CallLog rows and Metric rows.
// exophoneNumber is the VN number (e.g. "07948220638") used to classify Leg 1 vs Leg 2.
func ProcessCallRecords(txnID string, job models.MonitoringJob, exophoneNumber string, records []exotel.CallRecord) ([]models.CallLog, []models.Metric, CallSummary) {
	var callLogs []models.CallLog
	var mets []models.Metric
	var summary CallSummary
	var leg1Connected int // tracks Leg1 CDRs with ConversationDuration > 0 for Panel mode correction

	now := time.Now()

	for _, r := range records {
		dur := r.Duration
		convDur := r.Details.ConversationDuration // 0 = agent never answered; >0 = real conversation
		legNum := ClassifyLeg(r, exophoneNumber)

		cl := models.CallLog{
			TransactionID:        txnID,
			AccountID:            job.AccountID,
			ExophoneID:           job.ExophoneID,
			CallSID:              r.Sid,
			ParentCallSID:        r.ParentCallSid,
			LegNumber:            legNum,
			CallFrom:             r.From,
			CallTo:               r.To,
			Status:               r.Status,
			Direction:            r.Direction,
			DurationSec:          dur,
			ConversationDuration: convDur,
			RecordingURL:         r.RecordingUrl,
			Leg1Status:           r.Details.Leg1Status,
			Leg2Status:           r.Details.Leg2Status,
		}
		if r.StartTime != "" {
			if t, err := parseExotelTime(r.StartTime); err == nil {
				cl.StartTime = &t
				summary.HourlyDistribution[t.Hour()]++
			}
		} else if r.DateCreated != "" {
			if t, err := parseExotelTime(r.DateCreated); err == nil {
				cl.StartTime = &t
				summary.HourlyDistribution[t.Hour()]++
			}
		}
		if r.EndTime != "" {
			if t, err := parseExotelTime(r.EndTime); err == nil {
				cl.EndTime = &t
			}
		}
		callLogs = append(callLogs, cl)

		summary.TotalDurationSec += int64(dur)

		if legNum == models.Leg1 {
			summary.Leg1Total++
		} else {
			summary.Leg2Total++
		}

		// Normalise status to lowercase so Exotel capitalisation variants
		// ("Busy", "No-Answer", "Completed", etc.) are handled correctly.
		status := strings.ToLower(r.Status)

		switch status {
		case "completed":
			// Connected = ConversationDuration > 0: both parties were actually bridged.
			// Dropped = ConversationDuration == 0: no conversation happened.
			//
			// For IVR inbound (all Leg1 records), Leg2Status splits the two drop types:
			//   Leg2Status = ""        → no agent routing was attempted; customer abandoned
			//                            during IVR navigation                → Leg1 drop
			//   Leg2Status = "canceled"→ agent routing was attempted (phone rang)
			//                            but agent did not answer             → Leg2 drop
			//
			// For Panel mode (Leg2 records exist), leg_number determines the drop:
			//   leg_number = 2 AND ConvDur=0 → customer didn't answer         → Leg2 drop
			//   leg_number = 1 AND ConvDur=0 AND Leg2Status="" → agent didn't answer → Leg1 drop
			if convDur > 0 {
				summary.Connected++
				if legNum == models.Leg1 {
					leg1Connected++
				}
			} else if legNum == models.Leg2 {
				// Panel mode: customer-side (Leg2) no-answer
				summary.DroppedLeg2++
			} else if strings.ToLower(r.Details.Leg2Status) == "" {
				// IVR: no agent routing attempted — customer dropped in IVR
				summary.DroppedLeg1++
			} else {
				// IVR: agent routing attempted (Leg2Status="canceled") — agent no-answer
				summary.DroppedLeg2++
			}
		case "failed":
			summary.Failed++
		case "no-answer", "noanswer":
			summary.NoAnswer++
		case "busy":
			summary.Busy++
		case "canceled", "cancelled":
			summary.Canceled++
		case "in-progress", "queued", "initiated", "ringing":
			// Live calls still in flight when the API was polled; not terminal outcomes.
			// Count them separately so they don't inflate other buckets.
			summary.Other++
		default:
			summary.Other++
		}

		// Leg-level status breakdown — always populated when details=true is used.
		// These are independent of the top-level status switch above.
		switch strings.ToLower(r.Details.Leg1Status) {
		case "no-answer", "noanswer":
			summary.Leg1NoAnswer++
		case "busy":
			summary.Leg1Busy++
		case "failed":
			summary.Leg1Failed++
		}
		switch strings.ToLower(r.Details.Leg2Status) {
		case "no-answer", "noanswer":
			summary.Leg2NoAnswer++
		case "busy":
			summary.Leg2Busy++
		case "failed":
			summary.Leg2Failed++
		case "canceled", "cancelled":
			summary.Leg2Canceled++
		}
	}

	summary.Total = len(records)
	if summary.Total == 0 {
		return callLogs, mets, summary
	}

	// Panel mode produces two CDRs per bridged call: one Leg1 (VN→agent) and one Leg2
	// (VN→customer), both with ConversationDuration > 0. Without correction, Connected
	// would be double-counted. Subtract the Leg1 count so Connected equals the number
	// of unique calls where the customer was actually reached.
	if summary.Leg2Total > 0 {
		summary.Connected -= leg1Connected
	}

	summary.AvgDurationSec = float64(summary.TotalDurationSec) / float64(summary.Total)

	// Panel-based (Leg2 records exist): denominator = Leg2Total.
	// IVR/app-based (all inbound, no Leg2 records): denominator = Total.
	//   DropRate     = Leg2 drops / denom  — agent no-answer rate (primary actionable metric)
	//   Leg2DropRate = Leg2 drops / Total  — always Total-denominated for consistency
	//   Leg1DropRate = Leg1 drops / Total  — IVR abandon rate
	rateDenom := summary.Total
	if summary.Leg2Total > 0 {
		rateDenom = summary.Leg2Total
		summary.DropRate = float64(summary.DroppedLeg2) / float64(summary.Leg2Total) * 100
	} else {
		summary.DropRate = float64(summary.DroppedLeg2) / float64(summary.Total) * 100
	}
	if rateDenom > 0 {
		summary.AnswerRate = float64(summary.Connected) / float64(rateDenom) * 100
	}
	summary.SuccessRate = summary.AnswerRate
	// Leg1DropRate: drops per Leg1 call — consistent with AccumulateCallMetrics and the model definition.
	// Falls back to Total when Leg1Total is zero (pure Leg2-only edge case, should not occur in practice).
	if summary.Leg1Total > 0 {
		summary.Leg1DropRate = float64(summary.DroppedLeg1) / float64(summary.Leg1Total) * 100
	} else if summary.Total > 0 {
		summary.Leg1DropRate = float64(summary.DroppedLeg1) / float64(summary.Total) * 100
	}
	// Leg2DropRate uses Total as denominator for cross-mode consistency.
	summary.Leg2DropRate = float64(summary.DroppedLeg2) / float64(summary.Total) * 100

	// Find peak hour.
	maxCalls := 0
	for h, count := range summary.HourlyDistribution {
		if count > maxCalls {
			maxCalls = count
			summary.PeakHour = h
		}
	}

	failurePct := float64(summary.Failed+summary.NoAnswer) / float64(summary.Total) * 100

	mets = append(mets,
		buildMetric(txnID, job, "total_calls", float64(summary.Total), "count", now),
		buildMetric(txnID, job, "leg1_total", float64(summary.Leg1Total), "count", now),
		buildMetric(txnID, job, "leg2_total", float64(summary.Leg2Total), "count", now),
		buildMetric(txnID, job, "connected_calls", float64(summary.Connected), "count", now),
		buildMetric(txnID, job, "dropped_leg1", float64(summary.DroppedLeg1), "count", now),
		buildMetric(txnID, job, "dropped_leg2", float64(summary.DroppedLeg2), "count", now),
		buildMetric(txnID, job, "failed_calls", float64(summary.Failed), "count", now),
		buildMetric(txnID, job, "no_answer_calls", float64(summary.NoAnswer), "count", now),
		buildMetric(txnID, job, "busy_calls", float64(summary.Busy), "count", now),
		buildMetric(txnID, job, "canceled_calls", float64(summary.Canceled), "count", now),
		buildMetric(txnID, job, "other_calls", float64(summary.Other), "count", now),
		buildMetric(txnID, job, "avg_duration_sec", summary.AvgDurationSec, "seconds", now),
		buildMetric(txnID, job, "answer_rate", summary.AnswerRate, "percent", now),
		buildMetric(txnID, job, "drop_rate", summary.DropRate, "percent", now),
		buildMetric(txnID, job, "leg1_drop_rate", summary.Leg1DropRate, "percent", now),
		buildMetric(txnID, job, "leg2_drop_rate", summary.Leg2DropRate, "percent", now),
		buildMetric(txnID, job, "failure_percent", failurePct, "percent", now),
	)

	return callLogs, mets, summary
}

// HourlyDistributionJSON encodes the 24-element hourly array to a JSON string for DB storage.
func HourlyDistributionJSON(dist [24]int) string {
	slice := dist[:]
	b, err := json.Marshal(slice)
	if err != nil {
		return "[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]"
	}
	return string(b)
}

func buildMetric(txnID string, job models.MonitoringJob, name string, value float64, unit string, ts time.Time) models.Metric {
	return models.Metric{
		TransactionID:   txnID,
		ExophoneID:      job.ExophoneID,
		AccountID:       job.AccountID,
		MetricName:      name,
		MetricValue:     value,
		MetricUnit:      unit,
		MetricTimestamp: ts,
		AggregationType: "SUM",
		CreatedBy:       "SYSTEM_METRICS",
	}
}

// ist is the IST timezone (UTC+5:30).
// Exotel timestamps without a timezone indicator are always in IST.
var ist = time.FixedZone("IST", 5*60*60+30*60)

// parseExotelTime handles the common Exotel timestamp formats.
// The bare "YYYY-MM-DD HH:MM:SS" format has no timezone — Exotel returns IST,
// so we parse it with ParseInLocation to avoid a 5h30m UTC offset error.
func parseExotelTime(s string) (time.Time, error) {
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", s, ist); err == nil {
		return t, nil
	}
	for _, f := range []string{time.RFC3339, "Mon, 02 Jan 2006 15:04:05 -0700"} {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, nil
}
