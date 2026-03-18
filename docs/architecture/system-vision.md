# System Vision — Market Foundry

> Canonical document. Defines the identity, purpose, and evolutionary direction of Market Foundry.
> Approved: 2026-03-16. Supersedes all prior quality-service references.

---

## What Market Foundry Is

Market Foundry is a **domain-oriented runtime foundation for market data processing**.

It provides the structural skeleton — layered architecture, actor-based concurrency, message-driven communication, and configuration lifecycle management — upon which market-domain modules are composed, deployed, and evolved independently.

Market Foundry is not an application. It is a **foundry**: a controlled environment where domain modules are forged, validated, and operated under strict architectural governance.

### Core Identity

| Attribute          | Definition                                                                 |
|--------------------|---------------------------------------------------------------------------|
| **Type**           | Domain runtime foundation                                                 |
| **Runtime model**  | Actor-based (Hollywood framework), message-driven (NATS + JetStream)      |
| **Primary concern**| Hosting and orchestrating market-domain modules with structural guarantees |
| **Governance**     | Static analysis enforcement via raccoon-cli, layered dependency rules      |
| **Deployment unit**| Composable services behind a unified API gateway                          |

---

## Why Market Foundry Exists

Market Foundry exists to solve three problems that its predecessors exposed:

1. **Identity drift.** The original quality-service accumulated market-adjacent responsibilities without a coherent domain boundary. Renaming it was not enough — the architecture itself carried assumptions from a domain it no longer served.

2. **Structural contamination.** Kafka adapters, validation pipelines, and emulator services were coupled into the runtime in ways that prevented clean evolution. New domains could not be introduced without inheriting technical debt from removed ones.

3. **Absence of a neutral foundation.** There was no clean substrate onto which new market domains (observation, evidence, signal, strategy, risk, execution, portfolio) could be projected without carrying legacy constraints.

Market Foundry is the answer: a **deliberately emptied, structurally validated, architecturally governed foundation** that exists to receive new domain logic — not to perpetuate old logic under a new name.

---

## What Market Foundry Inherits

### From the sanitized quality-service (structural foundation)

These elements were retained because they are **domain-agnostic infrastructure**:

- **Layered architecture**: `domain → application → adapters → actors → interfaces → cmd`
- **Actor lifecycle**: Hollywood-based supervisors, event routers, graceful shutdown
- **NATS messaging**: Request/reply, JetStream publishing, CBOR codec, envelope pattern
- **Configuration lifecycle**: Draft → Validate → Compile → Activate → Deactivate (configctl)
- **HTTP gateway**: Health checks, readiness probes, RESTful routing
- **Shared primitives**: Settings schema, bootstrap, problem responses, event dispatcher, request context
- **Quality enforcement**: raccoon-cli with arch-guard, contract-audit, drift-detect, quality-gate
- **Module topology**: Go workspace with isolated modules per architectural layer

### From Market Raccoon (domain knowledge, not code)

Market Raccoon (raccoon-cli) contributes **domain understanding and invariants** to Market Foundry:

- It defines what valid architecture looks like (layer boundaries, dependency direction)
- It encodes quality contracts (messaging patterns, topology rules)
- It provides the vocabulary for structural reasoning (drift, impact, coverage)
- Its code is **not imported** by Go services — it operates externally as a static analysis tool

Market Raccoon is a **domain oracle**, not a code dependency.

### From MarketMonkey (future reference, not current implementation)

MarketMonkey is the **reference architecture for stream processing and actor composition** in market domains:

- It demonstrates how market data flows through observation → evidence → signal pipelines
- It provides proven patterns for actor-per-stream supervision
- It informs how new domain modules should be structured within Market Foundry

MarketMonkey is a **pattern catalogue**. Its code will not be copied — its lessons will be re-implemented natively within Market Foundry's architectural constraints.

---

## What Was Deliberately Discarded

The following were removed during sanitization and **must not return**:

