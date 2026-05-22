# ADR 0005: Layer sovereignty enforced statically

## Status

Accepted.

## Context

A common failure mode in long-lived codebases is **layer violation**:
business logic ends up depending on HTTP frameworks, domain types
import database drivers, infrastructure concerns leak into pure
modeling code.

Once these dependencies invert, they're hard to undo. The codebase
becomes tangled, refactoring becomes risky, and substituting one
implementation for another (e.g., switching from Kafka to NATS, or
PostgreSQL to ClickHouse) becomes a project rather than a config change.

market-foundry adopts the **clean architecture** discipline of strict
inward-only dependencies. The system has 6 layers:

```
domain → application → adapters → actors → interfaces → cmd
```

- **domain**: pure business types and rules, no I/O
- **application**: use cases, depending only on domain
- **adapters**: implementations of ports (NATS, ClickHouse, exchanges)
- **actors**: actor-based runtime, orchestrating use cases
- **interfaces**: HTTP handlers, gateway-level translation
- **cmd**: composition root for each binary

Imports flow outward only. A domain type cannot import a NATS adapter.
An application use case cannot import an HTTP handler. The cmd layer
sees everything; the domain layer sees only itself.

## Decision

**Layer sovereignty is enforced statically by raccoon-cli, not by
convention.** Every PR is checked, every CI run validates, every
local `make check` runs the layer guard.

A violating import does not ship.

## Consequences

### Positive

- **Substitutability**: implementations under `adapters/` can be
  swapped (e.g., add a new exchange adapter, switch HTTP framework)
  without touching domain or application.
- **Testability**: domain logic is pure and unit-testable without
  mocks. Application use cases test against domain types and
  application-level interfaces, not against NATS or ClickHouse.
- **Cognitive clarity**: when reading a file, you know its
  responsibility level from its directory. `internal/domain/` is
  business logic only; you don't expect to find HTTP code there.
- **No accidental coupling**: a junior contributor cannot accidentally
  introduce a layer violation; raccoon-cli catches it.
- **Refactoring safety**: structural refactors (e.g., moving a use
  case from one binary to another) don't require re-checking that
  layering is correct; it's continuously verified.

### Negative

- **Indirection cost**: pure layering adds indirection. A simple
  operation may pass through domain → application → adapter →
  interface, requiring 4 files instead of 1.
- **Boilerplate**: ports (interfaces in application that adapters
  implement) add to file count. For very simple operations, this
  feels disproportionate.
- **Learning curve**: contributors unfamiliar with clean architecture
  may initially struggle to know where new code goes.
- **Premature abstraction risk**: tempting to define ports for things
  that have only one implementation. This is countered by the principle
  "explicit duplication over premature abstraction" — interfaces with
  a single implementation are debt, not architecture.

## Alternatives considered

**Convention-only layering**: tried in the previous evolution model.
500+ stages accumulated cross-layer violations because no automated
check caught them. The retroactive cleanup would have been substantial.

**Lighter layering (e.g., 3 layers instead of 6)**: considered but
rejected because the domains and adapters genuinely have different
concerns (domain = business types, application = use cases, adapters
= infrastructure connectors). Collapsing them loses the distinction.

**Vertical slices (organize by feature, not layer)**: incompatible
with the chosen architecture style. Stream-mesh-as-architecture
requires strong horizontal layering to enable per-binary
deployment.

## References

- `internal/{domain,application,adapters,actors,interfaces}/` — the
  6 layers in code
- `cmd/{configctl,derive,execute,gateway,ingest,migrate,store,writer}/` —
  composition roots
- `tools/raccoon-cli/` → arch-guard rule set
- Makefile: `make arch-guard`
- [`../ARCHITECTURE.md`](../ARCHITECTURE.md) → "Foundational principles"
  → "Layer sovereignty"
- ADR [0004](0004-raccoon-cli-static-enforcement.md) — the tool that
  enforces this
