# Wave B Iteration 01 — Formal Gate Review

## Status: CONDITIONAL PASS

The first Wave B iteration (Signal/RSI family) is assessed as **successful within its stated scope**, but the expansion pattern remains fundamentally artisanal. The gate authorizes a second family iteration under strict conditions, not a blanket continuation of Wave B.

---

## 1. Gate Context

| Item | Detail |
|------|--------|
| Iteration scope | Signal (RSI) read-path family expansion |
| Pattern used | Wave B 9-artifact expansion (v1, hardened to v2 post-iteration) |
| Artifacts delivered | 4 new files, 8 modified files, 4 documentation files |
| Tests added | 29 unit tests (adapter, use case, handler, gateway) |
| CI status | GitHub Actions with unit-tests + smoke-analytical jobs (S166) |
| Write path changes | Zero (writer already consumed RSI signals from Wave A) |
| Constraint violations | None detected |

---

## 2. Gate Questions — Explicit Answers

### 2.1 Did the first family enter with correct boundaries and responsibilities?

**Yes, with one caveat.**

Boundaries are clean. The signal read path follows the same layered architecture as candles: adapter (query builder + row mapper) → use case (validation + delegation + timing) → handler (HTTP parsing + Server-Timing) → route (conditional registration). No cross-layer dependencies. Writer and reader remain fully decoupled via ClickHouse.

**Caveat:** The naming residue (`parseEvidenceKeyParams`) reveals that shared helpers were designed for the first family (candles/evidence) and reused without generalization. This is not a boundary violation — it is a naming debt that signals the pattern was extended, not redesigned. Acceptable for 2 families; must be addressed at family 3.

### 2.2 Is the Wave B pattern repeatable or still artisanal?

**It is repeatable but artisanal.**

The 9-artifact checklist provides structure. The left-to-right dependency chain enforces ordering. The gate review criteria provide a stop condition. These are genuine process improvements.

However, the expansion still requires a human to:
- Copy ~80% of code from the previous family and modify it
- Manually verify column alignment across DDL, mapper, and reader (3 locations)
- Extend the smoke test with a new section following an informal template
- Wire new dependencies into the gateway composition root by hand
- Remember to add handler constructor arguments (growing linearly)

None of these steps are automated, validated by tooling, or protected by compile-time checks. The pattern is a well-documented manual procedure, not a mechanized process. This is acceptable for 2–3 families. It becomes a liability at 4+.

**Evidence:**
- Schema coherence is verified by unit test assertions on row length and exported query builders — this is the strongest mechanization achieved
- Observability parity is automatic via the inserter/supervisor infrastructure — genuine structural win
- Everything else is copy-paste-modify with code review as the safety net

### 2.3 Do schema/writer/reader/gateway remain cohesive in expansion?

**Yes.**

| Layer | Candle | Signal | Coherent? |
|-------|--------|--------|-----------|
| DDL | 001_create_evidence_candles.sql | 002_create_signals.sql | Yes — same conventions |
| Writer mapper | mapCandleRow() | mapSignalRow() | Yes — same parseFloat/marshalJSON patterns |
| Reader adapter | QueryCandleHistory() | QuerySignalHistory() | Yes — same parameterized query + timing pattern |
| Use case | GetCandleHistoryUseCase | GetSignalHistoryUseCase | Yes — same validation + delegation + timing |
| Handler | GetCandleHistory | GetSignalHistory | Yes — same param parsing + Server-Timing |
| Route | /analytical/evidence/candles | /analytical/signal/history | Yes — conditional registration |
| Smoke | Phase 5 | Phase 5b | Yes — same validation structure |

Cohesion is maintained. The risk is not divergence — it is mechanical duplication. At 6 families, every layer will have 6 near-identical implementations. This is a scaling concern, not a correctness concern.

### 2.4 Is smoke-analytical in CI sufficient to sustain the second iteration?

**Sufficient, not ideal.**

The CI workflow (S166) runs unit tests then smoke-analytical. It catches compilation errors, test failures, and integration regressions. Log artifacts are collected on failure. This is a meaningful gate.

