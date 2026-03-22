# S365 — StrategyResolvedEvent Producer Spec and Derive Ownership Report

> Stage type: Audit / specification.
> Wave: Derive Integration Wave (Phase 37).
> Block: DI-1 (Producer Spec and Derive Ownership).
> Predecessor: S364 (Derive Integration Wave Charter and Scope Freeze).
> Date: 2026-03-22.

---

## 1. Executive Summary

S365 completes DI-1 by auditing derive's strategy resolver and publisher
against the S359 canonical contract, defining the producer ownership model,
documenting all invariants, and establishing clear boundaries between derive,
strategy, execution, and store.

**Key findings**:

- **Zero blocking mismatches** between derive producer output and S359 contract.
- **6 of 11 S359 invariants** are producer-relevant; all 6 are compliant.
- **5 of 11 S359 invariants** are consumer-side only (already proven in S358–S363).
- **NATS subject alignment** is exact: publisher pattern matches both execute and store consumer filters.
- **Correlation/causation chain** propagates correctly from decision through strategy event metadata.
- **Dedup key format** (`strat:` prefix) is intentionally different from execution's `exec:` prefix — each domain manages its own dedup for its own stream.
- **Validation gates** ensure invalid strategies are never published.

The derive producer is **ready for unit testing in S366**.

---

## 2. What Was Done

### 2.1 Code Audit

Audited the following files against S359 contract (INV-1 through INV-11):

| File | Purpose | Audit Result |
|---|---|---|
| `internal/application/strategy/mean_reversion_entry_resolver.go` | Pure resolution logic | COMPLIANT — all fields match S359 |
| `internal/actors/scopes/derive/strategy_resolver_actor.go` | Actor wrapper, metadata construction | COMPLIANT — correlation/causation propagated |
| `internal/actors/scopes/derive/strategy_publisher_actor.go` | NATS publication bridge | COMPLIANT — delegates to publisher correctly |
| `internal/adapters/nats/natsstrategy/publisher.go` | JetStream publish, subject/dedup | COMPLIANT — subject and dedup key match contract |
| `internal/adapters/nats/natsstrategy/registry.go` | Stream/consumer specs | COMPLIANT — execute consumer matches S360 spec |
| `internal/domain/strategy/strategy.go` | Domain type, validation | COMPLIANT — all fields present, validation gates correct |
| `internal/domain/strategy/events.go` | Event envelope | COMPLIANT — metadata + strategy |
| `internal/actors/scopes/derive/messages.go` | Internal actor messages | COMPLIANT — primitive data per DBI-9 |
| `internal/shared/events/event.go` | Metadata construction | COMPLIANT — UUID, correlation, causation |

### 2.2 Field-Level Compliance Matrix

Produced a complete field-level mapping of every S359 contract field against
the derive producer's actual source. All 16 fields are compliant.

### 2.3 Invariant Coverage Matrix

| Category | Invariants | Status |
|---|---|---|
| Producer-relevant | INV-1, INV-3, INV-5, INV-7, INV-8, INV-11 | All COMPLIANT |
| Consumer-side only | INV-2, INV-4, INV-6, INV-9, INV-10 | N/A (proven in S358–S363) |
| **Blocking mismatches** | — | **ZERO** |

### 2.4 Ownership Model

Defined ownership for every artifact in the producer chain:

| Owner | Artifacts |
|---|---|
| `application/strategy` | `MeanReversionEntryResolver` (pure logic) |
| `actors/scopes/derive` | `MeanReversionEntryResolverActor`, `StrategyPublisherActor` |
| `adapters/nats/natsstrategy` | `Publisher`, `Registry` |
| `domain/strategy` | `Strategy`, `StrategyResolvedEvent`, `DecisionInput` |
| `shared/events` | `Metadata` construction |

### 2.5 Boundary Documentation

Documented clear boundaries:
- **Derive → NATS**: serialized JSON, subject from registry, dedup key from domain
- **Decision → Strategy** (within derive): actor message with primitive data (DBI-9)
- **NATS → Store**: independent durable consumer, projection + KV write
- **NATS → Execute**: independent durable consumer, evaluation + execution intent
- **Store → Gateway**: NATS request/reply, stateless translation

### 2.6 Lifecycle Specification

Defined 8-phase event lifecycle from resolution through query, with 9 lifecycle
invariants (LI-1 through LI-9).

### 2.7 Limits Documentation

Documented 11 explicit limitations of the current model, including:
- No at-most-once delivery (at-least-once with dedup)
- No cross-partition ordering
- No backpressure from execution
- No multi-decision aggregation
- No confidence threshold gate
- No rate limiting / debounce

---

## 3. Answers to Governing Questions

### DIQ-1: Does the derive strategy resolver produce StrategyResolvedEvent payloads that satisfy all 11 S359 contract invariants?

**YES** — for the 6 producer-relevant invariants (INV-1, INV-3, INV-5, INV-7,
INV-8, INV-11), all are compliant. The remaining 5 are consumer-side invariants
already proven in S358–S363.

**Confidence: HIGH** — based on code audit of all 9 files in the producer chain.

