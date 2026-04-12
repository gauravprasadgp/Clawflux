package postgres

import (
	"context"
	"time"

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

func (r *AdminRepo) ListAllInstances(ctx context.Context) ([]domain.AdminInstance, error) {
	rows, err := r.db.QueryContext(ctx, `
select
  a.id, a.tenant_id, a.name, a.slug, a.desired_state, a.current_deployment_id, a.config, a.created_by, a.created_at, a.updated_at,
  coalesce(u.email, '') as user_email,
  d.id, d.tenant_id, d.app_id, d.version, d.image_ref, d.config_snapshot, d.status, coalesce(d.status_reason, ''), d.backend, d.backend_ref,
  d.requested_by, d.started_at, d.finished_at, d.created_at, d.updated_at
from apps a
left join tenant_members tm on tm.tenant_id = a.tenant_id and tm.role = 'owner'
left join users u on u.id = tm.user_id
left join lateral (
  select * from deployments dep
  where dep.app_id = a.id
  order by dep.version desc
  limit 1
) d on true
order by a.created_at desc
`)
	if err != nil {
		return nil, wrap("list all instances", err)
	}
	defer rows.Close()

	var out []domain.AdminInstance
	for rows.Next() {
		var inst domain.AdminInstance
		var rawAppConfig []byte
		var curDepID *string

		// nullable deployment fields
		var depID, depTenantID, depAppID, depImageRef, depBackend, depRequestedBy *string
		var depVersion *int
		var depStatus *domain.DeploymentStatus
		var depStatusReason *string
		var rawDepConfig, rawDepRef []byte
		var depStartedAt, depFinishedAt, depCreatedAt, depUpdatedAt *time.Time

		err := rows.Scan(
			&inst.App.ID, &inst.App.TenantID, &inst.App.Name, &inst.App.Slug,
			&inst.App.DesiredState, &curDepID, &rawAppConfig,
			&inst.App.CreatedBy, &inst.App.CreatedAt, &inst.App.UpdatedAt,
			&inst.UserEmail,
			&depID, &depTenantID, &depAppID, &depVersion, &depImageRef,
			&rawDepConfig, &depStatus, &depStatusReason, &depBackend, &rawDepRef,
			&depRequestedBy, &depStartedAt, &depFinishedAt, &depCreatedAt, &depUpdatedAt,
		)
		if err != nil {
			return nil, wrap("scan instance row", err)
		}
		if curDepID != nil {
			inst.App.CurrentDeploymentID = *curDepID
		}
		if err := fromJSON(rawAppConfig, &inst.App.Config); err != nil {
			return nil, wrap("decode app config", err)
		}
		if depID != nil {
			d := &domain.Deployment{
				ID:          *depID,
				TenantID:    *depTenantID,
				AppID:       *depAppID,
				Version:     *depVersion,
				ImageRef:    *depImageRef,
				Status:      *depStatus,
				Backend:     *depBackend,
				RequestedBy: *depRequestedBy,
				StartedAt:   depStartedAt,
				FinishedAt:  depFinishedAt,
				CreatedAt:   *depCreatedAt,
				UpdatedAt:   *depUpdatedAt,
			}
			if depStatusReason != nil {
				d.StatusReason = *depStatusReason
			}
			if err := fromJSON(rawDepConfig, &d.ConfigSnapshot); err != nil {
				return nil, wrap("decode dep config snapshot", err)
			}
			if err := fromJSON(rawDepRef, &d.BackendRef); err != nil {
				return nil, wrap("decode dep backend ref", err)
			}
			inst.Deployment = d
		}
		out = append(out, inst)
	}
	return out, wrap("list all instances rows", rows.Err())
}
