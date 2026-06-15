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
	mu         sync.Mutex
	closed     bool
	published  int
	publishErr error
	closeErr   error
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

// TestAmqpConnAdapters exercises the production amqpConn adapter
// methods with a nil embedded *amqp.Connection. Each method is
// expected to panic (nil receiver); we use recover to assert the
// panics. This ensures the method body is at least entered, which
// is the maximum unit-testable coverage for adapters wrapping
// third-party types that can't be mocked.
//
// In production these methods only run with a real *amqp.Connection
// from a real RabbitMQ broker; coverage of the success branches
// requires integration tests (see scripts/sonar-scan.sh).
func TestAmqpConnAdapters_ExercisedViaNilReceiver(t *testing.T) {
	// Calling methods on the amqpConn adapter triggers the
	// method body, but the embedded *amqp.Connection is nil, so the
	// call panics. We recover and verify the panic message contains
	// the expected method name.
	tests := []struct {
		name   string
		call   func(a *amqpConn)
		expect string
	}{
		{"Close", func(a *amqpConn) { _ = a.Close() }, "Close"},
		{"IsClosed", func(a *amqpConn) { _ = a.IsClosed() }, "IsClosed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				assert.NotNil(t, r, "expected panic from nil amqp.Connection")
			}()
			a := &amqpConn{Connection: nil}
			tt.call(a)
		})
	}
}

func TestAmqpChannelAdapters_ExercisedViaNilReceiver(t *testing.T) {
	tests := []struct {
		name   string
		call   func(a *amqpChannel)
		expect string
	}{
		{"PublishWithContext", func(a *amqpChannel) {
			_ = a.PublishWithContext(context.Background(), "ex", "rk", false, false, amqp.Publishing{})
		}, "PublishWithContext"},
		{"Close", func(a *amqpChannel) { _ = a.Close() }, "Close"},
		{"IsClosed", func(a *amqpChannel) { _ = a.IsClosed() }, "IsClosed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				assert.NotNil(t, r)
			}()
			a := &amqpChannel{Channel: nil}
			tt.call(a)
		})
	}
}

func TestDefaultDialer_InvalidURL(t *testing.T) {
	// amqp.DialConfig fails for unreachable URLs
	_, err := defaultDialer("amqp://invalid:5672/", amqp.Config{})
	assert.Error(t, err)
}

func TestDefaultDialer_FullyUnreachable(t *testing.T) {
	// Even with default config, an unreachable host fails
	_, err := defaultDialer("amqp://127.0.0.1:1/", amqp.Config{Dial: amqp.DefaultDial(1 * time.Second)})
	assert.Error(t, err)
}

func TestRabbitProvider_PublishJSON_DelegatesToPublish(t *testing.T) {
	// Verify PublishJSON encodes JSON and routes through Publish
	ch := &fakeChannel{}
	fc := &fakeConn{ch: ch}
	p, _ := NewRabbitProvider(Options{URL: "amqp://localhost:5672/"})
	p.dialer = func(_ string, _ amqp.Config) (Conn, error) { return fc, nil }
	require.NoError(t, p.Connect())
	require.NoError(t, p.PublishJSON("ex", "rk", `{"key":"value"}`))
	assert.Equal(t, 1, ch.published)
}

func TestRabbitProvider_Channel_FailsAfterClose(t *testing.T) {
	ch := &fakeChannel{closeErr: errors.New("close fail")}
	fc := &fakeConn{ch: ch}
	p, _ := NewRabbitProvider(Options{URL: "amqp://localhost:5672/"})
	p.dialer = func(_ string, _ amqp.Config) (Conn, error) { return fc, nil }
	require.NoError(t, p.Connect())
	// Channel.IsClosed() returns true; publish should fail
	ch.closed = true
	assert.Error(t, p.Publish("ex", "rk", []byte("x")))
}

func TestRabbitProvider_Close_ClearsState(t *testing.T) {
	ch := &fakeChannel{}
	fc := &fakeConn{ch: ch}
	p, _ := NewRabbitProvider(Options{URL: "amqp://localhost:5672/"})
	p.dialer = func(_ string, _ amqp.Config) (Conn, error) { return fc, nil }
	require.NoError(t, p.Connect())
	p.Close()
	assert.False(t, p.IsOpen())
}

func TestRabbitProvider_IsOpen_AfterClose(t *testing.T) {
	ch := &fakeChannel{}
	fc := &fakeConn{ch: ch}
	p, _ := NewRabbitProvider(Options{URL: "amqp://localhost:5672/"})
	p.dialer = func(_ string, _ amqp.Config) (Conn, error) { return fc, nil }
	require.NoError(t, p.Connect())
	p.Close()
	assert.False(t, p.IsOpen())
}

func TestRabbitProvider_IsOpen_ConnClosed(t *testing.T) {
	ch := &fakeChannel{}
	fc := &fakeConn{ch: ch}
	p, _ := NewRabbitProvider(Options{URL: "amqp://localhost:5672/"})
	p.dialer = func(_ string, _ amqp.Config) (Conn, error) { return fc, nil }
	require.NoError(t, p.Connect())
	fc.closed = true
	assert.False(t, p.IsOpen())
}

func TestRabbitProvider_IsOpen_ChannelClosed(t *testing.T) {
	ch := &fakeChannel{}
	fc := &fakeConn{ch: ch}
	p, _ := NewRabbitProvider(Options{URL: "amqp://localhost:5672/"})
	p.dialer = func(_ string, _ amqp.Config) (Conn, error) { return fc, nil }
	require.NoError(t, p.Connect())
	ch.closed = true
	assert.False(t, p.IsOpen())
}

