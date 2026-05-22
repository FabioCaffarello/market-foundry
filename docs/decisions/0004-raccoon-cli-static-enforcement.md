# ADR 0004: raccoon-cli for static architecture enforcement

## Status

Accepted.

## Context

Architecture rules in market-foundry need enforcement:

- Layer sovereignty (`domain → application → adapters → actors → interfaces → cmd`)
- Single-writer invariant per stream and KV bucket
- Naming and structural conventions (per-domain adapter packages, per-family use case groups)
- Cross-reference integrity in documentation

The alternatives for enforcement:

- **Code review only** (convention-based)
- **Go linters / static analysis tools** (custom golangci-lint rules)
- **A bespoke enforcement tool** (built specifically for this project)

The system's requirements:

- Enforce rules **automatically** in CI, not by reviewer discipline.
- Read code (Go source), configs (JSONC), and docs (Markdown) — across
  all artifact types, not just Go.
- Be independent of the Go build (so it can validate even if
  build is broken).
- Be invocable from `make` for local validation.

## Decision

**raccoon-cli is the architecture enforcement tool.** It is written
in Rust (independent from the Go build), reads files directly, and
runs as part of `make check`, `make verify`, `make tdd`, and CI.

Key characteristics:
- Never imported by Go code (Rust binary, separate toolchain).
- Reads files; runs subprocesses only for bounded support checks.
- Provides `check`, `inspect`, `change` command families.
- Surfaced through Makefile targets (`make arch-guard`, `make drift-detect`,
  `make quality-gate`, etc.).

## Consequences

### Positive

- **Independence**: raccoon-cli works even if Go code doesn't build.
  Layer violations can be detected before they become compile errors.
- **Cross-artifact**: enforces rules across Go, JSONC, Markdown
  uniformly. A bespoke Go linter would only cover Go.
- **Performance**: Rust binary is fast; large codebase checks complete
  in seconds.
- **Single point of enforcement**: all rules in one tool. Adding a
  new rule means adding to raccoon-cli, not to N different linters.
- **Static enforcement vs convention**: the rules **must pass** in
  CI; they're not discretionary.

### Negative

- **Custom tool maintenance**: raccoon-cli is bespoke; team must
  maintain it. No community contributions, no upstream upgrades.
- **Two-language repository**: Rust and Go both. Increases tooling
  setup complexity for new contributors.
- **Learning curve**: contributors need to understand raccoon-cli
  output and how to fix the errors it raises.
- **Coupling**: project velocity depends partly on raccoon-cli
  being maintained.

### Mitigation

- raccoon-cli is intentionally **small** in scope. It enforces rules,
  it doesn't try to be a general-purpose tool.
- The rule set is small enough to be auditable; new rules are added
  deliberately, not opportunistically.

## Alternatives considered

**Code review only**: rejected because human review drifts. Five years
of disciplined reviews can be undone by one rushed PR; static
enforcement doesn't drift.

**golangci-lint with custom analyzers**: workable for Go-only rules
but cannot enforce rules across JSONC configs, Markdown docs, or the
relationships between them. Multiple tools also create the "which
tool said what" problem.

**Convention documents alone (the previous model)**: tried and
failed in the previous evolution model — 822 architecture documents
accumulated, and rules were inconsistently applied because nobody
re-read all 822 before each change.

## References

- `tools/raccoon-cli/` — implementation in Rust
- `tools/raccoon-cli/Cargo.toml` — dependencies
- Makefile targets: `arch-guard`, `drift-detect`, `quality-gate`,
  `quality-gate-deep`, `tdd`
- [`../DEVELOPMENT.md`](../DEVELOPMENT.md) → "Architecture enforcement"
- [`../ARCHITECTURE.md`](../ARCHITECTURE.md) → "Foundational principles"
  → "Static enforcement over convention"
