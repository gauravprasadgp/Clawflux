alter table deployments
    add column if not exists repair_attempts integer not null default 0;

create index if not exists idx_deployments_active_reconcile
    on deployments (updated_at)
    where status in ('queued', 'provisioning', 'running', 'degraded', 'recovering', 'deleting');
