# Resumption

> This document is the first thing to read when returning to
> market-foundry after a pause. It captures the current state, known
> gaps, and the next concrete step.
>
> It is **honest, not aspirational.** If a capability is missing or
> partial, it says so. If a feature is broken, it says where.

Last meaningful state change: **Phase 1B closed — `.opencode/` exterminated; stale repository-consistency-check.sh replaced**.

---

## Current functional state

The system runs end-to-end in paper (dry-run) mode against Binance
WebSocket data. Specifically:

- **All eight binaries build and start cleanly** (`make build`, `make up`).
- **Stack health passes** (`make ps` shows all services healthy).
- **Smoke test passes** (`make smoke` runs the canonical end-to-end
  proof against a real compose stack).
- **Gateway boot is verified at CI time** by
  `cmd/gateway/boot_test.go`, which hermetically registers all routes
  and would fail before container boot if a route trie conflict were
  reintroduced.
- **60 HTTP endpoints are catalogued and reachable** through the
  gateway (subject to conditional registration — see below).
- **ClickHouse persistence is operational**: events from the stream
  mesh land in the analytical tables via the `writer` binary, and
  read endpoints serve them back through the gateway.
- **Forward-only migrations are tracked** in `_migrations` and
  applied via `make migrate-up`.

What was verified concretely during Phase 0 closure (May 2026):

| Verification | Status |
|---|---|
| `make bootstrap` | PASS |
| `make verify` | FAIL — `quality-gate` reports 61 errors from `raccoon-cli drift-detect` against missing `docs/architecture/*.md` (see G6). `make test` and `repo-consistency-check` both PASS. |
| `make build` | PASS for all services |
| `make up` → 9 services healthy | PASS |
| `make smoke` | PASS |
| Gateway boot test | PASS (introduced after P0.6) |
| Three route trie conflicts | FIXED (P0.6 removed lifecycle/list, renamed source-explain and session statics) |
| `cmd/gateway/boot_test.go` regression guard | IN PLACE |

---

## Known gaps in operational state

These are real gaps in the running system. They are not blockers for
development but operators should know they exist.

### G1 — `/execution-source-explain` is unreachable in any environment

The endpoint exists in code (`internal/interfaces/http/routes/source_explain.go`)
and registers conditionally on `deps.GetSourceExplanation != nil`. However,
**no code path in `cmd/gateway/` ever constructs a `GetSourceExplanation`
use case** — `NewGetSourceExplanationUseCase` (defined in
`internal/application/executionclient/get_source_explanation.go`) has no
caller in the gateway composition root. The dep is therefore always
`nil`, the route never registers, and the endpoint returns 404 in any
deployment, not just local default.

The handler also requires a `SourcePathConfigProvider` implementation;
no concrete implementation exists in the repository today.

**Source:** documented as gap WG-1 in
`docs/legacy/architecture/strategy-signal-integration-evidence-matrix-residual-gaps-and-next-ceremony.md`.
The doc is in legacy; the gap is still real.

**Fix:** in `cmd/gateway/` (likely `compose.go`), provide a
`SourcePathConfigProvider` implementation and call
`executionclient.NewGetSourceExplanationUseCase(gateway, configProvider)`,
then pass the result into the `SourceExplainFamilyDeps.GetSourceExplanation`
slot. Until then, expect 404.

### G2 — KV bucket coverage gaps for signals and strategies

Not every signal type and not every strategy type has a corresponding
`_LATEST` KV bucket. Verified against the codebase:

- **Signal:** 2 of 6 types have a bucket (`SIGNAL_RSI_LATEST`,
  `SIGNAL_EMA_CROSSOVER_LATEST`). The remaining 4 (bollinger, macd,
  vwap, atr) flow through `SIGNAL_EVENTS` and persist in ClickHouse
  but have no operational projection.
- **Strategy:** 2 of 3 types have a bucket
  (`STRATEGY_MEAN_REVERSION_ENTRY_LATEST`,
  `STRATEGY_TREND_FOLLOWING_ENTRY_LATEST`). The missing one is
  `squeeze_breakout_entry`.

What this means: events flow through the JetStream mesh and persist
in ClickHouse, but **operational read** via gateway returns nothing
for the uncovered types. Analytical reads (via writer + ClickHouse)
do work.

