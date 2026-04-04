package postgres

import (
	"context"

	"github.com/gauravprasad/clawcontrol/internal/domain"
)

type AdminRepo struct {
	Base
}

func NewAdminRepo(base Base) *AdminRepo {
	return &AdminRepo{Base: base}
}

func (r *AdminRepo) Summary(ctx context.Context) (*domain.AdminSummary, error) {
	summary := &domain.AdminSummary{}
	err := queryRowContext(ctx, r.db, `
select
  (select count(*) from users) as users,
  (select count(*) from tenants) as tenants,
  (select count(*) from apps) as apps,
  (select count(*) from deployments) as deployments,
  (select count(*) from deployments where status = 'failed') as failed_deployments
`).Scan(&summary.Users, &summary.Tenants, &summary.Apps, &summary.Deployments, &summary.FailedDeployments)
	return summary, wrap("admin summary", err)
}
