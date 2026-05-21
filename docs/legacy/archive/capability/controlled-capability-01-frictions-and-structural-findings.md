# Controlled Capability 01 — Frictions and Structural Findings

> Stage S122 — Capability-driven friction capture from CC-01 live validation.
> Date: 2026-03-19

---

## Classification Legend

| Category | Definition |
|----------|-----------|
| **Bug** | Incorrect behavior that must be fixed — the system does not do what it claims |
| **Operational fragility** | Works today but will break or mislead under plausible conditions |
| **Structural debt** | Architectural misalignment that creates ongoing maintenance cost |
| **Trade-off** | Intentional limitation accepted for the current stage |

| Priority | Criteria |
|----------|---------|
| **P0** | Blocks reliable multi-symbol operation or masks real failures |
| **P1** | High friction; affects every debugging or scaling scenario |
| **P2** | Moderate friction; affects specific operational workflows |
| **P3** | Minor friction; acceptable in current scope |

---

## CF-01: No Per-Symbol Tracker Breakdown in /statusz (Operational Fragility)

**Priority:** P1
**Source:** Predicted in S119 (CP-3, L2), confirmed in S121 Phase 8.

**Evidence:** The `healthz.Tracker` struct (`internal/shared/healthz/healthz.go:14-24`) counts events with a single `eventCount` atomic counter per tracker name. Projection actors (e.g., `candle_projection_actor.go`) record `tracker.RecordEvent()` without distinguishing symbol. Custom counters like `processed`, `filled`, `skipped_stale` are also aggregate.

Under multi-symbol operation, `/statusz` shows a single `event_count` value per tracker (e.g., `"store-candle": {"event_count": 847}`). There is no way to determine how many events belong to btcusdt vs ethusdt without scanning logs.

**Impact:** Diagnosing a per-symbol pipeline stall requires cross-referencing tracker counts with log timestamps. With 2 symbols this is tedious; with N symbols it becomes unworkable. The operator cannot answer "is ethusdt flowing?" from `/statusz` alone.

**Recommendation:** Add per-symbol counter keys (e.g., `tracker.Counter("processed:btcusdt").Add(1)`) to projection and publisher actors. The Tracker already supports arbitrary `Counter(name)` — no infrastructure change needed, only actor-level instrumentation.

---

## CF-02: No Endpoint to List Active Symbols (Operational Fragility)

**Priority:** P2
**Source:** Predicted in S119 (GW-1), confirmed in S121 Phase 6.

**Evidence:** The gateway routes (`internal/interfaces/http/routes/`) expose per-symbol point-lookup endpoints (e.g., `/evidence/candles/latest?symbol=btcusdt`) but no endpoint to discover which symbols are currently active. The configctl endpoint `GET /configctl/configs/active` returns the full config document including binding definitions, but extracting active symbols requires parsing the binding list from the response body.

**Impact:** An operator running multi-symbol monitoring must know in advance which symbols are configured. There is no discovery mechanism — the operator cannot ask "what symbols are running?" through the query surface.

**Recommendation:** Add a `GET /configctl/active-symbols` endpoint (or extend `/diagz` to include active symbol list). This is a thin query over the already-available active config — not a new domain concept.

---

## CF-03: Correlation ID Propagation Is Manual, Not Framework-Enforced (Structural Debt)

**Priority:** P1
**Source:** Predicted as primary friction in S118 (F1), S119 (DB-1), confirmed in S121 (L4).

**Evidence:** Correlation IDs exist in the `Envelope` struct (`internal/shared/envelope/envelope.go:29`) and in event `Metadata` (`internal/shared/events/event.go:24`). Each actor manually copies the correlation ID from incoming messages to outgoing events:
- `execution_evaluator_actor.go:80,100` — copies to intent and event
- `risk_evaluator_actor.go:81` — copies via `WithCorrelationID()`
- `venue_adapter_actor.go:127,175,221` — logs correlation_id

There is no middleware, interceptor, or context-based injection that ensures propagation. If a new actor or publisher omits the manual copy, the correlation chain silently breaks.

**Impact:** Cross-runtime event tracing under multi-symbol operation requires manual timestamp matching across 6 services × 2 symbols. This was the highest-likelihood friction predicted by S119 and is confirmed by operational experience: answering "what happened to this specific ethusdt trade at 14:32:07?" requires grepping multiple container logs by timestamp.

