package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

// Init initialises the global zap logger.
// Uses JSON production encoder in production, coloured console encoder otherwise.
func Init() {
	env := os.Getenv("APP_ENV")

	var cfg zap.Config
	if env == "production" {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	var err error
	Log, err = cfg.Build(zap.AddCallerSkip(0))
	if err != nil {
		panic(err)
	}

	zap.ReplaceGlobals(Log)
}

// Named returns a logger with a given name (useful for component-level logging).
func Named(name string) *zap.Logger {
	return Log.Named(name)
}
