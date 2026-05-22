SHELL := /usr/bin/env bash

.DEFAULT_GOAL := help

GO ?= go
DOCKER ?= docker
COMPOSE_FILE ?= deploy/compose/docker-compose.yaml
LOCAL_ENV_FILE ?= deploy/envs/local.env
COMPOSE := $(DOCKER) compose -f $(COMPOSE_FILE)
BUILD_DIR ?= bin

BUILDABLE_SERVICES := configctl derive execute gateway ingest migrate store writer
COMPOSE_BUILD_SERVICES := configctl derive execute gateway ingest store writer
COMPOSE_RUNTIME_SERVICES := clickhouse configctl derive execute gateway ingest nats store writer

RACCOON_DIR := tools/raccoon-cli
RACCOON_BIN := $(RACCOON_DIR)/target/release/raccoon-cli

.PHONY: help docs bootstrap \
	tidy test test-unit test-integration test-clickhouse test-behavioral test-behavioral-roundtrip \
	build docker-build compose-config up down restart logs ps clean \
	raccoon-build raccoon-test quality-gate quality-gate-ci quality-gate-deep lint lint-go \
	check check-deep verify repo-consistency-check smoke-help smoke smoke-multi smoke-analytical smoke-round-trip smoke-live-stack smoke-activation smoke-composed smoke-operational smoke-restart-recovery smoke-compose-wiring smoke-e2e-multi-binary smoke-failure-isolation smoke-live-dry-run smoke-segmented-compose smoke-spot-ingest smoke-unified-coexistence smoke-spot-venue-live smoke-futures-venue-live smoke-e2e-unified-spot smoke-e2e-unified-futures smoke-endurance-soak smoke-runtime-preflight smoke-backup-restore smoke-backup-offhost \
	ci-analytical ci-smoke ci-preflight ci-wait-ready seed seed-multi seed-spot seed-spot-multi seed-unified seed-unified-multi live live-check live-multi live-multi-check \
	diag coverage-map tdd arch-guard drift-detect snapshot recommend snapshot-diff baseline-drift briefing \
	migrate-up migrate-status migrate-validate \
	ch-backup ch-restore ch-backup-list ch-backup-auto \
	codegen-check codegen-test codegen-integrated codegen-equivalence codegen-validate-all codegen-status \
	stack-up stack-down stack-restart stack-logs \
	po-verify

define RUN_IN_MODULES
	@MODULE='$(MODULE)' ./scripts/utils/for-each-module.sh $(1)
endef

define RUN_GO_TEST
	@set -e; \
	modules=(); \
	if [[ -n "$(MODULE)" ]]; then \
		modules+=("$(MODULE)"); \
	else \
		while IFS= read -r module; do \
			modules+=("$$module"); \
		done < <(./scripts/utils/list-modules.sh); \
	fi; \
	for module in "$${modules[@]}"; do \
		[[ -z "$$module" ]] && continue; \
		packages="$$(cd "$$module" && $(GO) list $(1) ./... 2>/dev/null || true)"; \
		if [[ -z "$$packages" ]]; then \
			echo ">>> $$module: no packages matched"; \
			continue; \
		fi; \
		echo ">>> $$module: $(GO) test $(2) ./..."; \
		(cd "$$module" && $(GO) test $(2) $$packages); \
	done
endef

define REQUIRE_SERVICE
	@if [[ -n "$(SERVICE)" ]]; then \
		case " $(1) " in \
			*" $(SERVICE) "*) ;; \
			*) echo "unsupported SERVICE=$(SERVICE). Supported: $(1)" >&2; exit 1 ;; \
		esac; \
	fi
endef

define LOAD_LOCAL_ENV
	set -a && [ -f $(LOCAL_ENV_FILE) ] && . $(LOCAL_ENV_FILE); set +a;
endef

