# ClawPlane Backend

Kubernetes-only backend scaffold for multi-tenant OpenClaw control plane.

## Processes

- `cmd/api`: REST API for apps and deployments.
- `cmd/worker`: async deployment reconciler.
- `Redis`: queue transport using a minimal RESP client.
- `Kubernetes backend`: namespace-per-tenant naming and deployment refs.

## Request flow

1. `POST /v1/apps`
2. `POST /v1/apps/{appID}/deployments`
3. API persists desired state and enqueues `deployment.create`
4. Worker dequeues job and calls Kubernetes backend
5. Deployment status moves `queued -> provisioning -> running|failed`

## Current implementation choices

- Production runtime uses PostgreSQL repositories when `REPOSITORY_DRIVER=postgres`.
- In-memory repositories are still available for isolated development or tests.
- Auth middleware provisions a local actor from `X-User-Email` during development.
- Backend implementation is intentionally K8s-only.
- Deployment scheduling is now a first-class service boundary between API and queue.

## Next backend milestones

- Swap the lightweight K8s backend stub for `client-go` reconciliation.
- Add secret management and runtime log streaming.
- Add retries with delayed queue semantics.
