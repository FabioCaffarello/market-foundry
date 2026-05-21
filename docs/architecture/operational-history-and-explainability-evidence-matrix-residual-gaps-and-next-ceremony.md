# Operational History & Explainability -- Evidence Matrix, Residual Gaps, and Next Ceremony

**Stage**: S456A
**Wave**: Operational History & Explainability (S452A--S455A)
**Date**: 2026-03-24

---

## 1. Evidence Matrix

### 1.1 Capability Evidence

| ID | Capability | Grade | Code Evidence | Test Evidence | Doc Evidence |
|----|-----------|-------|---------------|---------------|-------------|
| C1 | Persistence Completeness Invariant | SUBSTANTIAL | `QueryLifecycleHistory()` enables cross-type CH query without type filter; `explain` endpoint detects KV/CH per-key divergence | 9 query builder tests + 6 explain tests cover divergence detection | Historical read model doc, consistency findings doc |
| C2 | Type/Status Disambiguation | FULL | `type` column exposed in all query responses (lifecycle, list, summary, explain); no translation differences between surfaces | 11 list query tests validate type filter; consistency audit confirms type parity | Field-level consistency matrix (15 fields audited) |
| C3 | Session Metadata Persistence | PARTIAL | No dedicated session model or KV bucket; explain endpoint provides per-key session-scoped view | No session metadata tests (not implemented) | Charter documents design but implementation deferred |
| C4 | Order Narrative Query | SUBSTANTIAL | `GET /analytical/execution/lifecycle` returns full execution timeline; `GET /analytical/execution/explain` combines KV + CH + consistency + narrative | 19 lifecycle history tests; 6 explain tests with narrative generation | Read model design doc, explainability surface doc |
| C5 | List Query Ergonomics | FULL | 3 new endpoints: execution list (relaxed filter), execution summary (GROUP BY), lifecycle list (KV via HTTP) | 23 tests covering all filter combinations, validation, error handling | List queries doc, filter semantics doc |
| C6 | KV-to-ClickHouse Consistency Audit | SUBSTANTIAL | `explain` endpoint compares 3 fields across KV and CH, returns consistent/divergent/unavailable per field; `intentToLifecycleEntry()` prevents field drift | 6 explain tests cover consistent, divergent, and unavailable states | Consistency findings doc (15-field matrix), parity corrections documented |
| C7 | Post-Session Verification Automation | PARTIAL | Data surfaces (list, summary, explain) provide the foundation; no dedicated PO check harness | No PO automation tests (not implemented) | S447 PO protocol referenced but not codified |

### 1.2 Governing Question Evidence

| ID | Question | Status | Answering Stage | Evidence |
|----|----------|--------|----------------|---------|
| Q1 | Why 50% persistence gap? | PARTIALLY ANSWERED | S453A/S455A | Lifecycle history enables precise gap detection; explain endpoint flags divergences. Root cause of S449-specific gap not forensically resolved. |
| Q2 | Live vs paper distinction? | FULLY ANSWERED | S453A/S454A/S455A | Type field consistent across all surfaces (Finding A1). Execution list supports type filter. |
| Q3 | Full lifecycle narrative? | SUBSTANTIALLY ANSWERED | S453A/S455A | Execution-domain lifecycle fully traceable. Cross-domain (signal->fill) not included. |
| Q4 | Divergence detection? | SUBSTANTIALLY ANSWERED | S455A | Per-key detection operational via explain endpoint. Batch detection absent. |
| Q5 | Automated PO checks? | NOT YET | -- | Data foundation exists but PO checks not codified as automation. |
| Q6 | Session metadata queryable? | NOT YET | -- | No first-class session entity. Deferred. |

### 1.3 Finding Closure Status

| Finding | Original Problem | Wave Response | Closed? |
|---------|-----------------|---------------|---------|
| F3 | 50% persistence gap | Lifecycle history surface makes gaps visible; explain endpoint detects per-key divergence | MITIGATED (detection improved; root cause not forensically resolved) |
| F4 | type=paper_order confusion | Type column exposed and verified consistent across surfaces | CLOSED |
| F5 | status=submitted stuck | Status field verified consistent across KV and CH; lifecycle timeline shows status progression | CLOSED |
| F7 | 2/9 PO checks executed | Data surfaces enable all checks; automation harness not built | PARTIALLY CLOSED |
| F10 | Undocumented infrastructure friction | Wave architecture docs improve operational knowledge base | MITIGATED |

---

## 2. Test Coverage Summary

| Stage | Package | Tests | All Pass |
|-------|---------|-------|----------|
| S453A | `internal/adapters/clickhouse` | 9 (query builder) | YES |
| S453A | `internal/application/analyticalclient` | 10 (use case) | YES |
| S454A | `internal/adapters/clickhouse` | 11 (query builder) | YES |
| S454A | `internal/application/analyticalclient` | 12 (use case) | YES |
| S455A | `internal/application/analyticalclient` | 6 (explain + consistency) | YES |
| **Total** | | **48** | **YES** |

Regression check: 334 test cases across all affected packages pass. Zero regressions.

---

## 3. HTTP Surface Inventory (Wave Additions)

| Endpoint | Stage | Backing | Purpose |
|----------|-------|---------|---------|
| `GET /analytical/execution/lifecycle` | S453A | ClickHouse | Cross-type lifecycle timeline |
| `GET /analytical/execution/list` | S454A | ClickHouse | Relaxed-filter execution list |
| `GET /analytical/execution/summary` | S454A | ClickHouse | Aggregate counts by type/status |
| `GET /execution/lifecycle/list` | S454A | KV via NATS | Enumerate all tracked partition keys |
| `GET /analytical/execution/explain` | S455A | KV + ClickHouse | Unified explainability + consistency |

