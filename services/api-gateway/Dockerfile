FROM golang:1.25-alpine AS builder

WORKDIR /workspace

# Copy workspace files first for module resolution
COPY go.work go.work.sum ./
COPY gen/go/ gen/go/
COPY services/_shared/ services/_shared/
COPY services/api-gateway/ services/api-gateway/

RUN cd services/api-gateway && \
    go build -o /app/api-gateway ./cmd/server/

FROM alpine:3.20

RUN addgroup -S app && adduser -S app -G app

COPY --from=builder /app/api-gateway /app/api-gateway

USER app

EXPOSE 8000

ENTRYPOINT ["/app/api-gateway"]