**Recommendation:** Evaluate a context-based or envelope-middleware pattern that injects correlation ID automatically when publishing. The fix is not urgent (current actors are consistent), but the fragility grows with each new actor addition.

---

## CF-04: No Automated Error-Level Log Scanning (Operational Fragility)

**Priority:** P2
**Source:** Identified in S121 (L1), confirmed by script inspection.

**Evidence:** The `live-pipeline-activate.sh` script validates health, readiness, diagnostics, and tracker activity, but does not grep container logs for `level=error` entries. Errors in domain logic are only discovered through manual `make logs SERVICE=<name>` inspection.

**Impact:** A domain-level error (e.g., malformed event, KV write failure) during sustained multi-symbol operation goes undetected until manual log review. The validation script may report "all healthy" while errors accumulate.

**Recommendation:** Add a `docker compose logs --no-log-prefix | grep -c '"level":"error"'` check to Phase 8 of the activation script. Low effort, high signal.

---

## CF-05: No Automated Memory Regression Tracking (Operational Fragility)

**Priority:** P2
**Source:** Identified in S121 (L2), confirmed by script inspection.

**Evidence:** Memory linearity validation (`docker stats` at t=10 and t=30) is documented in the procedure but not automated. There is no baseline snapshot, no comparison, and no alert threshold.

**Impact:** A goroutine leak or event buffer accumulation under multi-symbol load would go undetected until container OOM or operator intervention. With doubled throughput from 2 symbols, memory issues are more likely to emerge.

**Recommendation:** Add `docker stats --no-stream --format '{{.Name}}\t{{.MemUsage}}'` snapshots to Phase 8. Optionally, add a simple watchdog that compares t=10 and t=30 snapshots for >50% growth.

---

## CF-06: No Sustained Automated Validation (30-min Watchdog) (Trade-off)

**Priority:** P3
**Source:** Identified in S121 (L6).

**Evidence:** The S121 procedure defines a 30-minute sustained monitoring session (Phase 2), but execution is manual: the operator runs `make live-multi-check` at intervals and visually inspects tracker counts.

**Impact:** Continuous stability proof requires operator attention. For 2 symbols and 30 minutes, this is manageable. For longer soak periods or N>2 symbols, it becomes a bottleneck.

**Recommendation:** Consider a simple `watch -n 300 make live-multi-check` wrapper or a dedicated watchdog script. Not urgent — the current procedure is sufficient for CC-01 scope.

---

## CF-07: Kill Switch Is Global, Not Per-Symbol (Trade-off)

**Priority:** P3
**Source:** Documented in S120 (G5), confirmed by code inspection.

**Evidence:** The execution control gate (`internal/application/execution/safety_gate.go:52-59`) checks `GateChecker.IsHalted()` globally. Activating the kill switch halts paper order processing for all symbols simultaneously.

**Impact:** An operator who wants to halt execution for one symbol (e.g., ethusdt during a flash crash) while keeping the other running cannot do so. Both symbols stop or both continue.

**Recommendation:** Accept as trade-off for CC-01. Per-symbol control is a separate capability (requires per-symbol KV state in execution control store). Only revisit if operational need is demonstrated.

---

## CF-08: Client UseCase Boilerplate Partially Deduplicated (Structural Debt)

**Priority:** P2
**Source:** Identified in S110 (F04), partially addressed.

**Evidence:** The `internal/shared/usecase/usecase.go` provides generic `CommandUseCase` and `GatewayUseCase` types. The `configctlclient` package has migrated to type aliases:
```go
type GetConfigUseCase = usecase.CommandUseCase[contracts.GetConfigQuery, contracts.GetConfigReply]
```

However, the domain client packages (`decisionclient`, `signalclient`, `riskclient`, `strategyclient`, `evidenceclient`, `executionclient`) still hand-write the identical struct+Execute pattern (~30 LOC each). Approximately 6 packages × 1-3 files = ~180 lines of pure boilerplate remain.

**Impact:** Adding a new domain client operation still requires copy-pasting a file. Changing the nil-safety or validation pattern requires editing each file individually.

**Recommendation:** Migrate remaining domain clients to type aliases using the shared usecase types. This is a mechanical change — no logic or behavior modification. Best done opportunistically when touching these packages, not as a dedicated refactor.

---

## CF-09: RSI Warm-Up Delays Full-Chain Validation (Trade-off)

**Priority:** P3
**Source:** Known by design (DT-3), confirmed in S121 (L3).

