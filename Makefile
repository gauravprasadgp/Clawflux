# ClawPlane Makefile
# Usage: make <target>

BINARY_API    := bin/api
BINARY_WORKER := bin/worker
BINARY_MIGRATE := bin/migrate

VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS  := -s -w \
  -X github.com/gauravprasad/clawcontrol/internal/services.BuildVersion=$(VERSION) \
  -X github.com/gauravprasad/clawcontrol/internal/services.BuildCommit=$(COMMIT)

GO       := go
GOFLAGS  := -trimpath

.PHONY: all build api worker migrate run run-worker \
        test test-race lint fmt vet \
        docker-build docker-up docker-down \
        migrate-up clean help

# ── Default ───────────────────────────────────────────────────────────────────
all: build

# ── Build ─────────────────────────────────────────────────────────────────────
build: api worker migrate

api:
	@mkdir -p bin
	$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_API) ./cmd/api

worker:
	@mkdir -p bin
	$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_WORKER) ./cmd/worker

migrate:
	@mkdir -p bin
	$(GO) build $(GOFLAGS) -o $(BINARY_MIGRATE) ./cmd/migrate

# ── Run (local, requires .env) ────────────────────────────────────────────────
run: api
	./$(BINARY_API)

run-worker: worker
	./$(BINARY_WORKER)

migrate-up: migrate
	./$(BINARY_MIGRATE)

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

# ── Docker ────────────────────────────────────────────────────────────────────
docker-build:
	docker build \
		--target api \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		-t clawplane-api:$(VERSION) \
		-t clawplane-api:latest \
		.
	docker build \
		--target worker \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		-t clawplane-worker:$(VERSION) \
		-t clawplane-worker:latest \
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

# ── Help ─────────────────────────────────────────────────────────────────────
help:
	@echo ""
	@echo "ClawPlane — available targets:"
	@echo ""
	@echo "  build          Build all binaries (api, worker, migrate)"
	@echo "  run            Build and run the API server"
	@echo "  run-worker     Build and run the worker"
	@echo "  migrate-up     Run database migrations"
	@echo ""
	@echo "  test           Run all tests"
	@echo "  test-race      Run tests with race detector"
	@echo "  test-cover     Run tests and open HTML coverage report"
	@echo "  lint           Format, vet, and run golangci-lint"
	@echo ""
	@echo "  docker-build   Build Docker images (api + worker)"
	@echo "  docker-up      Start full stack with docker compose"
	@echo "  docker-down    Stop full stack"
	@echo "  docker-logs    Tail api + worker logs"
	@echo ""
	@echo "  clean          Remove build artifacts"
	@echo ""
