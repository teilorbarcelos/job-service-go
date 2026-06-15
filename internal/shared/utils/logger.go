package utils

import (
	"log/slog"
	"os"
	"strings"
)

func NewLogger(level, environment string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "info", "information":
		lvl = slog.LevelInfo
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: lvl}
	var handler slog.Handler = slog.NewTextHandler(os.Stdout, opts)
	if environment != "" {
		handler = handler.WithAttrs([]slog.Attr{slog.String("env", environment)})
	}
	return slog.New(handler)
}
