package postgres

import (
	"context"

	"github.com/gauravprasad/clawcontrol/internal/domain"
)

type AppRepo struct {
	Base
}

func NewAppRepo(base Base) *AppRepo {
	return &AppRepo{Base: base}
}

func (r *AppRepo) Create(ctx context.Context, app *domain.App) error {
	config, err := toJSON(app.Config)
	if err != nil {
		return wrap("marshal app config", err)
	}
	_, err = r.db.ExecContext(ctx, `
insert into apps (
  id, tenant_id, name, slug, desired_state, current_deployment_id, config, created_by, created_at, updated_at
) values ($1, $2, $3, $4, $5, nullif($6, ''), $7, $8, $9, $10)
`, app.ID, app.TenantID, app.Name, app.Slug, app.DesiredState, app.CurrentDeploymentID, config, app.CreatedBy, app.CreatedAt, app.UpdatedAt)
	return wrap("create app", mapSQLError(err))
}

func (r *AppRepo) Update(ctx context.Context, app *domain.App) error {
	config, err := toJSON(app.Config)
	if err != nil {
		return wrap("marshal app config", err)
	}
	_, err = r.db.ExecContext(ctx, `
update apps
set name = $3,
    slug = $4,
    desired_state = $5,
    current_deployment_id = nullif($6, ''),
    config = $7,
    updated_at = $8
where id = $1 and tenant_id = $2
`, app.ID, app.TenantID, app.Name, app.Slug, app.DesiredState, app.CurrentDeploymentID, config, app.UpdatedAt)
	return wrap("update app", mapSQLError(err))
}

func (r *AppRepo) GetByID(ctx context.Context, tenantID, appID string) (*domain.App, error) {
	var rawConfig []byte
	app := &domain.App{}
	err := queryRowContext(ctx, r.db, `
select id, tenant_id, name, slug, desired_state, coalesce(current_deployment_id::text, ''), config, created_by, created_at, updated_at
from apps
where tenant_id = $1 and id = $2
`, tenantID, appID).Scan(
		&app.ID,
		&app.TenantID,
		&app.Name,
		&app.Slug,
		&app.DesiredState,
		&app.CurrentDeploymentID,
		&rawConfig,
		&app.CreatedBy,
		&app.CreatedAt,
		&app.UpdatedAt,
	)
	if err != nil {
		return nil, mapSQLError(err)
	}
	if err := fromJSON(rawConfig, &app.Config); err != nil {
		return nil, wrap("decode app config", err)
	}
	return app, nil
}

func (r *AppRepo) ListByTenant(ctx context.Context, tenantID string) ([]domain.App, error) {
	rows, err := r.db.QueryContext(ctx, `
select id, tenant_id, name, slug, desired_state, coalesce(current_deployment_id::text, ''), config, created_by, created_at, updated_at
from apps
where tenant_id = $1
order by created_at asc
`, tenantID)
	if err != nil {
		return nil, wrap("list apps", err)
	}
	defer rows.Close()

	var items []domain.App
	for rows.Next() {
		var app domain.App
		var rawConfig []byte
		if err := rows.Scan(
			&app.ID,
			&app.TenantID,
			&app.Name,
			&app.Slug,
			&app.DesiredState,
			&app.CurrentDeploymentID,
			&rawConfig,
			&app.CreatedBy,
			&app.CreatedAt,
			&app.UpdatedAt,
		); err != nil {
			return nil, wrap("scan app", err)
		}
		if err := fromJSON(rawConfig, &app.Config); err != nil {
			return nil, wrap("decode app config", err)
		}
		items = append(items, app)
	}
	return items, wrap("list apps rows", rows.Err())
}
