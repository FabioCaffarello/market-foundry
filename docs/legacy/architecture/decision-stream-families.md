# Decision Stream Families — Market Foundry

> Canonical catalog of decision families in the Market Foundry mesh.
> Stage: S42 — Design only. Implementation deferred to S43+.
> Approved: 2026-03-17

---

## 1. What Is a Decision Family

A **decision family** is a named group of related decision flows sharing:
- A common evaluation boundary (what signals it consumes, what condition it evaluates)
- A single producing binary (derive)
- A consistent subject encoding pattern within `DECISION_EVENTS`
- Shared JetStream stream (`DECISION_EVENTS`)
- Common retention semantics
- One projection pipeline in store

Decision families follow the same structural rules as evidence families and signal families.

---

## 2. Family Catalog

### DF-01: RSI Oversold — Phase 1

| Field | Value |
|---|---|
| **Canonical name** | `rsi_oversold` |
| **Bounded context** | Decision |
| **Classification** | Evaluated |
| **Input signals** | RSI (SF-02) |
| **Evaluation logic** | RSI value crosses below configurable threshold (default: 30.0) |
| **Outcome semantics** | `triggered` = RSI below threshold; `not_triggered` = RSI at or above threshold; `insufficient` = warm-up not complete |
| **Publisher** | DecisionPublisherActor (derive) |
| **Consumer** | DecisionConsumerActor (store) |
| **Projection** | DecisionProjectionActor → `DECISION_RSI_OVERSOLD_LATEST` |
| **Query** | `decision.query.rsi_oversold.latest` |
| **HTTP** | `GET /decision/rsi_oversold/latest?source=X&symbol=Y&timeframe=Z` |
| **Phase** | 1 (S43) |
| **Readiness** | Design complete, pending implementation |

**Evaluator contract**:
```
Input:  signal type=rsi, value (decimal string), final=true
Output: Decision with outcome ∈ {triggered, not_triggered, insufficient}
Config: threshold (decimal string, default "30.0")
Pure:   yes — no I/O, no side effects
```

**Why RSI Oversold first**: It consumes a single signal (RSI), has the simplest possible
evaluation logic (threshold comparison), and proves the entire decision pipeline end-to-end
with minimal risk. This mirrors how RSI was the first signal family.

---

### DF-02: MACD Crossover — Phase 1 (stretch)

| Field | Value |
|---|---|
| **Canonical name** | `macd_crossover` |
| **Bounded context** | Decision |
| **Classification** | Evaluated |
| **Input signals** | MACD (SF-01) — requires MACD signal implementation first |
| **Evaluation logic** | MACD line crosses above signal line (bullish) or below (bearish) |
| **Outcome semantics** | `triggered` = crossover detected; `not_triggered` = no crossover; `insufficient` = warm-up |
| **Publisher** | DecisionPublisherActor (derive) |
| **Consumer** | DecisionConsumerActor (store) |
| **Projection** | DecisionProjectionActor → `DECISION_MACD_CROSSOVER_LATEST` |
| **Query** | `decision.query.macd_crossover.latest` |
| **HTTP** | `GET /decision/macd_crossover/latest?source=X&symbol=Y&timeframe=Z` |
| **Phase** | 1 (stretch — requires MACD signal, which is deferred) |
| **Readiness** | Blocked on MACD signal implementation |

**Note**: MACD crossover requires stateful detection (previous vs current relative position
of MACD and signal line). The evaluator must track the previous state to detect the cross.
This makes it a good second family — more complex than threshold comparison but still
single-signal input.

---

### DF-03: RSI-MACD Confluence — Deferred (S44+)

| Field | Value |
|---|---|
| **Canonical name** | `confluence_rsi_macd` |
| **Bounded context** | Decision |
| **Classification** | Evaluated (multi-signal) |
| **Input signals** | RSI (SF-02) + MACD (SF-01) |
| **Evaluation logic** | Both RSI oversold AND MACD bullish crossover active within a configurable window |
| **Phase** | Deferred — S44+ |
| **Readiness** | Requires DF-01 + DF-02 proven, plus multi-signal evaluation pattern design |

