# Family 03 — Success Criteria and Operability Scope

**Family**: Strategies (`mean_reversion_entry`)
**Gate**: Must pass all criteria below before Family 04 selection begins.

---

## 1. Success criteria

### SC — Schema Coherence

| ID    | Criterion                                                              | Pass/Fail |
|------ |----------------------------------------------------------------------- |---------- |
| SC-01 | DDL column count matches writer mapper value count (15)                | ☐         |
| SC-02 | DDL column types match writer value types for all 15 columns           | ☐         |
| SC-03 | Reader SELECT column count matches scan variable count (11)            | ☐         |
| SC-04 | Reader scan types match DDL types for all 11 domain columns            | ☐         |
| SC-05 | `decisions` JSON round-trips: write `[]DecisionInput` → read `[]DecisionInput` | ☐ |
| SC-06 | `parameters` JSON round-trips: write `map[string]string` → read `map[string]string` | ☐ |
| SC-07 | `metadata` JSON round-trips: write `map[string]string` → read `map[string]string` | ☐ |

### RP — Read Path

| ID    | Criterion                                                              | Pass/Fail |
|------ |----------------------------------------------------------------------- |---------- |
| RP-01 | Results ordered newest-first (DESC by timestamp)                       | ☐         |
| RP-02 | Direction filter returns only matching rows when set                   | ☐         |
| RP-03 | Direction filter returns all rows when empty                           | ☐         |
| RP-04 | Default limit (50) applied when limit parameter absent                 | ☐         |
| RP-05 | Max limit (500) enforced when exceeded                                 | ☐         |
| RP-06 | Empty result returns `[]` (not null) in JSON response                  | ☐         |
| RP-07 | Since/until time filters produce correct bounded results               | ☐         |

### AL — Application Layer

| ID    | Criterion                                                              | Pass/Fail |
|------ |----------------------------------------------------------------------- |---------- |
| AL-01 | Missing `type` returns 400 with problem JSON                           | ☐         |
| AL-02 | Missing `source` returns 400 with problem JSON                         | ☐         |
| AL-03 | Missing `symbol` returns 400 with problem JSON                         | ☐         |
| AL-04 | Zero/negative `timeframe` returns 400 with problem JSON                | ☐         |
| AL-05 | `since > until` returns 400 with problem JSON                          | ☐         |
| AL-06 | Nil use case / nil reader returns 503 with unavailable problem         | ☐         |

### HS — HTTP Surface

| ID    | Criterion                                                              | Pass/Fail |
|------ |----------------------------------------------------------------------- |---------- |
| HS-01 | `GET /analytical/strategy/history` returns 200 with valid params       | ☐         |
| HS-02 | Response body has `strategies`, `source`, `meta` fields                | ☐         |
| HS-03 | `Server-Timing` header includes `total` and `query` durations          | ☐         |
| HS-04 | Invalid `limit` (non-integer) returns 400                              | ☐         |
| HS-05 | Out-of-range `limit` (0 or >500) returns 400                           | ☐         |
| HS-06 | Missing required params return 400                                     | ☐         |

### IN — Integration

| ID    | Criterion                                                              | Pass/Fail |
|------ |----------------------------------------------------------------------- |---------- |
| IN-01 | `strategies` table exists and is queryable in ClickHouse               | ☐         |
| IN-02 | Writer pipeline consumes and inserts strategy events                   | ☐         |
| IN-03 | Smoke test exercises strategy endpoint and validates response shape    | ☐         |
| IN-04 | All pre-existing analytical smoke phases still pass                    | ☐         |
| IN-05 | All unit tests pass (`go test ./...`)                                  | ☐         |
| IN-06 | CI pipeline passes                                                     | ☐         |

### BP — Boundary Preservation

| ID    | Criterion                                                              | Pass/Fail |
|------ |----------------------------------------------------------------------- |---------- |
| BP-01 | Zero changes to writer mapper (`mapStrategyRow`)                       | ☐         |
| BP-02 | Zero changes to pipeline entry                                         | ☐         |
| BP-03 | Zero changes to migration 004                                          | ☐         |
| BP-04 | No cross-family queries (no joins between analytical tables)           | ☐         |
| BP-05 | No changes to candle, signal, or decision read paths                   | ☐         |
| BP-06 | Operational endpoints unaffected                                       | ☐         |

---

## 2. Non-goals

### Architectural

