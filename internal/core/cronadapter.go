package core

import (
	"errors"
	"time"

	"github.com/robfig/cron/v3"
)

type CronAdapter interface {
	Parse(expression string) (CronSchedule, error)
}

type CronSchedule interface {
	Next(from time.Time) time.Time
}

type RobfigAdapter struct{}

func NewRobfigAdapter() *RobfigAdapter {
	return &RobfigAdapter{}
}

func (RobfigAdapter) Parse(expression string) (CronSchedule, error) {
	if expression == "" {
		return nil, errors.New("empty cron expression")
	}
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	sched, err := parser.Parse(expression)
	if err != nil {
		return nil, err
	}
	return &robfigSchedule{sched: sched}, nil
}

type robfigSchedule struct {
	sched cron.Schedule
}

func (r *robfigSchedule) Next(from time.Time) time.Time {
	return r.sched.Next(from)
}
