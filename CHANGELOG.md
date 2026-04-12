# Changelog

---

## v0.1.0 — Initial Public Release

> **April 12, 2026** — First public release of Clawflux.

This is the first open-source release of Clawflux — a production-grade control plane for deploying [OpenClaw](https://github.com/openclaw/openclaw) on Kubernetes.

It started as an internal project to solve a real problem: giving OpenClaw a real deployment backend instead of shell scripts and manual kubectl commands. After building it out end-to-end — multi-tenant K8s deployments, async workers, a React admin UI, audit logs, and a clean API — we're opening it up.

---

### What's in this release

#### Core backend

- **Multi-tenant app management** — create and manage OpenClaw app definitions per user, with full config: image, replicas, CPU/memory requests and limits, environment, domain
- **Async deployment pipeline** — API queues jobs to Redis, background worker picks them up, drives them through to Kubernetes, retries on failure, syncs status back
- **Deployment lifecycle** — create, retry, cancel, delete. Full status tracking: `requested → queued → provisioning → running / failed`
- **Deployment events** — timestamped event log per deployment (provisioning steps, errors, sync updates)
- **Audit logs** — every API action recorded with actor, resource, and timestamp

#### Kubernetes backend

Full reconciliation with no external operators or CRDs required:

- **Namespace per tenant** — each user gets an isolated `tenant-<id>` namespace
- **Deployment + Service + Ingress** — created and updated idempotently on every deploy
- **Workspace PVC** — persistent volume for OpenClaw's workspace directory, survives redeployments
- **ConfigMap** — injects `AGENTS.md` and `settings.json` into the container
- **Secret** — stores `OPENCLAW_GATEWAY_TOKEN` and provider API keys (e.g. `OPENAI_API_KEY`)
- **Network policy** — baseline egress policy applied to tenant namespaces
- **Status sync** — worker polls K8s deployment conditions and maps them back to Clawflux status

#### React admin UI

A standalone Vite + React app in `frontend/`:

- **Dashboard** — live table of all OpenClaw instances across all tenants, status badges, auto-refresh
- **Instance detail** — deployment info, K8s namespace/ref, event log, retry/cancel/delete actions
- **Deploy** — full form to provision a user and launch an OpenClaw instance in one click
- **Users** — provision platform users, view audit log
- Admin identity stored in `localStorage`, injected as `X-Platform-Admin` headers

#### API

Full REST API with Swagger UI at `/swagger/`:

| Endpoint | Description |
|---|---|
| `GET /v1/me` | Current actor |
| `GET /v1/apps` | List apps for tenant |
| `POST /v1/apps` | Create app |
| `GET /v1/apps/{id}` | Get app |
| `PATCH /v1/apps/{id}` | Update app |
| `POST /v1/apps/{id}/deployments` | Create deployment |
| `GET /v1/apps/{id}/deployments` | List deployments |
| `GET /v1/deployments/{id}` | Get deployment |
| `GET /v1/deployments/{id}/events` | List deployment events |
| `POST /v1/deployments/{id}/retry` | Retry deployment |
| `POST /v1/deployments/{id}/cancel` | Cancel deployment |
| `POST /v1/deployments/{id}/delete` | Queue deletion |
| `POST /v1/api-keys` | Create API key |
| `GET /v1/api-keys` | List API keys |
| `DELETE /v1/api-keys/{id}` | Revoke API key |
| `GET /v1/admin/summary` | Platform stats |
| `GET /v1/admin/instances` | All instances across all tenants |
| `GET /v1/admin/audit-logs` | Audit log |
| `POST /v1/admin/users` | Provision user |
| `POST /v1/admin/openclaw/deploy` | Admin one-shot deploy |
| `GET /healthz` | Liveness |
| `GET /readyz` | Readiness |

#### Developer experience

- **`make dev`** — one command starts Postgres, Redis, migrations, API server, worker, and Vite dev server
- **`make dev-api`** — Go backend only
- **`make dev-ui`** — Vite frontend only
- **In-memory repository** — full in-memory implementation of every repo interface for fast local dev and tests without a database
- **Swagger auto-generation** — `make swag` regenerates docs from handler annotations
- **GitHub Actions** — build and test CI on every push to `main`

---

### Known limitations in v0.1.0

These are real gaps — not hidden, not glossed over. They're the next things to fix:

- **Auth is dev-mode only** — `X-User-Email` + `X-Platform-Admin: true` headers are trusted without verification. Fine for internal/trusted networks, not for public-facing deployments. Real token auth is on the roadmap.
- **No log streaming** — deployment status is tracked but pod logs aren't surfaced through the API yet
- **No dead-letter queue** — jobs that exhaust retries are currently just marked failed; there's no separate dead-letter view or alerting
- **Kubernetes only** — Docker Compose, ECS, Fly.io, and other backends are planned but not yet implemented
- **Test coverage is thin** — only `internal/worker` and `internal/worker/handlers` have tests; the services and repositories layers need more coverage

---

### Getting started

```bash
git clone https://github.com/gauravprasadgp/clawflux.git
cd clawflux
cp .env.example .env
go install github.com/swaggo/swag/cmd/swag@latest
make dev
```

API: http://localhost:8080  
UI: http://localhost:5173  
Swagger: http://localhost:8080/swagger/

---

### Contributing

If any of the known limitations above bother you, that's the best place to start. See [CONTRIBUTING.md](CONTRIBUTING.md) or the **Contributing** section in the README for how to get involved.

---

*Built with Go, React, Postgres, Redis, and a real K8s cluster to test against.*
