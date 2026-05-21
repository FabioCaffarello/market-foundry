# Market Foundry — Evolution Playbook

## Purpose

This playbook governs how Market Foundry grows. It is not a set of aspirational principles — it is an operational rulebook derived from 32+ stages of disciplined evolution, from sanitization through stream mesh canonicalization.

Every stage, PR, and architectural decision must be evaluated against this playbook. If a change cannot be justified by these rules, it does not belong in the repository.

## Identity

Market Foundry is a **domain-oriented runtime foundation** for market data processing. It is not an application, not a framework, not a renamed quality-service. It was born from deliberate sanitization and exists to provide:

- Actor-based concurrency (Hollywood framework)
- Message-driven communication (NATS + JetStream, sole messaging infrastructure)
- Configuration lifecycle management (configctl)
- Static analysis enforcement (raccoon-cli)
- Stream mesh as explicit architecture

---

## Foundational Principles

These principles are ordered by precedence. When two principles conflict, the higher-ranked principle wins.

1. **Layer Sovereignty** — `domain → application → adapters → actors → interfaces → cmd`. No outward or sideways imports. Enforced by `raccoon-cli arch-guard`.
2. **Domain Module Isolation** — Each domain is a self-contained vertical slice. Cross-domain communication happens exclusively through NATS messages.
3. **Messages as Boundaries** — All inter-service communication through NATS (request/reply or JetStream). No direct function calls between services.
4. **Actors Own Lifecycle** — Hollywood is the sole concurrency primitive. No unsupervised goroutines.
5. **Configuration as Domain Object** — Configctl manages the full lifecycle: Draft → Validated → Compiled → Active → Deactivated → Archived.
6. **Static Enforcement Over Convention** — raccoon-cli provides automated rules. Convention alone is not sufficient.
7. **No Premature Domain Implementation** — Boundaries, invariants, contracts, actor topology, and quality gates must be defined before code.
8. **Identity Integrity** — Market Foundry is not a renamed quality-service. No legacy contamination permitted.

---

## Stream Mesh as Central Architecture

The stream mesh is not an implementation detail — it is the architecture. Every data flow in Market Foundry is a named, typed, ownership-bound message flow organized by:

- **Family**: configctl, observation, evidence, signal, decision, strategy, risk, execution
- **Surface**: events, control, query, projection
- **Dimensions**: source, symbol, timeframe, aggregate, verb

### Mesh Properties (Inviolable)

1. **Single-Writer Streams** — Exactly one producer binary per JetStream stream.
2. **Fan-Out Consumption** — Multiple consumers per stream, each with independent durable positions.
3. **Partition-Aligned Isolation** — Actor trees mirror mesh partitioning for failure isolation.
4. **Deduplication by Design** — Every event carries a deterministic message ID.
5. **Envelope Uniformity** — All messages wrapped in `Envelope[T]` with Kind, Type, Source, Subject, CorrelationID.

### Stream Ownership (Current)

| Stream | Writer | Consumers |
|---|---|---|
| CONFIGCTL_EVENTS | configctl | ingest, derive |
| OBSERVATION_EVENTS | ingest | derive |
| EVIDENCE_EVENTS | derive | store, writer |
| SIGNAL_EVENTS | derive | store, writer |
| DECISION_EVENTS | derive | store, writer |
| STRATEGY_EVENTS | derive | store, writer |
| RISK_EVENTS | derive | store, writer |
| EXECUTION_EVENTS | derive | store, writer, execute |
| EXECUTION_FILL_EVENTS | execute | store |

### Data Flow Direction

```
configctl → ingest → derive
derive → store → gateway
derive → execute
derive / execute → writer → clickhouse → gateway
```

Acyclic and message-driven. No feedback loops. No binary-to-binary RPC chains.

---

## Binary Roles and Boundaries

Seven long-running binaries plus one standalone deployment tool are currently part of the governed baseline.

| Binary | Purpose (single sentence) | Owns | Never Does |
|---|---|---|---|
| **gateway** | Translates HTTP requests into NATS operations and returns results | HTTP listener, NATS request client | Domain logic, KV access, event publishing |
| **configctl** | Owns the full lifecycle of configuration documents | CONFIGCTL_EVENTS, config repository | Market data processing |
| **ingest** | Receives raw market data and publishes normalized observation events | OBSERVATION_EVENTS, exchange connections | Evidence derivation, query serving |
| **derive** | Consumes observation streams and produces downstream domain events | EVIDENCE_EVENTS, SIGNAL_EVENTS, DECISION_EVENTS, STRATEGY_EVENTS, RISK_EVENTS, EXECUTION_EVENTS | Persistent storage, query serving |
| **store** | Consumes domain events and builds read-optimized projections | KV buckets, latest-value and control-query replies | Domain event production, domain logic |
| **execute** | Consumes execution intents and materializes controlled execution state | EXECUTION_FILL_EVENTS, execution control/read-side surfaces | HTTP serving, schema migration |
| **writer** | Persists selected domain events into ClickHouse for analytical reads | ClickHouse inserts, analytical write-path consumers | Operational KV authority, control-plane ownership |
| **migrate** | Applies forward-only ClickHouse schema changes | Migration catalog execution, `_migrations` metadata | Long-running runtime behavior, NATS integration |

