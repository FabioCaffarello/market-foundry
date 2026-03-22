# Impact Analysis

Use this when the change question is “what else moves with this file?”

Default path:

- `make tdd`
- `make briefing TARGETS=...` for a short scoped summary
- `make recommend TARGETS=...` when choosing validation depth

Go direct to `raccoon-cli` when you need narrower analysis:

- `raccoon-cli change impact [TARGET...]`
- `raccoon-cli change briefing [TARGET...]`
- `raccoon-cli change rename <SYMBOL>`

Good target classes in this repo:

- `deploy/compose/` or `deploy/configs/` changes -> topology, smoke, and startup impact
- `scripts/` or `Makefile` changes -> workflow and guard-rail impact
- `internal/actors` or `internal/adapters/nats` changes -> runtime path plus `make smoke*`
- `tools/raccoon-cli/` changes -> `make raccoon-test` and wrapper checks
