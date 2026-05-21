# Repository Shape

Use this file when deciding where code, config, docs, or tests belong.

Repository anchors:

- `../../../README.md` for system identity and top-level layout
- `../../../cmd/README.md` for binary ownership and process entrypoints
- `../../../internal/README.md` for layer placement inside `domain -> application -> adapters -> actors -> interfaces -> cmd`
- `../../../deploy/README.md` for compose, configs, envs, and migrations
- `../../../scripts/README.md` and `../../../tests/README.md` for harness and shared test surfaces

Compression:

- `cmd/` owns binaries
- `internal/` owns implementation
- `deploy/` owns runtime assets
- `docs/` owns human explanation
- `docs/stages/` owns immutable evidence
- `tools/raccoon-cli/` owns structural inspection tooling

If placement changes ownership, contracts, or layering, confirm against
`../../../docs/architecture/market-foundry-evolution-playbook.md` and
`../../../docs/architecture/anti-debt-checklist.md` before editing.
