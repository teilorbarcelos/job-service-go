package core

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSchedule struct {
	mu   sync.Mutex
	next time.Time
}

func (f *fakeSchedule) Next(from time.Time) time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.next
}

func (f *fakeSchedule) setNext(t time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.next = t
}

type countingJob struct {
	name    string
	cron    string
	enabled bool
	calls   atomic.Int32
	delay   time.Duration
	runErr  error
}

func (j *countingJob) Name() string        { return j.name }
func (j *countingJob) Schedule() string     { return j.cron }
func (j *countingJob) Description() string { return "" }
func (j *countingJob) Enabled() bool        { return j.enabled }
func (j *countingJob) Run(ctx context.Context, _ JobContext) error {
	j.calls.Add(1)
	if j.delay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(j.delay):
		}
	}
	return j.runErr
}

func newCountingJob(name string, enabled bool) *countingJob {
	return &countingJob{name: name, cron: "* * * * *", enabled: enabled}
}

func newScheduler(t *testing.T, jobs []BaseJob, sched CronSchedule, timeout time.Duration, now func() time.Time) *Scheduler {
	t.Helper()
	opts := SchedulerOptions{
		Jobs:        jobs,
		CronAdapter: &staticAdapter{sched: sched},
		Timeout:     timeout,
		Logger:      silentLogger(),
		Now:         now,
	}
	return NewScheduler(opts)
}

type staticAdapter struct {
	sched CronSchedule
}

func (s *staticAdapter) Parse(expr string) (CronSchedule, error) {
	return s.sched, nil
}

func TestScheduler_Start_NoJobs(t *testing.T) {
	s := newScheduler(t, nil, &fakeSchedule{}, time.Second, nil)
	require.NoError(t, s.Start(context.Background()))
	require.NoError(t, s.Stop(context.Background()))
}

func TestScheduler_Start_DuplicateNames_Errors(t *testing.T) {
	s := newScheduler(t,
		[]BaseJob{newCountingJob("a", true), newCountingJob("a", true)},
		&fakeSchedule{}, time.Second, nil)
	err := s.Start(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate job name")
}

func TestScheduler_Start_InvalidCron_Errors(t *testing.T) {
	bad := &failingAdapter{err: errors.New("bad cron")}
	s := NewScheduler(SchedulerOptions{
		Jobs:        []BaseJob{newCountingJob("a", true)},
		CronAdapter: bad,
		Timeout:     time.Second,
		Logger:      silentLogger(),
	})
	err := s.Start(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cron expressions")
}

type failingAdapter struct{ err error }

func (f *failingAdapter) Parse(expr string) (CronSchedule, error) {
	return nil, f.err
}

func TestScheduler_Start_Twice_Errors(t *testing.T) {
	s := newScheduler(t, nil, &fakeSchedule{}, time.Second, nil)
	require.NoError(t, s.Start(context.Background()))
	err := s.Start(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already started")
}

func TestScheduler_Start_AfterStop_Errors(t *testing.T) {
	s := newScheduler(t, nil, &fakeSchedule{}, time.Second, nil)
	require.NoError(t, s.Start(context.Background()))
	require.NoError(t, s.Stop(context.Background()))
	err := s.Start(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "has been stopped")
}

func TestScheduler_DisabledJobs_NotScheduled(t *testing.T) {
	job := newCountingJob("a", false)
	sched := &fakeSchedule{}
	sched.setNext(time.Now().Add(10 * time.Millisecond))
	s := newScheduler(t, []BaseJob{job}, sched, time.Second, nil)
	require.NoError(t, s.Start(context.Background()))
	time.Sleep(100 * time.Millisecond)
	require.NoError(t, s.Stop(context.Background()))
	assert.Equal(t, int32(0), job.calls.Load())
}

func TestScheduler_Job_Runs_AtScheduledTime(t *testing.T) {
	job := newCountingJob("a", true)
	sched := &fakeSchedule{}
	sched.setNext(time.Now().Add(20 * time.Millisecond))
	s := newScheduler(t, []BaseJob{job}, sched, time.Second, nil)
	require.NoError(t, s.Start(context.Background()))
	require.Eventually(t, func() bool {
		return job.calls.Load() >= 1
	}, time.Second, 10*time.Millisecond, "job should run at least once")
	require.NoError(t, s.Stop(context.Background()))
}

func TestScheduler_JobTimeout_Cancels(t *testing.T) {
	job := &countingJob{name: "slow", cron: "* * * * *", enabled: true, delay: 500 * time.Millisecond}
	sched := &fakeSchedule{}
	sched.setNext(time.Now().Add(10 * time.Millisecond))
	s := newScheduler(t, []BaseJob{job}, sched, 50*time.Millisecond, nil)
	require.NoError(t, s.Start(context.Background()))
	time.Sleep(200 * time.Millisecond)
	require.NoError(t, s.Stop(context.Background()))
	assert.GreaterOrEqual(t, job.calls.Load(), int32(1))
}

func TestScheduler_Stop_WithoutStart(t *testing.T) {
	s := newScheduler(t, nil, &fakeSchedule{}, time.Second, nil)
	assert.NoError(t, s.Stop(context.Background()))
}
