# Resumption

> This document is the first thing to read when returning to
> market-foundry after a pause. It captures the current state, known
> gaps, and the next concrete step.
>
> It is **honest, not aspirational.** If a capability is missing or
> partial, it says so. If a feature is broken, it says where.

Last meaningful state change: **Phase 4 CLOSED (2026-05-23)** —
P0 backlog FULLY CLOSED (5/5 items: CI restoration via the P4.1
wave, `rate_limiter` + `Close` lifecycle, `context.Background()`
bounding, ControlGate fail-open posture formalized as ADR-0012,
and the Dependabot triage wave). 12 ADRs total; 20 design-meta
candidates queued (M1–M20, M19 closed during P4.5.c verification);
~9 errata observations accumulated.

`make verify` GREEN locally; CI 7/7 GREEN at `main` HEAD, sustained
since P4.1.1's SHA-pinning migration. Some intermediate Dependabot
merges show the documented `TestControlledActivation_FullLifecycle`
/ `TestRealVenueActivation_FullLifecycle` Integration Tests timing
flake; these are non-required and non-blocking per branch protection
(see Phase 4.5 narrative for full posture).

Phase 5 OPENED (2026-05-23) — environment work, distinct from
Phase 4's code/CI delivery. P5.0 audit (read-only, ~7 min wall-clock)
categorized 12 findings across `.claude/`, prompt templates,
operational tooling, and process debt; produced an 8-slot P5.x
candidate roadmap pending owner direction. See "Phase 4 outlook"
below for retrospective detail; Phase 5 narrative pending P5.1+.

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
| `make verify` | PASS (since P1D.4 — G6 resolved, see "Recently resolved"). All 6 active quality-gate analyzers green; 84 checks, 0 errors. |
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

**Source:** originally documented as gap WG-1 in the pre-reset
strategy-signal integration evidence matrix (retired in P2.Y; recoverable
via `git log`). The gap itself is still real.

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

## Recently resolved

### Phase 4.1 wave — CI restoration + quality gate cleanup

**Resolved** by 9 sub-prompts that took CI from red to fully green
on the quality-gate-ci job, clearing all 11 ci-profile warnings
surfaced after the Phase 4.1 SHA pinning migration lifted the
workflow-rejection layer that had masked latent failures since P3.3.

Sub-prompt summary:

- **P4.0** — documental hygiene sweep (DOC-1 through DOC-5) plus the
  P0-6 `SC2206` fix in `scripts/utils/lib.sh` that P3.5.safety had
  missed (scope was `scripts/*.sh`, not the `utils/` subtree).
- **P4.1** — CI workflow SHA pinning migration. 6 actions converted
  from tag refs (`@v4`, `@v5`) to commit SHAs. Branch protection
  rule `sha_pinning_required` (enabled in P3.3) became enforceable.
  Commit `4b5f14c`.
- **P4.1.1** — `golangci-lint-action` v6 → v9. The v9 binary takes
  `install-only` instead of the v6 `args` form; the v6 args were
  silently ignored on v9 (latent CI red). Commits `83e222e`,
  `899f4b5`.
- **P4.1.2** — Read-only investigation of `make quality-gate-ci`.
  Surfaced 11 pre-existing warnings now severity-promoted to errors
  by the `ci` profile (`tools/raccoon-cli/src/gate/mod.rs`). No
  fixes; categorisation only.
- **P4.1.3.a** — `drift-detect` `CANONICAL_STREAMS` aligned with
  the current `internal/adapters/nats/natsexecution/registry.go`
  set. G6.2: same pattern as the G6 fix at `557a508`, for streams
  added later. Commit `7ea24cd`.
- **P4.1.3.a'** — `contract-audit` alignment for the
  SessionLifecycle event: subject pattern widening, move from the
  ad-hoc `session_lifecycle_event.go` into the canonical
  `events.go`, addition of the `Metadata` field required by the
  domain event convention. Commit `41966a7`.
- **P4.1.3.b** — `_test.go` exemption added to the `deploy-boundary`
  check in `tools/raccoon-cli/src/analyzers/arch_guard.rs`. Tests
  asserting on canonical deploy paths is legitimate behaviour;
  extracting to constants would create indirection just to satisfy
  a scanner. Commit `6f9efd5`.
- **P4.1.3.c.i** — Read-only `cmd-boundary` mini-investigation.
  3 of 4 violations were TYPE-ONLY (composition wiring), 1 was
  MIXED (a single `execution.ComputeEffectiveMode` call from
  `cmd/execute/run.go` used for startup logging). Verdict: rule
  overshoots ADR-0005's "cmd sees everything" and is inconsistent
  with the application-client public contracts.
- **P4.1.3.c.ii** — `cmd-boundary` rule refined to flag domain
  function invocations only, permitting type/constant/struct-literal
  references. Implementation: text-pattern detection seeded by the
  codeintel `ProjectIndex` (functions known from the parsed AST).
  Go side adds `internal/application/executionclient/compute_effective_mode.go`
  wrapping the domain function; `cmd/execute/run.go` routes through
  the wrapper. Commit `25839ea`.
- **P4.1.5 / P4.1.6.a*** — NATS+JetStream infrastructure restoration
  for the Integration Tests job. Services-block startup was unreliable
  on the GitHub runner; switched to `docker run --network host` with
  the NATS monitor bound on port 8222 (`-m 8222`). Commits `d2238a0`,
  `5c8d0ff`.
- **P4.1.7** — Domain failure triage on the integration suite once
  NATS came up. Surfaced a P3 counter race: tests asserted on
  `tracker.Counter("filled")` immediately after the actor published
  the fill, but the counter was incremented after publish, leaving
  a sub-microsecond window for the read to miss the increment.
- **P4.1.8** — `eventuallyAtLeast` poll helper introduced and applied
  across 11 test sites that read execute-scope counters synchronously
  after a publish. Commit `81a2319`.
- **P4.1.8.a** — Suite timeout extension. The newly-polling tests
  pushed the suite above the 10-min default; bumped `-timeout 18m`
  in the Makefile target and the CI workflow timeout to 20 min.
  Commit `a5fff7c`.
- **P4.1.8.b** — Defensive completion: 5 additional counter-read
  sites identified during the scan-and-catch-up pass were converted
  to the helper. Commit `a378117`.
