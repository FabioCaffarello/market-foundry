# Configs, Compose, Streams

Use `make` first; inspect assets only when the issue is below the public workflow.

Where each runtime input lives:

- stack wiring: `../../../deploy/compose/docker-compose.yaml`
- shared env defaults: `../../../deploy/envs/local.env.example`, `../../../deploy/envs/local.env`
- service configs: `../../../deploy/configs/`
- config reference: `../../../deploy/configs/CONFIG-REFERENCE.md`
- migrations: `../../../deploy/migrations/`, `../../../cmd/migrate/`

When to open these files:

- compose/startup mismatch or ports/depends_on/healthchecks drift
- config key added or service boot fails on `-config`
- ClickHouse schema or migration issue after `make up`
- `raccoon-cli check topology` or `make arch-guard` points at wiring drift

Prefer `make compose-config`, `make migrate-status`, and `make migrate-validate` before manual edits.
