# Interleaved Execution Observability Minimum

## Context

S281 established that observability would be delivered as an interleaved concern within existing slices, not as a dedicated wave. After S291 closed the squeeze breakout vertical slice with operational proof, S292 adds the minimum useful observability layer.

## Design Principles

1. **No new infrastructure** — use existing `healthz.Tracker` counters exposed via `/statusz` and `/diagz`
2. **Publisher-level instrumentation** — counters live where the tracker already exists (publisher actors), avoiding config plumbing changes
3. **Domain-aware counter names** — `{layer}:{type}:{outcome}` convention enables filtering and correlation
4. **Zero dependencies added** — no Prometheus, no new packages, no external systems

## Architecture

### Instrumentation Point

All counters are recorded at the **publisher actor level** in the derive binary. Each domain publisher (signal, decision, strategy, risk, execution) already holds a `*healthz.Tracker` reference. On successful publish, the actor increments domain-specific counters alongside the existing `published:<symbol>` counter.

```
Trade → Candle → Signal → Decision → Strategy → Risk → Execution
                   ↓          ↓           ↓        ↓        ↓
               SignalPub  DecisionPub  StratPub  RiskPub  ExecPub
                   ↓          ↓           ↓        ↓        ↓
               Tracker    Tracker      Tracker  Tracker  Tracker
                   ↓          ↓           ↓        ↓        ↓
                            /statusz (JSON)
```

### Counter Naming Convention

```
{domain}:{family_type}:{outcome_or_variant}
```

Examples:
- `signal:bollinger` — bollinger signal published
- `decision:bollinger_squeeze:triggered` — squeeze detected
- `strategy:squeeze_breakout_entry:long` — long entry resolved
- `risk:position_exposure:approved` — risk approved
- `execution:paper_order:buy` — buy intent submitted
- `execution:paper_order:filled` — fill status recorded

### Exposure

Counters appear in:
- `GET /statusz` — per-tracker counter map in JSON
- `GET /diagz` — diagnostic summary with counter snapshot
- Heartbeat idle warnings — counters logged alongside idle detection

### Correlation Across Layers

Operators can correlate the squeeze path by comparing counter ratios:
- `signal:bollinger` → total bollinger signals
- `decision:bollinger_squeeze:triggered` / `signal:bollinger` → squeeze detection rate
- `strategy:squeeze_breakout_entry:long` / `decision:bollinger_squeeze:triggered` → entry conversion rate
- `risk:*:approved` + `risk:*:modified` / total risk → risk pass rate
- `execution:paper_order:buy` / `strategy:squeeze_breakout_entry:long` → execution conversion rate

## Scope and Limits

### What This Delivers
- Per-type event counts for the full squeeze breakout slice
- Outcome distribution visibility (triggered vs not, long vs flat, approved vs rejected)
- Execution gate halt counting
- Queryable via existing HTTP health endpoints

### What This Does NOT Deliver
- Latency histograms or percentile tracking
- Per-symbol breakdown beyond the existing `published:<symbol>` counters
- Time-series storage or graphing
- Alerting thresholds or dashboards
- Cross-binary correlation (derive ↔ writer ↔ store)

### Why These Limits Are Acceptable
The goal is operational awareness, not production monitoring. The counters answer: "Is the squeeze path flowing? At what rate? Where is it dropping off?" For deeper investigation, operators use structured logs (which already carry correlation_id and causation_id) and NATS stream inspection.
