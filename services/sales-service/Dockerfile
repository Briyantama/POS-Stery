FROM golang:1.24-alpine AS builder

WORKDIR /workspace

# Copy go workspace files
COPY go.work go.work.sum ./
COPY gen/go/go.mod gen/go/go.sum ./gen/go/
COPY services/_shared/go.mod services/_shared/go.sum ./services/_shared/
COPY services/sales-service/go.mod services/sales-service/go.sum ./services/sales-service/

# Download dependencies
RUN go work sync

# Copy source
COPY gen/go/ ./gen/go/
COPY services/_shared/ ./services/_shared/
COPY services/sales-service/ ./services/sales-service/

# Build
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /sales-service \
    ./services/sales-service/cmd/server/

# ── Runtime image ─────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /sales-service /sales-service

EXPOSE 8084

ENTRYPOINT ["/sales-service"]
