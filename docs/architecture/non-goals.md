# Non-Goals — Market Foundry

> Canonical document. Defines what Market Foundry explicitly does not aim to be, do, or become.
> Approved: 2026-03-16. These non-goals are as binding as the system principles.

---

## Purpose of This Document

Non-goals are not deferred goals. They are **deliberate exclusions** — things the system will not do even if it technically could. They exist to prevent scope creep, identity drift, and architectural contamination.

Every non-goal includes the reasoning behind its exclusion. If the reasoning no longer holds, the non-goal may be revisited — but the default answer is "no."

---

## Architectural Non-Goals

### NG-1: Market Foundry is not a quality service

Market Foundry does not validate, score, audit, or certify data quality. It does not maintain validation runtimes, quality results stores, or compliance pipelines.

**Rationale:** The system was born from the sanitization of a quality service. Reintroducing quality-domain concepts — even under new names — would recreate the structural contamination that necessitated the rewrite.

**Boundary:** If market-domain modules need data validation, that validation is a concern *within* the domain module (e.g., signal validation is a signal-domain concern), not a cross-cutting quality service.

---

### NG-2: Market Foundry is not a framework

Market Foundry does not expose reusable libraries, SDKs, or APIs for external consumers to build upon. It is not designed to be imported as a dependency by other repositories.

**Rationale:** Frameworks require stability guarantees, backward compatibility, and API versioning. Market Foundry's internal architecture must remain free to evolve without external consumers constraining its design.

**Boundary:** The raccoon-cli is a standalone tool with its own release cycle. It is not a framework — it is an enforcement tool that happens to live in the same repository.

---

### NG-3: Market Foundry is not a monolith

Market Foundry does not run all domains in a single process. Each domain module is a separately deployable service with its own `cmd/` entry point and actor tree.

**Rationale:** Monolithic deployment couples domain lifecycles. A bug in the signal domain should not take down the observation domain. Independent deployment is a structural requirement, not an optimization.

**Boundary:** Services share infrastructure (NATS, configuration patterns, shared primitives) but not process space.

---

### NG-4: Market Foundry is not a microservices platform

Market Foundry does not provide service discovery, API gateways (beyond its own HTTP gateway), circuit breakers, service meshes, or orchestration layers.

**Rationale:** Platform concerns belong to the deployment environment (Kubernetes, Nomad, etc.), not to the application layer. Embedding platform abstractions creates coupling to specific deployment targets.

**Boundary:** Market Foundry assumes NATS is available and that services can reach each other by configured addresses. Everything else is the platform's responsibility.

---

### NG-5: Market Foundry does not use Kafka

Market Foundry does not use Apache Kafka, Kafka Connect, Schema Registry, or any Kafka-adjacent technology. NATS with JetStream is the sole messaging infrastructure.

**Rationale:** The Kafka adapters were removed during sanitization because they introduced infrastructure coupling, operational complexity, and architectural assumptions that conflicted with the actor-based, NATS-native design. This decision is permanent.

**Boundary:** If a future integration requires Kafka interoperability, it must be implemented as an **external bridge service** outside the Market Foundry repository, never as an internal adapter.

---

### NG-6: Market Foundry does not support multiple messaging backends

There is no adapter abstraction layer allowing the message bus to be swapped. NATS is not behind an interface that could be replaced with RabbitMQ, Kafka, or Redis Streams.

**Rationale:** Messaging-backend abstraction layers add complexity without value when the system is committed to a single backend. NATS-specific features (JetStream, subject hierarchy, request/reply) are used directly and intentionally.

**Boundary:** If NATS were ever replaced (which is not planned), it would be a full migration, not a configuration change. The system does not pretend to be backend-agnostic.

---

## Domain Non-Goals

### NG-7: Market Foundry does not implement trading execution in Phase 2

The MarketMonkey absorption phase focuses on **observation, evidence, and signal** domains. Strategy, risk, execution, and portfolio are Phase 3 concerns at the earliest.

**Rationale:** Execution and portfolio management require regulatory considerations, external exchange connectivity, and risk controls that are premature to introduce before the data pipeline is stable.

**Boundary:** Domain types for execution and portfolio may be sketched in design documents, but no code for these domains enters the repository until the observation→evidence→signal pipeline is operational and validated.

---

### NG-8: Market Foundry does not store historical market data

Market Foundry processes market data streams. It does not operate as a data warehouse, time-series database, or historical data archive.

**Rationale:** Long-term storage introduces retention policies, storage scaling, query optimization, and compliance concerns that are orthogonal to stream processing. These responsibilities belong to dedicated storage infrastructure.

**Boundary:** Domain modules may maintain short-lived state (actor state, JetStream windows) for processing purposes. Persistent historical storage is an external concern.

---

### NG-9: Market Foundry does not provide a user interface

