package exotel

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// FetchHeartbeatResult holds the raw HTTP data alongside the parsed struct.
type FetchHeartbeatResult struct {
	HTTPStatus int
	LatencyMs  int64
	Response   *HeartbeatResponse
	RawBody    string
}

// vnDetailV2 is the response from the v2 incoming-phone-numbers endpoint.
// Docs: GET /v2/accounts/{sid}/incoming-phone-numbers/{exophone_sid}
// This endpoint returns real per-direction connectivity status from Exotel's
// heartbeat infrastructure — same data as the push webhook's "data" field.
type vnDetailV2 struct {
	Connectivity struct {
		IncomingCall ExoPhoneConnectivity `json:"incoming_call"`
		OutgoingCall ExoPhoneConnectivity `json:"outgoing_call"`
	} `json:"connectivity"`
}

// vnDetailV1 is the response from the legacy v1 IncomingPhoneNumbers endpoint.
type vnDetailV1 struct {
	IncomingPhoneNumber struct {
		PhoneNumber   string `json:"PhoneNumber"`
		VoiceUrl      string `json:"VoiceUrl"`
		HealthSetting string `json:"HealthSetting"` // "0" = healthy, non-zero = degraded
	} `json:"IncomingPhoneNumber"`
}

// FetchHeartbeat performs a VN-level health check.
//
// It tries the v2 endpoint first (returns real connectivity status from
// Exotel's heartbeat infrastructure). If v2 returns 404 or an unexpected
// response it falls back to the v1 IncomingPhoneNumbers endpoint.
//
// v2 endpoint (preferred):
//
//	GET /v2/accounts/{sid}/incoming-phone-numbers/{exophone_number}
//	Returns: connectivity.incoming_call.status / outgoing_call.status
//	         values: "active" | "major_outage"
//
// v1 fallback:
//
//	GET /v1/Accounts/{sid}/IncomingPhoneNumbers/{exophone_number}.json
//	Returns: IncomingPhoneNumber.HealthSetting ("0" = OK, else degraded)
func FetchHeartbeat(sid, subdomain, apiKey, apiToken, exophoneNumber string) (*FetchHeartbeatResult, error) {
	base := BuildBaseURL(subdomain)
	vnMode := exophoneNumber != ""

	// ── v2 attempt ────────────────────────────────────────────────────────
	if vnMode {
		v2URL := fmt.Sprintf("%s/v2/accounts/%s/incoming-phone-numbers/%s",
			base, strings.ToLower(sid), exophoneNumber)

		start := time.Now()
		resp, err := Client.R().
			SetBasicAuth(apiKey, apiToken).
			Get(v2URL)
		latency := time.Since(start).Milliseconds()

		if err == nil && resp.StatusCode() == 200 {
			var detail vnDetailV2
			if jsonErr := json.Unmarshal(resp.Body(), &detail); jsonErr == nil {
				hb := &HeartbeatResponse{}
				incStatus := detail.Connectivity.IncomingCall.Status
				outStatus := detail.Connectivity.OutgoingCall.Status

				switch {
				case incStatus == "major_outage" && outStatus == "major_outage":
					hb.StatusType = "CRITICAL"
					hb.IncomingAffected = []string{exophoneNumber}
					hb.OutgoingAffected = []string{exophoneNumber}
				case incStatus == "major_outage":
					hb.StatusType = "WARNING"
					hb.IncomingAffected = []string{exophoneNumber}
				case outStatus == "major_outage":
					hb.StatusType = "WARNING"
					hb.OutgoingAffected = []string{exophoneNumber}
				default:
					hb.StatusType = "OK"
				}

				// Populate Data map so callers get the same shape as a push webhook.
				hb.Data = map[string]ExoPhoneData{
					exophoneNumber: {
						Connectivity: struct {
							IncomingCall ExoPhoneConnectivity `json:"incoming_call"`
							OutgoingCall ExoPhoneConnectivity `json:"outgoing_call"`
						}{
							IncomingCall: detail.Connectivity.IncomingCall,
							OutgoingCall: detail.Connectivity.OutgoingCall,
						},
					},
				}

				return &FetchHeartbeatResult{
					HTTPStatus: resp.StatusCode(),
					LatencyMs:  latency,
					Response:   hb,
					RawBody:    string(resp.Body()),
				}, nil
			}
		}
	}

	// ── v1 fallback ───────────────────────────────────────────────────────
	var v1URL string
	if vnMode {
		v1URL = fmt.Sprintf("%s/v1/Accounts/%s/IncomingPhoneNumbers/%s.json",
			base, sid, exophoneNumber)
	} else {
		v1URL = fmt.Sprintf("%s/v1/Accounts/%s.json", base, sid)
	}

	start := time.Now()
	resp, err := Client.R().
		SetBasicAuth(apiKey, apiToken).
		Get(v1URL)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return &FetchHeartbeatResult{
			HTTPStatus: 0,
			LatencyMs:  latency,
			RawBody:    err.Error(),
		}, fmt.Errorf("heartbeat request failed: %w", err)
	}

	result := &FetchHeartbeatResult{
		HTTPStatus: resp.StatusCode(),
		LatencyMs:  latency,
		RawBody:    string(resp.Body()),
	}

	hb := &HeartbeatResponse{}

	switch {
	case resp.StatusCode() == 200 && vnMode:
		var detail vnDetailV1
		if err := json.Unmarshal(resp.Body(), &detail); err == nil &&
			detail.IncomingPhoneNumber.PhoneNumber != "" {
			hs := detail.IncomingPhoneNumber.HealthSetting
			if hs != "" && hs != "0" {
				hb.StatusType = "WARNING"
				hb.IncomingAffected = []string{exophoneNumber}
			} else {
				hb.StatusType = "OK"
			}
		} else {
			hb.StatusType = "OK"
		}
	case resp.StatusCode() == 200:
		hb.StatusType = "OK"
	case resp.StatusCode() == 404 && vnMode:
		hb.StatusType = "CRITICAL"
		hb.IncomingAffected = []string{exophoneNumber}
		hb.OutgoingAffected = []string{exophoneNumber}
	case resp.StatusCode() == 401 || resp.StatusCode() == 403:
		hb.StatusType = "CRITICAL"
	case resp.StatusCode() >= 500:
		hb.StatusType = "WARNING"
	default:
		hb.StatusType = "WARNING"
	}

	result.Response = hb
	return result, nil
}
