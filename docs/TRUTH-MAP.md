# TRUTH-MAP — Capability × Evidence cross-reference

**Status:** Active
**Date:** 2026-05-24
**Owner:** Repository maintainer
**Authority tier:** T1 — Canonical
([`AUTHORITY.md`](AUTHORITY.md))
**Relates to:**
[`decisions/`](decisions/README.md),
[`programs/PROGRAM-0001-foundation.md`](programs/PROGRAM-0001-foundation.md),
[`RESUMPTION.md`](RESUMPTION.md),
[`ARCHITECTURE.md`](ARCHITECTURE.md),
[`RUNTIME.md`](RUNTIME.md)

---

## Purpose

The **single authoritative cross-reference** between what the foundry
claims to do and the evidence backing each claim. Every capability
declared in CLAUDE.md, RESUMPTION.md, ARCHITECTURE.md, or RUNTIME.md
must appear here with at least one ADR/PRD link, one code anchor, and
one test anchor (where the capability has a test surface).

This document is the **runtime-evidence layer** for the architecture
docs. ARCHITECTURE.md says *what* the system is shaped like; TRUTH-MAP
says *where* you can verify it in the code.

---

## Invariants (rules that govern this document)

1. **Every claim has ADR/PRD + code anchor + test anchor** when the
   capability has a test surface. Documentation-only capabilities
   (e.g., process ADRs like pause-and-report) skip the test anchor
   and say so explicitly in `Notes`.
2. **No anchor is invented.** If the code or test does not exist,
   the row uses `NOT FOUND` and is listed in the
   "Capabilities sem TRUTH cobertura" section with a reason.
3. **Status taxonomy** is exactly one of:
   - `Implemented` — code + tests + ADR all present and aligned.
   - `Partially Implemented` — code present but coverage incomplete
     (gap documented in RESUMPTION or a `Notes` cell).
   - `Planned` — ADR or PRD declares it; code not yet shipped.
   - `Deferred` — explicitly deferred to a future onda with reason.
   - `Documentation` — non-code capability (process ADR, protocol).
4. **Anchor format:** `path/to/file.go:SymbolOrTestName`. Line
   numbers are avoided because they drift on every refactor.
5. **Updates are append-or-correct.** New rows are added as
   capabilities land. Existing rows are corrected immediately if
   the underlying anchor moves (P7 — sem perda de disciplina
   documental).

---

## Capability map

### Family domains (have their own NATS stream)

