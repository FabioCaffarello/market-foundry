# Architecture Decision Records

This directory holds Architecture Decision Records (ADRs) — short
documents capturing the durable structural decisions of
market-foundry.

## Why ADRs

Code shows **what** the system does. Comments explain **how** small
pieces work. ADRs explain **why** the system is shaped the way it is.

Without ADRs, future contributors (or future-you) face questions like
"why didn't we use Kafka instead of NATS" or "why is layer sovereignty
enforced statically rather than by convention" — and have to
reverse-engineer the answer from code archaeology.

## Format

Each ADR follows this shape:

| Section | Purpose |
|---|---|
| Status | Accepted, Superseded, Deprecated |
| Context | The situation that motivated the decision |
| Decision | What we decided |
| Consequences | Positive and negative outcomes |
| Alternatives considered | What else we considered, briefly |
| References | Cross-refs to docs and code |

ADRs are **append-only**. If a decision changes, write a new ADR that
supersedes the old one; do not edit historical records. (Exception:
typo fixes and broken-link corrections are fine.)

## Index

### Core decisions (P1A.8a)

| # | Title | Status |
|---|---|---|
| [0001](0001-nats-not-kafka.md) | NATS + JetStream as sole messaging infrastructure | Accepted |
| [0002](0002-hollywood-actor-framework.md) | Hollywood as sole concurrency primitive | Accepted |
| [0003](0003-clickhouse-analytical.md) | ClickHouse for analytical storage | Accepted |
| [0004](0004-raccoon-cli-static-enforcement.md) | raccoon-cli for static architecture enforcement | Accepted |
| [0005](0005-layer-sovereignty.md) | Layer sovereignty enforced statically | Accepted |

### Operational decisions and constraints (P1A.8b)

| # | Title | Status |
|---|---|---|
| [0006](0006-configctl-lifecycle-authority.md) | configctl as lifecycle authority | Accepted |
| [0007](0007-paper-venue-default.md) | paper venue adapter as default safe mode | Accepted |
| [0008](0008-single-writer-invariant.md) | Single-writer invariant per stream and KV bucket | Accepted |
| [0009](0009-subject-taxonomy.md) | NATS subject taxonomy with verb-last pattern | Accepted |
| [0010](0010-httprouter-trie-constraints.md) | httprouter trie constraints respected | Accepted |
| [0011](0011-no-oms-expansion-pairing.md) | No OMS expansion in pairing and effectiveness | Accepted |

### Phase 4 decisions

| # | Title | Status |
|---|---|---|
| [0012](0012-control-gate-fail-open-posture.md) | ControlGate fail-open posture | Accepted |

### Phase 5 decisions — process patterns

| # | Title | Status |
|---|---|---|
| [0013](0013-pause-and-report-protocol.md) | Pause-and-report protocol | Accepted |
| [0014](0014-defensive-scan-discipline.md) | Defensive-scan discipline | Accepted |
| [0015](0015-wave-closure-discipline.md) | Wave-closure discipline | Accepted |
| [0026](0026-claude-code-hooks-enforcement.md) | Claude Code hooks as enforcement layer for P2/P9 | Accepted |

### Fase Harvest decisions

| # | Title | Status |
|---|---|---|
| [0016](0016-harvest-from-market-raccoon.md) | Harvest from market-raccoon — selective wave protocol | Accepted |

### Fase Harvest — Foundation ADRs (Onda H-2)

Foundation ADRs delivered in **Onda H-2**. Each ADR codified a
structural decision before the implementing code landed; each
carries an explicit "Promoção para Accepted" section naming the
onda that ships the supporting code and flips the status. While
`Proposed`, these ADRs are T3 (Evolutionary) per
[`../AUTHORITY.md`](../AUTHORITY.md); they become T1 (Canonical) on
promotion. The hybrid status — design recorded now, code-grounded
acceptance later — is the P7 mechanism for "no aspirational claims"
(I9 in [`../operations/runtime-invariants.md`](../operations/runtime-invariants.md)).
Four of the seven have since been promoted by their implementing
ondas; 0021–0023 remain `Proposed`.

| # | Title | Status | Promoted by |
|---|---|---|---|
| [0017](0017-event-envelope-and-versioning.md) | Event envelope and versioning | Accepted | Onda H-3.b (2026-05-25) |
| [0018](0018-protobuf-contract-layer.md) | Protobuf contract layer | Accepted | Onda H-3.b (2026-05-25) |
| [0019](0019-deterministic-replay-time-invariants.md) | Deterministic replay and time invariants | Accepted | Onda H-4 (2026-05-25) |
| [0020](0020-sequencing-and-time-normalization.md) | Sequencing and time normalization | Accepted | Onda H-4 (2026-05-25) |
| [0021](0021-canonical-instrument-and-venue-model.md) | Canonical instrument and venue model | Proposed | Onda H-6 (atômica em H-6.f) |
| [0022](0022-multi-venue-normalization-policy.md) | Multi-venue normalization policy | Accepted (2026-06-12, H-7.b) | Onda H-7 |
| [0023](0023-storage-tier-roadmap.md) | Storage tier roadmap | Proposed | Partial: H-9; full: H-10 |
| [0027](0027-insights-decision-support.md) | Insights são decision-support read-only | Proposed | Onda H-8.a (PROGRAM-0005) |

### Fase Harvest — Observability decisions (Onda H-5)

Delivered and promoted to `Accepted` within Onda H-5
(PROGRAM-0003 Observability, PR #25, 2026-05-25).

| # | Title | Status |
|---|---|---|
| [0024](0024-metrics-policy.md) | Metrics policy | Accepted |
| [0025](0025-alerting-strategy.md) | Alerting strategy | Accepted |

## Adding new ADRs

When you make a structural decision worth documenting:

1. Pick the next sequential number (`0027`, `0028`, ...).
2. Use the template format above.
3. Cross-reference relevant code and other docs.
4. Reference the ADR from affected docs (e.g., from ARCHITECTURE.md
   if structural).

For PR rules around ADRs, see
[`../CONTRIBUTING.md`](../CONTRIBUTING.md).
