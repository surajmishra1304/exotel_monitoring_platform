package exotel

import (
	"testing"
)

// TestBuildBaseURL verifies the URL construction logic for all subdomain forms.
// Exotel clusters (from docs): api.exotel.com (Singapore), api.in.exotel.com (Mumbai).
func TestBuildBaseURL(t *testing.T) {
	cases := []struct {
		name      string
		subdomain string
		want      string
	}{
		{
			name:      "full hostname stored — Singapore cluster",
			subdomain: "api.exotel.com",
			want:      "https://api.exotel.com",
		},
		{
			name:      "full hostname stored — Mumbai cluster",
			subdomain: "api.in.exotel.com",
			want:      "https://api.in.exotel.com",
		},
		{
			name:      "legacy short prefix — appends .exotel.com",
			subdomain: "api",
			want:      "https://api.exotel.com",
		},
		{
			name:      "empty subdomain — falls back to empty config default (test env)",
			subdomain: "",
			want:      "", // config.App.Exotel.BaseURL is empty in test env
		},
		{
			name:      "old reglobe subdomain — would have been NXDOMAIN before fix",
			subdomain: "reglobe",
			want:      "https://reglobe.exotel.com",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildBaseURL(tc.subdomain)
			if got != tc.want {
				t.Errorf("BuildBaseURL(%q) = %q, want %q", tc.subdomain, got, tc.want)
			}
		})
	}
}

// TestExtractAfterCursor covers the cursor-pagination helper that parses After= from NextPageUri.
func TestExtractAfterCursor(t *testing.T) {
	cases := []struct {
		name string
		uri  string
		want string
	}{
		{
			name: "valid NextPageUri with After cursor",
			uri:  "/v1/Accounts/exotel/Calls.json?PageSize=100&After=CA12345",
			want: "CA12345",
		},
		{
			name: "empty uri returns empty cursor",
			uri:  "",
			want: "",
		},
		{
			name: "uri without After param returns empty",
			uri:  "/v1/Accounts/exotel/Calls.json?PageSize=100",
			want: "",
		},
		{
			name: "After at end of uri",
			uri:  "/v1/Accounts/exotel/Calls.json?After=CAABCDEF",
			want: "CAABCDEF",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractAfterCursor(tc.uri)
			if got != tc.want {
				t.Errorf("extractAfterCursor(%q) = %q, want %q", tc.uri, got, tc.want)
			}
		})
	}
}