| Capability | ADR / PRD | Code anchor | Test anchor | Status | Notes |
|---|---|---|---|---|---|
| Configctl lifecycle (Draft→Validated→Compiled→Active→Deactivated→Archived) | [ADR-0006](decisions/0006-configctl-lifecycle-authority.md) | `internal/domain/configctl/lifecycle.go:VersionLifecycle`; `internal/domain/configctl/config_set.go:ConfigSet` | `internal/domain/configctl/document_test.go:TestConfigSetLifecycleTransitions`; `…:TestConfigSetRejectsInvalidLifecycleTransitions` | Implemented | All seven states declared; transitions enforced. |
| Observation domain (Trade) | [ADR-0009](decisions/0009-subject-taxonomy.md) (subject), [ADR-0008](decisions/0008-single-writer-invariant.md) (writer) | `internal/domain/observation/trade.go`; `internal/adapters/nats/natsobservation/registry.go:DefaultRegistry` | `internal/domain/observation/trade_test.go` | Implemented | Single writer = `ingest`. |
| Evidence domain (Candle, Volume, TradeBurst) | [ADR-0008](decisions/0008-single-writer-invariant.md), [ADR-0009](decisions/0009-subject-taxonomy.md) | `internal/domain/evidence/`; `internal/adapters/nats/natsevidence/registry.go` | (per-type evidence tests under `internal/domain/evidence/`) | Implemented | Single writer = `derive`. |
| Signal domain (RSI, EMA crossover, MACD, Bollinger, VWAP, ATR) | [ADR-0008](decisions/0008-single-writer-invariant.md), [ADR-0009](decisions/0009-subject-taxonomy.md) | `internal/domain/signal/`; `internal/adapters/nats/natssignal/registry.go` | `internal/actors/scopes/derive/signal_sampler_actor_test.go:TestRSISignalSamplerActor_WarmupPeriod_NoSignal` | Partially Implemented | Only 2 of 6 signal types have a KV bucket (G2 in RESUMPTION). |
| Decision domain (evaluators per signal) | [ADR-0008](decisions/0008-single-writer-invariant.md), [ADR-0009](decisions/0009-subject-taxonomy.md) | `internal/domain/decision/`; `internal/adapters/nats/natsdecision/registry.go` | (per-evaluator tests) | Implemented | Single writer = `derive`. |
| Strategy domain (Long/Short/Flat with confidence) | [ADR-0008](decisions/0008-single-writer-invariant.md), [ADR-0009](decisions/0009-subject-taxonomy.md) | `internal/domain/strategy/`; `internal/adapters/nats/natsstrategy/registry.go` | (strategy tests) | Partially Implemented | 2 of 3 types have KV bucket (G2). |
| Risk domain (Drawdown, exposure, scaling) | [ADR-0008](decisions/0008-single-writer-invariant.md), [ADR-0009](decisions/0009-subject-taxonomy.md) | `internal/domain/risk/`; `internal/adapters/nats/natsrisk/registry.go` | (risk tests) | Implemented | Single writer = `derive`. |
| Execution domain (ExecutionIntent, FillRecord, FeeSource) | [ADR-0008](decisions/0008-single-writer-invariant.md), [ADR-0007](decisions/0007-paper-venue-default.md), [ADR-0012](decisions/0012-control-gate-fail-open-posture.md) | `internal/domain/execution/execution.go:ExecutionIntent` | `internal/domain/execution/execution_test.go:TestExecutionIntent_Validate_Valid` | Implemented | Writers split: `derive` publishes `EXECUTION_EVENTS`; `execute` publishes `EXECUTION_FILL_EVENTS`, `EXECUTION_REJECTION_EVENTS`, `SESSION_LIFECYCLE_EVENTS`. |

### Internal-only domains (no stream)

| Capability | ADR / PRD | Code anchor | Test anchor | Status | Notes |
|---|---|---|---|---|---|
| Effectiveness classification (Win/Loss/Breakeven/Unresolved) | [ADR-0011](decisions/0011-no-oms-expansion-pairing.md) | `internal/domain/effectiveness/effectiveness.go:Attribution`, `…:Classify` | `internal/domain/effectiveness/effectiveness_test.go` | Implemented | Read-side classifier; no writes. |
| Pairing (FIFO match into round-trips) | [ADR-0011](decisions/0011-no-oms-expansion-pairing.md) | `internal/domain/pairing/pairing.go:RoundTrip`, `…:Leg` | `internal/domain/pairing/pairing_test.go`; `…/reconciliation_test.go`; `…/s494_continuity_test.go`; `…/s495_continuity_summary_test.go`; `…/s496_continuity_reconciliation_test.go`; `…/s500_lifecycle_close_test.go` | Implemented | Read-side; no OMS expansion by ADR-0011. |

### Binaries

