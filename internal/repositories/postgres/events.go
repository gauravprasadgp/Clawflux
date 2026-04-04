package postgres

import (
	"context"

	"github.com/gauravprasad/clawcontrol/internal/domain"
)

type EventRepo struct {
	Base
}

func NewEventRepo(base Base) *EventRepo {
	return &EventRepo{Base: base}
}

func (r *EventRepo) Create(ctx context.Context, event *domain.DeploymentEvent) error {
	_, err := r.db.ExecContext(ctx, `
insert into deployment_events (id, deployment_id, tenant_id, type, message, created_at)
values ($1, $2, $3, $4, $5, $6)
`, event.ID, event.DeploymentID, event.TenantID, event.Type, event.Message, event.CreatedAt)
	return wrap("create deployment event", mapSQLError(err))
}

func (r *EventRepo) ListByDeployment(ctx context.Context, tenantID, deploymentID string) ([]domain.DeploymentEvent, error) {
	rows, err := r.db.QueryContext(ctx, `
select id, deployment_id, tenant_id, type, message, created_at
from deployment_events
where tenant_id = $1 and deployment_id = $2
order by created_at asc
`, tenantID, deploymentID)
	if err != nil {
		return nil, wrap("list deployment events", err)
	}
	defer rows.Close()

	var items []domain.DeploymentEvent
	for rows.Next() {
		var event domain.DeploymentEvent
		if err := rows.Scan(&event.ID, &event.DeploymentID, &event.TenantID, &event.Type, &event.Message, &event.CreatedAt); err != nil {
			return nil, wrap("scan deployment event", err)
		}
		items = append(items, event)
	}
	return items, wrap("list event rows", rows.Err())
}
