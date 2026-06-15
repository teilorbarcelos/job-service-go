package health

import (
	"context"
	"time"

	"job-service-go/internal/infra/database"
	"job-service-go/internal/infra/messaging"
	redisinfra "job-service-go/internal/infra/redis"
	"job-service-go/internal/shared/config"
)

type DefaultHealthChecker struct {
	db        *database.PgxProvider
	redis     *redisinfra.RedisProvider
	rabbit    *messaging.RabbitProvider
	settings  *config.AppSettings
}

func NewDefaultHealthChecker(
	db *database.PgxProvider,
	rd *redisinfra.RedisProvider,
	rb *messaging.RabbitProvider,
	settings *config.AppSettings,
) *DefaultHealthChecker {
	return &DefaultHealthChecker{db: db, redis: rd, rabbit: rb, settings: settings}
}

func (c *DefaultHealthChecker) CheckPostgres(ctx context.Context) HealthCheckResult {
	start := time.Now()
	pingCtx, cancel := context.WithTimeout(ctx, c.settings.DatabaseCommandTimeout)
	defer cancel()
	if err := c.db.Ping(pingCtx); err != nil {
		elapsed := time.Since(start).Milliseconds()
		return HealthCheckResult{Status: StatusDown, LatencyMs: elapsed, Error: err.Error()}
	}
	return HealthCheckResult{Status: StatusUp, LatencyMs: time.Since(start).Milliseconds()}
}

func (c *DefaultHealthChecker) CheckRedis(ctx context.Context) HealthCheckResult {
	start := time.Now()
	pingCtx, cancel := context.WithTimeout(ctx, c.settings.RedisCommandTimeout)
	defer cancel()
	if err := c.redis.Ping(pingCtx); err != nil {
		elapsed := time.Since(start).Milliseconds()
		return HealthCheckResult{Status: StatusDown, LatencyMs: elapsed, Error: err.Error()}
	}
	return HealthCheckResult{Status: StatusUp, LatencyMs: time.Since(start).Milliseconds()}
}

func (c *DefaultHealthChecker) CheckRabbit(ctx context.Context) HealthCheckResult {
	_ = ctx
	if !c.settings.MessagingEnabled {
		return HealthCheckResult{Status: StatusDisabled}
	}
	if c.rabbit.IsOpen() {
		return HealthCheckResult{Status: StatusUp}
	}
	return HealthCheckResult{Status: StatusDown, Error: "connection closed"}
}