| Capability | ADR / PRD | Code anchor | Test anchor | Status | Notes |
|---|---|---|---|---|---|
| `configctl` binary (lifecycle authority) | [ADR-0006](decisions/0006-configctl-lifecycle-authority.md) | `cmd/configctl/` | (integration via `make smoke`) | Implemented | |
| `gateway` binary (HTTP↔NATS translation, stateless) | [ADR-0010](decisions/0010-httprouter-trie-constraints.md) | `cmd/gateway/main.go`; `internal/interfaces/http/routes/core.go` | `cmd/gateway/boot_test.go:TestGatewayRouteRegistrationDoesNotPanic` | Implemented | 60 HTTP routes registered. |
| `ingest` binary (Binance WS → `OBSERVATION_EVENTS`) | [ADR-0001](decisions/0001-nats-not-kafka.md), [ADR-0008](decisions/0008-single-writer-invariant.md) | `cmd/ingest/`; `internal/actors/scopes/ingest/ingest_supervisor.go:IngestSupervisor`; `internal/adapters/exchanges/binance*` | (operational; smoke targets) | Implemented | |
| `derive` binary (observation → evidence/signal/decision/strategy/risk/execution; FamilyProcessor pattern) | [ADR-0002](decisions/0002-hollywood-actor-framework.md), [ADR-0008](decisions/0008-single-writer-invariant.md) | `cmd/derive/`; `internal/actors/scopes/derive/derive_supervisor.go:DeriveSupervisor`; `internal/actors/scopes/derive/source_scope_actor.go:SourceScopeActor` | `internal/actors/scopes/derive/producer_invariant_test.go:TestPI1_TypeAlwaysMeanReversionEntry` | Implemented | |
| `store` binary (KV projections + query serving; Pipeline pattern) | [ADR-0002](decisions/0002-hollywood-actor-framework.md), [ADR-0008](decisions/0008-single-writer-invariant.md) | `cmd/store/`; `internal/actors/scopes/store/store_supervisor.go:StoreSupervisor`, `…:Pipeline` | `internal/actors/scopes/store/fill_projection_actor_test.go:TestFillProjection_PutWritten_Materializes` | Implemented | Single KV writer per bucket. |
| `execute` binary (venue intake + fills) | [ADR-0007](decisions/0007-paper-venue-default.md), [ADR-0012](decisions/0012-control-gate-fail-open-posture.md) | `cmd/execute/`; `internal/actors/scopes/execute/execute_supervisor.go:ExecuteSupervisor` | `internal/application/execution/paper_venue_adapter_test.go`; `…/paper_order_evaluator_test.go`; `…/paper_fill_simulator_test.go` | Implemented | Paper default; testnet/mainnet opt-in. |
| `writer` binary (domain events → ClickHouse) | [ADR-0003](decisions/0003-clickhouse-analytical.md) | `cmd/writer/supervisor.go:writerSupervisor`; `internal/adapters/clickhouse/client.go:Client` | `cmd/writer/supervisor_test.go:TestPipelineLifecycleTransitions` | Implemented | |
| `migrate` binary (forward-only schema) | [ADR-0003](decisions/0003-clickhouse-analytical.md) | `cmd/migrate/engine/runner.go:Runner`; `deploy/migrations/000–007.sql` | (operational; applied in CI) | Implemented | 8 migrations; no Go unit test (operational). |

### Architectural invariants (cross-cutting)

