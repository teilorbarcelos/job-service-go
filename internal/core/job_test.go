package core

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeJob struct {
	name        string
	schedule    string
	description string
	enabled     bool
	run         func(ctx context.Context) error
}

func (f *fakeJob) Name() string        { return f.name }
func (f *fakeJob) Schedule() string     { return f.schedule }
func (f *fakeJob) Description() string { return f.description }
func (f *fakeJob) Enabled() bool        { return f.enabled }
func (f *fakeJob) Run(ctx context.Context, _ JobContext) error {
	return f.run(ctx)
}

func silentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestExecuteJob_Success(t *testing.T) {
	j := &fakeJob{name: "ok", run: func(ctx context.Context) error { return nil }}
	jc := JobContext{Logger: silentLogger()}
	res := ExecuteJob(j, jc, context.Background(), time.Second)
	assert.Equal(t, "ok", res.Job)
	assert.Equal(t, StatusSuccess, res.Status)
	assert.GreaterOrEqual(t, res.DurationMs, int64(0))
	assert.Empty(t, res.Error)
}

func TestExecuteJob_GenericError_ReturnsFailed(t *testing.T) {
	j := &fakeJob{name: "boom", run: func(ctx context.Context) error { return errors.New("kaboom") }}
	res := ExecuteJob(j, JobContext{Logger: silentLogger()}, context.Background(), time.Second)
	assert.Equal(t, StatusFailed, res.Status)
	assert.Equal(t, "kaboom", res.Error)
}

func TestExecuteJob_Timeout_ReturnsCancelled(t *testing.T) {
	j := &fakeJob{
		name: "slow",
		run: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}
	res := ExecuteJob(j, JobContext{Logger: silentLogger()}, context.Background(), 50*time.Millisecond)
	assert.Equal(t, StatusCancelled, res.Status)
}

func TestExecuteJob_ParentCancel_ReturnsCancelled(t *testing.T) {
	parentCtx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled
	j := &fakeJob{
		name: "x",
		run: func(ctx context.Context) error { return ctx.Err() },
	}
	res := ExecuteJob(j, JobContext{Logger: silentLogger()}, parentCtx, time.Second)
	assert.Equal(t, StatusCancelled, res.Status)
}

func TestExecuteJob_OtherContextError_StillFails(t *testing.T) {
	j := &fakeJob{
		name: "y",
		run: func(ctx context.Context) error {
			return context.Canceled
		},
	}
	res := ExecuteJob(j, JobContext{Logger: silentLogger()}, context.Background(), time.Second)
	assert.Equal(t, StatusFailed, res.Status)
}

func TestExecuteJob_ZeroTimeout(t *testing.T) {
	j := &fakeJob{
		name: "z",
		run: func(ctx context.Context) error { return ctx.Err() },
	}
	res := ExecuteJob(j, JobContext{Logger: silentLogger()}, context.Background(), 0)
	require.NotNil(t, res)
	assert.Equal(t, StatusCancelled, res.Status)
}
