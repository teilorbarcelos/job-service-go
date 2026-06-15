package health

import (
	"context"
	"errors"
	"testing"
	"time"

	"job-service-go/internal/shared/config"

	"github.com/stretchr/testify/assert"
)

type stubDB struct{ err error }

func (s stubDB) Ping(ctx context.Context) error { return s.err }

type stubRabbit struct{ open bool }

func (s stubRabbit) IsOpen() bool { return s.open }

func newChecker(db pingDB, rd pingDB, rb isOpen, settings *config.AppSettings) *DefaultHealthChecker {
	return &DefaultHealthChecker{db: nil, redis: nil, rabbit: nil, settings: settings}
}

type pingDB interface{ Ping(ctx context.Context) error }
type isOpen interface{ IsOpen() bool }

func TestCheckPostgres_Up(t *testing.T) {
	c := &DefaultHealthChecker{db: &fakePinger{err: nil}, settings: &config.AppSettings{DatabaseCommandTimeout: time.Second}}
	res := c.CheckPostgres(context.Background())
	assert.Equal(t, StatusUp, res.Status)
	assert.GreaterOrEqual(t, res.LatencyMs, int64(0))
}

func TestCheckPostgres_Down_OnError(t *testing.T) {
	c := &DefaultHealthChecker{db: &fakePinger{err: errors.New("conn refused")}, settings: &config.AppSettings{DatabaseCommandTimeout: time.Second}}
	res := c.CheckPostgres(context.Background())
	assert.Equal(t, StatusDown, res.Status)
	assert.Equal(t, "conn refused", res.Error)
}

func TestCheckRedis_Up(t *testing.T) {
	c := &DefaultHealthChecker{redis: &fakePinger{err: nil}, settings: &config.AppSettings{RedisCommandTimeout: time.Second}}
	res := c.CheckRedis(context.Background())
	assert.Equal(t, StatusUp, res.Status)
}

func TestCheckRedis_Down_OnError(t *testing.T) {
	c := &DefaultHealthChecker{redis: &fakePinger{err: errors.New("redis down")}, settings: &config.AppSettings{RedisCommandTimeout: time.Second}}
	res := c.CheckRedis(context.Background())
	assert.Equal(t, StatusDown, res.Status)
	assert.Equal(t, "redis down", res.Error)
}

func TestCheckRabbit_DisabledWhenMessagingOff(t *testing.T) {
	c := &DefaultHealthChecker{settings: &config.AppSettings{MessagingEnabled: false}}
	res := c.CheckRabbit(context.Background())
	assert.Equal(t, StatusDisabled, res.Status)
}

func TestCheckRabbit_UpWhenOpen(t *testing.T) {
	c := &DefaultHealthChecker{settings: &config.AppSettings{MessagingEnabled: true}, rabbit: &fakeRabbit{open: true}}
	res := c.CheckRabbit(context.Background())
	assert.Equal(t, StatusUp, res.Status)
}

func TestCheckRabbit_DownWhenClosed(t *testing.T) {
	c := &DefaultHealthChecker{settings: &config.AppSettings{MessagingEnabled: true}, rabbit: &fakeRabbit{open: false}}
	res := c.CheckRabbit(context.Background())
	assert.Equal(t, StatusDown, res.Status)
	assert.Equal(t, "connection closed", res.Error)
}

type fakePinger struct{ err error }

func (f *fakePinger) Ping(_ context.Context) error { return f.err }

type fakeRabbit struct{ open bool }

func (f *fakeRabbit) IsOpen() bool { return f.open }

var _ pingDB = (*fakePinger)(nil)
var _ isOpen = (*fakeRabbit)(nil)