| Capability | ADR / PRD | Code anchor | Test anchor | Status | Notes |
|---|---|---|---|---|---|
| Single-writer per stream / KV bucket | [ADR-0008](decisions/0008-single-writer-invariant.md) | `internal/adapters/nats/nats{configctl,observation,evidence,signal,decision,strategy,risk,execution}/registry.go:DefaultRegistry` (8 registries) | (enforced architecturally; no dedicated Go test) | Implemented | Each registry declares one writer per stream. |
| Layer sovereignty (`domain → application → adapters → actors → interfaces → cmd`) | [ADR-0005](decisions/0005-layer-sovereignty.md), [ADR-0004](decisions/0004-raccoon-cli-static-enforcement.md) | `tools/raccoon-cli/src/analyzers/arch_guard.rs` (LAYERS const + `is_allowed_dependency`) | `make arch-guard` (Rust analyzer; runs in `make verify`) | Implemented | Statically enforced. |
| Raccoon-cli arch-guard enforcement | [ADR-0004](decisions/0004-raccoon-cli-static-enforcement.md) | `tools/raccoon-cli/src/analyzers/arch_guard.rs` | `make verify` (quality-gate, 84 checks) | Implemented | |
| Raccoon-cli drift-detect const tables | [ADR-0004](decisions/0004-raccoon-cli-static-enforcement.md) | `tools/raccoon-cli/src/analyzers/drift_detect.rs:APP_BINARIES`; `…:CANONICAL_STREAMS` | `make drift-detect` | Implemented | 6 app binaries + 11 streams declared. |
| Subject taxonomy (verb-last) | [ADR-0009](decisions/0009-subject-taxonomy.md) | `internal/adapters/nats/nats*/registry.go` (subject declarations); `tools/raccoon-cli/src/analyzers/contracts/events.rs` | (enforced via raccoon-cli `contract-audit`) | Implemented | Pattern: `{domain}.{plane}.{type}.{verb}[.{key}]`. |
| Forward-only ClickHouse migrations | [ADR-0003](decisions/0003-clickhouse-analytical.md) | `cmd/migrate/engine/runner.go:Runner`; `deploy/migrations/*.sql`; `_migrations` metadata table | (operational) | Implemented | Rollback is forward fix; never revert. |
| Gateway httprouter trie regression guard | [ADR-0010](decisions/0010-httprouter-trie-constraints.md) | `cmd/gateway/boot_test.go` (`routes` slice) | `cmd/gateway/boot_test.go:TestGatewayRouteRegistrationDoesNotPanic` | Implemented | 60 routes enumerated; CI fails if route added without slice entry. |
| ControlGate fail-open posture | [ADR-0012](decisions/0012-control-gate-fail-open-posture.md) | `internal/domain/execution/control.go:ControlGate`, `…:DefaultControlGate`; `internal/adapters/nats/natsexecution/control_kv_store.go:IsHalted`, `…:Get`, `…:Put` | `internal/adapters/nats/natsexecution/control_kv_store_unit_test.go:TestIsHalted_NilReceiver_FailsOpenAndCountsNilBucket`; `…:TestIsHalted_UnstartedStore_FailsOpenAndCountsNilBucket` | Implemented | 5 failure modes counted; query path surfaces errors. |
| Paper venue default | [ADR-0007](decisions/0007-paper-venue-default.md) | `internal/application/execution/paper_venue_adapter.go:PaperVenueAdapter`, `…:NewPaperVenueAdapter`; `deploy/configs/execute.jsonc` (`"type": "paper_simulator"`) | `internal/application/execution/paper_venue_adapter_test.go` | Implemented | Live trading requires explicit config + credentials. |
| Effectiveness/pairing read-only (no OMS) | [ADR-0011](decisions/0011-no-oms-expansion-pairing.md) | `internal/domain/{effectiveness,pairing}/` (no publish, no new ClickHouse table) | `internal/domain/effectiveness/effectiveness_test.go`; `internal/domain/pairing/pairing_test.go` | Implemented | |

### NATS stream catalogue (11 streams)

