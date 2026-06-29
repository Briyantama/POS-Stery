SHELL := /bin/bash
.DEFAULT_GOAL := help

ROOT_DIR := $(shell pwd)
DOCKER_COMPOSE := docker compose -f deploy/docker/docker-compose.yml
GO_SERVICES := auth-service product-service inventory-service sales-service supplier-service customer-service api-gateway

# ─── Help ─────────────────────────────────────────────────────────────────────

.PHONY: help
help:
	@echo ""
	@echo "POS-Stery — available targets"
	@echo ""
	@echo "  Infrastructure:"
	@echo "    make dev           Start infra (Postgres, Redis, NATS, Jaeger, Prometheus)"
	@echo "    make dev-down      Stop infra"
	@echo "    make dev-reset     Destroy volumes + restart (DESTRUCTIVE)"
	@echo ""
	@echo "  Proto / gRPC:"
	@echo "    make proto-gen     buf generate  →  gen/go/ + gen/openapi/"
	@echo "    make proto-lint    buf lint"
	@echo "    make proto-breaking  buf breaking against main"
	@echo ""
	@echo "  Database migrations:"
	@echo "    make migrate-up              Apply all pending migrations"
	@echo "    make migrate-down            Roll back one step per service"
	@echo "    make migrate-create NAME=xxx SERVICE=inventory   Create pair"
	@echo "    make migrate-fresh           Drop all schemas + reapply (DESTRUCTIVE)"
	@echo "    make migrate-validate        Verify up/down pairs exist"
	@echo ""
	@echo "  Go:"
	@echo "    make build-go      Build all services"
	@echo "    make test-go       go test -short ./..."
	@echo "    make test-integration  INTEGRATION=true go test ./..."
	@echo "    make lint-go       golangci-lint"
	@echo "    make tidy          go mod tidy across workspace"
	@echo "    make sqlc-gen      sqlc generate for all services"
	@echo ""
	@echo "  Laravel gateway:"
	@echo "    make install-gateway   composer install"
	@echo "    make test-gateway      vendor/bin/pest"
	@echo "    make dev-gateway       php artisan serve"
	@echo ""
	@echo "  SvelteKit:"
	@echo "    make install-admin     npm install (admin)"
	@echo "    make dev-admin         npm dev (admin)"
	@echo "    make install-cashier   npm install (cashier)"
	@echo "    make dev-cashier       npm dev (cashier)"
	@echo ""
	@echo "  All-in-one:"
	@echo "    make all           proto-gen + migrate-up + build-go + install all frontends"
	@echo "    make ci            Local CI gate: proto-lint + migrate-validate + test-go + test-gateway"
	@echo ""

# ─── Infrastructure ───────────────────────────────────────────────────────────

.PHONY: dev
dev:
	$(DOCKER_COMPOSE) up -d --build
	@echo "Waiting for Postgres..."
	@$(DOCKER_COMPOSE) exec -T postgres pg_isready -U postgres -d pos_db --timeout=30 || true
	@echo "Infra ready. Jaeger UI: http://localhost:16686  Prometheus: http://localhost:9090"

.PHONY: dev-stack
dev-stack:
	$(DOCKER_COMPOSE) up -d --build
	@echo "Waiting for services..."
	@$(DOCKER_COMPOSE) exec -T postgres pg_isready -U postgres -d pos_db --timeout=30 || true
	@echo "Gateway: http://localhost:8000"
	@echo "Admin: http://localhost:5173"
	@echo "Cashier: http://localhost:5174"

.PHONY: dev-down
dev-down:
	$(DOCKER_COMPOSE) down -v --remove-orphans

.PHONY: dev-reset
dev-reset:
	$(DOCKER_COMPOSE) down -v --remove-orphans
	$(DOCKER_COMPOSE) up -d

.PHONY: dev-logs
dev-logs:
	$(DOCKER_COMPOSE) logs -f

# ─── Proto / gRPC ─────────────────────────────────────────────────────────────

.PHONY: proto-gen
proto-gen:
	cd proto && buf dep update && buf generate
	@echo "Proto generated → gen/go/ and gen/openapi/"

.PHONY: proto-lint
proto-lint:
	cd proto && buf lint

.PHONY: proto-breaking
proto-breaking:
	cd proto && buf breaking --against '.git#branch=main'

# ─── Database migrations ──────────────────────────────────────────────────────

.PHONY: migrate-up
migrate-up:
	@for svc in $(GO_SERVICES); do \
		echo "→ migrate up: $$svc"; \
		bash scripts/migrate-helper.sh up $$svc; \
	done

