# Family 06 — Abort Rationale

> Formal decision: manual analytical expansion is aborted at Family 06.
> No candidate satisfies the S190 gate conditions.
> The correct next step is codegen tranche scoping.

## Decision

```
┌─────────────────────────────────────────────────────────────────┐
│  FAMILY 06 MANUAL EXPANSION: ABORTED                            │
│                                                                  │
│  Reason: no candidate passes the "no write-path changes"         │
│  condition (S190 C1). The analytical read-path is already         │
│  generic enough to cover within-layer variants — the bottleneck   │
│  is the write side, which the gate explicitly protects.           │
│                                                                  │
│  Next step: codegen tranche scoping (S192).                      │
│  Manual expansion pattern: retired after 6 families.             │
└─────────────────────────────────────────────────────────────────┘
```

## Why Abort Is the Correct Decision

### 1. The Gate Conditions Are Not Negotiable

S190 set four conditions for Family 06 authorization. Condition C1 — "candidate must NOT require write-path changes" — exists for a specific reason: Wave B's safety guarantee was that each analytical expansion only added read-path artifacts without touching the writer. This guarantee held across 6 families and was the foundation of the "zero creative decisions" claim.

Relaxing C1 to accommodate EMA Crossover (the closest candidate) would:
- Break the 6-family precedent of zero write-path modifications.
- Set a precedent that gate conditions can be softened when convenient.
- Undermine the discipline that kept Wave B healthy.

**The gate exists to prevent exactly this kind of rationalization.**

### 2. The Read Path Already Covers Within-Layer Variants

The most surprising finding of this assessment is that the analytical read infrastructure is **already generic**:

| Reader | Type Parameter | Already Supports |
|--------|---------------|------------------|
| `SignalReader.QuerySignalHistory()` | `signalType` | Any signal type (rsi, ema_crossover, macd, ...) |
| `DecisionReader.QueryDecisionHistory()` | `decisionType` | Any decision type |
| `StrategyReader.QueryStrategyHistory()` | `strategyType` | Any strategy type |
| `RiskReader.QueryRiskHistory()` | `riskType` | Any risk type |
| `ExecutionReader.QueryExecutionHistory()` | `executionType` | Any execution type |
| `CandleReader.QueryCandleHistory()` | Implicit (candle) | Candle evidence |

For layers L2–L6, the reader, use case, handler, and HTTP endpoint already accept a `type` parameter. Querying `GET /analytical/signal/history?type=ema_crossover` works today — it returns empty results because no EMA crossover data exists in ClickHouse, not because the read path can't handle it.

**There is no analytical read-path work to do for within-layer variants.** The 9-artifact expansion pattern was designed for new layers (L1→L6). Within-layer variants only need write-path work — which the gate blocks.

### 3. EMA Crossover Is a Write-Path Problem, Not an Analytical Problem

EMA Crossover — the strongest candidate — requires exactly one change: a writer pipeline entry. Everything else is ready:

| Artifact | Status for EMA Crossover |
|----------|--------------------------|
| Migration (DDL) | ✅ `signals` table (002) already exists, schema is type-agnostic |
| Writer mapper | ✅ `mapSignalRow()` is generic for all `signal.SignalGeneratedEvent` types |
| Writer pipeline entry | ❌ **Missing** — needs `WriterEMACrossoverSignalConsumer()` + catalog entry |
| Reader adapter | ✅ `SignalReader` already parameterized by type |
| Use case | ✅ `GetSignalHistory` already generic |
| Handler | ✅ `GetSignalHistory` handler already accepts any type |
| Route | ✅ `/analytical/signal/history` already registered |
| Tests | ✅ Reader/handler tests already cover type-parameterized queries |
| Smoke | ⚠️ Would need new phase section |

The only missing artifact is write-side. Framing this as an "analytical family expansion" would be misleading — it's a writer pipeline addition that happens to enable analytical queries for a new event type.

### 4. The Manual Pattern Has Reached Its Natural Completion

Six families, six vertical layers, zero creative decisions, zero write-path modifications. The pattern delivered exactly what it was designed to deliver: proven, reproducible analytical read-path coverage for every layer in the trading pipeline.

Continuing the manual pattern requires either:
- (a) Relaxing gate conditions — which defeats the purpose of gates.
- (b) Redefining "family" to include write-path work — which changes the expansion pattern's scope.
- (c) Finding a pure read-path candidate — which doesn't exist given the generic readers.

None of these options are honest. The correct acknowledgment is: **the manual analytical expansion pattern is complete.**

## What Abort Does NOT Mean

1. **It does NOT mean EMA Crossover is unimportant.** It means EMA Crossover is a write-path enablement task, not an analytical expansion task. It should be pursued when the writer pipeline naturally extends — potentially as part of codegen scope.

2. **It does NOT mean the analytical layer is finished.** It means the next analytical improvements come from codegen (generating families at near-zero marginal cost) or from cross-family features (aggregations, composite queries) — not from hand-crafting a 7th copy of the same template.

3. **It does NOT mean Wave B failed.** Wave B succeeded beyond expectations: 6 families, full vertical coverage, proven pattern, quantified limits. Abort is the mark of a disciplined process, not a failure.

4. **It does NOT mean write-path work is blocked.** The writer can add EMA Crossover (or any other pipeline entry) independently of the analytical expansion pattern. Write-path work simply isn't governed by the analytical gate.

## Candidates Deferred and Their Proper Scope

| Candidate | Proper Scope | When |
|-----------|-------------|------|
| EMA Crossover | Writer pipeline extension (1 consumer spec + 1 pipeline entry) | When write-path extension is strategically justified |
| Venue Market Order | Writer pipeline extension (new mapper + consumer + pipeline) | When venue execution flow is active |
| Trade Burst | Full new family (migration + writer + reader) — codegen candidate | Post-codegen implementation |
| Volume Metrics | Full new family (migration + writer + reader) — codegen candidate | Post-codegen implementation |

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Abort perceived as loss of momentum | Medium | Low | Wave B delivered complete vertical coverage; abort is completion, not failure |
| Codegen tranche is deprioritized after abort | Low | High | S192 is the explicit next step; the gate mandated codegen scope before Family 07 |
| Write-path extensions happen without analytical coordination | Low | Medium | Codegen will unify write+read generation; ad-hoc extensions carry low risk |
| Manual pattern knowledge decays before codegen captures it | Medium | Medium | 6 families + extensive documentation preserve the pattern specification |

## Formal Triggers for Future Analytical Expansion

The manual expansion pattern is retired. Future analytical families are authorized only through:

1. **Codegen path**: After codegen implementation, families are generated (not hand-crafted). Gate conditions become codegen-template conditions.
2. **Write-path enablement**: When a new writer pipeline entry is added for operational reasons, the corresponding analytical coverage can be activated through codegen.
3. **New layer discovery**: If a genuinely new analytical layer emerges (not a within-layer variant), it may justify a manual expansion — but this would be exceptional and must be individually assessed.

## Conclusion

The S190 gate conditions worked exactly as designed. They prevented an expansion that would have either (a) violated the zero-write-path-change guarantee, or (b) produced a "family" with no meaningful analytical work. The manual expansion pattern is complete at 6 families. The next strategic investment is codegen, which will make future family expansion trivially cheap instead of artisanally expensive.
