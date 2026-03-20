# Wave B After Family 02 and Hardening — Gains, Trade-offs, and Open Debts

> **Purpose:** Explicit accounting of what the Wave B expansion pattern has gained, what it traded away, and what debts remain after two family expansions and one hardening tranche.

---

## 1. Gains

### G-1: Struct DI Eliminates Constructor Churn

**Before:** `NewAnalyticalWebHandler` accepted N positional arguments. Every family added one parameter, changed the signature, and broke all callers (compose, routes, 20 test constructors).

**After:** `NewAnalyticalWebHandler(deps AnalyticalHandlerDeps)` accepts a single struct. Adding Family N+1 requires one struct field, zero signature changes, zero caller updates beyond the wiring site.

**Payoff:** Real and structural. At 3 families (4 args), the positional pattern was approaching fragility. At 4+ families, it would have been brittle. The struct scales linearly to any number of families.

### G-2: Smoke Extraction Reduces Family Addition Cost by 91%

**Before:** Each family added ~80 lines of copy-paste-adjust validation in the smoke script. Divergence between families was invisible until failure. Manual editing introduced silent drift.

**After:** `validate_analytical_family()` is called with 6 parameters. Adding a family costs ~7 lines. Validation logic is shared, consistent, and maintained in one place.

**Payoff:** Real and operational. The smoke script was at 614 lines and growing linearly. At 5 families it would have been >800 lines of repetitive validation. The extraction makes the growth curve flat.

### G-3: Helper Naming Matches Actual Scope

**Before:** `parseEvidenceKeyParams` was consumed by 7 handler families (operational + analytical) but named after one specific family. Developers had to mentally translate the name when reusing it.

**After:** `parseQueryKeyParams` is family-neutral. The name describes what the function does (parses query key parameters), not where it was first created.

**Payoff:** Low but correct. This was the cheapest item in the tranche and the right time to do it. Waiting longer would have made the rename touch more files.

### G-4: 9-Artifact Pattern Proven Across Two Distinct Data Shapes

**Family 01 (Signals):** 12 columns, 1 JSON column (map), no domain-specific query filters.
**Family 02 (Decisions):** 15 columns, 2 JSON columns (array + map), 1 enum-like column (outcome), 1 domain-specific filter.

The pattern handled JSON shape diversity, column count variation, and optional parameter extension without structural changes. Write path required zero modifications across both expansions.

### G-5: Formal Process Governs Expansion

The Wave B expansion is now governed by:
- Pattern v2 document with hardening thresholds
- Comprehensive checklist with entry/exit criteria
- 5-point gate review (tests, smoke, CI, no regressions, schema coherence)
- Family-indexed hardening triggers (D-4 codegen at Family 4)
- Constraint set (C-1 through C-9) and non-goal set (NG-1 through NG-8)
- Friction threshold rule (>2 new frictions triggers pause)

### G-6: Optionality Invariant Holds After 3 Tables

ClickHouse is not in the gateway readiness check. All analytical endpoints return 503 when ClickHouse is unavailable. Operational services function identically with or without the analytical layer. This invariant has been tested across candles, signals, and decisions.

---

## 2. Trade-offs

### T-1: Manual Duplication Accepted Over Premature Codegen

**Given up:** DRY principle. ~80% of code across families is mechanically identical.
**Gained:** Full family independence. No shared abstractions that could couple families or create update cascades.
**Still acceptable because:** At 3 families, the duplication cost is manageable. Each family is a self-contained unit that can be understood, tested, and debugged independently. Codegen evaluation is committed at Family 4.
**Risk if not addressed:** At 5+ families, mechanical copy errors become likely. Manual verification burden scales linearly.

### T-2: Review-Enforced Schema Coherence Accepted Over Compile-Time Enforcement

**Given up:** Guaranteed correctness at compile time. Schema alignment across DDL → writer → reader is verified by unit test assertions, not the compiler.
**Gained:** Simplicity. No build tooling, no code generation, no custom linter.
**Still acceptable because:** At 9 tables (3 families × 3 locations), review cost is manageable. Unit tests catch most drift. Developer attention is sufficient.
**Risk if not addressed:** At ~12 tables, review fatigue may allow silent drift. Consider compile-time enforcement if families exceed 4.

### T-3: Sticky Degradation Accepted Over Auto-Recovery

**Given up:** Automatic recovery from ClickHouse outages. A writer consumer that enters degraded state stays degraded until manual restart.
**Gained:** Simple mental model. No reconnection logic, no circuit breakers, no recovery timing bugs.
**Still acceptable because:** At current scale (single operator, ~6 pipelines), manual intervention is practical and predictable.
**Risk if not addressed:** If the system runs unattended for extended periods, a ClickHouse blip causes data loss until someone notices and restarts.

