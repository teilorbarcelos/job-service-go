package jobs

import (
	"job-service-go/internal/core"
	"job-service-go/internal/infra/health"
	"job-service-go/internal/shared/config"
)

func RegisterJobs(checker health.IHealthChecker, settings *config.AppSettings) []core.BaseJob {
	healthCheck := NewHealthCheckJob(checker, settings)
	return []core.BaseJob{healthCheck}
}
