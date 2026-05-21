# Multi-Binary Handoffs: Subjects, Payloads, Invariants and Risks

> **Stage:** S371 — Binary Boundary and Event-Flow Audit
> **Scope:** Canonical pipeline (`mean_reversion_entry`) across operational binaries
> **Focus:** Handoff points, ordering assumptions, risks, and bridges

---

## 1. Handoff Map

A **handoff** is the point where an event crosses a binary boundary via NATS. The canonical pipeline has the following handoffs:

### H1: configctl → ingest (Configuration Activation)

| Property | Value |
|----------|-------|
| **Stream** | `CONFIGCTL_EVENTS` |
| **Subject** | `configctl.events.config.ingestion_runtime_changed` |
| **Payload** | `IngestionRuntimeChangedEvent` |
| **Publisher** | configctl (`ConfigSupervisor`) |
| **Consumer** | ingest (`ingest-binding-watcher`, durable) |
| **Trigger** | Config activation via `configctl.control.activate_config` |
| **Effect** | Ingest binds/unbinds exchange WebSocket connections |
| **Ordering** | Single partition — events arrive in publish order |
| **Idempotency** | Binding watcher applies desired-state reconciliation |

### H2: configctl → derive (Configuration Activation)

| Property | Value |
|----------|-------|
| **Stream** | `CONFIGCTL_EVENTS` |
| **Subject** | `configctl.events.config.ingestion_runtime_changed` |
| **Payload** | `IngestionRuntimeChangedEvent` |
| **Publisher** | configctl (`ConfigSupervisor`) |
| **Consumer** | derive (`derive-binding-watcher`, durable) |
| **Trigger** | Config activation via `configctl.control.activate_config` |
| **Effect** | Derive creates/destroys SourceScopeActors for sources |
| **Ordering** | Single partition — events arrive in publish order |
| **Idempotency** | Binding watcher applies desired-state reconciliation |

### H3: ingest → derive (Market Data Ingestion)

| Property | Value |
|----------|-------|
| **Stream** | `OBSERVATION_EVENTS` |
| **Subject** | `observation.events.market.trade.{source}` |
| **Payload** | `TradeReceivedEvent { Metadata, ObservationTrade }` |
| **Publisher** | ingest (`PublisherActor` per exchange scope) |
| **Consumer** | derive (`derive-observation`, durable) |
| **Dedup Key** | `obs:trade:{source}:{tradeID}:{timestamp}` |
| **Ordering** | Per-source partition — trades from same source arrive in order |
| **Volume** | High — one event per trade (hundreds/second per active symbol) |
| **Latency Sensitivity** | Medium — windowed sampling tolerates small delays |

**Invariants:**
- H3-INV-1: Every trade has a non-empty Source, Symbol, Price, Quantity, TradeID.
- H3-INV-2: DeduplicationKey is deterministic — replay produces identical MsgID.
- H3-INV-3: Derive routes by source to isolated SourceScopeActors — no cross-source interference.

### H4: derive → store (Domain Event Materialization)

| Property | Value |
|----------|-------|
| **Streams** | `EVIDENCE_EVENTS`, `SIGNAL_EVENTS`, `DECISION_EVENTS`, `STRATEGY_EVENTS`, `RISK_EVENTS`, `EXECUTION_EVENTS` |
| **Subjects** | `{domain}.events.{family}.{verb}.{source}.{symbol}.{timeframe}` |
| **Payloads** | Domain-specific event types (see contract audit) |
| **Publisher** | derive (family-specific PublisherActors) |
| **Consumer** | store (family-specific durable consumers) |
| **Dedup Key** | `{family}:{type}:{source}:{symbol}:{timeframe}:{unix_ts}` |
| **Ordering** | Per-source-symbol-timeframe partition — events arrive in causal order within partition |
| **Volume** | Medium — one event per timeframe tick per family per symbol |

**Invariants:**
- H4-INV-1: Store applies monotonicity guard — rejects events with timestamp <= persisted.
- H4-INV-2: Store applies `Final == true` gate on strategy/evidence projections.
- H4-INV-3: Store applies `Validate()` before KV write — invalid payloads are dropped.
- H4-INV-4: Store projects to domain-specific KV buckets — no cross-domain mixing.

### H5: derive → execute (Strategy-to-Execution Wiring, S360)

