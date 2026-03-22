# Deployment And Runtime Assets

`deploy/` contains the local runtime assets that make the repository operable.

Use this directory when you need to answer:

- which local stack assets back `make up`, `make live`, and `make smoke*`;
- where service config and env examples live;
- where ClickHouse and NATS runtime definitions come from.

## Area Map

| Directory | Purpose | Use it when |
|---|---|---|
| `compose/` | Docker Compose topology | you need to inspect or change the local stack definition |
| `configs/` | Service config files and examples | you need to know what a runtime consumes at startup |
| `docker/` | Docker build definitions | you are changing container build behavior |
| `envs/` | Shared local env files and examples | you need local overrides or onboarding defaults |
| `migrations/` | ClickHouse forward-only migration catalog | you are evolving analytical schema |
| `nats/` | NATS server configuration | you need transport-level runtime settings |
| `clickhouse/` | ClickHouse local server assets | you need local analytical database bootstrap details |

## Start Here By Task

| Task | Start with |
|---|---|
| bring up or debug the local stack | `compose/docker-compose.yaml` and [`../DEVELOPMENT.md`](../DEVELOPMENT.md) |
| understand config inputs for a service | `configs/CONFIG-REFERENCE.md` and the relevant `*.jsonc` file |
| change environment defaults | `envs/local.env.example` then `envs/local.env` |
| add or review ClickHouse schema | `migrations/` and `../cmd/migrate/` |
| debug broker/database runtime config | `nats/` or `clickhouse/` |

## Maintenance Rules

- Keep environment examples and config references aligned when new runtime
  settings are introduced.
- If a new runtime asset changes the public workflow, update this file and
  [`../README.md`](../README.md) or [`../DEVELOPMENT.md`](../DEVELOPMENT.md).
- Prefer linking to canonical docs instead of copying operational runbooks here.