**Why deferred**: Multi-signal confluence introduces temporal alignment questions
(how close in time must the two signals be?) that require careful design. Single-signal
families must prove the pipeline first.

---

### DF-04: Volume Spike Entry — Deferred (S45+)

| Field | Value |
|---|---|
| **Canonical name** | `volume_spike_entry` |
| **Bounded context** | Decision |
| **Classification** | Evaluated |
| **Input signals** | Volume-Weighted Momentum (SF-03) — not yet implemented |
| **Phase** | Deferred — S45+ |
| **Readiness** | Blocked on SF-03 signal implementation |

---

## 3. Stream Definition

```
Stream:          DECISION_EVENTS
Subjects:        decision.events.>
Retention:       72h, file-backed
Max bytes:       2 GB
Deduplication:   MsgId-based (Decision.DeduplicationKey())
Discard policy:  Old
```

Subject pattern:
```
decision.events.{type}.evaluated.{source}.{symbol}.{timeframe}
```

Examples:
```
decision.events.rsi_oversold.evaluated.binancef.btcusdt.60
decision.events.macd_crossover.evaluated.binancef.ethusdt.300
```

---

## 4. KV Bucket Naming

| Family | Latest Bucket | History Bucket (deferred) |
|---|---|---|
| rsi_oversold | `DECISION_RSI_OVERSOLD_LATEST` | `DECISION_RSI_OVERSOLD_HISTORY` |
| macd_crossover | `DECISION_MACD_CROSSOVER_LATEST` | `DECISION_MACD_CROSSOVER_HISTORY` |

Key format: `{source}.{symbol}.{timeframe}`
History key format (deferred): `{source}.{symbol}.{timeframe}.{timestamp_unix}`

---

## 5. Envelope Types

| Plane | Envelope |
|---|---|
| Events | `decision.events.v1.{type}_evaluated` |
| Query request | `decision.query.v1.{type}_latest_request` |
| Query reply | `decision.query.v1.{type}_latest_reply` |

---

## 6. Family Invariants

| ID | Invariant |
|---|---|
| **FI-1** | One stream for all decision families: `DECISION_EVENTS` |
| **FI-2** | One publisher actor shared across decision families within a scope |
| **FI-3** | One consumer + projection actor pair per decision family in store |
| **FI-4** | Each family has independent KV buckets — no shared state |
| **FI-5** | Family names are lowercase, singular, underscore-separated |
| **FI-6** | Each family must have a documented evaluator contract before implementation |
| **FI-7** | Single-signal families before multi-signal families |
| **FI-8** | No family may consume evidence directly — only signals |

---

## 7. Growth Pattern

Adding a new decision family requires:

1. **Design**: Document evaluator contract, input signals, outcome semantics
2. **Domain**: Add evaluator in `internal/application/decision/`
3. **Actor**: Add `DecisionEvaluatorActor` variant in derive scope
4. **Config**: Add family name to `pipeline.decision_families` schema
5. **Store**: Add consumer + projection actor pair, KV bucket(s)
6. **Query**: Add route in QueryResponderActor
7. **Gateway**: Add HTTP endpoint under `/decision/{type}/`
8. **Governance**: Update raccoon-cli drift rules
9. **Test**: Unit test evaluator (pure), integration test pipeline

This checklist mirrors the evidence and signal family growth patterns.

---

## References

- [decision-domain-design.md](decision-domain-design.md) — Domain design
- [signal-stream-families.md](signal-stream-families.md) — Signal family precedent
- [stream-family-catalog.md](stream-family-catalog.md) — Mesh-wide catalog
- [evidence-derivation-pattern.md](evidence-derivation-pattern.md) — Derivation pattern
