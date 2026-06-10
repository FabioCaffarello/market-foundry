# Contributing

How to make changes to market-foundry, codified from lessons learned
during the project's evolution.

This document is for **two audiences**:

1. **Humans contributing code, docs, or operational changes.** Whether
   you're the project owner returning after a pause or a new
   contributor, the rules here apply.
2. **AI agents** (Claude Code, other agents) executing change tasks
   on this repository. The protocols here are how the system stays
   coherent across many turns.

For higher-level architectural orientation, start with
[`README.md`](README.md), [`ARCHITECTURE.md`](ARCHITECTURE.md), and
[`RESUMPTION.md`](RESUMPTION.md). For daily workflow basics, see
[`DEVELOPMENT.md`](DEVELOPMENT.md).

---

## Core principles

The principles below are **enforced**, not aspirational. Each was
established because *not* enforcing it caused observable harm.

### 1. Code is the source of truth

When documentation and code disagree, **code wins**. Documentation is
explanation; code is reality.

Corollary for documentation work: **verify technical assertions against
code before saving**. Multiple times during Phase 1A documentation
reset, draft content claimed facts (number of streams, consumer
ownership, plane taxonomy) that disagreed with the codebase. Catching
these before save kept the documentation honest.

### 2. Single-writer invariant

Every JetStream stream, every NATS KV bucket, every NATS query subject
has **exactly one writer**. No exceptions.

This is the most important invariant in the system. Race conditions
that "shouldn't be possible" are almost always violations of this
invariant; preventing it by construction is much cheaper than
debugging it post-hoc.

See [`decisions/0008-single-writer-invariant.md`](decisions/0008-single-writer-invariant.md)
for the full rationale.

### 3. Static enforcement over convention

If a rule can be checked automatically by `raccoon-cli`, it must be.
Convention alone drifts; tools don't.

See [`decisions/0004-raccoon-cli-static-enforcement.md`](decisions/0004-raccoon-cli-static-enforcement.md).

### 4. Explicit duplication over premature abstraction

Three similar lines that are clear beat one helper that is fragile.
No `utils/` package. No interface with a single implementation. Structs
over interfaces unless polymorphism is actually needed.

### 5. Layer sovereignty is non-negotiable

Imports flow inward only: `domain → application → adapters → actors →
interfaces → cmd`. No exceptions. raccoon-cli enforces this.

See [`decisions/0005-layer-sovereignty.md`](decisions/0005-layer-sovereignty.md).

---

## Rules for code changes

### When adding an HTTP route

**You must update `cmd/gateway/boot_test.go`'s `routes` slice.**

This rule exists because three route trie conflicts were discovered
simultaneously in production during Phase 0, causing gateway
CrashLoopBackoff. The boot test was added as a regression guard:
it exercises all routes against a fresh httprouter and fails if any
conflict reappears.

If your PR adds a route to `internal/interfaces/http/routes/` without
adding to the test slice, CI fails. This is intentional.

See [`decisions/0010-httprouter-trie-constraints.md`](decisions/0010-httprouter-trie-constraints.md)
for the constraint details and the trade-offs.

### When adding a new domain or family

A new domain in `internal/domain/` should:

1. Follow the canonical patterns (FamilyProcessor in derive, Pipeline
   in store, FamilyDeps in gateway).
2. Have a `Validate() *problem.Problem` on its principal type
   (canonical signature).
3. Have a dedicated `internal/adapters/nats/nats<domain>/` adapter
   if it publishes to a stream.
4. Get its own ADR if it introduces a structural pattern not already
   established.
5. Get its own doc under `docs/domain/` (use existing docs as templates).

### When the canonical Validate signature doesn't fit

If your domain genuinely needs to return something other than
`*problem.Problem` (e.g., `[]ValidationDiagnostic` for multi-error
reporting, like `ConfigDocument.Validate()`):

1. **Justify with a code comment** explaining why the canonical
   signature is insufficient.
2. **Consider an ADR** if the deviation reflects a pattern others
   may want to follow.
3. **Don't deviate silently** — that produces drift, not design.

### When introducing a new NATS stream

