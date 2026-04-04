create table if not exists users (
    id uuid primary key,
    email text not null unique,
    display_name text not null,
    default_tenant_id uuid null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table if not exists tenants (
    id uuid primary key,
    slug text not null unique,
    name text not null,
    plan text not null default 'free',
    status text not null default 'active',
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table if not exists tenant_members (
    tenant_id uuid not null references tenants(id) on delete cascade,
    user_id uuid not null references users(id) on delete cascade,
    role text not null,
    created_at timestamptz not null default now(),
    primary key (tenant_id, user_id)
);

create table if not exists apps (
    id uuid primary key,
    tenant_id uuid not null references tenants(id) on delete cascade,
    name text not null,
    slug text not null,
    desired_state text not null,
    current_deployment_id uuid null,
    config jsonb not null default '{}'::jsonb,
    created_by uuid not null references users(id),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    unique (tenant_id, slug)
);

create table if not exists deployments (
    id uuid primary key,
    tenant_id uuid not null references tenants(id) on delete cascade,
    app_id uuid not null references apps(id) on delete cascade,
    version integer not null,
    image_ref text not null,
    config_snapshot jsonb not null,
    status text not null,
    status_reason text null,
    backend text not null,
    backend_ref jsonb not null default '{}'::jsonb,
    requested_by uuid not null references users(id),
    started_at timestamptz null,
    finished_at timestamptz null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    unique (app_id, version)
);

create table if not exists deployment_events (
    id uuid primary key,
    deployment_id uuid not null references deployments(id) on delete cascade,
    tenant_id uuid not null references tenants(id) on delete cascade,
    type text not null,
    message text not null,
    created_at timestamptz not null default now()
);
