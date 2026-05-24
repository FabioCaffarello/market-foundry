# Runtime invariants

**Status:** Active
**Date:** 2026-05-24
**Owner:** Repository maintainer
**Authority tier:** T1 — Canonical
([`../AUTHORITY.md`](../AUTHORITY.md))
**Relates to:** [`../TRUTH-MAP.md`](../TRUTH-MAP.md),
[`../decisions/`](../decisions/README.md), `Makefile`

---

## Purpose

The **Top-N runtime properties** that `make verify` and the smoke
suite validate every time they run. Each invariant maps:

- the **rule** (what must always be true)
- the **code enforcement** (where in the source the rule is
  encoded)
- the **guardrail** (smoke target or analyzer that detects a
  violation)
- the **failure mode** (what breaks first when the invariant
  drifts)
- the **rollback action** (the canonical correction)

This document is the **runtime-evidence layer** for
`ARCHITECTURE.md` foundational principles. ARCHITECTURE says
"layer sovereignty is non-negotiable"; runtime-invariants tells
you *which analyzer* catches the violation and *which* `make`
target runs that analyzer.

---

## Invariant catalogue (Top-10)

### I1 — Single writer per stream / KV bucket

| Field | Value |
|---|---|
| **Rule** | Every JetStream stream and every NATS KV bucket has exactly **one** publishing/writing binary or actor. No exceptions. |
| **Authority** | [ADR-0008](../decisions/0008-single-writer-invariant.md) |
| **Code enforcement** | `internal/adapters/nats/nats{configctl,observation,evidence,signal,decision,strategy,risk,execution}/registry.go:DefaultRegistry` — each registry declares one writer per stream; cross-binary writes do not exist in the code base. |
| **Guardrail** | `raccoon-cli` `topology-doctor` analyzer (`tools/raccoon-cli/src/analyzers/topology_doctor.rs`) runs in `make quality-gate` / `make verify`. |
| **Failure mode** | Two writers on a stream → racing publishes, KV monotonicity guard cannot reason, downstream consumers see interleaved state. |
| **Rollback** | Revert the second-writer change; restore the stream to its single owner. If the second writer was intentional (a new flow), restructure as message-passing (publish to a new stream owned by binary A, consumed by binary B). |

---

### I2 — Layer sovereignty enforced statically

| Field | Value |
|---|---|
| **Rule** | Imports flow inward only: `domain → application → adapters → actors → interfaces → cmd`. No outward or sideways imports. |
| **Authority** | [ADR-0005](../decisions/0005-layer-sovereignty.md), [ADR-0004](../decisions/0004-raccoon-cli-static-enforcement.md) |
| **Code enforcement** | `tools/raccoon-cli/src/analyzers/arch_guard.rs` — `LAYERS` const + `is_allowed_dependency()`. |
| **Guardrail** | `make arch-guard` (also via `make verify`). |
| **Failure mode** | Domain depends on adapter → cannot substitute infrastructure; tests need mocks; refactor cost increases over time. |
| **Rollback** | Move the cross-layer dependency to a port (interface defined in `application/` and implemented in `adapters/`). |

---

### I3 — Forward-only ClickHouse migrations

| Field | Value |
|---|---|
| **Rule** | `deploy/migrations/*.sql` are applied in numeric order; the `_migrations` table records what has run; **rollback is a new forward migration, never a revert**. |
| **Authority** | [ADR-0003](../decisions/0003-clickhouse-analytical.md) |
| **Code enforcement** | `cmd/migrate/engine/runner.go:Runner` reads the `_migrations` metadata table and applies pending migrations in numeric order. |
| **Guardrail** | `make migrate-up` is the only sanctioned entry point; CI applies migrations as part of `smoke-analytical`. |
| **Failure mode** | A migration applied out-of-order or skipped → schema drift between environments → reads return malformed rows or fail. |
| **Rollback** | Write a new migration that corrects the drift (e.g., `008_undo_column_x.sql`). Never edit a previously applied migration. |

---

### I4 — Gateway loopback default (HTTP auth = loopback)

| Field | Value |
|---|---|
| **Rule** | The default gateway deployment binds HTTP to `127.0.0.1`; HTTP authentication is **deliberately absent**; loopback binding is the access control. |
| **Authority** | G4 / N7 in [`../RESUMPTION.md`](../RESUMPTION.md); [ADR-0007](../decisions/0007-paper-venue-default.md) (companion safety stance). |
| **Code enforcement** | `cmd/gateway/main.go` reads the bind address from configuration; `deploy/configs/gateway*.jsonc` defaults to loopback. |
| **Guardrail** | Smoke tests run against loopback; any deployment binding `0.0.0.0` without an upstream auth proxy is a manual operator decision. |
| **Failure mode** | Gateway exposed to non-loopback interface without a reverse proxy → unauthenticated HTTP access. |
| **Rollback** | Restore loopback binding via configuration; introduce reverse proxy with auth before re-exposing. |

