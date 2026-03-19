# AGENTS.md — market-foundry

This file establishes the repository-wide operating contract for AI agents working in this codebase.

## Repository Status

**Post first-slice phase** — sanitized, first vertical slice complete, architectural recentralization done. Prepared for next vertical slice expansion.

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
| gateway | HTTP API gateway (HTTP↔NATS translator) |
| ingest | Market data capture: exchange WebSocket → observation events |
| derive | Evidence derivation: observations → candles, volumes |
| store | Read model materialization: NATS KV projections |
| execute | Execution control service |
| nats | Message bus infrastructure (external) |

## Prohibited Patterns

Do not reintroduce:
- Kafka adapters or infrastructure
- Old quality-service binaries (validator, consumer, emulator)
- Quality-service naming or identity
- `.context/` directory structure

See `docs/architecture/prohibited-carryovers.md` for the full list.
