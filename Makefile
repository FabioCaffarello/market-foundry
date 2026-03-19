
SHELL := /usr/bin/env bash

GO ?= go
DOCKER ?= docker
COMPOSE_FILE ?= deploy/compose/docker-compose.yaml
COMPOSE := $(DOCKER) compose -f $(COMPOSE_FILE)
BUILD_DIR ?= bin
BUILDABLE_SERVICES := configctl derive execute gateway ingest store

RACCOON_DIR := tools/raccoon-cli
RACCOON_BIN := $(RACCOON_DIR)/target/release/raccoon-cli

.DEFAULT_GOAL := help

define RUN_IN_MODULES
	@MODULE='$(MODULE)' ./scripts/utils/for-each-module.sh $(1)
endef

.PHONY: help tidy test test-integration build docker-build compose-config up down restart logs ps clean \
       raccoon-build raccoon-test quality-gate quality-gate-ci quality-gate-deep \
       check check-deep verify smoke smoke-multi seed seed-multi \
       coverage-map tdd arch-guard drift-detect snapshot recommend snapshot-diff baseline-drift briefing

help:
	@echo "Targets:"
	@echo "  make tidy                 - run go mod tidy in workspace modules"
	@echo "  make test                 - run go test ./... in workspace modules"
	@echo "  make test-integration     - run integration tests (requires embedded NATS)"
	@echo "  make build                - build local service binaries into $(BUILD_DIR)/"
	@echo "  make docker-build         - build docker images for local services"
	@echo "  make compose-config       - render and validate the compose file"
	@echo "  make up                   - start the stack (nats + configctl + gateway + ingest + derive + store + execute)"
	@echo "  make down                 - stop the compose stack"
	@echo "  make restart              - restart the whole stack or SERVICE=<name>"
	@echo "  make logs                 - stream logs for the whole stack or SERVICE=<name>"
	@echo "  make ps                   - show compose service status"
	@echo "  make clean                - remove local build artifacts and Go caches"
	@echo ""
	@echo "Workflow (recommended):"
	@echo "  make seed                 - seed configctl with single symbol (btcusdt)"
	@echo "  make seed-multi           - seed configctl with multi-symbol (btcusdt + ethusdt)"
	@echo "  make smoke                - E2E smoke test (single symbol)"
	@echo "  make smoke-multi          - E2E smoke test (2 symbols × 2 timeframes)"
	@echo "  make check                - pre-code guard rail (quality-gate fast)"
	@echo "  make verify               - post-change: Go tests + quality-gate"
	@echo "  make check-deep           - full validation"
	@echo "  make coverage-map         - show quality coverage map and gaps"
	@echo "  make tdd                  - TDD guide: what to validate for your changes"
	@echo "  make arch-guard           - architecture layer boundary check"
	@echo "  make drift-detect         - cross-layer drift detection"
	@echo "  make snapshot             - golden snapshot of code intelligence (JSON)"
	@echo "  make snapshot-diff        - compare two snapshots (SNAP1= SNAP2=)"
	@echo "  make baseline-drift       - detect drift against baseline (BASELINE=)"
	@echo "  make recommend            - smart recommendations from diff/baseline"
	@echo ""
	@echo "Quality (raccoon-cli):"
	@echo "  make quality-gate         - fast static checks (local dev, pre-commit)"
	@echo "  make quality-gate-ci      - CI pipeline checks (JSON output)"
	@echo "  make quality-gate-deep    - full validation"
	@echo "  make raccoon-build        - build raccoon-cli release binary"
	@echo "  make raccoon-test         - run raccoon-cli tests"
	@echo ""
	@echo "Optional:"
	@echo "  MODULE=./internal/shared  - scope tidy/test to one Go module"
	@echo "  SERVICE=gateway           - scope build/docker-build/logs/restart to one service"

tidy:
	$(call RUN_IN_MODULES,$(GO) mod tidy)

test:
	@modules=(); \
	if [[ -n "$(MODULE)" ]]; then \
		modules+=("$(MODULE)"); \
	else \
		while IFS= read -r module; do \
			modules+=("$$module"); \
		done < <(./scripts/utils/list-modules.sh); \
	fi; \
	for module in "$${modules[@]}"; do \
		[[ -z "$$module" ]] && continue; \
		echo ">>> $$module: $(GO) test ./..."; \
		packages="$$(cd "$$module" && $(GO) list ./... 2>/dev/null || true)"; \
		if [[ -z "$$packages" ]]; then \
			echo "no packages to test"; \
			continue; \
		fi; \
		(cd "$$module" && $(GO) test $$packages); \
	done

test-integration:
	@echo "Running integration tests (build tag: integration)..."
	@modules=(); \
	if [[ -n "$(MODULE)" ]]; then \
		modules+=("$(MODULE)"); \
	else \
		while IFS= read -r module; do \
			modules+=("$$module"); \
		done < <(./scripts/utils/list-modules.sh); \
	fi; \
	for module in "$${modules[@]}"; do \
		[[ -z "$$module" ]] && continue; \
		packages="$$(cd "$$module" && $(GO) list -tags=integration ./... 2>/dev/null || true)"; \
		if [[ -z "$$packages" ]]; then \
			continue; \
		fi; \
		echo ">>> $$module: $(GO) test -tags=integration ./..."; \
		(cd "$$module" && $(GO) test -tags=integration -count=1 $$packages); \
	done

