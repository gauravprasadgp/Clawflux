package postgres

import (
	"context"

	"github.com/gauravprasad/clawcontrol/internal/domain"
	"github.com/gauravprasad/clawcontrol/internal/platform/idgen"
)

type UserRepo struct {
	Base
}

func NewUserRepo(base Base) *UserRepo {
	return &UserRepo{Base: base}
}

func (r *UserRepo) UpsertByEmail(ctx context.Context, email string, displayName string) (*domain.User, error) {
	user := &domain.User{}
	query := `
insert into users (id, email, display_name, created_at, updated_at)
values ($1, $2, $3, now(), now())
on conflict (email) do update set
  display_name = excluded.display_name,
  updated_at = now()
returning id, email, display_name, coalesce(default_tenant_id::text, ''), created_at, updated_at`
	err := queryRowContext(ctx, r.db, query, idgen.NewUUID(), email, displayName).
		Scan(&user.ID, &user.Email, &user.DisplayName, &user.DefaultTenantID, &user.CreatedAt, &user.UpdatedAt)
	return user, wrap("upsert user", mapSQLError(err))
}

func (r *UserRepo) GetByID(ctx context.Context, userID string) (*domain.User, error) {
	user := &domain.User{}
	query := `
select id, email, display_name, coalesce(default_tenant_id::text, ''), created_at, updated_at
from users
where id = $1`
	err := queryRowContext(ctx, r.db, query, userID).
		Scan(&user.ID, &user.Email, &user.DisplayName, &user.DefaultTenantID, &user.CreatedAt, &user.UpdatedAt)
	return user, mapSQLError(err)
}
