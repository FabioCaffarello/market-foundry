# Decision Domain Drift Detection Rules

Raccoon-CLI enforces five decision-specific drift detection rules (DD-1 through DD-5), analogous to the signal domain rules (SD-1 through SD-5). These checks run as part of `raccoon-cli drift-detect` and the `quality-gate` profiles (fast/ci/deep).

## DD-1: Decision Documentation Completeness

**Check name:** `decision-docs-drift`

Verifies that all required decision architecture documents exist:

| Document | Purpose |
|----------|---------|
| `docs/architecture/decision-domain-design.md` | Canonical domain design and boundary invariants |
| `docs/architecture/decision-first-slice.md` | First slice implementation (RSI oversold) |
| `docs/architecture/decision-stream-families.md` | Stream family definitions for decision |
| `docs/architecture/decision-activation-and-ownership.md` | Activation model and ownership matrix |
| `docs/architecture/decision-query-surface-guidelines.md` | Query chain and HTTP surface rules |
| `docs/architecture/decision-family-01-contracts.md` | RSI oversold family contracts |
| `docs/architecture/decision-readiness-review.md` | Pre-entry readiness assessment |
| `docs/architecture/decision-entry-prerequisites.md` | Entry prerequisites checklist |

**Severity:** ERROR if any doc is missing.

## DD-2: Decision Adapter File Presence

**Check name:** `decision-adapter-drift`

Verifies that all expected NATS adapter files exist in `internal/adapters/nats/`:

| File | Role |
|------|------|
| `decision_registry.go` | Stream, consumer, and query spec definitions |
| `decision_publisher.go` | Publishes to DECISION_EVENTS |
| `decision_consumer.go` | Durable consumer for store |
| `decision_gateway.go` | NATS request/reply for gateway queries |
| `decision_kv_store.go` | KV bucket materialization |

**Severity:** ERROR if any adapter file is missing.

## DD-3: Decision Domain and Application Layer

**Check name:** `decision-domain-drift`

Verifies structural completeness across three sublayers:

**Domain/Application files:**
- `internal/domain/decision/decision.go` — entity with Validate(), PartitionKey(), DeduplicationKey()
- `internal/domain/decision/events.go` — DecisionEvaluatedEvent
- `internal/application/decision/rsi_oversold_evaluator.go` — RSI oversold evaluator
- `internal/application/decisionclient/contracts.go` — query/reply contracts
- `internal/application/decisionclient/get_latest_decision.go` — GetLatestDecision use case
- `internal/application/ports/decision.go` — DecisionGateway port interface

**Actor scope files:**
- `internal/actors/scopes/derive/decision_evaluator_actor.go`
- `internal/actors/scopes/derive/decision_publisher_actor.go`
- `internal/actors/scopes/store/decision_consumer_actor.go`
- `internal/actors/scopes/store/decision_projection_actor.go`

**HTTP interface files:**
- `internal/interfaces/http/handlers/decision.go`
- `internal/interfaces/http/routes/decision.go`

**Severity:** ERROR if any file is missing.

## DD-4: Decision Configuration Symmetry

**Check name:** `decision-config-drift`

Verifies that `pipeline.decision_families` is declared symmetrically in both `derive.jsonc` and `store.jsonc`.

| State | Severity | Meaning |
|-------|----------|---------|
| Both present | INFO | Pipeline is correctly wired |
| derive only | ERROR | Events produced but never consumed |
| store only | ERROR | Consumer idles, no events produced |
| Neither | WARNING | Pipeline inactive (safe but no processing) |

## DD-5: Decision Runtime Contracts

**Check name:** `decision-contracts-drift`

Scans Go source for expected runtime contracts:

**Subjects:**
- `decision.events.rsi_oversold.evaluated` — event subject for RSI oversold decisions
- `decision.query.rsi_oversold.latest` — query subject for latest decision

**Durable consumers:**
- `store-decision-rsi-oversold` — store consumer on DECISION_EVENTS

**KV buckets:**
- `DECISION_RSI_OVERSOLD_LATEST` — latest finalized decision per partition key

**Severity:** ERROR if any contract is missing in source.

## Relationship to Other Checks

These five checks complement the base drift rules (Rules 1–7) and the signal-specific rules (SD-1 through SD-5). Together they ensure three governance layers:

1. **Base rules** — config/compose/binary/docs/actor/stream alignment
2. **Signal rules** — signal domain structural and contract integrity
3. **Decision rules** — decision domain structural and contract integrity

The decision checks also integrate with `runtime-bindings`, which now validates `DECISION_EVENTS` as a canonical stream.
