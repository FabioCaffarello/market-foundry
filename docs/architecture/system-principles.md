# System Principles — Market Foundry

> Canonical document. Defines the non-negotiable principles governing Market Foundry's architecture and evolution.
> Approved: 2026-03-16. These principles are binding on all contributors and agents operating within this repository.

---

## Foundational Principles

These principles are **inviolable**. They cannot be overridden by convenience, deadline pressure, or scope creep.

---

### 1. Layer Sovereignty

**Dependencies flow inward. Never outward. Never sideways.**

```
domain → application → adapters → actors → interfaces → cmd
```

- `domain/` has zero imports from any other layer.
- `application/` depends only on `domain/` and `shared/`.
- `adapters/` implement interfaces defined in `application/ports/`.
- `actors/` orchestrate adapters and application use cases.
- `interfaces/` translate external protocols to application calls.
- `cmd/` wires everything together and starts the process.

A layer may never import from a layer above it. This is enforced by `raccoon-cli arch-guard` and verified in CI. Violations are build-breaking errors, not warnings.

**Why this matters:** Layer sovereignty prevents structural contamination — the exact problem that forced the quality-service sanitization. Without it, domain logic becomes entangled with infrastructure, making evolution impossible without full rewrites.

---

### 2. Domain Module Isolation

**Each domain is a self-contained vertical slice. Domains do not import each other.**

A domain module consists of:
- `internal/domain/{name}/` — Pure business types, events, invariants
- `internal/application/{name}/` — Use cases consuming domain types
- `internal/adapters/{transport}/{name}_*.go` — Transport-specific implementations
- `internal/actors/scopes/{name}/` — Actor topology for this domain
- `cmd/{name}/` — Entry point and wiring

Cross-domain communication happens **exclusively through messages** (NATS subjects). One domain never calls another domain's application layer directly.

**Why this matters:** Isolation ensures that adding, removing, or rewriting a domain module does not destabilize others. This is the foundation of the "foundry" concept — domains are independently forged.

---

### 3. Messages as Boundaries

**All inter-service and inter-domain communication passes through the message bus.**

- Synchronous queries use NATS request/reply.
- Asynchronous events use JetStream publishing.
- All messages use the envelope pattern with correlation IDs.
- Message schemas are contracts — changes require explicit versioning.

Direct function calls between services or domains are prohibited. HTTP is a user-facing boundary, not an internal communication mechanism.

**Why this matters:** Message-based boundaries make the system observable, debuggable, and evolvable. They also enable independent scaling and deployment of domain modules.

---

### 4. Actors Own Lifecycle

**Every runtime behavior is expressed as an actor. Actors own their own lifecycle.**

- Services are composed of actor trees managed by supervisors.
- Supervisors handle spawning, restarting, and graceful shutdown.
- No goroutines are spawned outside actor supervision.
- The Hollywood framework is the sole concurrency primitive.

**Why this matters:** Unsupervised goroutines are the source of resource leaks, race conditions, and ungraceful shutdowns. The actor model provides structural guarantees that goroutines alone cannot.

---

### 5. Configuration as Domain Object

**Configuration documents have a lifecycle. They are not static files loaded at boot.**

The configctl service manages configurations through explicit states:
```
Draft → Validated → Compiled → Active → Deactivated → Archived
```

Runtime projections and ingestion bindings are derived from compiled configurations, not from ad-hoc environment variables or config maps.

**Why this matters:** Treating configuration as a domain object with state transitions makes the system auditable, reproducible, and safe to modify at runtime.

---

### 6. Static Enforcement Over Convention

**Architectural rules are enforced by tooling, not by documentation alone.**

raccoon-cli provides automated enforcement:
- `arch-guard` validates layer dependencies
- `contract-audit` validates messaging contracts
- `drift-detect` identifies semantic drift between layers
- `quality-gate` aggregates checks into pass/fail profiles

Quality gates must pass before code is merged. A rule that exists only in documentation is not a rule — it is a suggestion.

**Why this matters:** Conventions degrade over time. Automated enforcement does not. The sanitization from quality-service was necessary precisely because conventions were followed inconsistently.

---

### 7. No Premature Domain Implementation

