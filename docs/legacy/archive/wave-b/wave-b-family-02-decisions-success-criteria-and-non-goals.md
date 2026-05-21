# Wave B Family 02 — Decisions — Success Criteria and Non-Goals

**Stage:** S168
**Family:** Decisions (RSI Oversold)
**Iteration:** Wave B, Family 02

---

## 1. Success Criteria

Each criterion must be independently verifiable. If any criterion fails, the iteration does not pass its gate review.

### 1.1 Schema Coherence (SC)

| ID | Criterion | Verification Method |
|----|-----------|---------------------|
| SC-1 | All 10 domain columns in SELECT match DDL column names exactly | Reader unit test |
| SC-2 | Column types in Go scan match ClickHouse DDL types | Reader unit test |
| SC-3 | `signals` JSON round-trip: write `[]SignalInput` → read `[]SignalInput` | Reader unit test |
| SC-4 | `metadata` JSON round-trip: write `map[string]string` → read `map[string]string` | Reader unit test (reuse ParseMetadataJSON) |
| SC-5 | `confidence` stored as Float64, read back as string | Reader unit test |
| SC-6 | `outcome` stored as LowCardinality(String), read back as Outcome type | Reader unit test |

### 1.2 Read Path Correctness (RP)

| ID | Criterion | Verification Method |
|----|-----------|---------------------|
| RP-1 | `QueryDecisionHistory()` returns decisions ordered DESC by timestamp | Reader unit test |
| RP-2 | `outcome` filter in WHERE clause when provided | Query builder unit test |
| RP-3 | `outcome` filter absent from WHERE when not provided | Query builder unit test |
| RP-4 | Time range filtering works with since/until | Query builder unit test |
| RP-5 | Limit defaults to 50, max 500 | Use case unit test |
| RP-6 | Empty result set returns empty array, not null | Use case unit test |

### 1.3 Application Layer Correctness (AL)

| ID | Criterion | Verification Method |
|----|-----------|---------------------|
| AL-1 | Validation rejects missing type, source, symbol, zero timeframe | Use case unit test |
| AL-2 | Validation rejects invalid outcome values | Use case unit test |
| AL-3 | Validation accepts empty outcome (means "all") | Use case unit test |
| AL-4 | Reader error maps to Unavailable problem | Use case unit test |
| AL-5 | QueryMeta populated with query_ms and row_count | Use case unit test |

### 1.4 HTTP Surface Correctness (HS)

| ID | Criterion | Verification Method |
|----|-----------|---------------------|
| HS-1 | `GET /analytical/decision/history` returns 200 with valid params | Handler unit test |
| HS-2 | Response JSON has `decisions` array, `source`, `meta` fields | Handler unit test |
| HS-3 | Server-Timing header present with total and query durations | Handler unit test |
| HS-4 | 400 returned for missing/invalid params | Handler unit test |
| HS-5 | 503 returned when ClickHouse unavailable | Handler unit test |
| HS-6 | `outcome` query parameter accepted and passed to use case | Handler unit test |

### 1.5 Integration (IN)

| ID | Criterion | Verification Method |
|----|-----------|---------------------|
| IN-1 | `decisions` table has rows after writer runs | smoke-analytical-e2e.sh |
| IN-2 | HTTP endpoint returns decisions with all domain fields | smoke-analytical-e2e.sh |
| IN-3 | `signals` field in response is valid JSON array | smoke-analytical-e2e.sh |
| IN-4 | `metadata` field in response is valid JSON object | smoke-analytical-e2e.sh |
| IN-5 | All pre-existing smoke phases (candles, signals) still pass | smoke-analytical-e2e.sh |
| IN-6 | CI pipeline passes on branch | GitHub Actions |

### 1.6 Boundary Preservation (BP)

| ID | Criterion | Verification Method |
|----|-----------|---------------------|
| BP-1 | Zero changes to writer code | git diff verification |
| BP-2 | Zero changes to candle reader/handler/route | git diff verification |
| BP-3 | Zero changes to signal reader/handler/route | git diff verification |
| BP-4 | Operational pipeline unaffected (NATS KV path unchanged) | smoke test |
| BP-5 | ClickHouse optionality preserved (503 when unavailable) | handler unit test |
| BP-6 | No cross-family queries introduced | code review |

---

## 2. Non-Goals

