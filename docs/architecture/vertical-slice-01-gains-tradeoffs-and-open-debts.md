# Vertical Slice 01 — Gains, Trade-offs, and Open Debts

> Consolidated assessment of what the `candle-to-paper-order` slice delivered, what it cost, and what remains outstanding.

---

## 1. Gains

### 1.1 Architectural Proof

The vertical slice is the first concrete evidence that the consolidated architecture (S96–S106) produces a working, end-to-end pipeline. Prior to the slice, the architecture was validated only in isolation (per-module tests, per-layer governance). The slice forced every layer to participate simultaneously.

**Gain: Confidence that the patterns compose correctly.**

| Layer | What Was Exercised |
|-------|--------------------|
| Domain | 8 families produced events that flow through the full pipeline |
| Application | Use cases, ports, and gateway clients wired for all domains |
| Adapters | NATS publishers, consumers, KV stores, request-reply gateways |
| Actors | Supervisors, child actors, binding watchers, projection actors |
| Interfaces | 25+ HTTP endpoints routing through gateway to NATS |
| Infrastructure | Docker Compose, NATS JetStream, KV buckets, healthchecks |
| Governance | raccoon-cli validated structural invariants end-to-end |

### 1.2 Bug Discovery

The slice surfaced **7 infrastructure/wiring bugs** that would have been invisible without integration pressure:

1. Healthcheck port mismatches (4 services)
2. Gateway test stub incompleteness
3. Implicit configctl HTTP config
4. Missing local.env for ClickHouse
5. Raccoon-CLI stale test functions (14 dead references)
6. Docker Compose env interpolation failure
7. Drift detector missing stream fixture

**Gain: All 7 bugs fixed. Zero domain logic bugs found — the domain layer is clean.**

### 1.3 Evidence-Based Refactoring

S110 produced 13 classified findings. S111 applied 4 targeted refactors:

| Refactor | Impact |
|----------|--------|
| Signal publisher correlation_id | Observability parity across 5 publishers |
| Projection stats normalization | Consistent shutdown invariant checking across 7 projections |
| Raccoon-CLI dead code cleanup | 26 warnings → 0 |
| Generic UseCase factory | ~150 LOC eliminated; new configctl operations: 5 lines instead of 30 |

**Gain: Refactors were small, precise, and justified by concrete evidence — not by aesthetic preference.**

### 1.4 Friction Catalog

The slice produced a prioritized friction catalog (F01–F13) with clear severity levels. This catalog is now the authoritative source for deciding what to fix next.

**Gain: Future investment decisions are data-driven, not intuition-driven.**

### 1.5 Governance Validation

raccoon-cli's 950 tests validated that structural invariants hold across the full codebase after all changes. The expansion playbooks and architecture decision records were stress-tested by the slice process.

**Gain: Governance tooling works and catches drift before it compounds.**

---

## 2. Trade-offs

### 2.1 Structural Proof vs. Operational Proof

The slice validated architecture through code review, unit tests, and static analysis. It did **not** validate the pipeline through live execution with real NATS messaging and real market data.

**Trade-off:** High structural confidence, but operational behavior (timing, failure recovery, cold-start) remains unproven. This was a deliberate scope decision (S108 defined the slice as architectural validation, not operational testing).

**Impact:** The next wave must include a live pipeline run to close this gap. Until then, confidence is bounded by what static analysis can prove.

### 2.2 Depth vs. Breadth of Slice

The slice exercised **all 8 domain families** but only **1 binding** (`binancef.btcusdt.60`). Multi-symbol, multi-timeframe, and multi-source behaviors were explicitly excluded.

**Trade-off:** The pipeline path is validated end-to-end, but scaling behavior and multi-binding interaction are not.

**Impact:** Adding a second binding should be low-risk (config-driven activation handles it), but edge cases in concurrent binding management are untested.

### 2.3 Refactor Scope Discipline

S111 applied only 4 of 13 findings. The remaining 9 were deliberately deferred with documented rationale.

**Trade-off:** Some friction remains (query client boilerplate, publisher duplication, untested actors). The cost of carrying this friction is lower than the cost of premature abstraction.

