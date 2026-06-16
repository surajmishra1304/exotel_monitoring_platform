package metrics

import (
	"encoding/json"
	"testing"

	"exotel-monitoring-platform/internal/exotel"
	"exotel-monitoring-platform/internal/models"
)

// makeJob returns a minimal MonitoringJob for use in tests.
func makeJob() models.MonitoringJob {
	return models.MonitoringJob{ID: 1, AccountID: 10, ExophoneID: 20, JobType: "CALLS"}
}

// TestProcessCallRecords_StatusClassification verifies that Exotel's four call status
// values map correctly to the CallSummary counters.
// Exotel status taxonomy (from API docs): completed, no-answer, failed, busy.
// Connected requires both status=completed AND ConversationDuration > 0 (details=true).
func TestProcessCallRecords_StatusClassification(t *testing.T) {
	records := []exotel.CallRecord{
		// Completed + ConversationDuration > 0 → Connected
		{Sid: "CA001", Status: "completed", StartTime: "2026-06-07 10:00:00", Duration: 120,
			Direction: "inbound", Details: exotel.CallDetails{ConversationDuration: 120}},
		{Sid: "CA002", Status: "completed", StartTime: "2026-06-07 10:30:00", Duration: 60,
			Direction: "inbound", Details: exotel.CallDetails{ConversationDuration: 60}},
		{Sid: "CA003", Status: "no-answer", StartTime: "2026-06-07 11:00:00", Direction: "inbound", Duration: 0},
		{Sid: "CA004", Status: "failed",    StartTime: "2026-06-07 14:00:00", Direction: "inbound", Duration: 0},
		{Sid: "CA005", Status: "busy",      StartTime: "2026-06-07 15:00:00", Direction: "inbound", Duration: 0},
		{Sid: "CA006", Status: "no-answer", StartTime: "2026-06-07 16:00:00", Direction: "inbound", Duration: 0},
	}

	_, _, summary := ProcessCallRecords("TXN-001", makeJob(), "08047493481", records)

	if summary.Total != 6 {
		t.Errorf("Total = %d, want 6", summary.Total)
	}
	if summary.Connected != 2 {
		t.Errorf("Connected = %d, want 2 (requires ConversationDuration > 0)", summary.Connected)
	}
	if summary.NoAnswer != 2 {
		t.Errorf("NoAnswer = %d, want 2", summary.NoAnswer)
	}
	if summary.Failed != 1 {
		t.Errorf("Failed = %d, want 1", summary.Failed)
	}
	if summary.Busy != 1 {
		t.Errorf("Busy = %d, want 1", summary.Busy)
	}
}

// TestProcessCallRecords_AnswerRate verifies that AnswerRate = Connected/Total * 100.
// ConversationDuration > 0 is required for a call to count as Connected (details=true API param).
func TestProcessCallRecords_AnswerRate(t *testing.T) {
	records := []exotel.CallRecord{
		{Sid: "CA001", Status: "completed", StartTime: "2026-06-07 09:00:00", Direction: "inbound",
			Duration: 90, Details: exotel.CallDetails{ConversationDuration: 90}},
		{Sid: "CA002", Status: "completed", StartTime: "2026-06-07 09:10:00", Direction: "inbound",
			Duration: 60, Details: exotel.CallDetails{ConversationDuration: 60}},
		{Sid: "CA003", Status: "completed", StartTime: "2026-06-07 09:20:00", Direction: "inbound",
			Duration: 45, Details: exotel.CallDetails{ConversationDuration: 45}},
		{Sid: "CA004", Status: "no-answer", StartTime: "2026-06-07 09:30:00", Direction: "inbound", Duration: 0},
	}

	_, _, summary := ProcessCallRecords("TXN-002", makeJob(), "08047493481", records)

	wantRate := 75.0 // 3/4 * 100
	if summary.AnswerRate != wantRate {
		t.Errorf("AnswerRate = %.2f, want %.2f", summary.AnswerRate, wantRate)
	}
	if summary.SuccessRate != summary.AnswerRate {
		t.Errorf("SuccessRate (%f) should equal AnswerRate (%f)", summary.SuccessRate, summary.AnswerRate)
	}
}