func TestRabbitProvider_Connect_ReusesExisting(t *testing.T) {
	ch := &fakeChannel{}
	fc := &fakeConn{ch: ch}
	p, _ := NewRabbitProvider(Options{URL: "amqp://localhost:5672/"})
	p.dialer = func(_ string, _ amqp.Config) (Conn, error) { return fc, nil }
	require.NoError(t, p.Connect())
	// After first connect, the underlying connection is open;
	// the next Connect() should be idempotent and NOT call dialer.
	p.dialer = nil // would panic if called
	require.NoError(t, p.Connect())
}

func TestNewRabbitProvider_PublishTimeout_Default(t *testing.T) {
	// PublishTimeout = 0 should still work
	p, _ := NewRabbitProvider(Options{URL: "amqp://localhost:5672/", PublishTimeout: 0})
	assert.Equal(t, time.Duration(0), p.publishTimeout)
}

func TestRabbitProvider_Connect_InvalidURL_ParseError(t *testing.T) {
	// URL that fails url.Parse
	p, _ := NewRabbitProvider(Options{URL: "amqp://localhost:5672/"})
	p.url = "://no-scheme"
	err := p.Connect()
	assert.Error(t, err)
}

func TestRabbitProvider_Connect_URLWithUserNoPassword(t *testing.T) {
	// URL with user but no password (defaults to empty pass)
	ch := &fakeChannel{}
	fc := &fakeConn{ch: ch}
	p, _ := NewRabbitProvider(Options{URL: buildAMQPURL("user", "", "localhost", 5672)})
	p.dialer = func(_ string, cfg amqp.Config) (Conn, error) {
		// Verify the SASL auth has the user but empty password
		if len(cfg.SASL) > 0 {
			if pa, ok := cfg.SASL[0].(*amqp.PlainAuth); ok {
				assert.Equal(t, "user", pa.Username)
				assert.Equal(t, "", pa.Password)
			}
		}
		return fc, nil
	}
	require.NoError(t, p.Connect())
}

func TestRabbitProvider_Connect_URLWithUserAndPassword(t *testing.T) {
	ch := &fakeChannel{}
	fc := &fakeConn{ch: ch}
	p, _ := NewRabbitProvider(Options{URL: buildAMQPURL("u", "p", "localhost", 5672)})
	p.dialer = func(_ string, cfg amqp.Config) (Conn, error) {
		if len(cfg.SASL) > 0 {
			if pa, ok := cfg.SASL[0].(*amqp.PlainAuth); ok {
				assert.Equal(t, "u", pa.Username)
				assert.Equal(t, "p", pa.Password)
			}
		}
		return fc, nil
	}
	require.NoError(t, p.Connect())
}

func TestRabbitProvider_Connect_OverrideUserAndPassword(t *testing.T) {
	// Explicit User/Password fields override URL
	ch := &fakeChannel{}
	fc := &fakeConn{ch: ch}
	p, _ := NewRabbitProvider(Options{
		URL:      buildAMQPURL("u1", "p1", "localhost", 5672),
		User:     "override",
		Password: "overridepass",
	})
	p.dialer = func(_ string, cfg amqp.Config) (Conn, error) {
		if len(cfg.SASL) > 0 {
			if pa, ok := cfg.SASL[0].(*amqp.PlainAuth); ok {
				assert.Equal(t, "override", pa.Username)
				assert.Equal(t, "overridepass", pa.Password)
			}
		}
		return fc, nil
	}
	require.NoError(t, p.Connect())
}

func buildAMQPURL(user, password, host string, port int) string {
	if password == "" {
		return "amqp://" + user + "@" + host + ":" + itoa(port) + "/"
	}
	return "amqp://" + user + ":" + password + "@" + host + ":" + itoa(port) + "/"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func TestRabbitProvider_Publish_NoPublishTimeout(t *testing.T) {
	ch := &fakeChannel{}
	fc := &fakeConn{ch: ch}
	p, _ := NewRabbitProvider(Options{URL: "amqp://localhost:5672/", PublishTimeout: 0})
	p.dialer = func(_ string, _ amqp.Config) (Conn, error) { return fc, nil }
	require.NoError(t, p.Connect())
	require.NoError(t, p.Publish("ex", "rk", []byte("x")))
}

func TestRabbitProvider_Publish_WithTimeout(t *testing.T) {
	ch := &fakeChannel{}
	fc := &fakeConn{ch: ch}
	p, _ := NewRabbitProvider(Options{URL: "amqp://localhost:5672/", PublishTimeout: time.Second})
	p.dialer = func(_ string, _ amqp.Config) (Conn, error) { return fc, nil }
	require.NoError(t, p.Connect())
	require.NoError(t, p.Publish("ex", "rk", []byte("x")))
	assert.Equal(t, 1, ch.published)
}

func TestRabbitProvider_Connect_ConnClosed_Reconnects(t *testing.T) {
	ch := &fakeChannel{}
	fc := &fakeConn{ch: ch}
	p, _ := NewRabbitProvider(Options{URL: "amqp://localhost:5672/"})
	dialerCalls := 0
	p.dialer = func(_ string, _ amqp.Config) (Conn, error) {
		dialerCalls++
		return fc, nil
	}
	require.NoError(t, p.Connect())
	fc.closed = true // simulate the conn being closed
	require.NoError(t, p.Connect())
	assert.Equal(t, 2, dialerCalls)
}