**Gaps:**
- Smoke test is monolithic — a single script with linearly growing phases. Failure in one family's section produces output that requires manual parsing.
- No per-family isolation in smoke — a regression in candle validation blocks signal validation.
- No load testing, concurrency testing, or latency assertions.
- Smoke wait time (120s for writer flush) is a fixed sleep, not a readiness probe — fragile under load or slow CI runners.

**Assessment:** CI is sufficient to catch regressions for 2–3 families. The extraction of `validate_analytical_family()` at family 3 is a hard requirement, not an optimization.

### 2.5 What frictions remain open?

| Friction | Severity | Trigger | Status |
|----------|----------|---------|--------|
| `parseEvidenceKeyParams` naming residue | Low | Family 3 | Deferred — documented |
| Constructor argument accumulation | Medium | Family 3 | Deferred — struct DI threshold committed |
| ~80% mechanical code duplication | Medium | Family 4 | Deferred — codegen evaluation committed |
| Smoke test linear growth | Medium | Family 3 | Deferred — extraction threshold committed |
| No signal-type validation at reader | Low | Never | Accepted — empty result is correct behavior |
| No backoff jitter in writer retry | Low | Not scheduled | Known debt, not blocking |
| No consumer lag visibility | Medium | Not scheduled | Invisible buffer pressure risk |
| Sticky degradation (no auto-recovery) | Medium | Not scheduled | Manual restart required after ClickHouse outages |
| Schema coherence is review-enforced | Medium | ~12 tables | Acceptable now, revisit at scale |
| No pagination beyond 500 rows | Low | Not scheduled | Hard limit acceptable for current use |

### 2.6 What is the acceptable next step?

**Option 2: Second family of Wave B — with conditions.**

The first iteration produced a functioning family expansion with clean boundaries, passing tests, CI integration, and no constraint violations. The pattern, while artisanal, is documented and proven for one additional iteration.

The second family is authorized under these conditions:

1. **Mandatory:** Family 2 follows the v2 pattern exactly (9 artifacts, CI gate, 5-point gate review, 4-section documentation).
2. **Mandatory:** Family 3 triggers all committed hardening thresholds (struct DI, smoke extraction, naming cleanup). This is not optional — it is a pre-committed obligation.
3. **Mandatory:** If family 2 reveals any new friction not captured in v2, the iteration pauses for assessment before family 3 begins.
4. **Recommended:** Select Decisions (RSI Oversold) as family 2 — it introduces JSON field complexity (signals array, metadata map) that will stress-test the pattern more than a simpler family would.

**Why not Option 1 (hardening first)?**
The pattern has no blocking defects. The committed thresholds at family 3 address the known ergonomic debts. Hardening now would delay expansion without new information — the next data point comes from executing family 2.

**Why not Option 3 (pause)?**
There is no evidence that the pattern is broken. The frictions are documented, tracked, and have committed resolution points. A pause without cause would add bureaucratic overhead without reducing risk.

---

## 3. Gate Verdict

| Criterion | Result | Evidence |
|-----------|--------|----------|
| All S162 constraints respected? | PASS | C-1 through C-9 verified; no violations |
| Pattern discipline followed? | PASS | 9-artifact unit delivered; left-to-right chain maintained |
| Schema coherence verified? | PASS | 12/12 domain columns type-aligned across DDL/writer/reader |
| Observability parity achieved? | PASS | Wall-clock timing, QueryMeta, Server-Timing, structured logging |
| CI integration active? | PASS | GitHub Actions with unit-tests + smoke-analytical |
| Optionality preserved? | PASS | Gateway starts without ClickHouse; operational routes unaffected |
| No regressions introduced? | PASS | Existing candle tests and smoke phases unchanged |
| Friction documented honestly? | PASS | 6 frictions cataloged with severity, trigger, and disposition |

**Verdict: The first Wave B iteration passes the gate. The second family iteration is authorized under the conditions specified in Section 2.6.**

---

## 4. Binding Commitments for Continuation

These are not suggestions — they are conditions that constrain the next iterations:

1. Family 2 must pass the same gate review before family 3 begins.
2. Family 3 must execute all three hardening thresholds (struct DI, smoke extraction, naming) as its primary deliverable, alongside the family expansion.
3. If any family iteration fails the gate review, expansion halts until the failure is resolved.
4. No family iteration may modify existing family artifacts (additive-only, C-9).
5. Pattern v2 is the governing document — deviations require explicit justification and gate review.
