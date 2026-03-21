# Codegen Reentry Wave: Gains, Trade-offs, and Open Debts

**Stage:** S263
**Wave:** Codegen Reentry (S258–S262)
**Date:** 2026-03-21

---

## 1. Gains

### 1.1 Coverage Expansion

| Metric | Before (S257) | After (S262) | Delta |
|--------|---------------|--------------|-------|
| Families governed | 2 (RSI, EMA) | 11 (all tier-1 + Bollinger) | +9 |
| Codegen entries in manifest | 4 | 22 | +18 |
| Golden snapshots | 4 | 22 | +18 |
| Codegen-first families | 0 | 1 (Bollinger) | +1 |
| Equivalence checks (automated) | 0 | 109 | +109 |
| Domain layers with codegen governance | 1 (signal) | 6 (all layers) | +5 |

### 1.2 Tooling and Automation

- **codegen-integrated-check.sh** — manifest-driven validation of production code against golden snapshots. Hardened with awk exact-match to prevent substring collisions (e.g., `rsi` vs `rsi_oversold`).
- **codegen-equivalence-check.sh** — 7-phase automated equivalence framework covering golden snapshots, integrated slices, spec validity, cross-artifact consistency, store coexistence, starter/mapper existence, and config methods.
- Both scripts are CI-ready and repeatable.

### 1.3 Spec Reconciliation

- All 11 family specs validate cleanly (zero collisions).
- Column-opaque design absorbed breadth wave enrichments (severity, rationale, confidence columns) without spec or template changes.
- Specs are proven source of truth for governed artifacts.

### 1.4 Codegen-First Proof

- Bollinger Bands bootstrapped entirely from spec → golden → markers → production → domain logic.
- Zero changes to codegen tooling or templates required.
- 6 domain-logic tests covering algorithm correctness and edge cases.
- Pattern is documented and reproducible for future families.

### 1.5 Boundary Clarity

- Generated/manual boundary is explicit: codegen governs consumer_spec + pipeline_entry; humans govern everything else.
- Markers create clear ownership zones — no overlap, no ambiguity.
- Behavioral tests (47) form hard regression gate throughout.

---

## 2. Trade-offs

### 2.1 Column-Opaque Spec Design

**What we gained:** Resilience to domain enrichment — specs didn't need changes when breadth wave added columns.

**What we gave up:** Type validation, mapper generation, DDL consistency checks. The spec treats `writer.columns` as a free-form string. Codegen cannot detect if a column was added to the INSERT but not to the DDL, or if a column type changed.

**Severity:** Medium. Acceptable while the artifact surface is small (22 items), but becomes a liability if codegen expands to typed mappers or store consumers.

### 2.2 Narrow Artifact Coverage

**What we gained:** Clean, provably correct governance of the 2 most repetitive artifact types.

**What we gave up:** The remaining ~118 artifacts stay manual with no codegen guardrails. Store consumers, layer starters, mappers, config methods, registry entries, domain constructors, and actor wiring are all hand-written.

**Severity:** Low for now. These artifacts are less repetitive and more semantically rich — they benefit less from codegen. But as families scale, the manual surface grows linearly.

### 2.3 Factory-to-Expanded Migration

**What we gained:** Expanded struct literals are diff-friendly and directly comparable to golden snapshots.

**What we gave up:** The factory pattern (`NewConsumerSpec(...)`) was more compact. Some readers may find the expanded form verbose.

**Severity:** Negligible. The expanded form is strictly more readable and maintainable.

### 2.4 Manual Marker Placement

**What we gained:** Simplicity — no need for AST manipulation or code rewriting tools.

**What we gave up:** Every new family requires a human to place `codegen:begin` / `codegen:end` markers in the correct files. This is a friction point in the codegen-first workflow.

**Severity:** Low. Marker placement takes seconds and is validated by the integrated-check script.

### 2.5 Single Codegen-First Proof

**What we gained:** Bollinger proved the pattern works.

**What we gave up:** A single proof point doesn't confirm the pattern works for all family shapes. Decision/strategy/risk/execution families may have nuances that Bollinger (signal layer) doesn't expose.

**Severity:** Low-Medium. Bollinger was the right choice for first proof (tier 1, signal layer, minimal dependencies), but the pattern should be confirmed on at least one non-signal family before scaling.

---

## 3. Open Debts

### 3.1 Codegen-Specific Debts

| ID | Description | Impact | Blocked By | Recommended Action |
|----|-------------|--------|------------|-------------------|
| OD-CG1 | Column-opaque spec: no type validation, no mapper generation | Medium | Spec schema evolution | Defer until mapper generation is attempted |
| OD-CG2 | Store consumer specs not codegen-governed | Low | Independent config divergence by design | Evaluate if store pattern is stable enough to template |
| OD-CG3 | Marker placement is manual | Low | Tooling investment | Accept as friction; automate only if family creation rate justifies |
| OD-CG4 | No codegen for registry non-writer entries (EventSpec, ControlSpec) | Low | Template complexity | Defer; these entries are per-family but semantically varied |
| OD-CG5 | No codegen for config registration (knownFamilies, dependsOn) | Low | Config schema complexity | Defer; config rules encode domain semantics, not structure |

### 3.2 Inherited Debts from Prior Waves

| ID | Description | Impact | Origin |
|----|-------------|--------|--------|
| OD-BW2 | Configurable scaling infrastructure absent (AckWait, MaxDeliver hardcoded) | Medium | Behavioral wave |
| OD-BW4 | Remaining actor chain wiring (incomplete) | Low | Behavioral wave |
| OD-BW5 | Performance budgets undefined | Low | Behavioral wave |
| OD-BW6 | configctl tooling absent | Low | Behavioral wave |

### 3.3 Items Explicitly Deferred

- **New artifact templates:** Not created. Store consumers, starters, mappers, and config methods remain candidates for future codegen expansion but were out of scope.
- **Multi-layer codegen-first proof:** Only signal layer was tested. Decision, strategy, risk, and execution layers were not validated with a codegen-first workflow.
- **DDL generation:** Not attempted. ClickHouse migrations remain manual.
- **JSON payload schema validation:** Not attempted. Payload structure lives in domain logic, outside codegen boundary.

---

## 4. Net Assessment

The reentry wave invested 5 stages into codegen and returned:

- **22 governed artifacts** (up from 4) with zero drift.
- **Automated equivalence validation** (109 checks, 7-phase framework).
- **Proof that codegen-first works** for the simplest case (signal family).
- **Clean boundary** between generated and manual code.

The investment was proportional to the return. The codegen system is healthier, better validated, and more trustworthy than before the wave. But it remains a narrow tool — governing 14% of the artifact surface — and should not be confused with a comprehensive generation system.

The open debts are real but manageable. None blocks the next wave. The most consequential debt (OD-CG1: column-opaque spec) only matters if the Foundry decides to expand codegen to typed artifacts, which is not recommended as the immediate next step.