| Property | Value |
|----------|-------|
| **Stream** | `STRATEGY_EVENTS` |
| **Subject** | `strategy.events.mean_reversion_entry.resolved.{source}.{symbol}.{timeframe}` |
| **Payload** | `StrategyResolvedEvent { Metadata, Strategy }` |
| **Publisher** | derive (`StrategyPublisherActor`) |
| **Consumer** | execute (`execute-strategy-mean-reversion-entry`, durable) |
| **Effect** | StrategyConsumerActor evaluates strategy, produces ExecutionIntent, forwards to VenueAdapterActor |
| **Ordering** | Per partition — strategies for same source/symbol/timeframe arrive in order |

**Invariants:**
- H5-INV-1: Execute validates strategy before processing — invalid strategies are dropped.
- H5-INV-2: Execute applies staleness guard — events older than `DefaultStalenessMaxAge` (120s) are skipped.
- H5-INV-3: Execute checks activation gate — inactive gate suppresses venue submission.

### H6: derive → execute (Paper Order Intake — Transitional Bridge)

| Property | Value |
|----------|-------|
| **Stream** | `EXECUTION_EVENTS` |
| **Subject** | `execution.events.paper_order.submitted.{source}.{symbol}.{timeframe}` |
| **Payload** | `PaperOrderSubmittedEvent { Metadata, ExecutionIntent }` |
| **Publisher** | derive (`ExecutionPublisherActor`) |
| **Consumer** | execute (`execute-venue-market-order-intake`, durable) |
| **Bridge Status** | **TRANSITIONAL** — paper mode only; will migrate to venue-specific subjects |
| **Effect** | VenueAdapterActor receives intent, submits to venue (paper or real), publishes fill |

**Invariants:**
- H6-INV-1: This consumer subscribes to paper_order subjects — NOT venue-specific subjects.
- H6-INV-2: When venue-specific intent subjects are introduced, this consumer spec migrates.
- H6-INV-3: The consumer receives `PaperOrderSubmittedEvent` regardless of venue type.

### H7: execute → store (Venue Fill Materialization)

| Property | Value |
|----------|-------|
| **Stream** | `EXECUTION_FILL_EVENTS` |
| **Subject** | `execution.fill.venue_market_order.{source}.{symbol}.{timeframe}` |
| **Payload** | `VenueOrderFilledEvent { Metadata, ExecutionIntent, VenueOrderID }` |
| **Publisher** | execute (`VenueAdapterActor`) |
| **Consumer** | store (`store-execution-venue-market-order-fill`, durable) |
| **Effect** | Store materializes fill result to `EXECUTION_VENUE_MARKET_ORDER_LATEST` KV bucket |

### H8: derive → writer (Analytical Persistence)

| Property | Value |
|----------|-------|
| **Streams** | All event streams |
| **Payloads** | All domain event types |
| **Publisher** | derive (all family publishers) |
| **Consumer** | writer (family-specific durable consumers) |
| **Effect** | Maps events to ClickHouse DDL schema, buffers, batch-inserts |
| **Ordering** | Per-consumer — writer processes events independently per family |

**Invariants:**
- H8-INV-1: Writer column count matches DDL exactly (compile-time contract).
- H8-INV-2: Metadata fields (event_id, occurred_at, correlation_id, causation_id) always at positions 0-3.
- H8-INV-3: Buffer eviction is FIFO with `events_dropped` counter — no silent loss.

### H9: execute → writer (Venue Fill Persistence)

| Property | Value |
|----------|-------|
| **Stream** | `EXECUTION_FILL_EVENTS` |
| **Subject** | `execution.fill.venue_market_order.>` |
| **Payload** | `VenueOrderFilledEvent` |
| **Publisher** | execute |
| **Consumer** | writer (`writer-execution-venue-fill`, durable) |
| **Effect** | Venue fills persisted to ClickHouse for analytics |

### H10: store ↔ gateway (Query Path)

| Property | Value |
|----------|-------|
| **Pattern** | NATS request/reply |
| **Subjects** | `{domain}.query.{family}.latest` |
| **Payloads** | Query → Reply (CBOR Envelope) |
| **Responder** | store (QueryResponderActors) |
| **Requester** | gateway (domain Gateways via `NATSRequestClient`) |
| **Timeout** | Configurable per gateway client |