// TestProcessCallRecords_HourlyDistribution verifies the 24-slot hourly bucketing.
func TestProcessCallRecords_HourlyDistribution(t *testing.T) {
	records := []exotel.CallRecord{
		{Sid: "CA001", Status: "completed", StartTime: "2026-06-07 09:00:00", Duration: 30},
		{Sid: "CA002", Status: "completed", StartTime: "2026-06-07 09:30:00", Duration: 30},
		{Sid: "CA003", Status: "completed", StartTime: "2026-06-07 09:45:00", Duration: 30},
		{Sid: "CA004", Status: "completed", StartTime: "2026-06-07 14:00:00", Duration: 30},
		{Sid: "CA005", Status: "no-answer", StartTime: "2026-06-07 14:30:00", Duration: 0},
	}

	_, _, summary := ProcessCallRecords("TXN-003", makeJob(), "08047493481", records)

	if summary.HourlyDistribution[9] != 3 {
		t.Errorf("Hour 09 count = %d, want 3", summary.HourlyDistribution[9])
	}
	if summary.HourlyDistribution[14] != 2 {
		t.Errorf("Hour 14 count = %d, want 2", summary.HourlyDistribution[14])
	}
	// All other hours should be 0
	for h := range 24 {
		if h == 9 || h == 14 {
			continue
		}
		if summary.HourlyDistribution[h] != 0 {
			t.Errorf("Hour %02d count = %d, want 0", h, summary.HourlyDistribution[h])
		}
	}
}

// TestProcessCallRecords_PeakHour verifies that PeakHour is the hour with most calls.
func TestProcessCallRecords_PeakHour(t *testing.T) {
	records := []exotel.CallRecord{
		{Sid: "CA001", Status: "completed", StartTime: "2026-06-07 09:00:00", Duration: 30},
		{Sid: "CA002", Status: "completed", StartTime: "2026-06-07 14:00:00", Duration: 30},
		{Sid: "CA003", Status: "completed", StartTime: "2026-06-07 14:30:00", Duration: 30},
		{Sid: "CA004", Status: "no-answer", StartTime: "2026-06-07 14:50:00", Duration: 0},
	}

	_, _, summary := ProcessCallRecords("TXN-004", makeJob(), "08047493481", records)

	if summary.PeakHour != 14 {
		t.Errorf("PeakHour = %d, want 14 (3 calls vs 1 at hour 9)", summary.PeakHour)
	}
}

// TestProcessCallRecords_AvgDuration verifies average duration across all calls.
func TestProcessCallRecords_AvgDuration(t *testing.T) {
	records := []exotel.CallRecord{
		{Sid: "CA001", Status: "completed", StartTime: "2026-06-07 09:00:00", Duration: 100},
		{Sid: "CA002", Status: "completed", StartTime: "2026-06-07 10:00:00", Duration: 200},
		{Sid: "CA003", Status: "no-answer", StartTime: "2026-06-07 11:00:00", Duration: 0},
	}

	_, _, summary := ProcessCallRecords("TXN-005", makeJob(), "08047493481", records)

	// AvgDuration = (100+200+0) / 3 = 100.0
	want := 100.0
	if summary.AvgDurationSec != want {
		t.Errorf("AvgDurationSec = %.2f, want %.2f", summary.AvgDurationSec, want)
	}
}

// TestProcessCallRecords_Empty verifies graceful handling of zero records.
func TestProcessCallRecords_Empty(t *testing.T) {
	_, _, summary := ProcessCallRecords("TXN-006", makeJob(), "08047493481", nil)
	if summary.Total != 0 {
		t.Errorf("empty input: Total = %d, want 0", summary.Total)
	}
	if summary.AnswerRate != 0 {
		t.Errorf("empty input: AnswerRate = %f, want 0", summary.AnswerRate)
	}
}

