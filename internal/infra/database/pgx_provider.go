package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxProvider struct {
	pool *pgxpool.Pool
}

func NewPgxProvider(ctx context.Context, dsn string) (*PgxProvider, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
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
