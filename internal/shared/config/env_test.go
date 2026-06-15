package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetEnv(t *testing.T) {
	t.Helper()
	keys := []string{
		"ENVIRONMENT", "LOG_LEVEL", "SHUTDOWN_TIMEOUT_SECONDS", "JOB_EXECUTION_TIMEOUT_SECONDS",
		"DATABASE_URL", "DATABASE_COMMAND_TIMEOUT_SECONDS", "REDIS_URL", "REDIS_HOST", "REDIS_PORT",
		"REDIS_PASSWORD", "REDIS_DB", "REDIS_COMMAND_TIMEOUT_SECONDS",
		"MESSAGING_ENABLED", "RABBIT_URL", "RABBIT_USER", "RABBIT_PASSWORD",
		"RABBITMQ_PUBLISH_TIMEOUT", "HEALTH_CHECK_CRON", "HEALTH_CHECK_ENABLED",
	}
	for _, k := range keys {
		t.Setenv(k, "")
	}
}

func TestLoad_Defaults(t *testing.T) {
	resetEnv(t)
	s, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "local", s.Environment)
	assert.Equal(t, "Information", s.LogLevel)
	assert.Equal(t, "*/1 * * * *", s.HealthCheckCron)
	assert.True(t, s.HealthCheckEnabled)
	assert.False(t, s.MessagingEnabled)
	assert.Equal(t, "localhost", s.RedisHost)
	assert.Equal(t, 6379, s.RedisPort)
}

func TestLoad_Overrides(t *testing.T) {
	resetEnv(t)
	t.Setenv("ENVIRONMENT", "prod")
	t.Setenv("JOB_EXECUTION_TIMEOUT_SECONDS", "60")
	t.Setenv("MESSAGING_ENABLED", "true")
	s, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "prod", s.Environment)
	assert.Equal(t, 60, int(s.JobExecutionTimeout.Seconds()))
	assert.True(t, s.MessagingEnabled)
}

func TestLoad_InvalidInt_Panics(t *testing.T) {
	resetEnv(t)
	t.Setenv("JOB_EXECUTION_TIMEOUT_SECONDS", "not-a-number")
	assert.Panics(t, func() { _, _ = Load() })
}

func TestLoad_InvalidBool_Panics(t *testing.T) {
	resetEnv(t)
	t.Setenv("MESSAGING_ENABLED", "maybe")
	assert.Panics(t, func() { _, _ = Load() })
}

func TestGetEnvBool_Truthy(t *testing.T) {
	t.Setenv("X", "true")
	assert.True(t, getEnvBool("X", false))
}

func TestGetEnvInt_Invalid(t *testing.T) {
	t.Setenv("X", "abc")
	assert.Panics(t, func() { getEnvInt("X", 0) })
}
