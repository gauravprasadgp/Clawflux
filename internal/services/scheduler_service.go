package services

import (
	"context"
	"time"

	"github.com/gauravprasad/clawcontrol/internal/domain"
	"github.com/gauravprasad/clawcontrol/internal/platform/idgen"
)

type SchedulerService struct {
	queue domain.JobQueue
}

func NewSchedulerService(queue domain.JobQueue) *SchedulerService {
	return &SchedulerService{queue: queue}
}

func (s *SchedulerService) ScheduleDeployment(ctx context.Context, deployment *domain.Deployment) error {
	return s.queue.Enqueue(ctx, domain.Job{
		ID:           idgen.New("job"),
		Type:         domain.JobTypeDeploymentCreate,
		TenantID:     deployment.TenantID,
		AppID:        deployment.AppID,
		DeploymentID: deployment.ID,
		Attempts:     0,
		CreatedAt:    time.Now().UTC(),
	})
}
