# ADR 0006: configctl as single lifecycle authority

## Status

Accepted.

## Context

market-foundry's configuration documents describe runtime behavior:
which symbols to observe, which evidence to derive, which signals to
compute, which strategies to evaluate, which risks to apply, which
executions to wire. They progress through a defined lifecycle:

```
Draft → Validated → Compiled → Active → Inactive → Archived
```

(plus `Rejected` as terminal alternative)

Multiple binaries depend on configuration state:
- `ingest` reads activation bindings to know which symbols to observe.
- `derive` reads bindings to know which evidence/signal/decision/
  strategy/risk to compute per partition.
- `execute` reads execution-related config to know which venues to
  contact and in which mode.

If any of these were to transition config state independently — even
just validation — the system would have multiple sources of truth.

## Decision

**configctl is the single authority over configuration lifecycle
state.** No other binary may transition state. Other binaries react to
**events** that configctl publishes when transitions happen.

Specifically:
- Only configctl publishes `CONFIGCTL_EVENTS`.
- Other binaries consume `CONFIGCTL_EVENTS` via durable binding-watcher
  consumers but never publish.
- HTTP endpoints for transitions (POST `/configctl/configs/validate`,
  `/configctl/config-versions/:id/compile`, `/activate`) are served by
  the gateway but always delegate to configctl over NATS request/reply
  or NATS publish.

## Consequences

### Positive

- **Single source of truth**: at any moment, configctl's state for
  a config version is *the* state. No reconciliation needed across
  binaries.
- **Centralized validation**: validation rules live in configctl;
  consumers can trust events they receive.
- **Auditable**: lifecycle transitions form a deterministic event
  trail in `CONFIGCTL_EVENTS`. Replaying the stream reconstructs
  all state changes.
- **Simpler ingest/derive/execute**: they only need to react to
  events, not implement their own validation logic.

### Negative

- **configctl becomes a critical dependency**: every binary needs
  configctl available at startup. Mitigated by `readyz` healthchecks
  that block downstream services until configctl is ready.
- **Transition latency**: a config activation requires
  validation + compilation + activation events to propagate.
  Acceptable for the operational tempo (config changes are rare).
- **No local overrides**: a binary cannot run with a "local override"
  config without going through configctl. This is by design but can
  feel restrictive during development. Mitigated by config variants
  in `deploy/configs/`.

## Alternatives considered

**Each binary validates its own config**: rejected because it creates
N validation surfaces that must be kept in sync. The previous evolution
model showed how this drifts over time.

**Distributed consensus on config state (Raft/etcd)**: rejected as
overkill. Configuration changes are infrequent and operator-driven;
they don't require distributed consensus.

**No centralized lifecycle (each binary picks up config files
directly)**: rejected because it loses the audit trail and forces
manual coordination of "everyone has the same version".

## References

- `internal/domain/configctl/` — lifecycle types and rules
- `internal/application/configctl/` — use cases for each transition
- `internal/adapters/nats/natsconfigctl/` — stream + binding-watcher
  consumer specs
- [`../domain/configctl.md`](../domain/configctl.md) — domain deep dive
- [`../HTTP-API.md`](../HTTP-API.md) → configctl endpoints
- ADR [0001](0001-nats-not-kafka.md) — messaging substrate that makes
  this practical
