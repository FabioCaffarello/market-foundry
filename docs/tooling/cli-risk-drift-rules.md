# CLI Risk Domain Drift Rules

> Stage S63 — Drift detection rules for risk domain governance.

## Rule Catalog

### RD-1: risk-docs-drift

**Phase:** Active (S63)
**Severity:** Error
**What:** Verifies all 7 risk architecture documents from S62 exist.
**Why:** Risk governance requires canonical design docs to prevent drift between architecture and future implementation.
**Files checked:**
- `docs/architecture/risk-domain-design.md`
- `docs/architecture/risk-stream-families.md`
- `docs/architecture/risk-activation-and-ownership.md`
- `docs/architecture/risk-query-surface-guidelines.md`
- `docs/architecture/risk-readiness-review.md`
- `docs/architecture/risk-entry-prerequisites.md`
- `docs/architecture/risk-risks-and-blockers.md`

### RD-2: risk-premature-implementation

**Phase:** Active (S63)
**Severity:** Error
**What:** Scans for risk implementation artifacts that must not exist before S64.
**Why:** Risk domain is under governance but implementation has not been formally approved.
**Scans for:**
- Adapter files: `risk_registry.go`, `risk_publisher.go`, `risk_consumer.go`, `risk_gateway.go`, `risk_kv_store.go`
- Domain files: `internal/domain/risk/`, `internal/application/risk/`, `internal/application/riskclient/`, `internal/application/ports/risk.go`
- Actor files: `risk_evaluator_actor.go`, `risk_publisher_actor.go`, `risk_consumer_actor.go`, `risk_projection_actor.go`
- HTTP files: `handlers/risk.go`, `routes/risk.go`
- Config entries: `risk_families` in `derive.jsonc` or `store.jsonc`

### RD-3: risk-adapter-drift (Prepared for S64)

**Phase:** Prepared, not active
**Severity:** Error (when activated)
**What:** Verifies 5 NATS adapter files exist for risk domain.
**Activates when:** S64 removes `RISK_EVENTS` from `PROHIBITED_STREAMS`.

### RD-4: risk-domain-drift (Prepared for S64)

**Phase:** Prepared, not active
**Severity:** Error (when activated)
**What:** Verifies risk domain entity, events, application, actors, and HTTP files exist.
**Activates when:** S64 removes `RISK_EVENTS` from `PROHIBITED_STREAMS`.

### RD-5: risk-config-drift (Prepared for S64)

**Phase:** Prepared, not active
**Severity:** Error/Warning (when activated)
**What:** Verifies symmetric `risk_families` configuration between `derive.jsonc` and `store.jsonc`.
**Activates when:** S64 removes `RISK_EVENTS` from `PROHIBITED_STREAMS`.

### RD-6: risk-contracts-drift (Prepared for S64)

**Phase:** Prepared, not active
**Severity:** Error (when activated)
**What:** Verifies risk subjects, durable consumers, and KV bucket names exist in Go source.
**Expected contracts:**
- Subject: `risk.events.position_exposure.assessed`
- Subject: `risk.query.position_exposure.latest`
- Durable: `store-risk-position-exposure`
- Bucket: `RISK_POSITION_EXPOSURE_LATEST`
**Activates when:** S64 removes `RISK_EVENTS` from `PROHIBITED_STREAMS`.

## Premature Domain Entry (Inherited)

The base drift rules already include `RISK_EVENTS` in the prohibited streams list. This provides a second layer of protection:

- **Stream guard:** `RISK_EVENTS` stream name must not appear in Go source
- **Subject guard:** `risk.events.*` and `risk.query.*` subjects must not appear in Go source

## S64 Activation Checklist

When opening risk implementation in S64, update `drift_detect.rs`:

1. Remove `("RISK_EVENTS", "risk domain governance is active...")` from `PROHIBITED_STREAMS`
2. Add `"RISK_EVENTS"` to `CANONICAL_STREAMS`
3. Add to `EXPECTED_STREAMS`: `("RISK_EVENTS", "carries risk assessments (position exposure) from derive")`
4. Add to `EXPECTED_DURABLES`: `("store-risk-position-exposure", "RISK_EVENTS", "store consumes position exposure risk events for projection")`
5. Add to `EXPECTED_QUERY_SUBJECTS`: `("risk.query.position_exposure.latest", "store serves latest risk assessment queries from gateway")`
6. In `analyze()`: replace `check_risk_premature_implementation` with `check_risk_adapter_drift`, `check_risk_domain_drift`, `check_risk_config_drift`, `check_risk_contracts_drift`
7. Remove `#[allow(dead_code)]` from the four prepared functions

Also update `runtime_bindings.rs`:
1. Add `RISK_EVENTS` to `EXPECTED_STREAMS`
2. Add `store-risk-position-exposure` to `EXPECTED_DURABLES`
3. Add `risk.query.position_exposure.latest` to `EXPECTED_QUERY_SUBJECTS`
4. Add risk adapter file checks to `check_adapter_files`
