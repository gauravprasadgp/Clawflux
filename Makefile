# Clawflux Makefile
# Usage: make <target>

BINARY_API     := bin/api
BINARY_WORKER  := bin/worker
BINARY_MIGRATE := bin/migrate

VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS  := -s -w \
  -X github.com/gauravprasad/clawcontrol/internal/services.BuildVersion=$(VERSION) \
  -X github.com/gauravprasad/clawcontrol/internal/services.BuildCommit=$(COMMIT)

GO      := go
GOFLAGS := -trimpath

.PHONY: all build api worker migrate \
        run run-worker migrate-up dev infra-up infra-down \
        test test-race test-cover \
        lint fmt vet swag \
        docker-build docker-up docker-down docker-logs \
        clean help

# ── Default ───────────────────────────────────────────────────────────────────
all: build

# ── Swagger doc generation ────────────────────────────────────────────────────
# Automatically installs swag and uses full path (no PATH issues)
SWAG := $(shell go env GOPATH)/bin/swag

swag:
	@echo "==> Ensuring swag is installed..."
	@test -x $(SWAG) || ( \
		echo "Installing swag..."; \
		go install github.com/swaggo/swag/cmd/swag@latest; \
	)

	@echo "==> Generating Swagger docs..."
	$(SWAG) init \
		--parseInternal \
		--generalInfo cmd/api/main.go \
		--output docs/swagger

	@echo "Swagger docs written to docs/swagger/"

# ── Build (regenerates swagger docs first) ────────────────────────────────────
build: swag api worker migrate

api:
	@mkdir -p bin
	$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_API) ./cmd/api

worker:
	@mkdir -p bin
	$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_WORKER) ./cmd/worker

migrate:
	@mkdir -p bin
	$(GO) build $(GOFLAGS) -o $(BINARY_MIGRATE) ./cmd/migrate

# ── Run individual processes ──────────────────────────────────────────────────
run: api
	./$(BINARY_API)

run-worker: worker
	./$(BINARY_WORKER)

migrate-up: migrate
	./$(BINARY_MIGRATE)

# ── Local dev: infra + migrate + api + worker in one command ──────────────────
# Starts Postgres and Redis via Docker Compose, waits for them to be healthy,
# runs migrations, then runs the API and worker in the background.
# Ctrl+C stops both processes and optionally leaves infra running.
dev: build infra-up migrate-up
	@echo ""
	@echo "  API     -> http://localhost:8080"
	@echo "  UI      -> http://localhost:5173  (Vite, hot reload)"
	@echo "  Swagger -> http://localhost:8080/swagger"
	@echo "  Health  -> http://localhost:8080/healthz"
	@echo ""
	@echo "Press Ctrl+C to stop everything (infra keeps running)."
	@echo ""
	@cd frontend && npm install --silent 2>/dev/null; cd ..
	@trap 'kill $$API_PID $$WORKER_PID $$UI_PID 2>/dev/null; echo "Stopped."; exit 0' INT TERM; \
	  ./$(BINARY_API) & API_PID=$$!; \
	  ./$(BINARY_WORKER) & WORKER_PID=$$!; \
	  (cd frontend && npm run dev -- --host) & UI_PID=$$!; \
	  wait $$API_PID $$WORKER_PID $$UI_PID

# Run only the Go backend
dev-api: build infra-up migrate-up
	./$(BINARY_API)

# Run only the Vite frontend (API must already be running on :8080)
dev-ui:
	cd frontend && npm install --silent && npm run dev

# ── Infrastructure only (Postgres + Redis) ────────────────────────────────────
infra-up:
	@echo "==> Starting Postgres and Redis..."
	docker compose up -d postgres redis
	@echo "==> Waiting for Postgres..."
	@until docker compose exec postgres pg_isready -U clawflux > /dev/null 2>&1; do sleep 1; done
	@echo "==> Waiting for Redis..."
	@until docker compose exec redis redis-cli ping > /dev/null 2>&1; do sleep 1; done
	@echo "==> Infrastructure ready."

infra-down:
	docker compose stop postgres redis

# ── Test ──────────────────────────────────────────────────────────────────────
test:
	$(GO) test ./... -count=1

test-race:
	$(GO) test -race ./... -count=1

test-cover:
	$(GO) test ./... -coverprofile=coverage.out -covermode=atomic
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# ── Quality ───────────────────────────────────────────────────────────────────
fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

lint: fmt vet
	@which golangci-lint > /dev/null 2>&1 || \
		(echo "golangci-lint not found — install from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...

# ── Docker (full stack) ───────────────────────────────────────────────────────
docker-build:
	docker build \
		--target api \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		-t clawflux-api:$(VERSION) \
		-t clawflux-api:latest \
		.
	docker build \
		--target worker \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		-t clawflux-worker:$(VERSION) \
		-t clawflux-worker:latest \
		.

docker-up:
	VERSION=$(VERSION) COMMIT=$(COMMIT) docker compose up --build -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f api worker

# ── Clean ─────────────────────────────────────────────────────────────────────
clean:
	rm -rf bin/ coverage.out coverage.html

# ── Help ──────────────────────────────────────────────────────────────────────
help:
	@echo ""
	@echo "Clawflux — available targets:"
	@echo ""
	@echo "  Local development"
	@echo "  -----------------"
	@echo "  dev            Start infra, migrate, then run api + worker (Ctrl+C to stop)"
	@echo "  infra-up       Start Postgres and Redis only"
	@echo "  infra-down     Stop Postgres and Redis"
	@echo "  migrate-up     Build and run database migrations"
	@echo "  run            Build and run the API server"
	@echo "  run-worker     Build and run the worker"
	@echo ""
	@echo "  Build"
	@echo "  -----"
	@echo "  build          Build all binaries (regenerates Swagger docs first)"
	@echo "  api            Build the API binary only"
	@echo "  worker         Build the worker binary only"
	@echo "  migrate        Build the migrate binary only"
	@echo "  swag           Regenerate Swagger docs from annotations"
	@echo ""
	@echo "  Testing & quality"
	@echo "  -----------------"
	@echo "  test           Run all tests"
	@echo "  test-race      Run tests with race detector"
	@echo "  test-cover     Run tests and open HTML coverage report"
	@echo "  lint           Format, vet, and run golangci-lint"
	@echo "  fmt            Run go fmt"
	@echo "  vet            Run go vet"
	@echo ""
	@echo "  Docker (full stack)"
	@echo "  -------------------"
	@echo "  docker-build   Build Docker images (api + worker)"
	@echo "  docker-up      Start full stack with docker compose"
	@echo "  docker-down    Stop full stack"
	@echo "  docker-logs    Tail api + worker logs"
	@echo ""
	@echo "  clean          Remove build artifacts"
	@echo ""
	@echo "  One-time setup:"
	@echo "    go install github.com/swaggo/swag/cmd/swag@latest"
	@echo "    brew install golangci-lint  (or see golangci-lint.run)"
	@echo ""
