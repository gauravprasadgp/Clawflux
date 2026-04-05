-- Performance indexes for ClawPlane
-- Safe to run multiple times (IF NOT EXISTS).

-- Apps: fast lookup by tenant
CREATE INDEX IF NOT EXISTS idx_apps_tenant_id
    ON apps (tenant_id);

-- Apps: slug uniqueness enforced per tenant
CREATE UNIQUE INDEX IF NOT EXISTS uidx_apps_tenant_slug
    ON apps (tenant_id, slug);

-- Deployments: list by app (most common query)
CREATE INDEX IF NOT EXISTS idx_deployments_app_id
    ON deployments (app_id);

-- Deployments: filter by tenant
CREATE INDEX IF NOT EXISTS idx_deployments_tenant_id
    ON deployments (tenant_id);

-- Deployments: monitor queued/provisioning jobs (used by scheduler)
CREATE INDEX IF NOT EXISTS idx_deployments_status
    ON deployments (status)
    WHERE status IN ('queued', 'provisioning', 'deleting');

-- Deployment events: ordered fetch per deployment
CREATE INDEX IF NOT EXISTS idx_deployment_events_deployment_id
    ON deployment_events (deployment_id, created_at DESC);

-- API keys: fast auth path (hash lookup is the hot path)
CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash
    ON api_keys (key_hash);

-- API keys: list by tenant
CREATE INDEX IF NOT EXISTS idx_api_keys_tenant_id
    ON api_keys (tenant_id);

-- Audit logs: list by tenant ordered by time
CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_created
    ON audit_logs (tenant_id, created_at DESC);

-- Auth identities: provider callback lookup
CREATE UNIQUE INDEX IF NOT EXISTS uidx_auth_identities_provider_uid
    ON auth_identities (provider, provider_user_id);