**Impact:** The deferred items are tracked and will be re-evaluated when concrete triggers are met (e.g., publisher generic extraction deferred until a 6th publisher is added).

### 2.4 Documentation Volume

The S96–S112 arc produced 40+ architecture documents and 17 stage reports. This volume provides comprehensive traceability but imposes a maintenance burden.

**Trade-off:** Every decision is traceable, but document staleness risk increases with volume.

**Impact:** Documents should be treated as **snapshots, not living contracts**. The code and raccoon-cli governance rules are the source of truth; documents provide context and rationale.

### 2.5 Generic UseCase Partial Application

The generic UseCase factory (S111) was applied only to configctlclient. The other 6 client modules still use per-file manual wiring.

**Trade-off:** Inconsistency between configctlclient (generic) and other clients (manual). But each query client has unique validation requirements that make a single generic less clean.

**Impact:** Extending generics to query clients is a P1 debt. The cost of carrying the inconsistency is low (each client works correctly), but the maintenance multiplier for adding new query operations remains higher than necessary.

---

## 3. Open Debts

### 3.1 Priority 0 — Safety-Critical

| ID | Debt | Risk | Trigger |
|----|------|------|---------|
| **D1** | Execute actor unit tests | Kill switch, staleness guard, and timeout logic are untested. A bug here could allow unintended order execution. | Must resolve before any expansion touching the execution pipeline. |

### 3.2 Priority 1 — High Maintenance Burden

| ID | Debt | Risk | Trigger |
|----|------|------|---------|
| **D3** | Query client generics | 6 client modules with per-file boilerplate. Each new query operation requires ~30 lines of copy-paste. | Resolve when next query operation is added to any client. |
| **PD-1** | Composition root integration tests | No automated test verifies that `cmd/*/run.go` wires dependencies correctly. Wiring bugs caught only at manual startup. | Resolve when a composition bug escapes to manual testing. |

### 3.3 Priority 2 — Moderate Burden

| ID | Debt | Risk | Trigger |
|----|------|------|---------|
| **D2** | Publisher actor generic extraction | 5 publisher actors share identical `Receive()` logic. Adding a 6th publisher means copy-pasting the same pattern. | Resolve when 6th publisher is added. |
| **D4** | Ingest actor unit tests | 611 LOC of dynamic exchange scope creation with no unit tests. | Resolve when ingest behavior is modified. |
| **D5** | Configctl actor unit tests | 612 LOC of control routing dispatch with no unit tests. | Resolve when configctl actor logic is modified. |
| **PD-2** | Cross-registration coherence | Derive processor families, store pipeline families, and settings catalog maintained independently. A family could be registered in one place but missing from others. | Resolve when next family is added. |

### 3.4 Priority 3 — Acceptable for Now

| ID | Debt | Risk | Trigger |
|----|------|------|---------|
| **D6** | Route registration boilerplate | Manageable at 7 families. | Revisit when family count exceeds 12. |
| **D7** | Gateway wiring repetition | Explicit wiring serves as documentation. | Revisit when family count exceeds 12. |
| **D8** | Derive-configctl dependency model | Correct eventual consistency. Not a problem. | Revisit only if operational issues arise. |
| **PD-3** | Correlation ID in structured logs | Cross-runtime tracing requires manual timestamp correlation. | Resolve when cross-runtime debugging becomes the primary bottleneck. |
| **PD-4** | Error classification taxonomy | Free-form error strings. | Resolve when alerting infrastructure exists. |
| **PD-5** | Raccoon-CLI governance constants | ~50 lines per new domain. | Include in expansion playbook (already documented). |

### 3.5 Debts That Do NOT Justify Investment Now

| Item | Why Not |
|------|---------|
| ClickHouse projection layer | No analytical queries needed yet; NATS KV sufficient for current read models |
| Event schema formalization (Protobuf/Avro) | Internal system with single producer per event type; JSON envelope is adequate |
| OpenTelemetry / distributed tracing | Log-based debugging has not been proven insufficient |
| Full E2E test suite | Manual validation sufficient at current scale; automate when pipeline runs regularly |
| Multi-binding stress testing | Single-binding must run live first |
| Performance baseline | No production load to measure against |
