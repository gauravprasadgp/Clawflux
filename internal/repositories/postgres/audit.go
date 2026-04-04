package postgres

import (
	"context"

	"github.com/gauravprasad/clawcontrol/internal/domain"
)

type AuditRepo struct {
	Base
}

func NewAuditRepo(base Base) *AuditRepo {
	return &AuditRepo{Base: base}
}

func (r *AuditRepo) Create(ctx context.Context, log *domain.AuditLog) error {
	_, err := r.db.ExecContext(ctx, `
insert into audit_logs (
  id, tenant_id, actor_user_id, actor_api_key_id, action, resource_type, resource_id, message, created_at
) values ($1, $2, nullif($3, ''), nullif($4, ''), $5, $6, $7, $8, $9)
`, log.ID, log.TenantID, log.ActorUserID, log.ActorAPIKeyID, log.Action, log.ResourceType, log.ResourceID, log.Message, log.CreatedAt)
	return wrap("create audit log", mapSQLError(err))
}

func (r *AuditRepo) ListByTenant(ctx context.Context, tenantID string, limit int) ([]domain.AuditLog, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, `
select id, tenant_id, coalesce(actor_user_id::text, ''), coalesce(actor_api_key_id::text, ''), action, resource_type, resource_id, message, created_at
from audit_logs
where tenant_id = $1
order by created_at desc
limit $2
`, tenantID, limit)
	if err != nil {
		return nil, wrap("list audit logs", err)
	}
	defer rows.Close()

	var items []domain.AuditLog
	for rows.Next() {
		var log domain.AuditLog
		if err := rows.Scan(&log.ID, &log.TenantID, &log.ActorUserID, &log.ActorAPIKeyID, &log.Action, &log.ResourceType, &log.ResourceID, &log.Message, &log.CreatedAt); err != nil {
			return nil, wrap("scan audit log", err)
		}
		items = append(items, log)
	}
	return items, wrap("list audit log rows", rows.Err())
}
