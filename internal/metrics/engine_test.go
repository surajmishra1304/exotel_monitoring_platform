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
func TestProcessCallRecords_StatusClassification(t *testing.T) {
	records := []exotel.CallRecord{
		{Sid: "CA001", Status: "completed",  StartTime: "2026-06-07 10:00:00", Duration: 120},
		{Sid: "CA002", Status: "completed",  StartTime: "2026-06-07 10:30:00", Duration: 60},
		{Sid: "CA003", Status: "no-answer",  StartTime: "2026-06-07 11:00:00", Duration: 0},
		{Sid: "CA004", Status: "failed",     StartTime: "2026-06-07 14:00:00", Duration: 0},
		{Sid: "CA005", Status: "busy",       StartTime: "2026-06-07 15:00:00", Duration: 0},
		{Sid: "CA006", Status: "no-answer",  StartTime: "2026-06-07 16:00:00", Duration: 0},
	}

	_, _, summary := ProcessCallRecords("TXN-001", makeJob(), "08047493481", records)

	if summary.Total != 6 {
		t.Errorf("Total = %d, want 6", summary.Total)
	}
	if summary.Connected != 2 {
		t.Errorf("Connected (completed) = %d, want 2", summary.Connected)
	}
	if summary.NoAnswer != 2 {
		t.Errorf("NoAnswer (no-answer) = %d, want 2", summary.NoAnswer)
	}
	if summary.Failed != 1 {
		t.Errorf("Failed = %d, want 1", summary.Failed)
	}
	if summary.Busy != 1 {
		t.Errorf("Busy = %d, want 1", summary.Busy)
	}
}

// TestProcessCallRecords_AnswerRate verifies that AnswerRate = Connected/Total * 100.
func TestProcessCallRecords_AnswerRate(t *testing.T) {
	records := []exotel.CallRecord{
		{Sid: "CA001", Status: "completed", StartTime: "2026-06-07 09:00:00", Duration: 90},
		{Sid: "CA002", Status: "completed", StartTime: "2026-06-07 09:10:00", Duration: 60},
		{Sid: "CA003", Status: "completed", StartTime: "2026-06-07 09:20:00", Duration: 45},
		{Sid: "CA004", Status: "no-answer", StartTime: "2026-06-07 09:30:00", Duration: 0},
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
