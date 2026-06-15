package jobs

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"job-service-go/internal/core"
	"job-service-go/internal/infra/health"
	"job-service-go/internal/shared/config"
)

type HealthCheckJob struct {
	checker     health.IHealthChecker
	cron        string
	mu          sync.RWMutex
	enabled     bool
}

func NewHealthCheckJob(checker health.IHealthChecker, settings *config.AppSettings) *HealthCheckJob {
	return &HealthCheckJob{
		checker: checker,
		cron:    settings.HealthCheckCron,
		enabled: settings.HealthCheckEnabled,
	}
}

func (j *HealthCheckJob) Name() string         { return "health-check" }
func (j *HealthCheckJob) Schedule() string      { return j.cron }
func (j *HealthCheckJob) Description() string  { return "Reports connection status with PostgreSQL, Redis and RabbitMQ" }
func (j *HealthCheckJob) Enabled() bool {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.enabled
}

func (j *HealthCheckJob) SetEnabled(v bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.enabled = v
}

func (j *HealthCheckJob) Run(ctx context.Context, jc core.JobContext) error {
	start := time.Now()
	pgCh := make(chan health.HealthCheckResult, 1)
	rdCh := make(chan health.HealthCheckResult, 1)
	rbCh := make(chan health.HealthCheckResult, 1)
	go func() { pgCh <- j.checker.CheckPostgres(ctx) }()
	go func() { rdCh <- j.checker.CheckRedis(ctx) }()
	go func() { rbCh <- j.checker.CheckRabbit(ctx) }()
	pg, rd, rb := <-pgCh, <-rdCh, <-rbCh
	elapsed := time.Since(start).Milliseconds()

	allUp := pg.Status == health.StatusUp && rd.Status == health.StatusUp && rb.Status == health.StatusUp
	jc.Logger.Info("health check completed",
		"elapsed_ms", elapsed,
		"postgres", string(pg.Status),
		"redis", string(rd.Status),
		"rabbitmq", string(rb.Status),
		"healthy", allUp,
	)
	fmt.Fprintf(os.Stdout, "[HealthCheck %s] postgres=%s redis=%s rabbitmq=%s\n",
		time.Now().UTC().Format(time.RFC3339), pg.Status, rd.Status, rb.Status)
	return nil
}
