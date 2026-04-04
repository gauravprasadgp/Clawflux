create table if not exists auth_identities (
    id uuid primary key,
    user_id uuid not null references users(id) on delete cascade,
    provider text not null,
    provider_user_id text not null,
    access_token_encrypted text null,
    refresh_token_encrypted text null,
    metadata jsonb not null default '{}'::jsonb,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    unique (provider, provider_user_id)
);

create table if not exists api_keys (
    id uuid primary key,
    tenant_id uuid not null references tenants(id) on delete cascade,
    user_id uuid not null references users(id) on delete cascade,
    name text not null,
    key_prefix text not null,
    key_hash text not null,
    last_used_at timestamptz null,
    expires_at timestamptz null,
    revoked_at timestamptz null,
    created_at timestamptz not null default now()
);

create index if not exists idx_api_keys_tenant_id on api_keys (tenant_id);
create index if not exists idx_deployments_tenant_app on deployments (tenant_id, app_id);
create index if not exists idx_deployment_events_lookup on deployment_events (tenant_id, deployment_id, created_at);
