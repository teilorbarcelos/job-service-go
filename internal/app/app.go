package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"job-service-go/internal/core"
	"job-service-go/internal/infra/health"
	"job-service-go/internal/jobs"
	"job-service-go/internal/shared/config"
)

// DB is the surface used by the app: ping for health + close for shutdown.
type DB interface {
	Ping(ctx context.Context) error
	Close()
}

// Redis is the surface used by the app: ping for health + close for shutdown.
type Redis interface {
	Ping(ctx context.Context) error
	Close() error
}

// Rabbit is the surface used by the app: isOpen for health + close for shutdown.
// May be nil if messaging is disabled.
type Rabbit interface {
	IsOpen() bool
	Close()
}

// Config bundles everything the app needs to run. main() builds this
// with the production providers; tests build it with fakes.
type Config struct {
	Settings    *config.AppSettings
	Logger      *slog.Logger
	DB          DB
	Redis       Redis
	Rabbit      Rabbit // nil when MessagingEnabled=false
	CronAdapter core.CronAdapter
	ShutdownCh  <-chan struct{} // closed → start draining
}

// Run is the app's lifecycle: wire up health + jobs + scheduler, start
// the scheduler, wait for the shutdown signal, then stop everything.
// Returns instead of os.Exit so it's testable.
func Run(cfg Config) error {
	checker := health.NewDefaultHealthChecker(cfg.DB, cfg.Redis, cfg.Rabbit, cfg.Settings)
	jobsList := jobs.RegisterJobs(checker, cfg.Settings)
	cfg.Logger.Info("jobs registered", "count", len(jobsList))

	scheduler := core.NewScheduler(core.SchedulerOptions{
		Jobs:        jobsList,
		CronAdapter: cfg.CronAdapter,
		Timeout:     cfg.Settings.JobExecutionTimeout,
		Logger:      cfg.Logger,
	})

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := scheduler.Start(runCtx); err != nil {
		return fmt.Errorf("scheduler start: %w", err)
	}
	defer func() {
		// best-effort stop if the supervisor didn't already stop it
		shutdownCtx, c := context.WithTimeout(context.Background(), cfg.Settings.ShutdownTimeout)
		defer c()
		_ = scheduler.Stop(shutdownCtx)
	}()

	<-cfg.ShutdownCh
	cfg.Logger.Info("shutdown signal received, draining...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Settings.ShutdownTimeout)
	defer shutdownCancel()
	if err := scheduler.Stop(shutdownCtx); err != nil {
		cfg.Logger.Warn("scheduler stop exceeded timeout", "err", err.Error())
	}

	cfg.DB.Close()
	if err := cfg.Redis.Close(); err != nil {
		cfg.Logger.Warn("redis close error", "err", err.Error())
	}
	if cfg.Rabbit != nil {
		cfg.Rabbit.Close()
	}
	cfg.Logger.Info("job service stopped")
	return nil
}

// Suppress unused import warning for time package when not used elsewhere.
var _ = time.Second