### Binary Invariants

- Gateway is the only public/domain HTTP surface.
- Configctl is the only config authority.
- Migrate is the only schema authority for ClickHouse.
- Writer is the only event-to-ClickHouse bridge.
- Every binary has a supervisor root actor.
- New long-running binaries require explicit architectural justification — existing binaries must be proven insufficient first.

---

## Mandatory Design Patterns

### Derive: FamilyProcessor Pattern

Declarative registration of evidence families. Adding a new evidence type means adding one entry to the processor list — not modifying the spawning loop.

```
DeriveSupervisor.start() — declares processors
  → SourceScopeActor.onActivateSampler — iterates processors
    → SamplerActor[per processor × symbol × timeframe] — type-safe transform
```

**Rules:**
- Transform logic is pure (no I/O, no actors, no NATS). Table-driven tests on synthetic data.
- Each scope owns its publisher. No shared publishers across scopes.
- FamilyProcessor is a struct, not an interface. No generic framework.

### Store: ProjectionPipeline Pattern

Declarative registration of projection families. Mirrors FamilyProcessor symmetry.

```
StoreSupervisor.start() — declares pipelines
  → spawning loop — iterates pipelines
    → ProjectionActor[per family] — single-writer to KV
    → ConsumerActor[per family] — durable JetStream consumer
```

**Rules:**
- One consumer per family with type-specific filter subject.
- One projection actor per family with exclusive write access to its buckets.
- Single-writer per KV bucket — no cross-family sharing.
- Only `Final=true` events materialized.
- Monotonicity guard on latest projections — never regress.

### Gateway: EvidenceFamilyDeps Pattern

Grouped use cases per evidence family. Stateless translation only.

**Rules:**
- Gateway never touches KV directly.
- Latest-value operational reads go through NATS request/reply to store; analytical history reads use ClickHouse reader adapters at the composition boundary.
- Evidence routes are optional (graceful degradation if store unavailable).
- One handler per query operation (parse → call use case → format).
- Configctl readiness is required; evidence readiness is not.

### Configctl: Configuration Lifecycle

Sole authority over configuration state transitions.

**Rules:**
- All activation flows go through configctl events.
- BindingWatcherActors in ingest and derive subscribe to configctl lifecycle.
- No binary hardcodes its own activation — configctl drives it.

---

## Evolution by Readiness

Market Foundry grows through readiness gates, not timelines.

### Readiness Evaluation Criteria

Before opening a new domain or capability:

1. **Pattern Proven** — The structural pattern the new capability depends on must be validated with at least 2 concrete implementations (e.g., FamilyProcessor proven with candle + tradeburst before adding volume).
2. **Governance Current** — actor-ownership.md, stream-family-catalog.md, raccoon-cli topology rules must reflect the actual codebase.
3. **Prerequisites Met** — All blocking gaps identified in readiness reviews must be resolved.
4. **Contracts Defined** — NATS subjects, message types, envelope specs, and query contracts must be documented before implementation.
5. **Activation Mechanism Verified** — Config-driven activation must work end-to-end for the new capability.

### Readiness Gate Sequence

```
governance current → contracts defined → activation verified → pattern proven → implementation
```

Never skip a gate. Never implement ahead of governance.

---

## Anti-Debt Rules

### Rule 1: No Temporary Implicit

If something is temporary, it must be:
- Documented as a known gap with explicit scope.
- Tracked in the stage report that introduced it.
- Scheduled for resolution (not "someday").

### Rule 2: No Premature Abstraction

Every abstraction must serve a current use case. The Foundry prefers:
- Explicit duplication over premature generalization.
- Structs over interfaces (unless polymorphism is actively needed).
- Compiled-in registration over dynamic plugin loading.
- Three similar lines over one clever helper.

### Rule 3: Structural Reuse, Not Framework

When adding a new evidence type, the pattern is followed explicitly — not inherited from a base class or generated by a framework. The 13-step evidence onboarding checklist is the mechanism, not an abstraction layer.

