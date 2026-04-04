package postgres

import (
	"context"
	"time"

	"github.com/gauravprasad/clawcontrol/internal/domain"
)

type APIKeyRepo struct {
	Base
}

func NewAPIKeyRepo(base Base) *APIKeyRepo {
	return &APIKeyRepo{Base: base}
}

func (r *APIKeyRepo) Create(ctx context.Context, key *domain.APIKey) error {
	_, err := r.db.ExecContext(ctx, `
insert into api_keys (
  id, tenant_id, user_id, name, key_prefix, key_hash, last_used_at, expires_at, revoked_at, created_at
) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
`, key.ID, key.TenantID, key.UserID, key.Name, key.KeyPrefix, key.KeyHash, nullableTime(key.LastUsedAt), nullableTime(key.ExpiresAt), nullableTime(key.RevokedAt), key.CreatedAt)
	return wrap("create api key", mapSQLError(err))
}

func (r *APIKeyRepo) ListByTenant(ctx context.Context, tenantID string) ([]domain.APIKey, error) {
	rows, err := r.db.QueryContext(ctx, `
select id, tenant_id, user_id, name, key_prefix, key_hash, last_used_at, expires_at, revoked_at, created_at
from api_keys
where tenant_id = $1
order by created_at desc
`, tenantID)
	if err != nil {
		return nil, wrap("list api keys", err)
	}
	defer rows.Close()

	var items []domain.APIKey
	for rows.Next() {
		var key domain.APIKey
		if err := rows.Scan(&key.ID, &key.TenantID, &key.UserID, &key.Name, &key.KeyPrefix, &key.KeyHash, &key.LastUsedAt, &key.ExpiresAt, &key.RevokedAt, &key.CreatedAt); err != nil {
			return nil, wrap("scan api key", err)
		}
		items = append(items, key)
	}
	return items, wrap("list api key rows", rows.Err())
}

func (r *APIKeyRepo) Revoke(ctx context.Context, tenantID, keyID string, revokedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
update api_keys
set revoked_at = $3
where tenant_id = $1 and id = $2
`, tenantID, keyID, revokedAt)
	return wrap("revoke api key", mapSQLError(err))
}

func (r *APIKeyRepo) GetByHash(ctx context.Context, keyHash string) (*domain.APIKey, error) {
	var key domain.APIKey
	err := queryRowContext(ctx, r.db, `
select id, tenant_id, user_id, name, key_prefix, key_hash, last_used_at, expires_at, revoked_at, created_at
from api_keys
where key_hash = $1
`, keyHash).Scan(&key.ID, &key.TenantID, &key.UserID, &key.Name, &key.KeyPrefix, &key.KeyHash, &key.LastUsedAt, &key.ExpiresAt, &key.RevokedAt, &key.CreatedAt)
	if err != nil {
		return nil, mapSQLError(err)
	}
	_, err = r.db.ExecContext(ctx, `update api_keys set last_used_at = now() where id = $1`, key.ID)
	if err == nil {
		now := time.Now().UTC()
		key.LastUsedAt = &now
	}
	return &key, nil
}