**Source:** discovered during P1A.4b runtime inventory.
**Status:** unclear whether this is intentional design (some signals
are analytical-only) or oversight. No documented decision either way.

### G4 — HTTP authentication

There is **no authentication** on any gateway endpoint. The default
local deployment binds gateway to `127.0.0.1` only, making loopback
isolation the primary access control. Live deployments are expected
to add a reverse proxy with auth in front.

**Status:** deliberate gap for the local single-operator phase.
Needs to be addressed before any non-loopback deployment.

### G5 — Conditional registration is universal

This is more a documentation gap than a system gap, but operators
need to know: **almost every endpoint is conditionally registered**
based on whether its backing dependency is wired in the gateway
composition root. If a dep is absent, the endpoint silently returns
404 with no indication it would exist when wired.

The conditional endpoints table in [`HTTP-API.md`](HTTP-API.md)
lists each dep gate.

**Status:** by design — allows gateway to start with partial
dependencies. But the silent 404 is operator-hostile and could be
improved (e.g., a `/debug/routes` endpoint listing actually-registered
routes). Future enhancement.

### G6 — `raccoon-cli drift-detect` hardcoded against old topology

`tools/raccoon-cli/src/analyzers/drift_detect.rs` contains 91 hardcoded
references to `docs/architecture/*.md` files, asserting the presence of
family architecture design docs (signal, decision, strategy, risk,
execution) that never existed in the working tree. The analyzer was
designed to enforce a future state.

After the Phase 1A reset, `docs/architecture/` (subdir) no longer
exists — replaced by `docs/ARCHITECTURE.md` (singular file) and
`docs/domain/` (subdir of 11 docs).

**Effect:** `make quality-gate` fails with 61 errors, which propagates
to `make verify` failing.

**Resolution path:** rewrite or remove the `drift_detect.rs` analyzer
to align with the Phase 1A docs topology. This is non-trivial Rust
work and is scoped for P1D (governance) or P2 (environment hardening),
not P1B.

**Status:** known debt; documented; `make verify` red until resolved.

---

## Known surface debt

These are quirks that don't block usage but are visible debt that a
future cleanup wave should address.

### D1 — Hyphenated HTTP paths from P0.6

Three paths use hyphens for naming, an unusual choice forced by
httprouter trie limitations:

- `/session-list` (was `/session/list`)
- `/session-batch-audit` (was `/session/batch-audit`)
- `/execution-source-explain` (was `/execution/source-explain`)

These coexist with non-hyphenated wildcard paths like `/session/:id`
which couldn't move. The result is a mildly inconsistent URL surface.

**Resolution path:** a future API redesign wave. Not urgent — the
endpoints work fine; only aesthetic.

### D2 — `execute` config sprawl + `s449` namespace residue

Seven of twelve config files under `deploy/configs/` are variants of
`execute`:

- `execute.jsonc`
- `execute-mainnet-dry-run.jsonc`
- `execute-mainnet-live.jsonc`
- `execute-mainnet-live-s449.jsonc`  ← stage-tagged
- `execute-unified.jsonc`
- `execute-venue-live.jsonc`
- `execute.env.example`

At least one (`execute-mainnet-live-s449.jsonc`) carries a stage
reference in its filename. Since stage-based governance was retired
(decision Y of the reset), the `s449` namespace is dead weight.

**Resolution path:** a config consolidation pass. Either flatten
into one execute config with environment-variant overlays, or at
minimum rename to drop `s449`.

### D3 — configctl subject namespace ambiguity (singular vs plural)

The configctl family currently uses **both** singular
(`configctl.event.config.*`) and plural (`configctl.events.config.*`)
subject patterns in parallel. This is a transitional surface — one
was being migrated to the other, but the migration never completed.

**Resolution path:** pick one, audit all publishers and consumers,
deprecate the other. Coordinated change required across multiple
files in `internal/adapters/nats/natsconfigctl/`.

### D4 — Stage-tagged smoke targets in Makefile

The Makefile has ~23 smoke targets in total, of which ~14 are
stage-tagged (`smoke-compose-wiring` (S372), `smoke-failure-isolation`
(S374), `smoke-live-listening` (S378), `smoke-live-dry-run` (S380),
`smoke-segmented-compose` (S394), and similar). These were used
during the previous evolution model where each stage produced a
dedicated smoke. They still exist but no longer fit the operational
model.

