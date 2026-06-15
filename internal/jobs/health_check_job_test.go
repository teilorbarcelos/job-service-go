package jobs

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"job-service-go/internal/core"
	"job-service-go/internal/infra/health"
	"job-service-go/internal/shared/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubChecker struct {
	pg   health.HealthCheckResult
	rd   health.HealthCheckResult
	rb   health.HealthCheckResult
	mu   sync.Mutex
}

func (s *stubChecker) CheckPostgres(_ context.Context) health.HealthCheckResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pg
}
func (s *stubChecker) CheckRedis(_ context.Context) health.HealthCheckResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rd
}
func (s *stubChecker) CheckRabbit(_ context.Context) health.HealthCheckResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rb
}

func newSettings() *config.AppSettings {
	return &config.AppSettings{HealthCheckCron: "0 9 * * *", HealthCheckEnabled: true}
}

func TestHealthCheckJob_Metadata(t *testing.T) {
	j := NewHealthCheckJob(&stubChecker{}, newSettings())
	assert.Equal(t, "health-check", j.Name())
	assert.Equal(t, "0 9 * * *", j.Schedule())
	assert.Contains(t, j.Description(), "PostgreSQL")
	assert.Contains(t, j.Description(), "Redis")
	assert.Contains(t, j.Description(), "RabbitMQ")
	assert.True(t, j.Enabled())
}

func TestHealthCheckJob_DefaultSchedule(t *testing.T) {
	s := &config.AppSettings{HealthCheckEnabled: true}
	j := NewHealthCheckJob(&stubChecker{}, s)
	assert.Equal(t, "*/1 * * * *", j.Schedule())
}

func TestHealthCheckJob_SetEnabled(t *testing.T) {
	j := NewHealthCheckJob(&stubChecker{}, newSettings())
	j.SetEnabled(false)
	assert.False(t, j.Enabled())
}

func TestHealthCheckJob_HealthyAllUp(t *testing.T) {
	stub := &stubChecker{
		pg: health.HealthCheckResult{Status: health.StatusUp, LatencyMs: 5},
		rd: health.HealthCheckResult{Status: health.StatusUp, LatencyMs: 1},
		rb: health.HealthCheckResult{Status: health.StatusUp},
	}
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = old }()

	j := NewHealthCheckJob(stub, newSettings())
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(io.MultiWriter(w, &buf), nil))
	require.NoError(t, j.Run(context.Background(), core.JobContext{Logger: logger}))
	w.Close()
	out, _ := io.ReadAll(r)
	assert.Contains(t, string(out), "postgres=up")
	assert.Contains(t, string(out), "redis=up")
	assert.Contains(t, string(out), "rabbitmq=up")
}

func TestHealthCheckJob_DegradedWhenPgDown(t *testing.T) {
	stub := &stubChecker{
		pg: health.HealthCheckResult{Status: health.StatusDown, Error: "conn refused"},
		rd: health.HealthCheckResult{Status: health.StatusUp},
		rb: health.HealthCheckResult{Status: health.StatusUp},
	}
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = old }()

	j := NewHealthCheckJob(stub, newSettings())
	require.NoError(t, j.Run(context.Background(), core.JobContext{Logger: slog.New(slog.NewTextHandler(w, nil))}))
	w.Close()
	out, _ := io.ReadAll(r)
	assert.Contains(t, string(out), "postgres=down")
}

func TestHealthCheckJob_RabbitDisabled(t *testing.T) {
	stub := &stubChecker{
		pg: health.HealthCheckResult{Status: health.StatusUp},
		rd: health.HealthCheckResult{Status: health.StatusUp},
		rb: health.HealthCheckResult{Status: health.StatusDisabled},
	}
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = old }()

	j := NewHealthCheckJob(stub, newSettings())
	require.NoError(t, j.Run(context.Background(), core.JobContext{Logger: slog.New(slog.NewTextHandler(w, nil))}))
	w.Close()
	out, _ := io.ReadAll(r)
	assert.Contains(t, string(out), "rabbitmq=disabled")
}

func TestHealthCheckJob_HandlesAllThree(t *testing.T) {
	stub := &stubChecker{
		pg: health.HealthCheckResult{Status: health.StatusUp, LatencyMs: 1},
		rd: health.HealthCheckResult{Status: health.StatusDown, Error: "x"},
		rb: health.HealthCheckResult{Status: health.StatusUp},
	}
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = old }()

	j := NewHealthCheckJob(stub, newSettings())
	require.NoError(t, j.Run(context.Background(), core.JobContext{Logger: slog.New(slog.NewTextHandler(w, nil))}))
	w.Close()
	out, _ := io.ReadAll(r)
	assert.Contains(t, string(out), "postgres=up")
	assert.Contains(t, string(out), "redis=down")
	assert.Contains(t, string(out), "rabbitmq=up")
}

func TestRegisterJobs_EnabledByDefault(t *testing.T) {
	s := newSettings()
	jobs := RegisterJobs(&stubChecker{}, s)
	require.Len(t, jobs, 1)
	assert.Equal(t, "health-check", jobs[0].Name())
	assert.True(t, jobs[0].Enabled())
}

func TestRegisterJobs_DisabledWhenSettingFalse(t *testing.T) {
	s := &config.AppSettings{HealthCheckEnabled: false}
	jobs := RegisterJobs(&stubChecker{}, s)
	assert.False(t, jobs[0].Enabled())
}

func TestHealthCheckJob_PassesContextToChecker(t *testing.T) {
	called := make(chan struct{}, 3)
	stub := &blockingChecker{called: called}
	j := NewHealthCheckJob(stub, newSettings())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, j.Run(ctx, core.JobContext{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}))
	assert.Len(t, called, 3)
}

type blockingChecker struct{ called chan struct{} }

func (b *blockingChecker) CheckPostgres(_ context.Context) health.HealthCheckResult {
	b.called <- struct{}{}
	return health.HealthCheckResult{Status: health.StatusUp}
}
func (b *blockingChecker) CheckRedis(_ context.Context) health.HealthCheckResult {
	b.called <- struct{}{}
	return health.HealthCheckResult{Status: health.StatusUp}
}
func (b *blockingChecker) CheckRabbit(_ context.Context) health.HealthCheckResult {
	b.called <- struct{}{}
	return health.HealthCheckResult{Status: health.StatusUp}
}
