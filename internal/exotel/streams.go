package exotel

import (
	"encoding/json"
	"fmt"
	"time"
)

// FetchStreamsResult holds the raw HTTP data alongside the parsed struct.
type FetchStreamsResult struct {
	HTTPStatus int
	LatencyMs  int64
	Response   *ActiveStreamsResponse
	RawBody    string
}

// FetchActiveStreams calls the Exotel Active Streams API for a given account SID.
func FetchActiveStreams(sid, subdomain, apiKey, apiToken string) (*FetchStreamsResult, error) {
	url := fmt.Sprintf("%s/v1/Accounts/%s/ActiveStreams", BuildBaseURL(subdomain), sid)

	start := time.Now()
	resp, err := Client.R().
		SetBasicAuth(apiKey, apiToken).
		Get(url)

	latency := time.Since(start).Milliseconds()

	if err != nil {
		return &FetchStreamsResult{
			HTTPStatus: 0,
			LatencyMs:  latency,
			RawBody:    err.Error(),
		}, fmt.Errorf("streams request failed: %w", err)
	}

	result := &FetchStreamsResult{
		HTTPStatus: resp.StatusCode(),
		LatencyMs:  latency,
		RawBody:    string(resp.Body()),
	}

	if resp.IsSuccess() {
		var sr ActiveStreamsResponse
		if err := json.Unmarshal(resp.Body(), &sr); err == nil {
			result.Response = &sr
		}
	}

	return result, nil
}