**Resolution path:** prune in a cleanup wave. Most likely keep the
~9 functional smoke targets (smoke, smoke-multi, smoke-analytical,
smoke-round-trip, smoke-composed, smoke-live-stack, smoke-operational,
smoke-restart-recovery, smoke-help) and move the stage-tagged ones
out — either delete, or relocate to `scripts/historical/` for
archaeology.

---

## Recently resolved (Phase 1B)

### G3 — `make verify` cross-references (originally framed as 9 failures from `.opencode/`)

The original framing of G3 ("9 failures, all from `.opencode/`
cross-refs") was inaccurate. P1B uncovered the truth in three layers:

1. **`.opencode/` directory** existed and had 1 cross-reference check
   failing. **Resolved** by deletion in P1B.

2. **`scripts/repository-consistency-check.sh`** had ~7 checks failing
   because the script was hardcoded against the pre-reset docs topology
   (`docs/product/`, `docs/architecture/`, `docs/development/`,
   `docs/stages/`, `docs/archive/`, `docs/tooling/`) which moved to
   `docs/legacy/` in P1A.1. The script was never updated during Phase
   1A because the failure was misattributed to `.opencode/`.
   **Resolved** in P1B by replacement with a minimal stub aligned with
   the current Phase 1A topology (`scripts/repository-consistency-check.sh`,
   ~100 lines).

3. **`tools/raccoon-cli/src/analyzers/drift_detect.rs`** is a separate
   failing layer (61 errors) that was invisible in the original
   framing. **Escalated as G6**, not resolved in P1B (out of scope —
   `tools/` was off-limits).

**Net effect:** P1B resolved two of the three underlying layers.
`make verify` is still red because of G6. The "9 failures from
`.opencode/`" narrative was triply wrong (count of root causes,
attribution of the root, and missing an entirely separate failing
layer) and is corrected here so future readers learn from the error
rather than inherit it.

### D5 — `.opencode/` directory still present

**Resolved** by P1B. The directory was the navigation layer for an
external agent tool (OpenCode CLI). It has been deleted in its
entirety (37 files). The agentic layer will be rebuilt from scratch
in P1C using the Anthropic ecosystem (CLAUDE.md root + `.claude/`).

---

## Deliberate non-features

This section is as important as the gaps section. Each item below
is **intentionally not implemented**. Adding any of them requires a
deliberate design decision (an ADR), not an opportunistic PR.

### N1 — No backtesting harness

There is no mechanism to replay historical ClickHouse data through
the pipeline deterministically. Strategies must currently be tested
in paper mode against live WebSocket data.

This is the most-likely **next major feature**. The infrastructure
exists (PaperVenueAdapter, ClickHouse history, deterministic event
deduplication), but the runner that pulls history and replays it is
absent.

### N2 — No PnL aggregation per strategy

The `effectiveness` domain classifies individual round-trips into
win/loss/breakeven/unresolved. There is no aggregator that produces
"strategy X earned Y net over period Z, with max drawdown W".
Without this, you cannot quantitatively rank strategies or decide
when to retire one.

### N3 — No portfolio-level position sizing

Decisions are local per symbol. The `risk` domain checks
position-exposure and drawdown limits per assessment, but there is
no central model managing aggregate exposure across the portfolio.

### N4 — No multi-exchange surface

A single venue family (Binance, with Spot and Futures sub-segments).
Adding OKX, Bybit, or any other exchange would require a new adapter
under `internal/adapters/exchanges/` and corresponding execution
adapters. This is not currently scoped.

### N5 — No market-making primitives

No order book depth tracking, no queue position estimation, no
inventory risk model. The system is currently designed for momentum
and mean-reversion strategies, not market making.

### N6 — No machine learning pipeline

Signals are deterministic indicators (RSI, EMA, MACD, Bollinger,
ATR, VWAP). There is no training loop, no model registry, no
inference service.

### N7 — No HTTP authentication

Already mentioned in G4. Restated here for completeness — this is a
deliberate gap for the local single-operator phase, not a missing
feature in the usual sense.

---

## Where we are in the resumption cycle

The resumption from a 2-month pause is being executed in phases.
Each phase has a clear exit criterion.

