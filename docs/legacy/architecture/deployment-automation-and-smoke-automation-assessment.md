# Deployment Automation and Smoke Automation Assessment

> Stage S351 — PRA-4 assessment artifact.
> Evaluates current automation maturity for deployment and smoke execution in the venue activation path.

## 1. Deployment Automation Shape

### 1.1 Current Stack Lifecycle

The Foundry stack lifecycle is fully automated for local development:

| Step | Command | Automation Level | Notes |
|------|---------|-----------------|-------|
| Prerequisite check | `make bootstrap` | Automated | Validates bash, curl, docker, git, go, make, python3, cargo |
| Image build | `make docker-build` | Automated | Builds all 8 service images locally |
| Stack bring-up | `make up` | Automated | Compose up + ClickHouse wait + migration apply |
| Configuration seed | `make seed` / `make seed-multi` | Automated | Draft-validate-compile-activate lifecycle |
| Stack teardown | `make down` | Automated | Compose down with orphan removal |
| Orchestrated bring-up | `make live` | Automated | Build + up + seed + validate in one command |

### 1.2 Compose Topology

Nine services orchestrated via `deploy/compose/docker-compose.yaml`:

- **Infrastructure**: nats (2.10.18), clickhouse (24.8.8)
- **Application**: configctl, gateway, ingest, derive, store, execute, writer

All services bind to `127.0.0.1` (no external exposure), use `cap_drop: ALL` and `no-new-privileges:true`, have structured healthchecks, and follow a dependency chain enforced by compose `depends_on` with `condition: service_healthy`.

### 1.3 Configuration Layer

- **Service configs**: 8 JSONC files in `deploy/configs/` with a reference document
- **Environment**: `deploy/envs/local.env` (ClickHouse credentials only)
- **Credential example**: `deploy/configs/execute.env.example` for venue API keys
- **Defaults**: Hardcoded in scripts/lib.sh (`BASE_URL`, timeouts, service lists)

## 2. Smoke Automation Shape

### 2.1 Canonical Smoke Targets

Nine canonical smoke surfaces, each with a dedicated script and make target:

| Target | Script | Stack Required | Prerequisites | Scope |
|--------|--------|---------------|---------------|-------|
| `make smoke` | smoke-first-slice.sh | Yes | up + seed | Single-symbol baseline E2E |
| `make smoke-multi` | smoke-multi-symbol.sh | Yes | up + seed-multi | Multi-symbol x multi-timeframe |
| `make smoke-analytical` | smoke-analytical-e2e.sh | Yes | up + seed | ClickHouse writer/reader |
| `make smoke-round-trip` | smoke-round-trip.sh | Yes | up + seed | Full persistence round-trip |
| `make smoke-live-stack` | smoke-live-stack.sh | Yes | up + seed | Live stack + gateway |
| `make smoke-activation` | smoke-activation.sh | Yes | up + seed | Activation control surface |
| `make smoke-composed` | smoke-composed-pipeline.sh | No | Go 1.23+ | Go tests only (no containers) |
| `make smoke-operational` | smoke-os-process-operational.sh | Yes | up + seed | OS-process/halt-resume |
| `make smoke-restart-recovery` | smoke-restart-recovery.sh | Yes | up + seed | Restart resilience |

### 2.2 Shared Infrastructure

All stack-dependent smokes share:

- **`scripts/utils/lib.sh`**: Common logging (pass/fail/info/warn/phase), HTTP helpers, error tracking, banner formatting, service lists, and compose wrappers
- **Exit codes**: 0 for pass, non-zero for any recorded failure via `ERRORS` counter
- **Timeouts**: Configurable via `SMOKE_WAIT`, `FLUSH_WAIT`, `CANDLE_WAIT_MAX`, `HEALTH_WAIT_MAX`
- **Output**: Colored terminal output with structured phase markers (suppressible via `NO_COLOR`)

### 2.3 Orchestration Layers

| Layer | Command | What It Does |
|-------|---------|--------------|
| Script | `./scripts/smoke-*.sh` | Direct script execution with env var overrides |
| Make target | `make smoke-*` | Canonical entrypoint, forwards to script |
| Live orchestrator | `make live` | Build + up + seed + validate pipeline |
| Diagnostics | `make diag` | Readiness/status/diag snapshot across services |

### 2.4 Guard Rails

Pre-change and post-change validation:

