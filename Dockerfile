# syntax=docker/dockerfile:1.6

# =============================================================================
# CrawlIQ Dockerfile
#
# Multi-stage build:
#   1. builder  - compile the API binary against a pinned Go version
#   2. runtime  - minimal image with only the compiled binary + CA certs
# =============================================================================

# -----------------------------------------------------------------------------
# Stage 1 — build
# -----------------------------------------------------------------------------
FROM golang:1.25-alpine AS builder

# Tools needed for CGO-free builds of pgx and other native deps.
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

# Cache go.mod/go.sum first so dependency downloads are cached separately
# from the source — gives much faster rebuilds on code-only changes.
COPY go.mod go.sum ./
RUN go mod download

# Now copy the rest of the source and build a static binary.
COPY . .

# CGO_ENABLED=0 keeps the resulting binary fully static so the runtime
# stage can use scratch / distroless without needing glibc.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/crawliq \
    ./cmd/api

# -----------------------------------------------------------------------------
# Stage 2 — runtime
# -----------------------------------------------------------------------------
FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates tzdata wget && \
    addgroup -S crawliq && adduser -S crawliq -G crawliq

WORKDIR /app
COPY --from=builder /out/crawliq /app/crawliq

# Default config lives in /app/config; copy the example as a starting
# point — operators can mount a real one to override.
COPY config/config.example.yaml /app/config/config.yaml

USER crawliq
EXPOSE 8080

# wget is used for the healthcheck loop below so we don't need to install
# curl on the slim alpine image.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --quiet --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/crawliq"]