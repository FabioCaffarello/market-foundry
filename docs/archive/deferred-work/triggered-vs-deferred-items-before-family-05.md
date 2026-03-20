# Triggered vs Deferred Items Before Family 05

## Purpose

Classify all tracked items into triggered (requiring action), deferred with committed triggers, and deferred without triggers — establishing what Family 05 inherits and what constraints it operates under.

---

## Triggered Items (3)

### TRIG-1: Codegen Evaluation — Complete, Implementation Deferred

- **Status**: TRIGGERED at S178 (D-4), evaluated, implementation deferred to Family 06 boundary.
- **Evidence**: 5 families produce ~800 lines mechanical duplication. Readers ~143 LOC each (80% identical), handlers ~100 LOC each (85% identical), use cases ~128 LOC each (70% identical).
- **Evaluation result**: Justified but not cost-effective until Family 06. At 5 families, manual duplication cost < template maintenance cost. At 6+, equation inverts.
- **Family 05 impact**: None. Proceeds without codegen. Codegen mandatory before Family 06 begins.

### TRIG-2: Handler Size Approaching Threshold

- **Status**: TRIGGERED — monitoring required during Family 05.
- **Evidence**: Handler file at 515 lines (5 methods). Each method adds ~80–100 lines. Projected Family 05: ~595–615 lines.
- **Thresholds**: <550 healthy, 550–600 concerning, >600 critical.
- **Family 05 impact**: Non-blocking, but Family 05 implementation must verify final line count. If >620 lines, `parseAnalyticalParams()` extraction becomes immediate.
- **Required action**: Measure handler file size post-Family 05. Report in validation findings.

### TRIG-3: Documentation Currency (PF-4/CI Smoke)

- **Status**: TRIGGERED — documentation stale, implementation resolved.
- **Evidence**: CI `smoke-analytical` job operational since S166/S172. PF-4 carried forward through 4 families despite being resolved.
- **Family 05 impact**: None.
- **Required action**: Mark PF-4 as RESOLVED in all future documents. Close tracking.

---

## Deferred Items with Committed Triggers (4)

### DEF-C1: Codegen Implementation

- **Trigger**: Family 06 boundary (6 analytical families in read path).
- **Rationale**: At 6 families, ~1,000+ LOC duplication. Template generation cheaper than manual copy-paste-modify. Covers readers, handlers, use cases, and optionally tests.
- **Estimated effort**: 2–3 days.
- **Pre-condition for**: Family 06 gate.

### DEF-C2: Schema Coherence Compile-Time Verification

- **Trigger**: ~12 analytical tables or 100+ DDL columns.
- **Rationale**: Currently review-enforced. At 12+ tables, manual verification becomes unreliable. Compile-time column count checks or DDL-to-struct alignment tooling needed.
- **Current state**: 6 tables, ~75 columns — well under threshold.
- **Estimated effort**: 1–2 days.

### DEF-C3: Handler File Split

- **Trigger**: Handler file exceeds ~600 lines (projected at Family 06).
- **Rationale**: Single file with 6+ handler methods becomes hard to navigate and review. Options: split by family, generate, or extract shared param parsing.
- **Current state**: 515 lines (Family 04), projected ~595–615 (Family 05).
- **Estimated effort**: 0.5–1 day.
- **Note**: May be absorbed into codegen tranche (DEF-C1) if implemented together.

### DEF-C4: Friction Count Gate

- **Trigger**: >2 new frictions in a single family expansion.
- **Rationale**: S167 gate condition. Ensures pattern health re-evaluated when friction accelerates.
- **Family 04 result**: 0 new frictions (well under threshold).
- **Status**: Active — evaluated every family.

---

## Deferred Items Without Committed Triggers (9)

### DEF-U1: Filter Case-Sensitivity (PF-3)

- **Severity**: Low.
- **Description**: Optional filters (`outcome`, `direction`, `disposition`) are case-sensitive and unvalidated. `disposition=APPROVED` returns 0 rows (values stored lowercase).
- **Rationale for deferral**: Consistent across all families, documented in runbook, no operational incidents.

### DEF-U2: No Pagination Beyond limit=500 (D-9)

- **Severity**: Low.
- **Description**: All analytical queries hard-capped at 500 rows. No cursor-based pagination.
- **Rationale for deferral**: Current data volumes well within limit. No dashboard or consumer requires >500 rows per query.

