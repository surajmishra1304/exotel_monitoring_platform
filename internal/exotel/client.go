package exotel

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"exotel-monitoring-platform/internal/config"

	"github.com/go-resty/resty/v2"
)

// insecureTransport skips TLS verification — needed because Exotel's API server
// (api.exotel.com) presents a *.exotel.in certificate that doesn't match the host.
var insecureTransport = &http.Transport{
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
}

var Client *resty.Client

// InitClient creates a shared resty client with sensible defaults.
func InitClient() {
	cfg := config.App.Exotel
	Client = resty.New().
		SetTimeout(time.Duration(cfg.TimeoutSeconds) * time.Second).
		SetRetryCount(0). // retries handled by our own retry layer
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/json").
		SetTransport(insecureTransport)
}

// BuildBaseURL returns the per-account Exotel API base URL.
//
// The subdomain field in the accounts table can hold:
//   - A full hostname: "api.exotel.com" → "https://api.exotel.com"      (preferred)
//   - A short prefix:  "api"            → "https://api.exotel.com"       (legacy)
//   - Empty string    → falls back to global config base_url
//
// Storing the full hostname avoids ambiguity and supports any cluster:
// api.exotel.com (India), sg1.exotel.com (Singapore), us1.exotel.com (US).
func BuildBaseURL(subdomain string) string {
	if subdomain == "" {
		return config.App.Exotel.BaseURL
	}
	// Full hostname already stored (contains a dot) — use it directly.
	if strings.Contains(subdomain, ".") {
		return "https://" + subdomain
	}
	// Legacy short prefix — append .exotel.com.
	return fmt.Sprintf("https://%s.exotel.com", subdomain)
}

// incomingPhoneNumbersResponse is the Exotel API response for listing phone numbers.
type incomingPhoneNumbersResponse struct {
	IncomingPhoneNumbers []struct {
		IncomingPhoneNumber struct {
			PhoneNumber string `json:"PhoneNumber"`
		} `json:"IncomingPhoneNumber"`
	} `json:"IncomingPhoneNumbers"`
	Metadata struct {
		Total       int    `json:"Total"`
		PageSize    int    `json:"PageSize"`
		NextPageUri string `json:"NextPageUri"`
	} `json:"Metadata"`
}

// ListIncomingPhoneNumbers fetches ALL phone numbers for the given Exotel account
// across all pages and returns them as a set (map[number]struct{}) for O(1) lookup.
//
// Exotel uses 0-indexed Page/PageSize pagination. NextPageUri contains Page=N so
// we increment Page until we get an empty page or no NextPageUri.
func ListIncomingPhoneNumbers(sid, subdomain, apiKey, apiToken string) (map[string]struct{}, error) {
	baseURL := fmt.Sprintf("%s/v1/Accounts/%s/IncomingPhoneNumbers.json", BuildBaseURL(subdomain), sid)

	set := make(map[string]struct{})

	for page := 0; page < 50; page++ { // cap at 50 pages = 5000 numbers
		var payload incomingPhoneNumbersResponse
		resp, err := resty.New().
			SetTimeout(15 * time.Second).
			SetTransport(insecureTransport).
			SetBasicAuth(apiKey, apiToken).
			R().
			SetQueryParams(map[string]string{
				"PageSize": "100",
				"Page":     strconv.Itoa(page),
			}).
			SetResult(&payload).
			Get(baseURL)

		if err != nil {
			return nil, fmt.Errorf("exotel API unreachable: %w", err)
		}
		if resp.StatusCode() == 401 {
			return nil, fmt.Errorf("exotel API returned 401: invalid credentials for account %s", sid)
		}
		if resp.StatusCode() != 200 {
			return nil, fmt.Errorf("exotel API returned %d for account %s", resp.StatusCode(), sid)
		}

		for _, item := range payload.IncomingPhoneNumbers {
			n := item.IncomingPhoneNumber.PhoneNumber
			if n != "" {
				set[n] = struct{}{}
			}
		}

		// Empty page or no next page → we have everything.
		if len(payload.IncomingPhoneNumbers) == 0 || payload.Metadata.NextPageUri == "" {
			break
		}
	}
	return set, nil
}

// ValidateExophoneOnAccount checks whether a specific phone number belongs to the account
// by querying the IncomingPhoneNumbers resource for that exact number.
// This is faster than listing all numbers and avoids pagination gaps.
func ValidateExophoneOnAccount(sid, subdomain, apiKey, apiToken, phoneNumber string) (bool, error) {
	// Try both with and without leading zero (Exotel may index either format).
	for _, candidate := range dedupeFormats(phoneNumber) {
		url := fmt.Sprintf("%s/v1/Accounts/%s/IncomingPhoneNumbers/%s.json",
			BuildBaseURL(subdomain), sid, candidate)

		resp, err := resty.New().
			SetTimeout(10 * time.Second).
			SetTransport(insecureTransport).
			SetBasicAuth(apiKey, apiToken).
			R().
			Get(url)

		if err != nil {
			continue
		}
		if resp.StatusCode() == 200 {
			return true, nil
		}
	}
	return false, nil
}

