# TRUTH-MAP — Capability × Evidence cross-reference

**Status:** Active
**Date:** 2026-05-26
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
| Configctl lifecycle (Draft→Validated→Compiled→Active→Deactivated→Archived) | [ADR-0006](decisions/0006-configctl-lifecycle-authority.md) | `internal/domain/configctl/lifecycle.go:VersionLifecycle`; `internal/domain/configctl/config_set.go:ConfigSet` | `internal/domain/configctl/config_set_test.go:TestConfigSetLifecycleTransitions`; `…:TestConfigSetRejectsInvalidLifecycleTransitions` | Implemented | All seven states declared; transitions enforced. |
| Observation domain (Trade) | [ADR-0009](decisions/0009-subject-taxonomy.md) (subject), [ADR-0008](decisions/0008-single-writer-invariant.md) (writer), [ADR-0021](decisions/0021-canonical-instrument-and-venue-model.md) (identity, partial) | `internal/domain/observation/trade.go:ObservationTrade` (now carries `Instrument CanonicalInstrument` + transitory `VenueSymbol()` method); `internal/adapters/nats/natsobservation/registry.go:DefaultRegistry` | `internal/domain/observation/trade_test.go:TestObservationTrade_VenueSymbol`, `…:TestObservationTrade_Validate` | Implemented | Single writer = `ingest`. H-6.a migrated the `Symbol string` field to `Instrument CanonicalInstrument`; `VenueSymbol()` is transitory (sunset H-6.f). |
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
| Subject taxonomy (verb-last, canonical symbol token) | [ADR-0009](decisions/0009-subject-taxonomy.md) (+ erratum 2026-06-10) | `internal/adapters/nats/nats*/registry.go` (subject declarations); `internal/domain/instrument/subject_token.go:SubjectToken` (única derivação sancionada do `{symbol}`, H-6.e; 4º componente `[_expiry]` ATIVO desde H-7.c — erratum 2026-06-12) + `…:FromSubjectToken` (parser canonical→canonical, H-6.f.1; aceita 3 ou 4 componentes desde H-7.c — premissa sem-underscore estendida, expiry digits-only); `tools/raccoon-cli/src/analyzers/check_subjects.rs` + `policies/subjects.toml` (seções `[subjects]` H-6.e, `[keys]` H-6.e.2, `[dedup]` H-6.f.1) | `internal/domain/instrument/subject_token_test.go` (lock-ins + roundtrips 3- e 4-componentes + rejeições + premissa); `internal/adapters/nats/natssignal/subject_cutover_canary_test.go` (integration, canonical+mixed-state); raccoon-cli `check subjects` (gate step 10; 18 unit tests — 8 H-6.e + 4 keys H-6.e.2 + 6 dedup H-6.f.1) | Implemented | Pattern: `{domain}.{plane}.{type}.{verb}[.{key}]`; `{symbol}` = `base_quote_contract[_expiry]` lowercase (corpo desde H-6.e; expiry desde H-7.c, byte-idêntico sem expiry) (legacy venue-native convive até TTL 72h; KV keys canônicas desde H-6.e.2 — `{source}.{subject_token}.{timeframe}`; dedup keys JetStream canônicas desde H-6.f.1 (9 sites; janela de 2min quebrada uma vez no deploy — risco aceito); contrato HTTP = trio `base/quote/contract`; ClickHouse WHERE legacy com valor derivado via `LegacyFilterValue()` até o flip em H-6.f.2). |
| Forward-only ClickHouse migrations | [ADR-0003](decisions/0003-clickhouse-analytical.md) | `cmd/migrate/engine/runner.go:Runner`; `deploy/migrations/*.sql`; `_migrations` metadata table | (operational) | Implemented | Rollback is forward fix; never revert. |
| Gateway httprouter trie regression guard | [ADR-0010](decisions/0010-httprouter-trie-constraints.md) | `cmd/gateway/boot_test.go` (`routes` slice) | `cmd/gateway/boot_test.go:TestGatewayRouteRegistrationDoesNotPanic` | Implemented | 61 routes enumerated (60→61 em H-7.a: `/venues/capabilities`); CI fails if route added without slice entry. |
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
| Claude Code hooks enforcement de P2/P9 | [ADR-0026](decisions/0026-claude-code-hooks-enforcement.md) | `../.claude/hooks/raccoon-readonly-guard.sh`; `../.claude/hooks/p9-branch-guard.sh`; `../.claude/settings.json` (permissions + wiring) | `../.claude/hooks/test-guards.sh` (13-scenario deny/allow/ask matrix) | Implemented | Primeiro ADR de processo com code+test anchors reais. Posture: deny (P2, push-main, --no-verify) / ask (gh pr merge), owner decision 2026-06-09. |

### Foundation ADRs — delivery state (mixed)

Foundation ADRs delivered in Onda H-2 (`0017–0023`). Initially all
seven landed as `Proposed` (T3 per [`AUTHORITY.md`](AUTHORITY.md))
with placeholder code/test anchors; each is promoted to `Accepted`
(T1) by the onda that ships the supporting code, in the same
commit that flips the `Status` field.

Current state (post-Onda H-6.d.2, 2026-05-28):
- **Accepted** (T1, `Implemented`): ADR-0017, ADR-0018 (promoted
  by Onda H-3.b); ADR-0019, ADR-0020 (promoted by Onda H-4 — dual
  promotion closing Fase Wire); ADR-0024, ADR-0025 (promoted by
  Onda H-5 — dual promotion in PROGRAM-0003 Observability).
- **Proposed** (T3, `Partially Implemented`): ADR-0021 — domain
  root + 2 adapters + analyzer landed in H-6.a; 7 derivative
  analytics domain types migrated in H-6.b; 3 execution-chain
  domain types migrated in H-6.b'; 2 pairing-chain domain types
  migrated + 1 query-filter type declared `string_filter` in
  H-6.b'' (closes the domain-layer migration surface within
  PROGRAM-0004 H-6 scope); application-layer pass-through
  migration for derive scope landed in H-6.c.1 + for execute
  scope landed in H-6.c.2 — `instrumentFromBinding` helper
  eliminated from 5 packages (signal/decision/strategy/risk/
  execution), 1 remaining (executionclient → H-6.f).
  `BindingTarget.Instrument()` boundary helper (error-returning,
  no silent zero) wired through derive actor cascade (H-6.c.1)
  and execute testnet adapters (H-6.c.2). 5 silent error-
  discard sites in ClickHouse `composite_reader.go` converted
  to warn-and-emit-zero (uniform pattern across all 13
  `reconstructInstrumentFromLegacy` callers). `ReviewTransform`
  application-layer DTO declared as permanent `string_filter`
  (H-6.c.2 commit 3). **Criterion #4b end-to-end LANDED**
  (H-6.d.1 + H-6.d.2). Writer-side (H-6.d.1): 6 ClickHouse
  migrations add canonical `base`/`quote`/`contract` columns
  (008-013); writer populates end-to-end via 14 codegen
  artifacts + 17 INSERT statements + 8 mappers in
  `writerpipeline/support.go`; canary integration test in
  `writerpipeline/canonical_columns_integration_test.go` (6
  tests / 1 per table). Reader-side (H-6.d.2): new helper
  `instrumentFromCanonicalColumns(base, quote, contract)` +
  exported `ErrLegacyRow` sentinel in
  `internal/adapters/clickhouse/canonical_instrument_columns.go`;
  7 reader files / 13 instrument-resolution sites / 13 SELECT
  column lists migrated to dual-path (canonical primary,
  legacy fallback, warn-and-emit-zero); reader canary
  `canonical_columns_reader_integration_test.go` (6 tests /
  18 subtests covering canonical_path / fallback_path /
  mixed_state per table). `reconstructInstrumentFromLegacy`
  RETAINED per Resolution 1 (correctness-driven through 90-day
  MergeTree TTL window; deletion + operational verification
  consolidated in H-6.f). Promotion to `Accepted` remains
  gated on criterion #2 (zero source-string-based
  reconstruction in production code) — 1 executionclient
  helper + 13 ClickHouse reconstruction callers (retained
  through TTL window) remain. Promotion is an atomic event
  in H-6.f when all criteria are literally true.
- **Accepted em H-7** (2026-06-12): ADR-0022 (multi-venue
  normalization policy — framework R1–R4 em H-7.a, primeiro venue
  não-Binance + promoção em H-7.b; 6 critérios verificados um a um
  na seção Status do ADR).
- **Proposed** (T3, `Planned`): ADR-0023 (H-9
  partial / H-10 full, may remain `Proposed` indefinitely if
  empirical triggers T1/T2/T3 never fire).

