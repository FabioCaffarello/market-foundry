# Stage S351 — Deployment and Smoke Automation Assessment Report

> PRA-4: Assess deployment automation and smoke automation readiness for venue activation path reproducibility.

## Executive Summary

The Foundry's deployment and smoke automation surface is **mature for local, attended operation** and **partially ready for automated/CI execution**. The local development path (clone → bootstrap → live → smoke) works in 3 commands with no undocumented steps. Nine canonical smoke targets cover the full venue activation path with consistent conventions and override-friendly configuration.

Eight concrete gaps were identified between the current state and reliable unattended automation. None are blockers for attended operation; four are P2 priorities for CI integration. The assessment deliberately stays in evaluation mode — no CI pipeline was built, no remote deployment was added.

## Automation Assessed

### Deployment Automation

| Capability | Automation Level | Verdict |
|-----------|-----------------|---------|
| Stack bring-up (compose + healthchecks + migrations) | `make up` — fully automated | READY |
| Image build (8 services) | `make docker-build` — fully automated | READY |
| Configuration seeding (single/multi-symbol) | `make seed` / `make seed-multi` — fully automated | READY |
| Orchestrated bring-up | `make live` — build+up+seed+validate | READY |
| Stack teardown | `make down` — idempotent | READY |
| Prerequisite validation | `make bootstrap` — checks 8 tools + compose + docs | READY |
| Remote/production deployment | Not implemented | NOT ASSESSED (out of scope) |

### Smoke Automation

| Target | Automation Level | Prerequisites | Verdict |
|--------|-----------------|---------------|---------|
| `make smoke` | Self-contained script | up + seed | READY |
| `make smoke-multi` | Self-contained script | up + seed-multi | READY |
| `make smoke-analytical` | Self-contained script | up + seed | READY |
| `make smoke-round-trip` | Self-contained script | up + seed | READY |
| `make smoke-live-stack` | Self-contained script | up + seed | READY |
| `make smoke-activation` | Self-contained script + trap cleanup | up + seed | READY |
| `make smoke-composed` | Go tests only (no stack) | Go 1.23+ | READY |
| `make smoke-operational` | Self-contained script | up + seed | READY |
| `make smoke-restart-recovery` | Self-contained script | up + seed | READY |

### Guard Rails and Validation

| Target | Purpose | Verdict |
|--------|---------|---------|
| `make check` | Pre-change: consistency + quality gate | READY |
| `make verify` | Post-change: tests + consistency + quality | READY |
| `make arch-guard` | Architecture boundary enforcement | READY |
| `make diag` | Diagnostic snapshot | READY |
| `make repo-consistency-check` | Doc/stage/link consistency | READY |

## Principal Findings

### Finding 1: Local Reproducibility Is High

A developer with Docker, Go 1.23+, and standard CLI tools can go from clone to validated stack in 3 commands. The `make live` orchestrator encapsulates the full bring-up sequence. No undocumented environment variables or manual steps are required for the standard path.

### Finding 2: Smoke Conventions Are Consistent

All 9 smoke scripts share `scripts/utils/lib.sh` for logging, HTTP helpers, error tracking, and compose wrappers. Exit codes are reliable (0 = pass, non-zero = failure). All scripts support `BASE_URL` and timeout overrides. The `make smoke-help` target provides selection guidance.

### Finding 3: CI Integration Has Four P2 Gaps

The path from "working locally" to "working in CI" requires:
1. A machine-readable output format (JSON summary)
2. An aggregate smoke target (`make smoke-suite`)
3. Failure artifact collection (logs + diag on error)
4. An idempotent reset command (`make reset`)

These are well-scoped, low-effort additions (~80 LOC total) that would not inflate the assessment into CI/CD platform engineering.

### Finding 4: Timeout Configuration Is Fragmented

Three different timeout variables (`SMOKE_WAIT`, `FLUSH_WAIT`, `CANDLE_WAIT_MAX`) with different defaults across scripts. This is a friction source for CI tuning but not a correctness issue.