// dedupeFormats returns a slice of format variants to try for a phone number.
func dedupeFormats(number string) []string {
	digits := func(s string) string {
		var b strings.Builder
		for _, r := range s {
			if r >= '0' && r <= '9' {
				b.WriteRune(r)
			}
		}
		return b.String()
	}
	d := digits(number)
	seen := map[string]struct{}{}
	var out []string
	for _, v := range []string{
		number,                // as-is
		d,                     // digits only
		"0" + d[len(d)-10:],  // leading-zero 11-digit
		"+91" + d[len(d)-10:], // E.164
		d[len(d)-10:],         // 10-digit
	} {
		if _, ok := seen[v]; !ok && v != "" {
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

// ExoPhoneConnectivity is the per-direction health detail inside HeartbeatResponse.Data.
// Docs: status is "active" or "major_outage".
type ExoPhoneConnectivity struct {
	Status            string  `json:"status"`             // "active" | "major_outage"
	LastCheckTime     string  `json:"last_check_time"`    // RFC 3339
	AlternateExophone *string `json:"alternate_exophone"` // fallback VN, may be null
}

// ExoPhoneData holds the per-VN connectivity block inside HeartbeatResponse.Data.
type ExoPhoneData struct {
	Connectivity struct {
		IncomingCall ExoPhoneConnectivity `json:"incoming_call"`
		OutgoingCall ExoPhoneConnectivity `json:"outgoing_call"`
	} `json:"connectivity"`
}

// HeartbeatResponse is the Exotel heartbeat webhook payload.
// Exotel POSTs this to our configured endpoint (POST /exotel/heartbeat).
// Docs: https://developer.exotel.com/docs/heartbeat/webhook-format
//
//	{
//	  "timestamp":         "2018-08-22T15:19:23Z",
//	  "status_type":       "WARNING",
//	  "incoming_affected": ["VN_SID_1"],
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
type HeartbeatResponse struct {
	Timestamp        string                    `json:"timestamp"`          // RFC 3339
	StatusType       string                    `json:"status_type"`        // OK | WARNING | CRITICAL | PAYLOAD_TOO_LARGE
	IncomingAffected []string                  `json:"incoming_affected"`  // VN SIDs with incoming issues
	OutgoingAffected []string                  `json:"outgoing_affected"`  // VN SIDs with outgoing issues
	Data             map[string]ExoPhoneData   `json:"data"`               // per-VN connectivity details
}

// IsOK returns true when no exophones are degraded.
func (h *HeartbeatResponse) IsOK() bool {
	return h.StatusType == "OK" || (len(h.IncomingAffected) == 0 && len(h.OutgoingAffected) == 0)
}

// AffectsVN returns (incomingDown, outgoingDown) for a specific exophone number/SID.
func (h *HeartbeatResponse) AffectsVN(vnNumber string) (bool, bool) {
	inc, out := false, false
	for _, sid := range h.IncomingAffected {
		if sid == vnNumber {
			inc = true
			break
		}
	}
	for _, sid := range h.OutgoingAffected {
		if sid == vnNumber {
			out = true
			break
		}
	}
	return inc, out
}

// CallsMetadata is the pagination envelope returned by the Bulk Calls API.
type CallsMetadata struct {
	Total       int    `json:"Total"`
	PageSize    int    `json:"PageSize"`
	NextPageUri string `json:"NextPageUri"`
}

// CallsResponse is the top-level response from the Exotel Bulk Calls API.
// Ref: GET /v1/Accounts/{sid}/Calls
type CallsResponse struct {
	Metadata CallsMetadata `json:"Metadata"`
	Result   []CallRecord  `json:"Calls"`
}

// CallLeg holds per-leg detail returned when details=true is requested.
type CallLeg struct {
	Id             int `json:"Id"`
	OnCallDuration int `json:"OnCallDuration"`
}

// CallDetails holds the Details block returned when details=true is added to the request.
// ConversationDuration is the definitive indicator of whether a real conversation happened:
// 0 = agent never answered; >0 = both parties were bridged for that many seconds.
type CallDetails struct {
	ConversationDuration int    `json:"ConversationDuration"`
	Leg1Status           string `json:"Leg1Status"` // "completed" | "no-answer" | "canceled" etc.
	Leg2Status           string `json:"Leg2Status"`
	Legs                 []struct {
		Leg CallLeg `json:"Leg"`
	} `json:"Legs"`
}

// CallRecord represents a single call returned by the Bulk Calls API.
type CallRecord struct {
	Sid           string      `json:"Sid"`
	ParentCallSid string      `json:"ParentCallSid"` // non-empty for Leg-2 calls bridged from a Leg-1
	From          string      `json:"From"`
	To            string      `json:"To"`
	PhoneNumber   string      `json:"PhoneNumber"`  // the VN that handled this call
	Status        string      `json:"Status"`        // completed|failed|busy|no-answer|canceled
	Direction     string      `json:"Direction"`     // inbound|outbound-dial|outbound-api
	Duration      int         `json:"Duration"`      // total IVR session time in seconds
	Price         float64     `json:"Price"`         // 0 = not billed (no conversation); >0 = billed
	AnsweredBy    string      `json:"AnsweredBy"`    // Human|Machine|NotSure|NA
	StartTime     string      `json:"StartTime"`
	EndTime       string      `json:"EndTime"`
	DateCreated   string      `json:"DateCreated"`   // ISO8601 timestamp; used for hourly distribution
	RecordingUrl  string      `json:"RecordingUrl"`
	Details       CallDetails `json:"Details"`       // populated when details=true; ConversationDuration=0 means no conversation
}

// ActiveStreamsResponse is the response from the Active Streams API.
type ActiveStreamsResponse struct {
	ActiveStreams      int    `json:"active_streams"`
	MaxAllowedStreams  int    `json:"max_allowed_streams"`
	AccountSid        string `json:"account_sid"`
	Status            string `json:"status"`
}