// TestProcessCallRecords_CallLogFields verifies that call logs are produced with the
// correct CallSID and Status fields from the Exotel record.
func TestProcessCallRecords_CallLogFields(t *testing.T) {
	records := []exotel.CallRecord{
		{Sid: "CA-ABC123", Status: "completed", StartTime: "2026-06-07 10:00:00", Duration: 45,
			From: "+919876543210", To: "+912233445566", Direction: "inbound"},
	}

	callLogs, _, _ := ProcessCallRecords("TXN-007", makeJob(), "08047493481", records)

	if len(callLogs) != 1 {
		t.Fatalf("expected 1 call log, got %d", len(callLogs))
	}
	cl := callLogs[0]
	if cl.CallSID != "CA-ABC123" {
		t.Errorf("CallSID = %q, want %q", cl.CallSID, "CA-ABC123")
	}
	if cl.Status != "completed" {
		t.Errorf("Status = %q, want %q", cl.Status, "completed")
	}
	if cl.DurationSec != 45 {
		t.Errorf("DurationSec = %d, want 45", cl.DurationSec)
	}
	if cl.AccountID != makeJob().AccountID {
		t.Errorf("AccountID = %d, want %d", cl.AccountID, makeJob().AccountID)
	}
}

// TestProcessCallRecords_MetricsProduced verifies that all 8 metric rows are emitted.
func TestProcessCallRecords_MetricsProduced(t *testing.T) {
	records := []exotel.CallRecord{
		{Sid: "CA001", Status: "completed", StartTime: "2026-06-07 10:00:00", Duration: 60},
	}

	_, mets, _ := ProcessCallRecords("TXN-008", makeJob(), "08047493481", records)

	wantMetrics := map[string]bool{
		"total_calls":     false,
		"connected_calls": false,
		"failed_calls":    false,
		"no_answer_calls": false,
		"busy_calls":      false,
		"avg_duration_sec": false,
		"answer_rate":     false,
		"failure_percent": false,
	}
	for _, m := range mets {
		wantMetrics[m.MetricName] = true
	}
	for name, found := range wantMetrics {
		if !found {
			t.Errorf("metric %q not emitted", name)
		}
	}
}

// TestClassifyLeg verifies all five leg-classification rules against Exotel's CDR fields.
// Rule priority: ParentCallSid → Direction=inbound → Direction=outbound+no-parent → To==VN → catch-all.
func TestClassifyLeg(t *testing.T) {
	const vn = "08047493481"
	cases := []struct {
		name   string
		r      exotel.CallRecord
		want   int
	}{
		{
			name: "Rule1: ParentCallSid set → Leg2 (C2C customer record)",
			r:    exotel.CallRecord{ParentCallSid: "CA-PARENT", Direction: "outbound-api", To: "9999999999"},
			want: models.Leg2,
		},
		{
			name: "Rule2: Direction=inbound → Leg1 (IVR customer calls VN)",
			r:    exotel.CallRecord{ParentCallSid: "", Direction: "inbound", To: vn},
			want: models.Leg1,
		},
		{
			name: "Rule3: outbound-api + no parent → Leg1 (C2C VN→agent initiating leg)",
			r:    exotel.CallRecord{ParentCallSid: "", Direction: "outbound-api", To: "9876543210"},
			want: models.Leg1,
		},
		{
			name: "Rule3: outbound-dial + no parent → Leg1 (simple outbound)",
			r:    exotel.CallRecord{ParentCallSid: "", Direction: "outbound-dial", To: "9876543210"},
			want: models.Leg1,
		},
		{
			name: "Rule4: To matches VN (no direction) → Leg1 fallback",
			r:    exotel.CallRecord{ParentCallSid: "", Direction: "", To: vn},
			want: models.Leg1,
		},
		{
			name: "Rule5: catch-all unknown direction → Leg2",
			r:    exotel.CallRecord{ParentCallSid: "", Direction: "unknown", To: "9876543210"},
			want: models.Leg2,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyLeg(tc.r, vn)
			if got != tc.want {
				t.Errorf("ClassifyLeg = %d, want %d", got, tc.want)
			}
		})
	}
}

// TestProcessCallRecords_ConversationDurationGate verifies that status=completed alone
// is NOT enough for Connected — ConversationDuration=0 means the call dropped.
func TestProcessCallRecords_ConversationDurationGate(t *testing.T) {
	records := []exotel.CallRecord{
		// Completed but ConversationDuration=0 → drop, NOT connected
		{Sid: "CA001", Status: "completed", Direction: "inbound",
			StartTime: "2026-06-07 10:00:00", Duration: 30,
			Details: exotel.CallDetails{ConversationDuration: 0}},
		// Completed with ConversationDuration > 0 → connected
		{Sid: "CA002", Status: "completed", Direction: "inbound",
			StartTime: "2026-06-07 10:05:00", Duration: 60,
			Details: exotel.CallDetails{ConversationDuration: 60}},
	}

	_, _, summary := ProcessCallRecords("TXN-GATE", makeJob(), "08047493481", records)

	if summary.Connected != 1 {
		t.Errorf("Connected = %d, want 1 (only CA002 has ConversationDuration > 0)", summary.Connected)
	}
	if summary.DroppedLeg1 != 1 {
		t.Errorf("DroppedLeg1 = %d, want 1 (CA001 completed+convDur=0+Leg2Status='')", summary.DroppedLeg1)
	}
}

