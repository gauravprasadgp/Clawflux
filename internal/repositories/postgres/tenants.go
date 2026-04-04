package postgres

import (
	"context"
	"strings"

	"github.com/gauravprasad/clawcontrol/internal/domain"
	"github.com/gauravprasad/clawcontrol/internal/platform/idgen"
)

type TenantRepo struct {
	Base
}

func NewTenantRepo(base Base) *TenantRepo {
	return &TenantRepo{Base: base}
}

func (r *TenantRepo) CreatePersonalTenant(ctx context.Context, user *domain.User) (*domain.Tenant, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, wrap("begin tenant tx", err)
	}
	defer tx.Rollback()

	tenant := &domain.Tenant{
		ID:     idgen.NewUUID(),
		Slug:   sanitizeSlug(user.Email),
		Name:   fallback(user.DisplayName, user.Email),
		Plan:   "free",
		Status: "active",
	}

	err = tx.QueryRowContext(ctx, `
insert into tenants (id, slug, name, plan, status, created_at, updated_at)
values ($1, $2, $3, $4, $5, now(), now())
returning created_at, updated_at
`, tenant.ID, tenant.Slug, tenant.Name, tenant.Plan, tenant.Status).
		Scan(&tenant.CreatedAt, &tenant.UpdatedAt)
	if err != nil {
		return nil, wrap("insert tenant", err)
	}

	_, err = tx.ExecContext(ctx, `
insert into tenant_members (tenant_id, user_id, role, created_at)
values ($1, $2, $3, now())
`, tenant.ID, user.ID, domain.RoleOwner)
	if err != nil {
		return nil, wrap("insert tenant member", mapSQLError(err))
	}

	_, err = tx.ExecContext(ctx, `
update users
set default_tenant_id = $2, updated_at = now()
where id = $1
`, user.ID, tenant.ID)
	if err != nil {
		return nil, wrap("update user tenant", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, wrap("commit tenant tx", err)
	}
	return tenant, nil
}

func (r *TenantRepo) GetByID(ctx context.Context, tenantID string) (*domain.Tenant, error) {
	tenant := &domain.Tenant{}
	err := queryRowContext(ctx, r.db, `
select id, slug, name, plan, status, created_at, updated_at
from tenants
where id = $1
`, tenantID).Scan(&tenant.ID, &tenant.Slug, &tenant.Name, &tenant.Plan, &tenant.Status, &tenant.CreatedAt, &tenant.UpdatedAt)
	return tenant, mapSQLError(err)
}

func (r *TenantRepo) AddMember(ctx context.Context, member domain.TenantMember) error {
	_, err := r.db.ExecContext(ctx, `
insert into tenant_members (tenant_id, user_id, role, created_at)
values ($1, $2, $3, $4)
on conflict (tenant_id, user_id) do update set role = excluded.role
`, member.TenantID, member.UserID, member.Role, member.CreatedAt)
	return wrap("add tenant member", mapSQLError(err))
}

func (r *TenantRepo) GetMember(ctx context.Context, tenantID, userID string) (*domain.TenantMember, error) {
	member := &domain.TenantMember{}
	err := queryRowContext(ctx, r.db, `
select tenant_id, user_id, role, created_at
from tenant_members
where tenant_id = $1 and user_id = $2
`, tenantID, userID).Scan(&member.TenantID, &member.UserID, &member.Role, &member.CreatedAt)
	return member, mapSQLError(err)
}

func sanitizeSlug(in string) string {
	in = strings.ToLower(strings.TrimSpace(in))
	in = strings.ReplaceAll(in, "@", "-")
	in = strings.ReplaceAll(in, ".", "-")
	in = strings.ReplaceAll(in, "_", "-")
	if in == "" {
		return "tenant"
	}
	return in
}

func fallback(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
