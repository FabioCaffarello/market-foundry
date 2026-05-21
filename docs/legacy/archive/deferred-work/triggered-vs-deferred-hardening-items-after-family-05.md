# Triggered vs Deferred Hardening Items After Family 05

## Purpose

Classify all tracked items into triggered (requiring action before Family 06), deferred with committed triggers, and deferred without triggers — based on the ceiling evidence produced by Family 05 (S187/S188).

---

## Triggered Items — Mandatory Before Family 06 (3)

### TRIG-1: Handler File at Hard Ceiling — Extract `parseAnalyticalParams()`

- **Status**: CRITICAL BLOCKER
- **Evidence**: Handler file at exactly 615 lines (620-line hard ceiling). Each family adds ~80–100 lines. Family 06 would push to ~715 lines — 95 lines past ceiling.
- **Root cause**: Limit parsing (13 lines), since/until parsing (17 lines), and nil-check boilerplate are copy-pasted verbatim in all 6 handler methods.
- **Required action**: Extract `parseAnalyticalParams()` helper to consolidate limit + since/until parsing. This reduces per-method body by ~30 lines, bringing handler from 615 to ~435 lines — buying runway for ~3 more families.
- **Why not handler split**: Split adds file navigation cost and architectural complexity. The ceiling is caused by parameter duplication, not method count. Extracting the shared parsing addresses the root cause directly.
- **Why not codegen**: Codegen would also solve this, but codegen has broader scope (readers, use cases, tests). Handler extraction is a 30-minute targeted fix; codegen is a multi-day initiative. Extraction is the minimum viable unblock.
- **Blast radius**: `analytical.go` only. Zero behavioral change. All tests must produce identical results.

### TRIG-2: Codegen Tranche Scope Definition — Mandatory Decision

- **Status**: MANDATORY — scope-defining, not implementation
- **Evidence**: 5 families delivered with zero creative decisions, 85% handler duplication, 80% reader duplication, 70% use case duplication. Manual cost: ~45 min/family. Codegen cost (once templates exist): ~2 min/family.
- **Required action for S189**: Define whether codegen enters scope for the pre-Family-06 gate, or whether it is deferred to a dedicated codegen stage after the handler extraction unblocks Family 06.
- **Assessment**: Codegen implementation is a multi-day effort. Handler extraction alone unblocks Family 06 mechanically. Codegen can follow as a strategic improvement without blocking expansion.
- **Decision**: **Codegen is deferred to a dedicated stage (post-S189).** Handler extraction is sufficient to unblock Family 06. Codegen remains the highest-priority strategic initiative but is not a blocker.

### TRIG-3: Reader 10-Parameter Positional Signature — Monitoring Escalation

- **Status**: TRIGGERED — at practical limit, non-blocking for one more family
- **Evidence**: Execution reader constructor takes 10 positional parameters. Family 06 would add an 11th, approaching Go function signature readability limits.
- **Required action**: Flag for codegen tranche. If Family 06 adds a new reader, the 11-param signature remains acceptable but is the hard limit for manual expansion.
- **Mitigation**: Codegen will eliminate positional params entirely (generated code). For one more manual family, 11 params is tolerable.
- **Decision**: **Monitored, not blocked.** The handler extraction is the critical path. Reader signature reaches its limit at Family 07 without codegen.

---

## Deferred Items with Committed Triggers (4)

### DEF-C1: Codegen Implementation

- **Trigger**: Now committed at Family 07 boundary (or voluntarily at Family 06 if pursued).
- **Rationale**: Handler extraction buys ~3 families of runway. Codegen becomes mandatory when that runway expires, or earlier if the team chooses to invest in it.
- **Updated from pre-F05**: Was committed at Family 06 boundary. Handler extraction shifts the hard dependency to Family 07+.
- **Estimated effort**: 2–3 days.

### DEF-C2: Schema Coherence Compile-Time Verification

- **Trigger**: ~12 analytical tables or 100+ DDL columns.
- **Current state**: 6 tables, ~95 DDL columns — approaching but under threshold.
- **Status**: Unchanged from pre-F05 assessment.

