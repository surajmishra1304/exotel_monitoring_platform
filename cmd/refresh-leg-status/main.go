// refresh-leg-status fetches today's call records (with details=true) for every
// active exophone and upserts leg1_status, leg2_status, conversation_duration into
// call_logs, then recomputes the call_metrics_snapshot for that date.
//
// Usage:
//
//	go run ./cmd/refresh-leg-status/                 # today
//	go run ./cmd/refresh-leg-status/ 2026-06-10      # specific date
package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"exotel-monitoring-platform/internal/config"
	"exotel-monitoring-platform/internal/database"
	"exotel-monitoring-platform/internal/exotel"
	"exotel-monitoring-platform/internal/logger"
	"exotel-monitoring-platform/internal/metrics"
	"exotel-monitoring-platform/internal/models"
	"exotel-monitoring-platform/internal/repository"
	"exotel-monitoring-platform/internal/utils"

	"gorm.io/gorm/clause"
)

func main() {
	config.Load()
	logger.Init()
	database.ConnectMySQL()
	exotel.InitClient()

	// Resolve target date.
	dateStr := time.Now().Format("2006-01-02")
	if len(os.Args) > 1 {
		dateStr = os.Args[1]
	}
	from, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
	if err != nil {
		fmt.Printf("ERROR: invalid date %q (want YYYY-MM-DD): %v\n", dateStr, err)
		os.Exit(1)
	}
	to := from.Add(24*time.Hour - time.Second)

	fmt.Printf("refresh-leg-status: date=%s window=%s → %s\n\n",
		dateStr,
		from.Format("2006-01-02 15:04:05"),
		to.Format("2006-01-02 15:04:05"),
	)

	exophones, err := repository.GetMonitoredExophones()
	if err != nil {
		fmt.Printf("ERROR: cannot load exophones: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Found %d monitored exophones\n\n", len(exophones))

	type cachedCreds struct {
		account  *models.Account
		apiKey   string
		apiToken string
	}
	credCache := map[uint64]*cachedCreds{}

	ok, failed, skipped := 0, 0, 0

	for _, e := range exophones {
		// Load + cache decrypted credentials per account.
		creds, hit := credCache[e.AccountID]
		if !hit {
			acc, loadErr := repository.GetAccountByID(e.AccountID)
			if loadErr != nil {
				fmt.Printf("  SKIP exophone=%-4d (%s): account %d not found: %v\n",
					e.ID, e.ExophoneNumber, e.AccountID, loadErr)
				skipped++
				continue
			}
			sk := config.App.Crypto.SecretKey
			apiKey, _ := utils.Decrypt(acc.APIKey, sk)
			if apiKey == "" {
				apiKey = acc.APIKey // dev fallback: stored unencrypted
			}
			apiToken, _ := utils.Decrypt(acc.APIToken, sk)
			if apiToken == "" {
				apiToken = acc.APIToken
			}
			creds = &cachedCreds{acc, apiKey, apiToken}
			credCache[e.AccountID] = creds
		}

		// Fetch all pages for today without the 20-page cap.
		records, pages, fetchErr := fetchAllPages(creds.account, creds.apiKey, creds.apiToken, e.ExophoneNumber, from, to)
		if fetchErr != nil {
			fmt.Printf("  FAIL exophone=%-4d (%s): API error (page %d): %v\n",
				e.ID, e.ExophoneNumber, pages, fetchErr)
			failed++
			continue
		}
		if len(records) == 0 {
			fmt.Printf("  SKIP exophone=%-4d (%s): 0 calls for %s\n",
				e.ID, e.ExophoneNumber, dateStr)
			skipped++
			continue
		}

		// Upsert: insert new rows + update leg status on duplicates.
		saved, upsertErr := upsertCallLogs(e, records, dateStr)
		if upsertErr != nil {
			fmt.Printf("  FAIL exophone=%-4d (%s): DB upsert error: %v\n",
				e.ID, e.ExophoneNumber, upsertErr)
			failed++
			continue
		}

		// Recompute the daily snapshot from the freshly upserted data.
		if snapErr := repository.RecomputeCallMetricsForDate(e.AccountID, e.ID, dateStr); snapErr != nil {
			fmt.Printf("  WARN exophone=%-4d (%s): snapshot recompute failed: %v\n",
				e.ID, e.ExophoneNumber, snapErr)
		}

		fmt.Printf("  OK   exophone=%-4d (%s): api_records=%d pages=%d db_rows=%d\n",
			e.ID, e.ExophoneNumber, len(records), pages, saved)
		ok++
	}

	fmt.Printf("\n── Summary ──────────────────────────────\n")
	fmt.Printf("  OK:      %d\n", ok)
	fmt.Printf("  Failed:  %d\n", failed)
	fmt.Printf("  Skipped: %d\n", skipped)
	fmt.Println("─────────────────────────────────────────")
}

// fetchAllPages calls the Exotel Calls API with details=true and follows cursor
// pagination until all records for the window are retrieved (up to 500 pages /
// 50 000 calls per exophone — well above any realistic daily volume).
func fetchAllPages(account *models.Account, apiKey, apiToken, exophoneNumber string, from, to time.Time) ([]exotel.CallRecord, int, error) {
	endpoint := fmt.Sprintf("%s/v1/Accounts/%s/Calls.json",
		exotel.BuildBaseURL(account.Subdomain), account.SID)

	params := map[string]string{
		"DateCreated": fmt.Sprintf("gte:%s;lte:%s",
			from.Format("2006-01-02 15:04:05"),
			to.Format("2006-01-02 15:04:05"),
		),
		"PageSize": "100",
		"SortBy":   "DateCreated:asc",
		"details":  "true",
	}
	if exophoneNumber != "" {
		params["PhoneNumber"] = exophoneNumber
	}

	var all []exotel.CallRecord
	pages := 0
	after := ""

	for {
		if after != "" {
			params["After"] = after
		} else {
			delete(params, "After")
		}

		resp, err := exotel.Client.R().
			SetBasicAuth(apiKey, apiToken).
			SetQueryParams(params).
			Get(endpoint)
		pages++

		if err != nil {
			return all, pages, fmt.Errorf("HTTP error: %w", err)
		}
		if !resp.IsSuccess() {
			return all, pages, fmt.Errorf("HTTP %d: %s", resp.StatusCode(), resp.Body())
		}

		var page exotel.CallsResponse
		if err := json.Unmarshal(resp.Body(), &page); err != nil {
			return all, pages, fmt.Errorf("JSON parse: %w", err)
		}

		all = append(all, page.Result...)

		after = extractAfter(page.Metadata.NextPageUri)
		if after == "" {
			break
		}
		if pages >= 500 {
			fmt.Printf("    [warn] 500-page safety cap reached for %s — stopping\n", exophoneNumber)
			break
		}
	}

	return all, pages, nil
}

// extractAfter parses the After cursor from a NextPageUri like
// /v1/Accounts/sid/Calls?...&After=<token>
func extractAfter(nextPageUri string) string {
	if nextPageUri == "" {
		return ""
	}
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

// upsertCallLogs builds CallLog rows from the API response and does an INSERT …
// ON DUPLICATE KEY UPDATE so that:
//   - new call_sids are inserted with all fields, and
//   - existing rows get leg1_status, leg2_status and conversation_duration refreshed.
func upsertCallLogs(e models.Exophone, records []exotel.CallRecord, dateStr string) (int, error) {
	txnID := "REFRESH-" + dateStr

	logs := make([]models.CallLog, 0, len(records))
	for _, r := range records {
		cl := models.CallLog{
			TransactionID:        txnID,
			AccountID:            e.AccountID,
			ExophoneID:           e.ID,
			CallSID:              r.Sid,
			ParentCallSID:        r.ParentCallSid,
			LegNumber:            metrics.ClassifyLeg(r, e.ExophoneNumber),
			CallFrom:             r.From,
			CallTo:               r.To,
			Status:               r.Status,
			Direction:            r.Direction,
			DurationSec:          r.Duration,
			ConversationDuration: r.Details.ConversationDuration,
			RecordingURL:         r.RecordingUrl,
			Leg1Status:           r.Details.Leg1Status,
			Leg2Status:           r.Details.Leg2Status,
		}
		if r.StartTime != "" {
			if t := parseTime(r.StartTime); !t.IsZero() {
				cl.StartTime = &t
			}
		} else if r.DateCreated != "" {
			if t := parseTime(r.DateCreated); !t.IsZero() {
				cl.StartTime = &t
			}
		}
		if r.EndTime != "" {
			if t := parseTime(r.EndTime); !t.IsZero() {
				cl.EndTime = &t
			}
		}
		logs = append(logs, cl)
	}

	// GORM upsert: on (exophone_id, call_sid) conflict, update only the three
	// status-derived fields — leave all other columns (call_from, status, etc.) as-is.
	res := database.DB.
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "exophone_id"},
				{Name: "call_sid"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"leg1_status",
				"leg2_status",
				"conversation_duration",
				"leg_number",
			}),
		}).
		CreateInBatches(&logs, 200)

	return len(logs), res.Error
}

// parseTime handles the common Exotel timestamp formats.
func parseTime(s string) time.Time {
	for _, f := range []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		"Mon, 02 Jan 2006 15:04:05 -0700",
	} {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