##@ Help
help: ## Show grouped help and common variables.
	@awk 'BEGIN {FS = ":.*## "; printf "Usage:\n  make <target>\n"} \
		/^##@/ {printf "\n%s\n", substr($$0, 5)} \
		/^[a-zA-Z0-9_.-]+:.*## / {printf "  %-24s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@printf "\nCommon variables\n"
	@printf "  %-24s %s\n" "MODULE=./internal/shared" "Scope module-aware Go targets."
	@printf "  %-24s %s\n" "SERVICE=gateway" "Scope build/logs/restart to one service."
	@printf "  %-24s %s\n" "TARGETS=path1,path2" "Input set for raccoon briefing/recommend."
	@printf "  %-24s %s\n" "SNAP1=before SNAP2=after" "Inputs for snapshot-diff."
	@printf "  %-24s %s\n" "BASELINE=baseline.json" "Input for baseline-drift."
	@printf "  %-24s %s\n" "BASE_URL=http://127.0.0.1:8080" "Override the gateway base URL used by smoke scripts."
	@printf "  %-24s %s\n" "SMOKE_WAIT=180" "Override wait/flush time for smoke scripts."
	@printf "  %-24s %s\n" "FLUSH_WAIT=180" "Legacy wait override for analytical/operational/restart smokes."

docs: ## Show primary docs for workflows, targets, and tooling.
	@printf "Primary docs\n"
	@printf "  README.md\n"
	@printf "  DEVELOPMENT.md\n"
	@printf "  docs/README.md\n"
	@printf "  docs/product/README.md\n"
	@printf "  docs/product/owners.md\n"
	@printf "  docs/development/README.md\n"
	@printf "  docs/development/owners.md\n"
	@printf "  docs/development/workflow.md\n"
	@printf "  docs/development/repository-map.md\n"
	@printf "  docs/development/commands-and-proofs.md\n"
	@printf "  docs/development/stages-and-governance.md\n"
	@printf "  docs/tooling/README.md\n"
	@printf "  docs/architecture/README.md\n"
	@printf "  docs/stages/INDEX.md\n"
	@printf "  docs/archive/README.md\n"

##@ Core Workflow
bootstrap: ## Validate local prerequisites and repository entrypoints for the official workflow.
	@./scripts/bootstrap-check.sh

check: repo-consistency-check quality-gate ## Pre-code guard rail (consistency + fast quality gate).
check-deep: repo-consistency-check quality-gate-deep ## Full validation profile for significant changes; not a substitute for `make smoke*`.
verify: test repo-consistency-check quality-gate lint-go ## Post-change validation: Go tests plus consistency, fast quality gate, and Go lint.
lint: check ## Alias for `make check`.
lint-go: ## Run golangci-lint across all workspace modules.
	@./scripts/lint-go.sh
repo-consistency-check: ## Run lightweight repository consistency checks.
	@./scripts/repository-consistency-check.sh
tdd: $(RACCOON_BIN) ## Show raccoon impact-driven validation guidance for current changes.
	$(RACCOON_BIN) --project-root . change tdd

coverage-map: $(RACCOON_BIN) ## Show raccoon inspection coverage and known gaps.
	$(RACCOON_BIN) --project-root . inspect coverage

briefing: $(RACCOON_BIN) ## Generate a raccoon change briefing for `TARGETS=...`.
	$(RACCOON_BIN) --project-root . change briefing $(TARGETS)

recommend: $(RACCOON_BIN) ## Generate raccoon validation recommendations from diff/baseline analysis.
	$(RACCOON_BIN) --project-root . change recommend $(TARGETS)

##@ Go And Test
tidy: ## Run `go mod tidy` across workspace modules.
	$(call RUN_IN_MODULES,$(GO) mod tidy)

test: ## Run `go test ./...` across workspace modules.
	$(call RUN_GO_TEST,,)

test-unit: test ## Alias for `make test`.

test-integration: ## Run integration tests (requires NATS at localhost:4222).
	@echo "Running integration tests (build tag: integration)..."
	$(call RUN_GO_TEST,-tags=integration,-tags=integration -count=1)

