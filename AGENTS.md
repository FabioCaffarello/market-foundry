# AGENTS.md — market-foundry

This file establishes the repository-wide operating contract for AI agents working in this codebase.

## Repository Status

**Pre-absorption phase** — sanitized from quality-service origin, prepared for marketmonkey absorption.

## Default Validation Workflow

1. `make check` — pre-code guard rail (quality-gate fast)
2. `make tdd` — impact-driven testing guide
3. Implement the smallest correct change
4. `make verify` — post-change validation (Go tests + quality-gate)
5. Escalate to `make check-deep` for significant changes

## Troubleshooting

```bash
make logs              # Stream service logs
make logs SERVICE=gateway  # Single service logs
make ps                # Show service status
```

## Primary Source Files

Read these first to understand the system:

- `Makefile` — all available targets
- `DEVELOPMENT.md` — development workflow
- `README.md` — project overview
- `docs/architecture/` — architecture decisions and audit records
- `docs/architecture/market-foundry-evolution-playbook.md` — evolution playbook (primary governance artifact)
- `docs/architecture/stage-definition-of-done.md` — what "done" means for a stage
- `docs/architecture/anti-debt-checklist.md` — practical debt prevention checklist
- `docs/architecture/opus-guidance-rules.md` — rules for conducting the Opus
- `docs/tooling/cli-overview.md` — CLI reference

## Architecture Layers

The Go workspace follows strict layering:

```
domain → application → adapters → actors → interfaces → cmd
```

Dependencies flow inward only. The raccoon-cli enforces this via `make arch-guard`.

## Current Services

| Service | Purpose |
|---------|---------|
| configctl | Config lifecycle management (NATS actors) |
| gateway | HTTP API gateway |
| nats | Message bus infrastructure |

## Foundation Components

These are preserved as building blocks for the next phase:

- `internal/shared/` — settings, bootstrap, memdb, problem, envelope, events, requestctx
- `internal/domain/configctl/` — config lifecycle domain
- `internal/actors/common/` — actor engine, lifecycle management
- `internal/adapters/nats/` — NATS connection, request/reply, configctl gateway
- `internal/interfaces/http/` — HTTP webserver, handlers, routing

## Prohibited Patterns

Do not reintroduce:
- Kafka adapters or infrastructure
- Validator/consumer/emulator services
- Quality-service naming or identity
- `.context/` directory structure

See `docs/architecture/prohibited-carryovers.md` for the full list.
