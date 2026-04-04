# ClawPlane Backend

Kubernetes-only backend scaffold for a multi-tenant OpenClaw control plane.

## Runtime Shape

```text
                         ┌─────────────────────┐
                         │   Admin / Future UI │
                         └──────────┬──────────┘
                                    │
                         ┌──────────▼──────────┐
                         │      HTTP API       │
                         │   cmd/api + router  │
                         └──────────┬──────────┘
                                    │
        ┌───────────────┬───────────┼───────────────┬───────────────┐
        │               │           │               │               │
 ┌──────▼──────┐ ┌──────▼──────┐ ┌──▼───────────┐ ┌─▼────────────┐ ┌▼─────────────┐
 │ Auth / IAM  │ │ App Service │ │ Deploy Svc   │ │ Admin / Audit│ │ Health Svc   │
 │ OAuth + key │ │ apps config │ │ desired state│ │ ops endpoints│ │ readiness    │
 └──────┬──────┘ └──────┬──────┘ └──────┬───────┘ └────┬─────────┘ └────┬─────────┘
        │               │                │               │                │
        └───────────────┴────────────────┼───────────────┴────────────────┘
                                         │
                              ┌──────────▼──────────┐
                              │ Repositories        │
                              │ Postgres or memory  │
                              └──────────┬──────────┘
                                         │
                              ┌──────────▼──────────┐
                              │ Scheduler Service   │
                              │ enqueue job intent  │
                              └──────────┬──────────┘
                                         │
                              ┌──────────▼──────────┐
                              │ Redis Job Queue     │
                              │ minimal RESP client │
                              └──────────┬──────────┘
                                         │
                              ┌──────────▼──────────┐
                              │ Worker              │
                              │ cmd/worker          │
                              └──────────┬──────────┘
                                         │
                              ┌──────────▼──────────┐
                              │ Deployment Backend  │
                              │ Kubernetes adapter  │
                              └─────────────────────┘
```

## Main Processes

- `cmd/api`: REST API for apps, deployments, auth, admin, audit, and health endpoints.
- `cmd/worker`: async deployment worker for create and delete jobs.
- `Redis`: queue transport using a minimal RESP client.
- `PostgreSQL`: production persistence for users, tenants, apps, deployments, events, audit logs, and API keys.
- `memory` repositories: isolated local development and tests.
- `Kubernetes backend`: deployment backend abstraction for K8s operations.

## Service Boundaries

- `AuthService`: local actor provisioning, OAuth callback handling, tenant membership lookup, and API key authentication.
- `APIKeyService`: machine auth for backend clients and automation.
- `AppService`: create, read, update, and list app definitions.
- `DeploymentService`: persist desired deployment state, append deployment events, and route jobs to the scheduler.
- `SchedulerService`: narrow boundary between services and async queue transport.
- `AdminService`: platform summary endpoints for operators.
- `AuditService`: tenant audit log recording and listing.
- `HealthService`: readiness checks for API dependencies.

## Request Flow

1. A client calls `POST /v1/apps` or `POST /v1/apps/{appID}/deployments`.
2. The HTTP layer authenticates the actor and dispatches to the service layer.
3. Services persist app or deployment state through repositories.
4. Deployment creation appends an event and enqueues `deployment.create`.
5. The worker dequeues the job and calls the Kubernetes deployment backend.
6. Deployment status moves `queued -> provisioning -> running|failed`.
7. Admin, audit, and readiness endpoints expose operational state to operators.

## Already Present In The Codebase

- Repository abstraction with PostgreSQL and in-memory implementations.
- Explicit scheduler boundary between deployment service and queue transport.
- Deployment event history in addition to deployment status.
- Admin summary and audit log endpoints.
- Readiness checks for database and Redis.
- OAuth provider plumbing plus API key authentication.

## Missing Bits For A Production-Grade Control Plane

- Real Kubernetes reconciliation:
  replace the current lightweight backend stub with `client-go`-backed apply, delete, and status reconciliation.
- Delayed retries and dead-letter handling:
  the Redis queue currently provides basic enqueue and blocking dequeue only.
- Continuous sync loop:
  reconcile actual backend state back into deployment status even when no new API request arrives.
- Secret management:
  support app secrets separately from plain app config and avoid treating all env as regular config.
- Runtime log and event streaming:
  surface pod logs and live deployment progress for operators and users.
- Stronger authn/authz:
  replace development-oriented shortcuts like `X-Platform-Admin: true` and placeholder OAuth state handling.
- Dedicated frontend:
  the repo currently provides backend APIs but not the Admin/UI application shown in high-level sketches.

## Current Implementation Choices

- Production runtime uses PostgreSQL repositories when `REPOSITORY_DRIVER=postgres`.
- In-memory repositories remain available for isolated development or tests.
- Auth middleware provisions a local actor from `X-User-Email` during development.
- API clients can authenticate with `X-API-Key`.
- Platform admin endpoints currently use `X-Platform-Admin: true` as a temporary operator guardrail.
- Backend implementation is intentionally K8s-only.
- Deployment scheduling is a first-class service boundary between API and queue.

## Recommended Next Additions

1. Replace the Kubernetes backend stub with a real reconciler and persist backend refs plus readiness details.
2. Add retry scheduling, attempt tracking, and dead-letter behavior to the queue/worker path.
3. Introduce deployment sync jobs so status can recover from worker restarts and backend drift.
4. Add secrets and logs as dedicated service boundaries instead of folding them into app config.
5. Build a frontend or operator console on top of the existing admin, audit, app, and deployment APIs.