---

### I5 — Boot-test regression guard for httprouter trie

| Field | Value |
|---|---|
| **Rule** | Every HTTP route registered in `internal/interfaces/http/routes/` must also appear in `cmd/gateway/boot_test.go`'s `routes` slice. The boot test exercises all routes against a fresh httprouter, catching trie conflicts at CI time. |
| **Authority** | [ADR-0010](../decisions/0010-httprouter-trie-constraints.md) |
| **Code enforcement** | `cmd/gateway/boot_test.go` — `routes` slice (60 entries at the time of writing) + `TestGatewayRouteRegistrationDoesNotPanic`. |
| **Guardrail** | `make test` (and therefore `make verify`) runs the boot test. CI fails if a route is added without the slice entry. |
| **Failure mode** | Static + wildcard path collision on the same prefix segment → httprouter panics at registration → gateway CrashLoopBackoff (real Phase 0 incident). |
| **Rollback** | Rename the static path to use hyphens (e.g., `/session/list` → `/session-list`), or restructure the conflict. ADR-0010 documents the resolution pattern. |

---

### I6 — Paper venue is the default execution mode

| Field | Value |
|---|---|
| **Rule** | The `execute` binary defaults to `PaperVenueAdapter` (synthesises fills locally, no venue contact). Live trading requires **explicit** configuration + credentials. |
| **Authority** | [ADR-0007](../decisions/0007-paper-venue-default.md) |
| **Code enforcement** | `internal/application/execution/paper_venue_adapter.go:PaperVenueAdapter`; `deploy/configs/execute.jsonc` declares `"type": "paper_simulator"`; `execute-mainnet-*.jsonc` are explicit opt-in variants requiring credentials. |
| **Guardrail** | `make smoke` runs paper mode; live modes have dedicated targets (`smoke-live-stack`, `smoke-spot-venue-live`) gated behind explicit operator intent. |
| **Failure mode** | A misconfigured deployment somehow defaults to live → real-money orders submitted unintentionally. |
| **Rollback** | Restore `execute.jsonc` to `paper_simulator`; rotate any compromised credentials; review the configuration loader for the path that allowed a non-paper default. |

---

### I7 — NATS subject taxonomy is verb-last

| Field | Value |
|---|---|
| **Rule** | Subjects follow `{domain}.{plane}.{type}.{verb}[.{key}]` with verb at the tail. Planes are: `events`, `event`, `control`, `command`, `reply`, `query`, `projection`, `fill`, `rejection`, `session`, `activation`. |
| **Authority** | [ADR-0009](../decisions/0009-subject-taxonomy.md) |
| **Code enforcement** | All `internal/adapters/nats/nats*/registry.go` files declare subjects following the pattern; `tools/raccoon-cli/src/analyzers/contracts/events.rs` validates subject conventions. |
| **Guardrail** | `make contract-audit` (part of `make verify` via `quality-gate`). |
| **Failure mode** | A new domain publishes `signal.generated.rsi.events.*` (verb in the middle) → consumer subscriptions need bespoke shape per domain → subject filter expressivity breaks. |
| **Rollback** | Rename the subjects in the publisher (and consumers) to verb-last form before merging. ADR-0009 has the canonical examples. |

---

### I8 — ControlGate fails open on read failure (operational path)

| Field | Value |
|---|---|
| **Rule** | `ControlKVStore.IsHalted` returns `false` (continue trading) on all five KV read failure modes: `nil_bucket`, `key_not_found`, `ctx_timeout`, `kv_error`, `unmarshal_error`. Each failure increments a labeled counter so the silent failure mode is monitorable. Query/admin paths surface the error to the caller. |
| **Authority** | [ADR-0012](../decisions/0012-control-gate-fail-open-posture.md) |
| **Code enforcement** | `internal/domain/execution/control.go:ControlGate`, `…:DefaultControlGate`; `internal/adapters/nats/natsexecution/control_kv_store.go:IsHalted` (inlines the JetStream read so each failure mode can be counted). |
| **Guardrail** | `internal/adapters/nats/natsexecution/control_kv_store_unit_test.go:TestIsHalted_NilReceiver_FailsOpenAndCountsNilBucket`; `…:TestIsHalted_UnstartedStore_FailsOpenAndCountsNilBucket`; counter `marketfoundry_execution_gate_read_failures_total{reason}` (`internal/shared/metrics/metrics.go`). |
| **Failure mode** | Trade submitted during a transient KV outage **while** an operator's halt intent was simultaneously in flight (compound failure surface; mitigated by eight-layer defense-in-depth, see ADR-0012). |
| **Rollback** | Posture is intentional; would only be re-evaluated if sustained non-zero rates appear on `kv_error` or `ctx_timeout` (see ADR-0012 "When to revisit"). Hybrid strategies M16/M17/M18 are deferred pending counter data. |

---

### I9 — RESUMPTION reflects current reality (no aspirational claims)