test-clickhouse: ## Run ClickHouse integration tests (requires `CLICKHOUSE_DSN`).
	@echo "Running ClickHouse integration tests (build tag: requireclickhouse)..."
	@if [[ -z "$$CLICKHOUSE_DSN" ]]; then \
		echo "CLICKHOUSE_DSN not set - skipping."; \
		exit 0; \
	fi
	$(call RUN_GO_TEST,-tags=requireclickhouse,-tags=requireclickhouse -count=1)

BEHAVIORAL_PACKAGES := ./internal/actors/scopes/derive/... ./internal/application/strategy/... ./internal/application/risk/...
BEHAVIORAL_PATTERN := ^(TestScenario_|TestActorChain_|TestPositionExposure_|TestDrawdown_|TestScaleConfidence|TestAdjustParam|TestFormatParam)
BEHAVIORAL_ROUNDTRIP_PACKAGES := ./internal/adapters/clickhouse/writerpipeline/...
BEHAVIORAL_ROUNDTRIP_PATTERN := ^TestBehavioralRoundTrip_

test-behavioral: ## Run charter-protected behavioral scenario tests.
	@echo "Running behavioral scenario tests (charter-protected surface)..."
	@$(GO) test $(BEHAVIORAL_PACKAGES) -run '$(BEHAVIORAL_PATTERN)' -v -count=1

test-behavioral-roundtrip: ## Run behavioral round-trip serialization tests.
	@echo "Running behavioral round-trip serialization tests (S255 full-stack proof)..."
	@$(GO) test $(BEHAVIORAL_ROUNDTRIP_PACKAGES) -run '$(BEHAVIORAL_ROUNDTRIP_PATTERN)' -v -count=1

build: ## Build local binaries into `$(BUILD_DIR)/` (optionally `SERVICE=...`).
	@mkdir -p $(BUILD_DIR)
	$(call REQUIRE_SERVICE,$(BUILDABLE_SERVICES))
	@set -e; \
	if [[ -n "$(SERVICE)" ]]; then \
		echo ">>> $(SERVICE)"; \
		$(GO) build -o $(BUILD_DIR)/$(SERVICE) ./cmd/$(SERVICE); \
	else \
		for service in $(BUILDABLE_SERVICES); do \
			echo ">>> $$service"; \
			$(GO) build -o $(BUILD_DIR)/$$service ./cmd/$$service; \
		done; \
	fi

clean: ## Remove local build artifacts and Go caches.
	rm -rf $(BUILD_DIR)
	$(GO) clean -cache -testcache

##@ Runtime Stack
compose-config: ## Render and validate the compose file.
	@$(COMPOSE) config > /dev/null
	@echo "compose config is valid"

up: ## Start the full compose stack, wait for ClickHouse, then apply migrations.
	$(COMPOSE) up -d --build
	@echo "Waiting for ClickHouse before applying migrations..."
	@$(LOAD_LOCAL_ENV) \
	attempts=0; \
	until $(COMPOSE) exec -T clickhouse \
		clickhouse-client --port 9000 --user "$${CLICKHOUSE_USER:-default}" --password "$${CLICKHOUSE_PASSWORD:-clickhouse}" \
		--query "SELECT 1" >/dev/null 2>&1; do \
		attempts=$$((attempts + 1)); \
		if [[ $$attempts -ge 24 ]]; then \
			echo "clickhouse did not become ready in time" >&2; \
			exit 1; \
		fi; \
		sleep 5; \
	done
	@$(MAKE) migrate-up

down: ## Stop the compose stack and remove orphaned containers.
	$(COMPOSE) down --remove-orphans

restart: ## Restart the whole stack or one runtime service via `SERVICE=...`.
	$(call REQUIRE_SERVICE,$(COMPOSE_RUNTIME_SERVICES))
	@if [[ -n "$(SERVICE)" ]]; then \
		$(COMPOSE) restart $(SERVICE); \
	else \
		$(COMPOSE) restart; \
	fi

logs: ## Stream compose logs for the stack or one runtime service.
	$(call REQUIRE_SERVICE,$(COMPOSE_RUNTIME_SERVICES))
	@if [[ -n "$(SERVICE)" ]]; then \
		$(COMPOSE) logs -f --tail=200 $(SERVICE); \
	else \
		$(COMPOSE) logs -f --tail=200; \
	fi

