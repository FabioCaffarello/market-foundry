# Developer Onboarding And Troubleshooting Guide

## Purpose

This guide is the task-oriented runbook for developers operating
`market-foundry` day to day.

Use it when you need a concrete path, not a taxonomy.

## Onboarding Path

### New machine or changed environment

Run:

```bash
make help
make bootstrap
```

If `make bootstrap` fails, fix that first. Do not skip ahead to smoke or code
changes while the machine-level prerequisites are unresolved.

### Fastest path to a working local stack

Run:

```bash
make live
make smoke
```

This is the default onboarding path for contributors who do not need to inspect
intermediate steps.

### Controlled manual bring-up

Run:

```bash
make up
make seed
make smoke
```

Use `make seed-multi` and `make smoke-multi` when you need the governed
multi-symbol path.

### First coding loop

Run:

```bash
make check
make tdd
# implement the smallest correct change
make verify
```

## Daily Operating Paths

| Situation | Command path |
|---|---|
| I want the repository to do the full bring-up for me | `make live` |
| I need to inspect startup step by step | `make up` then `make seed*` then `make smoke*` |
| I changed baseline runtime behavior | `make smoke` |
| I changed multi-symbol behavior | `make smoke-multi` |
| I changed analytical write/read behavior | `make smoke-analytical` |
| I changed lifecycle or restart behavior | `make smoke-operational` or `make smoke-restart-recovery` |
| I need a quick health snapshot | `make diag` |
| I need service status | `make ps` |
| I need one service log stream | `make logs SERVICE=gateway` |
| I need to restart one runtime service | `SERVICE=gateway make restart` |

## Troubleshooting By Symptom

### `make bootstrap` fails

Check in this order:

1. Install or expose the missing host command named in the failure.
2. Start Docker Desktop or the local Docker daemon if Docker is unreachable.
3. Re-run `make bootstrap`.
4. If the compose render step fails, inspect [`deploy/compose/docker-compose.yaml`](../../deploy/compose/docker-compose.yaml) and the env file at [`deploy/envs/local.env`](../../deploy/envs/local.env).

### `make live` or `make up` does not result in a healthy stack

Run:

```bash
make ps
make logs SERVICE=gateway
make logs SERVICE=ingest
make logs SERVICE=derive
make diag
```

Interpretation:

- if containers are not running, fix compose/runtime startup first;
- if containers are running but `make diag` fails readiness, inspect the affected service logs before restarting anything broader;
- if only one service is unhealthy, prefer `SERVICE=<name> make restart` over restarting the whole stack immediately.

### `make seed` fails

Most likely causes:

- gateway is not ready yet;
- the stack is up but services are still warming;
- the wrong base URL or symbol override was supplied.

Run:

```bash
make diag
make logs SERVICE=gateway
make logs SERVICE=configctl
```

Then retry `make seed` only after gateway and configctl are healthy.

### `make smoke` or another `make smoke*` fails

Check in this order:

1. Did you run the right smoke for the behavior you changed?
2. Is the stack healthy according to `make diag`?
3. Was the correct config seeded (`make seed` vs `make seed-multi`)?
4. Are the relevant services showing progress in `make logs SERVICE=...`?

Useful commands:

```bash
make diag
make logs SERVICE=store
make logs SERVICE=writer
make logs SERVICE=execute
```

Use direct script invocation only when you need a debug-only flag such as
`--wait`.

### `make check` or `make verify` fails

Use:

```bash
make tdd
MODULE=./internal/shared make test
make repo-consistency-check
```

Guidance:

- `make tdd` narrows what needs validation;
- `MODULE=... make test` is the surgical path when a single Go module is failing;
- `make repo-consistency-check` helps separate doc/support-surface drift from code/test failures.

### I need to reset the local runtime cleanly

Run:

```bash
make down
make up
make seed
make smoke
```

Use `make live` instead if you want the repository to own the full recovery
sequence again.

## Escalation Rules

- Prefer `make` targets first.
- Use direct `scripts/*.sh` only for debugging, extra flags, or harness work.
- Use direct `raccoon-cli` only for expert structural/tooling analysis.
- Use raw `docker compose`, `go`, or `cargo` only when debugging below the repository workflow contract.

## Related Documents

- [`developer-workflow-unification.md`](developer-workflow-unification.md)
- [`../../DEVELOPMENT.md`](../../DEVELOPMENT.md)
- [`makefile-targets-reference-and-conventions.md`](makefile-targets-reference-and-conventions.md)
- [`scripts-catalog-and-usage-guide.md`](scripts-catalog-and-usage-guide.md)
