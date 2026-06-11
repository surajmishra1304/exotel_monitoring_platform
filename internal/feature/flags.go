package feature

import (
	"sync/atomic"

	"exotel-monitoring-platform/internal/config"
)

var skipCallLogsWrite atomic.Bool

// Init seeds all runtime flags from config on startup.
func Init() {
	skipCallLogsWrite.Store(config.App.Scheduler.SkipCallLogsWrite)
}

// SkipCallLogsWrite reports whether the optimised no-call_logs path is active.
func SkipCallLogsWrite() bool {
	return skipCallLogsWrite.Load()
}

// SetSkipCallLogsWrite toggles the flag at runtime (resets to config value on restart).
func SetSkipCallLogsWrite(v bool) {
	skipCallLogsWrite.Store(v)
}