ps: ## Show compose service status.
	$(COMPOSE) ps

docker-build: ## Build compose-backed service images (optionally `SERVICE=...`).
	$(call REQUIRE_SERVICE,$(COMPOSE_BUILD_SERVICES))
	@if [[ -n "$(SERVICE)" ]]; then \
		$(COMPOSE) build $(SERVICE); \
	else \
		$(COMPOSE) build $(COMPOSE_BUILD_SERVICES); \
	fi

stack-up: up ## Alias for `make up`.
stack-down: down ## Alias for `make down`.
stack-restart: restart ## Alias for `make restart`.
stack-logs: logs ## Alias for `make logs`.

live: ## Ergonomic wrapper: build, start, seed, and validate the single-symbol live stack.
	@echo "Live pipeline activation (build + start + seed + validate)..."
	@./scripts/live-pipeline-activate.sh

live-check: ## Ergonomic wrapper: validate an already-running single-symbol stack.
	@echo "Live pipeline check (validate running stack)..."
	@./scripts/live-pipeline-activate.sh --check-only

live-multi: ## Ergonomic wrapper: build, start, seed, and validate the multi-symbol live stack.
	@echo "Live multi-symbol pipeline activation (build+up+seed+validate)..."
	@./scripts/live-pipeline-activate.sh --multi-symbol

live-multi-check: ## Ergonomic wrapper: validate an already-running multi-symbol stack.
	@echo "Live multi-symbol pipeline check (validate running stack)..."
	@./scripts/live-pipeline-activate.sh --multi-symbol --check-only

seed: ## Seed configctl with the default single-symbol configuration (Futures).
	@echo "Seeding configctl (single symbol, source=binancef)..."
	@./scripts/seed-configctl.sh

seed-multi: ## Seed configctl with the default multi-symbol configuration (Futures).
	@echo "Seeding configctl (multi-symbol, source=binancef)..."
	@./scripts/seed-configctl.sh --multi-symbol

seed-spot: ## Seed configctl with the Spot single-symbol configuration.
	@echo "Seeding configctl (single symbol, source=binances)..."
	@SOURCE=binances ./scripts/seed-configctl.sh

seed-spot-multi: ## Seed configctl with the Spot multi-symbol configuration.
	@echo "Seeding configctl (multi-symbol, source=binances)..."
	@SOURCE=binances ./scripts/seed-configctl.sh --multi-symbol

seed-unified: ## S400: Seed configctl with merged Spot+Futures bindings (single config).
	@echo "Seeding configctl (unified, sources=binancef+binances)..."
	@./scripts/seed-configctl.sh --merge

seed-unified-multi: ## S400: Seed configctl with merged Spot+Futures multi-symbol bindings.
	@echo "Seeding configctl (unified multi-symbol, sources=binancef+binances)..."
	@./scripts/seed-configctl.sh --merge --multi-symbol

