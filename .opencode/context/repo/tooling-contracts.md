# Tooling Contracts

Use this file when choosing the execution surface for a task.

Canonical owner docs:

- `../../../docs/operations/make-and-raccoon-cli-contract.md`
- `../../../docs/tooling/cli-overview.md`
- `../../../docs/operations/raccoon-cli-command-reference.md`
- `../../../tools/raccoon-cli/README.md`
- `../../../scripts/README.md`

Contract:

- `make` owns the public repository workflow
- `make smoke*` owns runtime proof-of-record
- `raccoon-cli` owns strategic inspection, impact analysis, TDD guidance, drift,
  and architecture safety
- `scripts/*.sh` own harness implementation detail behind `make`

Canonical `make` wrappers around tooling intelligence:

- `make tdd`
- `make coverage-map`
- `make briefing`
- `make recommend`
- `make arch-guard`
- `make drift-detect`
- `make quality-gate`
- `make quality-gate-deep`

Use direct `raccoon-cli` only when you need:

- narrower expert inspection
- JSON or structured output
- CLI implementation work under `tools/raccoon-cli/`

Do not turn `.opencode` into another command catalog. Point to the owner docs.