### T-4: No CI Smoke Integration Accepted Over Infra Complexity

**Given up:** End-to-end integration testing in CI pipeline. Analytical smoke tests require ClickHouse + NATS + writer + gateway — too heavy for GitHub Actions free tier.
**Gained:** Fast CI iteration. Unit tests run in seconds, no Docker-in-Docker.
**Still acceptable because:** Unit tests cover handler/use-case/reader contract boundaries. Schema coherence is testable without infrastructure. Manual smoke testing catches integration gaps.
**Risk if not addressed:** Regressions in the integration path (e.g., ClickHouse query syntax, writer-reader alignment) can ship undetected.

### T-5: Silent Mapper Fallbacks Accepted Over Strict Parsing

**Given up:** Data integrity guarantee at write time. Malformed events produce zero-value fields, not errors.
**Gained:** Pipeline resilience. One bad event doesn't halt ingestion for the entire stream.
**Still acceptable because:** Analytical layer is observational, not authoritative. Data has 90-day TTL. Malformed events are logged at WARN level.
**Risk if not addressed:** Silent data corruption could produce misleading analytical results without visible symptoms.

---

## 3. Open Debts

### Debts with Committed Resolution Points

| ID | Debt | Severity | Trigger | Notes |
|----|------|----------|---------|-------|
| D-4 | No codegen — 80% mechanical duplication | Medium | Family 4 | Evaluate whether code generation or templating reduces expansion cost |

### Debts Without Committed Resolution Points

| ID | Debt | Severity | Notes |
|----|------|----------|-------|
| D-5 | No backoff jitter in writer retry | Low | Thundering herd improbable at 6–9 pipelines |
| D-6 | No NATS consumer lag visibility | Medium | Buffer overflow risk invisible; monitoring gap |
| D-7 | Sticky degradation without auto-recovery | Medium | Manual restart required; acceptable at current scale |
| D-8 | No load testing baseline | Medium | Performance problems discovered late |
| D-9 | No pagination beyond 500 rows | Low | Hard limit acceptable for dashboard queries |
| D-10 | Metadata schema not validated at read | Low | Empty/malformed metadata rendered as-is |
| D-11 | Schema coherence review-enforced, not compile-time | Medium | Revisit at ~12 tables |
| D-12 | ClickHouse client timeout not configurable | Low | Hardcoded default; sufficient at current load |
| PF-4 | Outcome filter case-sensitive, unvalidated | Low | Returns empty, no security risk |
| PF-5 | No CI integration for analytical smoke | High | Architectural gap; integration regressions undetected in CI |

### Debts Explicitly Deferred (No Current Cost)

These items are documented as out of scope and carry no active cost:

- Dead-letter queue for failed events
- Prometheus/Grafana external observability
- Cold-start bootstrap / backfill
- Per-family batch configuration
- Event schema versioning
- Materialized views / aggregation tables
- Cross-family joins / composite queries
- Multi-instance ClickHouse
- Real-time streaming queries
- Dynamic pipeline registration

---

## 4. Debt Trajectory

| Milestone | Active Debts | Notes |
|-----------|-------------|-------|
| After Family 01 (S167) | 12 | 4 with committed triggers |
| After Family 02 (S170) | 14 | +2 new frictions (PF-4, PF-5) |
| After Hardening (S172) | 11 | -3 resolved (H-1, H-2, H-3) |
| After Family 03 (projected) | 11 | No new resolutions expected; D-4 evaluation triggered at Family 4 |
| After Family 04 (projected) | 10 or fewer | D-4 resolution applied; codegen may eliminate D-11 |

The debt count is stable. The hardening tranche reduced debts by 3 and the trajectory shows no accumulation pressure. The risk items (D-6 consumer lag, D-7 sticky degradation, PF-5 CI smoke) are carrying risk but not blocking expansion.

---

## 5. Items That Do Not Justify Their Cost Now

| Item | Why Not Now |
|------|------------|
| Codegen/templating | At 3 families, manual cost is ~30 minutes per family. Codegen infrastructure cost exceeds savings until Family 5+ |
| Compile-time schema coherence | Requires build tooling or code generation. At 9 tables, unit test assertions are sufficient |
| Auto-recovery for writer consumers | Circuit breaker + reconnection logic adds complexity disproportionate to current failure rate |
| CI smoke integration | Requires Docker-in-Docker or dedicated infra. Unit tests cover contract boundaries adequately |
| Consumer lag monitoring | Requires NATS JetStream API integration. Value is real but not blocking |
| Pagination | 500-row hard limit covers all current use cases. Implementation is straightforward when needed |
| Operational handler struct DI | Operational handlers have no expansion pressure. Converting them would be change for consistency's sake, not necessity |