smoke-help: ## Show smoke/proof selection, prerequisites, and common troubleshooting entrypoints.
	@printf "Operational smoke/proof selection\n"
	@printf "  %-24s %s\n" "make smoke" "Smallest baseline proof. Requires: make up && make seed"
	@printf "  %-24s %s\n" "make smoke-multi" "Broad multi-symbol proof. Requires: make up && make seed-multi"
	@printf "  %-24s %s\n" "make smoke-analytical" "ClickHouse writer/reader proof. Requires: make up && make seed*"
	@printf "  %-24s %s\n" "make smoke-round-trip" "Full persistence round-trip proof (S317). Requires: make up && make seed"
	@printf "  %-24s %s\n" "make smoke-live-stack" "Live stack smoke + gateway verification (S318). Requires: make up && make seed"
	@printf "  %-24s %s\n" "make smoke-activation" "Activation acceptance smoke (S340). Requires: make up && make seed"
	@printf "  %-24s %s\n" "make smoke-composed" "Composed pipeline smoke (S330). No stack needed"
	@printf "  %-24s %s\n" "make smoke-operational" "Process isolation + halt/resume proof. Requires: make up && make seed"
	@printf "  %-24s %s\n" "make smoke-restart-recovery" "Restart/recovery resilience proof. Requires: make up && make seed"
	@printf "  %-24s %s\n" "make smoke-compose-wiring" "S372: Compose wiring validation (boot+streams+consumers). Requires: make up"
	@printf "  %-24s %s\n" "make smoke-e2e-multi-binary" "S373: E2E multi-binary pipeline proof. Requires: make up && make seed"
	@printf "  %-24s %s\n" "make smoke-failure-isolation" "S374: Multi-binary failure isolation proof. Requires: make up && make seed"
	@printf "  %-24s %s\n" "make smoke-live-listening" "S378: Live exchange listening proof. Requires: make up && make seed"
	@printf "  %-24s %s\n" "make smoke-live-dry-run" "S380: E2E live-listen + dry-run proof. Requires: make up && make seed"
	@printf "  %-24s %s\n" "make smoke-segmented-compose" "S394: Segmented Binance compose proof. Requires: make up && make seed"
	@printf "  %-24s %s\n" "make smoke-spot-ingest" "S397: Spot ingest binding seed proof. Requires: make up && make seed-spot"
	@printf "  %-24s %s\n" "make smoke-unified-coexistence" "S402: Single-compose coexistence proof. Requires: make up && make seed-unified"
	@printf "  %-24s %s\n" "make smoke-spot-venue-live" "S405: Spot real venue acceptance/fill proof. No compose needed."
	@printf "  %-24s %s\n" "make smoke-futures-venue-live" "S416: Futures real venue acceptance/fill proof. No compose needed."
	@printf "  %-24s %s\n" "make smoke-e2e-unified-spot" "S408: Unified compose E2E proof for Spot. Requires: make up && make seed-unified"
	@printf "  %-24s %s\n" "make smoke-e2e-unified-futures" "S419: Unified compose E2E proof for Futures. Requires: make up && make seed-unified"
	@printf "  %-24s %s\n" "make smoke-endurance-soak" "S412: Endurance soak and persistence hardening. Phases 1-4: no compose."
	@printf "  %-24s %s\n" "make smoke-runtime-preflight" "S419: Consolidated runtime smoke & Futures preflight. No compose needed."
	@printf "\nCI and preflight\n"
	@printf "  %-24s %s\n" "make ci-smoke" "CI-safe stackless smoke suite (no compose needed)."
	@printf "  %-24s %s\n" "make ci-preflight" "Local pre-push: tests + consistency + quality gate + stackless smoke."
	@printf "  %-24s %s\n" "make ci-wait-ready" "Poll infra readiness before stack-dependent smokes."
	@printf "  %-24s %s\n" "make ci-analytical" "CI analytical gate: unit tests + smoke-analytical."
	@printf "\nCommon overrides\n"
	@printf "  %-24s %s\n" "SMOKE_WAIT=180" "Increase wait/flush timeout for a smoke run."
	@printf "  %-24s %s\n" "BASE_URL=http://host:8080" "Point smokes at a non-default gateway."
	@printf "\nFirst-line diagnosis\n"
	@printf "  %-24s %s\n" "make ps" "Check service/container state."
	@printf "  %-24s %s\n" "make logs SERVICE=gateway" "Inspect HTTP surface failures."
	@printf "  %-24s %s\n" "make logs SERVICE=writer" "Inspect analytical flush failures."
	@printf "  %-24s %s\n" "make diag" "Capture a lightweight runtime snapshot."

smoke: ## Canonical baseline operational proof for the single-symbol slice.
	@echo "Running first-slice E2E smoke test..."
	@./scripts/smoke-first-slice.sh

smoke-multi: ## Canonical broad operational proof for the governed multi-symbol slice.
	@echo "Running multi-symbol E2E smoke test..."
	@./scripts/smoke-multi-symbol.sh

