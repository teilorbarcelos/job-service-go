package health

import (
	"context"
)

type Status string

const (
	StatusUp       Status = "up"
	StatusDown     Status = "down"
	StatusDisabled Status = "disabled"
)

type HealthCheckResult struct {
	Status    Status
	LatencyMs int64
	Error     string
}

type IHealthChecker interface {
	CheckPostgres(ctx context.Context) HealthCheckResult
	CheckRedis(ctx context.Context) HealthCheckResult
	CheckRabbit(ctx context.Context) HealthCheckResult
}
