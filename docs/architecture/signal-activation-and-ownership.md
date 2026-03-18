# Signal Activation and Ownership

> Canonical activation and ownership map for the signal domain in Market Foundry.
> Produced in Stage S35. This is a design document — no implementation is included.
> Companion to: `signal-domain-design.md`, `signal-stream-families.md`, `config-driven-activation-hardening.md`.

---

## 1. How Signal Enters by Activation Model

Signal follows the same two-layer activation model established in S34 (config-driven activation hardening):

| Layer | What It Controls for Signal | Where | Mechanism | Runtime Dynamic? |
|-------|----------------------------|-------|-----------|-----------------|
| **Family activation** | Whether signal families (macd, rsi) are instantiated | derive, store | `pipeline.signal_families` in JSONC config | No (requires restart) |
| **Binding activation** | Which source/symbol pairs get signal samplers | derive | BindingWatcherActor (existing) | Yes (live via configctl events) |

### Activation Flow

```
1. deploy/configs/derive.jsonc includes:
     "pipeline": {
       "families": ["candle", "tradeburst", "volume"],
       "signal_families": ["macd", "rsi"]       ← NEW key
     }

2. DeriveSupervisor.start() reads signal_families
   → registers SignalFamilyProcessor entries for each enabled family.

3. BindingWatcherActor activates source/symbol pair (e.g., binancef/btcusdt).
   → DeriveSupervisor ensures SourceScopeActor[binancef] exists.

4. SourceScopeActor spawns for the new symbol:
   a. Evidence sampler actors (candle, tradeburst, volume) — existing flow.
   b. SignalSamplerActor per (signal_family × timeframe) — NEW flow.
   c. SignalPublisherActor (one per SourceScopeActor) — if not yet spawned.

5. Signal samplers receive finalized evidence via local actor messages.
   → Produce signal events via SignalPublisherActor → SIGNAL_EVENTS stream.
```

**Key rule**: Signal samplers are only spawned when **both** conditions are met:
- The signal family is listed in `pipeline.signal_families`.
- A binding is active for the source/symbol pair.

If either condition is missing, no signal sampler exists for that family/pair. This is identical to how evidence families activate today.

---

## 2. Activation Preconditions

These preconditions must be satisfied before signal families can activate. All are structural — they must be true at deploy time, not runtime.

| # | Precondition | Owner | Status | Notes |
|---|-------------|-------|--------|-------|
| P-1 | `pipeline.signal_families` key exists in settings schema | shared/settings | Not implemented | New key, separate from `families` |
| P-2 | `SIGNAL_EVENTS` JetStream stream created on startup | derive (registry) | Not implemented | Follows EVIDENCE_EVENTS pattern |
| P-3 | `IsFamilyEnabled` extended for signal families | shared/settings | Not implemented | New method or generalized check |
| P-4 | SignalFamilyProcessor registration in DeriveSupervisor | derive | Not implemented | Mirrors FamilyProcessor pattern |
| P-5 | Signal projection pipelines registered in StoreSupervisor | store | Not implemented | Mirrors ProjectionPipeline pattern |
| P-6 | Signal KV buckets created on store startup | store | Not implemented | `SIGNAL_MACD_LATEST`, `SIGNAL_RSI_LATEST` |
| P-7 | Signal query subjects registered in QueryResponderActor | store | Not implemented | `signal.query.{type}.latest` |
| P-8 | Signal HTTP routes registered in gateway | gateway | Not implemented | `/signal/{type}/latest` |
| P-9 | BindingWatcherActor fully wired in derive | derive | Active (S34) | Already resolved |
| P-10 | Evidence families operational (candle at minimum) | derive + store | Active | Signal consumes candle evidence |

**P-9 was the highest-severity gap identified in the signal-readiness-review (S25). It was resolved in S34.**

---

## 3. Ownership Matrix — Publication

Publication is who **writes** events to streams. Signal follows the single-writer invariant.

