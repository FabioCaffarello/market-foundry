# CLI Risk Domain Guardrails

> Stage S63 — Governance activation for risk domain.
> Risk is governed but NOT implemented. Implementation begins in S64.

## Overview

The `raccoon-cli` enforces **10 guardrails** for the risk domain, matching the pattern established for signal (S40), decision (S44), and strategy (S54). Because risk is pre-implementation, guardrails split into two phases:

- **Phase 1 (S63, active now):** Design integrity and premature entry prevention.
- **Phase 2 (S64, prepared):** Full implementation drift detection.

## Phase 1 Guardrails (Active)

### RG-01: Design Documentation

All 7 risk architecture documents produced by S62 must exist:

| Document | Purpose |
|----------|---------|
| `risk-domain-design.md` | Full domain model, boundaries, binary placement |
| `risk-stream-families.md` | Family definitions (RF-01 Position Exposure), stream specs |
| `risk-activation-and-ownership.md` | Two-layer activation, ownership matrix |
| `risk-query-surface-guidelines.md` | Four-layer query architecture, endpoints |
| `risk-readiness-review.md` | Formal readiness assessment (S59) |
| `risk-entry-prerequisites.md` | Concrete conditions for domain opening |
| `risk-risks-and-blockers.md` | Gap analysis, blocking/non-blocking risks |

**Check:** `drift-detect` → `risk-docs-drift`

### RG-02: Premature Stream Entry

`RISK_EVENTS` is in the **PROHIBITED_STREAMS** list. Any appearance of this stream name in Go source triggers an error.

**Check:** `drift-detect` → `premature-domain-entry`

### RG-03: Premature Subject Entry

Any NATS subject starting with `risk.events.*` or `risk.query.*` in Go source triggers an error.

**Check:** `drift-detect` → `premature-domain-entry`

### RG-04: Premature Adapter Files

Risk adapter files must NOT exist before S64:
- `risk_registry.go`
- `risk_publisher.go`
- `risk_consumer.go`
- `risk_gateway.go`
- `risk_kv_store.go`

**Check:** `drift-detect` → `risk-premature-implementation`

### RG-05: Premature Domain Files

Risk domain and application files must NOT exist before S64:
- `internal/domain/risk/risk.go`
- `internal/domain/risk/events.go`
- `internal/application/risk/position_exposure_evaluator.go`
- `internal/application/riskclient/contracts.go`
- `internal/application/riskclient/get_latest_risk.go`
- `internal/application/ports/risk.go`

**Check:** `drift-detect` → `risk-premature-implementation`

### RG-06: Premature Actor Files

Risk actor files must NOT exist before S64:
- `internal/actors/scopes/derive/risk_evaluator_actor.go`
- `internal/actors/scopes/derive/risk_publisher_actor.go`
- `internal/actors/scopes/store/risk_consumer_actor.go`
- `internal/actors/scopes/store/risk_projection_actor.go`

**Check:** `drift-detect` → `risk-premature-implementation`

### RG-07: Premature HTTP Files

Risk HTTP interface files must NOT exist before S64:
- `internal/interfaces/http/handlers/risk.go`
- `internal/interfaces/http/routes/risk.go`

**Check:** `drift-detect` → `risk-premature-implementation`

### RG-08: Premature Config Entries

`risk_families` must NOT appear in `derive.jsonc` or `store.jsonc` before S64.

**Check:** `drift-detect` → `risk-premature-implementation`

### RG-09: Subject Classification

The runtime binding scanner classifies `risk.events.*` as publish subjects and `risk.query.*` as query subjects, ready for S64.

**Check:** `runtime-bindings` (prepared)

### RG-10: Coverage Map

The `domain-risk` sensitive area is registered in the coverage map with required dimensions: `architecture`, `contracts`, `drift`.

**Check:** `coverage-map` → `coverage:domain-risk`

## Phase 2 Guardrails (Prepared for S64)

When S64 formally opens risk implementation:

1. Remove `RISK_EVENTS` from `PROHIBITED_STREAMS`
2. Add `RISK_EVENTS` to `CANONICAL_STREAMS`
3. Add risk entries to `EXPECTED_STREAMS`, `EXPECTED_DURABLES`, `EXPECTED_QUERY_SUBJECTS`
4. Activate `check_risk_adapter_drift`, `check_risk_domain_drift`, `check_risk_config_drift`, `check_risk_contracts_drift` in `analyze()`
5. Replace `check_risk_premature_implementation` with the active drift checks

The prepared functions already exist in the codebase with `#[allow(dead_code)]` annotations.

## Expected Contracts (S62 Design)

| Artifact | Name | Purpose |
|----------|------|---------|
| Stream | `RISK_EVENTS` | 72h retention, deduplication enabled |
| Subject | `risk.events.position_exposure.assessed` | Publish: risk assessment events |
| Subject | `risk.query.position_exposure.latest` | Query: latest risk assessment |
| Durable | `store-risk-position-exposure` | Store consumer for projection |
| KV Bucket | `RISK_POSITION_EXPOSURE_LATEST` | Latest-only risk projection |

## CLI Commands

```bash
# Verify risk governance (included in standard drift-detect)
raccoon-cli drift-detect

# Full quality gate (includes risk checks)
raccoon-cli quality-gate --profile fast

# Coverage map showing domain-risk area
raccoon-cli coverage-map
```
