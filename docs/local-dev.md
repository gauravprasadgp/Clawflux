# Local Development

## Dependencies

- Go 1.22+
- Docker / Docker Compose
- PostgreSQL 16
- Redis 7

## Start backing services

```bash
docker compose up -d postgres redis
```

## Environment

```bash
cp .env.example .env
```

## Apply schema

```bash
go run ./cmd/migrate
```

## Start API

```bash
go run ./cmd/api
```

Code-first Swagger generation:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
go generate ./cmd/api
```

Generated files will be written to:

```text
docs/swagger/
```

## Start worker

```bash
go run ./cmd/worker
```

## Development auth

When `DEVELOPMENT_AUTH=true`, requests without auth headers are treated as `developer@local`.

You can also set:

```text
X-User-Email: alice@example.com
X-User-Name: Alice
```

## API key flow

Create an API key:

```bash
curl -X POST http://localhost:8080/v1/api-keys \
  -H 'Content-Type: application/json' \
  -H 'X-User-Email: alice@example.com' \
  -d '{"name":"local-cli"}'
```

Use the returned secret:

```bash
curl http://localhost:8080/v1/me \
  -H 'X-API-Key: cc_your_secret'
```

## Admin summary

```bash
curl http://localhost:8080/v1/admin/summary \
  -H 'X-User-Email: admin@example.com' \
  -H 'X-Platform-Admin: true'
```
