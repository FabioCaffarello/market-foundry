# CC-02: Gains, Trade-offs, and Open Debts

> Evidence-based ledger of what CC-02 (S125–S129) produced, what it traded, and what it left behind.

---

## 1. Gains (Permanent, Compound with Scale)

### G1: Proven Code Extensibility

**Evidence:** `ema_crossover` implemented following Playbook 1 exactly — 3 new files, 7 modified files, ~414 lines total. Zero domain model changes. Zero infrastructure changes. Zero regressions.

**Compound effect:** Every future signal family can follow the same playbook with the same cost envelope. The playbook is now validated by two independent implementations (RSI, EMA crossover).

### G2: Domain Model Generality Confirmed

**Evidence:** `signal.Signal` with `string` Value and `map[string]string` Metadata handled both numeric (RSI: "72.45") and categorical (EMA crossover: "bullish"/"bearish"/"neutral") signal types without any structural modification.

**Compound effect:** Multi-parameter signals (EMA crossover uses 5 metadata keys vs. RSI's 1) are naturally supported. The domain model won't need changes for signals of comparable complexity.

### G3: Infrastructure Actor Reuse Validated

**Evidence:** `SignalPublisherActor`, `SignalProjectionActor`, `SignalConsumerActor`, and query responder — all reused without code changes. Only computation actor (`EMACrossoverSignalSamplerActor`) was new.

**Compound effect:** Adding family N+1 never requires infrastructure changes. The cost of extension is purely in domain logic and registration.

### G4: Coexistence Isolation Proven

**Evidence:** RSI code paths received zero modifications during CC-02. Separate KV buckets (`SIGNAL_RSI_LATEST`, `SIGNAL_EMA_CROSSOVER_LATEST`), separate consumers (distinct durable names), shared `SIGNAL_EVENTS` stream via wildcard subjects. Both families run concurrently for 2 symbols without interference.

**Compound effect:** Family independence is structural, not accidental. Config-driven activation means families can be enabled/disabled independently.

### G5: Playbook Reproducibility Confirmed

**Evidence:** Playbook 1 predicted 3 new files + 7 modified files. Actual: 3 new + 7 modified. Predicted line budget: ~400. Actual: ~414. Implementation followed the exact 7-step sequence.

**Compound effect:** New contributors can follow the playbook to add a signal family without architectural guidance. The playbook is a reliable contract.

### G6: HTTP Correlation ID Middleware Delivered

**Evidence:** S129 extracted correlation ID middleware from 7 handler files. 12 manual extractions removed. Every future handler inherits correlation ID automatically.

**Compound effect:** HTTP-layer observability scales without per-handler effort. Cross-request tracing is guaranteed.

### G7: Extensibility Cost Model Established

**Evidence (S128 friction capture):**

| Category | Lines per Family | Friction Level |
|----------|-----------------|---------------|
| Domain sampler + tests | ~240 | Low (unique logic) |
| Sampler actor | ~97 | Medium-High (boilerplate) |
| NATS registry/publisher | ~37 | Medium (scattered) |
| Store pipeline | ~25 | Low-Medium |
| Derive processor | ~10 | Low |
| Config schema | ~4 | Trivial |
| HTTP routes | 0 | None |
| Diagnostics | 0 | None |
| **Total** | **~414** | — |

**Compound effect:** Cost of extension is measurable and budgetable. Architecture decisions can be made against real numbers, not estimates.

---

## 2. Trade-offs Accepted

### T1: Actor Boilerplate at N=2

**What:** `EMACrossoverSignalSamplerActor` is ~95% identical to `RSISignalSamplerActor` (~97 lines each). No generic actor exists.

**Why accepted:** Two data points are insufficient to design a stable abstraction. The boilerplate is mechanical and error-free. Premature extraction risks wrong API surface.

**Revisit condition:** N=3 signal families. At that point, the pattern is stable enough for `SignalSamplerActor[T Sampler]`.

### T2: NATS Registry Switch Dispatch

**What:** 4 touch points per family across 3–4 files (registry, publisher, kv_store). Switch/case dispatch, not map-based.

**Why accepted:** Functional and type-safe at N=2. Map-based registry is a straightforward 1–2 hour conversion but not justified by two entries.

**Revisit condition:** N=3 families. Map-based registry centralizes all dispatch.

### T3: Hardcoded Signal Parameters

**What:** EMA periods (9/21) and RSI period (14) hardcoded in sampler constructors. No per-family configuration mechanism.

**Why accepted:** Intentional simplification. Configuration surface should be demand-driven. Adding parameterization without a consumer (A/B testing, per-binding tuning) creates unused complexity.

**Revisit condition:** Concrete A/B testing or multi-parameter optimization requirement.

### T4: Signal-Domain-Only Proof

**What:** CC-02 tested extensibility within the signal domain only. Decision, strategy, risk, execution families for EMA crossover were explicitly out of scope.

**Why accepted:** Bounded scope produces cleaner evidence. Cross-domain extensibility is a different question with different pressure points.

**Revisit condition:** Next capability wave, if cross-domain integration is the strategic goal.

### T5: 21-Minute Warm-Up Latency

**What:** EMA crossover requires 21 candles before first signal. At 60s timeframe, that's ~21 minutes of null responses.

**Why accepted:** Mathematical requirement of SMA seeding for EMA calculation. Cannot be shortened without algorithmic compromise.

**Revisit condition:** Never — inherent to EMA computation.

---

## 3. Open Debts

### 3.1 Debts Converging at N=3 (CC-03 Bundling Point)

| ID | Debt | Estimated Effort | Payoff at N=3 |
|----|------|-----------------|--------------|
| CF-08 | Generic `SignalSamplerActor` factory | ~2 hours | Eliminates ~97 lines/family; single maintenance point |
| CF-11 | Map-based NATS registry (replace switches) | ~1–2 hours | Centralizes 4 scattered touch points per family |
| CF-03 (actor) | Correlation ID injection in actor middleware | ~2–3 hours | Automatic propagation; eliminates silent-break risk |
| **Total** | | **~5–7 hours** | |

These three debts are **independently justified at N=3** and **synergistic when bundled**. The generic actor naturally becomes the injection point for correlation ID middleware. The map-based registry naturally integrates with the generic actor's type dispatch.

### 3.2 Debts with Later Triggers

| ID | Debt | Trigger | Estimated Effort |
|----|------|---------|-----------------|
| CF-12 | Store pipeline boilerplate reduction | N=5 families | ~2 hours |
| CF-02 | Active symbols endpoint | Next configctl route change or N>5 symbols | ~1 hour |
| CF-13 | Per-family algorithm configuration | A/B testing requirement | ~3–4 hours |
| D4 | Composition root unit tests | First wiring error escape | ~2–3 hours |
| D5 | Failure recovery validation | Pre-production deployment | ~4–6 hours |
| D6 | Soak testing infrastructure | N>5 symbols or 24-hour operation | ~2–3 hours |

### 3.3 Debts That Do NOT Warrant Action

| ID | Debt | Why Not Now |
|----|------|------------|
| CF-06 | Schema manual map entries (2 per family) | Maps are the right abstraction; ~4 lines/family is negligible |
| CF-09 | Diagnostic surface code | Already zero-friction; actor-driven and auto-include |
| CF-10 | HTTP route extensibility | Already zero-friction; type-parameterized |

---

## 4. Debt-to-Gain Ratio

**Total gains:** 7 permanent, compounding gains (code extensibility, domain model generality, infrastructure reuse, coexistence isolation, playbook reproducibility, HTTP middleware, cost model).

**Total accepted trade-offs:** 5 documented, with clear revisit conditions.

**Total open debt:** ~20–30 hours across all items. Of this:
- ~5–7 hours are naturally bundled at CC-03 (N=3 threshold)
- ~12–18 hours are deferred to later, concrete triggers
- ~0 hours are immediate

**Assessment:** The debt-to-gain ratio is healthy. No debt blocks the next capability. No debt has produced an incident. The governance model (threshold-based triggers) has proven accurate at predicting when debts become worth resolving.