1. Publish only from **one binary**. Document in the registry.
2. Document the stream in [`RUNTIME.md`](RUNTIME.md) → streams table.
3. If the topology decision is non-obvious (e.g., a new family of
   events that didn't fit existing flows), write an ADR.

### When introducing a new KV bucket

1. Write only from **one actor**. Single-writer invariant.
2. Document the bucket in [`RUNTIME.md`](RUNTIME.md) → KV buckets.
3. Follow naming conventions: `{TYPE}_LATEST` and `{TYPE}_HISTORY` for
   most domains; execution uses action-noun naming
   (see [`domain/execution.md`](domain/execution.md)).

### When introducing a new subject pattern

Subjects follow `{domain}.{plane}.{type}.{verb}[.{key}]` with verb
last. See [`decisions/0009-subject-taxonomy.md`](decisions/0009-subject-taxonomy.md).

If you're introducing a new plane (beyond `events`, `event`, `control`,
`command`, `reply`, `query`, `projection`, `fill`, `rejection`,
`session`, `activation`), it warrants ADR consideration.

---

## Rules for documentation changes

### Verify technical assertions against code before saving

For docs containing factual claims (stream counts, consumer ownership,
type lists, configuration shapes), **verify against the codebase**
before committing the doc.

Patterns from Phase 1A that worked:

- **Inventory-first**: for docs dense in facts, generate an
  inventory of the code state first (read-only grep/find pass),
  then write the doc against that inventory.
- **Slots in proposed content**: when the architect proposes content
  inline, embed `<!-- SLOT N: ... -->` markers where factual tables
  go, and fill from inventory/code at write time.
- **Pause-and-report on divergence**: if the proposed content
  disagrees with code reality, don't silently "fix" — pause, report
  the divergence, and wait for direction.

### Errata: correct immediately, don't accumulate

When documentation A says one thing and documentation B (newer or
verified against code) says another, **fix A in the next commit**.
Don't accumulate contradictions hoping to batch-fix them later.

Phase 1A established this pattern with P1A.4b.1 (errata against
ARCHITECTURE.md and GLOSSARY.md discovered while writing RUNTIME.md).
Small immediate erratta is cheaper than large accumulated ones.

Exception: minor refinements (clarifying wording, improving examples)
can accumulate to a single later pass. Factual contradictions cannot.

### ARCHITECTURE vs RUNTIME — boundary

It's tempting to put concrete operational facts in `ARCHITECTURE.md`.
Resist this.

| Document | Holds |
|---|---|
| `ARCHITECTURE.md` | Durable decisions, structural patterns, foundational principles |
| `RUNTIME.md` | Concrete operational catalog (binaries, streams, ports, KV buckets, subjects) |

If a fact will likely change (a new stream added, a port reassigned),
it belongs in RUNTIME. If a principle will likely persist for years
(layer sovereignty, single-writer invariant), it belongs in
ARCHITECTURE.

When in doubt, prefer RUNTIME. ARCHITECTURE should be the smaller doc.

### Promote guard-rail comments to ADRs

When you find a code-comment explicitly saying "no X here / never do
Y", that's a guard rail. Guard rails should be **promoted to ADRs**:

- Comments can be silently deleted; ADRs cannot.
- Comments are found only by readers of that file; ADRs are
  searchable.
- ADRs explain *why* the constraint exists; comments often don't.

[`decisions/0011-no-oms-expansion-pairing.md`](decisions/0011-no-oms-expansion-pairing.md)
is an example of this promotion — package-level comments in
`pairing.go` and `continuity.go` became an explicit ADR during
Phase 1A.

### Keep RESUMPTION.md current

`RESUMPTION.md` is the entry point for someone returning to the
project. It only earns its keep by being current.

Update RESUMPTION when:

- **Phase transition** (e.g., Phase 1A closes; mark P1A.X items done
  and Phase 1B as in progress).
- **New known gap discovered** (add to the G section).
- **Gap resolved** (remove from G section, add to "Recently resolved"
  appendix if useful, or just remove).
- **Significant feature shipped** (move from "Deliberate non-features"
  to "Current functional state").

If you find yourself wondering whether RESUMPTION reflects reality,
**that itself is the trigger to update it**.

