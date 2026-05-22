# CLAUDE.md

Operating instructions for Claude (and other AI agents) working on
market-foundry.

This file is read automatically by Claude when starting a session
in this repository. It is intentionally concise — for depth, follow
the references.

---

## What this repository is

market-foundry is a Go workspace foundation for cryptocurrency market
data processing. It is **not** a trading application — it is the
foundation on which trading capabilities are built.

Seven long-running binaries (configctl, gateway, ingest, derive, store,
execute, writer) plus one one-shot tool (migrate), communicating via
NATS+JetStream. ClickHouse for analytical storage. Rust raccoon-cli
for static architecture enforcement.

For higher-level orientation, see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

---

## Reading order for any new session

When starting work on this repository, read in this order:

1. **The prompt or task you received** — your immediate context.
2. **[docs/RESUMPTION.md](docs/RESUMPTION.md)** — current state, known
   gaps, next concrete step. Always start here.
3. **[docs/CONTRIBUTING.md](docs/CONTRIBUTING.md)** — operational rules,
   PR workflow, "Specifically for AI agents" section.
4. **[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)** — system shape and
   structural principles.
5. **Specific docs your task needs** (e.g., domain docs, HTTP-API,
   runtime, operations).

Time investment: 5-10 minutes for documents 2-3 minimum, on every
session. Skipping this leads to misaligned work.

---

## Core operating protocols

These protocols are non-negotiable. They emerged from real lessons
during Phases 0 and 1A.

### 1. Validate against code before claiming facts

Documentation can be stale. Code is the source of truth. Before
asserting a technical fact in a doc or commit message, **verify it
against the codebase** with a concrete grep, find, or read.

This rule emerged because multiple prompts during Phase 1A produced
draft content with factual divergences from code (stream counts,
consumer ownership, plane taxonomy, type lists). The "verify before
save" pattern caught them.

### 2. Pause and report on divergence

If you encounter:
- A blocker or improvement **outside the prompt's scope**,
- A **discrepancy** between expected and actual state,
- An **ambiguity** in the task that needs clarification,

**stop, report concisely, and present options (A/B/C/D).** Wait for
direction before proceeding.

This is the "authorized expansion protocol" — see
[docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) → "Authorized expansion
protocol" for the canonical 5-step procedure with examples.

Silent expansion is forbidden. Silent skipping is forbidden.

### 3. Honesty over convenience

When a failure has a convenient categorization, **investigate more, not
less**. Convenient categorization is exactly when verification is most
likely to lapse.

Concrete example: during Phase 1A, `make verify` failures were
attributed to ".opencode/ cross-refs" for 18 prompts. P1B investigation
revealed the framing was triply wrong (wrong count, wrong attribution,
missed an entire failing layer in `tools/raccoon-cli/`). The convenient
narrative cost real work.

If a report contains "and similar" or "and related" or any hedge
phrase, investigate the hedge before adopting the categorization.

### 4. Single-writer invariant

Every JetStream stream, every NATS KV bucket, every NATS query subject
has **exactly one writer**. No exceptions. This is the most important
invariant in the system; preventing it by construction is much cheaper
than debugging the race conditions it would otherwise allow.

See [docs/decisions/0008-single-writer-invariant.md](docs/decisions/0008-single-writer-invariant.md).

### 5. Adding an HTTP route requires updating boot_test.go

If your change adds a route to `internal/interfaces/http/routes/`, you
**must also** add it to `cmd/gateway/boot_test.go`'s `routes` slice.

The boot test exists as a regression guard for httprouter trie
conflicts (lesson from Phase 0 where 3 simultaneous conflicts caused
gateway CrashLoopBackoff). CI will fail your PR if you forget.

See [docs/decisions/0010-httprouter-trie-constraints.md](docs/decisions/0010-httprouter-trie-constraints.md).

### 6. Layer sovereignty is enforced

Imports flow inward only:
`domain → application → adapters → actors → interfaces → cmd`.

raccoon-cli enforces this in `make verify`. A violating import does
not ship.

See [docs/decisions/0005-layer-sovereignty.md](docs/decisions/0005-layer-sovereignty.md).

---

## Essential commands

For complete daily workflow, see [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md).
The basics:

```bash
make bootstrap           # validate prerequisites (always works)
make check               # pre-code guard rail
make tdd                 # impact-driven validation guidance
make verify              # post-change validation (green since P1D.4)
make smoke               # canonical end-to-end proof
make up                  # bring up stack
make down                # tear down stack
```

### State of make verify

`make verify` is **green** end-to-end. The historical G6 issue
(`tools/raccoon-cli/src/analyzers/drift_detect.rs` hardcoded against
the pre-reset documentation topology) was resolved in P1D.4
(commit 557a508), which realigned the const tables to the Phase 1A
topology while preserving the 27 other working drift checks. See
[docs/RESUMPTION.md](docs/RESUMPTION.md) → "Recently resolved" for
the historical detail.

A red `make verify` going forward indicates a real regression, not
historical debt. Investigate before merging.

---

## What this repository is NOT

This list is as important as what it is. Avoid assuming features
that don't exist. From [docs/RESUMPTION.md](docs/RESUMPTION.md):

- **No backtesting harness.** Strategies test in paper mode against
  live data.
- **No PnL aggregation per strategy.** Effectiveness classifies
  individual round-trips only.
- **No portfolio-level position sizing.** Decisions are local per
  symbol.
- **No multi-exchange surface.** Single venue family (Binance Spot +
  Futures).
- **No market-making primitives.**
- **No machine learning pipeline.**
- **No HTTP authentication.** Loopback binding is the access control.

If asked to use any of these, clarify with the user before proceeding.

---

## Boundaries by default

Unless a prompt explicitly authorizes otherwise, do not modify:

- `internal/`, `cmd/`, `tools/`, `deploy/` — code, configs, tooling.
- `docs/` already-written files (Phase 1A is closed; changes go in
  follow-up prompts).
- `Makefile`, `.gitignore`, `go.work`, `go.mod` — repository
  infrastructure.

If your task requires modifying these, expect the prompt to call it
out. If it doesn't, ask before touching.

---

## When in doubt

Pause and ask. The cost of one extra clarification turn is much less
than the cost of an incorrect autonomous decision that requires
unwinding. Multiple prompts during Phase 1A confirmed this empirically.

For the canonical pause-and-report procedure, see
[docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) → "Authorized expansion
protocol".

---

## Reading further (canonical map)

| If you want | Go to |
|---|---|
| Current state and gaps | [docs/RESUMPTION.md](docs/RESUMPTION.md) |
| Operating rules and protocols | [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) |
| System architecture | [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) |
| Runtime topology | [docs/RUNTIME.md](docs/RUNTIME.md) |
| HTTP endpoints | [docs/HTTP-API.md](docs/HTTP-API.md) |
| Daily workflow | [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) |
| Operations guides | [docs/operations/](docs/operations/README.md) |
| Architecture decisions | [docs/decisions/](docs/decisions/README.md) |
| Domain deep dives | [docs/domain/](docs/domain/README.md) |
| Terminology | [docs/GLOSSARY.md](docs/GLOSSARY.md) |
