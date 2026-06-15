package utils

import (
	"context"
	"os"
	"syscall"
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

func TestWaitForShutdown_RealSignal(t *testing.T) {
	ctx, cancel := WaitForShutdown(context.Background(), NewLogger("info", "ci"))
	defer cancel()
	// Send SIGINT to self — signal.Notify will receive it
	proc, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)
	require.NoError(t, proc.Signal(syscall.SIGINT))

	require.Eventually(t, func() bool {
		return ctx.Err() != nil
	}, time.Second, 10*time.Millisecond, "context should be cancelled after signal")
}