**Invariants:**
- H10-INV-1: Gateway never reads KV directly — always goes through request/reply.
- H10-INV-2: Store query responders read from KV — single source of truth.
- H10-INV-3: Errors returned as `Problem` in Envelope — no raw error strings.

### H11: configctl ↔ gateway (Config Query Path)

| Property | Value |
|----------|-------|
| **Pattern** | NATS request/reply |
| **Subjects** | `configctl.control.*` |
| **Responder** | configctl |
| **Requester** | gateway |

---

## 2. Ordering Assumptions

### Within a Single Binary

- **Derive internal pipeline** (observation → evidence → signal → decision → strategy → risk → execution) is **strictly ordered within a SourceScopeActor** — all processing for a given source happens sequentially in actor message order.
- **No cross-source ordering guarantee** — events for `binancef` and `coinbase` are independent.

### Across Binary Boundaries

| Assumption | Status | Evidence |
|-----------|--------|----------|
| Events in a single stream are ordered by publish time | **Guaranteed** | JetStream provides per-subject ordering |
| Consumer sees events in stream order | **Guaranteed** | Durable consumers with explicit ack deliver in order |
| Derive publishes events in causal order | **Guaranteed** | Actor model ensures sequential processing per scope |
| Store sees strategy before execution for same partition | **NOT guaranteed** | Different streams, different consumers — race possible |
| Execute sees strategy and paper_order for same partition | **Partial** | Same partition key but different consumer groups |

### Ordering Risk: Store Projection Race

Store has independent consumers for each domain stream. For the same `{source}.{symbol}.{timeframe}` partition:
- `store-strategy-*` may project before or after `store-execution-paper-order`.
- This is acceptable because KV projections are independent per bucket — no cross-bucket consistency requirement.

### Ordering Risk: Execute Dual Intake

Execute has two consumers:
1. `execute-strategy-mean-reversion-entry` (from `STRATEGY_EVENTS`)
2. `execute-venue-market-order-intake` (from `EXECUTION_EVENTS`)

Both may produce venue submissions for the same partition. In practice:
- The strategy consumer path (S360 wiring) is the canonical path forward.
- The paper order intake (transitional bridge) is the legacy path.
- **Risk**: Both paths active simultaneously could cause duplicate venue submissions for the same market event.

---

## 3. Subject Naming Convention Audit

### Pattern

```
{domain}.{message_class}.{entity_type}.{verb}[.{source}.{symbol}.{timeframe}]
```

### Verified Consistency

| Domain | Events | Queries | Consistent |
|--------|--------|---------|-----------|
| observation | `observation.events.market.trade.{source}` | — | Yes |
| evidence | `evidence.events.{type}.sampled.>` | `evidence.query.{type}.latest` | Yes |
| signal | `signal.events.{type}.generated.>` | `signal.query.{type}.latest` | Yes |
| decision | `decision.events.{type}.evaluated.>` | `decision.query.{type}.latest` | Yes |
| strategy | `strategy.events.{type}.resolved.>` | `strategy.query.{type}.latest` | Yes |
| risk | `risk.events.{type}.assessed.>` | `risk.query.{type}.latest` | Yes |
| execution | `execution.events.{family}.submitted.>` | `execution.query.{family}.latest` | Yes |
| execution (fill) | `execution.fill.{family}` | `execution.query.{family}.latest` | Yes |
| configctl | `configctl.events.config.{lifecycle}` | `configctl.control.{action}` | Yes |

**Finding**: Subject naming is consistent across all domains. No deviations detected.

---

## 4. Risk Classification

### RISK-1: Transitional Bridge in Execute Binary

| Dimension | Assessment |
|-----------|-----------|
| **What** | `execute-venue-market-order-intake` subscribes to `execution.events.paper_order.submitted.>` — a subject owned by derive's paper family |
| **Why It Exists** | Derive only produces PaperOrderSubmittedEvent; there is no venue-specific intent subject yet |
| **Impact** | Low for multi-binary proof — paper mode is the only mode tested |
| **Migration Path** | When venue-specific intent subjects are introduced, the consumer spec migrates to `execution.events.venue_intent.submitted.>` |
| **Risk Level** | **LOW** — well-documented, scoped to paper mode, no functional impact on proof |

### RISK-2: Dual Intake Paths in Execute

