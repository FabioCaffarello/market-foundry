# Post-Family-05 / Pre-Family-06 Gate — S190

## Purpose

This document is the formal gate review after Family 05 (Executions) and before any Family 06 authorization. Its purpose is to decide — based on accumulated evidence, not on momentum — whether Wave B continues, hardens further, or pauses.

## Gate Context

| Dimension | Value |
|-----------|-------|
| Families delivered | 6 (baseline + 5 expansions) |
| Vertical layers covered | L1 (Evidence) → L6 (Executions) — complete |
| Total analytical LOC | ~3,950 (impl + tests) |
| Creative decisions across 5 expansions | 0 |
| Write-path modifications across 6 expansions | 0 |
| Unit tests passing | 289 (47 execution-specific) |
| Mandatory hardening (H-5) | Complete, verified |
| Handler file after H-5 | 501 lines (ceiling was 615) |

## Formal Evaluation Criteria

### 1. Is the manual pattern still healthy?

**Yes, with qualifications.**

The 9-artifact expansion template has been applied 5 times with zero creative decisions and zero structural regressions. Family 05 was the most complex family yet (20 DDL columns, 4 JSON columns, 2 new type classes, dual filters) and it was delivered mechanically within the established pattern.

However, three structural metrics reached their practical limits during Family 05:

| Metric | Family 05 Value | Limit | Status |
|--------|----------------|-------|--------|
| Handler file | 615 lines | 620 hard ceiling | **At ceiling** (resolved by H-5 → 501) |
| Reader positional args | 10 | ~10 practical limit | **At limit** |
| Parser function count | 8 | 8 threshold | **At threshold** |

H-5 extraction resolved the handler ceiling and buys 2–3 more families of runway. The other two metrics are tolerable for one more family but will require intervention at Family 07.

### 2. Which hardenings are no longer optional?

| Item | Status | Trigger |
|------|--------|---------|
| Handler `parseAnalyticalParams()` extraction (H-5) | **DONE** — 615→501 lines | Was blocking Family 06 |
| Codegen tranche definition | **MANDATORY before Family 07** | 5 families with 0 creative decisions = template-ready |
| Reader query-object pattern | **MANDATORY at Family 07** | 10-param positional signature at practical limit |
| Generic JSON parser `parseJSON[T]` | **RECOMMENDED at parser count 9** | 8 parsers, each ~10 lines, identical shape |

### 3. Do schema/writer/reader/gateway remain cohesive under 5 families?

**Yes.** Each layer scales independently along its axis:

- **Schema**: 6 tables, ~95 DDL columns total. ClickHouse tolerates this trivially. Migration catalog is ordered and versioned.
- **Writer**: Zero modifications across 6 expansions. Pipeline, inserter, supervisor, mappers all scale by NATS subject routing. Immutability confirmed.
- **Reader**: 6 reader methods, each self-contained. Positional parameter count is the only pressure (10 args at F-05). Struct-based DI absorbs new families without constructor changes.
- **Gateway**: Analytical handler is the pressure point (resolved by H-5). Routes are additive. Response shapes are consistent.
- **Observability**: Server-Timing headers, structured logging, and health checks are automatic per family.

### 4. Has codegen become a real necessity?

**Not yet a blocker. Clearly a necessity by Family 07.**

Evidence:
- 0 creative decisions across 5 expansions proves the pattern is 100% templatable.
- ~85% handler duplication, ~80% reader duplication, ~70% use case duplication.
- Manual cost: ~45 min / ~780 LOC per family.
- Codegen cost (once built): ~2 min per family.
- Break-even: approximately Family 06–07.

Codegen is not blocking Family 06. But deferring past Family 07 would be accepting an engineering debt that has already proven itself unnecessary.

### 5. What is the acceptable next step?

**Family 06 is acceptable, with conditions.**

The gate finds that:
1. The pattern is structurally sound after H-5 extraction.
2. The handler has 2–3 families of runway (501 → ceiling at ~700–800 lines).
3. The reader signature is at its practical limit (10 args) but tolerable for one more family.
4. Full vertical coverage is already achieved; Family 06 would expand within-layer coverage (event type variants, not new layers).

**Conditions for Family 06 authorization:**
- Family 06 candidate must NOT require write-path changes.
- Family 06 must not introduce reader parameters beyond 11.
- Family 06 must include ceiling evidence measurement (same as Family 05).
- Codegen tranche must be formally scoped before Family 07 trigger assessment.

## Gate Decision

```
┌─────────────────────────────────────────────────────────┐
│  GATE DECISION: CONDITIONAL PROCEED TO FAMILY 06        │
│                                                         │
│  Family 06 is authorized under the conditions above.    │
│  Family 07 is NOT pre-authorized.                       │
│  Codegen tranche scoping is mandatory before Family 07. │
│  The pattern is healthy but approaching its manual       │
│  ceiling — this is the last family where the current     │
│  artisanal model is clearly sustainable.                 │
└─────────────────────────────────────────────────────────┘
```

## What Must Remain Small

- Family 06 scope: same 9-artifact template, no exceptions.
- No new infrastructure (no new binaries, no new compose services).
- No write-path changes (7th consecutive expansion must preserve this).
- No handler restructuring beyond what H-5 already provided.

## What Must Be Hardened Before Family 07

1. Codegen tranche: scope, templates, generation approach.
2. Reader query-object pattern: replace positional args with typed struct.
3. Generic JSON parser: `parseJSON[T]` for parser count ≥ 9.

## What Remains Deferred

- CI smoke integration (flagged 5 times, tolerable).
- Filter case-sensitivity and validation (consistent behavior, low impact).
- Pagination beyond limit=500 (no production need yet).
- NATS consumer lag visibility (writer operational concern, not reader).
- Sticky degradation without auto-recovery (writer supervision model).
- Backoff jitter in writer retry (writer internals).
- Silent mapper fallbacks (writer internals).

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Family 06 exceeds handler runway | Low | Medium | H-5 gives 2–3 family buffer |
| Reader 11-param signature becomes unwieldy | Medium | Low | Query-object pattern is documented and scoped |
| Codegen investment is deferred indefinitely | Medium | High | Gate explicitly blocks Family 07 without codegen scope |
| Momentum causes pattern expansion without review | Low | High | Each family requires gate; Family 07 requires codegen evidence |

## Conclusion

Wave B has delivered exceptional results: 6 families, full vertical coverage, zero creative decisions, zero write-path changes, zero structural regressions. The pattern is proven, reproducible, and approaching the point where automation is the natural next investment.

Family 06 proceeds under conditions. Family 07 does not proceed without codegen scoping. The gate fulfills its purpose: evidence-based authorization, not momentum-based continuation.
