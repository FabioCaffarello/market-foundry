# Analytical End-to-End Integration Proof

## Purpose

This document defines the integration proof for the analytical layer's complete data path: **NATS → writer → ClickHouse → reader → HTTP**. It establishes what is being proven, the method, and the acceptance criteria.

## Context

The S156 readiness review identified one remaining blocker for the analytical layer: the absence of an end-to-end integration test proving real data flow through the system. S157 reviewed responsibilities and S158 hardened boundaries. This proof closes the last S156 precondition.

## Scope

### What is proven

The minimum complete analytical data path:

```
NATS JetStream (EVIDENCE_EVENTS stream)
  → writer service (consumer + batch inserter)
  → ClickHouse (evidence_candles table)
  → reader (CandleReader parameterized SELECT)
  → GET /analytical/evidence/candles (gateway HTTP)
```

### What is NOT proven

| Exclusion | Reason |
|-----------|--------|
| Non-candle families (signals, decisions, strategies, risks, executions) | Candle path is representative; other families use identical machinery |
| Multi-symbol concurrent writes | Single symbol sufficient for path proof |
| Failure recovery (writer restart, ClickHouse downtime) | Covered by unit tests in S152/S154; live proof is future scope |
| Reader observability counters | Known open debt from S156 |
| Performance under load | Not in scope for integration proof |
| Time-range query correctness | Structural proof via candle_reader_test.go; HTTP error handling validated |

## Method

### Proof script

`scripts/smoke-analytical-e2e.sh` — automated 7-phase verification:

| Phase | What | How |
|-------|------|-----|
| 1. Infrastructure readiness | ClickHouse, writer, gateway healthy | Health/readiness probes |
| 2. Migration status | All 7 tables exist, 6+ migrations applied | Direct ClickHouse queries |
| 3. Writer pipeline health | Writer receiving events from NATS | Writer /statusz tracker inspection |
| 4. ClickHouse data verification | Rows persisted in evidence_candles | Direct row count + sample query |
| 5. Reader → HTTP query surface | Analytical endpoint returns candles | curl + response structure validation |
| 6. Error handling | Invalid params return 400 | Three negative test cases |
| 7. Writer observability | No degraded pipelines | Writer /diagz inspection |

### Prerequisites

1. Full stack running: `make up`
2. Data seeded: `make seed` (or `make seed-multi`)
3. Wait ~120s for ingest → derive → writer pipeline to produce and flush data

### Execution

```bash
make smoke-analytical
# or directly:
./scripts/smoke-analytical-e2e.sh
./scripts/smoke-analytical-e2e.sh --wait 180  # longer flush wait
```

## Acceptance Criteria

| Criterion | Metric |
|-----------|--------|
| Write path proven | evidence_candles row count > 0 |
| Read path proven | HTTP response contains candles with source="clickhouse" |
| Response structure correct | All 12 required candle fields present |
| Error handling correct | 400 for missing timeframe, invalid limit, since>until |
| No degraded pipelines | Writer /diagz shows 0 degraded families |
| All phases pass | Script exits with code 0 |

## Integration with existing workflow

| Target | Description |
|--------|-------------|
| `make smoke-analytical` | Run analytical E2E proof |
| `make smoke` | Operational path proof (unchanged) |
| `make smoke-multi` | Multi-symbol operational proof (unchanged) |
| `make diag` | Includes writer diagnostics (unchanged) |

The analytical proof is intentionally separate from operational smoke tests. The operational path (NATS KV → query → HTTP) and analytical path (NATS → ClickHouse → HTTP) are independent projections with different infrastructure dependencies.

## Design decisions

1. **Script-based, not Go test**: The proof validates cross-service integration against a live Docker stack. Go integration tests with embedded NATS cannot exercise the full writer→ClickHouse→reader path.

2. **Polling with timeout**: Writer batch flush is asynchronous (5s interval, 1000 batch size). The script polls with configurable timeout rather than assuming data is immediately available.

3. **Candle family only**: Evidence candles are the first and most exercised family. Proving one family proves the machinery; other families use identical consumer→inserter→table paths.

4. **Additive, not replacing**: The script adds analytical coverage alongside existing operational smoke tests. No existing tests were modified.
