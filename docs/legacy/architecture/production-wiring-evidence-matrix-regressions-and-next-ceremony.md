# Production Wiring Evidence Matrix, Regressions, and Next Ceremony

> Companion to the Production Wiring Evidence Gate (S331).
> Contains the objective evidence matrix, regression audit, residual gaps,
> and strategic recommendation for the next ceremony.

## 1. Evidence Matrix

### 1.1. Capability Composition Status

| Capability                         | Isolation Stage | Wired In | Evidence Level | Test Count |
|------------------------------------|-----------------|----------|----------------|------------|
| RetrySubmitter (backoff + jitter)  | S314, S323      | S328     | **FULL**       | 27 unit + 7 SC + 9 VP |
| Global retry deadline              | S323            | S328     | **FULL**       | 8 deadline tests + SC/VP |
| Kill switch halt check             | S323            | S328     | **FULL**       | SC-04, VP-06, 4 halt tests |
| Post200Reconciler                  | S322            | S328     | **FULL**       | 9 reconciliation + SC-01 + VP-02/08 |
| Structured retry observability     | S324            | S328     | **FULL**       | 6 obs tests + SC-07 + VP-03/05 |
| Venue error code classification    | S325            | S325     | **FULL**       | 10 EC tests (22 subtests) |
| Safety gate (staleness + kill)     | S308            | Existing | **FULL**       | VP-09 |
| Decorator composition order        | —               | S328     | **FULL**       | SC-01..07 prove order |
| Actor pipeline end-to-end          | —               | S329     | **FULL**       | VP-01..09 |
| Reproducible smoke                 | —               | S330     | **FULL**       | 5-phase script |
| Fill event field preservation      | —               | S329     | **FULL**       | VP-04 (12 fields + JSON) |

### 1.2. PWT Item Delivery Matrix

| PWT Item | Charter Target | Actual Delivery | Delta | Evidence |
|----------|----------------|-----------------|-------|----------|
| PWT-1    | S328           | S328            | On time | `venue_adapter_actor.go:120-126` |
| PWT-2    | S329           | S328            | 1 stage early | `venue_adapter_actor.go:133-139` |
| PWT-3    | S328           | S328            | On time | `venue_adapter_actor.go:127-131` |
| PWT-4    | S330           | S330            | On time | `make smoke-composed` PASS |

### 1.3. Invariant Preservation Matrix

| ID         | Invariant                    | S327 | S328 | S329 | S330 | Final |
|------------|------------------------------|------|------|------|------|-------|
| EC-1       | Deterministic client order ID | ✓    | ✓    | ✓    | ✓    | ✓     |
| EC-3       | Per-request deadline          | ✓    | ✓    | ✓    | ✓    | ✓     |
| F-1        | No bare errors / Problem type | ✓    | ✓    | ✓    | ✓    | ✓     |
| F-4        | Credential redaction          | ✓    | ✓    | ✓    | ✓    | ✓     |
| RF-1       | Retryable flag accuracy       | ✓    | ✓    | ✓    | ✓    | ✓     |
| PGR-08     | Intent immutability           | ✓    | ✓    | ✓    | ✓    | ✓     |
| INV-REC-1  | No duplicate execution        | ✓    | ✓    | ✓    | ✓    | ✓     |
| INV-RC-1   | Deadline independence         | ✓    | ✓    | ✓    | ✓    | ✓     |
| INV-OBS-1  | Zero noise on success         | ✓    | ✓    | ✓    | ✓    | ✓     |

All 9 invariants preserved across all 4 stages. No violations detected.

## 2. Regression Audit

### 2.1. Test Suite Regression

| Metric              | Baseline (S327) | Current (S331) | Delta      |
|---------------------|-----------------|----------------|------------|
| Total test count    | 186             | 186+           | +16 (SC+VP)|
| Failures            | 0               | 0              | 0          |
| Suite runtime       | ~32s            | ~32s           | Stable     |
| `go vet` warnings   | 0               | 0              | 0          |

**Verdict: ZERO REGRESSIONS.**

### 2.2. Interface Regression

| Check                            | Result |
|----------------------------------|--------|
| New interfaces introduced        | None   |
| Existing interfaces modified     | None   |
| New actor types created          | None   |
| New configuration knobs added    | None   |
| Retry policy parameters changed  | None   |

**Verdict: NO INTERFACE INFLATION.**

### 2.3. Behavioral Regression

| Scenario                            | Before Tranche          | After Tranche            | Regression? |
|-------------------------------------|-------------------------|--------------------------|-------------|
| First-attempt success               | Direct submit           | Submit via decorators    | No (SC-02)  |
| Non-retryable error                 | Direct passthrough      | Passthrough via stack    | No (SC-03)  |
| Safety gate block                   | Blocks before submit    | Blocks before decorators | No (VP-09)  |
| Paper mode (no query)               | Direct submit           | Retry-only decorators    | No (VP-07)  |

**Verdict: NO BEHAVIORAL REGRESSION.**

## 3. Residual Gaps

### 3.1. Accepted Gaps (Carried from Prerequisites)

These gaps were explicitly accepted during S322–S326 and remain unchanged.