### Keep GLOSSARY.md current

Add to glossary when:

- A term has system-specific meaning beyond generic English/Go usage.
- An existing term's definition changes (rare; usually means a
  major refactor).

Don't add generic terms (NATS, actor, goroutine) — they have upstream
documentation.

---

## PR workflow

### Branch convention

```
type/short-description
```

Examples:
- `feat/backtesting-harness`
- `fix/gateway-route-conflict`
- `docs/architecture-revision`
- `chore/dependabot-upgrade`

### Commit message convention

```
type(scope): summary in present tense

Detail in body, with cross-references where useful. Wrap at 72 chars.

Examples of references:
- Related ADR: see decisions/0010
- Resolves gap: G1 from RESUMPTION
- Test added: cmd/gateway/boot_test.go
```

Types in use:
- `feat`: new capability
- `fix`: bug fix
- `docs`: documentation only
- `chore`: tooling, deps, refactors without behavior change
- `refactor`: structural changes without behavior change

Scope is optional but useful for large changes (e.g.,
`docs(p1a.8a):`, `fix(gateway):`).

### Pre-push validation

Before `git push` on any branch touching production code or
integration paths, run the canonical Makefile targets — **not**
manually reconstructed `go test` commands:

- `make verify` — fast gate (`go test ./...` + repo-consistency
  + quality-gate + proto-lint + lint-go). Always run. **Uses the
  local/fast profile** of the quality gate — does NOT promote
  warnings to errors.
- `raccoon-cli quality-gate --profile ci` — required when the
  change adds or modifies analyzer policy files (anti-patterns,
  domain types, adapters, binaries allowlists). The CI profile
  promotes warning-severity findings to errors via the `[ci]`
  prefix mechanism (`tools/raccoon-cli/src/gate/mod.rs:408`);
  `make verify` alone will pass while CI fails. Run locally
  before pushing changes that touch
  `tools/raccoon-cli/policies/*.toml` or the analyzer source.
- `make test-integration` — integration-tagged tests against a
  local NATS at `localhost:4222` (canonical flags:
  `-tags=integration -short -timeout 18m -count=1`). Required
  before pushing changes that touch actors, adapters, or the
  end-to-end execution path. Start a NATS container first:
  `docker run -d --rm --name nats-local -p 4222:4222 nats:latest -js`.
- `make smoke-*` — operational proofs against a live compose
  stack (`make up && make seed`). Pick the target matching the
  scope of the change (`smoke-analytical`, `smoke-round-trip`,
  `smoke-live-stack`, etc.).

Reconstructing the commands manually (e.g.,
`go test -tags=integration -timeout 10m ./scopes/...`) risks
silent divergence from canonical flags. The `-short` flag in
particular is required to skip endurance tests that exceed
default timeouts. **Always invoke via `make`** so the project's
flag evolution is picked up automatically.

Lesson learned during H-6.b'' pre-push validation: a raw
`go test -tags=integration -timeout 10m ./scopes/...` (no
`-short`) ran for 10 minutes and timed out in
`TestEndurance_CounterMonotonicityUnderRepeatedBursts`, while
the canonical `make test-integration` completed in 1m36s
because the endurance test self-skips under `-short`.

Lesson learned during H-6.c.1 pre-push validation (commit 13):
PR #30 CI failed with 17 anti_patterns errors + 2 topology
errors despite `make verify` GREEN locally on every commit.
Root cause: `make verify` runs the local/fast quality-gate
profile, which leaves warning-severity findings as warnings.
The CI workflow runs `--profile ci` which promotes them to
errors via the `[ci]` prefix mechanism. Adding the
`raccoon-cli quality-gate --profile ci` step to pre-push
validation closes the gap — any analyzer policy change must
be validated against the same profile CI runs.

### PR description

A PR description should:

1. **State the goal** in one sentence at top.
2. **List what changed** at high level (files touched, behavior
   affected).