| Phase | Goal | Status |
|---|---|---|
| **Phase 0** | Unblock — fix git limbo, align Go version, get smoke passing | **CLOSED** (commits up to 8900694, mid-May 2026) |
| **Phase 1A** | Documentation reset — move legacy, write new docs | **CLOSED** (17 sub-prompts, 36 docs, May 2026) |
| **Phase 1B** | Exterminate `.opencode/` | **CLOSED** (G6 escalated; see Recently resolved) |
| Phase 1C | Build `.claude/` agentic layer | Pending — next |
| Phase 1D | PR-based governance (PR template, CONTRIBUTING, issue templates) | Pending — depends on P1C; also owns G6 resolution candidate |
| Phase 2 | Environment hardening (CI, Docker, scripts, Makefile cleanup) | Pending — depends on P1 closed; also owns G6 resolution candidate |
| Phase 3 | First feature wave (most likely: backtesting — see N1) | Future |
| Phase 4+ | Subsequent waves (PnL aggregation, multi-exchange, etc.) | Future |

Phase 1A subdivision (status at time of this doc):

| Sub-phase | Goal | Status |
|---|---|---|
| P1A.1 | Move docs/ → docs/legacy/ + new scaffolding | Done |
| P1A.2 | docs/README, docs/legacy/README, docs/GLOSSARY | Done |
| P1A.3 | docs/ARCHITECTURE.md | Done |
| P1A.4a | Runtime inventory (read-only, /tmp) | Done |
| P1A.4b | docs/RUNTIME.md | Done |
| P1A.4b.1 | Errata correcting ARCHITECTURE.md and GLOSSARY.md | Done |
| P1A.4c | docs/HTTP-API.md | Done |
| P1A.5a | docs/DEVELOPMENT.md | Done |
| P1A.5b | docs/RESUMPTION.md (this document) | Done |
| P1A.6 | Domain docs under docs/domain/ | Done |
| P1A.7 | Operations docs under docs/operations/ | Done |
| P1A.8 | Initial ADRs under docs/decisions/ | Done |
| P1A.9 | docs/CONTRIBUTING.md | Done |

---

## Concrete next step

The immediate next step is **P1C — build the `.claude/` agentic
layer** using the Anthropic ecosystem (CLAUDE.md root + `.claude/`
directory). This replaces the `.opencode/` navigation layer that was
deleted in P1B.

P1C scope (to be refined in its own prompt):
- Write `CLAUDE.md` at the repository root with concise, project-
  specific guidance for Claude Code.
- Create `.claude/` directory structure with agent definitions, slash
  commands, and any project-local skills relevant to market-foundry
  work.
- Wire it together so the agentic layer is discoverable from a fresh
  clone without reading the whole `docs/` tree.

After P1C: **P1D** (PR-based governance) opens. G6 (raccoon-cli
drift-detect) is a candidate for resolution in either P1D or P2.

`make verify` will remain red until G6 is addressed. This is
documented debt, not regression.

---

## How to keep this document fresh

`RESUMPTION.md` only earns its keep if it stays current. The trigger
for updating it is:

- **Phase transition** (e.g., when Phase 1A closes, update the phase
  table to show 1B in progress).
- **New known gap discovered** (add to G section).
- **Gap resolved** (move from G section to a "Recently resolved"
  appendix, or just remove).
- **Significant feature shipped** (add to "Current functional
  state", remove from "Deliberate non-features" if applicable).

If you find yourself wondering whether this doc reflects reality,
**that itself is the trigger to update it**.

---

## Reading further

| If you want | Go to |
|---|---|
| System overview | [`README.md`](README.md) |
| Architecture | [`ARCHITECTURE.md`](ARCHITECTURE.md) |
| Topology, ports, streams | [`RUNTIME.md`](RUNTIME.md) |
| HTTP endpoints | [`HTTP-API.md`](HTTP-API.md) |
| Daily workflow | [`DEVELOPMENT.md`](DEVELOPMENT.md) |
| PR rules | [`CONTRIBUTING.md`](CONTRIBUTING.md) |
| Domain deep dives | [`domain/`](domain/README.md) |
| Operational procedures | [`operations/`](operations/README.md) |
| Architecture decision records | [`decisions/`](decisions/README.md) |
| Historical material | [`legacy/`](legacy/README.md) |
| Terminology | [`GLOSSARY.md`](GLOSSARY.md) |
