package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"job-service-go/internal/core"
	"job-service-go/internal/infra/database"
	"job-service-go/internal/infra/health"
	"job-service-go/internal/infra/messaging"
	redisinfra "job-service-go/internal/infra/redis"
	"job-service-go/internal/jobs"
	"job-service-go/internal/shared/config"
	"job-service-go/internal/shared/utils"
)

func main() {
	settings, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config load failed:", err)
		os.Exit(1)
	}
	logger := utils.NewLogger(settings.LogLevel, settings.Environment)
	slog.SetDefault(logger)

	logger.Info("starting job service",
		"environment", settings.Environment,
		"job_execution_timeout_s", int(settings.JobExecutionTimeout.Seconds()),
	)

	bootCtx, bootCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer bootCancel()

	db, err := database.NewPgxProvider(bootCtx, settings.DatabaseURL)
	if err != nil {
		logger.Error("postgres connection failed", "err", err.Error())
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("postgres connected")

	rdb, err := redisinfra.NewRedisProvider(bootCtx, redisinfra.Options{
		URL:            settings.RedisURL,
		Host:           settings.RedisHost,
		Port:           settings.RedisPort,
		Password:       settings.RedisPassword,
		DB:             settings.RedisDB,
		CommandTimeout: settings.RedisCommandTimeout,
	})
	if err != nil {
		logger.Error("redis connection failed", "err", err.Error())
		os.Exit(1)
	}
	defer rdb.Close()
	logger.Info("redis connected")

	var rb *messaging.RabbitProvider
	if settings.MessagingEnabled {
		rb, err = messaging.NewRabbitProvider(messaging.Options{
			URL:            settings.RabbitURL,
			User:           settings.RabbitUser,
			Password:       settings.RabbitPassword,
			PublishTimeout: settings.RabbitPublishTimeout,
		})
		if err != nil {
			logger.Error("rabbit provider init failed", "err", err.Error())
			os.Exit(1)
		}
		if err := rb.Connect(); err != nil {
			logger.Error("rabbit connection failed", "err", err.Error())
			os.Exit(1)
		}
		defer rb.Close()
		logger.Info("rabbit connected")
	}

	checker := health.NewDefaultHealthChecker(db, rdb, rb, settings)
	jobsList := jobs.RegisterJobs(checker, settings)
	logger.Info("jobs registered", "count", len(jobsList))

	scheduler := core.NewScheduler(core.SchedulerOptions{
		Jobs:        jobsList,
		CronAdapter: core.NewRobfigAdapter(),
		Timeout:     settings.JobExecutionTimeout,
		Logger:      logger,
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	if err := scheduler.Start(ctx); err != nil {
		logger.Error("scheduler start failed", "err", err.Error())
		os.Exit(1)
	}

	<-ctx.Done()
	logger.Info("shutdown signal received, draining...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), settings.ShutdownTimeout)
	defer cancel()
	if err := scheduler.Stop(shutdownCtx); err != nil {
		logger.Warn("scheduler stop exceeded timeout", "err", err.Error())
	}
	logger.Info("job service stopped")
}