| Stream | Writer | ADR | Code anchor | Status |
|---|---|---|---|---|
| `CONFIGCTL_EVENTS` | `configctl` | [ADR-0006](decisions/0006-configctl-lifecycle-authority.md), [ADR-0008](decisions/0008-single-writer-invariant.md) | `internal/adapters/nats/natsconfigctl/registry.go:DefaultRegistry` | Implemented |
| `OBSERVATION_EVENTS` | `ingest` | [ADR-0008](decisions/0008-single-writer-invariant.md) | `internal/adapters/nats/natsobservation/registry.go:DefaultRegistry` | Implemented |
| `EVIDENCE_EVENTS` | `derive` | [ADR-0008](decisions/0008-single-writer-invariant.md) | `internal/adapters/nats/natsevidence/registry.go:DefaultRegistry` | Implemented |
| `SIGNAL_EVENTS` | `derive` | [ADR-0008](decisions/0008-single-writer-invariant.md) | `internal/adapters/nats/natssignal/registry.go:DefaultRegistry` | Implemented |
| `DECISION_EVENTS` | `derive` | [ADR-0008](decisions/0008-single-writer-invariant.md) | `internal/adapters/nats/natsdecision/registry.go:DefaultRegistry` | Implemented |
| `STRATEGY_EVENTS` | `derive` | [ADR-0008](decisions/0008-single-writer-invariant.md) | `internal/adapters/nats/natsstrategy/registry.go:DefaultRegistry` | Implemented |
| `RISK_EVENTS` | `derive` | [ADR-0008](decisions/0008-single-writer-invariant.md) | `internal/adapters/nats/natsrisk/registry.go:DefaultRegistry` | Implemented |
| `EXECUTION_EVENTS` | `derive` | [ADR-0008](decisions/0008-single-writer-invariant.md) | `internal/adapters/nats/natsexecution/registry.go:DefaultRegistry` | Implemented |
| `EXECUTION_FILL_EVENTS` | `execute` | [ADR-0008](decisions/0008-single-writer-invariant.md) | `internal/adapters/nats/natsexecution/registry.go:DefaultRegistry` | Implemented |
| `EXECUTION_REJECTION_EVENTS` | `execute` | [ADR-0008](decisions/0008-single-writer-invariant.md) | `internal/adapters/nats/natsexecution/registry.go:DefaultRegistry` | Implemented |
| `SESSION_LIFECYCLE_EVENTS` | `execute` | [ADR-0008](decisions/0008-single-writer-invariant.md) | `internal/adapters/nats/natsexecution/registry.go:DefaultRegistry` | Implemented |

### Process / protocol (documentation-only)

These have no runtime code surface; the ADR is the artifact, and
the test is human discipline backed by `make verify` gates.

| Capability | ADR / PRD | Code anchor | Test anchor | Status | Notes |
|---|---|---|---|---|---|
| Pause-and-report protocol | [ADR-0013](decisions/0013-pause-and-report-protocol.md) | — | — | Documentation | Operational discipline; enforced by reviewer + agent self-discipline. |
| Defensive-scan discipline | [ADR-0014](decisions/0014-defensive-scan-discipline.md) | — | — | Documentation | Post-fix scan recipe; captured in `.claude/skills/fix-prompt-skill/SKILL.md`. |
| Wave-closure discipline | [ADR-0015](decisions/0015-wave-closure-discipline.md) | — | — | Documentation | Closure-signal recognition; M-list captures deferred debt. |
| Fase Harvest under P1–P9 | [ADR-0016](decisions/0016-harvest-from-market-raccoon.md), [PROGRAM-0001](programs/PROGRAM-0001-foundation.md) | [`../CLAUDE.md`](../CLAUDE.md) → "Fase Harvest" (canonical P1–P9) | — | Documentation | Wave protocol; enforced by maintainer + branch protection (P9). |

### Planned capabilities — Foundation ADRs (Proposed)

Capabilities whose **decisions** are recorded in Foundation ADRs
(0017–0023, delivered in Onda H-2) but whose **code** has not yet
shipped. Each ADR is T3 (Evolutionary) per
[`AUTHORITY.md`](AUTHORITY.md) until promoted to `Accepted` by the
onda that ships the supporting code. Code anchors and test anchors
become real (and the Status flips from `Planned` to `Implemented`)
in the same commit that promotes the ADR's `Status` field.

ADR-0023 may legitimately remain `Planned` indefinitely if its
empirical triggers (T1/T2/T3) do not fire; that is a documented
steady state, not pending work.

