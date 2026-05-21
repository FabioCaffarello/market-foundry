# Strategy Stream Families — Market Foundry

> Canonical catalog of strategy families in the Market Foundry mesh.
> Stage: S53 — Design only. Implementation deferred to S54+.
> Date: 2026-03-17

---

## 1. What Is a Strategy Family

A **strategy family** is a named group of related strategy flows sharing:
- A common resolution boundary (what decisions it consumes, what trade intent it produces)
- A single producing binary (derive)
- A consistent subject encoding pattern within `STRATEGY_EVENTS`
- Shared JetStream stream (`STRATEGY_EVENTS`)
- Common retention semantics
- One projection pipeline in store

Strategy families follow the same structural rules as decision families, signal families, and evidence families.

---

## 2. Family Catalog

### STF-01: Mean Reversion Entry — Phase 1

| Field | Value |
|---|---|
| **Canonical name** | `mean_reversion_entry` |
| **Bounded context** | Strategy |
| **Classification** | Resolved |
| **Input decisions** | RSI Oversold (DF-01) |
| **Resolution logic** | When `rsi_oversold` triggers, resolve to LONG with entry at market, target at configurable offset above entry, stop at configurable offset below entry |
| **Direction semantics** | `long` = RSI oversold entry condition active; `flat` = no entry condition (decision not triggered or insufficient) |
| **Publisher** | StrategyPublisherActor (derive) |
| **Consumer** | StrategyConsumerActor (store) |
| **Projection** | StrategyProjectionActor → `STRATEGY_MEAN_REVERSION_ENTRY_LATEST` |
| **Query** | `strategy.query.mean_reversion_entry.latest` |
| **HTTP** | `GET /strategy/mean_reversion_entry/latest?source=X&symbol=Y&timeframe=Z` |
| **Phase** | 1 (S54) |
| **Readiness** | Design complete, pending implementation |

**Resolver contract**:
```
Input:  decision type=rsi_oversold, outcome ∈ {triggered, not_triggered, insufficient}, confidence (decimal), final=true
Output: Strategy with direction ∈ {long, flat}
Config: target_offset (decimal string, default "0.02" = 2%), stop_offset (decimal string, default "0.01" = 1%)
Pure:   yes — no I/O, no side effects
```

**Resolution rules**:
- `rsi_oversold` outcome = `triggered` → direction = `long`, confidence inherited from decision
- `rsi_oversold` outcome = `not_triggered` → direction = `flat`, confidence = `"0.0"`
- `rsi_oversold` outcome = `insufficient` → direction = `flat`, confidence = `"0.0"`, metadata includes `reason: "insufficient_data"`

**Parameters output** (when direction = `long`):
```json
{
  "entry": "market",
  "target_offset": "0.02",
  "stop_offset": "0.01"
}
```

**Why Mean Reversion Entry first**: It consumes a single decision (rsi_oversold), has the simplest possible resolution logic (single-decision threshold to directional intent), and proves the entire strategy pipeline end-to-end with minimal risk. This mirrors how rsi_oversold was the first decision family.

---

### STF-02: MACD Momentum Entry — Deferred (S55+)

| Field | Value |
|---|---|
| **Canonical name** | `macd_momentum_entry` |
| **Bounded context** | Strategy |
| **Classification** | Resolved |
| **Input decisions** | MACD Crossover (DF-02) — requires macd_crossover decision implementation first |
| **Resolution logic** | When `macd_crossover` triggers bullish, resolve to LONG; when bearish, resolve to SHORT |
| **Phase** | Deferred — S55+ |
| **Readiness** | Blocked on DF-02 (macd_crossover decision) implementation |

**Note**: MACD Momentum Entry is the first strategy that will produce both LONG and SHORT directions, making it a good second family to validate the full direction spectrum.

---

### STF-03: Confluence Entry — Deferred (S56+)

