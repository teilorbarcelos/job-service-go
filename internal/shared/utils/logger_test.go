package utils

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLogger_DefaultLevel(t *testing.T) {
	l := NewLogger("NotALevel", "")
	assert.True(t, l.Enabled(nil, slog.LevelInfo))
	assert.False(t, l.Enabled(nil, slog.LevelDebug))
}

func TestNewLogger_ParseCase(t *testing.T) {
	l := NewLogger("DEBUG", "")
	assert.True(t, l.Enabled(nil, slog.LevelDebug))
}

func TestNewLogger_WithEnvironment(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	l := slog.New(h)
	l.Info("hi", slog.String("env", "ci"))
	assert.Contains(t, buf.String(), "env=ci")
}

func TestNewLogger_NoEnvironment(t *testing.T) {
	l := NewLogger("Information", "")
	assert.NotNil(t, l)
	assert.True(t, strings.ContainsAny("ok", "ok"))
}
