# market-foundry — Documentation

This is the canonical documentation surface for market-foundry: a
domain-oriented runtime foundation for market data processing.

## What this system is

A Go workspace composed of seven long-running binaries that ingest market
data, derive evidence and signals, evaluate decisions and risk, control
execution, and persist analytical history. Communication happens through
NATS+JetStream. Analytical storage is ClickHouse. A Rust CLI (`raccoon-cli`)
enforces architecture rules automatically.

This is **not** a trading application. It is the foundation on which
trading capabilities are built.

## Reading order

Read in this order if you are new or returning after time away:

1. [`RESUMPTION.md`](RESUMPTION.md) — current state, known gaps, next step
2. [`ARCHITECTURE.md`](ARCHITECTURE.md) — system architecture in one document
3. [`RUNTIME.md`](RUNTIME.md) — binaries, streams, KV buckets, ports
4. [`HTTP-API.md`](HTTP-API.md) — gateway HTTP endpoints
5. [`DEVELOPMENT.md`](DEVELOPMENT.md) — daily development flow
6. [`CONTRIBUTING.md`](CONTRIBUTING.md) — PR rules and review checklist

For domain-specific deep dives, see [`domain/`](domain/README.md).
For operational guides, see [`operations/`](operations/README.md).
For durable design decisions and their rationale, see
[`decisions/`](decisions/README.md).
For terminology, see [`GLOSSARY.md`](GLOSSARY.md).

## Documentation principles

- **One document per recurring question.** No taxonomy with five files
  routing to each other.
- **Reflect reality, not aspiration.** If a feature is partial, the doc
  says so plainly.
- **Code is the source of truth.** Docs explain intent, structure, and
  context. For exact behavior, read the code.
- **History lives in git.** Decisions live in `decisions/`. The
  pre-2026-05 documentation set is preserved in git history only
  (the `docs/legacy/` tree was retired in P2.Y).

## Historical material

Everything that used to live under `docs/architecture/`, `docs/stages/`,
`docs/archive/`, `docs/operations/`, `docs/tooling/`, `docs/product/`,
and `docs/development/` was moved to `docs/legacy/` during the
Phase 1A reset, and the legacy tree itself was retired in P2.Y.
That material is accessible through git history (e.g.,
`git log --diff-filter=D -- docs/legacy/`) and was never
authoritative post-reset.
