# market-foundry

Go workspace foundation for cryptocurrency market data processing built
on NATS+JetStream and ClickHouse.

This is **not** a trading application — it is the foundation on which
trading capabilities are built.

## Current state

End-to-end paper-mode pipeline runs against Binance WebSocket data.
Eight binaries plus a Rust CLI for architecture enforcement. For
details, known gaps, and the next concrete step, see
[docs/RESUMPTION.md](docs/RESUMPTION.md).

## What this repository contains

- **configctl** — config lifecycle (create, validate, compile, activate).
- **gateway** — stateless HTTP↔NATS translation.
- **ingest** — exchange WebSocket → observation events.
- **derive** — observation → evidence, signal, decision, strategy, risk, execution.
- **store** — domain events → NATS KV projections + query serving.
- **execute** — controlled execution intake and fill-state handling.
- **writer** — domain events → ClickHouse analytical storage.
- **migrate** — forward-only ClickHouse schema management.
- **raccoon-cli** (Rust) — static architecture enforcement and quality gates.

## Quick start

```bash
make bootstrap      # validate prerequisites (once per machine / env change)
make up             # bring up the stack
make smoke          # canonical end-to-end proof
make down           # tear down the stack
```

For the full daily workflow, smoke selection, and troubleshooting, see
[docs/DEVELOPMENT.md](docs/DEVELOPMENT.md).

## Repository layout

```
cmd/               One directory per binary (8 total)
internal/          domain → application → adapters → actors → interfaces → cmd
tools/raccoon-cli/ Rust CLI for quality enforcement
deploy/            Docker Compose, configs, migrations, NATS config
```

## Where to start

New to the project? Read these in order:

1. **[docs/RESUMPTION.md](docs/RESUMPTION.md)** — current state, known
   gaps, next concrete step.
2. **[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)** — system shape and
   structural principles.
3. **[docs/DEVELOPMENT.md](docs/DEVELOPMENT.md)** — daily workflow.

15 minutes total, every visit.

## Where to read further

| If you want | Go to |
|---|---|
| Current state and known gaps | [docs/RESUMPTION.md](docs/RESUMPTION.md) |
| System architecture | [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) |
| Runtime topology | [docs/RUNTIME.md](docs/RUNTIME.md) |
| HTTP endpoints | [docs/HTTP-API.md](docs/HTTP-API.md) |
| Daily workflow | [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) |
| Contributing & PR rules | [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) |
| Operations guides | [docs/operations/README.md](docs/operations/README.md) |
| Architecture decisions | [docs/decisions/README.md](docs/decisions/README.md) |
| Domain deep dives | [docs/domain/README.md](docs/domain/README.md) |
| Terminology | [docs/GLOSSARY.md](docs/GLOSSARY.md) |
| AI agent instructions | [CLAUDE.md](CLAUDE.md) |

## Contributing

See [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) for PR rules,
authorized expansion protocol, and the "Specifically for AI agents"
section.

## License

This project is licensed under the **PolyForm Noncommercial 1.0.0**.
See [LICENSE](LICENSE) for full terms. Commercial use is not permitted.

For security reports, see [SECURITY.md](SECURITY.md).
