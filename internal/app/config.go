package app

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr            string
	DatabaseURL         string
	RepositoryDriver    string
	RedisAddr           string
	RedisQueue          string
	JobMaxAttempts      int
	JobRetryBackoff     time.Duration
	MediumClientID      string
	DevelopmentAuth     bool
	AllowedIngressHost  string
	DBMaxOpenConns      int
	DBMaxIdleConns      int
	DBConnMaxLifetime   time.Duration
}

func LoadConfig() Config {
	return Config{
		HTTPAddr:           env("HTTP_ADDR", ":8080"),
		DatabaseURL:        env("DATABASE_URL", ""),
		RepositoryDriver:   env("REPOSITORY_DRIVER", "postgres"),
		RedisAddr:          env("REDIS_ADDR", "127.0.0.1:6379"),
		RedisQueue:         env("REDIS_QUEUE", "clawplane:jobs"),
		JobMaxAttempts:     envInt("JOB_MAX_ATTEMPTS", 3),
		JobRetryBackoff:    envDuration("JOB_RETRY_BACKOFF", 2*time.Second),
		MediumClientID:     env("MEDIUM_CLIENT_ID", "dev-medium-client"),
		DevelopmentAuth:    envBool("DEVELOPMENT_AUTH", true),
		AllowedIngressHost: env("DEFAULT_INGRESS_HOST", "apps.localhost"),
		DBMaxOpenConns:     envInt("DB_MAX_OPEN_CONNS", 20),
		DBMaxIdleConns:     envInt("DB_MAX_IDLE_CONNS", 10),
		DBConnMaxLifetime:  envDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
