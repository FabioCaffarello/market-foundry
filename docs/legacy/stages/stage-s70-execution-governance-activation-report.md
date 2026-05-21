# Stage S70 — Execution Governance Activation Report

## Executive Summary

S70 placed the `execution` domain under active governance in `raccoon-cli` before any implementation code exists. The CLI now enforces 7-document presence, premature entry prevention (stream, subject, adapter, domain, actor, HTTP, config), and has 4 prepared drift checks ready for S75 activation. Additionally, risk domain drift checks (prepared in S63) were activated to reflect S64's completed implementation.

---

## Pre-Condition Verification

S69 (Execution Domain Design) produced:
- `execution-domain-design.md`
- `execution-stream-families.md`
- `execution-activation-and-ownership.md`
- `execution-query-surface-guidelines.md`
- `execution-readiness-review.md` (S68)
- `execution-entry-prerequisites.md` (S68)
- `execution-risks-and-blockers.md` (S68)

Pre-condition satisfied. Design is complete and formal.

---

## Changes Made

### 1. Execution Governance — Premature Entry Prevention (Active)

**Constants added to `drift_detect.rs`:**

| Constant | Count | Purpose |
|----------|-------|---------|
| `EXECUTION_DOCS` | 7 files | Design documentation that must exist |
| `EXECUTION_EXPECTED_SUBJECTS` | 2 subjects | `execution.events.paper_order.submitted`, `execution.query.paper_order.latest` |
| `EXECUTION_EXPECTED_DURABLES` | 1 durable | `store-execution-paper-order` |
| `EXECUTION_EXPECTED_BUCKETS` | 1 bucket | `EXECUTION_PAPER_ORDER_LATEST` |
| `EXECUTION_ADAPTER_FILES` | 5 files | Registry, publisher, consumer, gateway, KV store |
| `EXECUTION_DOMAIN_FILES` | 6 files | Entity, events, evaluator, client contracts, use case, port |

**Active checks (Phase 7 in `analyze()`):**

| Check | What it does |
|-------|-------------|
| `check_execution_docs_drift` | Verifies 7 S69 architecture docs exist |
| `check_execution_premature_implementation` | Blocks adapter, domain, actor, HTTP, and config files |

**Prohibited stream entry:**
- `EXECUTION_EVENTS` added to `PROHIBITED_STREAMS`
- Blocks stream name and `execution.events.*` / `execution.query.*` subjects in Go source

### 2. Execution Governance — Prepared Drift Checks (Dead Code for S75)

| Function | What it will check |
|----------|-------------------|
| `check_execution_adapter_drift` | 5 NATS adapter files exist |
| `check_execution_domain_drift` | Domain entity, events, evaluator, actors, HTTP files exist |
| `check_execution_config_drift` | Symmetric `execution_families` in derive.jsonc and store.jsonc |
| `check_execution_contracts_drift` | Subjects, durables, KV buckets in Go source |

### 3. Risk Governance Activation (S64 Catch-Up)

Risk was implemented in S64 but drift checks remained dormant. S70 activates them:

| Change | Detail |
|--------|--------|
| `RISK_EVENTS` moved to `CANONICAL_STREAMS` | Was in `PROHIBITED_STREAMS` |
| `check_risk_premature_implementation` removed | Replaced by active checks |
| `check_risk_adapter_drift` activated | Verifies 5 NATS adapter files |
| `check_risk_domain_drift` activated | Verifies domain, actor, HTTP files |
| `check_risk_config_drift` activated | Verifies symmetric risk_families config |
| `check_risk_contracts_drift` activated | Verifies subjects, durables, KV buckets |
| `#[allow(dead_code)]` removed | From all 4 risk drift functions |

**Test helpers updated:**
- `make_source_topology()` now includes `RISK_EVENTS` stream, `store-risk-position-exposure` durable, and risk/strategy subjects

---

## Rules Added to CLI

| Rule ID | Name | Phase | Severity |
|---------|------|-------|----------|
| ED-1 | `execution-docs-drift` | Active (S70) | Error |
| ED-2 | `execution-premature-implementation` | Active (S70) | Error |
| ED-3 | `execution-adapter-drift` | Prepared (S75) | Error |
| ED-4 | `execution-domain-drift` | Prepared (S75) | Error |
| ED-5 | `execution-config-drift` | Prepared (S75) | Error/Warning |
| ED-6 | `execution-contracts-drift` | Prepared (S75) | Error |