### Rule 4: Governance Keeps Pace

Documentation and raccoon-cli rules must be updated in the same stage that introduces structural changes. Governance debt is treated as a hard blocker for further expansion.

### Rule 5: No Scope Creep Within Stages

Each stage has one structural capability. If a stage discovers a new need, that need becomes a future stage — not a "quick addition" to the current one.

---

## Evaluating Changes

A change makes Market Foundry better if it satisfies ALL of these:

### More Canonical
- [ ] Follows established naming conventions (stream-taxonomy, evidence-read-model-guidelines).
- [ ] Uses existing patterns (FamilyProcessor, ProjectionPipeline, EvidenceFamilyDeps) — does not invent new ones without justification.
- [ ] Respects layer sovereignty — no outward or sideways imports.

### More Robust
- [ ] Failure in one scope does not affect other scopes (partition-aligned isolation).
- [ ] Replay safety — idempotent by design, monotonicity guards in place.
- [ ] Validation at domain boundary — not deferred to callers.

### More Observable
- [ ] Health trackers for new consumers and projections.
- [ ] Structured logging with actor context.
- [ ] Projection stats (materialized, skipped, rejected counts).

### More Scalable
- [ ] Linear resource growth with added symbols/timeframes (N × T, not N²).
- [ ] Single-writer invariant preserved.
- [ ] No shared mutable state between actors.

If a change fails any criterion without explicit justification in the stage report, it should be rejected or recalibrated.

---

## Golden Rules of the Foundry

1. **The mesh is the architecture.** If it is not in the stream mesh, it does not exist.
2. **One capability per stage.** Scope discipline is non-negotiable.
3. **Governance before expansion.** Stale docs and stale CLI rules are hard blockers.
4. **Explicit over clever.** Duplication that is clear beats abstraction that is fragile.
5. **Prove with two, scale with three.** No pattern is validated until two implementations prove it. No pattern is trusted until three exercise it.
6. **Transform logic is pure.** Samplers have no I/O, no actors, no NATS. Test with tables, not mocks.
7. **Store owns reads. Derive owns writes. Gateway owns translation.** No boundary violations.
8. **Configctl drives activation.** No binary decides its own lifecycle.
9. **raccoon-cli enforces, not suggests.** If it can be checked automatically, it must be.
10. **No legacy contamination.** Quality-service patterns, naming, and identity are permanently prohibited.

---

## The Triad: Foundry / MarketMonkey / Market Raccoon

Market Foundry does not evolve in isolation. Three systems inform its direction:

### Market Foundry (this repository)
- Structural foundation, stream mesh, contracts-first, projection authority.
- Source of truth for patterns, topology, and governance.
- Grows through readiness-gated stages.

### MarketMonkey (reference runtime)
- Pattern catalogue for observation → evidence → signal pipelines.
- Reference for per-source/symbol/timeframe actor organization.
- NOT a code source — patterns are re-implemented natively in Foundry.
- Validates that Foundry's patterns can support real-world market data processing at scale.

### Market Raccoon (domain reference)
- External domain authority for invariants, boundaries, and evolutionary direction.
- Prevents premature coupling by clarifying which domains are ready and which are not.
- Guards against scope expansion beyond what the domain model supports.

### How to Use the Triad

- **Before adding a domain**: Check Market Raccoon for boundary definitions and maturity assessment.
- **Before implementing a pattern**: Check MarketMonkey for proven runtime patterns and their constraints.
- **Before merging**: Check Foundry's own governance (this playbook, raccoon-cli, readiness reviews).

---

## How to Use This Playbook

### When Designing a Stage
1. Identify the single structural capability.
2. Verify all readiness gates are met.
3. Define contracts before implementation.
4. Reference this playbook's patterns and golden rules.

### When Reviewing a PR
1. Check against the evaluation criteria (canonical, robust, observable, scalable).
2. Verify governance is updated (docs, raccoon-cli rules, actor-ownership).
3. Verify no scope creep beyond the stage's declared capability.

### When Guiding the Opus
1. Reference the golden rules explicitly in prompts.
2. Require code + docs + report for every stage.
3. Interrupt and recalibrate if the Opus proposes changes outside the declared scope.
4. See [opus-guidance-rules.md](opus-guidance-rules.md) for detailed guidance.

### When Conducting a Readiness Review
1. Check every prerequisite in the readiness gate sequence.
2. Verify governance is current (not "mostly current" — current).
3. Document the verdict with specific evidence, not general impressions.
4. If conditionally ready, list every condition explicitly with resolution path.
