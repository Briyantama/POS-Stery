.PHONY: up down status build check

# The Go binaries read the process environment only, so hand them .env.
-include .env
export

COMPOSE := docker compose --env-file .env -f deploy/docker-compose.yml
RUN_DIR := .run
PID_FILE := $(RUN_DIR)/api.pid

up: build
	$(COMPOSE) up -d
	cd apps/api && go run ./cmd/migrate up
	mkdir -p $(RUN_DIR)
	cd apps/api && (../../$(RUN_DIR)/api > ../../$(RUN_DIR)/api.log 2>&1 & echo $$! > ../../$(PID_FILE))
	@echo "API started, pid $$(cat $(PID_FILE)), log $(RUN_DIR)/api.log"

down:
	-@if [ -f $(PID_FILE) ]; then kill $$(cat $(PID_FILE)) 2>/dev/null; rm -f $(PID_FILE); fi
	$(COMPOSE) down

status:
	@if [ -f $(PID_FILE) ] && kill -0 $$(cat $(PID_FILE)) 2>/dev/null; then echo "api running, pid $$(cat $(PID_FILE))"; else echo "api not running"; fi
	@curl -fsS http://localhost:$${PORT:-8000}/healthz || echo "healthz unreachable"

build:
	mkdir -p $(RUN_DIR)
	cd apps/api && go build -o ../../$(RUN_DIR)/api ./cmd/api

check:
	bash scripts/check.sh