| Stream | Writer Binary | Writer Actor | Subject Pattern | Notes |
|--------|--------------|-------------|----------------|-------|
| `SIGNAL_EVENTS` | derive | `SignalPublisherActor` (one per SourceScopeActor) | `signal.events.{type}.{verb}.{source}.{symbol}.{timeframe}` | Separate actor from EvidencePublisherActor |

### Actor Hierarchy for Publication

```
DeriveSupervisor
├── BindingWatcherActor              (existing — activates source/symbol pairs)
├── ConsumerActor                    (existing — consumes observation events)
└── SourceScopeActor[binancef]       (existing — one per source)
    ├── EvidencePublisherActor       (existing — writes to EVIDENCE_EVENTS)
    ├── SignalPublisherActor          (NEW — writes to SIGNAL_EVENTS)
    ├── SamplerActor[btcusdt/60]     (existing — candle evidence)
    ├── TradeBurstSamplerActor[...]  (existing — tradeburst evidence)
    ├── VolumeSamplerActor[...]      (existing — volume evidence)
    ├── SignalSamplerActor[macd/btcusdt/300]  (NEW — signal family)
    └── SignalSamplerActor[rsi/btcusdt/300]   (NEW — signal family)
```

**Ownership rules for publication:**
- `SignalPublisherActor` is a **separate actor** from `EvidencePublisherActor`. They share the same parent (SourceScopeActor) but own separate NATS connections to separate streams.
- Each `SignalSamplerActor` sends a `publishSignalMessage` to `SignalPublisherActor`. It never writes to NATS directly.
- Only derive publishes to `SIGNAL_EVENTS`. No other binary may write signal events.

---

## 4. Ownership Matrix — Projection

Projection is who **materializes** events into read models.

| KV Bucket | Writer Binary | Writer Actor | Key Format | MaxBytes |
|-----------|--------------|-------------|-----------|----------|
| `SIGNAL_MACD_LATEST` | store | `SignalProjectionActor[macd]` | `{source}.{symbol}.{timeframe}` | 64 MB |
| `SIGNAL_RSI_LATEST` | store | `SignalProjectionActor[rsi]` | `{source}.{symbol}.{timeframe}` | 64 MB |

### Actor Hierarchy for Projection

```
StoreSupervisor
├── (existing evidence consumer/projection pairs)
├── SignalConsumerActor[macd]
│   ← consumes: SIGNAL_EVENTS
│   │   Durable: store-signal-macd
│   │   Filter: signal.events.macd.generated.>
│   Forwards: signalReceivedMessage to SignalProjectionActor[macd]
│
├── SignalProjectionActor[macd]
│   ○ owns: NATS KV connection (write path)
│   │   KV Bucket: SIGNAL_MACD_LATEST
│   Materializes: latest MACD signal per source/symbol/timeframe
│
├── SignalConsumerActor[rsi]
│   ← consumes: SIGNAL_EVENTS
│   │   Durable: store-signal-rsi
│   │   Filter: signal.events.rsi.generated.>
│   Forwards: signalReceivedMessage to SignalProjectionActor[rsi]
│
├── SignalProjectionActor[rsi]
│   ○ owns: NATS KV connection (write path)
│   │   KV Bucket: SIGNAL_RSI_LATEST
│   Materializes: latest RSI signal per source/symbol/timeframe
│
└── QueryResponderActor
    ⇄ serves: evidence.query.* (existing)
    ⇄ serves: signal.query.macd.latest   (NEW)
    ⇄ serves: signal.query.rsi.latest    (NEW)
```

**Ownership rules for projection:**
- Each signal family gets its own consumer + projection actor pair, following the evidence ProjectionPipeline pattern exactly.
- Projection actors apply three gates: Final gate, Validate gate, Monotonicity guard — same as evidence.
- Only store writes to `SIGNAL_*_LATEST` buckets. No other binary or actor may write.
- Signal projection pipelines are registered in `pipeline.signal_families` in `store.jsonc`, separate from evidence families.

---

## 5. Ownership Matrix — Query

