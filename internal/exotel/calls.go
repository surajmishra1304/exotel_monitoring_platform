package exotel

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// FetchCallsResult holds the aggregated result across all pages.
type FetchCallsResult struct {
	HTTPStatus   int
	LatencyMs    int64
	Response     *CallsResponse
	RawBody      string // raw body of the first (or last failing) response
	PagesFetched int
}

// FetchCalls fetches all call records for a specific VN (exophoneNumber) in the
// given time window, following cursor pagination until all pages are consumed.
//
// Exotel Bulk Calls API params used (per docs):
//   - DateCreated  : gte:<from>;lte:<to>   (correct range format)
//   - PhoneNumber  : VN number             (server-side VN filter)
//   - PageSize     : 100                   (max allowed)
//   - SortBy       : DateCreated:asc       (oldest first for consistent ordering)
//
// Per Exotel docs: max 1-month range per request; up to 6 months of historical data.
func FetchCalls(sid, subdomain, apiKey, apiToken, exophoneNumber string, from, to time.Time) (*FetchCallsResult, error) {
	endpoint := fmt.Sprintf("%s/v1/Accounts/%s/Calls.json", BuildBaseURL(subdomain), sid)

	dateRange := fmt.Sprintf("gte:%s;lte:%s",
		from.Format("2006-01-02 15:04:05"),
		to.Format("2006-01-02 15:04:05"),
	)

	params := map[string]string{
		"DateCreated": dateRange,
		"PageSize":    "100",
		"SortBy":      "DateCreated:asc",
		"details":     "true",
	}
	if exophoneNumber != "" {
		params["PhoneNumber"] = exophoneNumber
	}

	var (
		allRecords   []CallRecord
		firstRawBody string
		lastHTTP     int
		totalLatency int64
		pages        int
		afterCursor  string
	)

	for {
		if afterCursor != "" {
			params["After"] = afterCursor
		}

		pageStart := time.Now()
		resp, err := Client.R().
			SetBasicAuth(apiKey, apiToken).
			SetQueryParams(params).
			Get(endpoint)
		totalLatency += time.Since(pageStart).Milliseconds()
		pages++

		if err != nil {
			return &FetchCallsResult{
				HTTPStatus:   0,
				LatencyMs:    totalLatency,
				RawBody:      err.Error(),
				PagesFetched: pages,
			}, fmt.Errorf("calls request failed: %w", err)
		}

		lastHTTP = resp.StatusCode()
		rawBody := string(resp.Body())
		if pages == 1 {
			firstRawBody = rawBody
		}

		if !resp.IsSuccess() {
			return &FetchCallsResult{
				HTTPStatus:   lastHTTP,
				LatencyMs:    totalLatency,
				RawBody:      rawBody,
				PagesFetched: pages,
			}, nil
		}

		var page CallsResponse
		if err := json.Unmarshal(resp.Body(), &page); err != nil {
			return &FetchCallsResult{
				HTTPStatus:   lastHTTP,
				LatencyMs:    totalLatency,
				RawBody:      rawBody,
				PagesFetched: pages,
			}, fmt.Errorf("failed to parse calls response: %w", err)
		}

		allRecords = append(allRecords, page.Result...)

		// Stop when there is no next page cursor.
		afterCursor = extractAfterCursor(page.Metadata.NextPageUri)
		if afterCursor == "" {
			break
		}
		// Safety: cap at 100 pages (10 000 calls per window) to avoid runaway loops.
		if pages >= 100 {
			break
		}
	}

	return &FetchCallsResult{
		HTTPStatus:   lastHTTP,
		LatencyMs:    totalLatency,
		RawBody:      firstRawBody,
		PagesFetched: pages,
		Response: &CallsResponse{
			Result: allRecords,
		},
	}, nil
}

// extractAfterCursor parses the After query param from a NextPageUri like:
// /v1/Accounts/sid/Calls?...&After=<token>
func extractAfterCursor(nextPageUri string) string {
	if nextPageUri == "" {
		return ""
	}
	// NextPageUri is a relative path; parse just the query string part.
	idx := strings.Index(nextPageUri, "?")
	if idx < 0 {
		return ""
	}
	q, err := url.ParseQuery(nextPageUri[idx+1:])
	if err != nil {
		return ""
	}
	return q.Get("After")
}
