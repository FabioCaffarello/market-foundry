# Decision Layer — Risks and Blockers

> Catalogs the risks, blockers, and mitigations associated with introducing
> a decision/strategy domain into Market Foundry.
>
> **Date**: 2026-03-17
> **Source**: S38 Decision Readiness Review

---

## Blockers (Must Resolve Before Decision Entry)

### B-1: Signal Adapter Tests Missing

**Impact**: High
**Layer**: Signal → Store

Signal's NATS adapters (registry, KV store) have no unit tests. The signal
registry defines stream subjects and consumer durable names that decision
consumers will depend on. Without tests, a refactor to signal registry could
silently break the contract that decision reads from.

**Mitigation**: Complete P-1 and P-2 (see `decision-entry-prerequisites.md`).

**If ignored**: Decision consumer binds to signal subjects that are not
contractually locked. Any signal registry change forces discovery-by-failure
in the decision layer.

---

### B-2: No Signal Governance in raccoon-cli

**Impact**: High
**Layer**: Governance

raccoon-cli validates evidence contracts (subjects, streams, registries) but
is blind to signal contracts. The quality gate (`make check`, `make verify`)
does not catch signal drift. This means:
- Signal subjects can diverge from documentation without detection
- Signal KV bucket naming can drift without warning
- Signal consumer durable names can conflict without audit

**Mitigation**: Complete P-4.

**If ignored**: Two ungoverned domains (signal + decision) stack. Drift
compounds exponentially — a signal change breaks decision, which has no
governance to catch it either.

---

### B-3: Signal Multi-Symbol Not Verified

**Impact**: Medium
**Layer**: Signal → Derive → Store

Signal pipeline has only been proven for single-symbol flows. The partition
key `{source}.{symbol}.{timeframe}` should prevent cross-symbol contamination,
but this has not been verified under concurrent multi-symbol operation.

**Mitigation**: Complete P-5.

**If ignored**: A decision layer processing multiple symbols may receive
stale or missing signals for non-primary symbols. Discovery happens in
production, not in test.

---

## Risks (Accept or Mitigate, Do Not Block)

### R-1: Zero Actor-Level Tests (Systemic)

**Impact**: Medium
**Probability**: Ongoing
**Layer**: All actor scopes

No actor scope has unit tests — not observation, not evidence, not signal,
not store. Actors are tested indirectly through smoke tests and integration
tests (`make smoke`, `make smoke-multi`), but there are no isolated tests
for actor message handling, supervision trees, or lifecycle behavior.

**Why not a blocker**: This is a systemic gap shared by all existing domains.
Evidence and observation have operated reliably without actor tests through
36 stages. The risk is real but equally distributed — it doesn't specifically
penalize decision more than any other layer.

**Mitigation path**: A dedicated "actor test infrastructure" stage could
introduce test helpers for Hollywood-based actors. This is orthogonal to
decision entry and can proceed in parallel.

---

### R-2: RSI-Only Signal Domain

**Impact**: Low-Medium
**Probability**: Certain (current state)
**Layer**: Signal

Signal currently produces RSI only. A decision layer designed for RSI
is viable but narrow. If decision needs MACD or other indicators, the
signal domain must expand first.

**Why not a blocker**: Decision can validly start with RSI as the sole
signal input. The signal registry's `LatestSpecByType()` dispatch is
ready for new types — adding MACD does not require architectural changes.

**Mitigation path**: Introduce MACD sampler as a parallel track once
decision design clarifies which signal types it needs.

---

### R-3: No Signal History Lookback

**Impact**: Medium
**Probability**: Depends on decision design
**Layer**: Signal → Store

Signal provides only the latest value per partition key. There is no
signal history bucket. If decision needs "RSI over the last N candles"
rather than "current RSI," the signal domain needs a history projection
following the candle history pattern.

**Why not a blocker**: Many decision strategies operate on the current
signal value. If the first decision slice is designed around latest-only
signals, history is not needed. If history is needed, it's a well-understood
extension of the existing pattern.

