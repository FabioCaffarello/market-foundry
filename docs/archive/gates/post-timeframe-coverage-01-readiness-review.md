# Post-Timeframe-Coverage-01: Readiness Review

> Formal assessment of Market Foundry readiness after TC-01 wave (S131–S135).
> Date: 2026-03-19 | Stage: S136

---

## 1. Executive Summary

TC-01 expanded the temporal matrix from 2 to 4 timeframes (adding 900s and 3600s) with **zero Go source code changes**. The entire expansion was config-driven, validating the core architectural thesis established in S10–S15. All 11 of 13 mandatory success criteria passed; 2 were deferred due to physics constraints (RSI warm-up at long windows requires hours of market data). Two P1 refactors were executed (config validation, recovery runbook). Seven items were deferred with explicit triggers. Nine items were permanently accepted as inherent to the domain.

**Verdict: The Foundry is architecturally robust for its current scope. The expansion proved real scalability. The system is ready for a next wave — but the next wave should not be another temporal expansion.**

---

## 2. Did the Expansion Prove Robustness?

### 2.1 Yes — Config-Driven Scaling Works

The single change to `derive.jsonc` propagated correctly through all six pipeline stages (evidence → signal → decision → strategy → risk → execution). Actor spawning, NATS subject routing, KV materialization, HTTP query surfaces, and deduplication keys all scaled linearly:

| Resource | Before (2 TF) | After (4 TF) | Growth |
|----------|---------------|---------------|--------|
| Evidence samplers/symbol | 6 | 12 | 2× |
| NATS subjects | ~32 | ~64 | 2× |
| KV keys | ~32 | ~64 | 2× |
| Write load increase | baseline | +<30% | sublinear |

Write load grew sublinearly because longer timeframes produce fewer writes per hour (3600s writes once/hour vs 60s writing 60×/hour).

### 2.2 Yes — No Combinatorial Explosion

Six anticipated problems did **not** materialize:
- NATS stream pressure from 64 subjects: negligible
- Fan-out latency at 4×: unmeasurable (<10μs overhead)
- KV write contention: higher TFs write less
- Dedup key collision: timeframe embedded in partition key
- Cross-timeframe signal interference: per-TF subject routing prevents it
- Memory accumulation at 3600s: O(1) accumulator, not O(trades)

### 2.3 Qualified — Two Criteria Remain Unverified

M7 (RSI convergence at 900s) and M8 (RSI convergence at 3600s) require extended runtime (6+ hours and 15+ hours respectively). These are physics constraints, not architecture failures. The RSI evaluator works identically across timeframes; only the warm-up period differs.

---

## 3. What Parts Are Genuinely Robust?

### Tier 1 — Solid (no reservations)

- **Config propagation**: Single config change reaches all actors, all domains
- **Actor lifecycle**: SourceScopeActor spawns per-symbol × per-timeframe correctly
- **NATS routing**: Subject partitioning by `{source}.{symbol}.{timeframe}` is clean
- **KV materialization**: Each TF gets its own key space, no collision
- **HTTP query surface**: All endpoints accept `timeframe` param, return correct data
- **Deduplication**: Partition keys include timeframe, zero cross-TF interference
- **Smoke/integration tests**: Cover all 4 timeframes across all domains

### Tier 2 — Adequate (works, with known limits)

- **Diagnostics/tracking**: Aggregate counters work; per-TF granularity absent but not yet needed
- **Operational docs**: Recovery runbook added in S135; adequate for current 4 TF scope
- **Config validation**: Range [10, 86400], no duplicates — catches misconfiguration at startup

### Tier 3 — Acceptable Risk at Current Scale

- **In-memory window state**: 3600s max loss on crash is acceptable; becomes problematic at 4h+
- **Global timeframe list**: All symbols share same TFs; fine until heterogeneous needs arise

---

## 4. What Still Imposes Recurrent Friction?

### Active Friction (felt during TC-01)

| ID | Friction | Impact | Status |
|----|----------|--------|--------|
| F-02 | No config validation | Could deploy invalid TFs | **Resolved in S135** |
| F-17 | No recovery runbook | Operators blind on crash | **Resolved in S135** |
| F-13 | Window state loss on crash | Up to 60 min data loss at 3600s | Accepted for TC-01; **hard gate for TC-02** |

### Latent Friction (not felt at 4 TFs, will surface at scale)

| ID | Friction | Trigger |
|----|----------|---------|
| F-01 | Global timeframe list | Per-symbol heterogeneous TF needs |
| F-04 | Single tracker for evidence | 8+ TFs or diagnostics incident |
| F-05 | No per-TF idle detection | 4h+ TFs where stalls are critical |
| F-15 | No interim candle snapshots | 4h+ TFs where partial state has value |
| F-19 | No gateway aggregate view | 5+ symbols or dashboard needs |

### Non-Friction (permanently accepted)

Integer-only TF representation, log scaling with cardinality, HTTP test file repetition, actor/KV/subject growth — these are inherent to the domain and carry zero structural cost.

---

## 5. Did Triggered Refactors Have Real Payoff?

### R-01: Config Validation — Yes

- **Cost**: ~25 lines production code, ~35 lines tests
- **Payoff**: Prevents silent deployment of invalid timeframes (duplicates, out-of-range values)
- **Evidence**: Would have caught a hypothetical `[60, 60, 300]` misconfiguration that would produce duplicate actors
- **Verdict**: Minimal cost, permanent value. Clear payoff.

### R-02: Recovery Runbook — Yes

- **Cost**: Documentation only, zero code
- **Payoff**: Quantifies data loss expectations per timeframe for operators
- **Evidence**: Without this, a 3600s crash would leave operators guessing about recovery
- **Verdict**: Zero cost, real operational value. Clear payoff.

### Both refactors were correctly scoped — neither over-engineered, neither left open-ended.

---

## 6. Assessment of Deferred Items

### Worth Doing When Triggered

| Item | When | Why |
|------|------|-----|
| F-13+F-15: State persistence | Before TC-02 4h+ TFs | 4-hour data loss on crash is operationally unacceptable |
| F-04: Per-TF tracker | At 8+ TFs or diagnostics need | Current aggregate is adequate at 4 |
| F-01: Per-binding TFs | When symbols need different TF sets | Global list works for uniform config |

### Not Worth Doing Now (or Possibly Ever)

| Item | Why Not |
|------|---------|
| F-07: List timeframes endpoint | No external consumers; config is the source of truth |
| F-08: Null response disambiguation | Only matters for non-expert operators; HTTP 200 with null is standard |
| F-19: Gateway aggregate | Single-symbol focus; premature until 5+ symbols |
| F-06: Log scaling | Inherent; no fix possible or needed |
| F-03: Integer-only TFs | Works; "15m" labels are sugar, not architecture |

---

## 7. Readiness Verdict

| Dimension | Rating | Notes |
|-----------|--------|-------|
| Config-driven scaling | **Strong** | Zero-code expansion proven |
| Pipeline correctness | **Strong** | All domains handle 4 TFs correctly |
| Query surface | **Strong** | HTTP endpoints work for all TFs |
| Test coverage | **Strong** | Unit + smoke + HTTP tests cover all 4 TFs |
| Diagnostics | **Adequate** | Aggregate tracking; per-TF deferred |
| State resilience | **Acceptable** | In-memory only; hard gate before 4h+ |
| Operational readiness | **Adequate** | Runbook exists; basic but sufficient |
| Documentation | **Strong** | TC-01 fully documented across 11 architecture docs |

**The Foundry has proven its temporal scaling thesis. It is ready for a next wave that is not another temporal expansion.**