Query is who **serves** read requests.

| NATS Subject | Server Binary | Server Actor | Queue Group | Client Binary |
|-------------|--------------|-------------|-------------|---------------|
| `signal.query.macd.latest` | store | QueryResponderActor | `signal.query` | gateway |
| `signal.query.rsi.latest` | store | QueryResponderActor | `signal.query` | gateway |

### Gateway Surface

| HTTP Route | Method | NATS Subject | Handler |
|-----------|--------|-------------|---------|
| `/signal/macd/latest` | GET | `signal.query.macd.latest` | `SignalHandler.GetLatest` |
| `/signal/rsi/latest` | GET | `signal.query.rsi.latest` | `SignalHandler.GetLatest` |

Query params: `source`, `symbol`, `timeframe` — same structure as evidence queries.

**Ownership rules for query:**
- Store is the **sole server** for all `signal.query.*` subjects.
- Gateway is a **stateless translator** — HTTP to NATS request/reply. No domain logic, no KV access.
- QueryResponderActor in store is extended (not replaced) to serve signal queries alongside evidence queries.
- Signal and evidence queries use **separate queue groups** (`signal.query` vs `evidence.query`) for independent scaling.

---

## 6. Derive ↔ Store ↔ Gateway Relationship

The relationship for signal is identical to evidence — a strict unidirectional pipeline:

```
derive (produces)  →  SIGNAL_EVENTS  →  store (projects + serves)  →  gateway (translates)
```

| Binary | Role in Signal Pipeline | Owns |
|--------|------------------------|------|
| **derive** | Produces signal events from evidence | `SignalPublisherActor`, `SignalSamplerActor[]`, writes to `SIGNAL_EVENTS` |
| **store** | Materializes signal projections, serves signal queries | `SignalConsumerActor[]`, `SignalProjectionActor[]`, `SIGNAL_*_LATEST` buckets, `signal.query.*` subjects |
| **gateway** | Translates HTTP to NATS for signal queries | `/signal/{type}/latest` routes, no state |

### What Each Binary Does NOT Do

| Binary | Signal Boundary |
|--------|----------------|
| **derive** | Does not read from `SIGNAL_EVENTS`. Does not serve queries. Does not access KV. |
| **store** | Does not produce signal events. Does not access `SIGNAL_EVENTS` for writes. Does not hold sampler logic. |
| **gateway** | Does not access `SIGNAL_EVENTS`. Does not access KV. Does not hold domain logic. |
| **ingest** | Has no signal involvement. Signal never touches observation events. |

---

## 7. Signal ↔ Evidence Internal Wiring in Derive

Signal samplers in derive consume evidence via **local actor messages**, not by subscribing to `EVIDENCE_EVENTS` JetStream.

```
CandleSamplerActor[btcusdt/300]
  │
  ├── publishCandleMessage → EvidencePublisherActor → EVIDENCE_EVENTS  (existing flow)
  │
  └── notifySignalSamplersMessage → SourceScopeActor                   (NEW fan-out)
        ├── → SignalSamplerActor[macd/btcusdt/300]
        └── → SignalSamplerActor[rsi/btcusdt/300]
```

**Why local messages, not JetStream:**
- The candle that triggers signal sampling was just produced by the same binary. Re-consuming from JetStream adds latency for data already in memory.
- Avoids a second consumer group within derive for evidence it just wrote.
- If future multi-evidence signals need cross-scope data, JetStream consumption can be introduced. The design supports both.

**Ownership of the fan-out:**
- `SourceScopeActor` owns the routing table — it knows which signal samplers depend on which evidence types.
- Signal samplers declare their evidence dependency at registration time (e.g., "I need candle events").
- The fan-out is static per binding activation — it does not change at runtime.

---

## 8. Configuration Surface

### derive.jsonc (extended)

```jsonc
{
  "pipeline": {
    "timeframes": [60, 300],
    "families": ["candle", "tradeburst", "volume"],
    "signal_families": ["macd", "rsi"]                  // NEW — separate key
  }
}
```