3. **Cross-reference** affected docs and ADRs.
4. **Identify risk** (e.g., "touches single-writer invariant of
   X stream — careful review needed").
5. **Specify how to verify** (which smoke, which test target).

### Review checklist

Before approving any PR:

- [ ] `make verify` passes (or only the documented G3 `.opencode/`
      failures, until P1B closes that).
- [ ] If routes added: `cmd/gateway/boot_test.go` updated.
- [ ] If new stream/bucket: single-writer invariant respected.
- [ ] If new domain type: canonical `Validate() *problem.Problem`
      signature, or documented deviation.
- [ ] If structural decision: ADR considered.
- [ ] If known-gap touched: RESUMPTION.md updated.
- [ ] If domain-specific deep dive needed: `docs/domain/<x>.md` updated.
- [ ] Layer sovereignty respected (raccoon-cli check passes).

---

## Git hooks (lefthook)

This project uses [lefthook](https://lefthook.dev/) for pre-commit and
commit-msg validation. Hooks check formatting, trailing whitespace,
YAML/JSON/TOML validity, and conventional commit message format.

### Install

```bash
# macOS
brew install lefthook

# Linux / cross-platform
go install github.com/evilmartians/lefthook@latest

# Then activate for this repo
make install-hooks
```

### What runs

- **pre-commit**: gofmt check on staged `.go` files, trailing
  whitespace, YAML/JSON/TOML validation. Fast (<2 sec typical).
- **commit-msg**: conventional commit format
  (`type(scope?): description`) via
  `scripts/validate-commit-msg.sh`. Shell-only — no Node.js dep.
- **post-commit**: `scripts/check-resumption-drift.sh` —
  warn-only check (stderr, exit 0) that surfaces drift when a
  commit references new `M<N>` design-meta identifiers but
  doesn't update `docs/RESUMPTION.md` in the same commit.
  Codifies P5.0 audit finding F4. Non-blocking by design;
  upgrade to commit-msg or pre-push if warn-only proves
  insufficient.
- **pre-push**: `make lint-go` and `make verify` available but
  disabled by default (`skip: true`). Enable in `lefthook.yml` when
  you want stricter local pushes.

### Bypass

Only when truly necessary (e.g., reverting in-flight commits, or
applying an emergency hotfix where the validator itself is broken):

```bash
LEFTHOOK=0 git commit ...
# or
git commit --no-verify
```

CI re-checks every push, so bypass only delays detection — it does
not avoid it.

---

## Claude Code automation

The `.claude/` directory hosts automation surfaces for Claude Code
sessions, codifying patterns proven repetitive across Phase 1+2 work.

- `.claude/commands/`: slash commands invocable as `/<name>` in a
  session.
  - `/check-clean` — pre-action verification (working tree + baseline).
  - `/check-refs <path>` — comprehensive ref search before deletion.
  - `/inventory <area>` — structured inventory production.
  - `/audit <area>` — read-only investigation skeleton.
  - `/version-check` — version consistency across canonical files.
  - `/pre-push` — canonical pre-push validation sequence with
    diff-aware conditionals (see "Pre-push validation" above).
- `.claude/agents/`: agent role templates.
  - `architect-agent` — scoping, framing, and decision discipline;
    the architect side of the two-agent collaboration model
    (added P5.2).
  - `execution-agent` — scoped executor with explicit pause-and-report
    protocol.
- `.claude/skills/`: procedural-knowledge skills (SKILL.md format)
  auto-loaded by Claude Code when semantically relevant to the task.
  - `investigation-skill` — read-only investigation pattern
    (time cap, `/tmp/` audit-file convention, categorization
    framework, A/B/C/D recommended-next-step). Codifies the
    Phase 4 "investigate before prescribe" discipline.
  - `fix-prompt-skill` — change-applying prompt pattern
    (bundle/split decision, defensive scan, structured commit
    message, CI monitoring).
  - `wave-prompt-skill` — Harvest wave-cycle pattern (pré-flight,
    cross-check protocol, prompt anatomy, mea culpa discipline,
    merge-gated closure).
- `.claude/settings.json`: session-level config —
  `RACCOON_REFERENCE_PATH` env, `permissions.deny` for raccoon
  writes, hooks wiring (ADR-0026).
- `.claude/hooks/`: enforcement hooks for P2 (raccoon read-only)
  and P9 (no agent push to `main`, no `--no-verify`; `gh pr merge`
  asks) + session-start orientation. See ADR-0026.

See `.claude/README.md` for the full index and philosophy. These
surfaces are descriptive helpers — `CLAUDE.md` (repo root) remains
the canonical operating instructions for Claude sessions. Skills
encode patterns observed across 21+ Phase 4 sub-prompts (per the
P5.0 audit) and complement, rather than replace, the slash
commands.

---

## Authorized expansion protocol

The single most important protocol in this repository.

### Why this exists

When executing a scoped task (a PR, a documentation prompt, a focused
refactor), it's common to discover **a blocker or improvement that's
outside the task's stated scope**. Two failure modes:

1. **Silent expansion**: agent (human or AI) decides to fix the
   blocker without notice. The PR balloons; the original scope is
   muddied; reviewers can't tell what was intended vs incidental.
2. **Silent skip**: agent ignores the blocker and proceeds with the
   original task. The blocker stays unfixed; the PR ships an
   incomplete fix or a workaround.

The authorized expansion protocol prevents both.

### Protocol

When you (human or AI) encounter a blocker or improvement outside
the stated scope:

1. **Pause work.** Do not modify files related to the unscoped concern.
2. **Report concisely.** Describe the finding in 3-5 lines.
3. **Present options.** Typically 2-4 labeled choices (A, B, C, D)
   covering: include in current scope; defer to follow-up; skip
   entirely; clarify with project owner.
4. **Wait for direction.** Do not act unilaterally.
5. **Proceed per direction.** If the response authorizes expansion,
   the commit message includes an "Authorized expansion:" section
   describing what was added beyond the original scope.

### Examples from Phase 1A

- **P0.2**: Go version upgrade revealed a stale test file requiring
  an unrelated fix. Pause-report-options-act protocol was followed;
  the fix shipped with explicit authorization.
- **P0.6**: One route conflict was the original target; a second and
  third were discovered during work. Each was paused, reported, and
  authorized before proceeding.
- **P1A.3**: ARCHITECTURE.md draft contained factual divergences
  from the codebase. Save was paused; divergences reported with
  patch options; correct content shipped after authorization.

This protocol works because it preserves the project owner's ability
to **direct the scope** without forcing every agent decision to be
pre-specified.

---

## When to write an ADR

Write an ADR when:

- The decision is **structurally durable** (will affect many future
  changes).
- The decision **inverts a default** or pattern that was previously
  established.
- The decision has **significant trade-offs** that future readers
  need to understand.
- The decision encodes a **guard rail** that should persist regardless
  of code churn.

Don't write an ADR for:

- Pure refactorings without policy change.
- Bug fixes (commit message is sufficient).
- Tool upgrades (commit message or chore PR is sufficient).
- Configuration values (these go in deployment docs).

### ADR format

See [`decisions/README.md`](decisions/README.md) and the existing
ADRs (`decisions/0001-*.md` through `decisions/0011-*.md`) for the
canonical format. Briefly:

- **Status**: Accepted, Superseded, Deprecated
- **Context**: the situation that motivated the decision
- **Decision**: what we decided
- **Consequences**: positive AND negative
- **Alternatives considered**: what we rejected and why
- **References**: cross-refs to code and docs

ADRs are **append-only**. To change a decision, write a new ADR
that supersedes the old one. Do not edit historical ADRs except for
typos and broken links.

---

## Coding standards

### Go code

- `gofmt` clean.
- `go vet` clean.
- `raccoon-cli quality-gate` passes (covers layer sovereignty, structural rules).
- Test coverage required for new domain logic (`make test` passes).
- Behavioral tests required for new operational paths (`make test-behavioral`).

### Rust code (raccoon-cli)

- `cargo fmt` clean.
- `cargo clippy` clean.
- `cargo test` passes.

### JSONC configs

- Valid JSONC (parseable with comments).
- Schema verified by `make codegen-validate-all`.

### Markdown docs

- No `make verify` cross-reference failures (except documented G3
  `.opencode/` until P1B).
- Internal links work.
- Headings are sentence-case.

---

## For AI agents (institutional knowledge)

If you are an AI agent (Claude Code, another agent) executing a task
on this repository, this section is the **cumulative knowledge base**
— what we've learned the hard way across Phase 1+2+3 work.

`CLAUDE.md` (repo root) describes session-level operating protocols.
This section complements it; it does not replace it.

### Read these documents first

In order:

1. The prompt you received (your immediate task).
2. [`RESUMPTION.md`](RESUMPTION.md) — current state.
3. [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape.
4. Any specific docs the prompt references.

### Apply the protocols rigorously

- **Validate against code** before claiming any technical fact.
- **Pause and report** if you find divergences from the prompt's
  expectations; do not silently "fix".
- **Pause and report** if you find blockers outside scope; use the
  authorized expansion protocol.
- **Update RESUMPTION.md** if your task transitions a phase or
  changes a documented gap.
- **Update affected docs** when behavior changes (don't let
  documentation drift while code moves).

### Commit messages: explicit about provenance

When AI-driven, commit messages should:
- Identify the prompt or task (e.g., `docs(p1a.9): write CONTRIBUTING.md`).
- Be clear about what the agent decided vs what the project owner
  authorized.
- Reference any "Authorized expansion:" if scope grew.

### When in doubt

Pause and ask. The cost of one extra clarification turn is much
less than the cost of an incorrect autonomous decision that requires
unwinding.

### Operating philosophy

Three principles, in priority order:

1. **Honest over impressive**: report what is, not what would sound
   confident. "I'm not sure" beats "Confirmed!" without evidence.
2. **Investigate before execute**: structure-shaping changes are
   preceded by inventory or audit phases. Producing structured
   `/tmp/<topic>.md` artifacts is normal and expected.
3. **Atomic per concern**: one commit = one responsibility. Multiple
   concerns produce multiple commits, even if they happen in one
   session.

### Pause-and-report protocol (5 steps)

When reality diverges from the prompt's premise (different file
count, different structure, different state), follow:

1. **Pause** — stop applying changes immediately.
2. **Report** — summarize what was expected vs what was found, with
   concrete evidence (paths, line numbers, exit codes).
3. **Options** — provide owner with 2–4 distinct paths forward
   (A/B/C/D), each with tradeoffs.
4. **Wait** — do not proceed without explicit direction.
5. **Proceed** — only after authorization. Reference the chosen
   option in the eventual commit message.

This protocol caught real divergences in Phase 1+2+3:

| Sub-phase | Divergence caught |
|---|---|
| P2.3 | `GO_VERSION` premise was about toolchain, not project-declared |
| P2.Y | `docs/legacy/` refs still active in `scripts/bootstrap-check.sh` |
| P3.3 | GitHub fork lockdown blocked by personal-tier platform policy |
| P3.5 | "Scripts missing `set -e`" audit finding was wrong (all 41 had `set -euo pipefail`) |
| P3.7 | `golangci-lint` version not pinned in CI workflow (action default) |

Without pause-and-report, each would have led to broken or
mis-prioritized work.

### Common patterns

#### Pre-action working tree verification

Every significant action begins with:

```bash
git status --porcelain          # must be empty
git fetch origin
git rev-list --count origin/main..HEAD   # must be 0
make verify                     # must PASS
```

The slash command `/check-clean` (in `.claude/commands/`) wraps
this. If any check fails, stop and report.

#### Cross-reference search before deletion

Before deleting or renaming any tracked path, search comprehensively
for active references across source code, config files, docs,
Makefile, and CI workflows. The slash command `/check-refs <path>`
wraps this.

This pattern surfaced repeatedly in Phase 1+2 as
stale-infrastructure-post-restructure (orientation/validation
breaks after a rename or deletion that missed a reference).

#### Inventory-first for fact-dense changes

When a change touches dense factual surface (file lists, version
declarations, configuration matrices), produce a structured
inventory first to `/tmp/<topic>-inventory.md`, then transform from
inventory. Used in P1A.4a (runtime inventory), P1A.6a (domain
inventory), P2.X.0 (scripts inventory), P3.0 (full environment
audit). The slash command `/inventory <area>` wraps this.

#### Atomic commits per concern

If a session naturally splits into multiple concerns (e.g., "fix
bug A and refactor B"), produce multiple commits — one per
concern. Sessions producing one gargantuan commit covering
everything are a signal of merged concerns.

### Validation discipline

#### Distinguish project-declared vs tool-environment version

When verifying versions, identify the source of truth:

| Source | Authoritative for |
|---|---|
| `go.work` | Go workspace version |
| `go.mod` per module | Go module version (typically matches workspace) |
| `tools/raccoon-cli/rust-toolchain.toml` | Rust version |
| `.tool-versions` | asdf/mise manifest (consumes the above) |
| `.github/workflows/ci.yml` | CI runtime version |
| Local toolchain | Whatever is installed (may legitimately differ) |

P2.3 mistake: investigation reported "go.work says 1.25.7, toolchain
says 1.26.2" and treated it as drift-to-fix. Reality: the toolchain
was newer than the workspace declaration, which is fine. The
slash command `/version-check` wraps cross-validation.

#### Audit-heuristic validation

When an audit reports "X% missing convention Y" where Y is widely
adopted, **double-check before remediating**.

Heuristics like `head -N | grep` miss content beyond the first N
lines. For findings about widely-adopted conventions (`set -e`,
`gofmt`, etc.), validate with a dedicated tool (`shellcheck`,
`gofmt -l`, `cargo clippy`, etc.) before planning work.

P3.5 mistake: audit claimed "39/39 scripts missing `set -e`".
Reality: all 41 had `set -euo pipefail` after the header comment
block (declared at lines 7–49). The audit's `head -10 | grep`
missed it. P3.5 was re-scoped to use shellcheck and surfaced 7
real safety issues instead.

#### Format validation pre-commit

`lefthook.yml` validates YAML/TOML/JSON, trailing whitespace, and
gofmt at commit time. Don't bypass without reason; CI re-checks
every push.

### Cross-platform quirks

#### Shell quoting

Word-splitting differs between zsh and bash. Prefer:

- Quote variables: `"$VAR"` not `$VAR`.
- Use `read -ra ARRAY <<< "$VAR"` for array splitting (vs the
  word-splitting `ARRAY=($VAR)` pattern flagged by shellcheck SC2206).
- For dynamic list generation, prefer Python (more predictable
  across shells).

#### `sed -i` macOS vs Linux

macOS BSD `sed` requires an extension arg; GNU `sed` does not:

```bash
# Portable wrapper
if [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i '' "$@"
else
    sed -i "$@"
fi
```

Or just use Python for in-place edits — fewer cross-platform
surprises.

### Lessons learned (Phase 1+2+3 errata)

Specific mistakes documented to avoid repetition:

1. **P2.3** — Don't conflate toolchain-environment with
   project-declared versions. They can legitimately differ.
2. **P2.Y** — Refs to legacy paths can survive in unexpected places
   (e.g., bootstrap validators). Cross-search broader than
   "obvious" locations.
3. **P3.3** — GitHub features have tier limits. Personal public
   repos can't disable forks via UI/API. Verify platform
   constraints before planning remediation.
4. **P3.5** — Heuristic audits (`head | grep`) can miss content
   beyond the inspection window. Validate with dedicated tools.
5. **P3.7** — CI workflows can use action defaults (no explicit
   `version:` pinning), creating drift between local-tested and
   CI-executed versions.

### Audit and investigation patterns

These patterns emerged from Phase 4.1 audit work. Each is grounded
in a real near-miss or catch — not aspirational advice. Apply
during code review, before commits, and especially before
recommending deletions or "alignment" fixes.

#### Audit recommendations are deletions-in-disguise

If an audit says "remove X" or "delete Y", apply the same cross-
reference rigor as actual deletion. Grep code, configs, Makefile,
docs for active references — especially for items whose purpose
isn't obvious from their location alone. Examples: empty-looking
directories that turn out to be load-bearing for Makefile targets;
`.gitignore` shims that turn out to be deliberate "preserve empty
dir" patterns.

P4.0's pre-audit caught a near-miss: a recommendation to
`rm -rf backups/` would have destroyed 96 files of operational
ClickHouse backup data and broken 6 Makefile targets.
Pause-and-report stopped the action before damage; verifying
references is the rule the near-miss established.

#### Read analyzer logic, not just its output

Before prescribing a "mechanical alignment" fix between two
components flagged by a static check, verify they're the same kind
of mechanism:

- **Static table ↔ static table**: probably mechanical alignment
  (e.g., adding an entry).
- **Static table ↔ scanner output**: investigate what the scanner
  discovers; the fix may require a source code change, not a table
  edit.
- **Scanner ↔ scanner**: investigate both; may need ground-truth
  identification before any change.

Reading the analyzer's actual Rust source clarifies which case
applies. Output alone is insufficient. P4.1.3.a's pause-and-report
caught this exact gap: the prompt assumed `contract-audit` had a
hardcoded table when it was a real cross-reference scanner.

#### Static enforcement rules should match architectural intent

A blanket rule that's stricter than the relevant ADR creates two
problems: false positives flag legitimate code, AND the rule's lax
shape may still miss real violations.

P4.1.3.c made this visible. A "cmd → domain import forbidden" rule
overshoots ADR-0005's "cmd sees everything" and yet missed the
real boundary concern (function-call invocation). The refined rule
— flagging only function calls, permitting type references —
aligned both with ADR intent and with day-to-day code reality.

When writing or refining a static check, articulate the
architectural primitive being enforced (composition vs invocation,
type vs value, layer-bound vs layer-crossing) and design the
predicate around that primitive, not its rough approximation.

#### CI red status accumulates technical debt invisibly

When CI is red for any reason, every subsequent commit erodes
signal value. New failures hide under the existing red status.
Phase 4.1's SHA-pinning migration revealed two latent failures
(`golangci-lint-action` v6 args silently ignored on v9, plus 11
quality-gate-ci warnings promoted to errors) that had accumulated
since P3.3 — masked by the workflow-rejection layer.

Operational implication: restoring green CI takes priority over
adding new work on a red baseline. Even when the existing red
status is "known and accepted", treat it as opaque debt —
investigate before proceeding with substantial code work.

#### Distinguish process debt from regression

When a scanner flags an existing violation that surfaced for the
first time, the question to answer is: did the code change
recently, or did the visibility change?

If the code is stable and the violation has existed for a while,
that's **process debt**: the scanner caught what should have been
caught earlier. No regression. Fix forward; no `git log` archaeology
needed.

If the code changed recently and broke an invariant, that's
**regression**. Identify the culprit commit; consider revert as
well as forward fix.

The two need different fix strategies. Conflating them leads to
unnecessary archaeology, or worse, suppressing the check when the
underlying invariant is sound. P4.1.2 framed all 11 quality-gate-ci
findings as process debt before any fix attempt, which kept the
P4.1.3 sub-prompts focused on the right kind of intervention.

### Anti-patterns to avoid

- **Reframe to fit**: if a prompt's premise doesn't match reality,
  don't reframe reality to make the prompt valid. Pause and report.
- **Aggregate concerns**: don't bundle unrelated changes into one
  commit "because they're small". Atomic per concern.
- **Trust narrative reference**: documentation describing past state
  may be outdated. Verify against current files before relying on
  doc claims.
- **Skip validation**: never declare success without `make verify`.
- **Bypass safety hooks** (`--no-verify`, `--no-gpg-sign`) without
  owner authorization. If a hook fails, investigate and fix; don't
  paper over.

---

## Reading further

| If you want | Go to |
|---|---|
| System overview | [`README.md`](README.md) |
| Current state and gaps | [`RESUMPTION.md`](RESUMPTION.md) |
| Architecture and patterns | [`ARCHITECTURE.md`](ARCHITECTURE.md) |
| Runtime topology | [`RUNTIME.md`](RUNTIME.md) |
| HTTP endpoints | [`HTTP-API.md`](HTTP-API.md) |
| Daily workflow | [`DEVELOPMENT.md`](DEVELOPMENT.md) |
| Operations | [`operations/`](operations/README.md) |
| Architecture decision records | [`decisions/`](decisions/README.md) |
| Domain deep dives | [`domain/`](domain/README.md) |
| Terminology | [`GLOSSARY.md`](GLOSSARY.md) |
| Historical material | [`legacy/`](legacy/README.md) |