| Gap ID     | Description                              | Risk    | Deferred To      |
|------------|------------------------------------------|---------|------------------|
| R-S322-1   | Single recovery attempt (no retry on query) | Low   | Post-tranche     |
| R-S322-2   | No persistence of ambiguous state         | Low     | OMS wave         |
| R-S322-3   | Fill granularity differs submit vs query  | V. Low  | Post-tranche     |
| R-S322-4   | Theoretical race: order not yet queryable | V. Low  | Monitor          |
| R-S323-1   | Deadline does not cancel in-flight submit | Low     | Post-tranche     |
| R-S323-2   | Halt check timeout fixed at 2s            | V. Low  | Post-tranche     |
| R-S325-1   | No real-world error code corpus           | Low     | Mainnet data     |
| R-S325-2   | No Retry-After header extraction          | Low     | Post-tranche     |
| R-S325-3   | Mapping is Binance-specific               | By design | Per-venue scope |
| R-S320-6   | Per-error-class retry policies            | Low     | Production phase |

### 3.2. Tranche-Specific Gaps

These are limitations noted during S327–S330 execution.

| Gap ID     | Description                                | Risk    | Deferred To       |
|------------|--------------------------------------------|---------|-------------------|
| R-S328-1   | Retry policy not config-driven             | Low     | Post-tranche      |
| R-S328-2   | Reconciliation timeout not config-driven   | Low     | Post-tranche      |
| R-S328-3   | No circuit breaker pattern                 | Low     | Design wave       |
| R-S328-4   | No OpenTelemetry/tracing                   | Low     | Observability wave|
| R-S330-1   | Smoke does not exercise live NATS          | Medium  | Stack smoke       |
| R-S330-2   | Smoke does not exercise real venue HTTP    | Medium  | Venue smoke       |
| R-S330-3   | Startup log field verification not automated | Low   | Ops wave          |

### 3.3. Integration Gaps (Pre-existing, Not Tranche Scope)

| Gap                                     | Mitigation                        | Priority |
|-----------------------------------------|-----------------------------------|----------|
| NATS consumer → actor message flow      | Documented as transitional bridge | High     |
| Control KV store live connection        | Fail-open pattern in code         | Medium   |
| Fill publisher end-to-end               | Separate concern from venue wiring| Low      |
| ClickHouse persistence round-trip       | Requires stack                    | Medium   |

## 4. Gate Verdict

### Formal Declaration

**PRODUCTION WIRING TRANCHE: CLOSED**

| Criterion                               | Status |
|-----------------------------------------|--------|
| All 4 PWT items completed and verified  | PASS   |
| Test suite: 0 failures                  | PASS   |
| All 9 invariants preserved              | PASS   |
| Composed pipeline in integration test   | PASS   |
| No new interfaces introduced            | PASS   |
| No scope inflation                      | PASS   |
| Retry metadata in actor structured logs | PASS   |

**Classification: FULL CLOSURE**

All capabilities proven in isolation are now composed and operationally verified
in the actor pipeline. The composition is exercised through reproducible smoke
with zero regressions. Residual gaps are explicitly bounded and do not block
closure.

## 5. Recommendation for Next Ceremony

### 5.1. What the Evidence Says

The venue execution domain has reached **operational composition**. The
decorator pipeline (retry → reconciliation → observability) is wired, tested,
and smoke-validated. The gap profile has shifted from "components exist but
aren't composed" to "composition exists but isn't exercised against live
infrastructure."

### 5.2. Candidate Macro-Fronts

Based on the residual gap profile, three directions emerge from evidence:

| Front                        | Evidence Driver                           | Risk if Deferred |
|------------------------------|-------------------------------------------|-------------------|
| **A. Live stack integration** | R-S330-1/2, NATS consumer gap, fill publisher gap | Medium — composition proven in-process only |
| **B. OMS / persistence wave** | R-S322-2, ClickHouse round-trip gap        | Low — testnet tolerates manual recovery |
| **C. Observability deepening** | R-S328-3/4, R-S330-3                      | Low — slog sufficient for testnet |

### 5.3. Recommendation

**Next ceremony: Charter for Live Stack Integration.**

Rationale:
- The highest-risk residual gaps (R-S330-1, R-S330-2, NATS consumer flow) all
  require exercising the composed pipeline against live infrastructure.
- This is the natural progression: isolation → composition → live verification.
- OMS and observability deepening can wait; the current slog-based observability
  and manual recovery are sufficient for testnet operation.

Suggested scope for charter:
1. NATS consumer → VenueAdapterActor message flow verification
2. Fill event publication and consumption round-trip
3. Control KV store live kill-switch exercise
4. Smoke with live NATS stack (`make smoke-live-stack`)
5. Optional: venue HTTP smoke with testnet credentials (`make smoke-venue`)

### 5.4. What NOT to Open Next

- Mainnet activation (no evidence supports readiness)
- Multi-venue expansion (single-venue not live-proven yet)
- Dashboard/monitoring UI (premature without live data)
- Per-error-class retry policies (requires production error corpus)
- Circuit breaker design (requires sustained load data)
