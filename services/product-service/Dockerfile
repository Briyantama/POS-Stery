FROM golang:1.24-alpine AS builder

WORKDIR /workspace

# Copy go workspace files
COPY go.work go.work.sum ./
COPY gen/go/go.mod gen/go/go.sum ./gen/go/
COPY services/_shared/go.mod services/_shared/go.sum ./services/_shared/
COPY services/product-service/go.mod services/product-service/go.sum ./services/product-service/

# Download dependencies
RUN go work sync

# Copy source
COPY gen/go/ ./gen/go/
COPY services/_shared/ ./services/_shared/
COPY services/product-service/ ./services/product-service/

# Build
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /product-service \
    ./services/product-service/cmd/server/

# ── Runtime image ─────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /product-service /product-service

EXPOSE 8082

ENTRYPOINT ["/product-service"]
