<!--
Pull Request template for market-foundry.
For full PR rules, see docs/CONTRIBUTING.md → "PR workflow".
-->

## Goal

<!-- One sentence: what does this PR accomplish? -->

## What changed

<!-- High-level summary of what was modified. List files or directories
     touched and the behavioral impact. -->

## Why

<!-- Motivation. Link to related issue if applicable. -->

Related: <!-- #issue-number or "none" -->

## Risk

<!-- Identify what could go wrong. Examples:
     - "touches single-writer invariant of EXECUTION_EVENTS stream"
     - "modifies route registration; boot_test.go updated to match"
     - "low risk; doc-only change"
-->

## How to verify

<!-- Specific commands or smokes to run. Examples:
     - "make smoke && make smoke-multi"
     - "make smoke-analytical"
     - "make test ./internal/domain/strategy/..."
-->

## Checklist

<!-- Mark with x. Skip irrelevant items but state why if asked. -->

- [ ] `make verify` passes (or only fails on documented G6 — see RESUMPTION.md)  
      <!-- Note: CI's `repository-checks` job is also red on G6 until the analyzer is fixed -->
- [ ] If routes added: `cmd/gateway/boot_test.go` `routes` slice updated  
      <!-- Required by ADR 0010 (docs/decisions/0010-httprouter-trie-constraints.md) -->
- [ ] If new stream or KV bucket: single-writer invariant respected  
      <!-- ADR 0008 (docs/decisions/0008-single-writer-invariant.md) -->
- [ ] If new domain type: canonical `Validate() *problem.Problem` signature, or documented deviation
- [ ] If structural decision made: ADR considered (`docs/decisions/`)
- [ ] If known gap touched: `docs/RESUMPTION.md` updated
- [ ] If behavior changed: affected docs updated (`docs/domain/`, `docs/operations/`, etc.)
- [ ] Layer sovereignty respected (`raccoon-cli quality-gate` would pass — note G6 caveat)

## Authorized expansion

<!-- If this PR's scope grew beyond the original task or issue,
     describe what was added and why it was authorized.
     Otherwise leave "None." -->

None.
