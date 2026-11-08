# Build context is the PARENT of the repo, so the stery-kit replace resolves:
#   docker build -f POS-Stery/apps/api/Dockerfile -t pos-api .   (run from stery-suite/, or CI's checkout root)
FROM golang:1.25-alpine AS build
WORKDIR /src/POS-Stery/apps/api
COPY stery-kit/go /src/stery-kit/go
COPY POS-Stery/apps/api/ ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM alpine:3.22
RUN adduser -D -u 10001 app
COPY --from=build /out/api /usr/local/bin/api
COPY --from=build /out/migrate /usr/local/bin/migrate
USER 10001
EXPOSE 8000
ENTRYPOINT ["/usr/local/bin/api"]
