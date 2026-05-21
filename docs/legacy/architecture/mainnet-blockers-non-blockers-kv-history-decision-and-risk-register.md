# Mainnet Blockers, Non-Blockers, KV History Decision, and Risk Register

> Stage: S430 | Date: 2026-03-23 | Type: Decision and Risk Register

## Purpose

This document is the actionable companion to the [Mainnet Readiness Audit](mainnet-readiness-audit-and-kv-history-strategy-decision.md). It provides a structured, classifiable matrix for every item relevant to a future mainnet authorization decision.

---

## 1. Classification Scheme

| Category | Definition | Action Required |
|----------|-----------|-----------------|
| **BLOCKER** | Prevents mainnet authorization; must be resolved before any mainnet ceremony | Yes — implementation required |
| **NON-BLOCKER** | Recommended improvement; does not prevent mainnet with current mitigations | No — backlog item |
| **ACCEPTED RISK** | Known limitation with documented mitigation; formally accepted | No — carry forward |
| **DEFERRED** | Out of scope for current wave; no risk implication | No — future wave |
| **CLOSED** | Previously open gap now resolved | No — reference only |

---

## 2. KV History Strategy Decision

| Attribute | Value |
|-----------|-------|
| **Gap ID** | RG-3 |
| **Previous Status** | Open (LOW severity, carried since S387) |
| **Decision** | Latest-only KV confirmed as production design |
| **New Status** | **CLOSED** |
| **Authority** | S430 Mainnet Readiness Audit |

### Decision Summary

NATS KV operates in latest-only mode by design. This is the correct operational model because:

1. KV answers "what is the current state?" — the only operational question that needs sub-second freshness
2. ClickHouse answers "what happened?" — the historical audit question, with SQL queryability and 90-day retention
3. JetStream 72-hour retention provides an interim buffer for writer catch-up after outages
4. KV history mode would add storage overhead and complexity with no clear consumer for intermediate revisions
5. Monotonicity enforcement in KV Put() prevents temporal violations regardless of history mode

### What This Means for Operations

- **To see current state:** Query KV via `execution.query.status.latest` or `execution.query.lifecycle.list`
- **To investigate past events:** Query ClickHouse `executions` table with time-range and status filters
- **To replay events:** Use JetStream consumer replay within 72-hour window
- **KV is not a backup:** ClickHouse backup/restore covers the historical audit requirement

---

## 3. Blockers Matrix

### B-1: No Mainnet Adapter Implementation

| Attribute | Value |
|-----------|-------|
| Category | BLOCKER |
| Severity | Critical |
| Current State | Only `binance_spot_testnet` and `binance_futures_testnet` adapters exist |
| Gap | Mainnet endpoints use different base URLs, authentication flows, and rate limits |
| Resolution | Implement `binance_spot_mainnet` and `binance_futures_mainnet` adapters |
| Effort Estimate | Medium (adapter pattern is proven; mainnet is a new instantiation) |
| Dependencies | B-2 (credential management) |

### B-2: No Mainnet Credential Management

| Attribute | Value |
|-----------|-------|
| Category | BLOCKER |
| Severity | Critical |
| Current State | Testnet credentials passed via environment variables in compose |
| Gap | Mainnet API keys require secure storage, rotation, and scoped access |
| Resolution | Integrate external secret manager (HashiCorp Vault, AWS Secrets Manager, or equivalent) |
| Effort Estimate | Medium |
| Dependencies | None |

### B-3: ClickHouse Backup/Restore Strategy

| Attribute | Value |
|-----------|-------|
| Category | **CLOSED** |
| Severity | High |
| Previous State | No backup/restore procedure |
| Resolution | Native `BACKUP TABLE`/`RESTORE TABLE` with bind-mount disk, automated proof (33/33 checks) |
| Resolved By | S435 |
| Evidence | [clickhouse-backup-restore-proof.md](clickhouse-backup-restore-proof.md), [clickhouse-recovery-runbook-rto-risks-and-limitations.md](clickhouse-recovery-runbook-rto-risks-and-limitations.md) |
| Residual | No automated schedule, no off-host replication (documented, acceptable for current topology) |

---

## 4. Non-Blockers Matrix

### NB-1: Rate Limiter for Venue API Calls

| Attribute | Value |
|-----------|-------|
| Category | NON-BLOCKER |
| Severity | Low |
| Current Mitigation | Testnet has lenient rate limits; single-symbol execution has low call frequency |
| Risk If Unaddressed | Mainnet rate limit violations could cause temporary API bans |
| Recommendation | Add token-bucket rate limiter as decorator in adapter stack before mainnet launch |

### NB-2: Per-Segment Kill Switch

| Attribute | Value |
|-----------|-------|
| Category | NON-BLOCKER |
| Severity | Low |
| Current Mitigation | Global kill switch halts all execution; segment isolation prevents cross-contamination |
| Risk If Unaddressed | Cannot halt one segment while keeping the other active |
| Recommendation | Extend EXECUTION_CONTROL KV to per-segment gate keys |

### NB-3: Per-Segment Idle Detection

| Attribute | Value |
|-----------|-------|
| Category | NON-BLOCKER |
| Severity | Low |
| Current Mitigation | Phase computation is cumulative; /statusz shows segment activity |
| Risk If Unaddressed | Segment that stops receiving events stays in "active" phase |
| Recommendation | Add recency-aware phase transition (active -> idle after configurable threshold) |

### NB-4: OTEL/Distributed Tracing

