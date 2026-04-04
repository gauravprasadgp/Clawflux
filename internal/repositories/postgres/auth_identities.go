package postgres

import (
	"context"

	"github.com/gauravprasad/clawcontrol/internal/domain"
)

type AuthIdentityRepo struct {
	Base
}

func NewAuthIdentityRepo(base Base) *AuthIdentityRepo {
	return &AuthIdentityRepo{Base: base}
}

func (r *AuthIdentityRepo) Upsert(ctx context.Context, identity *domain.AuthIdentity) error {
	metadata, err := toJSON(identity.Metadata)
	if err != nil {
		return wrap("marshal auth identity", err)
	}
	_, err = r.db.ExecContext(ctx, `
insert into auth_identities (
  id, user_id, provider, provider_user_id, access_token_encrypted, refresh_token_encrypted, metadata, created_at, updated_at
) values ($1, $2, $3, $4, nullif($5, ''), nullif($6, ''), $7, now(), now())
on conflict (provider, provider_user_id) do update set
  user_id = excluded.user_id,
  access_token_encrypted = excluded.access_token_encrypted,
  refresh_token_encrypted = excluded.refresh_token_encrypted,
  metadata = excluded.metadata,
  updated_at = now()
`, identity.ID, identity.UserID, identity.Provider, identity.ProviderUserID, identity.AccessToken, identity.RefreshToken, metadata)
	return wrap("upsert auth identity", mapSQLError(err))
}