// TestProcessCallRecords_IVRDropClassification verifies the Leg2Status split for IVR mode:
//   - Leg2Status=""        → customer abandoned before agent routing → Leg1 drop
//   - Leg2Status="canceled"→ agent routing attempted but agent didn't answer → Leg2 drop
func TestProcessCallRecords_IVRDropClassification(t *testing.T) {
	records := []exotel.CallRecord{
		// Leg1 drop: IVR inbound, completed, convDur=0, Leg2Status="" (no routing attempted)
		{Sid: "CA001", Status: "completed", Direction: "inbound",
			StartTime: "2026-06-07 10:00:00",
			Details:   exotel.CallDetails{ConversationDuration: 0, Leg2Status: ""}},
		// Leg2 drop: IVR inbound, completed, convDur=0, Leg2Status="canceled" (agent didn't answer)
		{Sid: "CA002", Status: "completed", Direction: "inbound",
			StartTime: "2026-06-07 10:05:00",
			Details:   exotel.CallDetails{ConversationDuration: 0, Leg2Status: "canceled"}},
		// Another Leg2 drop variant: Leg2Status="no-answer"
		{Sid: "CA003", Status: "completed", Direction: "inbound",
			StartTime: "2026-06-07 10:10:00",
			Details:   exotel.CallDetails{ConversationDuration: 0, Leg2Status: "no-answer"}},
	}

	_, _, summary := ProcessCallRecords("TXN-IVR", makeJob(), "08047493481", records)

	if summary.DroppedLeg1 != 1 {
		t.Errorf("DroppedLeg1 = %d, want 1 (CA001: Leg2Status='')", summary.DroppedLeg1)
	}
	if summary.DroppedLeg2 != 2 {
		t.Errorf("DroppedLeg2 = %d, want 2 (CA002+CA003: Leg2Status non-empty)", summary.DroppedLeg2)
	}
	if summary.Connected != 0 {
		t.Errorf("Connected = %d, want 0 (all convDur=0)", summary.Connected)
	}
	// Leg2Canceled should count the "canceled" Leg2Status entry
	if summary.Leg2Canceled != 1 {
		t.Errorf("Leg2Canceled = %d, want 1", summary.Leg2Canceled)
	}
	// Leg2NoAnswer should count the "no-answer" Leg2Status entry
	if summary.Leg2NoAnswer != 1 {
		t.Errorf("Leg2NoAnswer = %d, want 1", summary.Leg2NoAnswer)
	}
}

// TestProcessCallRecords_PanelModeConnectedDedup verifies that Connected is NOT
// double-counted when both the Leg1 (VN→agent) and Leg2 (VN→customer) CDRs have
// ConversationDuration > 0 in a Panel/C2C call.
//
// One real connected call must produce Connected=1, not Connected=2.
func TestProcessCallRecords_PanelModeConnectedDedup(t *testing.T) {
	const vn = "08047493481"
	records := []exotel.CallRecord{
		// Leg1: VN→agent (outbound-api, no parent) — connected
		{Sid: "CA-LEG1", Status: "completed", Direction: "outbound-api",
			ParentCallSid: "", From: vn, To: "9876543210",
			StartTime: "2026-06-07 10:00:00",
			Details:   exotel.CallDetails{ConversationDuration: 120}},
		// Leg2: VN→customer (outbound-api, ParentCallSid set) — connected
		{Sid: "CA-LEG2", Status: "completed", Direction: "outbound-api",
			ParentCallSid: "CA-LEG1", From: vn, To: "9999999999",
			StartTime: "2026-06-07 10:00:05",
			Details:   exotel.CallDetails{ConversationDuration: 115}},
	}

	_, _, summary := ProcessCallRecords("TXN-PANEL", makeJob(), vn, records)

	if summary.Leg1Total != 1 {
		t.Errorf("Leg1Total = %d, want 1", summary.Leg1Total)
	}
	if summary.Leg2Total != 1 {
		t.Errorf("Leg2Total = %d, want 1", summary.Leg2Total)
	}
	// Panel mode: leg1Connected correction removes the Leg1 CDR contribution.
	// Connected must equal 1, not 2.
	if summary.Connected != 1 {
		t.Errorf("Connected = %d, want 1 (Panel mode: Leg1 CDR must not double-count)", summary.Connected)
	}
	// AnswerRate = Connected/Leg2Total * 100 = 1/1 * 100 = 100%
	if summary.AnswerRate != 100.0 {
		t.Errorf("AnswerRate = %.2f, want 100.00 (1 connected / 1 customer-side attempt)", summary.AnswerRate)
	}
}