| Removed element              | Reason                                                            |
|------------------------------|-------------------------------------------------------------------|
| Kafka adapters               | Infrastructure coupling to a messaging system not aligned with the target architecture |
| Validator service            | Quality-domain artifact with no role in market domain             |
| Consumer service             | Kafka→NATS bridge with no equivalent need                        |
| Emulator service             | Synthetic data generator for a domain that no longer exists       |
| Dataplane topology           | Kafka-specific mapping logic tied to removed infrastructure       |
| Runtime bootstrap client     | Bound to validator runtime, not generalizable                     |
| Quality-specific HTTP routes | `/runtime/validator/*` endpoints serving a removed domain         |
| `.context/` directory        | Documentation artifacts reflecting quality-service identity       |
| Quality-service identity     | Docker images, networks, config paths bearing the old name        |

These removals are **permanent and non-negotiable**. See [prohibited-carryovers.md](prohibited-carryovers.md) for the authoritative prohibition list.

---

## Evolutionary Direction

Market Foundry will evolve through **domain module absorption**, not monolithic growth.

### Phase model

```
Phase 0 — Sanitization        ✅ Complete
Phase 1 — Canonical Vision    ← Current (this document)
Phase 2 — MarketMonkey Absorption
Phase 3 — Domain Module Implementation
Phase 4 — Operational Maturity
```

### Phase 2: MarketMonkey Absorption

MarketMonkey patterns will be absorbed as **new domain modules** within Market Foundry's existing layered architecture:

- Each market domain gets its own `internal/domain/{name}/` and `internal/application/{name}/`
- Actor scopes are introduced per domain under `internal/actors/scopes/{name}/`
- NATS adapters are added per domain under `internal/adapters/nats/`
- New services are registered as `cmd/{name}/` with their own `go.mod`
- HTTP routes extend the gateway under `internal/interfaces/http/routes/{name}.go`

### Phase 3: Domain Modules (anticipated, not committed)

The following domains are candidates for implementation. Their inclusion is conditional on architectural readiness, not on schedule:

1. **Observation** — Raw market data capture and normalization
2. **Evidence** — Structured observations with provenance metadata
3. **Signal** — Derived indicators from evidence streams
4. **Strategy** — Trading strategy definition and lifecycle
5. **Risk** — Risk assessment, limits, and exposure tracking
6. **Execution** — Order routing and trade lifecycle management
7. **Portfolio** — Position state, P&L, and performance attribution

Each domain must pass raccoon-cli quality gates before integration.

### Phase 4: Operational Maturity

- Observability (metrics, tracing, structured logging) integrated per domain
- Multi-environment deployment (dev, staging, production)
- Performance baselines and regression detection
- Operational runbooks per domain module

---

## Relationship Map

```
┌─────────────────────────────────────────────────────┐
│                  Market Foundry                      │
│                                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────┐  │
│  │   configctl   │  │   server     │  │  domain   │  │
│  │  (lifecycle)  │  │  (gateway)   │  │  modules  │  │
│  └──────┬───────┘  └──────┬───────┘  └─────┬─────┘  │
│         │                 │                │         │
│         └────────┬────────┘────────────────┘         │
│                  │                                    │
│         ┌────────▼────────┐                          │
│         │  NATS + JetStream│                          │
│         │  (message bus)   │                          │
│         └─────────────────┘                          │
│                                                      │
│  ┌─────────────────────────────────────────────────┐ │
│  │  raccoon-cli (external quality enforcement)     │ │
│  └─────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────┘

  MarketMonkey ─── pattern reference (not imported)
  Market Raccoon ── domain oracle (static analysis only)
```

---

## Summary

Market Foundry is a **governed, actor-based, message-driven runtime foundation** purpose-built to host market-domain modules. It inherits structural patterns from a sanitized predecessor, absorbs domain knowledge from MarketMonkey, and enforces architectural integrity through Market Raccoon. It exists to provide a clean, principled substrate for market data processing — nothing more, nothing less.
