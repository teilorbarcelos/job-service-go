package core

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

type JobStatus string

const (
	StatusSuccess   JobStatus = "success"
	StatusFailed    JobStatus = "failed"
	StatusCancelled JobStatus = "cancelled"
)

type JobContext struct {
	Logger *slog.Logger
}

type JobResult struct {
	Job        string
	Status     JobStatus
	DurationMs int64
	Error      string
}

type BaseJob interface {
	Name() string
	Schedule() string
	Description() string
	Enabled() bool
	Run(ctx context.Context, jc JobContext) error
}

func ExecuteJob(job BaseJob, jc JobContext, parent context.Context, timeout time.Duration) JobResult {
	start := time.Now()
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	err := job.Run(ctx, jc)
	elapsed := time.Since(start).Milliseconds()

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) && parent.Err() == nil {
			jc.Logger.Warn("job timeout", "job", job.Name(), "duration_ms", elapsed, "timeout", timeout)
			return JobResult{Job: job.Name(), Status: StatusCancelled, DurationMs: elapsed}
		}
		if errors.Is(err, context.Canceled) && parent.Err() != nil {
			jc.Logger.Warn("job cancelled", "job", job.Name(), "duration_ms", elapsed)
			return JobResult{Job: job.Name(), Status: StatusCancelled, DurationMs: elapsed}
		}
		jc.Logger.Error("job failed", "job", job.Name(), "err", err.Error(), "duration_ms", elapsed)
		return JobResult{Job: job.Name(), Status: StatusFailed, DurationMs: elapsed, Error: err.Error()}
	}
	jc.Logger.Info("job completed", "job", job.Name(), "duration_ms", elapsed)
	return JobResult{Job: job.Name(), Status: StatusSuccess, DurationMs: elapsed}
}