| Field | Value |
|---|---|
| **Rule** | No document in the foundry declares a capability that the code has not yet shipped. `RESUMPTION.md` is the sentinel; if it claims `Implemented`, code + tests must back the claim. `Partially Implemented`, `Planned`, and `Deferred` are honest alternatives — use them. |
| **Authority** | P7 of the Fase Harvest (see [`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest"); reinforced by [TRUTH-MAP](../TRUTH-MAP.md). |
| **Code enforcement** | None at the AST level (this is documental discipline). The drift hook (`scripts/check-resumption-drift.sh`) catches *one* drift class (new `M<N>` design-meta references without RESUMPTION updates). |
| **Guardrail** | `lefthook` post-commit `resumption-drift` hook (warn-only, exit 0); reviewer discipline at PR review time; TRUTH-MAP cross-reference. |
| **Failure mode** | RESUMPTION says "Implemented" but code has the gap → future agent assumes the feature exists, builds on top of it, discovers the lie at runtime. |
| **Rollback** | Correct the RESUMPTION entry to the truthful status (`Partially Implemented` + gap reference, or `Planned`); add the gap to a G-section if not already there. |

---

### I10 — `make verify` GREEN is the merge gate

| Field | Value |
|---|---|
| **Rule** | A PR cannot merge while `make verify` is RED. The gate composes: Go test suites (17 modules), `repo-consistency-check` (raccoon-cli drift detection), `quality-gate` (6 analyzers, 84 checks), `lint-go` (golangci-lint across 17 modules). Currently the canonical end-to-end PASS. |
| **Authority** | [ADR-0004](../decisions/0004-raccoon-cli-static-enforcement.md) (the analyzer that runs the gate); P7 of the Fase Harvest (RESUMPTION updated in the same commit). |
| **Code enforcement** | `Makefile:verify` target = `test repo-consistency-check quality-gate lint-go`. |
| **Guardrail** | Branch protection on `main` requires the CI workflow's required checks (Unit Tests, Repository Consistency & Quality Gate, Go Lint) to be GREEN before merge. |
| **Failure mode** | A red `make verify` ignored → architectural drift, broken anchors in TRUTH-MAP, lint regression — each contributes invisible debt. Per CLAUDE.md Core protocol #3 ("Honesty over convenience"), a convenient categorisation of red CI is exactly when verification is most likely to lapse. |
| **Rollback** | Investigate the failure (do not bypass `--no-verify` without explicit maintainer authorisation). Fix forward; restore GREEN before continuing the wave. If the failure is genuinely pre-existing (e.g., flake), open a follow-up ticket rather than bundle the fix. |

---

## Cross-reference: where each invariant runs in `make verify`

`make verify = test + repo-consistency-check + quality-gate + lint-go`.
Mapping each invariant to the layer that catches its violation:

| Invariant | Caught by |
|---|---|
| I1 (single-writer) | `quality-gate` (topology-doctor analyzer) |
| I2 (layer sovereignty) | `quality-gate` (arch-guard analyzer) — also `make arch-guard` standalone |
| I3 (forward-only migrations) | Operational (`make migrate-up`, `smoke-analytical`) — not enforced by `make verify` directly |
| I4 (loopback default) | Operational (deployment review) — not enforced by `make verify` |
| I5 (boot-test guard) | `test` (Go test suite via `cmd/gateway/boot_test.go`) |
| I6 (paper default) | Operational — `smoke` uses paper; live modes require explicit opt-in |
| I7 (subject taxonomy) | `quality-gate` (contract-audit analyzer) |
| I8 (ControlGate fail-open) | `test` (Go test suite via `control_kv_store_unit_test.go`) + Prometheus counter at runtime |
| I9 (RESUMPTION reflects reality) | Reviewer discipline + `lefthook` post-commit drift hook (warn-only) |
| I10 (`make verify` GREEN) | The composite gate itself — branch protection in CI; `make verify` locally |

---

## Adding a new invariant

When a new architectural rule is being enforced:

1. **Write or extend an ADR** justifying the rule (T1; see
   [`../decisions/`](../decisions/README.md)).
2. **Add an analyzer or test** that catches a violation (P5 of the
   Fase Harvest: an invariant without enforcement is intent, not
   architecture).
3. **Add a row in this catalogue** with all five fields (Rule,
   Authority, Code enforcement, Guardrail, Failure mode, Rollback).
4. **Cross-link in [`../TRUTH-MAP.md`](../TRUTH-MAP.md)** so the
   anchor is discoverable from the capability side.
5. **Update `make verify`** if the analyzer adds a new check;
   ensure the gate runs the analyzer in the `quality-gate` profile.

---

## Changelog

- **2026-05-24** — Initial version, shipped as H-1 deliverable.
  Top-10 invariants documented with code anchors verified against
  the codebase. I1–I8 are statically enforced; I9–I10 are
  discipline-and-gate enforced. Cross-reference table maps each
  invariant to the `make verify` layer that catches its violation.