// TestProcessCallRecords_PanelModePartialConnect verifies Panel mode where agent connects
// but customer does not: Leg1 connected, Leg2 dropped → Connected=0, DroppedLeg2=1.
func TestProcessCallRecords_PanelModePartialConnect(t *testing.T) {
	const vn = "08047493481"
	records := []exotel.CallRecord{
		// Leg1: agent picked up (ConversationDuration > 0)
		{Sid: "CA-LEG1", Status: "completed", Direction: "outbound-api",
			ParentCallSid: "", From: vn, To: "9876543210",
			StartTime: "2026-06-07 10:00:00",
			Details:   exotel.CallDetails{ConversationDuration: 30}},
		// Leg2: customer didn't answer (ConversationDuration = 0, Leg2 CDR)
		{Sid: "CA-LEG2", Status: "completed", Direction: "outbound-api",
			ParentCallSid: "CA-LEG1", From: vn, To: "9999999999",
			StartTime: "2026-06-07 10:00:05",
			Details:   exotel.CallDetails{ConversationDuration: 0}},
	}

	_, _, summary := ProcessCallRecords("TXN-PANEL2", makeJob(), vn, records)

	// Leg1 was "connected" but after Panel mode correction it is subtracted.
	// Leg2 was not connected. Net Connected = 0.
	if summary.Connected != 0 {
		t.Errorf("Connected = %d, want 0 (customer didn't answer, no real conversation)", summary.Connected)
	}
	if summary.DroppedLeg2 != 1 {
		t.Errorf("DroppedLeg2 = %d, want 1 (Leg2 CDR with convDur=0)", summary.DroppedLeg2)
	}
}

// TestHourlyDistributionJSON verifies the JSON encoding produces a 24-element array.
func TestHourlyDistributionJSON(t *testing.T) {
	var dist [24]int
	dist[9] = 3
	dist[14] = 2

	got := HourlyDistributionJSON(dist)

	var arr []int
	if err := json.Unmarshal([]byte(got), &arr); err != nil {
		t.Fatalf("HourlyDistributionJSON produced invalid JSON: %v", err)
	}
	if len(arr) != 24 {
		t.Errorf("expected 24 elements, got %d", len(arr))
	}
	if arr[9] != 3 {
		t.Errorf("arr[9] = %d, want 3", arr[9])
	}
	if arr[14] != 2 {
		t.Errorf("arr[14] = %d, want 2", arr[14])
	}
}

// TestParseExotelTime verifies all three timestamp formats Exotel uses in practice.
func TestParseExotelTime(t *testing.T) {
	cases := []struct {
		name  string
		input string
		wantOK bool
	}{
		{"standard DB format",     "2026-06-07 14:30:00",                  true},
		{"RFC3339",                 "2026-06-07T14:30:00Z",                 true},
		{"RFC2822 with offset",     "Mon, 07 Jun 2026 14:30:00 +0530",      true},
		{"empty string",            "",                                      false},
		{"garbage",                 "not-a-date",                            false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := parseExotelTime(tc.input)
			if tc.wantOK && got.IsZero() {
				t.Errorf("parseExotelTime(%q): expected non-zero time", tc.input)
			}
			if !tc.wantOK && !got.IsZero() {
				t.Errorf("parseExotelTime(%q): expected zero time, got %v", tc.input, got)
			}
		})
	}
}
