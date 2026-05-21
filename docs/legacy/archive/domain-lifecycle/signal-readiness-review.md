# Signal Readiness Review

> Formal assessment of whether the Foundry is ready for the `signal` domain layer. Evaluates each subsystem against concrete readiness criteria.

## Verdict

**The Foundry is conditionally ready for signal.** The observation → evidence → projection → query pipeline is proven with two evidence types. The structural patterns are sound. However, four specific gaps should be addressed before signal enters.

## Readiness by Subsystem

### 1. Observation Layer — READY

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Trade ingestion works E2E | PASS | Binance WS → ingest → OBSERVATION_EVENTS, multi-symbol proven |
| Observation events are domain-typed | PASS | `ObservationTrade` with `Validate()`, `TradeReceivedEvent` with metadata |
| Dedup at ingestion boundary | PASS | NATS `MsgID` = `{source}:{trade_id}` |
| Durable consumption by derive | PASS | `derive-observation` consumer with explicit ack |
| Multi-source readiness | PARTIAL | Architecture supports it; only binancef implemented |

**Gap:** Only one exchange adapter (binancef). Signal doesn't strictly need multiple sources, but the pattern should be proven before signal assumes diverse observation inputs.

### 2. Evidence Derivation — READY

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Pure sampler logic (no I/O) | PASS | `CandleSampler` and `TradeBurstSampler` are pure application logic with table-driven tests |
| Window-based finalization | PASS | `Final=true` emitted on window boundary, consistent across both samplers |
| Multi-evidence fan-out | PASS | Source scope actor spawns candle + trade burst samplers per symbol, all receive same trades |
| Evidence events published with dedup | PASS | JetStream `MsgID` per evidence type |
| Second evidence type proves pattern | PASS | Trade burst follows identical pipeline structure |

**Gap:** None blocking. The derivation pattern is clean and extensible.

### 3. Projections (Latest/History/Multi-Projection) — READY with caveats

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Latest projection with monotonicity guard | PASS | Both candle and trade burst use `PutResult` with OpenTime comparison |
| History projection with range query | PASS | Candle has `CANDLE_HISTORY` with since/until/limit |
| Multi-projection coexistence | PASS | Two types with independent consumers, projection actors, KV buckets, health trackers |
| Replay safety | PASS | Documented in `replay-idempotency-rules.md`, monotonicity guard prevents regression |
| Domain validation before write | PASS | `Validate()` gate in every projection actor |

**Gap:** Trade burst lacks a history projection. If signal needs historical trade burst data, `TRADE_BURST_HISTORY` must be added first. This is a known intentional limitation (latest-only for proof of pattern).

### 4. Store as Read-Side Authority — READY

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Store owns all read models | PASS | All KV buckets written exclusively by store projection actors |
| Gateway never touches KV directly | PASS | All queries through NATS request/reply |
| Per-projection health tracking | PASS | 4 independent trackers on `/statusz` |
| Query contracts are typed and versioned | PASS | CBOR envelopes with `v1` type strings |
| Projection inventory logged on startup | PASS | Supervisor logs `projections`, `buckets`, consumer names |

**Gap:** None blocking. The store boundary is clean.

### 5. Gateway — READY

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Gateway is a thin translation layer | PASS | HTTP → use case → NATS request/reply. No domain logic. |
| Evidence queries degrade gracefully | PASS | Nil use case = no route. Store unavailable = 503. |
| Common query param parser | PASS | `parseEvidenceKeyParams` shared across all handlers |
| No KV access from gateway | PASS | Enforced by architecture (no KV adapter in gateway binary) |

**Gap:** None blocking.

### 6. Raccoon-CLI Architecture Governance — PARTIAL

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Contract audit detects drift | NEEDS VERIFICATION | Raccoon-CLI exists but hasn't been run against the S23/S24 changes |
| Topology audit covers new actors | NEEDS VERIFICATION | Trade burst actors may not be in the topology manifest |
| Drift detection covers evidence registry | NEEDS VERIFICATION | New registry specs may not be covered by drift rules |

**Gap:** Raccoon-CLI governance coverage for the S23+ evidence types has not been verified. The CLI exists and is documented, but its rules may be stale relative to the current architecture.

### 7. Configuration-Driven Activation — NOT READY

| Criterion | Status | Evidence |
|-----------|--------|----------|
| BindingWatcher drives derive activation | PARTIAL | BindingWatcher exists but is partially stubbed |
| Store responds to binding changes | NOT IMPLEMENTED | Store spawns all projections unconditionally on startup |
| Signal would need activation per binding | BLOCKED | No mechanism to selectively enable signal derivation per symbol |

**Gap:** This is the most significant gap. Signal derivation would logically be activated per symbol/source via configctl bindings. The current system hardcodes all evidence derivations for all symbols. Before signal enters, the activation path should be solid.

## Signal Domain Boundaries (Architectural Preview)

If signal enters, it would follow this model:

```
evidence events (EVIDENCE_EVENTS stream)
  → signal derive service (new binary)
    → signal sampler (pure logic: combines candle + trade burst → signal event)
    → signal events (SIGNAL_EVENTS stream)
      → signal store (or extend current store)
        → signal projection (new KV bucket)
        → signal query (new NATS subjects)
          → gateway HTTP endpoint
```

### Key architectural rules for signal:

1. **Signal is a separate domain** — `internal/domain/signal/` with its own types and validation
2. **Signal consumes evidence, not observations** — signal samplers read from `EVIDENCE_EVENTS`, not `OBSERVATION_EVENTS`
3. **Signal does not modify evidence** — evidence projections are unchanged by signal's existence
4. **Signal may be a separate binary** — `cmd/signal-derive/` to maintain process-level isolation
5. **Signal projections follow the same KV pattern** — latest bucket, optional history, same query model

## Gaps Summary (Blocking Signal)

| # | Gap | Severity | Resolution |
|---|-----|----------|------------|
| 1 | Config-driven activation not complete | HIGH | Wire BindingWatcher for dynamic projection lifecycle |
| 2 | Raccoon-CLI governance coverage stale | MEDIUM | Run contract-audit and topology checks against current state |
| 3 | Trade burst history not implemented | LOW | Add `TRADE_BURST_HISTORY` following candle pattern |
| 4 | Single exchange adapter | LOW | Not blocking for signal, but limits observation diversity |

## Gaps Summary (Non-Blocking)

| # | Gap | Reason not blocking |
|---|-----|---------------------|
| 5 | No ClickHouse | Signal doesn't need it; NATS KV is sufficient for signal projections |
| 6 | No projection lag metric | Nice-to-have for operations, not a signal prerequisite |
| 7 | `EvidenceConsumerActor` legacy naming | Cosmetic; doesn't affect signal architecture |

## Recommendation

**Do not implement signal yet.** Address gaps #1 (config-driven activation) and #2 (raccoon-cli governance) first. These are structural prerequisites — without them, signal would enter the system without proper activation control or architectural governance.

**Recommended pre-signal stages:**
1. Wire config-driven activation in derive + store (1 stage)
2. Verify and update raccoon-cli rules for S23+ architecture (1 stage)
3. Signal can enter in the stage after these two are complete
