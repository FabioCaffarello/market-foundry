# CLI Execution Domain Guardrails

> Stage S70 — Governance activation for execution domain.
> Execution is governed but NOT implemented. Implementation begins in S75.

## Overview

The `raccoon-cli` enforces **10 guardrails** for the execution domain, matching the pattern established for signal (S40), decision (S44), strategy (S54), and risk (S63). Because execution is pre-implementation, guardrails split into two phases:

- **Phase 1 (S70, active now):** Design integrity and premature entry prevention.
- **Phase 2 (S75, prepared):** Full implementation drift detection.

## Phase 1 Guardrails (Active)

### EG-01: Design Documentation

All 7 execution architecture documents produced by S69 must exist:

| Document | Purpose |
|----------|---------|
| `execution-domain-design.md` | Full domain model, boundaries, binary placement |
| `execution-stream-families.md` | Family definitions (EF-01 Paper Order), stream specs |
| `execution-activation-and-ownership.md` | Two-layer activation, ownership matrix |
| `execution-query-surface-guidelines.md` | Four-layer query architecture, endpoints |
| `execution-readiness-review.md` | Formal readiness assessment (S68) |
| `execution-entry-prerequisites.md` | Concrete conditions for domain opening |
| `execution-risks-and-blockers.md` | Gap analysis, blocking/non-blocking risks |

**Check:** `drift-detect` -> `execution-docs-drift`

### EG-02: Premature Stream Entry

`EXECUTION_EVENTS` is in the **PROHIBITED_STREAMS** list. Any appearance of this stream name in Go source triggers an error.

**Check:** `drift-detect` -> `premature-domain-entry`

### EG-03: Premature Subject Entry

Any NATS subject starting with `execution.events.*` or `execution.query.*` in Go source triggers an error.

**Check:** `drift-detect` -> `premature-domain-entry`

### EG-04: Premature Adapter Files

Execution adapter files must NOT exist before S75:
- `execution_registry.go`
- `execution_publisher.go`
- `execution_consumer.go`
- `execution_gateway.go`
- `execution_kv_store.go`

**Check:** `drift-detect` -> `execution-premature-implementation`

### EG-05: Premature Domain Files

Execution domain and application files must NOT exist before S75:
- `internal/domain/execution/execution.go`
- `internal/domain/execution/events.go`
- `internal/application/execution/paper_order_evaluator.go`
- `internal/application/executionclient/contracts.go`
- `internal/application/executionclient/get_latest_execution.go`
- `internal/application/ports/execution.go`

**Check:** `drift-detect` -> `execution-premature-implementation`

### EG-06: Premature Actor Files

Execution actor files must NOT exist before S75:
- `internal/actors/scopes/derive/execution_evaluator_actor.go`
- `internal/actors/scopes/derive/execution_publisher_actor.go`
- `internal/actors/scopes/store/execution_consumer_actor.go`
- `internal/actors/scopes/store/execution_projection_actor.go`

**Check:** `drift-detect` -> `execution-premature-implementation`

### EG-07: Premature HTTP Files

Execution HTTP interface files must NOT exist before S75:
- `internal/interfaces/http/handlers/execution.go`
- `internal/interfaces/http/routes/execution.go`

**Check:** `drift-detect` -> `execution-premature-implementation`

### EG-08: Premature Config Entries

`execution_families` must NOT appear in `derive.jsonc` or `store.jsonc` before S75.

**Check:** `drift-detect` -> `execution-premature-implementation`

### EG-09: Subject Classification

The runtime binding scanner classifies `execution.events.*` as publish subjects and `execution.query.*` as query subjects, ready for S75.

**Check:** `runtime-bindings` (prepared)

### EG-10: Coverage Map

The `domain-execution` sensitive area is registered in the coverage map with required dimensions: `architecture`, `contracts`, `drift`.

**Check:** `coverage-map` -> `coverage:domain-execution`

## Phase 2 Guardrails (Prepared for S75)

When S75 formally opens execution implementation:

1. Remove `EXECUTION_EVENTS` from `PROHIBITED_STREAMS`
2. Add `EXECUTION_EVENTS` to `CANONICAL_STREAMS`
3. Activate `check_execution_adapter_drift`, `check_execution_domain_drift`, `check_execution_config_drift`, `check_execution_contracts_drift` in `analyze()`
4. Replace `check_execution_premature_implementation` with the active drift checks

The prepared functions already exist in the codebase with `#[allow(dead_code)]` annotations.

## Expected Contracts (S69 Design)

| Artifact | Name | Purpose |
|----------|------|---------|
| Stream | `EXECUTION_EVENTS` | 72h retention, deduplication enabled |
| Subject | `execution.events.paper_order.submitted` | Publish: paper order intent events |
| Subject | `execution.query.paper_order.latest` | Query: latest paper order intent |
| Durable | `store-execution-paper-order` | Store consumer for projection |
| KV Bucket | `EXECUTION_PAPER_ORDER_LATEST` | Latest-only execution intent projection |

## Boundary Invariants Protected

The following S69 boundary invariants are enforceable by the CLI:

| ID | Invariant | CLI Check |
|----|-----------|-----------|
| EBI-1 | No cross-domain imports | `arch-guard` (layer dependency) |
| EBI-2 | Evaluators are pure functions | Manual review (no AST body analysis yet) |
| EBI-4 | Only derive publishes to EXECUTION_EVENTS | `execution-premature-implementation` (pre-S75) |
| EBI-5 | Only store materializes projections | `execution-premature-implementation` (pre-S75) |
| EBI-7 | No external venue API interaction | Manual review |

## CLI Commands

```bash
# Verify execution governance (included in standard drift-detect)
raccoon-cli drift-detect

# Full quality gate (includes execution checks)
raccoon-cli quality-gate --profile fast

# Coverage map showing domain-execution area
raccoon-cli coverage-map
```
