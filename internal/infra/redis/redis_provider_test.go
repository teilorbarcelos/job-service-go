package redisinfra

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRedisProvider_URL(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	p, err := NewRedisProvider(context.Background(), Options{
		URL: "redis://" + mr.Addr() + "/0",
	})
	require.NoError(t, err)
	assert.NoError(t, p.Ping(context.Background()))
	assert.NoError(t, p.Close())
}

func TestNewRedisProvider_HostPort(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	host, port, err := splitAddr(mr.Addr())
	require.NoError(t, err)
	p, err := NewRedisProvider(context.Background(), Options{
		Host:           host,
		Port:           port,
		CommandTimeout: time.Second,
	})
	require.NoError(t, err)
	assert.NoError(t, p.Ping(context.Background()))
	assert.NoError(t, p.Close())
}

func TestNewRedisProvider_InvalidURL(t *testing.T) {
	_, err := NewRedisProvider(context.Background(), Options{URL: "not-a-url"})
	assert.Error(t, err)
}

func TestNewRedisProvider_UnreachableHost(t *testing.T) {
	_, err := NewRedisProvider(context.Background(), Options{
		Host:           "127.0.0.1",
		Port:           1,
		CommandTimeout: 100 * time.Millisecond,
	})
	assert.Error(t, err)
}

func TestNewRedisProviderForTest_Nil(t *testing.T) {
	_, err := NewRedisProviderForTest(nil)
	assert.Error(t, err)
}

func TestNewRedisProviderForTest_Valid(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()
	p, err := NewRedisProviderForTest(client)
	require.NoError(t, err)
	assert.NoError(t, p.Ping(context.Background()))
}

func TestRedisProvider_Close_Nil(t *testing.T) {
	p := &RedisProvider{}
	assert.NoError(t, p.Close())
}

func splitAddr(addr string) (string, int, error) {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			port := 0
			for _, c := range addr[i+1:] {
				if c < '0' || c > '9' {
					return "", 0, assert.AnError
				}
				port = port*10 + int(c-'0')
			}
			return addr[:i], port, nil
		}
	}
	return "", 0, assert.AnError
}
