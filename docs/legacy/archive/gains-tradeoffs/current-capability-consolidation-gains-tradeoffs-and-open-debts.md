# Current Capability Consolidation — Gains, Trade-offs, and Open Debts

> Honest accounting of what the consolidation wave (S137–S141) delivered, what it cost, and what remains unresolved.

---

## 1. Gains

### G-01: Canonical Baseline Definition
**What:** The operational baseline is formally defined with 30 pass/fail success criteria across 5 validation tiers.
**Why it matters:** Before S137, the system worked but the definition of "working" was implicit. Now there is a testable contract for what the Foundry does and does not guarantee.
**Evidence:** `current-capability-baseline-definition.md`, `current-capability-baseline-success-criteria.md`

### G-02: Operational Phase Classification
**What:** Every service exposes `/statusz` with automatic phase computation: starting → warming → active → idle → stalled.
**Why it matters:** Operators can distinguish between a system that is cold-starting, warming up, running normally, or degraded — without reading logs.
**Evidence:** `/statusz` endpoint, `healthz.go` phase computation logic

### G-03: Diagnostic Surface
**What:** Four diagnostic endpoints (`/healthz`, `/readyz`, `/statusz`, `/diagz`) on every service, plus `diag-check.sh` for quick stack health snapshots.
**Why it matters:** Before S139, diagnosing a problem required reading structured logs. Now there are dedicated machine- and human-readable diagnostic surfaces.
**Evidence:** `current-baseline-operational-diagnostics.md`, `scripts/diag-check.sh`

### G-04: Recovery Semantics Clarity
**What:** Explicit documentation of what survives restart (NATS streams, KV, consumer positions, config) versus what is lost (in-memory samplers, in-flight orders, tracker counters). Shutdown bounded to 15 seconds.
**Why it matters:** Before S140, recovery behavior was knowable only by reading code. Now operators and architects have a clear survival/loss matrix.
**Evidence:** `current-baseline-recovery-and-restart-semantics.md`

### G-05: Accepted Limitations Registry
**What:** Five accepted limitations (L-01 through L-05) formally documented with explicit rationale for acceptance.
**Why it matters:** Transforms implicit "we haven't done that yet" into deliberate "we accept this because X, and the trigger for changing it is Y."
**Evidence:** S140 report, `current-baseline-recovery-and-restart-semantics.md`

### G-06: Shared Script Infrastructure
**What:** `scripts/utils/lib.sh` provides canonical logging, JSON helpers, compose helpers, and default constants shared across all scripts.
**Why it matters:** Eliminates copy-paste drift between scripts. New scripts inherit consistent behavior. Port mappings, service lists, and timeframe arrays are defined once.
**Evidence:** `scripts/utils/lib.sh`

### G-07: Self-Documenting Configuration
**What:** `CONFIG-REFERENCE.md` documents every configuration field with types, defaults, constraints, and cross-layer dependencies. All JSONC files have inline comments.
**Why it matters:** Operators can configure the system without reading Go source code. Cross-layer dependency chains are visible without running the system.
**Evidence:** `deploy/configs/CONFIG-REFERENCE.md`, all `*.jsonc` files

### G-08: ClickHouse Entry Governance
**What:** Seven entry principles (P-01 through P-07) and migration catalog organization guidelines defined before any ClickHouse code is written.
**Why it matters:** Prevents ad-hoc ClickHouse integration. Schema, writer, and migration tooling will follow documented conventions from day one.
**Evidence:** `future-clickhouse-and-migrations-entry-principles.md`, `future-migration-catalog-organization-guidelines.md`

### G-09: Persistence Trigger Decision Matrix
**What:** Five pain points (P-01 through P-05) catalogued with explicit trigger thresholds for when ClickHouse persistence becomes justified.
**Why it matters:** The decision to introduce ClickHouse will be evidence-driven, not speculative. Each trigger has measurable thresholds.
**Evidence:** `future-state-persistence-and-clickhouse-trigger-notes.md`

---

## 2. Trade-offs

### T-01: Documentation Volume
**Cost:** 14+ architecture documents, 6 stage reports, CONFIG-REFERENCE, runbook — significant documentation surface to maintain.
**Justification:** The alternative was implicit knowledge that would be lost between sessions and between contributors. The documentation cost is front-loaded; maintenance cost is low if documents are treated as living references, not static artifacts.
**Mitigation:** Documents are organized by concern, not chronology. Each has a clear scope and owner (the stage that produced it).