- **P4.1.8.c** — Read-only investigation of the counter-ordering
  question raised in the architect META review ("is the helper a
  band-aid for an actor-ordering bug?"). Findings: 11 non-test
  counter readers, all intra-actor self-reads (race-free by
  Hollywood single-threaded mailbox); only external surface is HTTP
  `/statusz`, whose multi-ms timing dominates the ~500µs race
  window; Prometheus uses a separate counter set. No current
  production consumer can observe the invariant violation. Owner
  decision: **Option (C)** — accept helper, defer actor reorder,
  document the trade-off.
- **P4.1.8.d** — P4.1.8 wave closure. Counter-ordering decision
  documented in `internal/actors/scopes/execute/venue_adapter_actor.go`;
  M7 ("dual-semantic counter for pre-publish vs post-publish
  observability") added to the design-meta queue; `-short` flag
  added to the Makefile `test-integration` target so the existing
  `testing.Short()` guards on 6 endurance/extended-observation
  tests become active in PR CI, dropping the suite from ~18m to
  ~1-2m. Long-running tests remain runnable locally without
  `-short`, or in a future nightly schedule.
- **P4.1.6.b** — Smoke Analytical E2E moved out of PR CI to a
  dedicated workflow (`.github/workflows/smoke-analytical.yml`)
  with `workflow_dispatch` (manual via `gh workflow run
  smoke-analytical.yml`) and `schedule: cron '0 6 * * *'` (daily
  06:00 UTC) triggers. Architectural rationale: PR CI is a
  fast-feedback loop; integration tests against external services
  (live Binance WSS) don't belong there. Job definition preserved
  verbatim (same steps, SHA pins, env vars, timeout); only the
  trigger surface changed. M8 (synthetic seeder pre-requisite for
  restoring smoke-analytical to PR CI) and M9 (log-error scan
  robustness — current warn-vs-error grep missed the silent failure
  mode) added to the design-meta queue.
- **P4.1.10** — Strategy dedup key precision fix. P4.1.9
  investigation (read-only) diagnosed three persistently-failing
  rapid-publish family tests (S380-DR-4, S373-MB-2/phase-2,
  E2E-2/phase-2) as a domain-layer bug: `Strategy.DeduplicationKey()`
  used `Timestamp.Unix()` (whole-second precision), so multiple
  publishes within a single wall-clock second produced identical
  `Nats-Msg-Id` values and were silently dropped by JetStream's
  2-minute Duplicate Window. Production was unaffected (kline cadence
  ≥1s never exercises this); tests tripped the bug because they
  publish siblings in tight loops. Fix: switch to `Timestamp.UnixNano()`.
  Also added `PubAck.Duplicate` warn-log surfacing in
  `internal/adapters/nats/natsstrategy/publisher.go` so future
  similar bugs are not silent (the operational blind spot P4.1.9
  noted as surpresa #2). Bug introduced in commit `fa8f04a5`
  ("initial quick start") and was latent through Phases 1–3 and
  most of Phase 4 — surfaced only after P4.1 lifted CI SHA-pinning
  rejection. Counter increment for `dedup_dropped` was intentionally
  omitted to keep blast radius to a single file (Publisher has no
  tracker field; wiring one would change the constructor signature
  across 15 callsites).
- **P4.1.11** — Time-capped abbreviated investigation (5:20 min
  finish vs 20-min cap) of a newly-visible writerpipeline failure
  that surfaced once P4.1.10 unmasked the prior layer. Found the
  same Subject-as-prefix mismatch pattern as P4.1.3.a'
  SessionLifecycle: 9 test sites across two files
  (`writerpipeline/restart_recovery_test.go` and
  `natsexecution/restart_recovery_test.go`) build `ConsumerSpec` by
  hand using the bare `registry.PaperOrderSubmitted` EventSpec
  (subject `execution.events.paper_order.submitted`, no wildcard).
  The consumer fallback at `natsexecution/consumer.go:79` then sets
  `FilterSubject` to that bare value, which does not match
  publishers' qualified subjects
  (`execution.events.paper_order.submitted.{source}.{symbol}.{timeframe}`).
  Production paths use helper specs
  (`ExecuteStrategyMeanReversionEntryConsumer`,
  `ExecuteVenueMarketOrderIntakeConsumer`) that supply the `.>`
  wildcard form, which is why production and family tests were
  unaffected. Investigation report captured at
  `/tmp/p4.1.11-writerpipeline-investigation.md` (173 lines).
- **P4.1.11.a** — Bundled three-part fix that closes the Phase 4.1
  wave. Initial scope (subject-filter helper) discovered two more
  pre-existing layers during local repro; each was the **same bug
  class as an earlier wave fix**, not a genuinely new architectural
  concern, so they were folded into the same commit rather than
  spawning further sub-prompts:

  1. **Subject filter** — new
     `WriterPaperOrderExecutionConsumerForTest(durable string)`
     helper in `internal/adapters/nats/natsexecution/registry.go`
     mirroring the codegen-managed `WriterPaperOrderExecutionConsumer()`
     but accepting a caller-supplied durable. 9 spec construction
     sites updated across `writerpipeline/restart_recovery_test.go`
     (4) and `natsexecution/restart_recovery_test.go` (5);
     `natskit` import dropped from the writerpipeline test (no
     longer referenced). Same root-cause class as P4.1.3.a'
     SessionLifecycle subject mismatch.
  2. **Test-isolation reset** — new
     `ResetExecutionEventsStreamForTest(url string)` helper in the
     same registry file. Best-effort `js.DeleteStream` of
     `EXECUTION_EVENTS` at the top of each affected test so the
     shared NATS container (re-used across tests in the integration
     suite) does not replay one test's events into a later test's
     fresh durable. 9 reset calls inserted. The same
     `JSErrCodeStreamNotFound` swallow pattern as production
     `consumer.go` is used for the "first run, nothing to delete"
     case.
  3. **DeduplicationKey precision (completion of P4.1.10)** —
     P4.1.10 fixed `Strategy.DeduplicationKey()` (Unix → UnixNano)
     because the family tests it targeted only published
     strategies. The same `Timestamp.Unix()` precision bug existed
     in `ExecutionIntent`, `Decision`, `RiskAssessment`, and
     `Signal` (4 sibling timestamp-keyed types in
     `internal/domain/`). The restart_recovery tests publish
     `PaperOrderSubmittedEvent` which embeds `ExecutionIntent` —
     so the same silent JetStream Duplicate-Window drop reappeared
     for tests that publish siblings within a wall-clock second.
     All 4 sibling impls switched to `UnixNano()`; 4 unit-test
     format assertions updated (`execution_test.go`,
     `decision_test.go`, `risk_test.go`,
     `signal_test.go` — the last required adding `fmt` to its
     imports since the previous hardcoded literal was replaced
     with `fmt.Sprintf`). Production cadence (kline ≥1s) keeps
     this latent for all four types in prod; the latency surfaces
     only under tight-loop test publishes.

  Cumulative effect: Phase 4.1 wave fully closes; PR CI returns
  to 7/7 GREEN. M11 (subject-filter validation in `consumer.go:79`
  fallback) added to the design-meta queue as the architectural
  follow-up — the test-side helper prevents the manifestation but
  the fallback path remains a quiet footgun for any future test
  that bypasses production helpers. M12 (audit all timestamp-keyed
  `DeduplicationKey` impls in one pass when new domain types are
  added) is the systemic lesson — patching one type at a time
  cost three sub-prompts when the recipe was identical.

Quality-gate-ci error count across the wave:
**11 → 9 → 7 → 4 → 0**.

First fully-green `make quality-gate-ci` since P3.3 (`5830fc9`).

Process notes:

- The 11 errors were process debt (latent failures surfacing as the
  workflow-rejection layer cleared), not regressions. The same
  warnings had been present and unreported for many commits; only
  the `ci` profile severity promotion made them visible.
- Both formerly-red CI jobs are now resolved end-to-end:
  **Smoke Analytical E2E** moved off PR CI by P4.1.6.b (now on
  schedule/manual); **Integration Tests** restored to GREEN by the
  chain P4.1.5 → P4.1.6.a* → P4.1.7 → P4.1.8.* → P4.1.10 → P4.1.11.a.
  The wave revealed three layered, pre-existing failure classes
  (counter-ordering races, rapid-publish dedup precision, and
  subject-filter wildcard mismatch) that had been masked by earlier
  workflow-rejection layers. Each layer surfaced only when the layer
  above it cleared — see the per-P4.1.x entries above for the
  per-class root causes.

Institutional knowledge captured in `docs/CONTRIBUTING.md` →
"Audit and investigation patterns" (P4.1.4).

### CONTRIBUTING.md expansion + README refresh (Phase 3.9)

**Resolved** by codifying Phase 1+2+3 institutional knowledge in
`docs/CONTRIBUTING.md` and refining `README.md` for a public-visitor
audience. Closes P3.0 audit P1 findings "CONTRIBUTING missing AI
agent protocols (depth)" and "README gaps for public visitor".

`docs/CONTRIBUTING.md` expansion (existing "Specifically for AI
agents" section renamed to **"For AI agents (institutional
knowledge)"** and substantially extended):

- Preamble framing the section as "cumulative knowledge base — what
  we've learned the hard way" complementary to `CLAUDE.md`.
- Existing 4 subsections preserved (Read these documents first;
  Apply the protocols rigorously; Commit messages: explicit about
  provenance; When in doubt).
- New subsections added:
  - **Operating philosophy** (3 priority-ordered principles).
  - **Pause-and-report protocol (5 steps)** with a table of 5 worked
    examples from P2.3, P2.Y, P3.3, P3.5, P3.7.
  - **Common patterns** (working-tree verification, cross-ref search,
    inventory-first, atomic commits per concern; each cross-linked
    to its `.claude/commands/` slash command).
  - **Validation discipline** (project-vs-tool versions; audit-
    heuristic validation; format pre-commit checks).
  - **Cross-platform quirks** (shell quoting; `sed -i` macOS vs Linux).
  - **Lessons learned (Phase 1+2+3 errata)** — 5 specific mistakes
    documented to avoid repetition.
  - **Anti-patterns to avoid** (reframe-to-fit; aggregate concerns;
    trust narrative reference; skip validation; bypass safety hooks).

`README.md` refresh (conservative — no full rewrite):

- "Current state" section now leads with "Early-stage personal
  project. Active development by a single maintainer. Not
  production-ready; no API stability guarantees." plus an explicit
  "External contributions are not accepted at this stage" note with
  SECURITY.md pointer.
- "Contributing" section reframed for maintainers and AI agents
  with explicit pointers to `CLAUDE.md`, `docs/CONTRIBUTING.md`,
  and `.claude/`.
- "License" section refined with explicit permitted/not-permitted
  bullets (personal use vs commercial use).

`CLAUDE.md` unchanged — already robust post-P1C; `CONTRIBUTING.md`
expansion complements rather than duplicates.

### `.claude/` automation surfaces populated (Phase 3.8)

**Resolved** by populating `.claude/commands/` and `.claude/agents/`
with content codifying Phase 1+2 patterns. Closes P3.0 audit P1
finding "`.claude/` commands/agents/hooks empty".

Commands added (5 slash commands in `.claude/commands/`):

- **`/check-clean`** — pre-action verification (working tree clean +
  `make verify` / `make bootstrap` PASS). Used at session start.
- **`/check-refs <path>`** — cross-reference search across source,
  config, docs, Makefile, CI before deletion or rename. Prevents the
  stale-infrastructure-post-restructure pattern that surfaced
  repeatedly in Phase 1+2.
- **`/inventory <area>`** — structured inventory production (files,
  sizes, last-modified dates, subdirs). Used as foundation for
  fact-dense work in P1A, P2.X, P3.0.
- **`/audit <area>`** — read-only investigation skeleton with
  P0/P1/P2/P3 severity buckets and explicit "stop at recommendations"
  rule. Template for P3.0-style audits.
- **`/version-check`** — version consistency across `go.work`,
  `tools/raccoon-cli/rust-toolchain.toml`, `.tool-versions`, and CI.

Agent templates added (2 in `.claude/agents/`):

- **`investigation-agent`** — read-only investigator with structured
  output and severity categorization.
- **`execution-agent`** — scoped executor with explicit 5-step
  pause-and-report protocol (codifies lessons from P2.3, P2.Y, P3.3,
  P3.5.safety where pause-and-report caught factual divergence
  between premise and reality).

Hooks (`.claude/hooks/`) **not** added in P3.8: Claude Code hooks
feature remains exploratory; populated only when concrete repeated
needs surface. Possible follow-up as P3.8.1 or Phase 4.

Updated:
- `.claude/README.md`: added "Available commands" and "Available
  agent templates" sections; updated philosophy paragraph.
- `docs/CONTRIBUTING.md`: added "Claude Code automation" section
  between "Git hooks (lefthook)" and "Authorized expansion protocol".

`CLAUDE.md` (repo root) is unchanged — already robust post-P1C; the
new automation complements rather than replaces it.

### Editor configs and tool-versions added (Phase 3.7)

**Resolved** by adding three universal config files at the repo root.
Closes P3.0 audit P1 finding "editor/IDE configs absent".

- **`.editorconfig`**: cross-editor formatting standard. Go uses tabs
  (gofmt convention) and Makefiles use tabs (POSIX requirement); most
  other file types use 2-space indent with LF line endings, UTF-8,
  trailing-whitespace trim, and final newline. Markdown intentionally
  keeps trailing whitespace (line-break syntax). Editors with native
  or plugin EditorConfig support (VS Code, GoLand, vim, emacs, etc.)
  pick it up automatically.

- **`.gitattributes`**: git-level file handling. Forces LF line
  endings for tracked text files (cross-platform consistency); marks
  common binary extensions to prevent accidental diff/merge
  corruption; flags `go.sum` and `Cargo.lock` as
  `linguist-generated=true` so GitHub's language stats exclude them;
  marks `docs/**` and `*.md` as `linguist-documentation`. Pre-adoption
  CRLF audit confirmed zero tracked text files had CRLF endings — no
  re-checkout churn expected.

- **`.tool-versions`**: version manifest for [asdf](https://asdf-vm.com)
  and [mise](https://mise.jdx.dev). Currently pins:
  - `golang 1.25.7` (sourced from `go.work`)
  - `rust 1.90.0` (sourced from `tools/raccoon-cli/rust-toolchain.toml`)
  - `golangci-lint 2.12.2` (pinned in `.github/workflows/ci.yml` via
    `golangci-lint-action@v6` with explicit `version: v2.12.2`; the
    v2.x major series is also pinned in `.golangci.yml`'s
    `version: "2"`. Keep this manifest in sync with the CI pin.)

  Tools without asdf/mise plugins (`lefthook`, `shellcheck`) install
  separately via `brew` or `go install`.

**Not included (deferred)**:
- `.vscode/` — per-user IDE choice. Can be added in P3.7.1 if a VS
  Code workspace is desired.
- `.idea/` — same rationale.

### Shellcheck safety fixes + P3.0 audit retraction (Phase 3.5.safety)

**Resolved** by re-investigating P3.0's "scripts safety" finding via
`shellcheck` and applying targeted fixes for the real issues surfaced.
Closes P3.0 audit P0 finding "scripts safety" with corrected scope.

P3.0 audit had claimed **"39/39 scripts MISSING `set -e`"**. That
finding is **retracted**: re-investigation found all 41 scripts already
have `set -euo pipefail` (the audit's heuristic `head -10 | grep`
missed the directive which appears after the header comment block,
typically lines 7–49). Real safety state is broadly safe.

Shellcheck 0.11.0 across all 41 scripts surfaced 106 issues:
- **71 (67%) false positives**: SC2015 (`A && B || C` used for logging),
  SC1091 (dynamic `source` paths shellcheck can't statically resolve).
- **28 (26%) minor cleanups**: SC2034 (unused vars), SC2329 (dead
  functions), SC2155 (declare+assign), SC2064 (trap quoting), SC2012/
  SC2010 (`ls` vs `find`), SC2153, SC2001. Cosmetic — not safety risks.
  Deferred to optional P3.5.cleanup.
- **7 (7%) real safety issues**: 5 × SC2086 (word splitting on
  unquoted variables) + 2 × SC2206 (array assignment via word
  splitting). **Fixed in this phase**:
  - `scripts/diag-check.sh:183` — `exit "$ERRORS"`
  - `scripts/live-pipeline-activate.sh:116` — `sleep "$POLL_INTERVAL"`
  - `scripts/live-pipeline-activate.sh:402` — `exit "$ERRORS"`
  - `scripts/smoke-compose-wiring.sh:492` — `exit "$ERRORS"`
  - `scripts/smoke-first-slice.sh:98` — `sleep "$POLL_INTERVAL"`
  - `scripts/smoke-multi-symbol.sh:77–78` — `read -ra` instead of
    `ARRAY=($VAR)` for `SYMBOLS` and `TIMEFRAMES`.

Total post-fix shellcheck issue count: 99 (= 106 − 7), all warnings
or notes, zero errors.

P3.6 (scripts safety — group 2) is **retired** as no-op: it was based
on the same incorrect "missing set -e" premise.

Lesson institutionalized: audit heuristics like `head -N | grep` can
miss content beyond the first N lines. For findings about widely-
adopted conventions (`set -e`, `gofmt`, etc.), validate with a
dedicated tool (`shellcheck`, `gofmt -l`, `cargo clippy`, etc.) before
planning remediation. Pause-and-report on audit divergence caught this
before any unnecessary work shipped.

### lefthook adopted for pre-commit and commit-msg validation (Phase 3.4)

**Resolved** by introducing [lefthook](https://lefthook.dev/) as the
pre-commit framework. Closes P3.0 audit P1 finding "no pre-commit
framework" plus the related "no commitlint" finding without a Node.js
dependency.

Stages configured in the new `lefthook.yml`:

- **pre-commit**: `gofmt` check on staged `.go` files, trailing
  whitespace, and YAML/JSON/TOML validity. Fast (sub-2-second typical).
- **commit-msg**: conventional commit format
  (`type(scope?): description`) via the new
  `scripts/validate-commit-msg.sh`, which accepts `feat`, `fix`,
  `chore`, `docs`, `ci`, `refactor`, `test`, `style`, `perf`, `build`,
  `revert`. Tested against the last 10 commits — all pass.
- **pre-push**: `make lint-go` and `make verify` available but
  `skip: true` by default. Opt in by removing the skip lines when
  ready for stricter local push gating.

Activation is per-developer (hooks are NOT auto-installed by the
commit): `brew install lefthook` (macOS) or `go install
github.com/evilmartians/lefthook@latest`, then `make install-hooks`.
Bypass for emergencies via `LEFTHOOK=0 git commit ...` or
`git commit --no-verify`.

`docs/CONTRIBUTING.md` gained a "Git hooks (lefthook)" section
between "PR workflow" and "Authorized expansion protocol".
`scripts/README.md` table updated with the new validator.

### GitHub settings lockdown applied (Phase 3.3)

**Resolved** by applying remote settings via `gh CLI`. Closes P3.0
audit P0 findings #3, #4, #6. Finding #2 (fork lockdown) partially
deferred — see below.

Changes via `gh api`:

- **Branch protection on `main`**: required status checks (Unit Tests,
  Repository Consistency & Quality Gate, Go Lint (golangci-lint)),
  strict (branch up to date), linear history required, no force-push,
  no deletions. PR review NOT required (solo-dev workflow).
- **Security & Analysis**: `secret_scanning`,
  `secret_scanning_push_protection`, `dependabot_security_updates`,
  `private_vulnerability_reporting` all enabled.
- **Actions**: `sha_pinning_required: true` (allowed_actions kept at
  "all"). May surface tag-pinned actions in the next CI run; P3.3.1
  will migrate the workflow to SHA pins if so.

**Finding #2 (fork lockdown) — deferred**: GitHub rejects
`allow_forking: false` on personal-owned public repositories (HTTP
422 — "Allow forks setting can only be changed on org-owned private
repositories"). The repo still publishes `allow_forking: true`, but
`pull_request_creation_policy: collaborators_only` already blocks
external PRs, which was the underlying intent. Manual fallbacks
(transfer to a GitHub org, or accept the fork-able state) documented
in `docs/operations/github-settings.md`.

Canonical reference of all remote settings now lives in
`docs/operations/github-settings.md` (remote settings have no git
history; this file is the source of truth going forward).

### `.gitignore` hardened (Phase 3.2)

**Resolved** by expanding `.gitignore` from 17 lines (minimal) to ~180
lines organized in six categories with explanatory comments. Closes
P3.0 audit P0 finding #5 (.gitignore missing critical patterns for
a public repository).

The new file groups patterns by intent:

- **A. Secrets and credentials** (P0 for public repo): `*.env`,
  `.env.local`, `.env.*.local`, `.env.production`, etc.; `*.key`,
  `*.pem`, `*.p12`, `*.pfx`, `*.crt`; `credentials`, `credentials.json`,
  `credentials.yml`; SSH keys (`id_rsa`, `id_ed25519`, ...); cloud
  configs (`.aws/`, `.gcp/`, `.azure/`); generic stores (`.secrets/`,
  `secrets/`, `*.token`, `*.secret`); `.netrc`, `.npmrc`.
- **B. Build artifacts**: `bin/`, `build/`, `dist/`, `out/`, coverage
  outputs, tmp/, archives. Preserves project-specific patterns
  `trace-pack-*` and `references/`.
- **C. Editor/OS metadata**: vim swap, backup files, `.DS_Store`,
  `Thumbs.db`. `.vscode/` and `.idea/` intentionally NOT excluded
  (per-developer choice).
- **D. Runtime**: `*.log`, core dumps, `*.test`, `*.prof`.
- **E. Tool-specific**: Rust `target/`, Node `node_modules/`, Python
  caches.
- **F. Compiled service binaries at repo root**: the original
  `/configctl`, `/derive`, `/execute`, `/gateway`, `/ingest`, `/store`,
  `/writer`, `/migrate` guards preserved verbatim — `go build ./cmd/<x>`
  drops the binary in the repo root by default.

Audit before modification confirmed zero existing tracked files match
new secret patterns, and the tracked file count (979) is preserved.
The previous `*.env` pattern was retained so `deploy/envs/local.env`
remains ignored.

### LICENSE adopted + SECURITY.md added (Phase 3.1)

**Resolved** by creating `LICENSE` and `SECURITY.md` in the repository
root. Closes P0 finding #1 from the P3.0 environment audit (LICENSE
absent) and finding #11 (no SECURITY.md).

The license is **PolyForm Noncommercial 1.0.0** — designed for solo
developers wanting to forbid commercial use while keeping source
visible. Permits personal use, research, education, hobby projects, and
evaluation. Compatible with the Go module proxy, no impact on
dependency tooling. Reference:
<https://polyformproject.org/licenses/noncommercial/1.0.0/>.

`SECURITY.md` documents how to report vulnerabilities to a personal
project: out-of-band via the maintainer email, no SLA, no bounty, scope
limited to this repository's own code.

`README.md` gained a final "License" section linking both files.
Source files were intentionally **not** annotated with per-file headers
— the `LICENSE` file alone is legally sufficient and a 400+ file diff
was not justified by the cosmetic gain. May be revisited later.

### `docs/legacy/` removed definitively (Phase 2.Y)

**Resolved** by `git rm -rf docs/legacy/` and updating active
cross-references. The 1712 files preserved under the original
"C+Y+Q — preserve legacy in-repo" decision were deleted; owner chose
no tag and no archive branch, trusting `git log` for recovery.

Cumulative consultation rate of legacy material during Phases 1A
through 2.X.1 was zero, demonstrating documental sufficiency of the
new topology. Removing the tree also takes ~17 MB off git operations,
IDE indexing, and GitHub web UI.

Cross-references corrected in the same commit:

- `scripts/bootstrap-check.sh` — `required_paths` array realigned from
  15 legacy entries to the current Phase 1A topology (root docs + the
  three subdir READMEs). The "Next Steps" tail message also updated.
  This was the **10th instance** of the stale-validation-infrastructure
  pattern observed since the reset (`.opencode/`, the original 500-line
  `repository-consistency-check.sh`, `AGENTS.md`, root `DEVELOPMENT.md`,
  root `README.md`, CI workflow blast-radius visibility,
  `raccoon-cli drift_detect.rs` const tables, `scripts/stage-tooling.sh`,
  the 4 orphan P2.X.1 smokes, and now `bootstrap-check.sh`).
- `scripts/repository-consistency-check.sh` — narrative comment.
- `tools/raccoon-cli/src/analyzers/drift_detect.rs` — 2 rustdoc comments.
- `deploy/configs/execute-mainnet-live.jsonc` — removed dangling
  `// See: docs/legacy/...` pointer (authorized scope expansion).
- `docs/RESUMPTION.md`, `docs/DEVELOPMENT.md`, `CLAUDE.md`, `AGENTS.md`,
  `README.md` — narrative refs and reading-map rows.

For any future need to inspect pre-reset material, use
`git log -- docs/legacy/<path>` or `git show <SHA>:docs/legacy/<path>`
against the parent of the P2.Y commit.

### G6 — `raccoon-cli drift-detect` against old topology (Phase 1D.4)

**Resolved** by rewriting 6 const tables in
`tools/raccoon-cli/src/analyzers/drift_detect.rs` to align with the
Phase 1A topology:

- `SIGNAL_DOCS`, `DECISION_DOCS`, `STRATEGY_DOCS`, `RISK_DOCS`,
  `EXECUTION_DOCS` collapsed from 7–30 paths each (pre-reset granular
  family architecture design docs, retired in P1A.1) to
  1 path each (`docs/domain/<x>.md`).
- `ARCH_DOCS` rewritten from 27 pre-reset arch docs to 8 canonical
  root docs (`docs/ARCHITECTURE.md`, `docs/RUNTIME.md`,
  `docs/HTTP-API.md`, `docs/DEVELOPMENT.md`, `docs/RESUMPTION.md`,
  `docs/CONTRIBUTING.md`, `docs/GLOSSARY.md`, `docs/decisions/README.md`).
- The "runtime-target.md mentions all services" sub-check rewired to
  read `docs/RUNTIME.md` (was previously silently skipping because the
  hardcoded path didn't exist).

The 27 other checks in `drift_detect.rs` (per-domain adapter alignment,
domain Go files, NATS subjects/durables/buckets, contracts,
naming-identity guard against `DEFUNCT_NAMES = ["emulator", "validator"]`,
actor-scope, stream-registry, premature-domain-entry, etc.) preserved
unchanged. They were already passing; this change only touched the
6 constants and one sub-check path.

**Effect:** `make quality-gate` PASS (6/6 active analyzers, 84 checks,
0 errors, was 61 errors). `make verify` PASS **for the first time since
P1A.1** (18+ prompts ago). CI workflow `repository-checks` job will run
green on the next push.

**Pattern note:** this was the 7th instance of the
"stale-infrastructure-post-restructure" pattern observed across Phases
1A–1D (`.opencode/`, `scripts/repository-consistency-check.sh`,
`AGENTS.md`, root `DEVELOPMENT.md`, root `README.md`, the CI workflow's
silent G6 propagation, and finally `drift_detect.rs` itself). The
discipline now lives in `docs/CONTRIBUTING.md` "Rules for documentation
changes" and the `make` verification surface, with the analyzer itself
enforcing the new topology going forward.

## Earlier resolutions

### G3 — `make verify` cross-references (originally framed as 9 failures from `.opencode/`)

The original framing of G3 ("9 failures, all from `.opencode/`
cross-refs") was inaccurate. P1B uncovered the truth in three layers:

1. **`.opencode/` directory** existed and had 1 cross-reference check
   failing. **Resolved** by deletion in P1B.

2. **`scripts/repository-consistency-check.sh`** had ~7 checks failing
   because the script was hardcoded against the pre-reset docs topology
   (`docs/product/`, `docs/architecture/`, `docs/development/`,
   `docs/stages/`, `docs/archive/`, `docs/tooling/`) which was
   restructured in P1A.1. The script was never updated during Phase 1A
   because the failure was misattributed to `.opencode/`.
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
| **Phase 1C** | Build `.claude/` agentic layer | **CLOSED** (CLAUDE.md + .claude/ structure built) |
| **Phase 1D** | PR-based governance + G6 resolution | **CLOSED** (root files consolidated, .github/ templates, drift_detect.rs realigned) |
| **Phase 2** | Environment hardening (CI, Docker, scripts, Makefile cleanup) | **CLOSED** (11 sub-prompts; golangci-lint baseline, Dependabot, CI hardening, Docker contexts, Rust toolchain pinning) |
| **Phase 3** | Public-repo hygiene (license, security, hooks, editor configs, AI agent automation) | **CLOSED** (2026-05-22; 10 sub-prompts executed, 2 deferred. See "Phase 3 — closed summary" below.) |
| Phase 4 | CI restoration + P0 follow-through deferred from Phase 3 | **CLOSED** (2026-05-23; P0 backlog 5/5 closed across P4.0–P4.5.c.ii; 12 ADRs, 20 M-candidates queued, 0 open Dependabot PRs, 0 open security advisories) |
| Phase 5 | Environment work — `.claude/`, prompt templates, operational tooling, process-debt mitigation (distinct from feature work) | **IN PROGRESS** (P5.0 audit complete 2026-05-23; P5.1+ pending owner direction) |
| Phase 6+ | Subsequent waves (feature work; first capabilities likely include backtesting) | Future |

Phase 1A subdivision (status at time of this doc):

| Sub-phase | Goal | Status |
|---|---|---|
| P1A.1 | Restructure docs/ topology + new scaffolding (legacy tree later retired in P2.Y) | Done |
| P1A.2 | docs/README, docs/GLOSSARY | Done |
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

### Phase 3 — closed summary

Goal: engineering excellence for solo dev + Claude Code in a public
repo.

| Sub-phase | Status | Outcome |
|---|---|---|
| P3.0 | ✓ | Environment audit (1345 lines, 13 sections) |
| P3.1 | ✓ | LICENSE (PolyForm NC 1.0.0) + SECURITY.md |
| P3.2 | ✓ | .gitignore hardened (17 → 184 lines, 6 categories) |
| P3.3 | ✓ partial | Branch protection + security toggles; fork lockdown blocked by GitHub personal-tier policy (mitigated by `collaborators_only` PR policy + LICENSE) |
| P3.4 | ✓ | lefthook + custom shell `commit-msg` validator |
| P3.5.safety | ✓ | 7 shellcheck SC2086/SC2206 fixes; P3.0 "missing set -e" finding retracted (all 41 scripts already had `set -euo pipefail`) |
| P3.7 | ✓ | `.editorconfig`, `.gitattributes`, `.tool-versions` |
| P3.8 | ✓ | `.claude/` commands (5) + agent templates (2) |
| P3.9 | ✓ | `docs/CONTRIBUTING.md` expansion ("For AI agents — institutional knowledge"); README refresh |
| P3.10 | ✓ | Closing audit + this RESUMPTION refresh |
| P3.5.cleanup | ⏿ | Deferred (cosmetic — 28 minor shellcheck issues + 32 SC1091) |
| P3.6 | ⏿ | Retired (audit premise was wrong) |

Key lessons institutionalized (in `docs/CONTRIBUTING.md` under
"For AI agents — institutional knowledge"):

- 5-step pause-and-report protocol with 5 worked examples.
- Project-declared vs tool-environment version distinction.
- Audit-heuristic validation (heuristics like `head -N | grep` miss
  content beyond the inspection window — validate with dedicated
  tools).
- Cross-platform shell quirks (quoting, `sed -i` macOS vs Linux).
- Atomic commits per concern.

Surprises caught during Phase 3 via pause-and-report:

- **P3.3**: GitHub personal-tier doesn't allow fork disable; mitigated
  by collaborator-only PR policy + LICENSE.
- **P3.5**: P3.0 finding "39/39 scripts missing `set -e`" was wrong
  — audit grepped only `head -10`; all 41 scripts already had
  `set -euo pipefail` declared after the header comment block.
  Retracted in P3.5.safety; replaced by shellcheck-based audit that
  surfaced 7 real safety issues (SC2086/SC2206), all fixed.
- **P3.7**: original claim that golangci-lint was "not pinned in CI"
  was wrong — `.github/workflows/ci.yml:179-182` explicitly pins
  `version: v2.12.2` on `golangci-lint-action@v6`, matching
  `.tool-versions`. Drift is zero. Corrected in P4.0 (see DOC-3
  erratum below); ongoing task is **monitoring drift** when
  Dependabot bumps the action wrapper (e.g., `@v6 → @v9`) without
  necessarily bumping the underlying lint binary.
- **P3.5.safety scope omission** (caught by P4.0 pre-audit, DOC-4):
  shellcheck audit covered `scripts/*.sh` but not `scripts/utils/*.sh`.
  One real SC2206 in `scripts/utils/lib.sh` was missed at the time
  and fixed in P4.0 alongside the documental sweep. Methodology drift,
  not a new pattern — the same rule should have been applied to all
  `.sh` files, not the top-level only.

---

## Phase 4 outlook

Phase 4 essential delivery complete (2026-05-23). The 4.1 wave (CI
restoration + quality gate cleanup) closed on 2026-05-22 with
quality-gate-ci green (commit `25839ea`); P4.2–P4.5 closed all five
P0 items deferred from Phase 3, with read-only investigation
interleaved before each fix.

**Phase 4 P0 backlog FULLY CLOSED** (5/5):

| P0 item | Phase 4 prompt | Closure commit / date |
|---|---|---|
| P0-1 (CI restoration) | P4.0 + P4.1 wave (24+ sub-prompts) | b7eaa53, 2026-05-22 |
| P0-2 (rate_limiter + Close) | P4.2 / P4.2.a | a6f0175, 2026-05-23 |
| P0-3 (context bounding) | P4.3 / P4.3.a | 455f02e, 2026-05-23 |
| P0-4 (Dependabot triage) | P4.5 / P4.5.a-c.ii | this commit, 2026-05-23 |
| P0-5 (ControlGate fail-open) | P4.4 / P4.4.a | 7c2f09e, 2026-05-23 |

**Cumulative artifacts shipped during Phase 4**:

- 12 ADRs (added ADR-0012 for ControlGate fail-open posture).
- 20 design-meta candidates (M1–M20, with M10 reserved gap;
  M19 closed during P4.5.c verification).
- ~9 errata observations across CONTRIBUTING/investigation patterns.
- 7/7 CI consistently green on main.
- 0 open Dependabot PRs.
- 0 open security advisories.

**Remaining Phase 4 work** (all discretionary):

- **P4.X** — Tier E quality enhancements, opt-in (e.g., the ~60
  hardcoded timeout literals deferred from P4.3.a — operational
  tunability gap, not a bug).
- **Phase 4 design-meta discussion** — full conversation across M1–M20
  when momentum permits. Not blocking.

The existing "Outstanding work" section below records each item's
closure narrative for handoff context; preserved verbatim as
historical record.

### Outstanding work (post P4.1)

1. ✓ **Integration Tests + Smoke Analytical E2E** (P4.1.5 → P4.1.6
   scope). Closed across the Phase 4.1 wave. Smoke Analytical E2E
   deferred to a scheduled/manual workflow (P4.1.6.b, commit
   `e91b863`); Integration Tests stabilized via the NATS docker-run
   switch (P4.1.6.a..a.ii) and counter-race helpers (P4.1.8.a..d).
   The documented `TestControlledActivation_FullLifecycle` /
   `TestRealVenueActivation_FullLifecycle` 200 ms timing flake
   remains visible on some intermediate Dependabot merges (per the
   P4.5.a/b/c.ii closure narrative); non-required and non-blocking
   per branch protection. CI 7/7 GREEN at `main` HEAD.
2. ✓ **`rate_limiter` test + `Close` lifecycle** (P0-2 / P4.2).
   Closed 2026-05-23. 10 unit tests added (`rate_limiter_test.go`);
   `Close()` lifecycle wired at the 2 cmd/execute mainnet sites via
   a `closers []func()` field on `venueAdapterResult`. P4.2.a fixed
   a downstream goroutine-assertion flake. CI 7/7 green.
3. ✓ **`context.Background()` propagation in actors** (P0-3 / P4.3).
   Closed 2026-05-23. Reframed: Hollywood deliberately drops context
   at the mailbox boundary, so the right shape was "bound fresh
   Background with WithTimeout + config", not "propagate caller ctx".
   P4.3.a bounded 14 unbounded sites + enabled the `contextcheck`
   linter for prevention. Surfaced M13/M14/M15 (see design-meta).
4. ✓ **Kill switch fail-open decision** (P0-5 / P4.4 + P4.4.a).
   Closed 2026-05-23. Investigation reframed P0-5 as documentation +
   observability gap, not semantic gap — the audit's "kill switch" is
   the codebase's ControlGate, with fail-open intentionally chosen
   and protected by 8-layer defense-in-depth. P4.4.a formalized the
   posture as ADR-0012 and added `gate_read_failures_total`
   counter with 5 reason labels so the silent failure mode is
   monitorable. No semantic change. Future hybrid strategies
   deferred as M16/M17/M18 pending counter data.
5. ◐ **Dependabot security PRs** (P0-4 / P4.5). Triage closed 2026-05-23;
   security wave closed same day:
   - P4.5 investigation: 17 open PRs identified, all 1 day old. Six
     open security advisories cluster cleanly to 3 PRs (#16/#17/#18).
     All 17 PRs share one root cause — bases predate the P4.1
     SHA-pinning migration. Triage shape is 3 archetype waves, not
     17 individual reviews.
   - P4.5.a (closed 2026-05-23): closed obsolete PR #5
     (golangci-lint-action — already applied via P4.1.1); rebased +
     merged security PRs #16 (otel /clickhouse), #18 (otel /migrate),
     #17 (rustls-webpki /raccoon-cli). All 6 security advisories
     closed. Required CI checks (Unit Tests, Quality Gate, Go Lint)
     green for all three; Integration Tests flake (the documented
     `TestControlledActivation_FullLifecycle` / `TestRealVenueActivation_FullLifecycle`
     timing flake) ignored as non-required, non-regression.
   - P4.5.b (closed 2026-05-23): minor/patch batch — 8 PRs (#7, #9,
     #11, #10, #13, #15, #12, #14) rebased + merged sequentially.
     Order grouped 3 cargo singletons → 3 standalone gomod →
     in-module pair (#12/#14 share `internal/adapters/nats/go.mod`).
     All 8 cleared required CI (Unit Tests + Quality Gate + Go Lint);
     Integration Tests flake non-blocking per P4.5.a posture. No
     genuine test failures; no mirror-pair conflicts (Dependabot
     rebase-on-trigger handles each PR against current main).
     `go.work.sum` picked up transitive checksums for the
     `golang.org/x/{net,sync,term,text,tools,mod}` and otel/metric
     families pulled in by the nats.go/clickhouse-go bumps.
   - P4.5.c (closed 2026-05-23): 5 majors — 4 GitHub Actions (#6,
     #2, #3, #4) + ureq 2→3 (#8). Two phases:
       * **Phase 1** (verification + investigation, ~10 min):
         rebased PR #6 to test M19 hypothesis. Result: post-rebase
         diff was SHA-style with version comment
         (`actions/checkout@de0fac2e... # v6.0.2`); 8 sites in ci.yml +
         1 in smoke-analytical.yml all updated. M19 verified
         **self-correcting**; closed. ureq surface inventory: 1
         file (`tools/raccoon-cli/src/smoke/api.rs`), 6 call sites,
         3 patterns, ~25 LOC. Recommendation: migrate.
       * **Phase 2** (execution): merged #6 as validation; sequential
         rebase + merge for #2 (actions/cache 4→5), #3 (actions/setup-go
         5→6), #4 (actions/upload-artifact 4→7) — all four landed
         SHA-pinned with version comments. ureq 2→3 migrated in api.rs
         (header/StatusCode/Agent.config_builder().timeout_global/body_mut
         .read_json) preserving ApiClient public interface. PR #8 closed
         in favor of combined Cargo.toml + source commit.
   - **Phase 4.5 wave fully closed.** Final state: 0 open Dependabot
     PRs, 0 open security advisories.
   - **Phase 4 P0 backlog FULLY CLOSED** (5/5 items: P4.2 rate_limiter,
     P4.3 context bounding, P4.4 ControlGate ADR-0012, P4.5 Dependabot
     wave, and P4.0/P4.1 wave covering CI infrastructure restoration).

### Phase 4 design-meta candidates (deferred)

Twenty architectural questions surfaced across the Phase 4 wave
(M1–M20; M19 closed during P4.5.c verification). Captured here so
context isn't lost; not blocking. Each deserves a dedicated
discussion session — Phase 4 P0 work has now closed, so the
strategic view is informed.

The queue is the artifact; resolution is future work.

#### M1 — Auto-derive `CANONICAL_STREAMS` from Go AST

`tools/raccoon-cli/src/analyzers/drift_detect.rs` mirrors the stream
catalogue declared in `internal/adapters/nats/natsexecution/registry.go`.
Drift has hit twice (G6, G6.2) when new streams shipped without the
mirror being updated. A codegen step deriving `CANONICAL_STREAMS`
from the Go AST would eliminate the G-class drift surface
permanently.

#### M2 — `EventSpec.Subject` "prefix as published subject" convention

The `contract-audit` `event-stream-coverage` check treats
`EventSpec.Subject` as the literal published subject. Several
publishers (e.g., `PublishExecution`) append context tokens to the
spec prefix at publish time, so `Subject` is in practice a prefix.
3 of 4 execution publishers happen to align with their stream
wildcards by coincidence of prefix lengths; the SessionLifecycle
event surfaced because it did not. Extend the scanner to understand
prefix-then-context, removing the latent risk in EventSpecs that
pass only by happenstance.

#### M3 — Document raccoon-cli profile semantics

The `fast`, `ci`, and `deep` profiles run the same check set; `ci`
promotes warnings to errors and prefixes them with `[ci]`. The
mapping is hardcoded in `tools/raccoon-cli/src/gate/mod.rs` with no
external config and no user-facing documentation. Surface this in
`tools/raccoon-cli/README.md` or `docs/operations/` so the
promotion rule is discoverable rather than discovered.

#### M4 — `walk_go_files` doc-vs-reality cleanup

The doc comment on `walk_go_files` in `arch_guard.rs` claims
"non-test, non-vendor", but the function filters only `vendor/`.
The test-file filter lives inside `check_deploy_boundary`'s closure
(P4.1.3.b). Not a bug today (only deploy-boundary calls
`walk_go_files`), but a trap for future callers. Either align the
doc with the behaviour or move the filter into `walk_go_files`
and remove it from the closure.

#### M5 — Application clients exposing domain types in public contracts

`executionclient` and `monitoringclient` return and accept domain
types directly in their public APIs (e.g.,
`SessionListReply.Sessions []execution.Session`). This is why `cmd/`
must import `internal/domain/*` for composition wiring — the
clients don't hide domain behind DTOs. ADR-0005's "cmd sees
everything" makes the current state defensible; the question is
whether an anti-corruption boundary between application and its
consumers would be net positive (more isolation, more boilerplate,
more test surface). May spawn a sub-ADR.

#### M6 — ADR-0005 clarification: composition vs invocation

ADR-0005 says "cmd sees everything". P4.1.3.c.ii clarified what
that means in practice: cmd may reference domain types for
composition, but should not invoke domain functions directly
(those are routed through application clients). Add a companion
note to ADR-0005, or amend it in place, articulating the
composition-vs-invocation distinction so the refined raccoon-cli
rule and the ADR speak the same language.

#### M7 — Dual-semantic counter for pre-publish vs post-publish observability

`Counter("filled")` (and analogous counters in execute-scope actors)
is incremented AFTER the NATS publish that signals the same event.
This creates a sub-microsecond observability window where subscribers
see the published event before the counter reflects it.

Current consumers tolerate this: HTTP `/statusz` timing dominates the
race; intra-actor `logStats()` reads are race-free by Hollywood's
single-threaded mailbox; Prometheus `/metrics` uses a separate counter
set. P4.1.8 added an `eventuallyAtLeast` helper for test consumers.

**Candidate work**: introduce dual-semantic counters — e.g.,
`submit_attempted` (incremented before publish) and `submit_succeeded`
(incremented after publish ack). Tests synchronize on
`submit_attempted` for pre-publish observability; production `/statusz`
keeps `submit_succeeded` for honest post-publish accounting.

**When to revisit**: if new production observability surfaces require
sub-millisecond timing (real-time dashboards, alerting on counter
rates), or if cross-actor counter reads emerge.

Decision context: P4.1.8.c investigation; Option (C) accepted in
P4.1.8.d (keep eventually-poll helper, skip actor reorder).

#### M8 — Synthetic data path for analytical surface

The analytical pipeline (`writer` → ClickHouse → `gateway` queries)
depends on live Binance Futures WSS data via `ingest`. CI runners
typically cannot reach Binance (network egress / geo-blocking on
GitHub Actions). Smoke Composed Pipeline works around this with
Go-level synthetic data, but Smoke Analytical needs the full stack
plus the live feed.

**Candidate work**: introduce a synthetic ingester (or synthetic
data injection point upstream of `writer`) that emits the same
downstream events `ingest` would produce. Would unblock smoke-
analytical for PR CI.

**Status**: smoke-analytical deferred to scheduled/manual workflow
(P4.1.6.b) until the synthetic seeder exists.

#### M9 — CI log-error scan robustness

The "Scan for error-level logs" step in the smoke-analytical
workflow greps for `"level":"error"` only. Warn-level logs (e.g.,
`ingest` unable to reach an external service surfacing as `warn`,
not `error`) escape detection. The step PASSED even when the
end-to-end pipeline produced no data, contributing to false
confidence in the CI signal.

**Candidate work**: extend the scan to flag warn-level too, OR add
a health-endpoint assertion (writer `active_trackers`, gateway
readiness counters), OR fail-fast when upstream produces no events
within a fixed window.

**When to revisit**: pre-requisite for restoring smoke-analytical
to PR CI alongside M8 (synthetic seeder). Without M9, the restored
job would silently pass on broken pipelines again.

#### M11 — Subject-filter validation in NATS consumer fallback

`internal/adapters/nats/natsexecution/consumer.go:79` falls back to
`c.spec.Event.Subject` (bare base subject) when `FilterSubjects` is
not supplied on the `ConsumerSpec`. If the bare base subject has no
`.>` (or `.*` etc.) wildcard suffix and the publisher writes at
qualified sub-subjects, NATS JetStream silently delivers zero
messages to the consumer — the producer side is the same channel,
but the subscription pattern never matches.

P4.1.11 found 9 test sites across `writerpipeline` and
`natsexecution` integration tests that hit exactly this fallback
with the bare `registry.PaperOrderSubmitted` EventSpec
(`execution.events.paper_order.submitted`) while publishers wrote to
`execution.events.paper_order.submitted.{source}.{symbol}.{timeframe}`.
The tests were latent through the entire project history until
P4.1.10 unmasked them; production paths were safe because they go
through helper specs that supply the wildcard form.

This is the consumer-side counterpart to **M2**
(`EventSpec.Subject` "prefix as published subject" convention).
M2 closes the publisher-side scanner gap; M11 closes the consumer-
side runtime gap. Same underlying convention; different enforcement
surface.

**Candidate work**: at `consumer.go:79`, validate the subject
pattern before binding the consumer. If the spec's `Event.Subject`
contains no wildcard segment AND the publisher's known
`Event.Subject` convention is "prefix-then-context" (M2-aware), emit
a startup-time warning (or a panic in `_test.go`-compiled builds)
so future tests bypassing production helpers cannot silently miss
events. The check could also live in `natskit.NewConsumerSpec`
factory or in a separate static-analysis check; design choice is
open.

**Why deferred**: P4.1.11.a fix (single helper, 9 call sites)
prevents the specific manifestation. Defensive runtime validation
is broader architectural work that needs the M2 scanner design
finalised first (so both sides apply the same prefix-vs-context
heuristic).

#### M12 — Sweep all timestamp-keyed `DeduplicationKey` impls atomically

P4.1.10 fixed `Strategy.DeduplicationKey()` (Unix → UnixNano) when
the family tests it targeted surfaced the silent JetStream Duplicate-
Window drop. P4.1.11.a then had to extend the same recipe to
`ExecutionIntent`, `Decision`, `RiskAssessment`, and `Signal` once
the writerpipeline + natsexecution restart_recovery tests exercised
those types. Each was the *identical* one-line fix. The pattern is
clear: any `DeduplicationKey` method that interpolates
`Timestamp.Unix()` is latent — production cadence (kline ≥1s) hides
it for current callers, but any new tight-loop producer (test or
future code) will re-discover the bug.

Two `DeduplicationKey` impls are exempt because they don't use a
timestamp suffix: `SessionLifecycleEvent.DeduplicationKey()`
(session-id + status) and `ObservationTrade.DeduplicationKey()`
(source + tradeID).

**Candidate work**: add a quality-gate / raccoon-cli check that
flags any `DeduplicationKey` implementation containing
`Timestamp.Unix()` (without `Nano`). Alternatively, a domain test
that asserts no `DeduplicationKey` collides for two distinct
sub-second siblings. Either prevents the recipe from being
re-discovered piecemeal in future waves.

**Why deferred**: the four sibling impls were fixed in P4.1.11.a;
the check would be a guard, not a hotfix. Bundle with M2/M11 when
the broader "publish-side / consume-side contract validation" work
is scoped.

#### M13 — NATS header-extracted deadlines (responder layer)

P4.3.a fixed the unbounded `context.Background()` in
`natskit/request_reply_responder.go` by adding a configurable
`requestTimeout` field (default 5s). The alternative considered but
deferred was extracting the deadline from NATS request headers
(e.g., a `Nats-Expected-Deadline` header), allowing callers to
specify per-request bounds. More honest deadline propagation from
HTTP through gateway through actor handler.

**Candidate work**: define a header convention; update
`RequestReplyResponder` to honor the header if present, falling
back to its configured default otherwise; update gateway emitters
to populate the header from the HTTP request's deadline.

**Why deferred**: the configurable default is sufficient for current
operations; per-request deadline propagation matters more for
externally-driven load patterns we don't have yet. Single timeout
field in `ControlResponderConfig` adequate today.

#### M14 — Per-use-case ControlRouter timeouts

`ControlRouterActor` uses a single `RequestTimeout` for every
use-case dispatch (P4.3.a `handlerContext` helper). Some use cases
are heavier than others — `compileConfig` involves JSON Schema
validation; `getConfig` is a single KV read. A single timeout
forces compromise: large enough for the slow case, looser than
needed for the fast case.

**Candidate work**: extend `ControlRouterConfig` with optional
per-use-case overrides; `handlerContext` accepts a use-case
identifier and applies the appropriate timeout per operation.

**Why deferred**: single timeout adequate for current operations
(none yet exhibit measurable timeout-driven friction). Pull into
scope only when a specific use case routinely hits the global cap.

#### M15 — Custom raccoon-cli context analyzer

P4.3.a enabled the standard `contextcheck` golangci-lint linter
to flag bare `context.Background()` patterns. `contextcheck` catches
generic Go patterns but doesn't understand the Hollywood mailbox
boundary: it can't distinguish a legitimate fresh-context creation
(actor handler that has no caller context) from an accidental one
(handler that has a context but ignores it). The 3 `//nolint:contextcheck`
suppressions added in P4.3.a are project-specific rules that
contextcheck cannot express.

**Candidate work**: extend `tools/raccoon-cli`'s `arch-guard`
analyzer with project-specific context flow rules — e.g., "inside
a `Receive(c *actor.Context)` method, fresh `context.Background()`
is allowed; inside a function that takes `ctx context.Context`,
deriving from `Background()` requires a justification annotation".

**Why deferred**: `contextcheck` + `//nolint` comments are
sufficient today (3 known suppressions, all with rationale). The
custom analyzer earns its keep only if more Hollywood-boundary
patterns surface that contextcheck cannot classify correctly.

#### M16 — ControlGate cached state with staleness threshold (H1)

P4.4 design discussion option H1, deferred at P4.4.a in favour of
documenting the current fail-open posture (ADR-0012) and adding
observability (`gate_read_failures_total`). The cached-state variant
would memoize the last successful gate read in process memory and
serve it during transient KV failures, falling back to fail-closed
only after a configured staleness threshold (e.g., 30s). Combines
availability of pure fail-open with the safety of fail-closed
during sustained outages.

**Why deferred**: requires operational data from the new counter
before the threshold can be chosen non-arbitrarily. A non-zero
`kv_error` or `ctx_timeout` rate at scale would make this concrete;
a flat-zero rate confirms M16 is unnecessary.

#### M17 — ControlGate conditional fail-closed (H2)

P4.4 H2: bifurcate the IsHalted contract by adapter mode.
`AdapterVenue` + `CredentialPresent` callers would fail-closed
(safety prioritized on the live path); paper / dry-run callers
would keep the current fail-open. Matches the risk surface (only
the live + creds path can cause real harm) to the safety posture.

**Why deferred**: adds a second code path with mode-conditional
semantics; subtle bugs possible around mode transitions. Need
M16's operational data first to judge whether the simpler
single-path posture has a real cost.

#### M18 — ControlGate `ErrKeyNotFound` distinction (H3)

P4.4 H3: split today's "any read failure = fail-open" into
"first-boot (no operator write yet) = fail-open by design" vs
"real read failure = different posture". With the counter in place,
operators can already see `key_not_found` separately from
`kv_error` / `ctx_timeout` / `unmarshal_error`. M18 would change
behaviour on the latter three categories independently of the
first.

**Why deferred**: composes with M16/M17. Choosing M18 alone (e.g.,
strict fail-closed only on `kv_error` + `ctx_timeout` + `unmarshal_error`,
keeping `nil_bucket` and `key_not_found` fail-open) would be the
smallest semantic step away from current posture; worth considering
once counter data exists.

#### M19 — Dependabot SHA-pinning behavior verification — CLOSED

P4.5 investigation flagged "GitHub Actions in Dependabot" as a
potential structural friction: PRs reference v-tags (`@v5`) which the
SHA-pinning policy rejects. P4.5.a's deeper inspection hypothesized
the friction is largely self-correcting on rebase — Dependabot
preserves the existing workflow file's pin style, so once the PR
is rebased onto a base where actions are SHA-pinned (post-P4.1),
the regenerated diff is SHA-style and passes CI.

**Verified in P4.5.c Phase 1 (2026-05-23) via PR #6 rebase test**:
- Pre-rebase diff: `-uses: actions/checkout@v4` / `+uses: actions/checkout@v6`
  (tag-style, generated against pre-pinning base).
- Post-rebase diff: `-uses: actions/checkout@34e1148... # v4.3.1` /
  `+uses: actions/checkout@de0fac2e... # v6.0.2` (SHA-style with version
  comment, generated against current SHA-pinned base).
- 8 sites in `ci.yml` + 1 in `smoke-analytical.yml` updated automatically.
- All 4 Actions PRs (#6/#2/#3/#4) merged in P4.5.c with the same
  rebase pattern; each preserved SHA-pinning.

**Outcome**: no config change required. Future Dependabot Actions PRs
will be auto-mergeable after a single `@dependabot rebase` comment.
M19 closes.

#### M20 — Dependabot dedup for manually-applied upgrades

P4.5 surpresa #2: when a dependency is upgraded manually (e.g.,
P4.1.1 bumped `golangci-lint-action` 6→9 via direct workflow edit),
Dependabot does not auto-close the corresponding open PR. Manual
close required (done in P4.5.a for PR #5). With weekly Dependabot
cadence + 17 PRs from a single sync, a similar drift on multiple
PRs is plausible.

**Candidate work**: investigate whether Dependabot has a config
option to detect "target version already at or beyond what is on
the default branch" and auto-close. Alternatively, a small post-merge
GitHub Action that closes any open Dependabot PR whose target
version is now ≤ main's current pin.

**Why deferred**: low frequency to date (1 known instance, #5).
Worth revisiting if the pattern recurs after the routine batch
(P4.5.b) or after future manual upgrades.

### Available work (P1/P2, opt-in)

- Code-side audit of `internal/` and `cmd/` (Phase 3 was
  environment-only).
- Test coverage analysis (current ratio ≈ 0.71; ~288 test files vs
  ~402 prod files).
- Security deep dive post the P3.3 toggles (real residual exposure?).
- Performance audit (compose stack startup, smoke duration trends).
- **P3.5.cleanup**: 28 minor shellcheck issues + 32 SC1091 (source-
  path directives). Cosmetic.
- **P3.7.1**: `.vscode/` configs if owner uses VS Code.
- **P3.8.1**: `.claude/hooks/` if a concrete pattern surfaces.

### Architectural decisions registry (Phase 3)

For session orientation; full ADRs are in `docs/decisions/`.

| Decision | Source |
|---|---|
| License: PolyForm Noncommercial 1.0.0 | P3.1; LICENSE + SECURITY.md |
| Pre-commit framework: lefthook (Go-based, no Node) | P3.4; `lefthook.yml` |
| Commit message convention: custom shell validator (no commitlint) | P3.4; `scripts/validate-commit-msg.sh` |
| Editor formatting standard: EditorConfig | P3.7; `.editorconfig` |
| Line-ending normalization: LF everywhere via `.gitattributes` | P3.7 |
| Tool-version manifest: `.tool-versions` (asdf/mise compatible) | P3.7 |
| Branch protection: 3 required status checks, linear history, no force-push | P3.3; `docs/operations/github-settings.md` |
| Security toggles: secret scanning + push protection + dependabot + private-vuln-reporting | P3.3 |
| Actions: SHA pinning required (workflow migration pending) | P3.3 |
| Issue/PR templates: kept (4 templates from pre-Phase 3) | Pre-P3 |
| AI agent automation: `.claude/commands/` (5) + `.claude/agents/` (2) | P3.8 |
| Institutional knowledge: `docs/CONTRIBUTING.md` "For AI agents" section | P3.9 |

`make verify` is green locally. Any new red on `make verify` is a
real regression — not historical debt.

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
