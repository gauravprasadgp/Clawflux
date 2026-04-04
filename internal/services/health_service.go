package services

import (
	"context"
	"database/sql"
	"time"

	"github.com/gauravprasad/clawcontrol/internal/domain"
)

type HealthService struct {
	db    *sql.DB
	queue domain.HealthChecker
}

func NewHealthService(db *sql.DB, queue domain.HealthChecker) *HealthService {
	return &HealthService{db: db, queue: queue}
}

func (s *HealthService) Readiness(ctx context.Context) map[string]string {
	status := map[string]string{
		"api": "ok",
	}
	if s.db != nil {
		dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if err := s.db.PingContext(dbCtx); err != nil {
			status["database"] = "error"
		} else {
			status["database"] = "ok"
		}
	}
	if s.queue != nil {
		queueCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if err := s.queue.Check(queueCtx); err != nil {
			status["redis"] = "error"
		} else {
			status["redis"] = "ok"
		}
	}
	return status
}
