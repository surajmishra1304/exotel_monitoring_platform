package cache

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/go-redis/redis/v8"

	"exotel-monitoring-platform/internal/database"
)

const (
	// KeyCallDedup is the Redis SET key for per-(exophone, IST-date) call SID deduplication.
	KeyCallDedup = "call_dedup:%d:%s"

	// TTLCallDedup is 25 h — slightly over a full IST day so the key always survives
	// until midnight IST + 1 h of buffer before auto-expiring.
	TTLCallDedup = 25 * time.Hour
)

// CallDedupKey returns the Redis key for the per-exophone per-IST-day SID dedup SET.
func CallDedupKey(exophoneID uint64, dateIST string) string {
	return fmt.Sprintf(KeyCallDedup, exophoneID, dateIST)
}

// AddCallSids registers sids in today's IST dedup SET for the given exophone and
// reports which ones are genuinely new (first time seen today).
//
// The returned []bool is parallel to sids: true = newly added, false = already present.
//
// All N SAdd commands plus one Expire are sent in a single pipelined round-trip.
//
// Fails open on any Redis error: returns all-true so the caller processes every record
// rather than silently dropping data.
func AddCallSids(ctx context.Context, exophoneID uint64, sids []string) []bool {
	result := make([]bool, len(sids))
	if len(sids) == 0 {
		return result
	}

	ist := time.FixedZone("IST", 5*60*60+30*60)
	today := time.Now().In(ist).Format("2006-01-02")
	key := CallDedupKey(exophoneID, today)

	cmds := make([]*goredis.IntCmd, len(sids))
	_, err := database.Redis.Pipelined(ctx, func(pipe goredis.Pipeliner) error {
		for i, sid := range sids {
			cmds[i] = pipe.SAdd(ctx, key, sid)
		}
		pipe.Expire(ctx, key, TTLCallDedup)
		return nil
	})
	if err != nil {
		// Redis unavailable — fail open: every record is treated as new.
		for i := range result {
			result[i] = true
		}
		return result
	}

	for i, cmd := range cmds {
		result[i] = cmd.Val() == 1
	}
	return result
}