Market Foundry exposes HTTP APIs for programmatic interaction. It does not include dashboards, web UIs, CLIs for end users, or visualization tools.

**Rationale:** UI concerns evolve on a fundamentally different cadence than backend systems. Coupling them creates deployment friction and splits focus.

**Boundary:** The raccoon-cli is a developer tool, not a user-facing interface. External systems may consume Market Foundry's APIs to build UIs.

---

## Process Non-Goals

### NG-10: Market Foundry does not copy code from predecessors

Code from quality-service, MarketMonkey, or any other repository is not copied into Market Foundry. Patterns are studied and **re-implemented** within Market Foundry's architectural constraints.

**Rationale:** Copied code carries hidden assumptions about the source system's architecture, dependencies, and invariants. Re-implementation forces each pattern to justify itself within the target architecture.

**Boundary:** Shared primitives (e.g., CBOR codec, envelope types) that are identical in intent may be re-typed, but never aliased or sym-linked from external sources.

---

### NG-11: Market Foundry does not introduce speculative abstractions

No interface, adapter, or utility is added "in case we need it later." Every abstraction must serve a current, concrete use case.

**Rationale:** Speculative abstractions are the primary source of accidental complexity. They create maintenance burden, obscure intent, and constrain future design when the actual need turns out to differ from the speculation.

**Boundary:** Design documents may describe future abstractions. Code may not implement them until the triggering use case exists.

---

### NG-12: Market Foundry does not relax quality gates for velocity

Quality gates (raccoon-cli profiles: fast, ci, deep) are not bypassed, weakened, or deferred to accelerate delivery. A failing quality gate blocks the change — always.

**Rationale:** Quality gate relaxation was a contributing factor in the structural degradation that led to the quality-service sanitization. Speed gained by bypassing gates is repaid with compound interest during cleanup.

**Boundary:** Quality gate *profiles* may be tuned (e.g., adding new checks to `deep`), but the gate mechanism itself is non-optional.

---

### NG-13: Market Foundry does not maintain backward compatibility with quality-service

There is no migration path from quality-service to Market Foundry. Quality-service configurations, data, APIs, and deployment artifacts are not compatible and no compatibility layer will be provided.

**Rationale:** Backward compatibility would require preserving the very structures that were deliberately removed. The sanitization was a clean break by design.

**Boundary:** Operational knowledge from quality-service (e.g., "how NATS subjects should be structured") is carried forward as principles, not as code or configuration.

---

## Technology Non-Goals

### NG-14: Market Foundry does not use ORMs or query builders

Database interactions (when introduced) use direct SQL or purpose-built repository patterns. No ORM, query builder, or database abstraction layer will be adopted.

**Rationale:** ORMs obscure the actual data access patterns, generate unpredictable queries, and create a false sense of database-agnosticism that breaks under real-world performance constraints.

**Boundary:** The existing `memdb` adapter demonstrates the preferred pattern: explicit, interface-based repositories with no magic.

---

### NG-15: Market Foundry does not adopt gRPC for internal communication

Internal service communication uses NATS request/reply with CBOR encoding. gRPC is not used for inter-service communication within Market Foundry.

**Rationale:** gRPC would introduce protobuf compilation, code generation, and a parallel type system alongside Go's native types. NATS request/reply with CBOR achieves the same goals with less tooling and tighter integration with the actor model.

**Boundary:** If external systems require gRPC, an edge adapter may be considered, but internal communication remains NATS-native.

---

## Summary Table

| ID    | Non-Goal                                     | Category     |
|-------|----------------------------------------------|--------------|
| NG-1  | Not a quality service                        | Architecture |
| NG-2  | Not a framework                              | Architecture |
| NG-3  | Not a monolith                               | Architecture |
| NG-4  | Not a microservices platform                 | Architecture |
| NG-5  | No Kafka                                     | Architecture |
| NG-6  | No messaging backend abstraction             | Architecture |
| NG-7  | No execution/portfolio in Phase 2            | Domain       |
| NG-8  | No historical data storage                   | Domain       |
| NG-9  | No user interface                            | Domain       |
| NG-10 | No code copying from predecessors            | Process      |
| NG-11 | No speculative abstractions                  | Process      |
| NG-12 | No quality gate relaxation                   | Process      |
| NG-13 | No backward compatibility with quality-service| Process     |
| NG-14 | No ORMs                                      | Technology   |
| NG-15 | No gRPC internally                           | Technology   |

---

## Revisiting Non-Goals

A non-goal may be revisited only when:

1. The original rationale is demonstrably invalid
2. A concrete use case (not a hypothetical) demands the change
3. The impact on existing principles is analyzed and accepted
4. The change is documented with the same rigor as the original non-goal

The burden of proof is on the party requesting the change, not on the non-goal.
