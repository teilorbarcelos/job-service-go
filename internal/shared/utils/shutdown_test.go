package utils

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateTimeoutContext_AppliesTimeout(t *testing.T) {
	ctx, cancel := CreateTimeoutContext(context.Background(), 50*time.Millisecond)
	defer cancel()
	assert.NoError(t, ctx.Err())
	time.Sleep(100 * time.Millisecond)
	assert.Error(t, ctx.Err())
}

func TestCreateTimeoutContext_ZeroTimeout_NoCancel(t *testing.T) {
	ctx, cancel := CreateTimeoutContext(context.Background(), 0)
	defer cancel()
	time.Sleep(50 * time.Millisecond)
	assert.NoError(t, ctx.Err())
}

func TestCreateTimeoutContext_ParentCancel(t *testing.T) {
	parent, cancelParent := context.WithCancel(context.Background())
	ctx, cancel := CreateTimeoutContext(parent, time.Second)
	defer cancel()
	cancelParent()
	require.Error(t, ctx.Err())
}

func TestWaitForShutdown_StopsOnSignal(t *testing.T) {
	ctx, cancel := WaitForShutdown(context.Background(), NewLogger("info", ""))
	defer cancel()
	cancel()
	assert.Error(t, ctx.Err())
}