### Finding 5: Environment Variable Landscape Is Manageable

15 environment variables across all scripts, most with sensible defaults. One gap: `CLICKHOUSE_DSN` is undocumented for integration tests. Venue credentials are documented via `execute.env.example` but not referenced by automation.

### Finding 6: Security Posture Is Sound for Local Development

All ports bind to `127.0.0.1`. Containers run with `cap_drop: ALL` and `no-new-privileges:true`. Credentials are local-only (`default/clickhouse`). No secrets in scripts or compose files.

## Gaps and Priorities

| ID | Gap | Priority | Effort | Value |
|----|-----|----------|--------|-------|
| GAP-1 | No machine-readable smoke output | P2 | ~40 LOC | CI integration |
| GAP-2 | No aggregate smoke target | P2 | ~15 LOC | CI + operator ergonomics |
| GAP-3 | No failure artifact collection | P2 | ~20 LOC | CI triage |
| GAP-4 | Timeout variables not centralized | P3 | ~30 LOC | Ergonomics |
| GAP-5 | No port availability pre-check | P3 | ~15 LOC | Ergonomics |
| GAP-6 | CLICKHOUSE_DSN not in local.env | P3 | 1 line | Documentation gap |
| GAP-7 | Venue credentials not in automation | P3 | ~10 LOC | Future live-venue |
| GAP-8 | No idempotent reset+rerun command | P2 | ~15 LOC | CI retry |

Total effort for P2 gaps: ~90 LOC across Makefile and lib.sh.

## Remaining Limits

1. **No remote deployment**: All automation assumes local Docker. This is intentional and appropriate for the current wave.
2. **No container registry**: Images are built locally. Registry push is not needed for assessment or local proof.
3. **No CI pipeline**: Gaps were identified but no pipeline was built. Building a pipeline is a future stage decision.
4. **No smoke result dashboard**: Output is terminal-only. A dashboard would be inflation beyond assessment scope.
5. **S350 monitoring gaps remain open**: Prometheus export, alerting, and consumer lag visibility are tracked separately and are not deployment automation concerns.

## Relationship to Prior Stages

| Stage | What It Assessed | S351 Relationship |
|-------|-----------------|-------------------|
| S347 | Wave charter and scope freeze | S351 executes PRA-4 per charter |
| S348 | Live testnet connectivity | S351 confirms credential handling is documented but not automated |
| S349 | Endurance and sustained activation | S351 confirms smoke timeouts can accommodate extended observation |
| S350 | Monitoring and alertability | S351's gaps are complementary (deployment vs. runtime observability) |

## Preparation for S352

S352 (PRA-5) is the production readiness evidence gate — the formal wave closure that evaluates whether the venue activation capability is ready for promoted operation.

### What S351 Provides to S352

- Deployment automation is **ready for local, attended operation** — no blockers for evidence collection
- Smoke automation covers the **full venue activation path** — 9 canonical surfaces
- **Eight concrete gaps** are documented with priorities and effort estimates
- The gap between "works locally" and "works in CI" is **well-scoped and small** (~90 LOC for P2 gaps)

### What S352 Should Evaluate

1. Whether the cumulative evidence from S348–S351 meets the production readiness bar
2. Whether P2 automation gaps must be resolved before promoting, or can be tracked as known limitations
3. Whether the wave can close with an honest assessment of attended-only operation readiness

## Delivered Artifacts

| Artifact | Path |
|----------|------|
| Deployment and smoke automation assessment | [`../architecture/deployment-automation-and-smoke-automation-assessment.md`](../architecture/deployment-automation-and-smoke-automation-assessment.md) |
| Reproducibility, automation gaps, and operational frictions | [`../architecture/reproducibility-automation-gaps-and-operational-frictions.md`](../architecture/reproducibility-automation-gaps-and-operational-frictions.md) |
| Stage report (this document) | `docs/stages/stage-s351-deployment-and-smoke-automation-assessment-report.md` |
