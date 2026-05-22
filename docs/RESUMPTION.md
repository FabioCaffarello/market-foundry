# Resumption

> This document is the first thing to read when returning to
> market-foundry after a pause. It captures the current state, known
> gaps, and the next concrete step.
>
> It is **honest, not aspirational.** If a capability is missing or
> partial, it says so. If a feature is broken, it says where.

Last meaningful state change: **Phase 1D closed — G6 resolved (raccoon-cli drift_detect.rs aligned with Phase 1A topology); `make verify` PASSES for the first time since P1A.1**.

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
  - `golangci-lint 2.12.2` (locally validated; v2.x major series
    pinned in `.golangci.yml`'s `version: "2"`; CI uses the
    golangci-lint-action default since no explicit pin)

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
| Phase 2 | Environment hardening (CI, Docker, scripts, Makefile cleanup) | Pending — next |
| Phase 3 | First feature wave (most likely: backtesting — see N1) | Future |
| Phase 4+ | Subsequent waves (PnL aggregation, multi-exchange, etc.) | Future |

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

---

## Concrete next step

Phase 1 is closed. The immediate next step is **Phase 2 —
environment hardening**. Candidate scope (to be refined in its own
prompt):

- CI workflow review (currently 7 jobs in `.github/workflows/ci.yml`,
  no Dependabot, no separate lint job, no caching beyond raccoon-cli).
- Docker image hygiene (multi-stage builds, base image policy, layer
  cache).
- Scripts audit (`scripts/*.sh` — what's used by which Make targets;
  prune the unused).
- Makefile cleanup (D4 — stage-tagged smoke targets that are dead
  weight: `smoke-compose-wiring`, `smoke-failure-isolation`,
  `smoke-live-listening`, etc.).
- D2/D3 surface debt (`execute` config sprawl with `s449` namespace
  residue; configctl singular vs plural subject ambiguity).

`make verify` is green. Any new red on `make verify` after Phase 1D.4
is a real regression — not historical debt.

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
