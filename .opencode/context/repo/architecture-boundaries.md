# Architecture Boundaries

Use this file when a change may cross layers, binaries, streams, or old
prohibitions.

Canonical owner docs:

- `../../../docs/architecture/README.md`
- `../../../docs/architecture/market-foundry-evolution-playbook.md`
- `../../../docs/architecture/stage-definition-of-done.md`
- `../../../docs/architecture/anti-debt-checklist.md`
- `../../../docs/architecture/actor-ownership.md`
- `../../../docs/architecture/stream-family-catalog.md`
- `../../../docs/architecture/prohibited-carryovers.md`
- `../../../internal/README.md`

Binding layer order:

`domain -> application -> adapters -> actors -> interfaces -> cmd`

Enforcement entrypoints:

- `make arch-guard`
- `make drift-detect`
- `make check`
- `make verify`

Use these repository maps when the boundary question is physical as well as
logical:

- `../../../cmd/README.md` for process/bootstrap ownership
- `../../../internal/README.md` for code placement and dependency direction
- `../../../deploy/README.md` for runtime assets, compose, envs, and migrations

Do not reintroduce:

- Kafka adapters or infrastructure
- old quality-service binaries or naming
- `.context/` structure

If the change alters boundaries, update the canonical architecture docs instead
of expanding `.opencode`.
