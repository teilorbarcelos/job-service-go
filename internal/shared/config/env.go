package config

import (
	"os"
	"strconv"
	"time"

	apperrors "job-service-go/internal/shared/errors"
)

type AppSettings struct {
	Environment              string
	LogLevel                 string
	ShutdownTimeout          time.Duration
	JobExecutionTimeout      time.Duration
	DatabaseURL              string
	DatabaseCommandTimeout   time.Duration
	RedisURL                 string
	RedisHost                string
	RedisPort                int
	RedisPassword            string
	RedisDB                  int
	RedisCommandTimeout      time.Duration
	MessagingEnabled         bool
	RabbitURL                string
	RabbitUser               string
	RabbitPassword           string
	RabbitPublishTimeout     time.Duration
	HealthCheckCron          string
	HealthCheckEnabled       bool
}

func Load() (*AppSettings, error) {
	return &AppSettings{
		Environment:            getEnv("ENVIRONMENT", "local"),
		LogLevel:               getEnv("LOG_LEVEL", "Information"),
		ShutdownTimeout:        time.Duration(getEnvInt("SHUTDOWN_TIMEOUT_SECONDS", 30)) * time.Second,
		JobExecutionTimeout:    time.Duration(getEnvInt("JOB_EXECUTION_TIMEOUT_SECONDS", 300)) * time.Second,
		DatabaseURL:            getEnv("DATABASE_URL", ""),
		DatabaseCommandTimeout: time.Duration(getEnvInt("DATABASE_COMMAND_TIMEOUT_SECONDS", 10)) * time.Second,
		RedisURL:               getEnv("REDIS_URL", ""),
		RedisHost:              getEnv("REDIS_HOST", "localhost"),
		RedisPort:              getEnvInt("REDIS_PORT", 6379),
		RedisPassword:          getEnv("REDIS_PASSWORD", ""),
		RedisDB:                getEnvInt("REDIS_DB", 0),
		RedisCommandTimeout:    time.Duration(getEnvInt("REDIS_COMMAND_TIMEOUT_SECONDS", 5)) * time.Second,
		MessagingEnabled:       getEnvBool("MESSAGING_ENABLED", false),
		RabbitURL:              getEnv("RABBIT_URL", "amqp://guest:guest@localhost:5672/"),
		RabbitUser:             getEnv("RABBIT_USER", "guest"),
		RabbitPassword:         getEnv("RABBIT_PASSWORD", "guest"),
		RabbitPublishTimeout:   time.Duration(getEnvInt("RABBITMQ_PUBLISH_TIMEOUT", 5)) * time.Second,
		HealthCheckCron:        getEnv("HEALTH_CHECK_CRON", "*/1 * * * *"),
		HealthCheckEnabled:     getEnvBool("HEALTH_CHECK_ENABLED", true),
	}, nil
}

func getEnv(key, fallback string) string {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	return v
}

func getEnvInt(key string, fallback int) int {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		panic(apperrors.NewConfigurationError("invalid integer for " + key + ": '" + raw + "'"))
	}
	return v
}

func getEnvBool(key string, fallback bool) bool {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return fallback
	}
	switch raw {
	case "true", "True", "1":
		return true
	case "false", "False", "0":
		return false
	}
	panic(apperrors.NewConfigurationError("invalid boolean for " + key + ": '" + raw + "'"))
}
