go mod tidy

docker run --name clawplane-postgres \
  -e POSTGRES_USER=admin \
  -e POSTGRES_PASSWORD=secret \
  -e POSTGRES_DB=clawplane \
  -p 5432:5432 \
  -d postgres

docker run --name clawplane-redis \
  -p 6379:6379 \
  -d redis

go run cmd/api/main.go