| Capability | ADR / PRD | Code anchor | Test anchor | Status | Notes |
|---|---|---|---|---|---|
| Canonical event envelope (9 fields incl. seq, ts_*, idempotency_key) | [ADR-0017](decisions/0017-event-envelope-and-versioning.md) | TODO (Onda H-3 — `internal/shared/contracts/envelope/`) | TODO (Onda H-3) | Planned | Coexists with legacy transport envelope (`internal/shared/envelope/`); legacy retired only after all 11 streams migrate. |
| Protobuf contract layer (proto wire + buf tooling + raccoon-cli `check proto`) | [ADR-0018](decisions/0018-protobuf-contract-layer.md) | TODO (Onda H-3 — `proto/`, `internal/shared/contracts/`) | TODO (Onda H-3 — `make proto-gate`) | Planned | Proto primary for mesh; JSON fallback during migration; HTTP-API stays JSON. |
| Deterministic replay invariants (INV-D1..D4) | [ADR-0019](decisions/0019-deterministic-replay-time-invariants.md) | TODO (Onda H-4 — `internal/shared/replay/`, ports for clock/random) | TODO (Onda H-4 — golden tests + N=50 byte-stability) | Planned | Backs the "backtest = production" thesis; enforced statically by raccoon-cli `check determinism` (per P5). |
| Sequencer producing monotonic seq per stream key | [ADR-0020](decisions/0020-sequencing-and-time-normalization.md) | TODO (Onda H-4 — `internal/shared/sequencer/`, KV bucket `SEQUENCER_STATE_LATEST`) | TODO (Onda H-4 — monotonicity unit tests + gap-detection counter) | Planned | Stream key = `(venue, instrument, event_type)`; owner per ADR-0008 single-writer. |
| Canonical instrument & venue model | [ADR-0021](decisions/0021-canonical-instrument-and-venue-model.md) | TODO (Onda H-6 — `internal/domain/instrument/`) | TODO (Onda H-6) | Planned | Requires refactor of existing `binances/` and `binancef/` adapters to `ToCanonical`/`FromCanonical`. |
| Multi-venue normalization policy (Capabilities + `check venue-parity`) | [ADR-0022](decisions/0022-multi-venue-normalization-policy.md) | TODO (Onda H-7 — adapter `Capabilities()`; `/venues/capabilities` HTTP route; raccoon-cli `check venue-parity`) | TODO (Onda H-7 — `cmd/gateway/boot_test.go` entry; analyzer tests) | Planned | First non-Binance adapter is typically Bybit; route registration updates the gateway boot test per ADR-0010. |
| Storage tier roadmap (Stage 1 → Stage 2 with empirical triggers) | [ADR-0023](decisions/0023-storage-tier-roadmap.md) | Stage 1: existing ClickHouse + KV (no new code); Stage 2 TODO (Onda H-10 — `internal/adapters/storage/timescale/`) | Stage 1: existing analytical + projection tests; Stage 2 TODO (Onda H-10) | Planned (partial) | Stage 1 active today on existing ClickHouse + KV. Stage 2 (TimescaleDB) opens only when triggers T1/T2/T3 fire; may remain `Planned` indefinitely. |

### Gate (verification surface)

| Capability | ADR / PRD | Code anchor | Test anchor | Status | Notes |
|---|---|---|---|---|---|
| `make verify` GREEN gate | [ADR-0004](decisions/0004-raccoon-cli-static-enforcement.md) (analyzer), P7 (discipline) | `Makefile:verify` target = `test repo-consistency-check quality-gate lint-go` | All of: 17 module test suites; raccoon-cli quality-gate (84 checks, 6 analyzers); golangci-lint | Implemented | 15/15 docs checks pass on H-0 closure. |

---

## Capabilities sem TRUTH cobertura

Capabilities that are real (declared in code or RESUMPTION) but
not yet anchored in this map, with explicit reason and the onda
that will address each.