### DEF-C3: Handler File Split (by Domain)

- **Trigger**: Handler exceeds ~700 lines (post-extraction ceiling).
- **Rationale**: Extraction reduces handler to ~435 lines. At ~100 lines per family, split needed at ~7 more families (~Family 13). Effectively eliminated as a near-term concern.
- **Status**: Superseded by TRIG-1 extraction. Split becomes relevant only if codegen is never adopted.

### DEF-C4: Friction Count Gate

- **Trigger**: >2 new frictions in a single family expansion.
- **Family 05 result**: 3 frictions (handler ceiling, dual filters, 10-param reader). Two are escalations of existing items; one is new.
- **Assessment**: Technically at threshold but the frictions are all positional/structural — they point to the same root cause (mechanical duplication at ceiling). Not indicative of pattern breakdown.
- **Status**: Active — evaluate at Family 06.

---

## Deferred Items Without Committed Triggers (9) — Unchanged

| # | Item | Severity | Status |
|---|------|----------|--------|
| DEF-U1 | Filter case-sensitivity (PF-3) | Low | Unchanged |
| DEF-U2 | No pagination beyond 500 (D-9) | Low | Unchanged |
| DEF-U3 | NATS consumer lag visibility (D-6) | Medium | Unchanged |
| DEF-U4 | Sticky degradation without auto-recovery (D-7) | Medium | Unchanged |
| DEF-U5 | Silent mapper fallbacks (D-10) | Low | Unchanged |
| DEF-U6 | Backoff jitter (D-5) | Low | Unchanged |
| DEF-U7 | Smoke JSON content verification (PF-6) | Low | Unchanged |
| DEF-U8 | Consumer/inserter naming (H-4) | Low | Unchanged |
| DEF-U9 | Metadata validation (D-11) | Low | Unchanged |

None of these items escalated in severity during Family 05. None approach their trigger conditions.

---

## Resolved Items (7)

| Item | Resolution | Resolved At |
|------|-----------|-------------|
| D-1 | `parseEvidenceKeyParams` → `parseAnalyticalKeyParams` | S172 (H-3) |
| D-2 | Struct-based DI (`AnalyticalHandlerDeps`) | S172 (H-1) |
| D-3 | Smoke test extraction (`validate_analytical_family()`) | S172 (H-2) |
| PF-4 | CI smoke integration (`smoke-analytical` job) | S166/S172 |
| D-4 | Codegen evaluation (justified, deferred) | S178 |
| CT-ceiling | Family 04 ceiling test | S182 |
| F05-coverage | Full vertical coverage L1–L6 | S187 |

---

## New Items Added by Family 05 (2)

| # | Item | Origin | Severity | Disposition |
|---|------|--------|----------|-------------|
| PF-7 | Reader 10-param positional signature | S188 | Low | TRIG-3 (monitored) |
| PF-1-ESC | Handler at hard ceiling (615/620) | S188 escalation | **Critical** | TRIG-1 (mandatory) |

---

## Summary Table

| Category | Count | Blocks Family 06? |
|----------|-------|-------------------|
| Triggered (action required) | 3 | **Yes — TRIG-1 is blocking** |
| Deferred with committed trigger | 4 | No |
| Deferred without trigger | 9 | No |
| Resolved | 7 | — |
| **Total tracked** | **23** | **1 blocking** |

## Debt Trajectory

| Checkpoint | Active Items | High Severity | Blocking |
|------------|-------------|---------------|----------|
| Pre-hardening (S166) | 14 | 1 | 0 |
| Post-hardening (S172) | 11 | 0 | 0 |
| Pre-Family 04 (S178) | 15 | 0 | 0 |
| Pre-Family 05 (S183) | 16 | 0 | 0 |
| **Post-Family 05 (S189)** | **16** | **1 (handler ceiling)** | **1** |

The first true blocker in the analytical layer's lifecycle. Handler extraction resolves it with minimal scope.
