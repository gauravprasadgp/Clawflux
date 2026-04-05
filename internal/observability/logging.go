package observability

import (
	"log/slog"
	"os"
	"strings"
)

func NewLogger(service string) *slog.Logger {
	level := levelFromEnv()
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})).With("service", service)
}

func NewLoggerWithLevel(service, levelStr string) *slog.Logger {
	level := parseLevel(levelStr)
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})).With("service", service)
}

func levelFromEnv() slog.Level {
	return parseLevel(os.Getenv("LOG_LEVEL"))
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
