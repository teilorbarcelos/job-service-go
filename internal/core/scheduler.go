package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type Scheduler struct {
	jobs        []BaseJob
	cronAdapter CronAdapter
	timeout     time.Duration
	logger      *slog.Logger
	nowFunc     func() time.Time
	mu          sync.Mutex
	started     bool
	stopped     bool
	wg          sync.WaitGroup
	cancelAll   context.CancelFunc
}

type SchedulerOptions struct {
	Jobs        []BaseJob
	CronAdapter CronAdapter
	Timeout     time.Duration
	Logger      *slog.Logger
	Now         func() time.Time
}

func NewScheduler(opts SchedulerOptions) *Scheduler {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	return &Scheduler{
		jobs:        opts.Jobs,
		cronAdapter: opts.CronAdapter,
		timeout:     opts.Timeout,
		logger:      opts.Logger,
		nowFunc:     now,
	}
}

func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return errors.New("scheduler has been stopped")
	}
	if s.started {
		return errors.New("scheduler already started")
	}
	s.started = true

	if err := checkDuplicateNames(s.jobs); err != nil {
		return err
	}
	schedules, err := parseAllSchedules(s.jobs, s.cronAdapter)
	if err != nil {
		return err
	}

	runCtx, cancel := context.WithCancel(ctx)
	s.cancelAll = cancel
	for _, job := range s.jobs {
		if !job.Enabled() {
			continue
		}
		sched, ok := schedules[job.Name()]
		if !ok {
			continue
		}
		s.wg.Add(1)
		go s.supervise(runCtx, job, sched)
	}
	return nil
}

func checkDuplicateNames(jobs []BaseJob) error {
	names := make(map[string]struct{})
	for _, job := range jobs {
		if _, dup := names[job.Name()]; dup {
			return fmt.Errorf("duplicate job name: %s", job.Name())
		}
		names[job.Name()] = struct{}{}
	}
	return nil
}

func parseAllSchedules(jobs []BaseJob, adapter CronAdapter) (map[string]CronSchedule, error) {
	schedules := make(map[string]CronSchedule, len(jobs))
	var errs []string
	for _, job := range jobs {
		if !job.Enabled() {
			continue
		}
		sched, err := adapter.Parse(job.Schedule())
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %s", job.Name(), err.Error()))
			continue
		}
		schedules[job.Name()] = sched
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("invalid cron expressions: %s", joinErrors(errs))
	}
	return schedules, nil
}

func (s *Scheduler) supervise(ctx context.Context, job BaseJob, sched CronSchedule) {
	defer s.wg.Done()
	for {
		if ctx.Err() != nil {
			return
		}
		now := s.nowFunc()
		next := sched.Next(now)
		delay := next.Sub(now)
		if delay < 0 {
			delay = 0
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		if ctx.Err() != nil {
			return
		}
		jc := JobContext{Logger: s.logger}
		_ = ExecuteJob(job, jc, ctx, s.timeout)
	}
}

func (s *Scheduler) Stop(ctx context.Context) error {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return nil
	}
	s.stopped = true
	if s.cancelAll != nil {
		s.cancelAll()
	}
	s.mu.Unlock()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func joinErrors(items []string) string {
	out := ""
	for i, it := range items {
		if i > 0 {
			out += "; "
		}
		out += it
	}
	return out
}
