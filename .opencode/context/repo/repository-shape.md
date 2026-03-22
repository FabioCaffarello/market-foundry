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
- `docs/` owns canonical explanation
- `tools/raccoon-cli/` owns structural inspection tooling

Use `../../../docs/architecture/monorepo-structure-and-engineering-conventions.md`
or `../../../docs/architecture/repository-architecture-census-and-refactor-map.md`
when task placement depends on architectural detail, not just path selection.
