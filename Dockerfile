# ────────────────────────────────────────────────────────────────────────────
# Clawflux — multi-stage Dockerfile
#
# Stages:
#   builder  – compiles both binaries (api + worker) with version injection
#   api      – minimal runtime image for the HTTP server
#   worker   – minimal runtime image for the queue consumer
# ────────────────────────────────────────────────────────────────────────────

ARG GO_VERSION=1.23
ARG ALPINE_VERSION=3.20

# ── Stage 1: builder ─────────────────────────────────────────────────────────
FROM golang:${GO_VERSION}-alpine AS builder

# ca-certificates is needed at runtime for outbound TLS (OAuth callbacks, etc.)
RUN apk add --no-cache ca-certificates git tzdata

WORKDIR /src

# Download dependencies in a cacheable layer (only re-runs when go.sum changes)
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Capture build metadata
ARG VERSION=dev
ARG COMMIT=unknown

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
      -trimpath \
      -ldflags="-s -w \
        -X github.com/gauravprasad/clawcontrol/internal/services.BuildVersion=${VERSION} \
        -X github.com/gauravprasad/clawcontrol/internal/services.BuildCommit=${COMMIT}" \
      -o /out/api ./cmd/api

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
      -trimpath \
      -ldflags="-s -w \
        -X github.com/gauravprasad/clawcontrol/internal/services.BuildVersion=${VERSION} \
        -X github.com/gauravprasad/clawcontrol/internal/services.BuildCommit=${COMMIT}" \
      -o /out/worker ./cmd/worker

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
      -trimpath \
      -ldflags="-s -w" \
      -o /out/migrate ./cmd/migrate

# ── Stage 2: api ─────────────────────────────────────────────────────────────
FROM alpine:${ALPINE_VERSION} AS api

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S clawflux && adduser -S -G clawflux clawflux

WORKDIR /app
COPY --from=builder /out/api      ./api
COPY --from=builder /out/migrate  ./migrate

USER clawflux
EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/healthz || exit 1

ENTRYPOINT ["./api"]

# ── Stage 3: worker ──────────────────────────────────────────────────────────
FROM alpine:${ALPINE_VERSION} AS worker

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S clawflux && adduser -S -G clawflux clawflux

WORKDIR /app
COPY --from=builder /out/worker ./worker

USER clawflux

ENTRYPOINT ["./worker"]