### DEF-U3: NATS Consumer Lag Visibility (D-6)

- **Severity**: Medium.
- **Description**: No metric or alert for consumer lag between NATS publish and ClickHouse write.
- **Rationale for deferral**: Writer pipeline operates at low volume. Lag monitoring relevant when write volume increases (e.g., tick-level data).

### DEF-U4: Sticky Degradation Without Auto-Recovery (D-7)

- **Severity**: Medium.
- **Description**: If ClickHouse goes down and comes back, analytical readers remain in degraded state until gateway restart.
- **Rationale for deferral**: ClickHouse restarts are rare. Manual gateway restart is acceptable. Auto-recovery adds connection-pool complexity.

### DEF-U5: Silent Mapper Fallbacks (D-10)

- **Severity**: Low.
- **Description**: `parseFloat()` and `marshalJSON()` silently return zero/empty on error.
- **Rationale for deferral**: Silent fallbacks are intentional — partial row data preferred over dropped events. Logging would create noise at volume.

### DEF-U6: Backoff Jitter (D-5)

- **Severity**: Low.
- **Description**: Writer reconnection uses fixed backoff without jitter.
- **Rationale for deferral**: Single writer instance. Thundering herd not possible. Jitter relevant only with multiple writer replicas.

### DEF-U7: Smoke JSON Content Verification (PF-6)

- **Severity**: Low.
- **Description**: Smoke test checks field presence but not JSON column content structure.
- **Rationale for deferral**: Unit tests provide comprehensive JSON round-trip coverage. Adding JSON schema validation to smoke creates maintenance burden without proportional value.

### DEF-U8: Consumer/Inserter Naming (H-4)

- **Severity**: Low.
- **Description**: Internal naming in writer service could be more consistent.
- **Rationale for deferral**: Cosmetic. No operational impact.

### DEF-U9: Metadata Validation (D-11)

- **Severity**: Low.
- **Description**: No validation of metadata JSON structure at write time.
- **Rationale for deferral**: Metadata is opaque by design. Validation would require schema per family, contradicting the generic metadata pattern.

---

## Resolved Items (6)

| Item | Resolution | Resolved At |
|------|-----------|-------------|
| D-1 | `parseEvidenceKeyParams` → `parseAnalyticalKeyParams` | S172 (H-3) |
| D-2 | Struct-based DI (`AnalyticalHandlerDeps` / `AnalyticalFamilyDeps`) | S172 (H-1) |
| D-3 | Smoke test extraction (`validate_analytical_family()` helper) | S172 (H-2) |
| PF-4 | CI smoke integration (`smoke-analytical` job) | S166/S172 |
| D-4 | Codegen evaluation (justified, deferred to Family 06) | S178 |
| CT-ceiling | Family 04 ceiling test (4 JSON, free-text, 17 DDL, struct parser) | S182 |

---

## Summary Table

| Category | Count | Blocks Family 05? |
|----------|-------|-------------------|
| Triggered (action required) | 3 | No (none blocking) |
| Deferred with committed trigger | 4 | No |
| Deferred without trigger | 9 | No |
| Resolved | 6 | — |
| **Total tracked** | **22** | **None blocking** |

## Debt Trajectory

| Checkpoint | Active Items | High Severity | Blocking |
|------------|-------------|---------------|----------|
| Pre-hardening (S166) | 14 | 1 (CI) | 0 |
| Post-hardening (S172) | 11 | 0 | 0 |
| Pre-Family 04 (S178) | 15 | 0 | 0 |
| **Pre-Family 05 (S183)** | **16** | **0** | **0** |

Increase from 15 to 16 reflects TRIG-2 (handler size monitoring) added as explicit tracked item. No severity escalation. No new high-severity items. Pattern debt is stable and well-governed.

---

## Implications for Family 05

1. **No item blocks Family 05 implementation.**
2. **Handler size is the only metric requiring active monitoring** during implementation.
3. **Codegen tranche is the dominant post-Family-05 obligation** — must be resolved before Family 06.
4. **Family 05 inherits a clean friction slate** — Family 04 produced zero new frictions.
5. **Nine deferred items remain stable** — none escalated in severity, none approaching their trigger conditions.
