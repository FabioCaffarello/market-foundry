# Operational vs. Analytical Query Boundaries

> **Stage:** S149 — Historical Query Surface Minimal Extension
> **Status:** Definitive
> **Scope:** Defines the boundary between operational queries (NATS KV) and analytical queries (ClickHouse) in the gateway.

---

## 1. Purpose

With the introduction of analytical endpoints in S149, the gateway now serves two distinct categories of queries. This document defines the boundary between them, why the boundary exists, and how it is enforced.

---

## 2. The Two Query Domains

### 2.1 Operational Queries

| Aspect | Value |
|--------|-------|
| **Data source** | NATS KV (via store service, request/reply) |
| **Route prefix** | `/evidence/*`, `/signal/*`, `/decision/*`, `/strategy/*`, `/risk/*`, `/execution/*` |
| **Latency** | Sub-millisecond (in-memory KV) |
| **Freshness** | Latest value only (per-key overwrite) |
| **Depth** | Current state + short in-memory history (store ring buffer) |
| **Availability** | Required for gateway readiness |
| **ClickHouse dependency** | None |
| **Purpose** | Real-time operational monitoring and pipeline validation |

### 2.2 Analytical Queries

| Aspect | Value |
|--------|-------|
| **Data source** | ClickHouse (direct connection from gateway) |
| **Route prefix** | `/analytical/*` |
| **Latency** | Milliseconds to seconds (disk-backed columnar store) |
| **Freshness** | Eventual (writer batches: 1000 rows or 5s) |
| **Depth** | Full history within TTL (90 days) |
| **Availability** | Optional — not in gateway readiness |
| **ClickHouse dependency** | Required for these endpoints only |
| **Purpose** | Historical inspection, backtesting verification, trend analysis |

---

## 3. Boundary Rules

### Rule B-01: Route Prefix Separation

Operational and analytical queries use distinct URL prefixes. No endpoint path is shared.

| Domain | Operational Route | Analytical Route |
|--------|------------------|------------------|
| Evidence candles | `/evidence/candles/latest` | `/analytical/evidence/candles` |
| Evidence candles | `/evidence/candles/history` | `/analytical/evidence/candles` (with time range) |
| Signals | `/signal/latest` | (future) `/analytical/signals` |
| Decisions | `/decision/latest` | (future) `/analytical/decisions` |

The `/analytical/` prefix makes the data source explicit in the URL.

### Rule B-02: Independent Failure Domains

| Failure | Operational Impact | Analytical Impact |
|---------|-------------------|-------------------|
| NATS down | All operational queries fail | No impact |
| Store down | Operational queries return 503 | No impact |
| ClickHouse down | No impact | Analytical queries return 503 |
| Writer down | No impact | No new data flows to ClickHouse (stale results) |

Failures in one domain never propagate to the other.

### Rule B-03: No Shared State

| Resource | Operational | Analytical |
|----------|------------|------------|
| Connection | NATS request client | ClickHouse client |
| Data path | Gateway → NATS → Store → NATS KV | Gateway → ClickHouse |
| Health check | In `/readyz` | Not in `/readyz` |
| Configuration | `nats` section | `clickhouse` section |

The two query domains share no connections, no state, and no health coupling.

### Rule B-04: No Cross-Domain Queries

A single HTTP request never queries both NATS KV and ClickHouse. Each endpoint resolves from exactly one data source.

### Rule B-05: Operational Routes Are Immutable

Existing operational routes (`/evidence/candles/latest`, `/evidence/candles/history`, etc.) are never modified to conditionally use ClickHouse. Their behavior is identical whether ClickHouse is configured or not.

---

## 4. Why This Boundary Matters

### 4.1 Operational Safety

The operational pipeline was designed, validated, and proven without ClickHouse. Mixing analytical concerns into operational endpoints would:

- Add a new failure mode to existing, proven endpoints
- Create testing combinatorics (with/without ClickHouse)
- Risk breaking operational monitoring during ClickHouse maintenance

### 4.2 Deployment Flexibility

With clear boundaries, operators can:

- Run the pipeline without ClickHouse (minimal deployment)
- Add ClickHouse later without changing any operational configuration
- Perform ClickHouse maintenance without affecting operational queries
- Scale ClickHouse independently of the operational pipeline

### 4.3 Cognitive Clarity

Developers and operators always know which data source serves a given endpoint:

- `/evidence/candles/latest` → NATS KV (store)
- `/analytical/evidence/candles` → ClickHouse

No ambiguity. No conditional branching. No "it depends on configuration."

---

## 5. Common Questions

### Q: Why not merge operational and analytical history?

The operational `/evidence/candles/history` endpoint serves from the store's in-memory ring buffer. It provides short, fast history for the most recent candles. The analytical endpoint provides deep, slower history from ClickHouse. They serve different use cases with different performance characteristics.

### Q: Will operational endpoints ever read from ClickHouse?

No. This is a hard architectural boundary. Operational endpoints read from NATS KV. If a use case requires enriching operational data with historical context, it should be a new analytical endpoint, not a modification of an existing operational one.

### Q: Can the same data appear in both?

Yes. The most recent candles exist in both NATS KV (via store) and ClickHouse (via writer). This is expected and intentional — they serve different access patterns. The data is eventually consistent between them.

### Q: What if ClickHouse has data but the store doesn't?

This happens when the store is restarted (loses in-memory state) but ClickHouse retains history. The operational endpoint returns what the store has (possibly nothing). The analytical endpoint returns what ClickHouse has. They are independent.

---

## 6. Enforcement

| Mechanism | What It Enforces |
|-----------|-----------------|
| Route prefix (`/analytical/`) | Visual and structural separation in URL space |
| Separate handler (`AnalyticalWebHandler`) | No code sharing with operational handlers |
| Separate use case package (`analyticalclient`) | No code sharing with operational use cases |
| Conditional route registration | Analytical routes only appear when ClickHouse is configured |
| Readiness check exclusion | ClickHouse failures don't affect gateway readiness |
| Independent connection lifecycle | ClickHouse client created and closed independently of NATS |

---

## 7. Boundary Map

```
Gateway HTTP Server
│
├── /healthz, /readyz          ── Core (always present)
│
├── /config/*                  ── Operational: configctl (via NATS)
│
├── /evidence/*                ── Operational: evidence (via NATS → store)
├── /signal/*                  ── Operational: signal (via NATS → store)
├── /decision/*                ── Operational: decision (via NATS → store)
├── /strategy/*                ── Operational: strategy (via NATS → store)
├── /risk/*                    ── Operational: risk (via NATS → store)
├── /execution/*               ── Operational: execution (via NATS → store)
│
└── /analytical/*              ── Analytical: ClickHouse (optional, direct)
    └── /analytical/evidence/candles  ── S149: minimal historical candle query
```

The `/analytical/*` subtree is the only gateway surface that touches ClickHouse. Everything above it is ClickHouse-free, now and in the future.
