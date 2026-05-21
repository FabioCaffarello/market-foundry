# Controlled Capability 01 — Success Criteria and Out of Scope

> Stage: S119 | Status: Defined | Date: 2025-03-19

## 1. Success Criteria

Each criterion is binary (pass/fail) and objectively verifiable.

### 1.1 Activation Criteria

| # | Criterion | Verification |
|---|-----------|-------------|
| A1 | Config activation with 2+ ingestion bindings (btcusdt + ethusdt) succeeds without error | `POST /configctl/configs/{id}/activate` returns 200 |
| A2 | Both bindings appear in active config | `GET /configctl/configs/active?scope_kind=global&scope_key=default` lists both |
| A3 | Ingest runtime discovers and connects both bindings without restart | `/statusz` shows two active WS trackers |

### 1.2 Pipeline Flow Criteria

| # | Criterion | Verification |
|---|-----------|-------------|
| P1 | Observation events flow for both symbols | ingest `/statusz` tracker event_count > 0 for both |
| P2 | Evidence (candle) materializes for both symbols | `GET /evidence/candles/latest?symbol=btcusdt` and `?symbol=ethusdt` both return non-null candle |
| P3 | Signal (RSI) computes for both symbols | `GET /signal/rsi/latest?symbol=ethusdt` returns valid signal |
| P4 | Decision evaluates for both symbols | `GET /decision/rsi_oversold/latest?symbol=ethusdt` returns valid decision |
| P5 | Strategy resolves for both symbols | `GET /strategy/mean_reversion_entry/latest?symbol=ethusdt` returns valid strategy |
| P6 | Risk assesses for both symbols | `GET /risk/position_exposure/latest?symbol=ethusdt` returns valid risk |
| P7 | Execution (paper order) processes for both symbols | `GET /execution/paper_order/latest?symbol=ethusdt` returns valid execution |
| P8 | Full chain latency per symbol is comparable to single-symbol baseline | Manual comparison of tracker idle times |

### 1.3 Diagnostic Criteria

| # | Criterion | Verification |
|---|-----------|-------------|
| D1 | All `/healthz` endpoints return 200 during multi-symbol operation | Repeated checks over 5+ minutes |
| D2 | All `/readyz` endpoints return ready status | Repeated checks over 5+ minutes |
| D3 | `/statusz` trackers reflect doubled activity (event counts increasing for both symbols) | Manual inspection |
| D4 | `/diagz` readiness checks all pass | Single check after stabilization |

### 1.4 Stability Criteria

| # | Criterion | Verification |
|---|-----------|-------------|
| S1 | No runtime crashes during 30+ minutes of sustained dual-symbol operation | Docker container status remains "healthy" |
| S2 | No error-level log entries from domain logic (infrastructure retries are acceptable) | `docker compose logs` filtered for ERROR |
| S3 | Memory usage does not grow unbounded (checked at 10-min and 30-min marks) | `docker stats` comparison |
| S4 | Zero data loss in the event chain (every observation produces downstream effects) | Tracker event counts are monotonically increasing |

### 1.5 Automation Criteria

| # | Criterion | Verification |
|---|-----------|-------------|
| T1 | `make smoke-multi` passes with both symbols validated end-to-end | Exit code 0 |
| T2 | raccoon-cli quality gate passes (`make check`) | Exit code 0 |
| T3 | All existing unit tests pass (`make test`) | Exit code 0 |

## 2. Minimum Viable Success

The capability is considered **delivered** when:
- All A* (activation) criteria pass
- All P* (pipeline flow) criteria pass
- All D* (diagnostic) criteria pass
- S1 and S2 (no crashes, no domain errors) hold for at least 15 minutes
- T1 and T3 pass

Criteria S3, S4, and T2 are **desired** but not blocking. If they fail, they produce friction records for S120, not blockers for S119 completion.

## 3. Out of Scope

### 3.1 Explicitly Excluded

| Item | Reason |
|------|--------|
| **Third symbol or more** | CC-01 proves the pattern with 2 symbols. Scaling to N is a follow-up. |
| **Per-symbol pipeline configuration** | Both symbols use the same family chain. Different families per symbol is a separate capability. |
| **Per-symbol tracker breakdown in /statusz** | Suspected useful, but adding it is a separate enhancement. Capture as friction if confirmed. |
| **Cross-runtime correlation ID injection** | Known friction (F1 from S118). Address in S120 if this capability confirms it's blocking. |
| **Cold-start documentation** | Known friction (F3 from S118). Useful but not required for CC-01. |
| **Soak test infrastructure** | CC-01 creates natural soak pressure. Dedicated infrastructure is a follow-up. |
| **Live venue adapter** | Remains gated behind activation ceremony. Paper-only for CC-01. |
| **New signal/decision/strategy families** | Adding MACD, Bollinger, etc. is a separate capability (CC-02+). |
| **ClickHouse write path** | Started but inactive. Not needed for KV-based monitoring. |
| **MarketMonkey absorption** | Explicitly deferred until Foundry proves it can deliver capability independently. |
| **OTel / distributed tracing** | Infrastructure investment. Not justified by current evidence. |
| **Composition root integration tests** | Live run proves wiring. Automated tests deferred. |
| **Use-case pattern unification** | Two patterns coexist without bugs. Unify when adding a new domain. |

### 3.2 Architectural Boundaries Preserved

These architectural decisions from prior stages are **not reopened** by CC-01:

- Actor hierarchy and supervision model (S96-S97)
- Boundary naming and interface hygiene (S98)
- Config-driven activation lifecycle (S104)
- Execution safety model (S113)
- Event stream topology (9 streams, 11 consumers)
- Runtime composition root patterns
- Graceful degradation contracts

### 3.3 Decision: What Happens If a Deferred Item Triggers?

If running CC-01 triggers a deferred item from S116 or prior:
1. **Record the trigger** with evidence (log line, error, operator confusion)
2. **Do not fix it in S119** unless it blocks CC-01 completion
3. **Carry it forward** as a P1 item for S120
4. If it blocks CC-01, escalate to the operator for a scope decision

## 4. Non-Objectives

These are things that CC-01 explicitly **does not aim to prove**:

| Non-Objective | Why |
|---------------|-----|
| The system handles 10+ symbols | 2 is sufficient to prove horizontal scaling. N-scaling is a follow-up. |
| The system survives network partitions | Failure modes are untested. CC-01 runs under normal conditions. |
| The system recovers from cold restart | Cold-start behavior is undocumented (F3). Not a CC-01 goal. |
| Performance benchmarks are established | CC-01 captures resource usage as a baseline, not as a benchmark. |
| The architecture needs no further changes | CC-01 will likely reveal friction. That is a feature, not a failure. |

## 5. Friction Capture Protocol

During CC-01 execution, any friction encountered is recorded using this template:

```markdown
### Friction: [short name]
- **Severity:** Low / Medium / High
- **Evidence:** [what happened, with log/screenshot]
- **Impact:** [what it prevented or made harder]
- **Recommendation:** Fix in S120 / Defer / Investigate
```

Friction records are collected in the S119 stage report and carried forward.
