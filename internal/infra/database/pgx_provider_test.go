package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
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