| Dimension | Assessment |
|-----------|-----------|
| **What** | Execute consumes both `STRATEGY_EVENTS` (S360 wiring) and `EXECUTION_EVENTS` (paper bridge) |
| **Why** | S360 added strategy-to-execution wiring; paper bridge remains for backward compatibility |
| **Impact** | Potential duplicate venue submissions if both paths produce intents for the same market event |
| **Mitigation** | Staleness guard (120s) + activation gate + venue adapter idempotency |
| **Risk Level** | **MEDIUM** — needs validation in compose-level proof; could cause unexpected duplicate fills |

### RISK-3: Event Metadata Loss in KV Projections

| Dimension | Assessment |
|-----------|-----------|
| **What** | Store projects domain entities (e.g., `Strategy`) to KV without event-level metadata (correlation_id, causation_id, occurred_at, event_id) |
| **Why** | Design choice — KV stores latest entity state, not event history |
| **Impact** | Gateway queries cannot trace back to originating events via KV |
| **Mitigation** | ClickHouse preserves full metadata; NATS retains raw events 72h; structured logs capture metadata |
| **Risk Level** | **LOW** — acceptable for operational queries; analytics path preserves full lineage |

### RISK-4: Compose Startup Order vs Stream Creation

| Dimension | Assessment |
|-----------|-----------|
| **What** | Store and execute depend on derive being healthy, partly because derive creates/updates streams on startup |
| **Why** | JetStream streams must exist before consumers can bind |
| **Impact** | If derive hasn't created a stream before store/execute attempt to consume, consumer binding fails |
| **Mitigation** | Compose `depends_on: derive: condition: service_healthy` + readiness check ensures derive has started |
| **Risk Level** | **MEDIUM** — stream creation happens in actor `Started()` lifecycle, which is after `/readyz` returns 200. There may be a window where derive is "ready" but streams are not yet created |

### RISK-5: Gateway Readiness Depends on Store

| Dimension | Assessment |
|-----------|-----------|
| **What** | Gateway's readiness check probes store and configctl via HTTP/NATS |
| **Why** | Gateway cannot serve queries without backend availability |
| **Impact** | If store is slow to start, gateway stays unready, blocking external access |
| **Risk Level** | **LOW** — expected behavior; compose `depends_on` handles ordering |

### RISK-6: Writer Buffer Eviction Under Backpressure

| Dimension | Assessment |
|-----------|-----------|
| **What** | Writer's inserter buffer uses FIFO eviction when max-pending is exceeded |
| **Why** | Prevents unbounded memory growth when ClickHouse is slow or unavailable |
| **Impact** | Events may be dropped from analytical path (not from event streams) |
| **Mitigation** | `events_dropped` counter provides observability; NATS streams retain events for re-consumption |
| **Risk Level** | **LOW** — analytical path is best-effort; event streams are the source of truth |

### RISK-7: NATS Reconnection During Binary Restart

| Dimension | Assessment |
|-----------|-----------|
| **What** | When a binary restarts, its NATS connection is re-established and durable consumers resume |
| **Why** | JetStream durable consumers track delivery state server-side |
| **Impact** | Unacknowledged messages are redelivered after AckWait (30s) |
| **Mitigation** | Deduplication via MsgID (JetStream server-side) + idempotent processing in store/writer |
| **Risk Level** | **LOW** — standard JetStream recovery pattern |

---

## 5. Bridges and Transitional Dependencies

### Active Bridges

| Bridge | From | To | Type | Status | Removal Condition |
|--------|------|----|----|--------|-------------------|
| Paper order intake | derive (paper_order publisher) | execute (venue-market-order-intake consumer) | **Transitional** | Active in paper mode | When venue-specific intent subjects are introduced |
| S360 strategy wiring | derive (strategy publisher) | execute (strategy consumer) | **Canonical** | Active (S360+) | Permanent — this is the target architecture |

### Dependency Matrix

| Binary | Hard Dependencies (must be running) | Soft Dependencies (async) |
|--------|--------------------------------------|--------------------------|
| configctl | nats | — |
| ingest | nats, configctl | — |
| derive | nats | configctl (binding watcher is async) |
| store | nats | derive (streams must exist, but no runtime dependency) |
| execute | nats | derive (streams must exist) |
| gateway | nats, configctl, store | — |
| writer | nats, clickhouse | — |

---

## 6. Canonical Pipeline Summary (End-to-End)

