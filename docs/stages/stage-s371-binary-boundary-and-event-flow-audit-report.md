# Stage S371 — Binary Boundary and Event-Flow Audit Report

> **Wave:** Multi-Binary Orchestration Proof (Phase 38)
> **Parent:** [S370 — Charter and Scope Freeze](stage-s370-multi-binary-orchestration-charter-report.md)
> **Next:** S372 — Compose Orchestration Wiring Validation
> **Method:** Static code audit of registries, supervisors, compose topology, and architecture docs
> **Status:** Complete

---

## 1. Executive Summary

S371 audited the binary boundaries, event-flow contracts, handoff points, and runtime invariants for all 7 operational binaries participating in the canonical multi-binary pipeline (`mean_reversion_entry`).

**Key findings:**
- All binary boundaries are clean — communication is exclusively through NATS subjects. No direct cross-binary function calls exist.
- 9 JetStream streams with single-writer ownership are correctly defined.
- 11 distinct handoff points are documented with subjects, payloads, ordering assumptions, and invariants.
- 1 transitional bridge (paper order intake in execute) is identified and scoped.
- 7 risks are classified (2 MEDIUM, 5 LOW) — none block the compose-level proof.
- The correlation/causation chain spans 3 binary boundaries and 7+ hops — fully propagated.

**Verdict:** The system is ready for compose-level wiring validation (S372). No boundary mismatches or contract violations were found that would block the proof.

---

## 2. Boundaries Audited

### 2.1 Binary Inventory

| Binary | Role | Streams Owned | Compose Dependency |
|--------|------|---------------|-------------------|
| nats | Message backbone | All (infrastructure) | Root |
| clickhouse | Analytical storage | N/A | Root |
| configctl | Config lifecycle | `CONFIGCTL_EVENTS` | nats |
| ingest | Data boundary | `OBSERVATION_EVENTS` | nats, configctl |
| derive | Processing core | 6 event streams | nats |
| store | Read model | KV buckets | nats, derive |
| execute | Venue executor | `EXECUTION_FILL_EVENTS` | nats, derive |
| gateway | HTTP API | None | nats, configctl, store |
| writer | Analytics writer | ClickHouse rows | nats, clickhouse |

### 2.2 Ownership Verification

Every JetStream stream has exactly one publishing binary. No ownership conflicts detected:

| Stream | Single Owner | Verified |
|--------|-------------|----------|
| `OBSERVATION_EVENTS` | ingest | Yes |
| `EVIDENCE_EVENTS` | derive | Yes |
| `SIGNAL_EVENTS` | derive | Yes |
| `DECISION_EVENTS` | derive | Yes |
| `STRATEGY_EVENTS` | derive | Yes |
| `RISK_EVENTS` | derive | Yes |
| `EXECUTION_EVENTS` | derive | Yes |
| `EXECUTION_FILL_EVENTS` | execute | Yes |
| `CONFIGCTL_EVENTS` | configctl | Yes |

### 2.3 Boundary Enforcement Mechanisms

1. **NATS subjects** — all cross-binary communication goes through NATS.
2. **Domain isolation** — domain packages do not import each other.
3. **Layer sovereignty** — enforced by `raccoon-cli arch-guard`.
4. **Single-writer** — each stream has one publisher.
5. **KV ownership** — only store writes to KV buckets.

---

## 3. Handoffs and Contracts

### 3.1 Handoff Summary

11 handoff points identified across the canonical pipeline:

| ID | From → To | Stream/Pattern | Type |
|----|-----------|---------------|------|
| H1 | configctl → ingest | `CONFIGCTL_EVENTS` | Config activation |
| H2 | configctl → derive | `CONFIGCTL_EVENTS` | Config activation |
| H3 | ingest → derive | `OBSERVATION_EVENTS` | Market data ingestion |
| H4 | derive → store | 6 event streams | Domain event materialization |
| H5 | derive → execute | `STRATEGY_EVENTS` | Strategy-to-execution (S360) |
| H6 | derive → execute | `EXECUTION_EVENTS` | Paper order intake (transitional) |
| H7 | execute → store | `EXECUTION_FILL_EVENTS` | Venue fill materialization |
| H8 | derive → writer | All event streams | Analytical persistence |
| H9 | execute → writer | `EXECUTION_FILL_EVENTS` | Venue fill persistence |
| H10 | store ↔ gateway | NATS request/reply | Query path |
| H11 | configctl ↔ gateway | NATS request/reply | Config query path |