### T-02: Validation Remains Manual
**Cost:** The 30 success criteria and 5-tier validation are documented as manual procedures, not automated CI checks.
**Justification:** Automating live pipeline validation requires infrastructure (test harness, timing coordination, NATS state verification) that exceeds the consolidation scope. The manual procedures are reliable and repeatable.
**Risk:** Manual validation may drift from documented procedures over time.

### T-03: No Code Changes to Core Pipeline
**Cost:** Consolidation deliberately avoided modifying the core pipeline (ingest → derive → store → execute). This means identified friction points (per-TF idle detection, gateway trackers, query observability) remain unaddressed.
**Justification:** The consolidation wave's mandate was to document and govern, not to change. Changing core pipeline during consolidation would have undermined the baseline definition.

### T-04: ClickHouse Preparation Is Documentation-Only
**Cost:** Entry principles, catalog guidelines, and signal candidates exist as documents. No migration tool, no schema, no writer. The gap between "documented intent" and "working infrastructure" is real.
**Justification:** Implementing ClickHouse tooling during a consolidation wave would violate the wave's charter. The documentation establishes guardrails that prevent rushed implementation later.

---

## 3. Open Debts

### Critical (block future expansion)

| ID | Debt | Impact | Trigger |
|---|---|---|---|
| OD-01 | **No state persistence for in-memory samplers** | Candle samplers and RSI accumulators are lost on restart. Cold-start warm-up for RSI at 3600s takes ~15 hours. | Hard gate for TC-02 (more timeframes) and for ClickHouse writer (which needs stable event history) |
| OD-02 | **No migration tooling** | `cmd/migrate` does not exist. Cannot create, apply, or verify ClickHouse schemas. | Hard gate for any ClickHouse implementation |

### Significant (affect quality at scale)

| ID | Debt | Impact | Trigger |
|---|---|---|---|
| OD-03 | **Gateway lacks tracker integration** | `/statusz` and `/diagz` on gateway return empty tracker data. Gateway health is inferred from downstream services. | Becomes problematic when gateway handles multiple consumers or needs independent health assessment |
| OD-04 | **No automated baseline validation** | 30 success criteria are manual checks. Regression detection depends on human discipline. | Becomes costly as the system grows and changes more frequently |
| OD-05 | **No query observability** | No request latency, throughput, or error rate metrics on the HTTP query surface. | Becomes critical when external consumers exist or SLA expectations are set |

### Acceptable (known, bounded, low urgency)

| ID | Debt | Impact | Trigger |
|---|---|---|---|
| OD-06 | Per-timeframe idle detection | All timeframes share a single idle threshold. 3600s candles would trigger false idle warnings. | Only relevant with TFs > current max (3600s) |
| OD-07 | RSI convergence formal proof | Empirically observed but not mathematically proven. | Only relevant for compliance or formal verification contexts |
| OD-08 | Per-binding timeframe customization | Global timeframe list applies to all symbols. | Only relevant when symbols need different temporal resolutions |
| OD-09 | Gateway aggregate view | No single endpoint showing cross-service status. | Only relevant at higher cardinality (10+ symbols) |
| OD-10 | Null response disambiguation | Empty query results don't distinguish "no data yet" from "unknown key." | Operator can verify via config; no external consumers yet |

---

## 4. Items That Do NOT Justify Cost Now

The following improvements were considered during consolidation and explicitly deemed not worth pursuing at this stage:

1. **Automated CI pipeline validation** — The 30 success criteria could theoretically be automated, but the infrastructure cost (test harness with timing, NATS state verification, multi-minute validation runs) exceeds the benefit for a single-developer project with manual validation taking 20 minutes.

2. **Per-timeframe tracker granularity** — Splitting trackers per timeframe would improve idle detection accuracy but adds complexity to every service. Not justified until timeframes exceed 3600s.

3. **Persistent diagnostic history** — Storing `/diagz` snapshots over time would enable trend analysis but requires either a file-based logger or ClickHouse — both premature.

4. **Request tracing / distributed tracing** — Would improve cross-service debugging but the current pipeline is small enough that structured logs + NATS subject inspection suffice.

5. **Hot-reload of configuration** — Currently requires restart. Config changes are infrequent enough that restart cost is acceptable.
