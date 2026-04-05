# Clawflux

Clawflux is the open-source control plane for deploying OpenClaw.

It gives OpenClaw a real backend for app lifecycle management, deployments, async workers, API auth, and operator workflows on top of Kubernetes, without dragging in the weight of a full PaaS.

## Why Clawflux

- It is purpose-built for deploying OpenClaw, so the repo story is much clearer than a generic "container control plane."
- It gives you a small, hackable Go codebase instead of a giant platform you need weeks to understand.
- It separates API, scheduling, queueing, and workers cleanly, so you can extend one part without rewriting the whole system.

## Demo

Swagger UI is available at `http://localhost:8080/swagger/` after startup, which makes the current API easy to explore live.

Recommended demo flow for a GIF/video:

1. Start the stack with `make dev`
2. Open `http://localhost:8080/swagger/`
3. Create an API key with `X-User-Email`
4. Create an OpenClaw app definition
5. Trigger an OpenClaw deployment
6. Watch deployment status and events update

If you want a polished OSS landing page, record that flow and save it as `docs/demo.gif`, then replace this line with:

```md
![Clawflux demo](docs/demo.gif)
```

## Quick Start

Prerequisites:

- Go 1.22+
- Docker / Docker Compose

Copy-paste local setup:

```bash
cp .env.example .env
go install github.com/swaggo/swag/cmd/swag@latest
go generate ./cmd/api
make dev
```

That starts:

- API: `http://localhost:8080`
- Swagger UI: `http://localhost:8080/swagger/`
- Health: `http://localhost:8080/healthz`
- Readiness: `http://localhost:8080/readyz`

Create an API key:

```bash
curl -X POST http://localhost:8080/v1/api-keys \
  -H 'Content-Type: application/json' \
  -H 'X-User-Email: alice@example.com' \
  -H 'X-User-Name: Alice' \
  -d '{"name":"local-cli"}'
```

Create an app:

```bash
curl -X POST http://localhost:8080/v1/apps \
  -H 'Content-Type: application/json' \
  -H 'X-User-Email: alice@example.com' \
  -d '{
    "name":"openclaw",
    "slug":"openclaw",
    "config":{
      "image":"ghcr.io/openclaw/openclaw:latest",
      "port":3000,
      "env":{},
      "replicas":1,
      "cpu_request":"250m",
      "memory_request":"256Mi",
      "cpu_limit":"500m",
      "memory_limit":"512Mi",
      "public":true
    }
  }'
```

Create an OpenClaw deployment:

```bash
curl -X POST http://localhost:8080/v1/apps/<app-id>/deployments \
  -H 'X-User-Email: alice@example.com'
```

## What It Solves

Clawflux gives OpenClaw a deployable control-plane backend with:

- Multi-tenant app management
- Deployment orchestration with async workers
- Deployment event history
- API key auth for automation and backend clients
- Admin and audit endpoints for operators
- Kubernetes-first control-plane workflows

It is especially useful if you want to build:

- A hosted OpenClaw offering
- A self-hosted OpenClaw deployment backend
- A control plane around OpenClaw environments and automation

## Architecture

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

See [docs/architecture.md](docs/architecture.md) for the fuller breakdown.

## Developer Experience

- Swagger generation: `go generate ./cmd/api`
- Swagger UI: `http://localhost:8080/swagger/`
- Full local stack: `make dev`
- Infra only: `make infra-up`
- Tests: `make test`
- Lint: `make lint`

## API Surface

Current endpoints include:

- `/v1/me`
- `/v1/auth/providers`
- `/v1/auth/medium/login`
- `/v1/auth/medium/callback`
- `/v1/api-keys`
- `/v1/apps`
- `/v1/apps/{appID}/deployments`
- `/v1/deployments/{deploymentID}`
- `/v1/deployments/{deploymentID}/events`
- `/v1/admin/summary`
- `/v1/admin/audit-logs`

## Current Status

Clawflux is a strong backend scaffold, not a finished hosted platform yet.

Already here:

- PostgreSQL and in-memory repositories
- Redis-backed job queue
- Worker-driven deployment flow
- Deployment retries and sync jobs
- Swagger docs and live API UI
- Admin, audit, health, and API key endpoints

Still worth building next:

- Real Kubernetes reconciliation with `client-go`
- Secret management
- Runtime logs and streaming deployment feedback
- Dead-letter queue behavior
- A dedicated web UI

## Contributing

Contributions are welcome if you want to help push Clawflux toward a real OSS control plane.

Good first areas:

- Kubernetes backend reconciliation
- richer deployment status syncing
- UI/dashboard
- DX improvements for local setup
- tests around queue and worker behavior
