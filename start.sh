#!/usr/bin/env bash
# start.sh — quick local development bootstrap
# Requires: Go 1.23+, Docker
set -euo pipefail

echo "==> Tidying Go modules..."
go mod tidy

echo "==> Starting Postgres and Redis via Docker Compose..."
docker compose up -d postgres redis

echo "==> Waiting for Postgres to be ready..."
until docker compose exec postgres pg_isready -U clawplane > /dev/null 2>&1; do
  sleep 1
done

echo "==> Running migrations..."
go run ./cmd/migrate

echo "==> Starting API server (background)..."
go run ./cmd/api &
API_PID=$!

echo "==> Starting worker (background)..."
go run ./cmd/worker &
WORKER_PID=$!

echo ""
echo "  API     → http://localhost:8080"
echo "  Swagger → http://localhost:8080/swagger"
echo "  Health  → http://localhost:8080/healthz"
echo "  Ready   → http://localhost:8080/readyz"
echo ""
echo "Press Ctrl+C to stop."

trap 'echo "Stopping..."; kill $API_PID $WORKER_PID 2>/dev/null; docker compose stop postgres redis' INT TERM

wait $API_PID $WORKER_PID
