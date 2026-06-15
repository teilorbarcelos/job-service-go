package redisinfra

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisProvider struct {
	client *redis.Client
}

type Options struct {
	URL             string
	Host            string
	Port            int
	Password        string
	DB              int
	CommandTimeout  time.Duration
}

func NewRedisProvider(ctx context.Context, opts Options) (*RedisProvider, error) {
	var client *redis.Client
	if strings.HasPrefix(opts.URL, "redis://") || strings.HasPrefix(opts.URL, "rediss://") {
		parsed, err := redis.ParseURL(opts.URL)
		if err != nil {
			return nil, fmt.Errorf("parse redis url: %w", err)
		}
		client = redis.NewClient(parsed)
	} else {
		addr := fmt.Sprintf("%s:%d", opts.Host, opts.Port)
		client = redis.NewClient(&redis.Options{
			Addr:         addr,
			Password:     opts.Password,
			DB:           opts.DB,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  opts.CommandTimeout,
			WriteTimeout: opts.CommandTimeout,
		})
	}
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	return &RedisProvider{client: client}, nil
}

func NewRedisProviderForTest(client *redis.Client) (*RedisProvider, error) {
	if client == nil {
		return nil, errors.New("client is nil")
	}
	return &RedisProvider{client: client}, nil
}

func (p *RedisProvider) Client() *redis.Client {
	return p.client
}

func (p *RedisProvider) Ping(ctx context.Context) error {
	return p.client.Ping(ctx).Err()
}

func (p *RedisProvider) Close() error {
	if p.client == nil {
		return nil
	}
	return p.client.Close()
}