### 3.2 Contract Consistency

All handoffs use the canonical `Envelope[T]` with CBOR encoding. Contract properties verified:

| Property | Consistent Across All Handoffs |
|----------|-------------------------------|
| Envelope wrapper | Yes — `Envelope[T]` with Kind, Type, Source, Subject |
| CBOR encoding | Yes — all event streams use `application/cbor` |
| CorrelationID propagation | Yes — preserved across all hops |
| CausationID chaining | Yes — parent event ID linked in each hop |
| Deduplication key | Yes — deterministic MsgID per event type |
| Consumer AckWait | Yes — 30s standard |
| Consumer MaxDeliver | Yes — 5 standard |
| Consumer AckPolicy | Yes — Explicit standard |
| Error reporting | Yes — `Problem` type in reply Envelopes |

### 3.3 Correlation Chain Verification

The canonical pipeline preserves a full correlation chain across 3 binary boundaries:

```
ingest (binary 1) → NATS → derive (binary 2) → NATS → execute (binary 3)
                                    ↓ NATS
                              store (binary 4) ← NATS ← execute
                                    ↓ NATS request/reply
                              gateway (binary 5)
```

CorrelationID is set once (at ingest or HTTP entry) and never regenerated. CausationID chains through each processing step. This was verified in S365-S369 at the unit/integration level; S371 confirms the contracts support cross-binary preservation.

---

## 4. Risks and Limits

### 4.1 Risk Register

| ID | Risk | Level | Impact | Mitigation |
|----|------|-------|--------|-----------|
| RISK-1 | Transitional bridge in execute (paper order intake) | LOW | Well-documented, paper-mode only | Scoped, documented migration path |
| RISK-2 | Dual intake paths in execute (strategy + paper) | MEDIUM | Potential duplicate venue submissions | Staleness guard + activation gate |
| RISK-3 | Event metadata loss in KV projections | LOW | Gateway can't trace to originating events | ClickHouse preserves full metadata |
| RISK-4 | Stream creation timing vs compose startup | MEDIUM | Consumer binding may fail if stream not yet created | Compose depends_on + readiness, but window exists |
| RISK-5 | Gateway readiness depends on store | LOW | Expected behavior | Compose ordering handles it |
| RISK-6 | Writer buffer eviction under backpressure | LOW | Analytical events dropped | Counter + stream retention for replay |
| RISK-7 | NATS reconnection during binary restart | LOW | Redelivery of unacked messages | Dedup + idempotent processing |

### 4.2 Limits Identified

1. **KV projections are eventually consistent** — there is no cross-bucket transactional guarantee. Gateway may see strategy projected but not yet execution for the same partition.
2. **No cross-binary ordering guarantee across different streams** — store may process evidence events before or after strategy events for the same market tick.
3. **Execute binary has two independent consumer paths** — S360 strategy wiring and legacy paper bridge. Both are active. This needs validation in S373.
4. **Derive's soft dependency on configctl** — derive starts without configctl and waits for binding events asynchronously. If configctl is slow, derive sits idle but healthy.

### 4.3 Blocking Issues for S372

**None.** All identified risks are documented with mitigations and can be validated in the compose-level proof.

---

## 5. Non-Goals Reaffirmed

Per the S370 charter, the following remain out of scope:

| Non-Goal | Status |
|----------|--------|
| New strategy families | Frozen — only `mean_reversion_entry` |
| Multi-venue | Frozen — paper mode only |
| OMS or portfolio risk | Frozen |
| Runtime redesign | Frozen |
| Multi-symbol orchestration testing | Frozen |
| Dashboards or UI | Frozen |
| ClickHouse schema changes | Frozen |
| Mainnet or live trading | Frozen |
| Contract redesign | Frozen — audit only, no changes |
| Multiple pipelines | Frozen — canonical pipeline only |

---

## 6. Preparation Recommended for S372

### 6.1 What S372 Should Validate

S372 (Compose Orchestration Wiring Validation) should focus on:

1. **Compose boot sequence** — all 7 Go binaries + 2 infrastructure containers reach healthy state in dependency order.
2. **Stream existence after boot** — verify all 9 JetStream streams exist after derive reaches ready.
3. **Consumer binding** — verify all durable consumers are bound to their streams after respective binaries reach ready.
4. **Config activation flow** — activate a config via configctl, verify ingest and derive receive binding events.
5. **NATS connectivity matrix** — each binary can publish/subscribe to its owned/consumed subjects.

### 6.2 Specific Attention Points

1. **RISK-4 (stream creation timing)**: S372 should verify that streams are created before store/execute consumers attempt to bind. If the readiness-to-stream-creation window is too large, consider adding stream creation to derive's readiness check or pre-creating streams in NATS config.

2. **RISK-2 (dual intake paths)**: S372 should log which execute consumer paths are active and verify no duplicate processing occurs in steady state.

3. **Port conflicts**: configctl (8080 internal) and gateway (8080 external) share the same port number but are in different containers — no conflict in compose, but attention needed if testing locally.

### 6.3 Recommended S372 Smoke Tests

| Test | Validates |
|------|----------|
| `docker compose up -d` → all healthy within 60s | MBI-1 (startup ordering) |
| NATS monitoring shows 9 streams | Stream creation |
| NATS monitoring shows all durable consumers | Consumer binding |
| Activate config → ingest logs binding | H1 handoff |
| Activate config → derive logs binding | H2 handoff |
| Publish test trade → store KV has candle | H3 + H4 handoffs |
| Gateway `/readyz` returns 200 | H10 + H11 paths |
| Kill derive → restart → consumers resume | RISK-7 validation |

---

## 7. Deliverables

| Deliverable | Path | Status |
|-------------|------|--------|
| Binary boundary and event-flow contract audit | [`docs/architecture/binary-boundary-and-event-flow-contract-audit.md`](../architecture/binary-boundary-and-event-flow-contract-audit.md) | Complete |
| Multi-binary handoffs, subjects, payloads, invariants and risks | [`docs/architecture/multi-binary-handoffs-subjects-payloads-invariants-and-risks.md`](../architecture/multi-binary-handoffs-subjects-payloads-invariants-and-risks.md) | Complete |
| Stage report (this document) | `docs/stages/stage-s371-binary-boundary-and-event-flow-audit-report.md` | Complete |

---

## 8. Acceptance Criteria Verification

| Criterion | Met | Evidence |
|-----------|-----|---------|
| Binary boundaries are explicit and auditable | Yes | 7 binaries mapped with ownership, streams, dependencies |
| Handoffs and real contracts are documented | Yes | 11 handoffs with subjects, payloads, ordering, invariants |
| Stage prepares compose-level orchestration with low risk | Yes | 7 risks classified (2 MEDIUM, 5 LOW), none blocking |
| Bridges and remaining limits are clear | Yes | 1 transitional bridge documented, 4 limits identified |

---

## 9. Stage Lineage

| Relation | Stage |
|----------|-------|
| Charter | [S370 — Multi-Binary Orchestration Charter](stage-s370-multi-binary-orchestration-charter-report.md) |
| Predecessor evidence | [S369 — Derive Integration Evidence Gate](stage-s369-derive-integration-evidence-gate-report.md) |
| This stage | **S371 — Binary Boundary and Event-Flow Audit** |
| Next | S372 — Compose Orchestration Wiring Validation |
| Wave closure | S375 — Evidence Gate and Wave Closure |