**A domain module is introduced only when its boundaries, invariants, and contracts are fully defined.**

Before any domain code is written:
1. Its domain types must be specified (entities, value objects, events)
2. Its application use cases must be enumerated
3. Its messaging contracts must be declared
4. Its actor topology must be designed
5. Its quality gate profile must be configured

Exploratory code goes in branches or prototypes — never in `main`.

**Why this matters:** Premature implementation creates technical debt that is indistinguishable from intentional architecture. The sanitization proved that removing debt later is orders of magnitude more expensive than preventing it.

---

### 8. Identity Integrity

**Market Foundry is Market Foundry. It is not a renamed quality-service.**

- No references to quality-service, quality-pipeline, or validation-runtime may exist in code, configuration, documentation, or commit messages.
- Docker images, networks, volumes, and config paths use the `market-foundry` namespace exclusively.
- New domain modules use market-domain vocabulary, not quality-domain vocabulary.

**Why this matters:** Identity contamination was the original disease. The cure is eternal vigilance.

---

## Operational Principles

These principles govern day-to-day development practices.

---

### 9. Module-per-Layer Topology

**Each architectural layer is its own Go module with an explicit `go.mod`.**

The `go.work` workspace coordinates modules. This ensures:
- Import paths are explicit and auditable
- Dependency graphs are visible per layer
- Circular imports are structurally impossible across modules

New modules are added to `go.work` only after passing `raccoon-cli doctor`.

---

### 10. Contract-First Messaging

**Message schemas are defined before their producers and consumers are implemented.**

For every NATS subject:
1. Define the message type in the appropriate domain or application package
2. Define the envelope structure
3. Register the contract with `contract-audit`
4. Then implement producers and consumers

---

### 11. Test Proximity

**Tests live next to the code they test. Integration tests live in `tests/`.**

- Unit tests: `*_test.go` in the same package
- Integration tests: `tests/` directory with HTTP or messaging scenarios
- No test utilities in production packages
- No mocking frameworks — use interface-based test doubles

---

### 12. Minimal External Dependencies

**Every external dependency must justify its presence.**

- NATS is the messaging backbone — justified and committed.
- Hollywood is the actor framework — justified and committed.
- Standard library is preferred over third-party packages for HTTP, JSON, testing.
- New dependencies require explicit justification and review.

---

## Anti-Patterns — Explicitly Prohibited

The following patterns are **banned** in Market Foundry:

| Anti-pattern                         | Why it is prohibited                                                |
|--------------------------------------|---------------------------------------------------------------------|
| Cross-layer imports                  | Destroys layer sovereignty; leads to structural contamination       |
| Direct domain-to-domain calls        | Bypasses message bus; creates hidden coupling                       |
| Unsupervised goroutines              | Uncontrollable lifecycle; resource leaks; ungraceful shutdown       |
| Environment-variable-driven behavior | Configuration must flow through configctl lifecycle                 |
| Copy-paste from MarketMonkey         | Patterns must be re-implemented natively; copied code carries assumptions |
| Copy-paste from quality-service      | The predecessor is dead; its code is not a template                 |
| Shared mutable state between actors  | Violates actor isolation; source of race conditions                 |
| Catch-all error handling             | Errors must be typed and propagated through the problem system      |
| Feature flags for incomplete domains | Ship nothing or ship complete; no half-states in `main`             |
| God actors (single actor doing everything) | Violates single-responsibility; makes supervision meaningless  |
| Kafka or any secondary message broker| NATS is the sole messaging infrastructure                           |

---

## Principle Hierarchy

When principles conflict, resolve using this precedence (highest first):

1. **Layer sovereignty** — structural integrity is non-negotiable
2. **Domain isolation** — coupling between domains is never acceptable
3. **Message boundaries** — all communication through the bus
4. **Actor lifecycle** — no unmanaged concurrency
5. **Static enforcement** — if it's not enforced, it's not real
6. **Identity integrity** — no legacy contamination

---

## Governance

These principles are maintained alongside the codebase. Changes to this document require:
1. A clear rationale for why the change is necessary
2. An impact analysis on existing code
3. Verification that raccoon-cli enforcement rules are updated accordingly

Principles are not aspirational. They are descriptive of how the system **must** operate.