smoke-analytical: ## Canonical specialized proof for the analytical write/read path.
	@echo "Running analytical layer E2E integration proof..."
	@./scripts/smoke-analytical-e2e.sh

smoke-round-trip: ## S317: Full persistence round-trip proof (adapter → NATS → ClickHouse → HTTP).
	@echo "Running full persistence round-trip smoke (S317)..."
	@./scripts/smoke-round-trip.sh

smoke-live-stack: ## Canonical live stack smoke: venue path + persistence + composite + kill-switch (S335).
	@echo "Running canonical live stack smoke (S335)..."
	@./scripts/smoke-live-stack.sh

smoke-activation: ## S340+S341+S342+S343: Activation smoke — acceptance transitions + controlled live path + extended observation.
	@echo "Running activation smoke (S340+S341+S342+S343)..."
	@./scripts/smoke-activation.sh

smoke-operational: ## Canonical specialized proof for OS-process/container operational behavior.
	@echo "Running OS-process operational smoke (S279)..."
	@./scripts/smoke-os-process-operational.sh

smoke-composed: ## S330: Composed pipeline operational smoke (no stack needed).
	@echo "Running composed pipeline smoke (S330)..."
	@./scripts/smoke-composed-pipeline.sh

smoke-restart-recovery: ## Canonical specialized proof for restart/recovery behavior.
	@echo "Running restart and recovery smoke..."
	@./scripts/smoke-restart-recovery.sh

smoke-compose-wiring: ## S372: Compose-level orchestration wiring validation (boot, streams, consumers, connectivity).
	@echo "Running compose-level wiring validation (S372)..."
	@./scripts/smoke-compose-wiring.sh

smoke-e2e-multi-binary: ## S373: End-to-end multi-binary pipeline proof (derive→NATS→execute→store→gateway).
	@echo "Running end-to-end multi-binary pipeline proof (S373)..."
	@./scripts/smoke-e2e-multi-binary.sh

smoke-failure-isolation: ## S374: Multi-binary failure isolation proof (restart one, others survive).
	@echo "Running multi-binary failure isolation proof (S374)..."
	@./scripts/smoke-failure-isolation-multi-binary.sh

smoke-live-listening: ## S378: Compose live exchange listening proof (real trades, paper mode).
	@echo "Running compose live exchange listening proof (S378)..."
	@./scripts/smoke-live-exchange-listening.sh

smoke-live-dry-run: ## S380: End-to-end live-listen + dry-run proof (live data → dry-run fill → read/explain).
	@echo "Running end-to-end live-listen + dry-run proof (S380)..."
	@./scripts/smoke-e2e-live-listen-dry-run.sh

smoke-segmented-compose: ## S394: Compose-level segmented Binance proof (Futures/Spot configs, dry-run, segment isolation).
	@echo "Running segmented compose proof (S394)..."
	@./scripts/smoke-segmented-compose.sh

smoke-spot-ingest: ## S397: Spot ingest binding seed and runtime projection validation.
	@echo "Running Spot ingest binding seed proof (S397)..."
	@./scripts/smoke-spot-ingest-binding.sh

smoke-unified-coexistence: ## S402: Single-compose coexistence proof (Spot+Futures unified config, dry-run, isolation).
	@echo "Running single-compose coexistence proof (S402)..."
	@./scripts/smoke-unified-coexistence.sh

smoke-spot-venue-live: ## S405: Spot real venue acceptance/fill proof (unit tests, no compose needed).
	@echo "Running Spot real venue acceptance/fill proof (S405)..."
	@./scripts/smoke-spot-venue-live.sh

smoke-futures-venue-live: ## S416: Futures real venue acceptance/fill proof (unit tests, no compose needed).
	@echo "Running Futures real venue acceptance/fill proof (S416)..."
	@./scripts/smoke-futures-venue-live.sh

smoke-e2e-unified-spot: ## S408: Unified compose E2E proof for Spot segment. Requires: make up && make seed-unified
	@echo "Running unified compose E2E proof for Spot segment (S408)..."
	@./scripts/smoke-e2e-unified-spot.sh