build:
	@mkdir -p $(BUILD_DIR)
	@if [[ -n "$(SERVICE)" ]]; then \
		case " $(BUILDABLE_SERVICES) " in \
			*" $(SERVICE) "*) ;; \
			*) echo "unsupported SERVICE=$(SERVICE). Supported: $(BUILDABLE_SERVICES)" >&2; exit 1 ;; \
		esac; \
		echo ">>> $(SERVICE)"; \
		$(GO) build -o $(BUILD_DIR)/$(SERVICE) ./cmd/$(SERVICE); \
	else \
		for service in $(BUILDABLE_SERVICES); do \
			echo ">>> $$service"; \
			$(GO) build -o $(BUILD_DIR)/$$service ./cmd/$$service; \
		done; \
	fi

docker-build:
	@if [[ -n "$(SERVICE)" ]]; then \
		case " $(BUILDABLE_SERVICES) " in \
			*" $(SERVICE) "*) ;; \
			*) echo "unsupported SERVICE=$(SERVICE). Supported: $(BUILDABLE_SERVICES)" >&2; exit 1 ;; \
		esac; \
		$(COMPOSE) build $(SERVICE); \
	else \
		$(COMPOSE) build $(BUILDABLE_SERVICES); \
	fi

compose-config:
	@$(COMPOSE) config > /dev/null
	@echo "compose config is valid"

up:
	$(COMPOSE) up -d --build

down:
	$(COMPOSE) down --remove-orphans

restart:
	@if [[ -n "$(SERVICE)" ]]; then \
		$(COMPOSE) restart $(SERVICE); \
	else \
		$(COMPOSE) restart; \
	fi

logs:
	@if [[ -n "$(SERVICE)" ]]; then \
		$(COMPOSE) logs -f --tail=200 $(SERVICE); \
	else \
		$(COMPOSE) logs -f --tail=200; \
	fi

ps:
	$(COMPOSE) ps

clean:
	rm -rf $(BUILD_DIR)
	$(GO) clean -cache -testcache

# --- raccoon-cli (quality tooling) ---

$(RACCOON_BIN): $(shell find $(RACCOON_DIR)/src -type f -name '*.rs' 2>/dev/null) $(RACCOON_DIR)/Cargo.toml
	cargo build --release --manifest-path $(RACCOON_DIR)/Cargo.toml

raccoon-build: $(RACCOON_BIN)

raccoon-test:
	cargo test --manifest-path $(RACCOON_DIR)/Cargo.toml

quality-gate: $(RACCOON_BIN)
	$(RACCOON_BIN) --project-root . quality-gate

quality-gate-ci: $(RACCOON_BIN)
	$(RACCOON_BIN) --project-root . quality-gate --profile ci --json

quality-gate-deep: $(RACCOON_BIN)
	$(RACCOON_BIN) --project-root . quality-gate --profile deep

# --- workflow targets (developer-facing) ---

check: quality-gate

check-deep: quality-gate-deep

verify: test quality-gate

smoke:
	@echo "Running first-slice E2E smoke test..."
	@./scripts/smoke-first-slice.sh

smoke-multi:
	@echo "Running multi-symbol E2E smoke test..."
	@./scripts/smoke-multi-symbol.sh

seed:
	@echo "Seeding configctl (single symbol)..."
	@./scripts/seed-configctl.sh

seed-multi:
	@echo "Seeding configctl (multi-symbol)..."
	@./scripts/seed-configctl.sh --multi-symbol

coverage-map: $(RACCOON_BIN)
	$(RACCOON_BIN) --project-root . coverage-map

tdd: $(RACCOON_BIN)
	$(RACCOON_BIN) --project-root . tdd

briefing: $(RACCOON_BIN)
	$(RACCOON_BIN) --project-root . briefing $(TARGETS)

arch-guard: $(RACCOON_BIN)
	$(RACCOON_BIN) --project-root . arch-guard

drift-detect: $(RACCOON_BIN)
	$(RACCOON_BIN) --project-root . drift-detect

snapshot: $(RACCOON_BIN)
	$(RACCOON_BIN) --project-root . --json snapshot

recommend: $(RACCOON_BIN)
	$(RACCOON_BIN) --project-root . recommend $(TARGETS)

snapshot-diff: $(RACCOON_BIN)
	@if [[ -z "$(SNAP1)" || -z "$(SNAP2)" ]]; then \
		echo "Usage: make snapshot-diff SNAP1=before.json SNAP2=after.json"; exit 1; \
	fi
	$(RACCOON_BIN) --project-root . snapshot-diff $(SNAP1) $(SNAP2)

baseline-drift: $(RACCOON_BIN)
	@if [[ -z "$(BASELINE)" ]]; then \
		echo "Usage: make baseline-drift BASELINE=baseline.json"; exit 1; \
	fi
	$(RACCOON_BIN) --project-root . baseline-drift $(BASELINE)
