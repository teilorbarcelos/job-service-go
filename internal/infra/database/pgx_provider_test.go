package database

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPgxProvider_InvalidDSN(t *testing.T) {
	_, err := NewPgxProvider(context.Background(), "not-a-valid-dsn")
	assert.Error(t, err)
}

func TestPgxProvider_Close_Idempotent(t *testing.T) {
	p := &PgxProvider{}
	p.Close()
	p.Close()
}

func TestPgxProvider_Pool_NilSafe(t *testing.T) {
	p := &PgxProvider{}
	assert.Nil(t, p.Pool())
}

func TestPgxProvider_Ping_NoRealDB(t *testing.T) {
	p, err := NewPgxProvider(context.Background(), "postgres://invalid:5432/none?sslmode=disable&connect_timeout=1")
	if err != nil {
		return
	}
	defer p.Close()
	assert.Error(t, p.Ping(context.Background()))
}

func TestNewPgxProviderWithFactory_NilFactory(t *testing.T) {
	// Nil factory should fall back to default
	_, err := NewPgxProviderWithFactory(context.Background(), "not-a-valid-dsn", nil)
	assert.Error(t, err)
}

func TestNewPgxProviderWithFactory_FactoryError(t *testing.T) {
	factory := func(_ context.Context, _ *pgxpool.Config) (*pgxpool.Pool, error) {
		return nil, errors.New("factory fail")
	}
	_, err := NewPgxProviderWithFactory(context.Background(), "postgres://localhost:5432/db", factory)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "factory fail")
}

// fakePoolForFactory wraps a *pgxpool.Pool via the factory to test
// the success path. Since we cannot construct a real *pgxpool.Pool
// without a real DB, we document the success path here as untested
// in unit tests (covered in integration tests).
func TestNewPgxProviderWithFactory_SuccessPath_Documented(t *testing.T) {
	// In a real test, you'd construct a real *pgxpool.Pool against a
	// test Postgres. We can't do that without docker/testcontainers,
	// so this test simply asserts that the factory is called with a
	// valid cfg.
	called := false
	factory := func(ctx context.Context, cfg *pgxpool.Config) (*pgxpool.Pool, error) {
		called = true
		assert.NotNil(t, cfg)
		return nil, errors.New("would normally return a pool")
	}
	_, _ = NewPgxProviderWithFactory(context.Background(), "postgres://localhost:5432/db", factory)
	assert.True(t, called)
}
