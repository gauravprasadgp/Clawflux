package postgres

import (
	"context"

	"github.com/gauravprasad/clawcontrol/internal/domain"
)

type DeploymentRepo struct {
	Base
}

func NewDeploymentRepo(base Base) *DeploymentRepo {
	return &DeploymentRepo{Base: base}
}

func (r *DeploymentRepo) Create(ctx context.Context, deployment *domain.Deployment) error {
	config, err := toJSON(deployment.ConfigSnapshot)
	if err != nil {
		return wrap("marshal deployment config", err)
	}
	ref, err := toJSON(deployment.BackendRef)
	if err != nil {
		return wrap("marshal deployment ref", err)
	}
	_, err = r.db.ExecContext(ctx, `
insert into deployments (
  id, tenant_id, app_id, version, image_ref, config_snapshot, status, status_reason, backend, backend_ref,
  requested_by, started_at, finished_at, created_at, updated_at
) values ($1, $2, $3, $4, $5, $6, $7, nullif($8, ''), $9, $10, $11, $12, $13, $14, $15)
`, deployment.ID, deployment.TenantID, deployment.AppID, deployment.Version, deployment.ImageRef, config, deployment.Status, deployment.StatusReason, deployment.Backend, ref, deployment.RequestedBy, nullableTime(deployment.StartedAt), nullableTime(deployment.FinishedAt), deployment.CreatedAt, deployment.UpdatedAt)
	return wrap("create deployment", mapSQLError(err))
}

func (r *DeploymentRepo) Update(ctx context.Context, deployment *domain.Deployment) error {
	config, err := toJSON(deployment.ConfigSnapshot)
	if err != nil {
		return wrap("marshal deployment config", err)
	}
	ref, err := toJSON(deployment.BackendRef)
	if err != nil {
		return wrap("marshal deployment ref", err)
	}
	_, err = r.db.ExecContext(ctx, `
update deployments
set image_ref = $4,
    config_snapshot = $5,
    status = $6,
    status_reason = nullif($7, ''),
    backend = $8,
    backend_ref = $9,
    started_at = $10,
    finished_at = $11,
    updated_at = $12
where id = $1 and tenant_id = $2 and app_id = $3
`, deployment.ID, deployment.TenantID, deployment.AppID, deployment.ImageRef, config, deployment.Status, deployment.StatusReason, deployment.Backend, ref, nullableTime(deployment.StartedAt), nullableTime(deployment.FinishedAt), deployment.UpdatedAt)
	return wrap("update deployment", mapSQLError(err))
}

func (r *DeploymentRepo) GetByID(ctx context.Context, tenantID, deploymentID string) (*domain.Deployment, error) {
	deployment := &domain.Deployment{}
	var rawConfig []byte
	var rawRef []byte
	err := queryRowContext(ctx, r.db, `
select id, tenant_id, app_id, version, image_ref, config_snapshot, status, coalesce(status_reason, ''), backend, backend_ref,
       requested_by, started_at, finished_at, created_at, updated_at
from deployments
where tenant_id = $1 and id = $2
`, tenantID, deploymentID).Scan(
		&deployment.ID,
		&deployment.TenantID,
		&deployment.AppID,
		&deployment.Version,
		&deployment.ImageRef,
		&rawConfig,
		&deployment.Status,
		&deployment.StatusReason,
		&deployment.Backend,
		&rawRef,
		&deployment.RequestedBy,
		&deployment.StartedAt,
		&deployment.FinishedAt,
		&deployment.CreatedAt,
		&deployment.UpdatedAt,
	)
	if err != nil {
		return nil, mapSQLError(err)
	}
	if err := fromJSON(rawConfig, &deployment.ConfigSnapshot); err != nil {
		return nil, wrap("decode deployment config", err)
	}
	if err := fromJSON(rawRef, &deployment.BackendRef); err != nil {
		return nil, wrap("decode deployment ref", err)
	}
	return deployment, nil
}

func (r *DeploymentRepo) ListByApp(ctx context.Context, tenantID, appID string) ([]domain.Deployment, error) {
	rows, err := r.db.QueryContext(ctx, `
select id, tenant_id, app_id, version, image_ref, config_snapshot, status, coalesce(status_reason, ''), backend, backend_ref,
       requested_by, started_at, finished_at, created_at, updated_at
from deployments
where tenant_id = $1 and app_id = $2
order by version asc
`, tenantID, appID)
	if err != nil {
		return nil, wrap("list deployments", err)
	}
	defer rows.Close()

	var items []domain.Deployment
	for rows.Next() {
		var d domain.Deployment
		var rawConfig []byte
		var rawRef []byte
		if err := rows.Scan(&d.ID, &d.TenantID, &d.AppID, &d.Version, &d.ImageRef, &rawConfig, &d.Status, &d.StatusReason, &d.Backend, &rawRef, &d.RequestedBy, &d.StartedAt, &d.FinishedAt, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, wrap("scan deployment", err)
		}
		if err := fromJSON(rawConfig, &d.ConfigSnapshot); err != nil {
			return nil, wrap("decode deployment config", err)
		}
		if err := fromJSON(rawRef, &d.BackendRef); err != nil {
			return nil, wrap("decode deployment ref", err)
		}
		items = append(items, d)
	}
	return items, wrap("list deployment rows", rows.Err())
}

func (r *DeploymentRepo) NextVersion(ctx context.Context, tenantID, appID string) (int, error) {
	var next int
	err := queryRowContext(ctx, r.db, `
select coalesce(max(version), 0) + 1
from deployments
where tenant_id = $1 and app_id = $2
`, tenantID, appID).Scan(&next)
	return next, wrap("next deployment version", err)
}
