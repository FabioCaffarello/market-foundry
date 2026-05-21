# Production Readiness Hardening -- Capabilities, Governing Questions, and Non-Goals

> Companion to the [Wave Charter and Scope Freeze](production-readiness-hardening-wave-charter-and-scope-freeze.md).

---

## 1. Capabilities

| ID | Capability | Target Stage | Source |
|---|---|---|---|
| PRH-C1 | ClickHouse rejection event persistence | S411 | RG-1 from S409 |
| PRH-C2 | Rejection analytical queryability | S411 | RG-1 from S409 |
| PRH-C3 | Fill/rejection schema consistency in ClickHouse | S411 | RG-1 from S409 |
| PRH-C4 | Multi-symbol concurrent Spot execution | S412 | New (soak) |
| PRH-C5 | Sustained multi-cycle operation stability | S412 | New (soak) |
| PRH-C6 | Memory and goroutine leak absence | S412 | New (soak) |
| PRH-C7 | Graceful shutdown/restart without state corruption | S412 | New (soak) |
| PRH-C8 | Transient error recovery without state corruption | S412 | New (soak) |
| PRH-C9 | Commission asset type capture | S413 | RG-5 from S409 |
| PRH-C10 | Segment-scoped list query for operational diagnostics | S413 | RG-4 partial from S409 |
| PRH-C11 | Consolidated fill/rejection operational read surface | S413 | New (consolidation) |

---

## 2. Governing Questions

| ID | Question | Target Stage | Capability |
|---|---|---|---|
| PRH-Q1 | Do rejection events reach ClickHouse with correct schema and queryable fields? | S411 | PRH-C1, PRH-C2 |
| PRH-Q2 | Are fill and rejection records structurally consistent in the analytical store? | S411 | PRH-C3 |
| PRH-Q3 | Can the pipeline sustain 50+ order cycles across 3+ symbols without state corruption, memory leaks, or goroutine leaks? | S412 | PRH-C4, PRH-C5, PRH-C6 |
| PRH-Q4 | Does the system recover from transient venue errors (HTTP 429, 503, timeouts) without permanent lifecycle corruption? | S412 | PRH-C8 |
| PRH-Q5 | Does graceful shutdown preserve all in-flight state, and does restart resume without duplication or loss? | S412 | PRH-C7 |
| PRH-Q6 | Is the commission asset type captured from fill responses and available in the read-path? | S413 | PRH-C9 |
| PRH-Q7 | Can an operator list all Spot intents (or rejections) for a given symbol without knowing individual partition keys? | S413 | PRH-C10 |
| PRH-Q8 | Is there a single operational surface that exposes both fill and rejection lifecycle data? | S413 | PRH-C11 |

---

## 3. Non-Goals

### Frozen Non-Goals (Wave Level)

| ID | Non-Goal | Rationale |
|---|---|---|
| NG-36 | No Futures testnet venue execution proof | Separate wave. Spot hardening first. |
| NG-37 | No mainnet or production connectivity | Testnet only. Mainnet requires separate risk ceremony. |
| NG-38 | No multi-exchange support | Binance only. Multi-exchange is a separate wave. |
| NG-39 | No OMS expansion beyond current model | OMS Foundation (S388) model consumed as-is. No new order types, margin, leverage. |
| NG-40 | No broad analytics or observability platform | Only ClickHouse rejection writer (RG-1 closure). No dashboards, no Grafana, no metrics aggregation. |
| NG-41 | No portfolio risk management | Out of scope for this wave and the current system maturity. |
| NG-42 | No runtime architecture redesign | Unified runtime consumed as-is from S403. No actor topology changes. |
| NG-43 | No config schema changes | Existing schema sufficient. No new fields, no breaking changes. |
| NG-44 | No limit order support | Market orders only. Limit orders are a separate capability wave. |
| NG-45 | No per-segment dry_run toggle | Global dry_run only, inherited from prior waves. |
| NG-46 | No concurrent Spot + Futures venue_live | Only Spot venue_live. Futures venue_live is a separate proof. |
| NG-47 | No KV history or JetStream stream redesign | Latest-only KV semantics accepted. Historical queries via ClickHouse. RG-3 deferred. |
| NG-48 | No cross-intent analytical listing at scale | Partial operational listing in S413. Full analytical cross-intent queries deferred. |
| NG-49 | No credential management hardening | Credential handling consumed as-is. Secrets management is an operational concern outside this wave. |
| NG-50 | No CI/CD pipeline changes | Build and deployment infrastructure unchanged. |

### Inherited Non-Goal Compliance

All 35 non-goals from prior waves (NG-1 through NG-35) remain in force. This wave adds NG-36 through NG-50.

---

## 4. Capability-to-Gap Traceability

| S409 Gap | Severity | This Wave? | Target | Disposition |
|---|---|---|---|---|
| RG-1: ClickHouse rejection writer | Medium | YES | S411 (PRH-C1, PRH-C2, PRH-C3) | Full closure |
| RG-2: Partial fill live observation | Low | NO | Deferred | Venue constraint; structural proof sufficient |
| RG-3: Latest-only KV semantics | Low | NO | Deferred (NG-47) | ClickHouse covers historical queries |
| RG-4: Segment-scoped list queries | Low | PARTIAL | S413 (PRH-C10) | Operational listing only; analytical deferred |
| RG-5: Commission asset type | Low | YES | S413 (PRH-C9) | Full closure |

---

## 5. Success Criteria Summary

The wave succeeds if all governing questions (PRH-Q1 through PRH-Q8) are answered with FULL or SUBSTANTIAL classification, with zero NONE classifications, and no regressions against the 82+ tests from the S404--S408 wave or any prior wave capability.

---

## References

| Document | Path |
|---|---|
| Wave Charter | `docs/architecture/production-readiness-hardening-wave-charter-and-scope-freeze.md` |
| S409 Evidence Gate | `docs/stages/stage-s409-testnet-venue-execution-unified-runtime-spot-first-evidence-gate-report.md` |
| S409 Evidence Matrix | `docs/architecture/testnet-venue-execution-unified-runtime-spot-first-evidence-matrix-residual-gaps-and-next-ceremony.md` |