| ID    | Non-goal                                                               | Rationale                               |
|------ |----------------------------------------------------------------------- |---------------------------------------- |
| NG-01 | No smoke parameterization (H-2)                                        | Deferred hardening; tracked separately  |
| NG-02 | No naming cleanup (H-3)                                                | Deferred hardening; tracked separately  |
| NG-03 | No code generation for reader/handler                                  | Pattern is simple enough to be manual   |
| NG-04 | No generic reader abstraction                                          | Not enough families to justify (3 → 4)  |
| NG-05 | No struct DI refactoring beyond current state                          | H-1 already completed in S172           |

### Feature

| ID    | Non-goal                                                               | Rationale                               |
|------ |----------------------------------------------------------------------- |---------------------------------------- |
| NG-06 | No direction aggregation (count by direction)                          | Analytical deepening, not coverage      |
| NG-07 | No confidence filtering                                                | Not needed for operational debugging    |
| NG-08 | No decision drill-down (join strategies → decisions)                   | Cross-family; violates boundary rule    |
| NG-09 | No write-time validation in pipeline                                   | Writer is append-only by design         |
| NG-10 | No cross-timeframe queries                                             | Out of scope for all families           |
| NG-11 | No parameters filtering                                                | JSON column; not indexed                |

### Operational

| ID    | Non-goal                                                               | Rationale                               |
|------ |----------------------------------------------------------------------- |---------------------------------------- |
| NG-12 | No Prometheus / OpenTelemetry                                          | Observability via inserter counters + Server-Timing |
| NG-13 | No pagination beyond limit 500                                         | Sufficient for operational queries      |
| NG-14 | No auto-recovery or backfill                                           | Writer handles via supervisor restart   |
| NG-15 | No custom TTL per query                                                | 90-day TTL from DDL is sufficient       |

---

## 3. Observability scope (minimal)

| Signal                  | Source                   | Already exists? |
|------------------------ |------------------------- |---------------- |
| Inserter event counter  | Writer supervisor        | Yes             |
| Inserter batch timing   | Writer inserter          | Yes             |
| Consumer event counter  | Writer consumer          | Yes             |
| Query elapsed time      | Reader adapter log       | S176            |
| HTTP Server-Timing      | Handler header           | S176            |
| Query row count in log  | Reader adapter log       | S176            |
| statusz/diagz visibility| Writer pipeline tracker  | Yes             |

No new observability infrastructure. Reader adapter logs and Server-Timing header provide the same coverage as candle, signal, and decision families.

---

## 4. Runbook scope (minimal)

### 4.1 Diagnostic queries

```sql
-- Verify strategies table has data
SELECT count() FROM strategies;

-- Check recent strategy events
SELECT type, source, symbol, timeframe, direction, timestamp
FROM strategies
ORDER BY timestamp DESC
LIMIT 10;

-- Check strategy distribution by direction
SELECT direction, count() FROM strategies GROUP BY direction;
```

### 4.2 Failure modes

| Symptom                                | Likely cause                        | Action                                  |
|--------------------------------------- |------------------------------------ |---------------------------------------- |
| Endpoint returns 503                   | ClickHouse not configured/connected | Check `clickhouse.addr` in config       |
| Empty results for known events         | Writer pipeline not enabled         | Check `IsStrategyFamilyEnabled("mean_reversion_entry")` |
| JSON fields return empty `[]` or `{}`  | Serialization mismatch              | Check writer `marshalJSON` output        |
| Slow queries (>1s)                     | Table growth beyond TTL retention   | Verify TTL is active; check partition count |

### 4.3 Recovery

- **Writer restart**: Supervisor handles automatic reconnection. No manual intervention needed.
- **Reader degradation**: Gateway returns 503 for strategy endpoint only; other analytical endpoints unaffected.
- **Schema mismatch**: Compare DDL columns against `mapStrategyRow()` value count and `Scan()` variable count.

---

## 5. Gate review checklist

Before Family 04 selection:

- [ ] All SC criteria pass
- [ ] All RP criteria pass
- [ ] All AL criteria pass
- [ ] All HS criteria pass
- [ ] All IN criteria pass
- [ ] All BP criteria pass
- [ ] Schema coherence documented
- [ ] Frictions captured (if any)
- [ ] No regressions in existing families

### Stop conditions

- **>2 new frictions** → pause and assess before continuing
- **CI unreliability** → halt until stable
- **Schema verification failure** → halt and investigate DDL/mapper/reader alignment
- **Writer degradation** → halt; writer correctness is non-negotiable
- **Pre-existing family test breakage** → halt; fix regressions before proceeding
