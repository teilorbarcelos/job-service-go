package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolFactory is the strategy for creating a *pgxpool.Pool. Production
// code uses defaultPoolFactory; tests can inject a fake.
type PoolFactory func(ctx context.Context, cfg *pgxpool.Config) (*pgxpool.Pool, error)

type PgxProvider struct {
	pool *pgxpool.Pool
}

func NewPgxProvider(ctx context.Context, dsn string) (*PgxProvider, error) {
	return NewPgxProviderWithFactory(ctx, dsn, pgxpool.NewWithConfig)
}

func NewPgxProviderWithFactory(ctx context.Context, dsn string, factory PoolFactory) (*PgxProvider, error) {
	if factory == nil {
		factory = pgxpool.NewWithConfig
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	pool, err := factory(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	return &PgxProvider{pool: pool}, nil
}

func (p *PgxProvider) Pool() *pgxpool.Pool {
	return p.pool
}

func (p *PgxProvider) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

func (p *PgxProvider) Close() {
	if p.pool != nil {
		p.pool.Close()
	}
}
