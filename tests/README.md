# Test Surface Map

`tests/` holds repository-level test assets that are not just package-local Go
tests.

## Current Layout

| Directory | Purpose | Use it when |
|---|---|---|
| `http/` | HTTP request collections for gateway and read/query surfaces | you need to inspect or manually exercise public HTTP behavior |

## How This Relates To Other Tests

- Package-level Go tests live next to the code they validate under `cmd/`,
  `internal/`, and `tools/raccoon-cli/`.
- `tests/http/` is the shared repository test asset area for manual or
  request-file-driven verification.
- Operational proofs still start from `make smoke*`, not from this directory.

## Maintenance Rules

- Add new top-level subdirectories here only when the test asset is shared
  across repository areas and would be hard to discover in-package.
- If a new shared test surface becomes important for daily work, link it from
  [`../DEVELOPMENT.md`](../DEVELOPMENT.md) or the relevant operations doc.