| Field | Value |
|---|---|
| **Canonical name** | `confluence_entry` |
| **Bounded context** | Strategy |
| **Classification** | Resolved (multi-decision) |
| **Input decisions** | RSI Oversold (DF-01) + MACD Crossover (DF-02) |
| **Resolution logic** | Both decisions must agree on direction within a configurable time window |
| **Phase** | Deferred — S56+ |
| **Readiness** | Requires STF-01 + STF-02 proven, plus multi-decision resolution pattern design |

**Why deferred**: Multi-decision confluence introduces temporal alignment questions (how close in time must the two decisions be?) that require careful design. Single-decision families must prove the pipeline first.

---

## 3. Stream Definition

```
Stream:          STRATEGY_EVENTS
Subjects:        strategy.events.>
Retention:       72h, file-backed
Max bytes:       2 GB
Deduplication:   MsgId-based (Strategy.DeduplicationKey())
Discard policy:  Old
```

Subject pattern:
```
strategy.events.{type}.resolved.{source}.{symbol}.{timeframe}
```

Examples:
```
strategy.events.mean_reversion_entry.resolved.binancef.btcusdt.60
strategy.events.macd_momentum_entry.resolved.binancef.ethusdt.300
```

---

## 4. KV Bucket Naming

| Family | Latest Bucket | History Bucket (deferred) |
|---|---|---|
| mean_reversion_entry | `STRATEGY_MEAN_REVERSION_ENTRY_LATEST` | `STRATEGY_MEAN_REVERSION_ENTRY_HISTORY` |
| macd_momentum_entry | `STRATEGY_MACD_MOMENTUM_ENTRY_LATEST` | `STRATEGY_MACD_MOMENTUM_ENTRY_HISTORY` |

Key format: `{source}.{symbol}.{timeframe}`
History key format (deferred): `{source}.{symbol}.{timeframe}.{timestamp_unix}`

---

## 5. Envelope Types

| Plane | Envelope |
|---|---|
| Events | `strategy.events.v1.{type}_resolved` |
| Query request | `strategy.query.v1.{type}_latest_request` |
| Query reply | `strategy.query.v1.{type}_latest_reply` |

---

## 6. Durable Consumers

| Durable Name | Binary | Filter Subject | Purpose |
|---|---|---|---|
| `store-strategy-mean-reversion-entry` | store | `strategy.events.mean_reversion_entry.resolved.>` | Materialize mean reversion entry projections |

Each strategy family gets its own durable consumer in store, following the per-family consumer pattern established by evidence and signal.

---

## 7. Family Invariants

| ID | Invariant |
|---|---|
| **FI-1** | One stream for all strategy families: `STRATEGY_EVENTS` |
| **FI-2** | One publisher actor shared across strategy families within a scope |
| **FI-3** | One consumer + projection actor pair per strategy family in store |
| **FI-4** | Each family has independent KV buckets — no shared state |
| **FI-5** | Family names are lowercase, singular, underscore-separated |
| **FI-6** | Each family must have a documented resolver contract before implementation |
| **FI-7** | Single-decision families before multi-decision families |
| **FI-8** | No family may consume signals directly — only decisions |
| **FI-9** | Every strategy output must include a valid Direction (long/short/flat) |

---

## 8. Growth Pattern

Adding a new strategy family requires:

1. **Design**: Document resolver contract, input decisions, direction semantics, parameters
2. **Domain**: Add resolver in `internal/application/strategy/`
3. **Actor**: Add `StrategyResolverActor` variant in derive scope
4. **Config**: Add family name to `pipeline.strategy_families` schema
5. **Store**: Add consumer + projection actor pair, KV bucket(s)
6. **Query**: Add route in QueryResponderActor
7. **Gateway**: Add HTTP endpoint under `/strategy/{type}/`
8. **Governance**: Update raccoon-cli drift rules
9. **Test**: Unit test resolver (pure), integration test pipeline

This checklist mirrors the decision, signal, and evidence family growth patterns.

---

## References

- [strategy-domain-design.md](strategy-domain-design.md) — Domain design
- [decision-stream-families.md](decision-stream-families.md) — Decision family precedent
- [signal-stream-families.md](signal-stream-families.md) — Signal family precedent
- [stream-family-catalog.md](stream-family-catalog.md) — Mesh-wide catalog