### store.jsonc (extended)

```jsonc
{
  "pipeline": {
    "families": ["candle", "tradeburst", "volume"],
    "signal_families": ["macd", "rsi"]                  // NEW — mirrors derive
  }
}
```

### Activation Rules

| Condition | Result |
|-----------|--------|
| `signal_families` absent or `[]` | No signal families activated (safe default) |
| `signal_families: ["macd"]` in derive only | MACD samplers run but store does not project — events accumulate in stream |
| `signal_families: ["macd"]` in store only | Store creates consumer but no events arrive — consumer idles |
| `signal_families: ["macd"]` in both | Full pipeline: derive produces, store projects and serves |
| `signal_families: ["unknown"]` | Binary fails to start with clear error |

**Key difference from evidence `families`:** The default for `signal_families` absent is **no activation** (not all-families). This is intentional — signal is opt-in, not backward-compatible.

---

## 9. Consumer Cursor Summary (Signal)

| Durable Name | Binary | Stream | Filter Subject | Deliver Policy | Purpose |
|-------------|--------|--------|----------------|----------------|---------|
| `store-signal-macd` | store | SIGNAL_EVENTS | `signal.events.macd.generated.>` | DeliverAll | MACD projection |
| `store-signal-rsi` | store | SIGNAL_EVENTS | `signal.events.rsi.generated.>` | DeliverAll | RSI projection |

All consumers use AckWait=30s, MaxDeliver=5 — same as evidence consumers.

---

## 10. What Is Explicitly Out of Scope for S35

S35 is design-only. The following are explicitly deferred:

| Topic | Target Stage | Reason |
|-------|-------------|--------|
| Implementation of `internal/domain/signal/` | S36 | S35 produces design; S36 implements |
| Implementation of SignalPublisherActor | S36 | Actor code is S36 scope |
| Implementation of SignalSamplerActor (MACD/RSI) | S36 | Sampler logic is S36 scope |
| Implementation of SignalConsumerActor/ProjectionActor | S36 | Store-side actors are S36 scope |
| Signal HTTP routes in gateway | S36 | Routes are S36 scope |
| Raccoon-CLI signal drift rules | S36 | Enters alongside signal implementation |
| `pipeline.signal_families` in settings schema | S36 | Schema change is S36 scope |
| `SIGNAL_EVENTS` stream definition in registry | S36 | Infrastructure setup is S36 scope |
| Signal history projections | S37+ | Latest-only first, same sequencing as evidence |
| Multi-evidence signals (candle + volume) | S37+ | Requires validated multi-source fan-out |
| Signal-to-signal composition | S38+ | Flat dependency graph in Phase 1 |
| Decision domain design | S38+ | Requires operational signal layer |
| Separate signal binary | Indefinite | No architectural benefit over derive with separate publisher/stream |

---

## 11. Dependency Map for S36/S37

### S36 Depends On (from this document)

| Dependency | Source |
|-----------|--------|
| `pipeline.signal_families` config key and schema | Section 8 |
| SignalFamilyProcessor registration pattern | Section 3 |
| SignalPublisherActor as separate actor in SourceScopeActor | Section 3 |
| Local fan-out from evidence samplers to signal samplers | Section 7 |
| ProjectionPipeline entries for signal in store | Section 4 |
| Signal query subjects in QueryResponderActor | Section 5 |
| Signal HTTP routes under `/signal/` | Section 5 |
| `SIGNAL_EVENTS` stream with retention/dedup per signal-domain-design.md Section 6 | Section 9 |

### S37 Depends On (deferred decisions)

| Dependency | Source |
|-----------|--------|
| Multi-evidence fan-out validation (candle + volume → signal) | signal-domain-design.md Section 4.4 |
| Signal history projection pattern (if needed) | signal-domain-design.md Section 9 |
| Per-type domain structs (if Metadata proves insufficient) | signal-domain-design.md Section 5.2 |
| Signal expiration lifecycle | signal-domain-design.md Section 10 |