**Pre-wave endpoints preserved** (no modifications to existing routes):
- `GET /execution/:type/latest` -- KV single-bucket latest
- `GET /execution/status/latest` -- KV composite status
- `GET /analytical/execution/history` -- CH type-specific history

---

## 4. Residual Gaps

### 4.1 Bounded Gaps (Known, Documented, Non-Blocking)

| # | Gap | Severity | Why Non-Blocking | Possible Future Stage |
|---|-----|----------|-----------------|----------------------|
| G1 | No first-class session metadata entity (C3) | LOW | Per-key explain endpoint provides session-scoped views; session metadata is a convenience, not a correctness concern | Future operational tooling wave |
| G2 | No batch KV-to-CH consistency audit | LOW | Per-key checking via explain is operational; batch checking is quality-of-life | Script or cron job; no new infrastructure |
| G3 | No automated PO check harness (C7) | LOW | All 9 PO checks can be manually executed using the new query surfaces; automation is a scripting exercise | Future operational tooling wave |
| G4 | No cross-domain lifecycle trace (signal -> fill) | LOW | Execution-domain trace is complete; cross-domain requires `/analytical/composite/chain` which exists separately | Out of wave scope by design |
| G5 | No sub-second timestamp in HTTP responses | NEGLIGIBLE | Millisecond precision preserved in storage; HTTP RFC3339 is sufficient for operational review | Accept as-is |
| G6 | No cursor-based pagination | LOW | Offset/limit sufficient for current data volumes; cursor pagination needed only at scale | Future performance wave |
| G7 | F3 root cause not forensically resolved | LOW | New surfaces make similar gaps immediately detectable; the specific S449 gap is historical | Accept -- detection is the durable fix |

### 4.2 Gaps That Do NOT Exist (Addressed by Wave)

| Concern | Resolution |
|---------|-----------|
| Type confusion in read surfaces | C2 FULL -- type exposed and consistent |
| Status stuck at derive-side | Status consistent across KV and CH per audit |
| No historical read model | C4 SUBSTANTIAL -- lifecycle timeline operational |
| No way to list/filter executions | C5 FULL -- 3 new endpoints with relaxed filters |
| KV/CH field parity drift | D1 corrected (Risk, Parameters added); `intentToLifecycleEntry()` prevents future drift |
| No cross-surface consistency view | C6 SUBSTANTIAL -- explain endpoint with per-field checks |

---

## 5. Regression Risk Assessment

| Risk | Assessment | Evidence |
|------|-----------|---------|
| New endpoints break existing ones | NONE | All 334 pre-existing tests pass; new routes are additive |
| Gateway composition changes break boot | NONE | `go build ./...` succeeds for all 17 modules |
| Contract changes break wire compatibility | NONE | New types are additive; existing types unchanged except LifecycleHistoryEntry (Risk, Parameters added -- strictly additive) |
| ClickHouse query changes break existing queries | NONE | New query builders are separate functions; existing `BuildExecutionQuery` unchanged |

---

## 6. Wave Scorecard

| Dimension | Score | Rationale |
|-----------|-------|-----------|
| Core value delivered | HIGH | System can now explain execution lifecycle through unified surfaces |
| Scope discipline | HIGH | No infrastructure added; existing KV + CH + NATS reused throughout |
| Test rigor | HIGH | 48 new tests; zero regressions across 334 existing tests |
| Documentation quality | HIGH | 8 architecture docs + 4 stage reports; all limitations documented |
| Gap honesty | HIGH | C3 and C7 explicitly marked PARTIAL; Q5 and Q6 marked NOT YET |
| Guard rails respected | FULL | No dashboards, no live sessions, no OMS changes, no schema migrations |

---

## 7. Formal Wave Verdict

**WAVE CLOSED -- SUBSTANTIALLY COMPLETE**

The Operational History & Explainability wave achieved its primary objective: the system can now explain what it executed. The 5 new HTTP endpoints, 48 tests, field-level consistency audit, and structured explain surface represent a material improvement in operational confidence.

Two capabilities (C3 session metadata, C7 PO automation) remain PARTIAL. These are bounded gaps that do not undermine the wave's value and do not require a closure micro-stage.

---

## 8. Next Ceremony Recommendation

The wave closes without opening the next wave. The next direction should be determined by the most pressing operational need:

### Option A: Second Supervised Live Session
- **Motivation**: Validate the full pipeline under real market conditions with the new observability surfaces active.
- **Prerequisite**: Live Session Stabilization track (if running in parallel) must be ready.
- **Value**: Proves that the new read surfaces detect issues that S449 missed.

### Option B: Automated Operational Verification Wave
- **Motivation**: Codify PO checks, batch consistency audit, session metadata, and pre-flight validation as automated tooling.
- **Scope**: Small wave (3--4 stages). Addresses G1, G2, G3 from this gate.
- **Value**: Reduces operator burden and enables unattended verification.

### Option C: Performance and Resilience Hardening
- **Motivation**: Stress test query surfaces under load; validate writer pipeline throughput under sustained ingestion.
- **Scope**: Medium wave. Addresses pagination (G6) and writer pipeline resilience.
- **Value**: Prepares for multi-symbol and sustained live trading.

**Recommendation**: Option A (second live session) has the highest strategic value. The new read surfaces are best validated by actual operational use. Option B can run in parallel as short incremental work.
