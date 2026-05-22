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

### Operational decisions and constraints (P1A.8b — pending)

ADRs 0006-0011 will cover configctl as lifecycle authority, paper
venue adapter as default safe mode, single-writer invariant, subject
taxonomy, httprouter trie constraints, and pairing no-OMS boundary.

## Adding new ADRs

When you make a structural decision worth documenting:

1. Pick the next sequential number (`0012`, `0013`, ...).
2. Use the template format above.
3. Cross-reference relevant code and other docs.
4. Reference the ADR from affected docs (e.g., from ARCHITECTURE.md
   if structural).

For PR rules around ADRs, see
[`../CONTRIBUTING.md`](../CONTRIBUTING.md).
