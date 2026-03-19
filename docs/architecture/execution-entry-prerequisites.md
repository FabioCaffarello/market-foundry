# Execution Entry Prerequisites

> Mandatory conditions that must be satisfied before any `execution` domain code enters Market Foundry.
> Date: 2026-03-18 | Stage: S68

## Purpose

This document defines the hard prerequisites for opening an `execution` layer in Market Foundry. Unlike previous domain entries (signal, decision, strategy, risk), execution crosses the **action boundary** — the first domain that can produce real-world financial side effects. The prerequisites are therefore stricter, and no prerequisite may be waived without explicit architectural justification documented in a stage report.

---

## Prerequisite Categories

### Category A: Test Debt Resolution (Blocking)

These gaps have been carried since early stages and are documented as acceptable for analytical domains. They are not acceptable for a domain that may trigger financial operations.

#### A-1: Adapter Test Coverage

**Requirement**: Every NATS publisher and consumer adapter across all domains must have unit tests covering:
- Successful publish/consume round-trip
- Encoding/decoding correctness
- Deduplication key propagation
- Error handling (connection failure, timeout, invalid payload)

**Affected domains**: observation, evidence, signal, decision, strategy, risk (6 domains × 2 adapter types = 12 untested components)

**Verification**: `go test ./internal/adapters/nats/...` must cover all publisher and consumer types.

**Resolution stage**: S69

#### A-2: Derive Actor Test Coverage

**Requirement**: Derive scope actors (samplers, evaluators, resolvers) must have unit tests covering:
- Message routing from SourceScopeActor to child actors
- Fan-out correctness (signal → decision, decision → strategy, strategy → risk)
- Error isolation (one actor failure does not propagate to siblings)

**Verification**: `go test ./internal/actors/scopes/derive/...` must cover all actor message paths.

**Resolution stage**: S69 (can be combined with A-1)

---

### Category B: Traceability Hardening (Blocking)

Execution auditability requires stronger traceability guarantees than log-based reconstruction.

#### B-1: Automated Traceability Verification

**Requirement**: An integration test that:
1. Publishes a synthetic observation event
2. Waits for the full chain to process (evidence → signal → decision → strategy → risk)
3. Asserts that all events in the chain share the same `correlation_id`
4. Asserts that each event's `causation_id` points to the immediately preceding event's `Metadata.ID`
5. Fails if any link in the chain is broken

**Verification**: Integration test passes in CI with a running NATS server.

**Resolution stage**: S71

#### B-2: Trace Metadata Persistence Decision

**Requirement**: A formal design decision on how trace metadata is persisted for post-trade analysis:

| Option | Pros | Cons |
|--------|------|------|
| Persist in KV projection | Simple, co-located with domain state | Increases KV entry size, not needed for real-time queries |
| Separate audit KV bucket | Clean separation, purpose-built | Additional infrastructure, sync complexity |
| JetStream replay | Zero additional storage, already available | Requires stream retention beyond operational window |
| Dedicated audit stream | Purpose-built, append-only, immutable | New stream introduces new ownership questions |

**Verification**: Design decision documented with rationale in a dedicated architecture document.

**Resolution stage**: S72

---

### Category C: Governance Alignment (Blocking)

#### C-1: Risk Drift Rules Verification

**Requirement**: Verify that risk-specific drift rules exist and pass in the current codebase via `raccoon-cli drift-detect`. If missing, create them following the established pattern (RD-1..RD-5).

**Verification**: `raccoon-cli drift-detect` passes with risk rules active.

**Resolution stage**: S70

#### C-2: Execution Governance Rules

**Requirement**: Before any execution code is written, the following must exist:
- Execution drift rules (ED-1..ED-5) in raccoon-cli
- Execution guardrails preventing premature implementation
- `knownExecutionFamilies` closed set in config validation
- `executionDependsOnRisk` dependency map
- Execution domain entry in actor-ownership.md
- Execution streams in stream-family-catalog.md

**Verification**: `raccoon-cli quality-gate --profile fast` passes with execution rules active.

**Resolution stage**: S74 (after domain design)

---

### Category D: Domain Design (Blocking)

#### D-1: Execution Domain Boundary Definition

**Requirement**: A formal design document (`execution-domain-design.md`) answering:

1. **What is an execution intent?** — Domain model definition with fields, validation rules, enums.
2. **What is the execution lifecycle?** — State machine: intent → submitted → filled → cancelled → expired?
3. **What does execution own?** — Boundaries: intent creation, order tracking, fill reconciliation, position state?
4. **What does execution NOT own?** — Boundaries: portfolio aggregation, P&L calculation, margin management?
5. **How does execution relate to risk?** — One risk assessment → one execution intent? Or aggregation?
6. **What is the first family?** — Concrete type definition for the first execution family.

**Verification**: Document reviewed and approved following the strategy-domain-design.md pattern.

**Resolution stage**: S73

#### D-2: Venue Adapter Architecture Decision

**Requirement**: A formal design decision on venue adapter integration:

1. **First slice**: Paper execution only (no venue adapter). Execution intent is recorded and projected but no external API is called.
2. **Second slice**: Simulated venue adapter that validates order format and returns synthetic fills.
3. **Third slice**: Real venue adapter with circuit breaker, rate limiting, and retry logic.

**Verification**: Design decision documented with clear boundaries per slice.

**Resolution stage**: S73 (included in domain design)

---

### Category E: Safety Mechanisms (Blocking for Live Execution)

These prerequisites are NOT blocking for paper execution but MUST be resolved before any real venue interaction.

#### E-1: Kill Switch Design

**Requirement**: A mechanism to halt all execution activity without requiring a full binary restart:
- Configuration-driven (e.g., configctl command)
- Immediate effect (within 1 event cycle)
- Auditable (kill switch activation is itself an event)
- Recoverable (can be re-enabled without restart)

**Resolution stage**: S76

#### E-2: Circuit Breaker for Venue Adapters

**Requirement**: Automatic execution halt when:
- Venue API returns errors above threshold
- Order rejection rate exceeds threshold
- Position limits are breached
- Network partition detected

**Resolution stage**: Future (not in current roadmap until venue adapter is introduced)

#### E-3: Execution Rate Limiting

**Requirement**: Configurable limit on:
- Maximum orders per second per symbol
- Maximum orders per minute per source
- Maximum total position exposure

**Resolution stage**: Future (not in current roadmap until venue adapter is introduced)

---

## Prerequisite Dependency Graph

```
A-1 (adapter tests) ──┐
                       ├──→ B-1 (trace verification) ──┐
A-2 (actor tests) ────┘                                 │
                                                         ├──→ D-1 (domain design) ──→ C-2 (governance) ──→ S75 (first slice)
C-1 (risk drift) ──────────────────────────────────────┘                                                       │
                                                                                                                ↓
B-2 (trace persistence) ──────────────────────────────────────────────────────────────────────────────── E-1 (kill switch)
```

## Verification Checklist

Before any execution code enters the repository, all items must be checked:

- [ ] A-1: Adapter test coverage sweep complete
- [ ] A-2: Derive actor test coverage complete
- [ ] B-1: Automated traceability verification test passing in CI
- [ ] B-2: Trace metadata persistence design decision documented
- [ ] C-1: Risk drift rules verified and passing
- [ ] C-2: Execution governance rules active in raccoon-cli
- [ ] D-1: Execution domain design document approved
- [ ] D-2: Venue adapter architecture decision documented
- [ ] E-1: Kill switch design documented (blocking only for live execution)

**No prerequisite may be skipped.** If a prerequisite is later deemed unnecessary, it must be formally removed via a stage report with rationale.
