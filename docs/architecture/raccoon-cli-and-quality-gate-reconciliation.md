# Raccoon-CLI and Quality-Gate Reconciliation

**Stage:** S224
**Purpose:** Reconcile governance tooling with the post-S218/S219/S220 repository architecture.

## Scope

This reconciliation updates the governance tooling so that `raccoon-cli` and the `quality-gate` express the current `market-foundry` topology instead of the pre-restructure layout.

The scope is intentionally narrow:

- update obsolete structural assumptions in the analyzers
- preserve existing guard rails that still represent real architectural invariants
- avoid introducing new governance categories or relaxing legitimate checks
- document what changed and what intentionally did not change

## What Changed

### 1. Registry discovery now matches the real NATS adapter layout

Before S218, the tooling assumed legacy flat files such as:

- `internal/adapters/nats/signal_registry.go`
- `internal/adapters/nats/decision_registry.go`
- `internal/adapters/nats/execution_registry.go`

After S218, the canonical layout is sub-packaged:

- `internal/adapters/nats/natssignal/registry.go`
- `internal/adapters/nats/natsdecision/registry.go`
- `internal/adapters/nats/natsexecution/registry.go`

The scanner now accepts both:

- legacy `*_registry.go`
- current `*/registry.go`

This keeps the tooling compatible with the current repository without hard-coding the old flat surface.

### 2. Consumer discovery now understands `natskit.NewConsumerSpec(...)`

The old scanners depended on explicit `ConsumerSpec{ ... Durable: ... }` blocks.

The current codebase frequently declares consumers through:

```go
natskit.NewConsumerSpec("store-signal-rsi", "...", "...", "SIGNAL_EVENTS")
```

The scanners in:

- `contract-audit`
- `runtime-bindings`
- `topology-doctor`

now extract durable and stream bindings from factory calls as well as explicit struct literals.

### 3. Store-side consumer governance now follows the generic actor topology

Before S219, drift rules expected one store consumer actor per domain:

- `signal_consumer_actor.go`
- `decision_consumer_actor.go`
- `strategy_consumer_actor.go`
- `risk_consumer_actor.go`
- `execution_consumer_actor.go`

After S219, those wrappers were intentionally removed and replaced by:

- `internal/actors/scopes/store/generic_consumer_actor.go`
- `internal/actors/scopes/store/store_supervisor.go`

The drift checks now validate:

- generic consumer infrastructure is present
- domain-specific store wiring exists in `store_supervisor.go`

This preserves the original guard rail intent, but aligns it to the actual post-migration architecture.

### 4. Domain doc expectations now point to canonical surviving documents

Several drift checks still required documents that were consolidated away during the documentation cleanup waves. The reconciliation replaced those stale expectations with the canonical documents that remain active in `docs/architecture/`.

Examples:

- decision now tracks `decision-projection-pattern.md` and `decision-replay-idempotency-rules.md` instead of removed readiness-entry documents
- strategy now tracks `strategy-first-slice.md`, `strategy-projection-pattern.md`, and `strategy-replay-idempotency-rules.md`
- risk now tracks `risk-first-slice.md`, `risk-projection-pattern.md`, `risk-family-01-contracts.md`, and `risk-replay-idempotency-rules.md`
- execution now tracks the current active execution governance set, including fill, control, status, recovery, and operational validation documents

### 5. Operational tooling docs were updated to reflect the new topology

Minimum operational documentation was updated in:

- `tools/raccoon-cli/README.md`
- `docs/tooling/cli-architecture-guardrails.md`
- `docs/tooling/cli-topology-audit.md`

These docs now describe:

- NATS sub-packages under `internal/adapters/nats/`
- registry discovery across `*/registry.go`
- generic store consumer wiring
- the operational role of `execute` in the current service surface

## What Stayed Intentionally Unchanged

The reconciliation did **not**:

- add new guard-rail categories
- widen the topology checks into a new analytical governance program
- relax stream, durable, query, or layer invariants
- suppress real drift by downgrading legitimate errors into warnings

The existing checks still enforce:

- architecture layer boundaries
- stream and durable continuity
- contract completeness
- config and compose alignment
- documentation-to-code alignment

## Result

After the reconciliation, `make quality-gate` passes against the actual repository structure without requiring fake legacy files, deleted actor wrappers, or consolidated-away documents.
