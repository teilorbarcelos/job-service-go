package messaging

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Conn represents an AMQP connection's surface area used by the provider.
// Tests can implement this with a fake to cover the success paths
// without a real RabbitMQ.
type Conn interface {
	Channel() (Channel, error)
	Close() error
	IsClosed() bool
}

type Channel interface {
	PublishWithContext(ctx context.Context, exchange, routingKey string, mandatory, immediate bool, msg amqp.Publishing) error
	Close() error
	IsClosed() bool
}

type RabbitProvider struct {
	mu             sync.Mutex
	conn           Conn
	ch             Channel
	url            string
	user           string
	password       string
	publishTimeout time.Duration
	dialer         func(url string, cfg amqp.Config) (Conn, error)
}

type Options struct {
	URL            string
	User           string
	Password       string
	PublishTimeout time.Duration
}

func NewRabbitProvider(opts Options) (*RabbitProvider, error) {
	if opts.URL == "" {
		return nil, errors.New("rabbit url is empty")
	}
	return &RabbitProvider{
		url:            opts.URL,
		user:           opts.User,
		password:       opts.Password,
		publishTimeout: opts.PublishTimeout,
		dialer:         defaultDialer,
	}, nil
}

func defaultDialer(u string, cfg amqp.Config) (Conn, error) {
	c, err := amqp.DialConfig(u, cfg)
	if err != nil {
		return nil, err
	}
	return &amqpConn{Connection: c}, nil
}

type amqpConn struct{ *amqp.Connection }

func (a *amqpConn) Channel() (Channel, error) {
	ch, err := a.Connection.Channel()
	if err != nil {
		return nil, err
	}
	return &amqpChannel{Channel: ch}, nil
}

func (a *amqpConn) Close() error { return a.Connection.Close() }
func (a *amqpConn) IsClosed() bool { return a.Connection.IsClosed() }

type amqpChannel struct{ *amqp.Channel }

func (a *amqpChannel) PublishWithContext(ctx context.Context, exchange, routingKey string, mandatory, immediate bool, msg amqp.Publishing) error {
	return a.Channel.PublishWithContext(ctx, exchange, routingKey, mandatory, immediate, msg)
}

func (a *amqpChannel) Close() error     { return a.Channel.Close() }
func (a *amqpChannel) IsClosed() bool   { return a.Channel.IsClosed() }

func (p *RabbitProvider) Connect() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.conn != nil && !p.conn.IsClosed() {
		return nil
	}
	uri, err := url.Parse(p.url)
	if err != nil {
		return fmt.Errorf("parse rabbit url: %w", err)
	}
	user := p.user
	if user == "" && uri.User != nil {
		user = uri.User.Username()
	}
	pass := p.password
	if pass == "" && uri.User != nil {
		if pwd, ok := uri.User.Password(); ok {
			pass = pwd
		}
	}
	cfg := amqp.Config{
		Heartbeat: 10 * time.Second,
		Locale:   "en_US",
		SASL:     []amqp.Authentication{&amqp.PlainAuth{Username: user, Password: pass}},
	}
	conn, err := p.dialer(p.url, cfg)
	if err != nil {
		return err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return err
	}
	p.conn = conn
	p.ch = ch
	return nil
}

func (p *RabbitProvider) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ch != nil {
		_ = p.ch.Close()
		p.ch = nil
	}
	if p.conn != nil {
		_ = p.conn.Close()
		p.conn = nil
	}
}

func (p *RabbitProvider) IsOpen() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.conn != nil && !p.conn.IsClosed() && p.ch != nil && !p.ch.IsClosed()
}

func (p *RabbitProvider) Publish(exchange, routingKey string, body []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.conn == nil || p.conn.IsClosed() || p.ch == nil || p.ch.IsClosed() {
		return errors.New("rabbit is not connected")
	}
	ctx := context.Background()
	if p.publishTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.publishTimeout)
		defer cancel()
	}
	return p.ch.PublishWithContext(ctx, exchange, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

func (p *RabbitProvider) PublishJSON(exchange, routingKey, jsonBody string) error {
	return p.Publish(exchange, routingKey, []byte(jsonBody))
}