| Attribute | Value |
|-----------|-------|
| Category | NON-BLOCKER |
| Severity | Low |
| Current Mitigation | Structured JSON logging, correlation IDs, /diagz endpoint |
| Risk If Unaddressed | Harder to trace cross-binary event flow in production |
| Recommendation | Add OTEL instrumentation when operational complexity justifies |

### NB-5: Alerting Rules

| Attribute | Value |
|-----------|-------|
| Category | NON-BLOCKER |
| Severity | Low |
| Current Mitigation | Health signals exposed via /statusz and /diagz HTTP endpoints |
| Risk If Unaddressed | Operators must manually poll for degradation |
| Recommendation | Define alerting rules in operator's monitoring stack (Prometheus/Grafana) |

### NB-6: Pagination on Lifecycle List

| Attribute | Value |
|-----------|-------|
| Category | NON-BLOCKER |
| Severity | Low |
| Current Mitigation | Bounded cardinality (<100 partition keys in current deployment) |
| Risk If Unaddressed | Performance degradation at high cardinality |
| Recommendation | Add cursor-based pagination if cardinality exceeds 500 |

### NB-7: Rejection Code as ClickHouse Column

| Attribute | Value |
|-----------|-------|
| Category | NON-BLOCKER |
| Severity | Low |
| Current Mitigation | Queryable via `JSONExtractString(metadata, 'rejection_code')` |
| Risk If Unaddressed | Slower analytical queries on rejection codes |
| Recommendation | Promote to first-class column if rejection analytics become frequent |

### NB-8: Futures Commission from /fapi/v1/userTrades

| Attribute | Value |
|-----------|-------|
| Category | NON-BLOCKER |
| Severity | Low |
| Current Mitigation | Fee="0", CostBasis captures cumulative notional (S428 normalization) |
| Risk If Unaddressed | P&L calculations for Futures lack actual commission values |
| Recommendation | Add /fapi/v1/userTrades integration if P&L reporting requires actual commission |

### NB-9: Parallel Spot+Futures Live Execution Proof

| Attribute | Value |
|-----------|-------|
| Category | NON-BLOCKER |
| Severity | Low |
| Current Mitigation | Each segment proven independently on testnet; isolation proven structurally |
| Risk If Unaddressed | Unknown interaction under simultaneous dual-segment execution |
| Recommendation | Test under parallel load before mainnet dual-segment activation |

### NB-10: Documentation Index (97 Untracked Docs)

| Attribute | Value |
|-----------|-------|
| Category | NON-BLOCKER |
| Severity | Low |
| Current Mitigation | No runtime impact; docs serve as reference material |
| Risk If Unaddressed | Developer navigation friction |
| Recommendation | Separate documentation ceremony |

---

## 5. Accepted Risks Register

| ID | Risk | Severity | Mitigation | Accepted Since |
|----|------|----------|------------|----------------|
| RG-2 | Partial fill live observation limited by testnet | LOW | Structural proof sufficient; domain invariants cover all transitions | S409 |
| RG-8 | Endurance testing is synthetic (cycle-based, not wall-clock) | LOW | 2,000+ cycles proven stable; compose smoke phases cover multi-binary interaction | S414 |
| RG-9 | No time-based drift detection in endurance | LOW | Actor health tracker logs idle components; /statusz shows phase | S414 |
| RG-11 | Lifecycle list eventually consistent (<1s lag) | LOW | Operational queries tolerate sub-second lag; ClickHouse provides exact history | S413 |
| RG-15 | Single symbol at compose level | LOW | Multi-symbol structurally supported; compose proves single execution path | S420 |

---

## 6. Closed Gaps (Reference)

| ID | Gap | Closed By | Stage |
|----|-----|-----------|-------|
| RG-1 | ClickHouse rejection writer not wired | Rejection persistence pipeline | S411 |
| RG-3 | KV latest-only semantics | S430 audit decision (this document) | S430 |
| RG-5 | Commission asset not captured | FeeAsset field in FillRecord | S413 |
| RG-13 | Fee semantic divergence (Spot vs Futures) | Canonical Fee/FeeAsset/CostBasis model | S428 |

---

## 7. Mainnet Authorization Prerequisites

For a future mainnet authorization ceremony, the following must be true:

| Prerequisite | Status | Owner |
|-------------|--------|-------|
| B-1 resolved (mainnet adapters implemented) | Pending | Engineering |
| B-2 resolved (credential management integrated) | Pending | Engineering + Ops |
| B-3 resolved (ClickHouse backup strategy defined) | Pending | Ops |
| All accepted risks reviewed and reconfirmed | Pending | Architecture |
| Operational runbooks exist for kill-switch, degradation, restart | Pending | Ops |
| Dry-run execution on mainnet endpoints (with dry_run=true) | Pending | Engineering |
| Smoke test passes on mainnet-targeted compose | Pending | CI/CD |

---

## 8. Risk Severity Definitions

| Severity | Definition |
|----------|-----------|
| Critical | System cannot operate safely; data loss or financial loss possible |
| High | Major functionality gap; workaround exists but is fragile |
| Medium | Functionality limitation with documented mitigation |
| Low | Known limitation with adequate mitigation; acceptable for production |
| Informational | Design decision, not a risk |

---

## 9. Limitations

- This register reflects the state as of S430 (2026-03-23)
- Mainnet behavior may introduce risks not visible on testnet (rate limits, fill latency, error semantics)
- The register should be re-evaluated at each future wave that touches execution or persistence
- Operational risks (on-call, incident response, monitoring) are not covered here