These are explicitly out of scope for this iteration. Any work on these items violates the iteration boundary.

### 2.1 Architectural Non-Goals

| ID | Non-Goal | Rationale |
|----|----------|-----------|
| NG-1 | Struct-based DI for handler constructor | Committed for Family 03 (H-1). Do not pull forward unless blocking. |
| NG-2 | Rename `parseEvidenceKeyParams` | Committed for Family 03 (H-3). Accept naming residue for one more iteration. |
| NG-3 | Smoke test parameterization | Committed for Family 03 (H-2). Linear growth acceptable for 3 families. |
| NG-4 | Code generation or abstraction | Deferred to Family 04 evaluation (D-4). Mechanical duplication is intentional. |

### 2.2 Feature Non-Goals

| ID | Non-Goal | Rationale |
|----|----------|-----------|
| NG-5 | Outcome aggregation queries (e.g., "hit rate") | Cross-row analytics is beyond the append-and-query model. Future wave. |
| NG-6 | Outcome validation at write time | Writer mapper stores whatever the domain produces. Schema validation is domain responsibility. |
| NG-7 | Signal drill-down from decision response | The response includes `signals` as JSON. Clients use the signal endpoint for drill-down. No joins. |
| NG-8 | Confidence-based filtering | `outcome` filter is sufficient for this iteration. Numeric range filters add query complexity without proven demand. |
| NG-9 | Decision comparison across timeframes | Cross-timeframe queries violate single-family isolation. |

### 2.3 Operational Non-Goals

| ID | Non-Goal | Rationale |
|----|----------|-----------|
| NG-10 | Prometheus/OpenTelemetry integration | External monitoring deferred (D-5 through D-12). Structured logging is sufficient. |
| NG-11 | Pagination beyond 500 rows | No proven demand. Limit is consistent across all families. |
| NG-12 | Auto-recovery from ClickHouse disconnection | Sticky degradation is accepted (T-4). Restart is the recovery mechanism. |
| NG-13 | Backfill or historical import | C-4 constraint. No external infrastructure. |

---

## 3. Gate Review Criteria (5-Point — from Pattern v2)

The Family 02 gate review must answer all 5 questions affirmatively:

| # | Question | Pass Condition |
|---|----------|----------------|
| 1 | Do all unit tests pass? | `go test ./...` exits 0 across all affected modules |
| 2 | Does smoke-analytical pass end-to-end? | All phases including new Decision phase pass |
| 3 | Does CI pass on branch? | GitHub Actions workflow succeeds |
| 4 | Are there any regressions? | No existing test failures, no existing smoke failures |
| 5 | Is schema coherence documented? | Coherence table in implementation notes matches DDL/writer/reader |

### 3.1 Stop Conditions

Expansion halts immediately if any of:

- Family 02 introduces >2 new frictions not already cataloged in pattern v2
- CI becomes unreliable during implementation
- Schema coherence cannot be verified
- Writer pipeline stability degrades
- Any existing family's tests or smoke phases break

### 3.2 Friction Threshold

The S167 gate set a friction limit: if >2 **new** frictions emerge that are not in the v2 pattern document, expansion pauses. Known frictions (PF-1 through PF-6) do not count — only genuinely new discoveries.

---

## 4. Iteration Boundary

### 4.1 This Iteration Includes

- Decision reader adapter + tests
- Decision use case + tests
- Decision handler method + tests
- Decision route registration
- Decision contracts (query/reply)
- Gateway composition wiring
- Smoke test extension (Decision phase)
- Schema coherence documentation
- Implementation notes with friction log

### 4.2 This Iteration Excludes

- Any modification to the writer service
- Any modification to existing families (candles, signals)
- Any new migration files
- Any horizontal refactoring
- Any hardening work (struct DI, naming, smoke parameterization)
- Any third family definition or implementation
- Any cross-family query capability
- Any external monitoring integration

### 4.3 Post-Iteration Gate

After implementation (S169), a formal gate review will determine:

1. Whether Family 02 passed all success criteria
2. How many new frictions were discovered
3. Whether the hardening tranche (Family 03) can proceed as planned
4. Whether any H-1/H-2/H-3 commitments need acceleration

The gate review result determines whether Family 03 can begin. Family 03 is explicitly a **combined expansion + hardening** iteration — it must resolve H-1, H-2, and H-3 alongside adding the third family.