- `make check` = repo-consistency-check + quality-gate (pre-change)
- `make verify` = tests + consistency + quality-gate (post-change)
- `make tdd` = impact-driven validation guidance
- `make arch-guard` / `make drift-detect` = architecture boundary enforcement

## 3. Automation Maturity Assessment

### 3.1 What Is Already Automatable Today

| Capability | Status | Confidence |
|-----------|--------|-----------|
| Full stack bring-up from zero | Automatable | HIGH — `make live` covers build+up+seed+validate |
| Individual smoke execution | Automatable | HIGH — each `make smoke-*` is self-contained |
| Stack teardown | Automatable | HIGH — `make down` is idempotent |
| Prerequisite validation | Automatable | HIGH — `make bootstrap` checks all tools |
| Repository consistency | Automatable | HIGH — `make check` runs without manual input |
| Configuration seeding | Automatable | HIGH — `make seed`/`make seed-multi` are scripted |
| Diagnostic capture | Automatable | HIGH — `make diag` queries all services |

### 3.2 What Requires Manual Intervention

| Step | Friction | Severity |
|------|----------|----------|
| Docker daemon must be running | Operator must start Docker Desktop or dockerd | LOW |
| Port availability (4222, 8080, 8123, 8222, 9000) | No pre-check for port conflicts | LOW |
| Venue credentials for live paths | Must be set manually in environment | MEDIUM |
| Smoke result interpretation | Terminal output only, no machine-readable report | MEDIUM |
| Sequential smoke ordering | No "run all smokes" aggregate target | LOW |
| Image registry push | No registry; images are local-only | LOW (assessment scope) |
| Remote deployment | All automation assumes local Docker | LOW (assessment scope) |

### 3.3 Ergonomic Strengths

1. **Single-command paths**: `make live` from zero to validated stack
2. **Consistent conventions**: All smokes follow the same lib.sh patterns
3. **Override-friendly**: `BASE_URL`, `SMOKE_WAIT`, `SERVICE` variables allow tuning without script edits
4. **Self-documenting**: `make smoke-help` provides selection guide with prerequisites
5. **Safety exits**: smoke-activation.sh auto-restores gate state on exit
6. **Healthcheck-gated**: Compose dependency chain prevents premature smoke execution

## 4. Environment Variable Landscape

### 4.1 Documented and Defaulted

| Variable | Default | Source |
|----------|---------|--------|
| `BASE_URL` | `http://127.0.0.1:8080` | lib.sh |
| `SMOKE_WAIT` | 75-180s (varies by smoke) | per-script |
| `FLUSH_WAIT` | 120s | per-script |
| `HEALTH_WAIT_MAX` | 120s | lib.sh |
| `CANDLE_WAIT_MAX` | 90s | lib.sh |
| `CLICKHOUSE_USER` | `default` | local.env |
| `CLICKHOUSE_PASSWORD` | `clickhouse` | local.env |
| `SOURCE` | `binancef` | seed-configctl.sh |
| `NO_COLOR` | (unset) | lib.sh |

### 4.2 Undocumented or Implicit

| Variable | Where Used | Issue |
|----------|-----------|-------|
| `CLICKHOUSE_DSN` | `make test-clickhouse` | Not in local.env; must be set manually |
| Venue API credentials | execute.env.example | Example exists but no automation references it |
| `COMPOSE_FILE` | Makefile | Hardcoded, not overridable via env |

## 5. Reproducibility Verdict

### 5.1 Local Development: HIGH Reproducibility

Given Docker, Go 1.23+, and standard CLI tools, a developer can go from clone to validated stack with:

```bash
make bootstrap   # verify prerequisites
make live         # build + start + seed + validate
make smoke        # baseline proof
```

This path is reliable, documented, and requires no undocumented knowledge.

### 5.2 Automated/CI Execution: MEDIUM Reproducibility

The same commands would work in a CI environment with Docker support, but several gaps exist:

- No machine-readable output format (CI needs parseable results)
- No aggregate "run all relevant smokes" target
- Timeout values may need tuning for CI runners (slower than local)
- No artifact collection (logs, diagnostic snapshots on failure)
- Port binding to 127.0.0.1 assumes single-tenant runner

### 5.3 Remote/Production Deployment: NOT ASSESSED

This is explicitly out of scope for the current wave. No remote deployment automation exists, and none is expected at this stage.