### DIQ-2: Is there a documented field-level compliance mapping between derive resolver output and the S359 contract?

**YES** — the producer spec document contains a 16-field compliance matrix
mapping every S359 contract field to its derive producer source, with compliance
status for each.

**Confidence: HIGH** — every field has a documented source and status.

---

## 4. Promoted Documents

| Document | Location | Purpose |
|---|---|---|
| StrategyResolvedEvent Producer Spec and Derive Ownership Model | [`docs/architecture/strategy-resolved-event-producer-spec-and-derive-ownership-model.md`](../architecture/strategy-resolved-event-producer-spec-and-derive-ownership-model.md) | Producer model, field compliance matrix, ownership, NATS alignment |
| Derive Producer: Boundaries, Invariants, Lifecycle, and Limits | [`docs/architecture/derive-producer-boundaries-invariants-lifecycle-and-limits.md`](../architecture/derive-producer-boundaries-invariants-lifecycle-and-limits.md) | Boundary map, lifecycle phases, invariants, explicit limits |

---

## 5. Capability Assessment

| Capability | Status | Evidence |
|---|---|---|
| **DC-1**: Derive producer contract compliance | **VERIFIED (audit)** | Field-level compliance matrix: 16/16 fields compliant, 6/6 producer invariants compliant |
| **DC-2**: Derive publisher correctness | **VERIFIED (audit)** | Subject, dedup, envelope type all match registry and consumer specs |
| **DC-3**: Store materialization | PENDING | DI-3 (S367) |
| **DC-4**: Gateway read path | PENDING | DI-3 (S367) |
| **DC-5**: Correlation chain | **VERIFIED (audit)** | Chain propagation traced through all 9 files |
| **DC-6**: End-to-end pipeline | PENDING | DI-4 (S368) |

---

## 6. Risk Update

| Risk (from S364) | Original Likelihood | S365 Assessment |
|---|---|---|
| Derive resolver output doesn't match S359 contract | MEDIUM | **RESOLVED** — audit found zero mismatches |
| NATS subject mismatch between derive publisher and execute consumer | LOW | **RESOLVED** — subject patterns match exactly |
| Store projection rejects derive-produced events | LOW | UNCHANGED — verification in DI-3 |
| Test infrastructure insufficient for multi-scope integration | MEDIUM | UNCHANGED — relevant for DI-4 |

---

## 7. Preparation for S366

S366 (DI-2: Canonical Derive Producer Wiring) has a clear path:

1. **Unit tests for resolver**: one test per invariant (PI-1 through PI-6, BI-1 through BI-6)
   - triggered → long direction, severity-scaled confidence
   - not_triggered → flat, zero confidence
   - insufficient → flat, zero confidence, reason metadata
   - unknown outcome → no strategy produced
   - all DecisionInput fields preserved
   - Final=true, Type=mean_reversion_entry always

2. **Unit tests for publisher**: subject format, dedup key format, envelope type, correlation/causation pass-through

3. **Unit tests for resolver actor**: metadata construction, correlation propagation, validation gate, message dispatch

4. **Code fixes**: none expected (zero mismatches found), but tests may reveal edge cases

**Estimated scope**: test-only. No architectural changes needed.

---

## 8. Verification

| Check | Result |
|---|---|
| Code audit completed (9 files) | YES |
| Field-level compliance matrix produced | YES — 16 fields, all compliant |
| Invariant coverage matrix produced | YES — 11 invariants mapped, 6 producer-relevant all compliant |
| Ownership model defined | YES — 8 artifacts, 5 ownership rules verified |
| Boundary rules documented | YES — 6 boundaries, 5 domain restriction sets |
| Lifecycle defined | YES — 8 phases, 9 lifecycle invariants |
| Limits documented | YES — 11 explicit limitations |
| DIQ-1 answered | YES — HIGH confidence |
| DIQ-2 answered | YES — HIGH confidence |
| Zero blocking mismatches | YES |
| S366 preparation documented | YES — 3 test categories, 0 code changes expected |

---

## References

- [StrategyResolvedEvent Producer Spec and Derive Ownership Model](../architecture/strategy-resolved-event-producer-spec-and-derive-ownership-model.md)
- [Derive Producer: Boundaries, Invariants, Lifecycle, and Limits](../architecture/derive-producer-boundaries-invariants-lifecycle-and-limits.md)
- [Derive Integration Wave Charter (S364)](../architecture/derive-integration-wave-charter-and-scope-freeze.md)
- [Derive Integration Capabilities, Questions, and Non-Goals (S364)](../architecture/derive-integration-capabilities-questions-and-non-goals.md)
- [Source Selection and Canonical Integration Contract (S359)](../architecture/source-selection-and-canonical-integration-contract.md)
- [Source-to-Execution Contract: Boundaries, Invariants, and Limits (S359)](../architecture/source-to-execution-contract-boundaries-invariants-and-limits.md)
- [Strategy Domain Design (S53)](../architecture/strategy-domain-design.md)
- [Strategy Activation and Ownership (S53)](../architecture/strategy-activation-and-ownership.md)
