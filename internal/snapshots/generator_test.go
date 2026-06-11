package snapshots

import (
	"testing"

	"exotel-monitoring-platform/internal/models"
)

// TestHeartbeatToAvailability verifies the mapping from Exotel heartbeat status
// strings to availability percentages used in the health snapshot.
func TestHeartbeatToAvailability(t *testing.T) {
	cases := []struct {
		status string
		want   float64
	}{
		{models.HeartbeatOK,      100.0},
		{models.HeartbeatDegraded, 50.0},
		{models.HeartbeatOutage,    0.0},
		{"UNKNOWN",                 0.0},
		{"",                        0.0},
	}

	for _, tc := range cases {
		t.Run(tc.status, func(t *testing.T) {
			got := heartbeatToAvailability(tc.status)
			if got != tc.want {
				t.Errorf("heartbeatToAvailability(%q) = %.1f, want %.1f", tc.status, got, tc.want)
			}
		})
	}
}