---

## Files Changed

### Code Changes

| File | Change |
|------|--------|
| `tools/raccoon-cli/src/analyzers/drift_detect.rs` | Added execution constants (7 const blocks), 2 active checks, 4 prepared checks; activated risk drift checks; updated CANONICAL_STREAMS and PROHIBITED_STREAMS; updated test helpers |

### Documentation (New)

| File | Purpose |
|------|---------|
| `docs/tooling/cli-execution-guardrails.md` | 10 guardrails (EG-01 through EG-10) with CLI check mappings |
| `docs/tooling/cli-execution-drift-rules.md` | 6 drift rules (ED-1 through ED-6) with activation checklist |
| `docs/stages/stage-s70-execution-governance-activation-report.md` | This report |

---

## Governance Gaps That Remain

### What the CLI Cannot Yet Protect

| Gap | Why | Mitigation |
|-----|-----|------------|
| EBI-2: Pure function enforcement | No Go function body analysis; codeintel indexes signatures only | Code review; test coverage (S75) |
| EBI-3: Actor message primitive types | Cannot verify message field types without AST body analysis | Code review; integration test (S71) |
| EBI-7: No venue API interaction | Cannot detect HTTP client imports within execution scope | `arch-guard` catches infra imports in domain layer; code review for adapters |
| EBI-8: RiskInput domain ownership | Cannot verify struct field-level isolation without type resolution | Code review; `arch-guard` import rule covers package-level isolation |
| EBI-9/10: No aggregation, no cumulative state | Semantic constraints beyond structural checks | Code review; test design (S75) |
| Cross-domain import within execution | `arch-guard` enforces layer rules but execution actors don't exist yet | Activates automatically when files are created in S75 |
| Ownership enforcement (single-writer) | CLI verifies file existence, not runtime publish/consume patterns | Runtime smoke test; integration test (S71) |

### Known Limitations

1. **Execution history bucket**: S69 deferred history projection design to S72. The CLI has no governance for execution history until that design is finalized.
2. **Venue adapter families (EF-02, EF-03)**: Only `paper_order` is governed. Future families will need additional constants when designed (S77+).
3. **Kill switch / circuit breaker**: S76 design concerns not yet in scope for CLI governance.

---

## Impact on Readiness for S71-S75

### S71: Automated Traceability Verification
- No direct impact. S71 is an integration test stage.
- Execution governance ensures trace metadata requirements (CorrelationID, CausationID) are documented in S69 design docs, which the CLI now protects.

### S72: Trace Metadata Persistence Design
- No direct impact. S72 decides where to persist trace metadata.
- Execution history bucket design depends on S72 outcome.

### S75: Execution Implementation (First Slice)
- **Directly enabled.** S75 can proceed knowing:
  - All 7 design docs are protected and will not silently disappear
  - No premature execution code can slip in before S75 formally opens
  - The S75 activation checklist is documented in `cli-execution-drift-rules.md`
  - 4 drift check functions are pre-written and ready to activate
  - Risk governance is now fully active (no governance debt from S64)
- **S75 activation cost:** 7 lines changed in `drift_detect.rs` (remove from PROHIBITED, add to CANONICAL, swap checks in `analyze()`)

---

## Stage Closure Checklist

- [x] Single structural capability declared (execution governance activation)
- [x] drift_detect.rs compiles cleanly (cargo check)
- [x] All 29 drift_detect tests pass (cargo test)
- [x] No execution implementation code added
- [x] Execution docs, subjects, families, buckets governed
- [x] Premature entry prevention active (stream, subject, adapter, domain, actor, HTTP, config)
- [x] Prepared drift checks for S75 (4 functions with `#[allow(dead_code)]`)
- [x] Risk governance activated (S64 catch-up)
- [x] RISK_EVENTS moved from PROHIBITED_STREAMS to CANONICAL_STREAMS
- [x] Guardrails documented (10 guardrails in cli-execution-guardrails.md)
- [x] Drift rules documented (6 rules in cli-execution-drift-rules.md)
- [x] S75 activation checklist documented
- [x] Governance gaps honestly recorded
- [x] No refactor of runtime code
- [x] No decorative checks added
- [x] Execution included in quality-gate fast/ci/deep profiles (via drift-detect)