**Evidence:** RSI computation requires 14 candles (~14-15 minutes at 60s timeframe) before producing non-null values. During warm-up, Signal, Decision, Strategy, Risk, and Execution endpoints for a newly-added symbol return null/empty responses.

**Impact:** A newly-activated symbol cannot be fully validated for ~15 minutes. The smoke test must be run after warm-up completes, which the procedure documents but does not enforce programmatically.

**Recommendation:** Accept as inherent to RSI indicator design. Document in operational runbook. The smoke script already handles this with appropriate wait comments.

---

## CF-10: 300s Timeframe Requires Extended Wait (Trade-off)

**Priority:** P3
**Source:** Known (L5 from S121).

**Evidence:** The 300-second (5-minute) timeframe candles require at least one complete window before materialization. The smoke test includes 300s checks that may soft-fail if run within 5 minutes of activation.

**Impact:** Minimal — the 60s timeframe provides sufficient validation signal. The 300s timeframe is a convenience for longer-horizon analysis.

**Recommendation:** Accept. The smoke script handles this by running after sufficient elapsed time.

---

## Items That Did NOT Confirm as Problems

These pressure points were predicted in S119 but did **not** materialize as friction during CC-01 live validation:

### NF-01: Cross-Symbol State Contamination (DT-1)

**Predicted:** Actor state isolation might fail if KV key construction has a bug, causing btcusdt data to appear in ethusdt responses.

**Outcome:** Not confirmed. The smoke test's cross-symbol isolation checks (comparing OHLCV values between symbols) pass consistently. Actor state is correctly keyed by `source.symbol.timeframe`. The composite key design works as intended.

### NF-02: WebSocket Goroutine Leak (WS-1)

**Predicted:** Concurrent WS connection management might leak goroutines if per-binding scoping is incorrect.

**Outcome:** Not confirmed. Both WS connections maintain independent lifecycle. No goroutine growth observed during 30-minute sustained operation.

### NF-03: Actor Mailbox Backpressure Under Doubled Load (DT-2)

**Predicted:** Proto.Actor mailboxes might show increasing idle times under 2× event throughput.

**Outcome:** Not confirmed. Tracker idle times remain well within acceptable bounds. The doubled throughput from 2 symbols is handled without observable backpressure.

### NF-04: KV Write Contention (KV-2)

**Predicted:** Projection actors sharing a single NATS connection might show write contention under doubled load.

**Outcome:** Not confirmed. Store trackers show zero error counts and steady write throughput for both symbols.

### NF-05: Staleness Guard False Positives (EX-1)

**Predicted:** ethusdt events arriving slightly delayed might trigger incorrect staleness rejections.

**Outcome:** Not confirmed. The 120s staleness window is generous enough that normal network jitter does not trigger false rejections.

### NF-06: Projection Actor Inconsistency (F05 from S110)

**Predicted:** Inconsistent `received` counters, `checkStatsInvariant()` presence, and logger initialization across projection actors.

**Outcome:** Resolved. All 7 projection actors now have consistent:
- `received` counter in stats struct
- `checkStatsInvariant()` call on `actor.Stopped`
- Lazy logger initialization in `Receive()` method

### NF-07: Signal Publisher Missing Correlation ID (F06 from S110)

**Predicted:** `signal_publisher_actor.go` was missing `correlation_id` log field.

**Outcome:** Resolved. All 4 publisher actors (`signal`, `decision`, `risk`, `strategy`) now consistently log `correlation_id` on publish errors.

---

## Summary

| ID | Category | Priority | Status | Impact Area |
|----|----------|----------|--------|-------------|
| CF-01 | Operational fragility | P1 | Open | Diagnostics / per-symbol observability |
| CF-02 | Operational fragility | P2 | Open | Operator discovery / query surface |
| CF-03 | Structural debt | P1 | Open | Cross-runtime debugging |
| CF-04 | Operational fragility | P2 | Open | Error detection automation |
| CF-05 | Operational fragility | P2 | Open | Memory regression detection |
| CF-06 | Trade-off | P3 | Accepted | Sustained validation automation |
| CF-07 | Trade-off | P3 | Accepted | Per-symbol execution control |
| CF-08 | Structural debt | P2 | Partial | Client usecase boilerplate |
| CF-09 | Trade-off | P3 | Accepted | RSI warm-up inherent |
| CF-10 | Trade-off | P3 | Accepted | 300s timeframe inherent |