| Capability | Reason no anchor here | Onda planned |
|---|---|---|
| `/execution-source-explain` HTTP endpoint registration | Endpoint exists in code (`internal/interfaces/http/routes/source_explain.go`) but `GetSourceExplanation` use case is never constructed in `cmd/gateway/`, so the route conditionally registers to nil and returns 404 in every deployment (G1 in [`RESUMPTION.md`](RESUMPTION.md)) | Future feature onda; not Foundation |
| KV bucket coverage for 4 signal types (bollinger, macd, vwap, atr) and 1 strategy type (squeeze_breakout_entry) | Streams exist; KV projection actors absent (G2 in [`RESUMPTION.md`](RESUMPTION.md)) | Future feature onda; not Foundation |
| `configctl` subject singular vs plural namespace transition | Mid-migration; both `configctl.event.config.*` and `configctl.events.config.*` are live (D3 in [`RESUMPTION.md`](RESUMPTION.md)) | Future cleanup onda; not Foundation |
| Hyphenated HTTP paths (`/session-list`, `/session-batch-audit`, `/execution-source-explain`) | Surface debt from P0.6 route trie resolution (D1 in [`RESUMPTION.md`](RESUMPTION.md)); ADR-0010 explains | Future API redesign onda; not Foundation |
| Stage-tagged Makefile smoke targets (`smoke-compose-wiring`, `smoke-failure-isolation`, …) | ~14 targets carry pre-reset stage tags (D4 in [`RESUMPTION.md`](RESUMPTION.md)); functional but cosmetic debt | Future cleanup onda; not Foundation |
| HTTP authentication | Deliberate absence (G4 / N7 in [`RESUMPTION.md`](RESUMPTION.md)); loopback binding is the control | Not before non-loopback deployment |
| Insights, replay, multi-venue, proto layer, observability (per ADR-0016) | Capabilities catalogued in raccoon; foundry-side ondas not yet opened | H-3 onwards per ADR-0016; H-1 maps the protocol, not the capabilities |

Inclusion in this table is **not** a TODO list — it is honest
disclosure that the foundry has more surface than the canonical
capability map covers today.

---

## How to update this document

When you ship code that adds, moves, or removes a capability:

1. **Add** a row in the appropriate section above with a real
   anchor (no placeholders).
2. **Move** a `Planned` row to `Implemented` once code + tests
   ship; keep the historical state in the row only if it would
   surprise a reader (rare).
3. **Remove** the row if the capability is excised entirely;
   record the excision in the Changelog so future readers can
   trace.
4. **Update anchors** the same commit that moves the underlying
   symbol. Do not let anchor drift accumulate (P7).
5. **`make verify`** must remain GREEN; this document is T1
   (canonical) — broken anchors are not acceptable end-state.

When in doubt about whether a capability deserves a row: if it
appears as a claim in `ARCHITECTURE.md`, `RUNTIME.md`,
`HTTP-API.md`, or `RESUMPTION.md`, it deserves a row here. If it
does not appear in any of those docs, it likely does not need
a TRUTH-MAP row either.

---

## Summary counts (2026-05-24)

- HTTP routes registered: **60** (in `cmd/gateway/boot_test.go`).
- NATS streams declared: **11**.
- NATS adapter registry files: **8** (one per writer family).
- Go test files under `internal/` and `cmd/`: **~289**.
- ADRs published: **23** (0001–0016 `Accepted`; 0017–0023 `Proposed`,
  delivered in Onda H-2 of the Fase Harvest).
- PRDs published: **1** (PROGRAM-0001, `Active`).
- `make verify` checks executed: **84** (across 6 active analyzers).

---

## Changelog

- **2026-05-24** — Initial version, shipped as H-1 deliverable.
  All current ADRs (0001–0016), PRDs (PROGRAM-0001), 8 binaries,
  11 NATS streams, and major architectural invariants
  cross-referenced to code and test anchors. Process ADRs
  (0013/0014/0015/0016) marked `Documentation`. Five capabilities
  with active gaps (G1, G2, D1, D3, D4 in RESUMPTION) explicitly
  listed in "Capabilities sem TRUTH cobertura" rather than
  silently omitted.
- **2026-05-24** — Onda H-2 closure: seven new ADRs (0017–0023,
  Foundation ADRs, `Proposed`) added under the new "Planned
  capabilities — Foundation ADRs (Proposed)" section. Each row
  declares the implementing onda (H-3, H-4, H-6, H-7, or H-10)
  that promotes the ADR and fills in real code/test anchors.
  ADR-0023's Stage 1 (ClickHouse + KV) is the active topology;
  Stage 2 (TimescaleDB) is conditional on empirical triggers and
  may remain `Planned` indefinitely. Summary count updated:
  16 → 23 ADRs published.
