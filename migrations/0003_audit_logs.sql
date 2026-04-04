create table if not exists audit_logs (
    id uuid primary key,
    tenant_id uuid not null references tenants(id) on delete cascade,
    actor_user_id uuid null references users(id) on delete set null,
    actor_api_key_id uuid null references api_keys(id) on delete set null,
    action text not null,
    resource_type text not null,
    resource_id text not null,
    message text not null,
    created_at timestamptz not null default now()
);

create index if not exists idx_audit_logs_tenant_created_at on audit_logs (tenant_id, created_at desc);
