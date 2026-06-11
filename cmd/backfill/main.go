package main

import (
	"fmt"

	"exotel-monitoring-platform/internal/config"
	"exotel-monitoring-platform/internal/database"
	"exotel-monitoring-platform/internal/logger"
	"exotel-monitoring-platform/internal/repository"
)

// Backfill tool: recomputes call_metrics_snapshot for every exophone+date that has
// call_logs data. Run after any schema change or leg_number backfill with:
//
//	go run ./cmd/backfill/
func main() {
	config.Load()
	logger.Init()
	database.ConnectMySQL()

	type pair struct {
		accountID  uint64
		exophoneID uint64
		date       string
	}

	// All distinct (account_id, exophone_id, date) tuples present in call_logs.
	pairs := []pair{
		{1, 12, "2026-06-09"},
		{1, 22, "2026-06-09"},
		{1, 51, "2026-06-09"},
		{1, 85, "2026-06-09"},
		{1, 12, "2026-06-08"},
		{1, 22, "2026-06-08"},
		{1, 51, "2026-06-08"},
		{1, 22, "2026-06-07"},
		{1, 29, "2026-06-07"},
		{1, 57, "2026-06-07"},
		{1, 12, "2026-06-05"},
		{1, 51, "2026-06-05"},
		{1, 12, "2026-06-04"},
	}

	ok, failed := 0, 0
	for _, p := range pairs {
		fmt.Printf("  exophone=%d  date=%s ... ", p.exophoneID, p.date)
		if err := repository.RecomputeCallMetricsForDate(p.accountID, p.exophoneID, p.date); err != nil {
			fmt.Printf("FAIL: %v\n", err)
			failed++
		} else {
			fmt.Println("OK")
			ok++
		}
	}
	fmt.Printf("\nBackfill complete: %d OK, %d failed\n", ok, failed)
}
