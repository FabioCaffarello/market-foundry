# Wave B After Family 03 — Gains, Trade-offs, and Open Debts

> Accounting ledger for the Wave B expansion after four analytical families (Candles, Signals, Decisions, Strategies).

---

## 1. Gains

### G-1: Four-layer analytical coverage

- **Before**: 3 of 6 analytical layers covered (candles, signals, decisions).
- **After**: 4 of 6 layers covered — strategies added.
- **Payoff**: Read path now spans from raw evidence through computed signals, decision outcomes, and strategic interpretations. Only risk assessments and executions remain.

### G-2: Three JSON columns proven

- **Before**: 2 JSON columns was the ceiling (decisions: signal_inputs, metadata).
- **After**: 3 JSON columns handled cleanly (strategies: decisions, parameters, metadata).
- **Payoff**: JSON parsing composes through reuse — `ParseMetadataJSON` handles 2 of 3 columns; only `ParseDecisionInputsJSON` was new. Validates that Family 04's 4 JSON columns are structurally feasible.

### G-3: Direction filter as mechanical addition

- **Before**: Only one optional enum filter proven (decisions: outcome).
- **After**: Direction filter (long/short/flat) follows identical pattern — optional WHERE clause, no validation, invalid values return empty results.
- **Payoff**: Domain-specific optional filters are now a proven mechanical step, not an architectural decision.

### G-4: Struct DI validated under real expansion pressure

- **Before**: H-1 hardening (struct-based DI) implemented but untested with a new family.
- **After**: Adding `GetStrategyHistory` to `AnalyticalHandlerDeps` required zero changes to existing wiring — purely additive.
- **Payoff**: Constructor churn eliminated as a friction source. Struct DI is confirmed as load-bearing infrastructure.

### G-5: Write path immutability at four families

- **Before**: Write path unchanged through 3 family expansions.
- **After**: Write path unchanged through 4 family expansions — all 6 mappers pre-staged, zero modifications.
- **Payoff**: Write path is a fixed cost, not a variable one. No expansion touches it.

### G-6: Pattern applied three times with zero regressions

- **Before**: Pattern v2 applied twice (F-01, F-02).
- **After**: Pattern v2 applied three times — no correctness regressions, no behavioral changes to prior families.
- **Payoff**: The pattern is empirically proven as regression-safe. Each family is isolated by design.

### G-7: D-4 codegen trigger evaluated and resolved

- **Before**: Codegen evaluation was a committed trigger at Family 03/04 boundary.
- **After**: Evaluated in S178 — justified but not cost-effective until Family 06. Explicitly non-blocking.
- **Payoff**: Removes uncertainty about codegen timing. The committed trigger is resolved with a clear threshold.

---

## 2. Trade-offs

### T-1: Mechanical duplication tolerated over premature abstraction

- **Given up**: ~800 lines of near-identical code across 4 readers, 4 handlers, 4 use cases (~80% structural similarity).
- **Gained**: Each artifact is concrete, self-contained, and independently testable. No shared abstractions that could couple families.
- **Still acceptable?**: Yes, through Family 05. At Family 06 (~1200 lines), codegen becomes mandatory.

### T-2: Review-enforced schema coherence over compile-time checks

- **Given up**: Automated verification that DDL, mapper, reader, and response stay aligned.
- **Gained**: Simplicity — no build tooling, no code generation, no schema DSL.
- **Still acceptable?**: Yes, through ~8 families. Compile-time checks recommended at ~12 tables.

### T-3: No filter validation over strict input checking

- **Given up**: Server-side validation of enum values (direction, outcome, type).
- **Gained**: Consistent behavior — invalid values return empty results, no special error handling.
- **Still acceptable?**: Yes. No incidents from this pattern. Client-side validation is sufficient.

### T-4: No pagination over cursor-based query scaling

- **Given up**: Ability to page through results beyond 500 rows.
- **Gained**: Simpler query contract — limit is bounded, no cursor state, no offset tracking.
- **Still acceptable?**: Yes. Current volumes fit within 500. Revisit when data density makes 500 insufficient.

### T-5: Handler file as monolith over split-by-family

- **Given up**: Clean separation of handler methods into per-family files.
- **Gained**: Single entry point for all analytical HTTP surface — easier to review and navigate.
- **Still acceptable?**: Yes, at ~417 lines. Split recommended at ~600 lines (projected Family 06).

