FROM golang:1.24-alpine AS builder

WORKDIR /workspace

# Copy go workspace files
COPY go.work go.work.sum ./
COPY gen/go/go.mod gen/go/go.sum ./gen/go/
COPY services/_shared/go.mod services/_shared/go.sum ./services/_shared/
COPY services/auth-service/go.mod services/auth-service/go.sum ./services/auth-service/

# Download dependencies
RUN go work sync

# Copy source
COPY gen/go/ ./gen/go/
COPY services/_shared/ ./services/_shared/
COPY services/auth-service/ ./services/auth-service/

# Build
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /auth-service \
    ./services/auth-service/cmd/server/

# ── Runtime image ─────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /auth-service /auth-service

EXPOSE 8081

ENTRYPOINT ["/auth-service"]
