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

type RabbitProvider struct {
	mu            sync.Mutex
	conn          *amqp.Connection
	ch            *amqp.Channel
	url           string
	user          string
	password      string
	publishTimeout time.Duration
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
	}, nil
}

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
	conn, err := amqp.DialConfig(p.url, cfg)
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
