package utils

import (
	"context"
	"math"
	"math/rand"
	"time"

	"exotel-monitoring-platform/internal/logger"
	"go.uber.org/zap"
)

// RetryConfig describes exponential back-off parameters.
type RetryConfig struct {
	MaxRetries  int
	BaseDelayMs int
	MaxDelayMs  int
}

// DefaultRetryConfig is a safe default for vendor API retries.
var DefaultRetryConfig = RetryConfig{
	MaxRetries:  3,
	BaseDelayMs: 1000,
	MaxDelayMs:  30000,
}

// IsRetryable returns true for HTTP status codes that warrant a retry.
func IsRetryable(httpStatus int) bool {
	switch httpStatus {
	case 429, 500, 502, 503, 504:
		return true
	}
	return false
}

// BackoffDuration calculates exponential back-off with full jitter.
//   delay = random(0, min(maxDelay, baseDelay * 2^attempt))
func BackoffDuration(attempt int, cfg RetryConfig) time.Duration {
	exp := math.Pow(2, float64(attempt))
	maxMs := float64(cfg.BaseDelayMs) * exp
	if maxMs > float64(cfg.MaxDelayMs) {
		maxMs = float64(cfg.MaxDelayMs)
	}
	jitter := rand.Float64() * maxMs
	return time.Duration(jitter) * time.Millisecond
}

// DoWithRetry executes fn up to cfg.MaxRetries+1 times (first attempt + retries).
// fn must return (bool shouldRetry, error).
func DoWithRetry(ctx context.Context, name string, cfg RetryConfig, fn func() (bool, error)) error {
	var lastErr error
	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := BackoffDuration(attempt, cfg)
			logger.Log.Info("retry attempt",
				zap.String("operation", name),
				zap.Int("attempt", attempt),
				zap.Duration("delay", delay),
			)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		shouldRetry, err := fn()
		if err == nil {
			return nil
		}
		lastErr = err
		if !shouldRetry {
			break
		}
	}
	return lastErr
}
