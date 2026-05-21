# CLI Execution Domain Drift Rules

> Stage S70 — Updated S86. Drift detection rules for execution domain governance.

## Rule Catalog

### ED-1: execution-docs-drift

**Phase:** Active (S70, updated S86)
**Severity:** Error
**What:** Verifies all 14 execution architecture documents exist.
**Why:** Execution governance requires canonical design docs to prevent drift between architecture and implementation.
**Files checked:**
- `docs/architecture/execution-domain-design.md`
- `docs/architecture/execution-stream-families.md`
- `docs/architecture/execution-activation-and-ownership.md`
- `docs/architecture/execution-query-surface-guidelines.md`
- `docs/architecture/execution-readiness-review.md`
- `docs/architecture/execution-entry-prerequisites.md`
- `docs/architecture/execution-risks-and-blockers.md`
- `docs/architecture/execute-runtime-and-activation-model.md`
- `docs/architecture/execute-governance-and-activation-model.md`
- `docs/architecture/execution-family-separation-after-paper-step.md` (added S85)
- `docs/architecture/venue-routing-and-ownership-split.md` (added S85)
- `docs/architecture/post-paper-action-boundary-readiness-review.md` (added S86)
- `docs/architecture/post-paper-risks-and-blockers.md` (added S86)
- `docs/architecture/next-frontier-entry-prerequisites.md` (added S86)

### ED-2: execution-premature-implementation

**Phase:** REMOVED (S83)
**Reason:** Execution domain is fully implemented (S71-S82). The premature guard produced false positives and was dead code. Replaced by active drift checks ED-3 through ED-6.

### ED-3: execution-adapter-drift

**Phase:** Active (S71, updated S83)
**Severity:** Error
**What:** Verifies 7 NATS adapter files exist for execution domain.
**Files:**
- `execution_registry.go` — stream, consumer, and query specs
- `execution_publisher.go` — publishes execution events and fill events
- `execution_consumer.go` — durable consumer for execution events
- `execution_gateway.go` — gateway adapter for request/reply queries
- `execution_kv_store.go` — KV bucket store for latest execution projections
- `execution_control_gateway.go` — gateway adapter for control gate request/reply
- `execution_control_kv_store.go` — KV bucket store for execution control gate

### ED-4: execution-domain-drift

**Phase:** Active (S71, updated S83)
**Severity:** Error
**What:** Verifies execution domain entity, events, control, application, actors, and HTTP files exist.
**Domain/application files (13):**
- `internal/domain/execution/execution.go`
- `internal/domain/execution/events.go`
- `internal/domain/execution/control.go`
- `internal/application/execution/paper_order_evaluator.go`
- `internal/application/execution/paper_venue_adapter.go`
- `internal/application/execution/staleness_guard.go`
- `internal/application/executionclient/contracts.go`
- `internal/application/executionclient/control_contracts.go`
- `internal/application/executionclient/get_latest_execution.go`
- `internal/application/executionclient/get_execution_status.go`
- `internal/application/executionclient/get_execution_control.go`
- `internal/application/ports/execution.go`
- `internal/application/ports/venue.go`

**Actor files (6):**
- `internal/actors/scopes/derive/execution_evaluator_actor.go`
- `internal/actors/scopes/derive/execution_publisher_actor.go`
- `internal/actors/scopes/store/execution_consumer_actor.go`
- `internal/actors/scopes/store/execution_projection_actor.go`
- `internal/actors/scopes/execute/execute_supervisor.go`
- `internal/actors/scopes/execute/venue_adapter_actor.go`

**HTTP files (2):**
- `internal/interfaces/http/handlers/execution.go`
- `internal/interfaces/http/routes/execution.go`

### ED-5: execution-config-drift

**Phase:** Active (S71, updated S83)
**Severity:** Error/Warning
**What:** Verifies symmetric `execution_families` configuration between `derive.jsonc`, `store.jsonc`, and `execute.jsonc`. Also verifies venue config presence in `execute.jsonc`.

### ED-6: execution-contracts-drift

**Phase:** Active (S71, updated S83)
**Severity:** Error
**What:** Verifies execution subjects, durable consumers, and KV bucket names exist in Go source.
**Expected contracts:**
- Subject: `execution.events.paper_order.submitted`
- Subject: `execution.query.paper_order.latest`
- Subject: `execution.fill.venue_market_order`
- Subject: `execution.query.status.latest`
- Subject: `execution.control.get`
- Subject: `execution.control.set`
- Durable: `store-execution-paper-order` (paper family)
- Durable: `execute-venue-market-order-intake` (venue family intake — transitional bridge)
- Durable: `store-execution-venue-market-order-fill` (venue family fill)
- Bucket: `EXECUTION_PAPER_ORDER_LATEST`
- Bucket: `EXECUTION_VENUE_MARKET_ORDER_LATEST`
- Bucket: `EXECUTION_CONTROL`

## Binary Drift

With `execute` added to `APP_BINARIES` (S83), the existing binary drift checks now verify:
- `cmd/execute/` directory exists
- Compose service `execute` exists (warning if absent)

## Comparison with Risk Governance

| Aspect | Risk | Execution |
|--------|------|-----------|
| Design stage | S62 | S69 |
| Governance stage | S63 | S70 |
| Implementation stage | S64 | S71-S82 |
| Execute binary governance | N/A | S83 |
| Streams | RISK_EVENTS | EXECUTION_EVENTS, EXECUTION_FILL_EVENTS |
| Adapter files | 5 | 7 |
| Domain files | 6 | 13 |
| Actor scopes | 2 (derive, store) | 3 (derive, store, execute) |
| Subjects tracked | 2 | 6 |
| Durable consumers | 1 | 3 |
| KV buckets | 1 | 3 |