smoke-e2e-unified-futures: ## S419: Unified compose E2E proof for Futures segment. Requires: make up && make seed-unified
	@echo "Running unified compose E2E proof for Futures segment (S419)..."
	@./scripts/smoke-e2e-unified-futures.sh

smoke-endurance-soak: ## S412: Endurance soak and persistence hardening proof. Phases 1-4: no compose. Phases 5-8: make up && make seed-unified
	@echo "Running endurance soak and persistence hardening proof (S412)..."
	@./scripts/smoke-endurance-soak.sh

smoke-runtime-preflight: ## S419: Consolidated runtime smoke & Futures preflight (stackless). No compose needed.
	@echo "Running consolidated runtime smoke & Futures preflight (S419)..."
	@./scripts/smoke-unified-runtime-preflight.sh

smoke-backup-restore: ## S435: ClickHouse backup/restore proof. Requires ClickHouse running.
	@echo "Running ClickHouse backup/restore proof (S435)..."
	@$(LOAD_LOCAL_ENV) \
	./scripts/smoke-clickhouse-backup-restore.sh

diag: ## Capture a lightweight diagnostic snapshot of the running stack.
	@./scripts/diag-check.sh

ci-smoke: smoke-composed ## CI-safe smoke suite: all stackless smokes runnable without compose infrastructure.

ci-preflight: test repo-consistency-check quality-gate smoke-composed ## Local pre-push preflight: tests + consistency + quality gate + stackless smoke.

ci-analytical: test smoke-analytical ## CI-oriented analytical gate: unit tests plus smoke-analytical.

ci-wait-ready: ## Poll infrastructure readiness (ClickHouse + gateway) before running stack-dependent smokes.
	@./scripts/ci-wait-ready.sh

##@ Architecture And Analysis
arch-guard: $(RACCOON_BIN) ## Enforce architecture layer boundaries via raccoon strategic checks.
	$(RACCOON_BIN) --project-root . check arch

drift-detect: $(RACCOON_BIN) ## Detect cross-layer drift via raccoon strategic checks.
	$(RACCOON_BIN) --project-root . check drift

snapshot: $(RACCOON_BIN) ## Generate a JSON code-intelligence snapshot.
	$(RACCOON_BIN) --project-root . --json snapshot

snapshot-diff: $(RACCOON_BIN) ## Compare two snapshots (`SNAP1=... SNAP2=...`).
	@if [[ -z "$(SNAP1)" || -z "$(SNAP2)" ]]; then \
		echo "Usage: make snapshot-diff SNAP1=before.json SNAP2=after.json"; exit 1; \
	fi
	$(RACCOON_BIN) --project-root . snapshot-diff $(SNAP1) $(SNAP2)

baseline-drift: $(RACCOON_BIN) ## Detect drift from a baseline snapshot (`BASELINE=...`).
	@if [[ -z "$(BASELINE)" ]]; then \
		echo "Usage: make baseline-drift BASELINE=baseline.json"; exit 1; \
	fi
	$(RACCOON_BIN) --project-root . baseline-drift $(BASELINE)

##@ Raccoon CLI
$(RACCOON_BIN): $(shell find $(RACCOON_DIR)/src -type f -name '*.rs' 2>/dev/null) $(RACCOON_DIR)/Cargo.toml
	cargo build --release --manifest-path $(RACCOON_DIR)/Cargo.toml

raccoon-build: $(RACCOON_BIN) ## Build the raccoon-cli release binary.

raccoon-test: ## Run raccoon-cli tests.
	cargo test --manifest-path $(RACCOON_DIR)/Cargo.toml

quality-gate: $(RACCOON_BIN) ## Run the fast quality gate profile.
	$(RACCOON_BIN) --project-root . check gate

quality-gate-ci: $(RACCOON_BIN) ## Run the CI quality gate profile with JSON output.
	$(RACCOON_BIN) --project-root . check gate --profile ci --json

quality-gate-deep: $(RACCOON_BIN) ## Run the deep quality gate profile.
	$(RACCOON_BIN) --project-root . check gate --profile deep