.PHONY: migrate-down
migrate-down:
	@for svc in $(GO_SERVICES); do \
		echo "→ migrate down 1: $$svc"; \
		bash scripts/migrate-helper.sh down $$svc 1; \
	done

.PHONY: migrate-create
migrate-create:
	@[ -n "$(NAME)" ] || (echo "Usage: make migrate-create NAME=xxx SERVICE=inventory"; exit 1)
	@[ -n "$(SERVICE)" ] || (echo "Usage: make migrate-create NAME=xxx SERVICE=inventory"; exit 1)
	bash scripts/migrate-helper.sh create $(SERVICE) $(NAME)

.PHONY: migrate-fresh
migrate-fresh:
	@echo "WARNING: This drops all service schemas. Ctrl-C to cancel, Enter to continue."
	@read
	@for svc in $(GO_SERVICES); do \
		bash scripts/migrate-helper.sh drop $$svc; \
	done
	@$(MAKE) migrate-up

.PHONY: migrate-validate
migrate-validate:
	bash scripts/migrate-validate.sh

# ─── Go ───────────────────────────────────────────────────────────────────────

.PHONY: build-go
build-go:
	@for svc in $(GO_SERVICES); do \
		echo "→ build: services/$$svc"; \
		(cd services/$$svc && go build ./cmd/server/...); \
	done

.PHONY: test-go
test-go:
	@for svc in $(GO_SERVICES); do \
		echo "→ test (short): services/$$svc"; \
		(cd services/$$svc && go test -short ./...); \
	done
	@echo "→ test (short): services/_shared"
	@(cd services/_shared && go test -short ./...)

.PHONY: test-integration
test-integration:
	@for svc in $(GO_SERVICES); do \
		echo "→ test (integration): services/$$svc"; \
		(cd services/$$svc && INTEGRATION=true go test ./...); \
	done

.PHONY: lint-go
lint-go:
	@which golangci-lint > /dev/null || (echo "Install: https://golangci-lint.run/usage/install/"; exit 1)
	@for svc in $(GO_SERVICES); do \
		echo "→ lint: services/$$svc"; \
		(cd services/$$svc && golangci-lint run ./...); \
	done

.PHONY: fmt-go
fmt-go:
	@for svc in $(GO_SERVICES); do \
		echo "→ fmt: services/$$svc"; \
		(cd services/$$svc && go fmt ./...); \
	done

.PHONY: tidy
tidy:
	@# go work sync is skipped: internal modules use replace directives pointing to
	@# local paths, so they cannot be resolved via the module proxy.
	@for svc in _shared $(GO_SERVICES); do \
		echo "→ tidy: services/$$svc"; \
		(cd services/$$svc && go mod tidy -e); \
	done
	@(cd gen/go && go mod tidy -e)

.PHONY: sqlc-gen
sqlc-gen:
	@for svc in $(GO_SERVICES); do \
		if [ -f services/$$svc/sqlc.yaml ]; then \
			echo "→ sqlc generate: services/$$svc"; \
			(cd services/$$svc && sqlc generate); \
		fi \
	done

# ─── Laravel gateway ──────────────────────────────────────────────────────────

.PHONY: install-gateway
install-gateway:
	cd apps/api-gateway && PATH="$$(echo "$$PATH" | tr ':' '\n' | grep -vi 'chocolatey' | tr '\n' ':')" composer install

.PHONY: test-gateway
test-gateway:
	cd apps/api-gateway && vendor/bin/pest

.PHONY: dev-gateway-legacy
dev-gateway-legacy:
	cd apps/api-gateway && php artisan serve --port=8000

.PHONY: dev-gateway
dev-gateway:
	cd services/api-gateway && go run ./cmd/server/...

# ─── SvelteKit apps ───────────────────────────────────────────────────────────

.PHONY: install-admin
install-admin:
	cd apps/admin && npm install

.PHONY: dev-admin
dev-admin:
	cd apps/admin && npm dev

.PHONY: install-cashier
install-cashier:
	cd apps/cashier && npm install

.PHONY: dev-cashier
dev-cashier:
	cd apps/cashier && npm dev --port 5174

# ─── All-in-one ───────────────────────────────────────────────────────────────

.PHONY: all
all: proto-gen migrate-up build-go install-gateway install-admin install-cashier
	@echo "Bootstrap complete."

.PHONY: ci
ci: proto-lint migrate-validate test-go test-gateway
	@echo "CI gate passed."

# ─── Preflight ────────────────────────────────────────────────────────────────

.PHONY: preflight
preflight:
	bash scripts/preflight.sh