```
                          BINARY BOUNDARIES
                          ─────────────────
  ┌─────────┐   NATS    ┌─────────────────────────────────────┐
  │ configctl├──────────►│ ingest (binding watcher)            │
  └────┬─────┘           │   └─ ExchangeScopeActor             │
       │                 │       └─ WebSocketActor              │
       │   NATS          │           └─ PublisherActor          │
       ├────────────┐    └───────────────┬─────────────────────┘
       │            │                    │
       │            │    OBSERVATION_EVENTS (H3)
       │            │                    │
       │            ▼                    ▼
       │    ┌─────────────────────────────────────────────────┐
       │    │ derive                                           │
       │    │   ├─ BindingWatcherActor (H2)                   │
       │    │   └─ SourceScopeActor                           │
       │    │       ├─ ConsumerActor (observation)             │
       │    │       ├─ CandleSamplerActor → evidence          │
       │    │       ├─ RSISignalSamplerActor → signal         │
       │    │       ├─ RSIOversoldEvaluatorActor → decision   │
       │    │       ├─ MeanReversionResolverActor → strategy  │
       │    │       ├─ PositionExposureEvaluatorActor → risk  │
       │    │       └─ PaperOrderEvaluatorActor → execution   │
       │    └──────┬──────────┬──────────┬────────────────────┘
       │           │          │          │
       │   STRATEGY_EVENTS   │   EXECUTION_EVENTS
       │      (H5)           │      (H6)
       │           │         │          │
       │           ▼         │          ▼
       │    ┌──────────────┐ │   ┌──────────────┐
       │    │ execute      │ │   │ execute      │
       │    │ (strategy    │ │   │ (paper order │
       │    │  consumer)   │ │   │  intake)     │
       │    └──────┬───────┘ │   └──────┬───────┘
       │           │         │          │
       │           ▼         │          ▼
       │    ┌──────────────────────────────────┐
       │    │ execute VenueAdapterActor         │
       │    │   └─ VenueOrderFilledEvent        │
       │    └──────┬───────────────────────────┘
       │           │
       │    EXECUTION_FILL_EVENTS (H7)
       │           │
       ▼           ▼
  ┌──────────────────────────────────────────────────────┐
  │ store                                                 │
  │   ├─ ConsumerActors (all domain streams) (H4)        │
  │   ├─ ProjectionActors → KV buckets                   │
  │   └─ QueryResponderActors (H10)                      │
  └──────────────────┬───────────────────────────────────┘
                     │
              NATS Request/Reply (H10)
                     │
                     ▼
  ┌──────────────────────────────────────────────────────┐
  │ gateway                                               │
  │   └─ HTTP handlers → use cases → NATS gateways       │
  └──────────────────────────────────────────────────────┘

  ┌──────────────────────────────────────────────────────┐
  │ writer (parallel path)                                │
  │   ├─ ConsumerActors (all domain streams) (H8, H9)   │
  │   ├─ MapperActors → ClickHouse DDL                   │
  │   └─ InserterActors → ClickHouse batch insert        │
  └──────────────────────────────────────────────────────┘
```

---

## 7. Invariants That Must Hold in Compose-Level Proof

| ID | Invariant | Verification Method |
|----|-----------|-------------------|
| **MBI-1** | All 8 binaries reach `/readyz` in dependency order | Compose healthcheck + sequential startup |
| **MBI-2** | Events flow from ingest → derive → store across process boundaries | End-to-end test: publish trade, verify KV projection |
| **MBI-3** | Correlation chain (5+ hops) is preserved across binaries | Query ClickHouse or store for matching correlation_ids |
| **MBI-4** | KV materialization in store is readable by gateway | HTTP query through gateway returns projected data |
| **MBI-5** | Execute receives strategy events from derive | Execute health tracker shows strategy-consumer events |
| **MBI-6** | Execute produces fills readable by store | Venue fill KV bucket populated after execute processes |
| **MBI-7** | Writer persists all event types to ClickHouse | ClickHouse query returns rows for all 6 families |
| **MBI-8** | Single-binary restart recovers without data loss | Kill + restart binary, verify no event gaps |
| **MBI-9** | Kill-switch propagates from gateway to execute | Set gate inactive via gateway, verify execute stops submitting |
| **MBI-10** | No duplicate venue submissions in steady state | Verify execute produces exactly one fill per strategy event |
