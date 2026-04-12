# Contributing to Clawflux

Thanks for your interest. Clawflux is early-stage and contributions are genuinely welcome — the codebase is small, the patterns are consistent, and there are real things to build.

---

## Before you start

- Check [open issues](https://github.com/gauravprasadgp/clawflux/issues) to see if someone's already working on it
- For significant changes, open an issue first to discuss the approach — saves everyone time
- Small fixes (typos, doc improvements, bug fixes) can go straight to a PR

---

## Setup

```bash
git clone https://github.com/gauravprasadgp/clawflux.git
cd clawflux
cp .env.example .env
go install github.com/swaggo/swag/cmd/swag@latest
make dev
```

This starts everything: Postgres, Redis, API server, background worker, and the Vite dev server.

- API: http://localhost:8080
- UI: http://localhost:5173
- Swagger: http://localhost:8080/swagger/

---

## Where things live

```
cmd/              → entrypoints (api, worker, migrate)
internal/
  api/http/       → handlers, router, middleware — touch this for new endpoints
  services/       → business logic — touch this for new features
  domain/         → types and interfaces — touch this when adding new concepts
  backends/       → deployment backends — add new ones here
  repositories/   → postgres/ and memory/ — add new repos here
  worker/handlers → job handlers — add new job types here
frontend/src/
  pages/          → one file per page
  components/     → shared components
  api.js          → all fetch calls live here
```

---

## Good first contributions

| Task | Where to look | Difficulty |
|---|---|---|
| Add unit tests for `services/` | `internal/services/` | Easy |
| Add unit tests for `repositories/memory/` | `internal/repositories/memory/` | Easy |
| Improve frontend loading states / error handling | `frontend/src/pages/` | Easy |
| Add `GET /v1/admin/dead-letters` endpoint | `internal/services/`, `internal/api/http/` | Medium |
| Implement Docker Compose deployment backend | `internal/backends/` | Medium |
| Add real-time deployment log streaming | `internal/api/http/`, K8s client | Medium |
| Add webhook notifications on deployment status change | `internal/services/` | Medium |
| Implement Fly.io deployment backend | `internal/backends/` | Medium |

---

## Adding a deployment backend

This is one of the most impactful contributions you can make. The interface is small:

```go
type DeploymentBackend interface {
    Name() string
    Submit(ctx context.Context, req BackendDeployRequest) (*BackendStatus, error)
    Delete(ctx context.Context, ref BackendRef) error
    GetStatus(ctx context.Context, ref BackendRef) (*BackendStatus, error)
}
```

Steps:
1. Create `internal/backends/<name>/backend.go`
2. Implement the three methods above
3. Add a config flag in `internal/app/config.go` to select it
4. Wire it in `internal/app/bootstrap.go`
5. Add a row to the backends table in `README.md`

See `internal/backends/kubernetes/backend.go` as the reference (~600 lines, well-commented).

---

## Code style

- Run `go fmt ./...` and `go vet ./...` before committing
- Follow existing patterns — the codebase is intentionally consistent
- Keep handlers thin — business logic belongs in services, not handlers
- Errors should use `domain.ErrNotFound`, `domain.ErrForbidden`, etc. — don't invent new error types
- New domain concepts go in `internal/domain/types.go`

For frontend:
- No CSS frameworks — plain CSS using the existing variables in `index.css`
- Keep components small and focused
- New API calls go in `frontend/src/api.js`

---

## Running tests

```bash
make test          # all tests
make test-race     # with race detector
make test-cover    # with HTML coverage report
```

---

## Submitting a PR

1. Fork the repo, create a branch: `git checkout -b feat/your-thing`
2. Make your changes
3. Run `make test` and `make lint` — both should pass
4. If you changed API annotations, run `make swag` to regenerate Swagger docs
5. Open a PR with:
   - What you changed and why
   - How to test it
   - Any known trade-offs or follow-up work

---

## Questions?

Open an issue or start a discussion. We'd rather answer a question than have someone go down the wrong path for hours.
