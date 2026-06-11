package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"exotel-monitoring-platform/internal/database"
	"exotel-monitoring-platform/internal/logger"
)

// Key prefix constants — all cache keys must use these.
const (
	KeyAccount       = "account:%d"
	KeyExophone      = "exophone:%d"
	KeyMetrics       = "metrics:%d"
	KeyTransaction   = "transaction:%s"
	KeyDashboardSum  = "dashboard:summary"
	KeyAlertsActive  = "alerts:active"
	KeyHealthSnap    = "health_snap:%d"

	TTLAccount      = 1 * time.Hour
	TTLExophone     = 24 * time.Hour
	TTLMetrics      = 15 * time.Minute
	TTLTransaction  = 5 * time.Minute
	TTLDashboard    = 15 * time.Minute
	TTLAlerts       = 2 * time.Minute
	TTLHealthSnap   = 5 * time.Minute
)

// Set stores a value as JSON with a TTL.
func Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache marshal error: %w", err)
	}
	return database.Redis.Set(ctx, key, data, ttl).Err()
}

// Get retrieves a JSON value and unmarshals it into dest.
// Returns (false, nil) on cache miss.
func Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	val, err := database.Redis.Get(ctx, key).Result()
	if err != nil {
		// redis.Nil means key not found — not a real error.
		if err.Error() == "redis: nil" {
			return false, nil
		}
		return false, fmt.Errorf("cache get error: %w", err)
	}

	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return false, fmt.Errorf("cache unmarshal error: %w", err)
	}
	return true, nil
}

// Delete removes a key from the cache.
func Delete(ctx context.Context, key string) error {
	return database.Redis.Del(ctx, key).Err()
}

// Exists checks whether a key exists in cache.
func Exists(ctx context.Context, key string) bool {
	n, err := database.Redis.Exists(ctx, key).Result()
	if err != nil {
		logger.Log.Sugar().Warnf("cache exists check error for key %s: %v", key, err)
		return false
	}
	return n > 0
}

// SetString stores a raw string value with a TTL.
func SetString(ctx context.Context, key string, value string, ttl time.Duration) error {
	return database.Redis.Set(ctx, key, value, ttl).Err()
}

// GetString retrieves a raw string value.
func GetString(ctx context.Context, key string) (string, bool, error) {
	val, err := database.Redis.Get(ctx, key).Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			return "", false, nil
		}
		return "", false, err
	}
	return val, true, nil
}

// AccountKey returns the Redis key for an account.
func AccountKey(id uint64) string { return fmt.Sprintf(KeyAccount, id) }

// ExophoneKey returns the Redis key for an exophone.
func ExophoneKey(id uint64) string { return fmt.Sprintf(KeyExophone, id) }

// MetricsKey returns the Redis key for exophone metrics.
func MetricsKey(exophoneID uint64) string { return fmt.Sprintf(KeyMetrics, exophoneID) }

// TransactionKey returns the Redis key for a transaction.
func TransactionKey(txnID string) string { return fmt.Sprintf(KeyTransaction, txnID) }

// HealthSnapKey returns the Redis key for an exophone health snapshot.
func HealthSnapKey(exophoneID uint64) string { return fmt.Sprintf(KeyHealthSnap, exophoneID) }