### T-6: No cross-family queries over analytical joins

- **Given up**: Ability to query across analytical layers (e.g., "show me strategies that led to executions").
- **Gained**: Complete family isolation — no coupling, no join complexity, no schema evolution pressure.
- **Still acceptable?**: Yes. Cross-family queries are a fundamentally different capability. Not a trade-off — a deliberate non-goal.

---

## 3. Open Debts

### Debts with committed resolution points

| ID | Description | Severity | Trigger | Projected |
|----|-------------|----------|---------|-----------|
| DEF-C1 | Codegen implementation | Medium | Mandatory before Family 06 | After Family 05 |
| DEF-C2 | Schema coherence compile-time check | Medium | ~12 analytical tables | After Family 08+ |
| DEF-C3 | Handler file split | Low-Medium | File exceeds ~600 lines | Family 06 |
| DEF-C4 | Friction count gate | Structural | >2 new frictions in single family | Each family |

### Debts without committed resolution points

| ID | Description | Severity | Notes |
|----|-------------|----------|-------|
| DEF-U1 | Filter case-sensitivity | Low | Consistent behavior, no incidents |
| DEF-U2 | No pagination | Low | 500-row limit sufficient |
| DEF-U3 | NATS consumer lag visibility | Medium | Operational gap — not blocking expansion |
| DEF-U4 | Sticky degradation without auto-recovery | Medium | Manual restart required — acceptable for current scale |
| DEF-U5 | Silent mapper fallbacks | Low | Logged, not alerted |
| DEF-U6 | Backoff jitter | Low | Cosmetic improvement |
| DEF-U7 | Smoke JSON content verification | Low | Unit tests cover JSON correctness |
| DEF-U8 | Consumer/inserter naming | Low | Naming inconsistency, no functional impact |
| DEF-U9 | Metadata validation | Low | No schema enforcement on JSON keys/values |

### Resolved debts

| ID | Description | Resolved in |
|----|-------------|-------------|
| D-1 | parseEvidenceKeyParams naming | S172 (H-3) |
| D-2 | Constructor positional arguments | S172 (H-1) |
| D-3 | Smoke test duplication | S172 (H-2) |
| PF-4 | CI smoke integration | S166/S172 |
| D-4 | Codegen evaluation | S178 |

---

## 4. Debt Trajectory

| Milestone | Active debts | Resolved | Net change |
|-----------|-------------|----------|------------|
| Post-F-01 (S167) | 12 | 0 | — |
| Post-F-02 + hardening (S173) | 11 | 5 | -1 |
| Post-F-03 (S179) | 13 | 5 | +2 |
| Projected post-F-04 | 14–15 | 5 | +1–2 |
| Projected post-F-05 + codegen | 10–11 | 8–9 | -4 |

The debt count increased by 2 (PF-3, PF-6) — both low severity. No accumulation pressure. The committed trigger at Family 06 (codegen) is projected to reduce active debts by ~4, returning the count below the post-F-01 baseline.

---

## 5. Items That Do Not Justify Their Cost Now

| Item | Estimated cost | Projected benefit | Verdict |
|------|---------------|-------------------|---------|
| Codegen implementation | 2–3 days | Reduces ~800 lines of duplication | **Not yet** — cost-effective at 6+ families, premature at 4 |
| Handler file split | 0.5–1 day | Cleaner navigation | **Not yet** — 417 lines is comfortable; split at ~600 |
| Schema coherence compile-time | 1–2 days | Catches misalignment at build time | **Not yet** — 4 tables is manageable; needed at ~12 |
| Generic reader abstraction | 3–5 days | Single parameterized reader | **Not yet** — premature; each reader is ~138 lines with unique parsing |
| Pagination | 1–2 days | Query results beyond 500 | **Not yet** — no demand; current volumes fit within limit |
| Cross-family queries | 5+ days | Analytical joins across layers | **Not now** — fundamentally different capability; out of Wave B scope |
| Filter validation | 0.5 day | 400 errors for invalid enum values | **Not now** — consistent empty-result behavior is acceptable |
| Auto-recovery | 2–3 days | Automatic reconnection on degradation | **Not now** — manual restart is acceptable at current scale |
