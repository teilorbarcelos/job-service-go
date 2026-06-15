package messaging

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeConn struct {
	mu       sync.Mutex
	closed   bool
	ch       *fakeChannel
	chErr    error
	closeErr error
}

func (f *fakeConn) Channel() (Channel, error) {
	if f.chErr != nil {
		return nil, f.chErr
	}
	return f.ch, nil
}

func (f *fakeConn) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return f.closeErr
}

func (f *fakeConn) IsClosed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closed
}

type fakeChannel struct {
	mu            sync.Mutex
	closed        bool
	published     int
	publishErr    error
	closeErr      error
}

func (f *fakeChannel) PublishWithContext(_ context.Context, _, _ string, _, _ bool, _ amqp.Publishing) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.published++
	return f.publishErr
}

func (f *fakeChannel) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return f.closeErr
}

func (f *fakeChannel) IsClosed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closed
}

func TestNewRabbitProvider_EmptyURL(t *testing.T) {
	_, err := NewRabbitProvider(Options{})
	assert.Error(t, err)
}

func TestRabbitProvider_Connect_InvalidURL(t *testing.T) {
	p, err := NewRabbitProvider(Options{URL: "://invalid"})
	assert.NoError(t, err)
	err = p.Connect()
	assert.Error(t, err)
}

func TestRabbitProvider_Connect_InvalidURL_Parse(t *testing.T) {
	// Unparseable URL after escaping
	p, _ := NewRabbitProvider(Options{URL: "amqp://localhost:5672/"})
	p.url = "\x00invalid"
	err := p.Connect()
	assert.Error(t, err)
}

func TestRabbitProvider_Connect_DialerFails(t *testing.T) {
	p, _ := NewRabbitProvider(Options{URL: "amqp://localhost:5672/"})
	p.dialer = func(_ string, _ amqp.Config) (Conn, error) { return nil, errors.New("dial fail") }
	err := p.Connect()
	assert.Error(t, err)
}

func TestRabbitProvider_Connect_ChannelFails(t *testing.T) {
	p, _ := NewRabbitProvider(Options{URL: "amqp://localhost:5672/"})
	fc := &fakeConn{chErr: errors.New("channel fail")}
	p.dialer = func(_ string, _ amqp.Config) (Conn, error) { return fc, nil }
	err := p.Connect()
	assert.Error(t, err)
	assert.True(t, fc.closed)
}

func TestRabbitProvider_Connect_Succeeds(t *testing.T) {
	p, _ := NewRabbitProvider(Options{URL: "amqp://localhost:5672/", PublishTimeout: time.Second})
	ch := &fakeChannel{}
	fc := &fakeConn{ch: ch}
	p.dialer = func(_ string, _ amqp.Config) (Conn, error) { return fc, nil }
	require.NoError(t, p.Connect())
	assert.True(t, p.IsOpen())
}

func TestRabbitProvider_Connect_Idempotent(t *testing.T) {
	p, _ := NewRabbitProvider(Options{URL: "amqp://localhost:5672/"})
	ch := &fakeChannel{}
	fc := &fakeConn{ch: ch}
	dialCalls := 0
	p.dialer = func(_ string, _ amqp.Config) (Conn, error) {
		dialCalls++
		return fc, nil
	}
	require.NoError(t, p.Connect())
	require.NoError(t, p.Connect())
	assert.Equal(t, 1, dialCalls)
}

func TestRabbitProvider_Close_AfterConnect(t *testing.T) {
	p, _ := NewRabbitProvider(Options{URL: "amqp://localhost:5672/"})
	ch := &fakeChannel{}
	fc := &fakeConn{ch: ch}
	p.dialer = func(_ string, _ amqp.Config) (Conn, error) { return fc, nil }
	require.NoError(t, p.Connect())
	p.Close()
	assert.True(t, ch.closed)
	assert.True(t, fc.closed)
}

func TestRabbitProvider_Close_Idempotent(t *testing.T) {
	p, _ := NewRabbitProvider(Options{URL: "amqp://localhost:5672/"})
	p.Close()
	p.Close()
}

func TestRabbitProvider_IsOpen_BeforeConnect(t *testing.T) {
	p, _ := NewRabbitProvider(Options{URL: "amqp://localhost:5672/"})
	assert.False(t, p.IsOpen())
}

func TestRabbitProvider_Publish_BeforeConnect(t *testing.T) {
	p, _ := NewRabbitProvider(Options{URL: "amqp://localhost:5672/"})
	err := p.Publish("ex", "rk", []byte("x"))
	assert.Error(t, err)
}

func TestRabbitProvider_PublishJSON_BeforeConnect(t *testing.T) {
	p, _ := NewRabbitProvider(Options{URL: "amqp://localhost:5672/"})
	err := p.PublishJSON("ex", "rk", "{}")
	assert.Error(t, err)
}

func TestRabbitProvider_Publish_Success(t *testing.T) {
	p, _ := NewRabbitProvider(Options{URL: "amqp://localhost:5672/", PublishTimeout: time.Second})
	ch := &fakeChannel{}
	fc := &fakeConn{ch: ch}
	p.dialer = func(_ string, _ amqp.Config) (Conn, error) { return fc, nil }
	require.NoError(t, p.Connect())
	require.NoError(t, p.Publish("ex", "rk", []byte("payload")))
	assert.Equal(t, 1, ch.published)
}

func TestRabbitProvider_Publish_ZeroTimeout(t *testing.T) {
	p, _ := NewRabbitProvider(Options{URL: "amqp://localhost:5672/", PublishTimeout: 0})
	ch := &fakeChannel{}
	fc := &fakeConn{ch: ch}
	p.dialer = func(_ string, _ amqp.Config) (Conn, error) { return fc, nil }
	require.NoError(t, p.Connect())
	require.NoError(t, p.Publish("ex", "rk", []byte("x")))
	assert.Equal(t, 1, ch.published)
}

func TestRabbitProvider_Publish_ChannelError(t *testing.T) {
	p, _ := NewRabbitProvider(Options{URL: "amqp://localhost:5672/"})
	ch := &fakeChannel{publishErr: errors.New("publish fail")}
	fc := &fakeConn{ch: ch}
	p.dialer = func(_ string, _ amqp.Config) (Conn, error) { return fc, nil }
	require.NoError(t, p.Connect())
	assert.Error(t, p.Publish("ex", "rk", []byte("x")))
}

func TestRabbitProvider_PublishJSON_Success(t *testing.T) {
	p, _ := NewRabbitProvider(Options{URL: "amqp://localhost:5672/"})
	ch := &fakeChannel{}
	fc := &fakeConn{ch: ch}
	p.dialer = func(_ string, _ amqp.Config) (Conn, error) { return fc, nil }
	require.NoError(t, p.Connect())
	require.NoError(t, p.PublishJSON("ex", "rk", "{}"))
	assert.Equal(t, 1, ch.published)
}
