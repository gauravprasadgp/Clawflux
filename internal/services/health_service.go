package services

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/gauravprasad/clawcontrol/internal/domain"
)

// BuildVersion can be set at link time: -ldflags "-X services.BuildVersion=v1.2.3"
var BuildVersion = "dev"

// BuildCommit can be set at link time: -ldflags "-X services.BuildCommit=abc1234"
var BuildCommit = "unknown"

type HealthService struct {
	db    *sql.DB
	queue domain.HealthChecker
}

func NewHealthService(db *sql.DB, queue domain.HealthChecker) *HealthService {
	return &HealthService{db: db, queue: queue}
}

func (s *HealthService) Readiness(ctx context.Context) map[string]string {
	status := map[string]string{
		"api":     "ok",
		"version": BuildVersion,
		"commit":  BuildCommit,
		"host":    hostname(),
	}

	if s.db != nil {
		dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if err := s.db.PingContext(dbCtx); err != nil {
			status["database"] = fmt.Sprintf("error: %v", err)
		} else {
			dbStats := s.db.Stats()
			status["database"] = "ok"
			status["db_open_connections"] = fmt.Sprintf("%d", dbStats.OpenConnections)
			status["db_in_use"] = fmt.Sprintf("%d", dbStats.InUse)
		}
	}

	if s.queue != nil {
		queueCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if err := s.queue.Check(queueCtx); err != nil {
			status["redis"] = fmt.Sprintf("error: %v", err)
		} else {
			status["redis"] = "ok"
		}
	}

	return status
}

func hostname() string {
	h, _ := os.Hostname()
	return h
}