**Mitigation path**: Defer to decision domain design (S42). If the chosen
strategy needs lookback, add signal history as a dependency before the
decision implementation stage.

---

### R-4: Evidence KV Store Test Gaps

**Impact**: Low-Medium
**Probability**: Low (stores follow proven candle pattern)
**Layer**: Evidence → Store

`trade_burst_kv_store.go` and `volume_kv_store.go` have no unit tests.
They follow the candle KV store pattern, which is tested. The risk is
that subtle divergences from the pattern (different error handling, missing
nil guards) go undetected.

**Why not a blocker at the decision level**: Decision primarily consumes
candle and signal data. Trade burst and volume are supplementary inputs.
However, closing this gap (P-3) is low-effort and high-value.

---

### R-5: No Formal Contracts Between Layers

**Impact**: Medium
**Probability**: Increases with domain count
**Layer**: Cross-cutting

The contract between observation → evidence → signal is implicit (shared
NATS subjects, CBOR encoding, event types). There is no formal contract
specification (like protobuf schemas or JSON Schema) that can be validated
at compile time. raccoon-cli performs static analysis on Go code, but the
message payload contracts are defined by convention, not by schema.

**Why not a blocker**: This has worked for three domains. The risk grows
with each new domain but doesn't cross a critical threshold with the
addition of one more (decision). The convention-based approach is the
documented architectural choice (NG-15: no gRPC/protobuf internally).

**Mitigation path**: Consider a lightweight contract specification if the
convention-based approach shows friction during decision implementation.

---

### R-6: Ack-Before-Projection Window

**Impact**: Low
**Probability**: Low (bounded by design)
**Layer**: Store

All projection actors (evidence and signal) ack the JetStream message
before writing to KV. A crash between ack and write loses at most one
event per partition key. For evidence, the next candle close overwrites.
For signal, the next signal computation overwrites.

**Why not a blocker**: The window is bounded and self-healing. A decision
layer reading from KV always gets the latest written value — it cannot
observe the gap.

---

## Risk-Reward Assessment

| Factor | Assessment |
|--------|-----------|
| Structural readiness | High — patterns are consistent across all layers |
| Contract stability | Medium — no formal schemas, convention-based |
| Test confidence | Medium — domain/application tested, actors not tested |
| Governance coverage | Medium — evidence governed, signal not yet |
| Operational proof | Medium — smoke tests exist, no production deployment |

**Bottom line**: The risk of opening decision is manageable IF the two
blockers (B-1/B-2 signal tests + governance, B-3 multi-symbol) are resolved
first. The remaining risks (R-1 through R-6) are acceptable trade-offs
that can be mitigated in parallel with decision design and implementation.

---

## Smallest Acceptable Decision Layer Design

If the prerequisites are met, the minimal decision domain would be:

```
internal/domain/decision/        — decision entity + Validate()
internal/application/decision/   — decision evaluator (consumes signal + evidence)
internal/actors/scopes/decision/ — decision supervisor, consumer, evaluator actor
internal/adapters/nats/          — decision_registry, decision_publisher, decision_kv_store
internal/interfaces/http/        — handlers/decision.go, routes/decision.go
cmd/decision/                    — service entrypoint
deploy/configs/decision.jsonc    — config with pipeline.decision_families
```

This follows the exact structural pattern of every existing domain:
- Actor-per-family supervision
- JetStream stream + durable consumer
- KV bucket for latest projection
- NATS request/reply query path
- HTTP gateway routes
- Config-driven activation

The decision evaluator would:
1. Consume latest signal (RSI) via NATS request/reply
2. Consume latest evidence (candle) via NATS request/reply
3. Apply a decision rule (e.g., RSI threshold crossing)
4. Publish decision events to a `DECISION_EVENTS` stream
5. Project decisions to a `DECISION_LATEST` KV bucket

This design adds zero new patterns. It is the observation→evidence→signal
pattern applied one layer higher.
