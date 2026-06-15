package utils

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func WaitForShutdown(parent context.Context, logger *slog.Logger) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		select {
		case <-ctx.Done():
			return
		case sig := <-sigCh:
			logger.Info("shutdown signal received", "signal", sig.String())
			cancel()
		}
	}()
	return ctx, cancel
}
