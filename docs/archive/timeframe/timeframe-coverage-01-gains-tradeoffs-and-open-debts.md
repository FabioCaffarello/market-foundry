# TC-01: Gains, Trade-offs, and Open Debts

> Honest accounting of what TC-01 bought, what it cost, and what it left unresolved.
> Date: 2026-03-19 | Stage: S136

---

## 1. Gains

### G1 — Architectural Proof of Config-Driven Scaling

**What**: 2→4 timeframes with zero Go code changes.
**Value**: This is the single most important outcome. It proves the S10–S15 design actually works as intended — timeframe is a genuine first-class config dimension, not an afterthought requiring code changes per TF.
**Durability**: Permanent. This proof holds for any future TF within [10s, 86400s].

### G2 — Linear Resource Growth Confirmed

**What**: 2× actors, 2× subjects, 2× KV keys, <30% write load increase.
**Value**: Eliminates the fear of combinatorial explosion. Growth is predictable and bounded.
**Durability**: Holds as long as TFs don't require qualitatively different processing (which they don't in current architecture).

### G3 — Six Anticipated Problems Did Not Materialize

**What**: NATS pressure, fan-out latency, KV contention, dedup collision, cross-TF interference, memory accumulation — none occurred.
**Value**: Reduces the risk surface for future expansions. These were legitimate concerns that can now be dismissed with evidence.
**Durability**: Holds at current and moderately higher scales.

### G4 — Config Validation Gate

**What**: `ValidateTimeframes()` rejects invalid configs at startup.
**Value**: Prevents a class of misconfiguration bugs permanently.
**Durability**: Permanent.

### G5 — Operational Runbook for Crash Recovery

**What**: Documented data loss expectations per timeframe.
**Value**: Operators can make informed decisions during incidents.
**Durability**: Must be updated if new TFs are added.

### G6 — Full Test Coverage Across 4 Timeframes

**What**: Unit tests, smoke scripts, and HTTP test files exercise all 4 TFs across all 6 domains.
**Value**: Regression safety for any future change.
**Durability**: Maintained as long as tests run in CI.

---

## 2. Trade-offs Accepted

### T1 — Global Timeframe List (All Symbols Share Same TFs)

**What we gave up**: Per-symbol timeframe customization.
**What we gained**: Simpler config, simpler actor spawning, no conditional logic.
**When this hurts**: If a symbol needs 4h candles but another doesn't, you must add 3600s for all.
**Reversal cost**: Medium. Requires config schema change + SourceScopeActor modification.

### T2 — No Interim Candle Snapshots

**What we gave up**: Ability to query partial candle state before window close.
**What we gained**: Simpler evidence publisher, no "in-progress" state concept.
**When this hurts**: At 3600s, there's no signal for 59 minutes if RSI uses only closed candles.
**Reversal cost**: Medium. Requires new event type and publisher logic.

### T3 — In-Memory-Only Window State

**What we gave up**: Crash resilience for long windows.
**What we gained**: Simpler accumulator, no persistence overhead, no WAL complexity.
**When this hurts**: At 3600s, crash loses up to 60 minutes of trade accumulation.
**Reversal cost**: High. Requires WAL or periodic snapshot mechanism in evidence sampler.

### T4 — Aggregate-Only Tracking (No Per-TF Counters)

**What we gave up**: Per-timeframe diagnostic visibility.
**What we gained**: Simpler tracker, fewer metrics to manage.
**When this hurts**: At 8+ TFs or when diagnosing a TF-specific issue.
**Reversal cost**: Low. Add per-TF counter keys to existing tracker.

### T5 — Two Success Criteria Deferred (M7, M8)

**What we gave up**: Full signal convergence proof at 900s and 3600s.
**What we gained**: Completed TC-01 without multi-hour validation runs.
**When this hurts**: If RSI at 3600s has a subtle bug that only manifests after warm-up.
**Reversal cost**: Low (time cost only). Run extended validation when operationally feasible.

---

## 3. Open Debts

### D1 — Window State Persistence (F-13 + F-15) — HARD GATE

**Status**: Unresolved. Explicitly documented as blocking condition for TC-02.
**Debt**: Evidence sampler holds candle state in memory only. Crash at minute 59 of a 3600s window loses all accumulated OHLCV data.
**Impact at current scale**: Acceptable. 60-minute max loss. Recovery = wait for next window.
**Impact at TC-02 scale (4h+)**: Unacceptable. 4-hour data loss with no interim output.
**Resolution path**: WAL-based persistence or periodic snapshot in evidence sampler.
**Estimated effort**: Medium-high. Touches core accumulator loop.

### D2 — Per-TF Idle Detection (F-05)

**Status**: Not needed at 4 TFs. Becomes relevant at 4h+ where stalls are harder to distinguish from normal long-window silence.
**Debt**: No mechanism to detect that a specific timeframe's sampler has stopped receiving trades.
**Impact**: At 3600s, silence for 30 minutes could be normal (low-volume period) or a stall. No way to differentiate.
**Resolution path**: Per-TF heartbeat or last-seen timestamp in tracker.
**Estimated effort**: Low.

### D3 — RSI Convergence at Long Windows (M7, M8)

**Status**: Deferred. Not a code debt — a validation debt.
**Debt**: RSI at 900s and 3600s has not been observed through full warm-up cycle.
**Impact**: Low. The RSI evaluator is timeframe-agnostic; only the warm-up duration differs.
**Resolution path**: Extended validation run (6h for 900s, 15h for 3600s).
**Estimated effort**: Time only, no code changes.

### D4 — Per-Binding Timeframe Config (F-01)

**Status**: Not needed. Global list works for uniform config.
**Debt**: Cannot assign different TF sets to different symbols.
**Impact**: Zero at current single-symbol/uniform-TF setup.
**Resolution path**: Config schema change + conditional spawning in SourceScopeActor.
**Estimated effort**: Medium.

### D5 — Query Surface Observability (F-07, F-08)

**Status**: P3. No external consumers.
**Debt**: No "list configured timeframes" endpoint. HTTP 200 with null body is ambiguous (not configured vs warming vs expired).
**Impact**: Only affects non-expert operators or external integrations (neither exists yet).
**Resolution path**: New discovery endpoint + structured error responses.
**Estimated effort**: Low.

---

## 4. Debt Priority Map

| Debt | Priority | Trigger | Blocks |
|------|----------|---------|--------|
| D1: State persistence | P2 | TC-02 commits to 4h+ | TC-02 execution |
| D2: Per-TF idle detection | P2 | 4h+ TFs deployed | Extended TF operations |
| D3: RSI convergence proof | P3 | Operational feasibility | Nothing (validation only) |
| D4: Per-binding TFs | P3 | Heterogeneous symbol needs | Per-symbol customization |
| D5: Query observability | P3 | External consumers | Nothing critical |

**Total open debt: 5 items. 1 hard gate (D1). 1 soft gate (D2). 3 optional.**
