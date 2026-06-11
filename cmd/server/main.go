package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"exotel-monitoring-platform/internal/config"
	"exotel-monitoring-platform/internal/database"
	"exotel-monitoring-platform/internal/exotel"
	"exotel-monitoring-platform/internal/feature"
	"exotel-monitoring-platform/internal/logger"
	"exotel-monitoring-platform/internal/scheduler"
	"exotel-monitoring-platform/internal/server"
	"exotel-monitoring-platform/internal/workers"

	"go.uber.org/zap"
)

func main() {
	// ── Bootstrap ──────────────────────────────────────────────────────
	logger.Init()
	config.Load()

	logger.Log.Info("starting exotel monitoring platform")

	// ── Infrastructure ──────────────────────────────────────────────────
	database.ConnectMySQL()
	database.ConnectRedis()

	// ── Feature flags ───────────────────────────────────────────────────
	feature.Init()

	// ── Exotel HTTP client ──────────────────────────────────────────────
	exotel.InitClient()

	// ── Worker pool ─────────────────────────────────────────────────────
	workers.InitPool()

	// ── Scheduler ───────────────────────────────────────────────────────
	scheduler.Start()

	// ── HTTP server ─────────────────────────────────────────────────────
	srv := server.New()

	// Start server in background
	go func() {
		logger.Log.Info("HTTP server listening", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("HTTP server error", zap.Error(err))
		}
	}()

	// ── Graceful shutdown ────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("shutdown signal received — draining...")

	// Give in-flight HTTP requests 30 s to complete.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Error("HTTP server forced shutdown", zap.Error(err))
	}

	// Stop the scheduler (waits for in-flight cron jobs to finish).
	scheduler.Stop()

	// Wait for all pool workers to drain.
	workers.DefaultPool.Wait()

	logger.Log.Info("exotel monitoring platform stopped cleanly")
}
