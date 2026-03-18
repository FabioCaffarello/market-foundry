# Strategy Risks and Blockers

> Detailed analysis of gaps preventing safe strategy entry.
> Date: 2025-03-17 | Stage: S49

## Blocking Gaps (Must resolve before strategy)

### BG-1: Evidence Adapter Test Gap

**Severity**: CRITICAL
**Affected files**:
- `internal/adapters/nats/evidence_publisher.go` — 0 tests
- `internal/adapters/nats/evidence_consumer.go` — 0 tests
- `internal/adapters/nats/evidence_gateway.go` — 0 tests
- `internal/adapters/nats/trade_burst_consumer.go` — 0 tests
- `internal/adapters/nats/volume_consumer.go` — 0 tests

**Risk**: Serialization bugs, stream creation failures, or consumer decode errors would be silent. Strategy depends on evidence flowing correctly through these untested paths.

**Why this matters for strategy**: Strategy → decision → signal → **evidence**. If evidence adapters have a latent bug (e.g., incorrect dedup key, malformed JSON, wrong subject), signal and decision layers receive corrupted data. Strategy would inherit this corruption silently.

**Remediation**: Unit tests covering encode/decode/publish/consume paths. Estimated effort: 1 stage.

---

### BG-2: Observation/Ingest Pipeline Test Gap

**Severity**: CRITICAL
**Affected files**:
- `internal/adapters/nats/observation_publisher.go` — 0 tests
- `internal/adapters/nats/observation_consumer.go` — 0 tests
- `internal/actors/scopes/ingest/*` — 0 tests (5 actor files)
- `internal/adapters/exchanges/binancef/websocket.go` — 0 tests

**Risk**: The entire data ingestion pipeline — from WebSocket connection through trade normalization to OBSERVATION_EVENTS publication — has zero automated validation. If the binancef adapter silently corrupts trade data (price, quantity, timestamp), all downstream domains inherit the error.

**Why this matters for strategy**: Observation is the root of the data pipeline. Every decision and every future strategy evaluation traces back to observation trade events. An unverified root means an unverified system.

**Remediation**: Unit tests for observation adapters + integration test for binancef normalization. Estimated effort: 0.5-1 stage.

---

### BG-3: Evidence Projection Actor Test Gap

**Severity**: HIGH
**Affected files**:
- `internal/actors/scopes/store/candle_projection_actor.go` — 0 tests
- `internal/actors/scopes/store/trade_burst_projection_actor.go` — 0 tests
- `internal/actors/scopes/store/volume_projection_actor.go` — 0 tests

**Risk**: Projection gates (final, validate, monotonicity) and stats counters are implemented but never verified by tests. If a gate is accidentally bypassed or a counter drifts, data integrity and observability are silently compromised.

**Why this matters for strategy**: Store is the read-model authority. If evidence projections are unreliable, signal samplers consume corrupted state, decisions evaluate on wrong signals, and strategy operates on false premises.

**Note**: Signal and decision projection actors follow the same pattern and also lack actor-level tests, but their KV stores and domain logic are independently well-tested. Evidence projections have neither actor tests nor comprehensive adapter tests.

**Remediation**: Unit tests with mock KV stores testing all gates + stats invariants. Estimated effort: 1 stage.

---

### BG-4: TradeBurst Domain Validation Tests

**Severity**: MEDIUM
**Affected file**: `internal/domain/evidence/trade_burst.go` — 0 domain tests

**Risk**: Domain validation assumptions untested. Candle has 14 domain tests, Volume has 11, TradeBurst has 0. If validation logic has an edge case (e.g., zero-value burst detection, ordering check), it will not be caught.

**Remediation**: Add domain tests matching candle/volume pattern. Estimated effort: 0.25 stage.

---

### BG-5: Evidence HTTP Handler Tests (TradeBurst/Volume)

