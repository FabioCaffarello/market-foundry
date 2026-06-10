---
name: pre-push
description: Run the canonical pre-push validation sequence (verify → profile-ci quality gate → integration tests) with diff-aware conditionals.
---

Run the canonical pre-push validation sequence from
`docs/CONTRIBUTING.md` → "Pre-push validation" before pushing the
current branch. The conditional steps are decided by what the
branch actually changed relative to `origin/main`.

## Why

H-6.c.1 (PR #30) failed CI with 17 anti_patterns + 2 topology
errors despite `make verify` GREEN on every local commit: the CI
quality-gate profile promotes warning-severity findings to errors,
the local profile does not. H-6.c.2 commit 5 found 6
integration-tagged stragglers that `make verify` cannot see.
This command exists so the full sequence is one invocation away.

## Steps

1. **Compute the changed surface**:

   ```bash
   git fetch origin main --quiet
   CHANGED=$(git diff --name-only origin/main...HEAD)
   echo "$CHANGED"
   ```

2. **Always** — `make verify`. Must be GREEN. If a test fails,
   check `docs/RESUMPTION.md` → flake registry (G7, G8, G9, ...) and
   apply the registered isolated re-run procedure before treating
   it as a regression.

3. **If** `$CHANGED` touches `tools/raccoon-cli/policies/*.toml`
   or `tools/raccoon-cli/src/` —
   `(cd tools/raccoon-cli && cargo run --quiet -- quality-gate --profile ci)`
   (or the repo's canonical invocation). The CI profile promotes
   warnings to errors; local verify alone will pass while CI fails.

4. **If** `$CHANGED` touches `internal/actors/`,
   `internal/adapters/`, or the end-to-end execution path —
   `make test-integration` against a local NATS:

   ```bash
   docker run -d --rm --name nats-local -p 4222:4222 nats:latest -js
   make test-integration
   ```

5. **Report** the three outcomes (run / skipped-with-reason / PASS
   / FAIL) before pushing. A skipped step needs its reason stated —
   silent skipping is forbidden (CLAUDE.md core protocol 2).

## See also

- `docs/CONTRIBUTING.md` → "Pre-push validation" — the canonical
  sequence this command automates (authoritative on flags and
  lessons).
- `lefthook.yml` → pre-push — git-side automatic `make verify` on
  every push (this command adds the diff-aware conditional steps).
- `.claude/skills/fix-prompt-skill/SKILL.md` → Execução step 6.
