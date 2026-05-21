# Runtime Navigation

Canonical owner docs:

- `../../../README.md`
- `../../../DEVELOPMENT.md`
- `../../../docs/development/workflow.md`
- `../../../docs/development/commands-and-proofs.md`
- `../../../deploy/README.md`
- `../../../docs/architecture/current-baseline-runbook.md`

Start here by question:

- what runs locally -> `services-topology.md`
- where compose, env, config, and migrations live -> `configs-compose-streams.md`
- which flow to run: `make live`, `make up`, `make seed*`, `make smoke*` -> `smoke-and-live-flows.md`
- what to do when runtime proof fails -> `troubleshooting-paths.md`

Default entrypoints:

- `make bootstrap` before first bring-up or after machine drift
- `make live` for fastest bring-up
- `make up` + `make seed` or `make seed-multi` for controlled bring-up
- `make smoke-help` before choosing a proof
- `make smoke*` as proof-of-record
- `make diag`, `make ps`, `make logs SERVICE=...`, `SERVICE=... make restart` first in troubleshooting

Use direct scripts or substrate tools only for harness debugging or low-level
runtime investigation. `raccoon-cli` is not the runtime orchestrator.