##@ Codegen
codegen-check: ## Verify generated output matches golden snapshots.
	@echo "Running codegen golden equivalence check (all families × all artifacts)..."
	@cd codegen && CODEGEN_ROOT=. $(GO) run . check-all

codegen-test: ## Run codegen unit tests.
	@echo "Running codegen unit tests..."
	@cd codegen && $(GO) test ./... -count=1

codegen-integrated: ## Verify integrated slices match golden snapshots.
	@echo "Running codegen integrated slice verification..."
	@./scripts/codegen-integrated-check.sh

codegen-equivalence: ## Run the cross-artifact codegen equivalence wrapper.
	@echo "Running codegen equivalence checks..."
	@./scripts/codegen-equivalence-check.sh

codegen-validate-all: ## Validate all specs, including cross-spec uniqueness.
	@echo "Running cross-spec validation (per-spec + uniqueness)..."
	@cd codegen && CODEGEN_ROOT=. $(GO) run . validate-all

codegen-status: ## Show governance status of codegen families and integrated slices.
	@echo "=== Codegen Governance Status ==="
	@echo ""
	@echo "Families with specs:"
	@ls -1 codegen/families/*.yaml 2>/dev/null | while read -r f; do \
		name=$$(basename "$$f" .yaml); \
		if grep -qE "^  - family: $$name$$" codegen/integrated.yaml 2>/dev/null; then \
			echo "  $$name  [GOVERNED] (markers + CI gate)"; \
		else \
			echo "  $$name  [MANUAL]   (golden-only, no markers)"; \
		fi; \
	done
	@echo ""
	@echo "Integrated slices (from codegen/integrated.yaml):"
	@awk '/^  - family:/{f=$$3} /artifact:/{gsub(/^ +/,""); sub(/artifact: /,""); print "  " f "/" $$0}' \
		codegen/integrated.yaml 2>/dev/null || echo "  (none)"
	@echo ""

##@ Migrations
migrate-up: ## Apply pending ClickHouse migrations.
	@$(LOAD_LOCAL_ENV) \
	$(GO) run ./cmd/migrate up

migrate-status: ## Show migration status (applied and pending).
	@$(LOAD_LOCAL_ENV) \
	$(GO) run ./cmd/migrate status

migrate-validate: ## Verify checksums of applied migrations.
	@$(LOAD_LOCAL_ENV) \
	$(GO) run ./cmd/migrate validate

##@ ClickHouse Backup
ch-backup: ## Backup all ClickHouse tables (or TABLE=<name> for one).
	@mkdir -p backups/clickhouse
	@$(LOAD_LOCAL_ENV) \
	./scripts/clickhouse-backup.sh $(TABLE)

ch-restore: ## Restore from backup. Usage: make ch-restore BACKUP=mf_20260323_120000 [TABLE=executions]
	@$(LOAD_LOCAL_ENV) \
	./scripts/clickhouse-restore.sh $(BACKUP) $(TABLE)

ch-backup-list: ## List available ClickHouse backups.
	@ls -1 backups/clickhouse/ 2>/dev/null | grep -v '^\.' || echo "(no backups found)"

ch-backup-auto: ## Automated backup + off-host replication. Set BACKUP_OFFHOST_TARGET for replication.
	@mkdir -p backups/clickhouse backups/logs
	@$(LOAD_LOCAL_ENV) \
	./scripts/clickhouse-scheduled-backup.sh

smoke-backup-offhost: ## S440: Automated backup + off-host replication proof. Requires ClickHouse running.
	@echo "Running automated backup + off-host replication proof (S440)..."
	@$(LOAD_LOCAL_ENV) \
	./scripts/smoke-automated-backup-offhost.sh

##@ Post-Operation Verification
po-verify: ## S461: Run automated PO checks. SESSION_ID=<id> --json --save
	@$(LOAD_LOCAL_ENV) \
	./scripts/po-verify.sh $(if $(SESSION_ID),--session-id $(SESSION_ID),) $(PO_FLAGS)