**Severity**: MEDIUM
**Affected files**:
- `internal/interfaces/http/handlers/evidence.go` — TradeBurst handler untested
- `internal/interfaces/http/handlers/evidence.go` — Volume handler untested

**Risk**: Query parameter parsing, error handling, and null response behavior are not validated for these two endpoints. Candle endpoints are fully tested (16 cases).

**Remediation**: Add handler tests following candle pattern. Estimated effort: 0.25 stage.

---

### BG-6: Candle Dual-Write Atomicity

**Severity**: MEDIUM
**Affected file**: `internal/actors/scopes/store/candle_projection_actor.go`

**Risk**: Candle projection writes to `CANDLE_LATEST` first, then `CANDLE_HISTORY`. If the history write fails, latest has data that history doesn't. This creates an inconsistency between latest and history views.

**Remediation**: Either (a) document acceptance with rationale (history is supplementary), or (b) reverse write order (history-first). Estimated effort: 0.25 stage.

---

## Non-Blocking Risks (Acceptable for strategy entry)

### NBR-1: Binding Deactivation Requires Restart

**Severity**: LOW
**Impact**: Clearing a binding is logged but actors are not stopped dynamically. Requires service restart.
**Mitigation**: Operational procedure. Not blocking because strategy activation follows the same pattern.

### NBR-2: Single Exchange Adapter

**Severity**: LOW
**Impact**: Only binancef is implemented. Architecture supports multi-source, but proof is missing.
**Mitigation**: Not blocking for strategy. Multi-source is a horizontal scaling concern, not a layering concern.

### NBR-3: No Projection Lag Metrics

**Severity**: LOW
**Impact**: Cannot detect store falling behind derive in event processing.
**Mitigation**: Health trackers provide basic liveness. Lag detection is operational tooling, not a strategy prerequisite.

### NBR-4: QueryResponderActor Not Family-Filtered

**Severity**: LOW
**Impact**: Opens KV stores for all evidence types even if only some are enabled.
**Mitigation**: Minor inefficiency, zero correctness impact. Not blocking.

### NBR-5: No Signal/Decision History Projections

**Severity**: LOW
**Impact**: Signal and decision are latest-only. No historical lookback available.
**Mitigation**: Intentional design. History is added when analytical need justifies it. Strategy can operate on latest values for its first slice.

### NBR-6: Zero Actor-Level Tests (Systemic)

**Severity**: MEDIUM (systemic, not strategy-specific)
**Impact**: No actor-level tests exist anywhere in the codebase (ingest, derive, store). This is a systemic gap affecting all domains equally.
**Mitigation**: BG-3 addresses evidence projection actors specifically. Full actor test coverage is a broader initiative beyond strategy readiness.

---

## Risk-Impact Matrix

| Risk | Probability | Impact | Priority |
|------|------------|--------|----------|
| BG-1: Evidence adapter silent corruption | Medium | Critical (data pipeline) | **P0** |
| BG-2: Observation pipeline silent corruption | Medium | Critical (root data) | **P0** |
| BG-3: Projection gate bypass | Low | High (read model integrity) | **P1** |
| BG-4: TradeBurst validation edge case | Low | Medium (one evidence type) | **P2** |
| BG-5: HTTP handler edge case | Low | Low (query surface only) | **P2** |
| BG-6: Candle dual-write inconsistency | Low | Medium (history accuracy) | **P2** |
| NBR-6: Systemic actor test gap | Medium | Medium (all domains) | **P3** |

---

## Resolution Timeline

| Stage | Scope | Resolves |
|-------|-------|----------|
| S50 | Adapter test coverage sweep (evidence + observation) | BG-1, BG-2, BG-4 |
| S51 | Projection actor tests + HTTP handler tests + dual-write review | BG-3, BG-5, BG-6 |
| S52 | Strategy domain design (readiness review re-run) | P-6, P-7 from prerequisites |

After S51, all blocking gaps are closed and strategy design can begin in S52.
