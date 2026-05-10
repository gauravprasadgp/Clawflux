package services

import (
	"context"
	"time"

	"github.com/gauravprasad/clawcontrol/internal/domain"
)

type ReliabilityService struct {
	queue       domain.JobQueue
	deployments domain.DeploymentRepository
	scheduler   domain.Scheduler
}

type ReliabilityStatus struct {
	Queue domain.QueueStats `json:"queue"`
}

type ReconcileResult struct {
	Scanned   int `json:"scanned"`
	Scheduled int `json:"scheduled"`
	Skipped   int `json:"skipped"`
}

func NewReliabilityService(queue domain.JobQueue, deployments domain.DeploymentRepository, scheduler domain.Scheduler) *ReliabilityService {
	return &ReliabilityService{
		queue:       queue,
		deployments: deployments,
		scheduler:   scheduler,
	}
}

func (s *ReliabilityService) Status(ctx context.Context, actor domain.Actor) (*ReliabilityStatus, error) {
	if !actor.IsPlatformAdmin {
		return nil, domain.ErrForbidden
	}
	stats, err := s.queueStats(ctx)
	if err != nil {
		return nil, err
	}
	return &ReliabilityStatus{Queue: *stats}, nil
}

func (s *ReliabilityService) ListDeadLetters(ctx context.Context, actor domain.Actor, limit int) ([]domain.DeadLetterJob, error) {
	if !actor.IsPlatformAdmin {
		return nil, domain.ErrForbidden
	}
	deadLetters, ok := s.queue.(domain.DeadLetterQueue)
	if !ok {
		return nil, domain.ErrValidation
	}
	return deadLetters.ListDeadLetters(ctx, limit)
}

func (s *ReliabilityService) ReplayDeadLetter(ctx context.Context, actor domain.Actor, id string) (*domain.Job, error) {
	if !actor.IsPlatformAdmin {
		return nil, domain.ErrForbidden
	}
	deadLetters, ok := s.queue.(domain.DeadLetterQueue)
	if !ok {
		return nil, domain.ErrValidation
	}
	return deadLetters.ReplayDeadLetter(ctx, id)
}

func (s *ReliabilityService) ReconcileActive(ctx context.Context, actor domain.Actor, limit int) (*ReconcileResult, error) {
	if !actor.IsPlatformAdmin {
		return nil, domain.ErrForbidden
	}
	return s.ReconcileOnce(ctx, limit)
}

func (s *ReliabilityService) ReconcileOnce(ctx context.Context, limit int) (*ReconcileResult, error) {
	if limit <= 0 {
		limit = 100
	}
	items, err := s.deployments.ListActive(ctx, limit)
	if err != nil {
		return nil, err
	}
	result := &ReconcileResult{Scanned: len(items)}
	for i := range items {
		deployment := items[i]
		scheduled, err := s.scheduleReconcile(ctx, &deployment)
		if err != nil {
			return result, err
		}
		if scheduled {
			result.Scheduled++
		} else {
			result.Skipped++
		}
	}
	return result, nil
}

func (s *ReliabilityService) PromoteDue(ctx context.Context) {
	if scheduled, ok := s.queue.(domain.JobScheduler); ok {
		_, _ = scheduled.PromoteDue(ctx, 100)
	}
	if reclaimer, ok := s.queue.(domain.JobLeaseReclaimer); ok {
		_, _ = reclaimer.ReclaimStale(ctx, 10*time.Minute, 100)
	}
}

func (s *ReliabilityService) queueStats(ctx context.Context) (*domain.QueueStats, error) {
	inspector, ok := s.queue.(domain.JobQueueInspector)
	if !ok {
		return &domain.QueueStats{}, nil
	}
	return inspector.Stats(ctx)
}

func (s *ReliabilityService) scheduleReconcile(ctx context.Context, deployment *domain.Deployment) (bool, error) {
	switch deployment.Status {
	case domain.DeploymentStatusQueued:
		return true, s.scheduler.ScheduleDeployment(ctx, deployment)
	case domain.DeploymentStatusDeleting:
		return true, s.scheduler.ScheduleDelete(ctx, deployment)
	case domain.DeploymentStatusProvisioning,
		domain.DeploymentStatusRunning,
		domain.DeploymentStatusDegraded,
		domain.DeploymentStatusRecovering:
		if deployment.BackendRef.Deployment == "" || deployment.BackendRef.Namespace == "" {
			return true, s.scheduler.ScheduleDeployment(ctx, deployment)
		}
		return true, s.scheduler.ScheduleSync(ctx, deployment)
	default:
		return false, nil
	}
}