| Capability | ADR / PRD | Code anchor | Test anchor | Status | Notes |
|---|---|---|---|---|---|
| Canonical event envelope (9 fields incl. seq, ts_*, idempotency_key) | [ADR-0017](decisions/0017-event-envelope-and-versioning.md) | Proto schema: `proto/envelope/v1/envelope.proto:Envelope` (H-3.a). Generated Go: `internal/shared/contracts/envelope/v1/envelope.pb.go:Envelope` (H-3.b). Converter + domain projection: `internal/shared/contracts/envelope/v1/converter.go:CanonicalEvent`, `…:ToProto`, `…:FromProto` (H-3.b). | `make proto-lint` (H-3.a). `internal/shared/contracts/envelope/v1/envelope_test.go:TestEnvelopeRoundTrip`, `…:TestEnvelopeRoundTrip_TsExchangeAbsent`, `…:TestEnvelopeByteStability` (H-3.b). `internal/shared/contracts/envelope/v1/converter_test.go:TestRoundTrip_AllFieldsPresent`, `…:TestRoundTrip_TsExchangeAbsent`, `…:TestToProto_RequiredFieldValidation`, `…:TestFromProto_RequiredFieldValidation` (H-3.b). | Implemented | ADR promoted to `Accepted` in Onda H-3.b. Coexists with legacy transport envelope (`internal/shared/envelope/`); stream migration is execution-of-decision (future phase) per the 2026-05-25 erratum. |
| Protobuf contract layer (proto wire + buf tooling + raccoon-cli `check proto`) | [ADR-0018](decisions/0018-protobuf-contract-layer.md) | Schemas + tooling: `proto/buf.yaml`, `proto/buf.gen.yaml`, `proto/registry.json` (H-3.a). Generated Go boundary: `internal/shared/contracts/` (H-3.b — `envelope/v1/envelope.pb.go` + `marketdata/v1/trade.pb.go` tracked, gitignored entry G removed). Analyzer: `tools/raccoon-cli/src/analyzers/check_proto.rs:analyze` (H-3.b). | `make proto-lint`, `make proto-gen`, `make proto-breaking` (H-3.a). `make proto-check` + `raccoon-cli check proto` analyzer with 9 unit tests (H-3.b). `make verify` invokes both `proto-lint` and `check proto` (via `quality-gate`). | Implemented | ADR promoted to `Accepted` in Onda H-3.b. Proto primary for mesh; JSON fallback during migration; HTTP-API stays JSON. `protoc-gen-go` pinned at v1.36.8 in `scripts/bootstrap-check.sh` matching the runtime in `internal/shared/go.mod`. |
| Deterministic replay invariants (INV-D1..D4) | [ADR-0019](decisions/0019-deterministic-replay-time-invariants.md) | Ports: `internal/shared/clock/clock.go:Clock`, `internal/shared/random/random.go:Source`. Replay: `internal/shared/replay/recorder.go:Recorder`, `…:Player`. Analyzer: `tools/raccoon-cli/src/analyzers/check_determinism.rs:analyze`. Domain migration: `internal/domain/execution/{control,session,activation,verification}.go` (5 production call sites, all migrated to `clock.Clock`). | `internal/shared/clock/clock_test.go`, `internal/shared/random/random_test.go`, `internal/shared/replay/replay_test.go`, `internal/shared/replay/golden_test.go:TestGolden_Synthetic100_ByteIdentical`, `…:TestGolden_ByteStability_N50`. `make verify` runs `check determinism` as Step 7 of the gate. | Implemented | ADR promoted to `Accepted` in Onda H-4 (dual promotion with ADR-0020). `internal/domain/` production code mechanically free of `time.Now`. Test files exempted from analyzer per documented rationale in ADR References. |
| Sequencer producing monotonic seq per stream key | [ADR-0020](decisions/0020-sequencing-and-time-normalization.md) | Package: `internal/shared/sequencer/sequencer.go:Sequencer`, `…:StreamKey`. KV adapter: `internal/adapters/nats/natssequencer/store.go:Store`, `…:SequencerStateBucket`. Counter: `internal/shared/metrics/sequencer_metrics.go:IncSeqGap`. | `internal/shared/sequencer/sequencer_test.go:TestSequencer_MonotonicWithinKey`, `…:TestSequencer_ConcurrentSafe`, `…:TestSequencer_RestoreResumesFromSnapshot`. Integration: `internal/adapters/nats/natssequencer/store_roundtrip_test.go` (`//go:build integration`). | Implemented | ADR promoted to `Accepted` in Onda H-4. Per-writer Sequencer integration in the running stack (ADR-0020 critério 5) explicitly deferred to a successor fase as execution-of-decision; the architectural decision and shipping primitives are Accepted. |
| Canonical instrument & venue model | [ADR-0021](decisions/0021-canonical-instrument-and-venue-model.md) | Domain package: `internal/domain/instrument/asset.go:BaseAsset`, `…:QuoteAsset`; `internal/domain/instrument/venue.go:Venue`; `internal/domain/instrument/contract_type.go:ContractType`; `internal/domain/instrument/canonical.go:CanonicalInstrument` (campo opcional `Expiry` YYMMDD desde H-7.c — ADR-0021 erratum 2026-06-12, fecha G10), `…:New`, `…:NewDelivery` (H-7.c), `…:Symbol` (sufixo `@expiry` quando não-vazio), `…:FromSymbol`; `internal/domain/instrument/subject_token.go:SubjectToken` (4º componente `[_expiry]` ativo desde H-7.c), `…:FromSubjectToken` (H-6.f.1; 4-parts H-7.c). Migrated domain types (H-6.a + H-6.b + H-6.b' + H-6.b''): `internal/domain/observation/trade.go:ObservationTrade`, `internal/domain/evidence/{candle,trade_burst,volume}.go`, `internal/domain/signal/signal.go:Signal`, `internal/domain/decision/decision.go:Decision`, `internal/domain/strategy/strategy.go:Strategy`, `internal/domain/risk/risk.go:RiskAssessment`, `internal/domain/execution/execution.go:ExecutionIntent`, `internal/domain/effectiveness/effectiveness.go:Attribution`, `internal/domain/execution/audit_bundle.go:AuditLifecycleEntry`, `internal/domain/pairing/pairing.go:Leg`, `internal/domain/pairing/pairing.go:RoundTrip` — all carry `Instrument CanonicalInstrument` + `VenueSymbol()` transitory accessor. Query-filter type with permanent string field (Decisão #2 of H-6.b''): `internal/domain/pairing/continuity.go:CrossSessionWindow` (field renamed `Symbol string` → `VenueSymbol string` with inline godoc rationale; `string_filter` policy state). Triage projection at the application boundary: `internal/application/triageclient/get_roundtrip_triage.go:74` adopts `review.VenueSymbol()` (Decisão #4 of H-6.b''). Adapter integration: `internal/adapters/exchanges/binances/aggtrade.go:parseSpotSymbol`; `internal/adapters/exchanges/binancef/aggtrade.go:parseFuturesSymbol` (with `deliverySuffix` regex). Per-package `instrumentFromBinding` transitory helpers — **ALL 6 ELIMINATED** (signal/decision/strategy/risk in H-6.c.1 commits 7a-7d + execution in H-6.c.2 commit 5 + executionclient in H-6.f.1 commit 2: `audit_session.go` migrated to `instrument.FromSubjectToken` over the canonical partition-key token, fixing the silent-zero audit regression introduced by the H-6.e.2 key cutover; `anti_patterns.toml` entry now severity=error with empty exception list). H-6.c.2 testnet adapters (`binance_spot_testnet_adapter.go:391`, `binance_futures_testnet_adapter.go:395`) now use `BindingTarget.Instrument()` boundary helper with warn-and-emit-zero fallback (commit 4); the residual architectural-debt option (a) port-signature refactor is recorded in PROGRAM-0004 H-6.f scope notes. **New canonical boundary helper** (H-6.c.1 commit 6): `internal/application/ingest/binding.go:BindingTarget.Instrument()` with error-returning signature (no silent zero) — registered venues are `binances`→Spot and `binancef`→Perpetual; synthetic sources (`"binance"`, `"binance_spot"`, `"derive"`, `"clickhouse"`, `"unknown_exchange"`, `"execute.venue-adapter"`) intentionally absent, surfacing the H-6.b' 37f8ddd regression-shape rather than hiding it. Derive actors compute Instrument once at `internal/actors/scopes/derive/source_scope_actor.go:onActivateSampler` and pass through the signal/decision/strategy/risk/execution cascade. ClickHouse-side `reconstructInstrumentFromLegacy` at `internal/adapters/clickhouse/candle_reader.go:150` has 13 callers, **all now operating as the fallback branch of the H-6.d.2 dual-path** (canonical first → legacy fallback → warn-and-emit-zero). **Schema migration LANDED in H-6.d.1**: 6 migrations (`deploy/migrations/008_add_canonical_columns_evidence_candles.sql` → `013_add_canonical_columns_executions.sql`) add `base`/`quote`/`contract LowCardinality(String) DEFAULT '' AFTER symbol/base/quote` to all 6 Instrument-bearing tables. Writer populates canonical columns end-to-end via 14 YAML family specs + 14 golden snapshots + 17 INSERT SQL strings in `cmd/writer/pipeline.go` + 8 mappers in `internal/adapters/clickhouse/writerpipeline/support.go` (each mapper appends `string(x.Instrument.Base), string(x.Instrument.Quote), string(x.Instrument.Contract)` after `VenueSymbol()`). `internal/adapters/clickhouse/client.go:Client.Exec` (added H-6.d.1 commit 3b) supports DDL via native protocol for integration test reset paths. **Reader cutover LANDED in H-6.d.2**: new helper `internal/adapters/clickhouse/canonical_instrument_columns.go:instrumentFromCanonicalColumns` + exported `ErrLegacyRow` sentinel for idiomatic `errors.Is` discrimination between expected-legacy-row case and hard validation regressions. 7 reader files / 13 sites scan `&base, &quote, &contract` and dispatch through the dual-path (`instrumentFromCanonicalColumns` first; on `ErrLegacyRow` fall back to `reconstructInstrumentFromLegacy(src, sym)`; warn-and-emit-zero on second failure). 13 SELECT column lists across 8 query builders + 5 composite inline SELECTs insert `base, quote, contract` after `symbol`, matching writer column ordering. `reconstructInstrumentFromLegacy` RETAINED per Resolution 1 (helper deletion deferred to H-6.f post-90-day-TTL operational verification). Analyzer: `tools/raccoon-cli/src/analyzers/check_instruments.rs:analyze`; policies: `tools/raccoon-cli/policies/adapters.toml` (adapter allowlist) + `tools/raccoon-cli/policies/domain_types.toml` (per-type migration state — 12 migrated, 1 string_filter, 0 pending) + `tools/raccoon-cli/policies/anti_patterns.toml` (H-6.c.1 commit 1; declarative anti-pattern function names with warning severity during migration window). | `internal/domain/instrument/instrument_test.go` (21 tests); `internal/domain/observation/trade_test.go:TestObservationTrade_VenueSymbol`; `internal/adapters/exchanges/binancef/aggtrade_test.go:TestNormalize_DeliveryFuturesPattern`, `…:TestNormalize_PerpetualClassification`, `…:TestNormalize_RejectsNonUSDTQuote`; `internal/adapters/exchanges/binances/aggtrade_test.go:TestNormalize_RejectsNonUSDTQuote`; per-type `TestSignal_VenueSymbol`, `TestDecision_VenueSymbol`, `TestStrategy_VenueSymbol`, `TestRisk_VenueSymbol` (+ multi-symbol partition-key isolation tests); `internal/domain/effectiveness/effectiveness_test.go` (Attribution migrated via `btcUSDTPerp(t)` helper); `internal/domain/pairing/pairing_test.go:TestMatchFIFO_PerfectPair` (asserts `rts[0].Instrument == btcUSDTSpot` AND `rts[0].VenueSymbol() == "btcusdt"`), `…:TestMatchFIFO_DifferentInstrumentsDoNotPair` (M1 invariant via native struct equality); `internal/domain/pairing/s494_continuity_test.go:TestCrossSessionWindow_Valid` (with renamed VenueSymbol field); `internal/application/triageclient/get_roundtrip_triage_test.go:TestGetRoundTripTriage_ProjectsVenueSymbolFromInstrument`, `…:TestGetRoundTripTriage_ZeroInstrumentProducesEmptyString` (Decisão #5β projection + regression-canary); smoke `/analytical/composite/pairing/review` instrument.base canary in `scripts/smoke-analytical-e2e.sh` Phase 5 (Decisão #5γ; tri-state PASS/WARN/FAIL); H-6.c.1 pass-through canary: `internal/application/ingest/binding_test.go:TestBindingTarget_Instrument_RejectsUnknownSource` (6 synthetic sources) + `…:TestBindingTarget_Instrument_RejectsInvalidShape` (4 invalid shapes); H-6.c.1 derive-scope canary: `internal/actors/scopes/derive/synthetic_source_canary_integration_test.go` (3 tests / 15 subtests — rejection at boundary, full activation flow with structured log assertion, legitimate-activation-proceeds inverse canary); per-package `instrument_passthrough_test.go` in `internal/application/{signal,decision,strategy,risk}/` (6+3+3+2 = 14 NewXxxForInstrument constructors exercised end-to-end); H-6.c.2 execute-scope canary: `internal/actors/scopes/execute/execute_venue_adapter_canary_test.go` (2 tests / 2 passes — `TestPaperOrderEvaluator_PreservesInstrument_WithSyntheticSource` unit shape + `TestStrategyConsumerActor_PreservesInstrument_WithSyntheticSource` actor shape, locking the 37f8ddd regression contract for the "execute.venue-adapter" synthetic source); H-6.d.1 writer-population canary: `internal/adapters/clickhouse/writerpipeline/canonical_columns_integration_test.go` (~527 LoC, `//go:build requireclickhouse`, 6 tests / 1 per table — `TestWriter_PopulatesCanonicalColumns_EvidenceCandles`/`Signals`/`Decisions`/`Strategies`/`RiskAssessments`/`Executions` each insert via writer mapper then SELECT base/quote/contract from the per-table fixture to assert non-empty population); H-6.d.2 reader-resolution canary: `internal/adapters/clickhouse/canonical_columns_reader_integration_test.go` (~714 LoC, `//go:build requireclickhouse`, 6 tests / 18 subtests — `TestReader_CanonicalColumns_<Table>` per table with `canonical_path` / `fallback_path` / `mixed_state` subtests; `mixed_state` is the literal proof of Resolution 1 — coexisting canonical and legacy rows both resolve to equivalent CanonicalInstrument values via different branches; ETH/USDT/spot fixture disambiguates the canonical path from the binances→BTC/USDT/spot reconstruction default); H-6.d.2 helper unit tests: `internal/adapters/clickhouse/canonical_instrument_columns_test.go` (4 tests / 9 sub-cases — all-empty → ErrLegacyRow, single-empty → ErrLegacyRow, valid triples → CanonicalInstrument, invalid contract → non-ErrLegacyRow regression guard); `cargo test analyzers::check_instruments` (20 tests — +5 for anti_patterns scan from H-6.c.1 commit 1); H-6.f.1: `internal/domain/instrument/subject_token_test.go:TestFromSubjectToken_RoundtripPerContractType` (+ `_Rejections` 10 cases + `_NoUnderscoreInComponents` premise lock-in); audit-regression canaries `internal/application/executionclient/s462_audit_session_test.go:TestAuditSession_LifecycleInstrumentCanary` (+ `_LegacyOrphanIsZero` + non-zero assert no FullBundle); `cmd/migrate/engine/splitter_test.go` (14 real shapes pinned single-statement + synthetic multi-statement); `cargo test analyzers::check_subjects` (18 tests — +6 `[dedup]`). | Partially Implemented | ADR-0021 stays `Proposed` through PROGRAM-0004 H-6.a–H-6.f.1 (criterion #2 literally closed in H-6.e.2 per erratum 2026-06-10); flips to `Accepted` only in **H-6.f.2** (gate temporal pós-TTL ~2026-08-26 + operational verification — split f.1/f.2 de 2026-06-11, ver PRD "Erratum de sequenciamento"). H-6.a erratum split criterion #4 into #4a (writer-side adapt, this onda) and #4b (ClickHouse migration, H-6.d). H-6.b migrated 7 types (Evidence × 3 + Signal/Decision/Strategy/Risk); H-6.b' migrated 3 types (ExecutionIntent + Attribution + AuditLifecycleEntry); H-6.b'' migrated 2 pairing-chain types (Leg + RoundTrip) and declared CrossSessionWindow as `string_filter` per Decisão #2 (rename only, no Instrument upgrade — the field is query metadata, never read by matching algorithm; promoting would force regression-prone source-string reconstruction per the 37f8ddd precedent). **H-6.d.1** landed criterion #4b writer-side: 6 migrations + writer canonical column population end-to-end + integration canary. **H-6.d.2** landed criterion #4b reader-side: new helper `instrumentFromCanonicalColumns` + `ErrLegacyRow` sentinel; 7 reader files / 13 sites migrated to dual-path; reader canary 6/18 subtests covering canonical_path / fallback_path / mixed_state per table. `reconstructInstrumentFromLegacy` retained through 90-day TTL window per Resolution 1; deletion + operational verification consolidated in H-6.f. **H-6.f scope revision** (post-pre-flight 6 of H-6.b'' + post-H-6.d.1 expansion): cleanup pass now explicitly includes (1) audit and removal of all 6 `instrumentFromBinding` helpers in `application/{signal,decision,strategy,risk,execution,executionclient}/` — **COMPLETE in H-6.f.1** (all 6 eliminated; anti_patterns entry severity=error); (2) audit `reconstructInstrumentFromLegacy` in `adapters/clickhouse/candle_reader.go:150` (13 callers uniformly warn-and-emit-zero post-H-6.c.2; helper retention through TTL window per Resolution 1 — H-6.f.2); (3) migration runner multi-statement support — **DELIVERED in H-6.f.1 commit 4** (`cmd/migrate/engine/splitter.go:SplitStatements`); (4) exception list shrinking — 7 ClickHouse entries in `anti_patterns.toml` (currently tagged "H-6.d helper removal") removed post-cutover + TTL window; (5) operational verification post-TTL — confirm legacy-only rows expired, canonical-only reads PASS sem fallback; (6) promote ADR-0021 to `Accepted` when literal criterion #2 + #4b satisfaction confirmed (zero source-string-based instrument reconstruction in production code). |
| Multi-venue normalization policy (Capabilities + `check venue-parity`) | [ADR-0022](decisions/0022-multi-venue-normalization-policy.md) | **Framework LANDED in H-7.a** (sem venue novo): contrato `internal/application/ports/capabilities.go:Capabilities` (+`EventTypeSupport`, `Allows`, `Validate` — em ports per arch-guard: interfaces/actors consomem sem importar adapters); retrofit `binances.Capabilities()` + `binancef.Capabilities()` (declarações estáticas/literais); guard R3 em `internal/actors/scopes/ingest/websocket_actor.go:declared` + counter `marketfoundry_adapter_undeclared_event_total{venue,event_type,contract}` (`shared/metrics`); rota `GET /venues/capabilities` (`routes/venues.go` + `handlers/venues.go`, wired em `cmd/gateway/compose.go`); analyzer `tools/raccoon-cli/src/analyzers/check_venue_parity.rs` + `policies/venue_parity.toml` (gate step 11). **Bybit LANDED in H-7.b** (primeiro venue não-Binance): `internal/adapters/exchanges/bybits/` (spot) + `…/bybitf/` (linear perpetual) — `ParsePublicTrade` tri-state (frames de controle v5 skipados), `Normalize` em batch (data[] com N trades/frame; BuyerMaker = taker `S`=="Sell"), `parseBybit*Symbol` USDT-only (bybitf rejeita delivery — gate G10), WSClient subscribe-frame + ping app-level; `instrument.VenueBybit`/`VenueBybitFutures`; `venueSourceContract` +bybits/+bybitf; `adapters.toml` allowlist (4); união do gateway (4 venues); RUNTIME.md → "Venue ingest sources". | `internal/application/ports/capabilities_test.go` (7 tests / 10 subcases); lock-ins `binances/binancef/bybits/bybitf capabilities_test.go`; guard canaries `websocket_actor_guard_test.go` (rejeita E conta); `routes/venues_test.go`; `cmd/gateway/boot_test.go` entry; `cargo test analyzers::check_venue_parity` (8 tests); bybits 9 tests + bybitf 6 tests unit; canário integration `natsobservation/bybit_ingest_canary_test.go` (batch não colapsa no dedup window + ambos os sources roteiam com instrument canônico no payload) PASS vs NATS vivo ×2 runs. | Implemented | **ADR-0022 `Accepted` (2026-06-12, H-7.b)** — 6 critérios literais verificados na seção Status do ADR (incl. a divergência de layout bybits/bybitf vs o path único "bybit/" esboçado, registrada lá). Observação multi-venue; execução segue Binance-only (CLAUDE.md N4 re-escopado). Decisões da abertura H-7 no PROGRAM-0004 → "Sub-ondas H-7". |
| Storage tier (staged: ClickHouse cold + NATS KV hot; TimescaleDB trigger-gated) | [ADR-0023](decisions/0023-storage-tier-roadmap.md), [PROGRAM-0007](programs/PROGRAM-0007-storage-tier.md) | **Stage 1 (atual)**: cold = ClickHouse (ADR-0003, `internal/adapters/clickhouse/`, writer binary); hot = NATS KV (ADR-0008, `store` projections). Insights/cross-venue persistem em ambos (H-8). **Gatilhos do Stage 2 instrumentados (H-9)**: recording rules `storage:gateway_op_query_p99_5m` (T1) + `storage:store_rss_bytes` (T2) + alertas `StorageTriggerT1GatewayLatency`/`StorageTriggerT2StoreMemory` em `deploy/observability/prometheus/{recording,alerts}.rules.yml`; SLIs em `docs/operations/slo.md` + `docs/operations/runbooks/storage-triggers.md`. **Stage 2 (TimescaleDB, H-10) NÃO construído** — trigger-gated. | `promtool check rules` (46 recording + 15 alert); T1 via `marketfoundry_http_request_duration_seconds` (`InstrumentHTTPHandler`, `webserver/server.go`); T2 via `process_resident_memory_bytes` (store `/metrics`). | Stage 1 Implemented (H-9); Stage 2 Deferred | **ADR-0023 Stage 1 `Accepted` (2026-06-14, H-9); Stage 2 `Proposed` pending triggers T1/T2/T3.** Gatilho primeiro, onda depois — H-10 abre só quando um gatilho disparar (pode nunca abrir; steady state legítimo). PROGRAM-0007 `Deferred`. |
| Insights — volume profile (VPVR), decision-support read-only | [ADR-0027](decisions/0027-insights-decision-support.md), [PROGRAM-0005](programs/PROGRAM-0005-insights.md) | Domínio `internal/domain/insights/` (`VolumeProfile` price-bucketed, `BucketLevel` binning big.Rat determinístico, `OverloadLevel` L0–L3); sampler `internal/application/derive/volume_profile_sampler.go` no derive scope; família NATS `internal/adapters/nats/natsinsights/` (`INSIGHTS_EVENTS` single-writer, publisher/consumer, `INSIGHTS_VOLUME_PROFILE_LATEST` KV, KV-direct gateway); store projection `internal/actors/scopes/store/volume_profile_projection_actor.go`; read `GET /insights/volume-profile/latest` (`routes/insights.go` + `handlers/insights.go` + `insightsclient`); analyzer `tools/raccoon-cli/src/analyzers/check_insights.rs` + `policies/insights.toml` (gate step 12). **Trades-only** (sem depth). **Persistência ClickHouse write-path LANDED na H-8.a.1 (G12 resolvido):** migration `deploy/migrations/014_create_insights_volume_profile.sql` (3 colunas `Array(String)` paralelas + canônicas base/quote/contract), family codegen `codegen/families/volume_profile.yaml` (layer `insights` evidence-style — `codegen/spec.go:usesFamilySpecificNaming`), consumer `natsinsights.WriterVolumeProfileConsumer` (durable `writer-volume-profile`), `writerpipeline.NewVolumeProfileStarter` + `mapVolumeProfileRow`, `settings.PipelineConfig.IsInsightsFamilyEnabled`, `insights.OverloadLevel.Label()`; pipeline entry em `cmd/writer/pipeline.go`. Single-writer: writer dono da tabela, store dono do KV (ADR-0008). Read de history CH deferido (KV-latest atende o read). | `internal/domain/insights/insights_test.go`; `internal/application/derive/volume_profile_sampler_test.go` (binning/finalize/overload); `internal/interfaces/http/routes/insights_test.go`; `cmd/gateway/boot_test.go` entry; `cargo test analyzers::check_insights` (6 tests); canário integration `natsinsights/volume_profile_canary_test.go` (publish→consume→KV→read vs NATS vivo). **H-8.a.1:** `writerpipeline/volume_profile_mapper_test.go` (3 arrays paralelos index-aligned, 1-row); canário `writerpipeline/volume_profile_clickhouse_test.go` (`//go:build requireclickhouse` — Array round-trip vs CH vivo); `codegen` `TestDerivedFields_Insights`; drift-detect `insights-contracts-drift` (durables + tabela). | Implemented (H-8.a + H-8.a.1) | **ADR-0027 `Accepted` (2026-06-13, H-8.a)** — insights nunca emitem directives (I1), read-only sobre o pipeline (I2, enforçado por `check insights`), trades-only (I4). TPO (H-8.b) e cross-venue (H-8.c) seguem; liquidity heatmap fora até ingestão de depth. |
| Insights — TPO profile (Time-Price Opportunity), decision-support read-only | [ADR-0027](decisions/0027-insights-decision-support.md), [PROGRAM-0005](programs/PROGRAM-0005-insights.md) | Domínio `internal/domain/insights/tpo.go` (`TPOProfile{Periods[],Levels[]}`, `TPOPeriod{Letter,High,Low}`, `TPOLevel{PriceLevel,Letters,Count}`; `PeriodLetter`, `PointOfControl`, `ValueArea` (~70% greedy), `InitialBalance`, `PriceRange` — puros, big.Rat; `EventTPOProfileSampled`); sampler `internal/application/derive/tpo_sampler.go` (timeframe-anchored T1, trades-only T2, períodos A–X cap 24 T3, POC/VA/IB/range T4); actor `internal/actors/scopes/derive/tpo_sampler_actor.go` + FamilyProcessor "tpo"; publisher `natsinsights.Publisher.PublishTPOProfile` (subject `insights.events.tpo.sampled`); store consumer `natsinsights.StoreTPOConsumer` (durable `store-tpo`) + `tpo_kv_store` (`INSIGHTS_TPO_LATEST`) + `TPOProjectionActor`; read `GET /insights/tpo/latest` (gateway KV-direct). **Trades-only**. **Persistência ClickHouse write-path LANDED na H-8.b.1:** migration `deploy/migrations/015_create_insights_tpo.sql` (períodos + níveis em Array-columns paralelas, `level_count Array(Int32)`, + métricas escalares + canônicas), family codegen `codegen/families/tpo.yaml` (layer `insights`; `tpo→TPO` em `spec.go`), consumer `natsinsights.WriterTPOConsumer` (durable `writer-tpo`), `writerpipeline.NewTPOStarter` + `mapTPOProfileRow`, pipeline entry em `cmd/writer/pipeline.go`. | `internal/domain/insights/tpo_test.go` (letras/POC/value-area/IB/range/Validate); `internal/application/derive/tpo_sampler_test.go` (períodos/níveis/finalize); `internal/interfaces/http/routes/insights_test.go` (serve-TPO + HasAny); `cmd/gateway/boot_test.go` entry; canário integration `natsinsights/tpo_canary_test.go` (publish→consume→KV→read vs NATS vivo); drift-detect `insights-contracts-drift` (`store-tpo` + `writer-tpo` + `insights_tpo`). **H-8.b.1:** `writerpipeline/tpo_mapper_test.go` (6 arrays paralelos, Int32 counts); canário `writerpipeline/tpo_clickhouse_test.go` (`//go:build requireclickhouse` — Array round-trip vs CH vivo); `codegen` `TestDerivedFields_TPO`. | Implemented (H-8.b + H-8.b.1) | **ADR-0027 `Accepted`** — TPO é decision-support (I1/I2/I4): time-at-price, nunca directive. Timeframe-anchored (T1; session-anchored é Fase futura). Cross-venue em H-8.c. |
| Delivery WebSocket — push read-only de insights events | [ADR-0028](decisions/0028-delivery-websocket-protocol.md), [PROGRAM-0006](programs/PROGRAM-0006-delivery.md) | Domínio `internal/domain/delivery/` (`Session`, `Subscription`, `SubjectMatches` — matcher de subject NATS puro: `*`=1 token, `>`=tail terminal); port `internal/application/ports/delivery.go` (`DeliveryConn`/`DeliverySession`/`DeliveryHub` — mantém `interfaces/` sem importar `actors/`, ADR-0005); consumer durável `deliver-insights` `internal/adapters/nats/natsdelivery/consumer.go` (desde H-11.b lê `insights.events.>` — TODAS as famílias; decode dispatched por subject CBOR→JSON; frame de fio `{subject,event}` p/ o cliente demuxar; reusa `natsinsights.DefaultRegistry`); fan-out `internal/actors/scopes/delivery/` (`RouterActor` broadcast + `SessionActor` dono da `Session` + buffer outbound bounded com **política configurável** I4 — `domain.BackpressurePolicy` DropNewest default + DropOldest, H-11.c; PriorityDrop deferido; `delivery.Config`/`ConfigFromEnv`; write goroutine; `Hub`/`SessionHandle`; `Runtime.Start`); métricas Prometheus `marketfoundry_delivery_frames_total{outcome}` + `…_sessions` (gauge) em `internal/shared/metrics`; endpoint `GET /ws` `internal/interfaces/http/handlers/delivery.go` (gorilla upgrade + control frames JSON `{action,subject}` subscribe/unsubscribe — único inbound, I1; loopback I2) + `routes/delivery.go` (`DeliveryFamilyDeps`); wiring `cmd/gateway/run.go` (`delivery.Start` no engine; degrada se NATS off). Analyzer `tools/raccoon-cli/src/analyzers/check_delivery.rs` + `policies/delivery.toml` (P5, H-11.d — gate Step 12b: reader-only + stream-bound). Single-writer (ADR-0008): delivery é **leitor**, derive segue único publisher. | `internal/domain/delivery/delivery_test.go` (matcher literal/`*`/`>`/combos + validação de Subscription + ciclo de vida de Session + cópia defensiva); `internal/actors/scopes/delivery/delivery_actors_test.go` (DropNewest determinístico + fan-out com filtro de subscrição, engine real + WSConn fake); `cmd/gateway/boot_test.go` entry `/ws`; `internal/adapters/nats/natsdelivery/consumer_test.go` (decode CBOR→JSON por família VP/TPO/cross-venue); canários integration `internal/actors/scopes/delivery/delivery_ws_canary_test.go` + `…/delivery_ws_multi_canary_test.go` (`//go:build integration` — VP, TPO, cross-venue, e multi-família/1-sessão vs NATS vivo; subject único por run evita flood do histórico); `internal/domain/delivery/backpressure_test.go` + `…/backpressure_actor_test.go` (Parse/round-trip + DropOldest evicta oldest determinístico); `cargo test analyzers::check_delivery` (6 tests) + gate Step 12b `check delivery` (4 checks) em `make verify`; drift-detect `insights-contracts-drift` (durable `deliver-insights`). | Implemented (H-11.a + H-11.b + H-11.c + H-11.d) | **ADR-0028 `Accepted` (2026-06-13, H-11.a)** — read-only transport (I1), loopback (I2), escopo decision-support/insights (I3 — todas as famílias desde H-11.b), backpressure bounded **configurável** DropNewest/DropOldest (I4, H-11.c; PriorityDrop deferido — insights equi-advisory, ADR-0027), single-writer respeitado (I5). Frame de fio `{subject,event}` (H-11.b). Métricas Prometheus de delivery (H-11.c). Enforcement estático `check delivery` (P5, H-11.d). **PROGRAM-0006 Closed (Fase Delivery completa, H-11.a–d).** |
| Insights — cross-venue trade fusion, decision-support read-only | [ADR-0027](decisions/0027-insights-decision-support.md), [PROGRAM-0005](programs/PROGRAM-0005-insights.md) | Domínio `internal/domain/insights/cross_venue.go` (`CrossVenueSnapshot{Venues[],spread/mid/dominant}`, `VenueRow{Venue,TradeCount,Notional,Last/High/Low}`; `ConsolidatedSpread`/`DominantVenue` puros, big.Rat; `EventCrossVenueSampled`). **Topologia nova (C1)**: fusion no nível do supervisor — `internal/application/derive/cross_venue_fusion.go` (`CrossVenueFusion` windowed, keyed por canonical instrument) + `internal/actors/scopes/derive/cross_venue_fusion_actor.go` (actor ÚNICO; `DeriveSupervisor.routeTrade` faneia todo trade; publisher próprio). Publisher `natsinsights.Publisher.PublishCrossVenue`; store consumer `natsinsights.StoreCrossVenueConsumer` (durable `store-cross-venue`) + `cross_venue_kv_store` (`INSIGHTS_CROSS_VENUE_LATEST`, key sem source) + `CrossVenueProjectionActor`; read `GET /insights/cross-venue/latest` (gateway KV-direct, sem source). **Trades-only, windowed (C2)**. **Persistência ClickHouse write-path LANDED na H-8.c.1:** migration `deploy/migrations/016_create_insights_cross_venue.sql` (venue rows em Array-columns paralelas, `venue_trade_count Array(Int64)`, sem source), family codegen `codegen/families/cross_venue.yaml`, consumer `natsinsights.WriterCrossVenueConsumer` (durable `writer-cross-venue`), `writerpipeline.NewCrossVenueStarter` + `mapCrossVenueRow`, pipeline entry. | `internal/domain/insights/cross_venue_test.go` (spread 2-venue/1-venue, dominant, rejeições); `internal/application/derive/cross_venue_fusion_test.go` (fusão 2-venue + finalize, single-venue); `internal/interfaces/http/routes/insights_test.go` (serve-cross-venue + HasAny); `cmd/gateway/boot_test.go` entry; canário integration `natsinsights/cross_venue_canary_test.go` (publish→consume→KV→read vs NATS vivo); drift-detect `insights-contracts-drift` (`store-cross-venue` + `writer-cross-venue` + `insights_cross_venue`). **H-8.c.1:** `writerpipeline/cross_venue_mapper_test.go` (6 arrays paralelos, Int64 counts); canário `writerpipeline/cross_venue_clickhouse_test.go` (`//go:build requireclickhouse`); `codegen` `TestDerivedFields_CrossVenue`. | Implemented (H-8.c + H-8.c.1) | **ADR-0027 `Accepted`** — cross-venue é decision-support (I1/I2/I4): fusão descritiva, nunca directive. C1: fusion actor no supervisor (não per-source). Persistência CH em H-8.c.1 (fecha a Fase). |
| Storage tier roadmap (Stage 1 → Stage 2 with empirical triggers) | [ADR-0023](decisions/0023-storage-tier-roadmap.md) | Stage 1: existing ClickHouse + KV (no new code); Stage 2 TODO (Onda H-10 — `internal/adapters/storage/timescale/`) | Stage 1: existing analytical + projection tests; Stage 2 TODO (Onda H-10) | Planned (partial) | Stage 1 active today on existing ClickHouse + KV. Stage 2 (TimescaleDB) opens only when triggers T1/T2/T3 fire; may remain `Planned` indefinitely. |
| Metrics policy (naming + label budget + cardinality + log compensation pattern) | [ADR-0024](decisions/0024-metrics-policy.md) | Policy ratifies existing pattern in `internal/shared/metrics/{metrics,sequencer_metrics}.go`. Refactor (drop `instrument` from `consumer_seq_gap_total`) shipped in `internal/shared/metrics/sequencer_metrics.go:IncSeqGap` (now `(venue, eventType)`). Analyzer: `tools/raccoon-cli/src/analyzers/check_metrics.rs:analyze`. Policy file: `tools/raccoon-cli/policies/binaries.toml`. | `internal/shared/metrics/sequencer_metrics_test.go:TestIncSeqGap_*` (3 tests covering new label shape). `make verify` invokes `check metrics` via gate Step 8 (3 checks). `cargo test analyzers::check_metrics` (10 tests). | Implemented | ADR promoted to `Accepted` in PROGRAM-0003 H-5. Naming convention grandfathered for `marketfoundry_http_*`; new metrics conform to MP-1. Label validation against MP-2 is documented as future-onda analyzer extension. |
| Alerting strategy (SLO status taxonomy + burn-rate windows + severity tiers) | [ADR-0025](decisions/0025-alerting-strategy.md) | Recording rules: `deploy/observability/prometheus/recording.rules.yml` (44 rules). Alert rules: `deploy/observability/prometheus/alerts.rules.yml` (13 rules). SLO doc: `docs/operations/slo.md` (status taxonomy section, F1-F4 all Observing). | `promtool check rules` validates both YAML files (44+13 = 57 rules SUCCESS). `make verify` GREEN with new files committed. | Implemented | ADR promoted to `Accepted` in PROGRAM-0003 H-5. SLO status taxonomy (Proposed/Observing/Committed) formally documented; F1-F4 currently Observing with alerts at `ticket` severity per AS-3. |
| Observability stack (Prometheus + Grafana, compose profile) | [PROGRAM-0003](programs/PROGRAM-0003-observability.md), [ADR-0024](decisions/0024-metrics-policy.md), [ADR-0025](decisions/0025-alerting-strategy.md) | `deploy/observability/prometheus/{prometheus,recording.rules,alerts.rules}.yml`; `deploy/observability/grafana/{provisioning,dashboards}/`. 5 dashboards (ingest/derive/store/gateway/determinism-health). Compose: `deploy/compose/docker-compose.yaml` profile `observability`. Makefile: `obs-up`/`obs-down`/`obs-reload`/`metrics-check`. | `make verify` runs `check metrics` analyzer as gate Step 8. `make obs-up` brings stack up; manual validation via Prometheus :9090 + Grafana :3000. | Implemented | Opt-in profile (does not come up under `make up`). Single phase (H-5) of PROGRAM-0003. Operator guide: [`operations/observability.md`](operations/observability.md). |
| `marketfoundry_consumer_seq_gap_total` label refactor (drop instrument; log compensation) | [ADR-0024](decisions/0024-metrics-policy.md) MP-2 + MP-5 | `internal/shared/metrics/sequencer_metrics.go:consumerSeqGapTotal` (label set now `{venue, event_type}`); `IncSeqGap(venue, eventType string)` helper documents log compensation pattern inline. | `internal/shared/metrics/sequencer_metrics_test.go:TestIncSeqGap_IncrementsCounter`, `…:TestIncSeqGap_LabelsAreIndependent`, `…:TestSeqGapTotal_ExposedOnMetricsEndpoint` (assert new labels appear + `stream_key` absent). | Implemented | H-4 declared counter with composite `stream_key`; H-5 refactored per ADR-0024 MP-2 (instrument is high-cardinality, prohibited). Log compensation pattern (MP-5) documented inline at IncSeqGap docstring for future callers. |
| Raccoon-cli `check metrics` analyzer (every long-running `cmd/*/main.go` exposes `/metrics`) | [ADR-0024](decisions/0024-metrics-policy.md), [PROGRAM-0003](programs/PROGRAM-0003-observability.md) | `tools/raccoon-cli/src/analyzers/check_metrics.rs:analyze`; `tools/raccoon-cli/policies/binaries.toml` (declarative allowlist: `one_shot = ["migrate"]`, `transitive_registration = ["gateway"]`); CLI variant + dispatch + gate Step 8 integration. | `cargo test analyzers::check_metrics` (10 tests). `make verify` GREEN includes `check metrics` PASS. `make metrics-check` standalone target. | Implemented | Declarative allowlist over inferred patterns (per H-5 user refinement). Transitive registration list documented as known tech debt (future scan via `go list -deps`). |
| Raccoon-cli `check instruments` analyzer (adapter normalization + domain-type migration state) | [ADR-0021](decisions/0021-canonical-instrument-and-venue-model.md), [PROGRAM-0004](programs/PROGRAM-0004-multi-venue.md) | `tools/raccoon-cli/src/analyzers/check_instruments.rs:analyze`; `tools/raccoon-cli/policies/adapters.toml` (adapter allowlist: `["binances", "binancef"]`); `tools/raccoon-cli/policies/domain_types.toml` (domain-type migration state, H-6.b extension + H-6.b'' `string_filter` state); CLI variant + dispatch + gate Step 9 integration. | `cargo test analyzers::check_instruments` (15 tests covering adapter + domain-type checks + `string_filter` tolerance). `make verify` GREEN includes `check-instruments` PASS (6 checks: adapters-dir, policy-present, adapter-allowlisted, adapter-uses-canonical-constructor, domain-types-policy-present, domain-type-migration-state). | Implemented | Declarative two-policy enforcement: adapters.toml gates new adapters; domain_types.toml gates per-type migration state. The schema recognizes three states: `migrated` (analyzer requires `instrument.CanonicalInstrument` field reference + `VenueSymbol() string` method — enforced); `pending` (legacy `Symbol string` tolerated, transient until the type's own sub-onda migrates it); `string_filter` (venue-native string field by design, permanent — declared in H-6.b'' for CrossSessionWindow to record that promotion would force regression-prone source-string reconstruction). H-6.f will sunset the VenueSymbol checks when accessors are removed. |

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

## Summary counts (2026-05-28, post-H-6.d.2)

- HTTP routes registered: **60** (in `cmd/gateway/boot_test.go`).
- NATS streams declared: **11**.
- NATS adapter registry files: **8** (one per writer family).
- NATS KV buckets: **17** (16 read-model + `SEQUENCER_STATE_LATEST`
  added in Onda H-4).
- Go test files under `internal/` and `cmd/`: **~317** (recounted
  2026-06-10 via `find`; the previous ~295 was maintained by manual
  increments and had drifted).
- ClickHouse migrations: **13** (007 + 008-013 H-6.d.1 canonical column splits).
- ADRs published: **26** (0001–0020 `Accepted` + 0024–0026
  `Accepted`; 0021–0023 `Proposed`). 0017+0018 promoted by Onda
  H-3.b; 0019+0020 promoted by Onda H-4; 0024+0025 promoted by
  Onda H-5 (dual promotion in PROGRAM-0003); 0021 carries an
  erratum landed in H-6.a (criterion #4 split into #4a/#4b);
  0026 (Claude Code hooks enforcement, harness/process scope like
  0013–0015) accepted with its implementation in P5.6.
- PRDs published: **4** (PROGRAM-0001 `Active`; PROGRAM-0002
  `Closed` by Onda H-4; PROGRAM-0003 `Active` opened by Onda H-5;
  PROGRAM-0004 `Active` opened by Onda H-6.a).
- `make verify` checks executed: **102** (across 9 static analyzers
  in the gate; `check-instruments` grew from 4 → 6 checks in Onda
  H-6.b with the addition of the domain-type migration state check
  via `policies/domain_types.toml`).
- Prometheus recording rules: **44** (4 SLOs × ~10 rules each +
  runtime-aggregates group). Alert rules: **13** (8 SLO burn-rate
  + 5 runtime-safety).
- Grafana dashboards: **5** (ingest-health, derive-health,
  store-health, gateway-health, determinism-health).

---

## Changelog

- **2026-06-10/11 (H-6.e.2)** — Read-contract canonical cutover:
  HTTP query contract = trio `base/quote/contract` (legacy `symbol`
  param retired; zero external consumers); client query structs +
  reader/KV ports carry `CanonicalInstrument`; KV partition keys
  compose via `SubjectToken()` (write+read); ClickHouse builders
  byte-identical with `LegacyFilterValue()`-derived args (WHERE flip
  in H-6.f post-TTL). Critério #2 do ADR-0021 literalmente
  satisfeito (erratum chain e → e.2 → f).

- **2026-06-10 (H-6.e)** — Subject-taxonomy row updated for the
  canonical symbol-token cutover: anchors gain
  `instrument.SubjectToken()` (única derivação sancionada),
  `check_subjects.rs` + `policies/subjects.toml` (gate step 10),
  lock-in tests + integration canary (canonical + mixed-state).
  KV partition keys mantêm o layout VenueSymbol-derived até
  H-6.e.2 (erratum ao critério #2 do ADR-0021, mesma data).
- **2026-06-10** — FASE 3.2 (harness/docs evolution): ADR-0026 row
  added to "Process / protocol" with real code+test anchors
  (`.claude/hooks/*` + `test-guards.sh`) — the first process ADR
  with a test anchor; Status `Implemented` rather than
  `Documentation` because the enforcement is mechanical
  (PreToolUse hooks + permissions), not human discipline.
- **2026-05-28** — Onda H-6.d.2 closure: **ClickHouse reader-side
  cutover para canonical columns com legacy fallback**. 4 commits.
  **Critério #4b reader-side LANDED** — completa ADR-0021 erratum
  #4b end-to-end (writer-side H-6.d.1 + reader-side H-6.d.2).
  Novo helper
  `internal/adapters/clickhouse/canonical_instrument_columns.go`
  com `ErrLegacyRow` sentinel exportada +
  `instrumentFromCanonicalColumns(base, quote, contract)`.
  Sentinel pattern (`errors.Is`) per Decisão #3 — idiomatic Go
  discrimination entre expected-legacy-row case e validation
  regressions em rows com canonical populados mas inválidos.
  4 unit tests / 9 sub-cases. Reader dual-path migration: 7
  reader files / 13 instrument-resolution sites / 13 SELECT
  column lists atualizados uniformemente. Pattern uniforme
  através dos 13 sites (validated em pré-flight 3). Per-table
  query builders (8): `BuildCandleQuery` / `BuildSignalQuery` /
  `BuildDecisionQuery` / `BuildStrategyQuery` / `BuildRiskQuery`
  + 3 em `execution_reader.go` (`BuildExecutionQuery` /
  `BuildLifecycleHistoryQuery` / `BuildExecutionListQuery`).
  Composite reader inline SELECTs (5):
  `querySignalByCorrelation` / `queryDecisionByCorrelation` /
  `queryStrategyByCorrelation` / `queryRiskByCorrelation` /
  `queryExecutionByCorrelation`. 8 test files atualizados:
  expectedCols slices estendidas, column counts bumped (candle
  12→15, signal 8→11, decision 12→15, strategy 11→14, risk
  13→16, execution 16→19). Reader canary integration test
  `canonical_columns_reader_integration_test.go` (~714 LoC,
  `//go:build requireclickhouse`) com 6 tests / 18 subtests
  (canonical_path / fallback_path / mixed_state per table) —
  mixed_state subtest é a prova literal da Resolução 1
  (ambas shapes coexistem durante 90-day TTL window).
  `reconstructInstrumentFromLegacy` **RETAINED** per Resolução
  1; deletion deferida para H-6.f post-TTL operational
  verification. ADR-0021 row code + test anchors atualizados.
  ADR-0021 permanece `Proposed`; promoção atómica em H-6.f
  post-TTL + helper deletion + executionclient migration +
  literal critério #2 satisfaction.

- **2026-05-27** — Onda H-6.d.1 closure: **ClickHouse schema
  migration + writer canonical column population end-to-end**.
  4 commits. **Critério #4b writer-side** do ADR-0021 erratum
  LANDED. 6 migrations adicionadas
  (`deploy/migrations/008_add_canonical_columns_evidence_candles.sql`
  → `013_add_canonical_columns_executions.sql`), uma por
  Instrument-bearing table — split per-table após Decisão #1 (A)
  (ClickHouse Go driver multi-statement constraint surfaced
  contra production CH instance; runner enhancement deferred
  para H-6.f scope expansion). Cada migration adiciona
  `base`/`quote`/`contract LowCardinality(String) DEFAULT ''
  AFTER symbol/base/quote` idempotently (`ADD COLUMN IF NOT
  EXISTS`). Codegen self-consistency atomic bundle (commit 2):
  14 YAML family specs + 14 golden snapshots regenerated + 17
  INSERT SQL strings em `cmd/writer/pipeline.go` + 8 mappers em
  `writerpipeline/support.go` (each appends 3 canonical values
  após `VenueSymbol()`) + ~120 test row position shifts em
  `support_test.go` + `behavioral_roundtrip_test.go`.
  Integration fixture migration (commit 3a): 34 positional
  INSERTs em `composite_reader_integration_test.go` para
  explicit column lists + 20 pre-H-6.b drift fixes (`.Symbol`
  → `.VenueSymbol()` em fixtures + analytical test files —
  tagged-build invisibility lesson, 3 months undetected).
  Writer canary (commit 3b): `Client.Exec(ctx, query, args)`
  adicionado em `internal/adapters/clickhouse/client.go` para
  DDL via native protocol (clickhouse-go/v2 Query returns EOF
  on DDL); novo
  `internal/adapters/clickhouse/writerpipeline/canonical_columns_integration_test.go`
  (~527 LoC, `//go:build requireclickhouse`, 6 tests / 1 per
  table). **Resolution 1 — Helper retention through 90-day
  TTL**: composite_reader 5 callers + 8 sister-site readers de
  `reconstructInstrumentFromLegacy` MANTÊM warn-and-emit-zero
  fallback até H-6.f post-TTL operational verification.
  **Lessons registered**: (1) positional INSERT pre-flight
  discipline for schema changes future-onward; (2) tagged-build
  drift detection — files com `//go:build requireclickhouse`
  invisíveis ao default `make verify`; (3) codegen
  self-consistency atomic bundle invariant reaffirmed.
  **H-6.f scope expansion** (registered): helper deletion +
  migration runner multi-statement support + exception list
  shrinking (7 ClickHouse entries) + operational verification
  post-TTL + ADR-0021 promotion. ADR-0021 row code + test
  anchors updated. ADR-0021 remains `Proposed`; promotion
  atomic in H-6.f post-reader-cutover + helper deletion +
  TTL verification.

- **2026-05-27** — Onda H-6.c.2 closure: **application-layer
  pass-through migration for execute scope + ClickHouse
  composite_reader uniformization**. 8 commits.
  `instrumentFromBinding` helper **completely eliminated** from
  the 5th application package (execution — commit 5). 1 package
  remaining (executionclient → H-6.f, blocked by LifecycleEntry
  contract migration). Testnet adapters
  (`binance_spot_testnet_adapter.go:391`,
  `binance_futures_testnet_adapter.go:395`) migrated to use
  `BindingTarget.Instrument()` boundary helper with
  warn-and-emit-zero fallback (commit 4 — per Decisão #2 after
  pre-flight verification showed the option (a) port-signature
  refactor cascade exceeds the sub-onda threshold at 12 files).
  ClickHouse-side `reconstructInstrumentFromLegacy` callers
  uniformized: the 5 silent-discard sites in
  `composite_reader.go` (lines 188/243/302/360/423) converted
  to warn-and-emit-zero, matching the 8 existing sister sites
  (commit 2). All 13 callers now uniform pending H-6.d schema
  migration. `ReviewTransform` application-layer DTO declared
  as permanent `string_filter` in `policies/domain_types.toml`
  + inline godoc on ReviewTransform.Symbol + DecisionTriageItem
  .Symbol fields (commit 3). H-6.c.2 commit 6 adds the explicit
  37f8ddd canary in execute scope
  (`execute_venue_adapter_canary_test.go` — 2 tests / 2 passes).
  anti_patterns.toml exception list shrunk 11 → 8 entries
  (commit 7 — removed the 3 execution package entries; kept 1
  executionclient + 7 ClickHouse readers). **8 cross-scope
  test stragglers migrated** in commit 5 (1 derive
  s470_lineage_causality + 2 risk_scaling + 6 integration-
  tagged: 1 writerpipeline + 4 natsexecution + 1
  live_consumer_flow) — surfaced by the explicit
  `go test -tags=integration -run DOES_NOT_EXIST` check
  (lesson 13 of H-6.c.1 reinforced: make verify masks
  integration-tagged build failures). ADR-0021 row code +
  test anchors updated for the 5/6 helper-elimination state +
  new boundary-helper wiring in execute scope + new canary
  test anchor. ADR-0021 remains `Proposed`; promotion gated on
  literal critério #2 satisfaction (zero source-string-based
  reconstruction in production), atomic in H-6.f. **H-6.f
  architectural-debt note recorded**: option (a) port-signature
  refactor for QueryOrder eliminates the residual reconstruction
  in adapter layer (architecturally cleaner than the current (b)
  BindingTarget.Instrument() approach); cascade analysis = 12
  files; tractable as dedicated H-6.f sub-task alongside
  executionclient migration.

- **2026-05-27** — Onda H-6.c.1 closure: **application-layer
  pass-through migration for derive scope**. 10 commits.
  `instrumentFromBinding` helper **completely eliminated** from
  4 application packages (signal/decision/strategy/risk —
  commits 7a-7d); 2 packages remaining (execution → H-6.c.2;
  executionclient → H-6.f). New canonical boundary helper
  `internal/application/ingest/binding.go:BindingTarget.Instrument()`
  (commit 6) returns explicit error for unknown sources —
  synthetic sources (`"binance"`, `"binance_spot"`, `"derive"`,
  `"clickhouse"`, `"unknown_exchange"`,
  `"execute.venue-adapter"`) intentionally absent from the
  registry, surfacing the H-6.b' 37f8ddd silent-zero
  regression-shape at its source rather than hiding it. Derive
  actors compute Instrument once at
  `source_scope_actor.onActivateSampler` and pass through the
  signal/decision/strategy/risk/execution cascade. 14
  `NewXxxForInstrument` constructors added across the 4
  packages (commits 2-5); 5 derive Config structs gain
  canonical Instrument field; 10 derive actor files migrate;
  derive_supervisor cascades inst parameter through 12 factory
  NewActor callbacks. ~250 application test sites migrated
  via sed/Python-driven uniform pattern. Derive-scope canary
  integration tests added (commit 8 — 3 tests / 15 subtests
  asserting rejection at boundary, full activation flow with
  structured log assertion, legitimate-activation-proceeds
  inverse canary). New raccoon-cli policy file
  `tools/raccoon-cli/policies/anti_patterns.toml` (commit 1)
  declares forbidden source-string reconstruction function
  names; `check_instruments` analyzer extended with
  `scan_anti_pattern` + 5 unit tests; severity is `warning`
  during migration window, flips to `error` in H-6.f.
  ADR-0021 row updated to reflect 4 helpers eliminated + new
  boundary helper + derive actor cascade + canary tests.
  **Pattern observed**: 5 instances of gofmt drift discovered
  during H-6.c.1 (commits 4, 6, 7a, 7c, 7d) — pre-existing
  drift surfaced opportunistically by pre-commit hook on
  touched files. Mitigations registered in RESUMPTION.md
  retrospective section (decision deferred — candidate
  options: full-repo gofmt audit + cleanup, pre-commit hook
  on whole repo, or CI step validating zero drift).
  **Schema decision recorded**: `anti_patterns.toml` schema
  remains function-based (not per-package with status field)
  per the architectural decision that filesystem is source
  of truth for migration status; future per-package refactor
  is only justified if drift appears between policy
  declaration and filesystem reality.

- **2026-05-26** — Onda H-6.b'' closure: **pairing chain domain
  migration + CrossSessionWindow `string_filter` declaration**.
  ADR-0021 row code anchor extended to include the 2 H-6.b''
  migrated types (pairing.Leg, pairing.RoundTrip) and the renamed
  CrossSessionWindow field (`Symbol string` → `VenueSymbol
  string`), plus the triage-projection pull-forward at
  `triageclient/get_roundtrip_triage.go:74`. Notes updated to
  reflect 12 of 15 initial domain types migrated + 1 declared
  `string_filter` (CrossSessionWindow). check-instruments
  analyzer grows to 15 unit tests (`+1` for `string_filter`
  state tolerance) — gate check count unchanged (still 6 checks
  PASS). **H-6.f scope revision** captured in ADR-0021 Notes:
  cleanup pass now explicitly includes audit + removal of the
  6 `instrumentFromBinding` reconstructors in `application/*`
  and the 11 discarded errors from `reconstructInstrumentFromLegacy`
  in `adapters/clickhouse`; promotion to `Accepted` requires
  literal zero source-string-based instrument reconstruction in
  production. **8 commits delivered** (plan declared 9 —
  consolidation via compile pressure documented in commits 3
  and 8). **P4/P9 deviation observed**: H-6.b'' work started on
  branch `feat/h-6-b1-execution-chain` before H-6.b' merged in
  `main` (PR #28); branch was rebased on `origin/main` (commit
  `6b62d89`) post-H-6.b'-merge to reconcile history and produce
  a clean PR containing only the 9 H-6.b'' commits. Lesson
  registered in PR description and in `CONTRIBUTING.md`
  pre-push validation discipline.

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
- **2026-05-25** — Onda H-3.a closure: rows for ADR-0017 and
  ADR-0018 partially populated. ADR-0017 anchor now points to
  `proto/envelope/v1/envelope.proto` (schema delivered in H-3.a);
  generated Go and converter pending H-3.b. ADR-0018 anchor now
  points to `proto/buf.yaml`, `proto/buf.gen.yaml`,
  `proto/registry.json`, and the three `make proto-*` targets
  (skeleton + tooling delivered in H-3.a); raccoon-cli `check
  proto` analyzer pending H-3.b. Implementing onda labels for
  both ADRs split H-3 → H-3.a / H-3.b per the 2026-05-25 erratum
  to their "Promoção para Accepted" sections. ADR count
  unchanged (23).
- **2026-05-25** — Onda H-3.b closure: **first ADR promotions of
  the Fase Harvest**. ADR-0017 and ADR-0018 flipped
  `Proposed` → `Accepted`; rows updated with real code/test
  anchors (no TODOs); status moved from `Planned` to
  `Implemented`. Section "Planned capabilities — Foundation ADRs
  (Proposed)" renamed to "Foundation ADRs — delivery state
  (mixed)" to reflect that the section now holds entries in two
  states (Accepted/Implemented vs Proposed/Planned). Summary count
  updated: 0001–0018 Accepted; 0019–0023 Proposed.
- **2026-05-25** — Onda H-4 closure: **dual ADR promotion closing
  Fase Wire**. ADR-0019 and ADR-0020 flipped
  `Proposed` → `Accepted`; rows for both moved from `Planned` to
  `Implemented` with full code/test anchors covering replay
  (recorder + player + JSONL fixture format), sequencer
  (in-memory monotonic counter + KV adapter), ports (clock.Clock
  + random.Source), domain migration (5 production call sites in
  `internal/domain/execution/`), `check determinism` analyzer
  (raccoon-cli Step 7 of the gate), and golden test + N=50
  byte-stability validation. PROGRAM-0002 transitioned to
  `Closed`. Summary counts updated: 23 ADRs (0001–0020 Accepted,
  0021–0023 Proposed); 17 KV buckets (added
  `SEQUENCER_STATE_LATEST`); 93 `make verify` checks (added
  +3 from `check determinism`); 2 PRDs (PROGRAM-0001 Active,
  PROGRAM-0002 Closed).
- **2026-05-25** — Onda H-5 closure: **PROGRAM-0003 opened +
  dual ADR promotion**. ADR-0024 (metrics policy) and ADR-0025
  (alerting strategy) flipped `Proposed` → `Accepted` in the
  same onda they were introduced (different pattern from
  PROGRAM-0002 which inherited Proposed ADRs from H-2). New rows
  added to the Foundation ADRs section covering: metrics-policy
  + `consumer_seq_gap_total` refactor + alerting-strategy +
  observability stack + `check metrics` analyzer. PROGRAM-0003
  opened `Active`. Summary counts updated: 25 ADRs (added 0024 +
  0025 both Accepted); 96 `make verify` checks (+3 from
  `check metrics`); 3 PRDs (added PROGRAM-0003 Active); 44
  recording rules + 13 alert rules + 5 Grafana dashboards new
  metrics infrastructure declared.
- **2026-05-26** — Onda H-6.b' closure: **execution chain domain
  types migrated**. Three additional domain types migrated `Symbol
  string` → `Instrument CanonicalInstrument` + `VenueSymbol()`
  transitory accessor: ExecutionIntent (with PartitionKey and
  DeduplicationKey composers updated via `VenueSymbol()`),
  Attribution (derived from `intent.Instrument` in Classify /
  ClassifyPair), AuditLifecycleEntry (reconstructed at conversion
  boundary via new per-package `instrumentFromBinding` helper in
  `internal/application/executionclient/`). LifecycleEntry DTO
  remains string-based — read-path migration deferred to H-6.f
  along with VenueSymbol sunset. Total domain types now migrated:
  10 of 15 with Symbol field (3 from H-6.a/H-6.b + 7 from H-6.b +
  3 from H-6.b'). Policy file `policies/domain_types.toml` flipped
  the 3 H-6.b' entries from `pending` → `migrated`; check-instruments
  analyzer remains at 6 checks PASS. Summary counts unchanged at
  102 verify checks; ADR-0021 row stays `Partially Implemented`
  pending Pairing chain (H-6.b'') and the H-6.f atomic promotion.
  Triage drop closure note: zero population sites required migration
  in this sub-wave — DecisionTriageItem is buffered by ReviewTransform
  DTO (domain→DTO boundary migrated in H-6.b; DTO→Triage remains
  string until H-6.c migrates ReviewTransform); ExecutionTriageItem
  does not exist in codebase; RoundTripTriageItem deferred to
  H-6.b''. Sub-wave naming convention documented: prose uses
  apostrophes (H-6.b, H-6.b', H-6.b''); branch names use numeric
  suffix (feat/h-6-b1-…, feat/h-6-b2-…) for shell portability.
- **2026-05-26** — Onda H-6.b closure: **derivative analytics
  domain types migrated**. Seven domain types migrated `Symbol
  string` → `Instrument CanonicalInstrument` + `VenueSymbol()`
  transitory accessor: EvidenceCandle, EvidenceTradeBurst,
  EvidenceVolume, Signal, Decision, Strategy, RiskAssessment. The
  four PartitionKey-pattern types compose KV keys via
  `VenueSymbol()` preserving bucket layout back-compat. Application
  samplers/evaluators (6 signal samplers + 3 decision evaluators
  + 3 strategy resolvers + 2 risk evaluators) gain a per-package
  `instrumentFromBinding(source, venueNative)` transitory helper
  that drives an internal `instrument CanonicalInstrument` field —
  full sampler/evaluator API migration deferred to H-6.c. ClickHouse
  readers reuse `reconstructInstrumentFromLegacy` from H-6.a;
  writers map `.VenueSymbol()` to the legacy `symbol` column.
  Analyzer extended: `check-instruments` grew from 4 to 6 checks,
  reading new `policies/domain_types.toml` and enforcing
  Instrument-field + VenueSymbol-method invariant on every type
  marked `migrated`. Summary counts updated: 102 verify checks
  (+2 from check-instruments domain-type checks); ADR-0021 row
  stays `Partially Implemented` (more types migrated, but `Proposed`
  remains pending H-6.f promotion gate).
- **2026-05-25** — Onda H-6.a closure: **PROGRAM-0004 opened +
  partial ADR-0021 implementation**. PROGRAM-0004
  (Multi-venue) opened with 6 sub-ondas H-6.a–H-6.f + H-7 (sub-
  onda sequencing policy stricter than P4). Erratum to ADR-0021
  splitting criterion #4 into #4a (writer-side adapt, this
  onda — zero schema change) and #4b (ClickHouse migration,
  H-6.d). `internal/domain/instrument/` package shipped (Venue,
  BaseAsset, QuoteAsset, ContractType, CanonicalInstrument with
  21 tests). `ObservationTrade.Symbol string` migrated to
  `Instrument CanonicalInstrument` atomically with both Binance
  adapters (binances spot, binancef perpetual + delivery futures
  pattern detection via `_\d{6}$` regex) — option (C) transitory
  accessor `VenueSymbol()` with semantically distinct name
  documents the sunset onda (H-6.f). New `check instruments`
  analyzer added to the gate at Step 9, backed by
  `policies/adapters.toml` (allowlist `binances`/`binancef`).
  ADR-0021 remains `Proposed` — promotion is atomic in H-6.f
  after criterion #2 literally satisfied. Summary counts updated:
  100 `make verify` checks (+4 from `check-instruments`); 4 PRDs
  (added PROGRAM-0004 Active); ADR-0021 row state changed from
  `Planned` to `Partially Implemented` while staying `Proposed`.
