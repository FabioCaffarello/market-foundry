# Command Entrypoints

`cmd/` is the binary-entrypoint layer of `market-foundry`.

Start here when you need to answer one of these questions:

- which binary owns a runtime concern;
- where a service starts, wires dependencies, and loads configuration;
- which binaries are long-running services versus one-shot utilities.

## How To Read This Area

1. Start with the binary directory for the service you care about.
2. Read `main.go` first for process bootstrap and CLI semantics.
3. Read `run.go` or composition files next for dependency wiring.
4. Jump into [`../internal/README.md`](../internal/README.md) once you need the
   layers behind the binary.

## Binary Map

| Directory | Role | Start with | Notes |
|---|---|---|---|
| `configctl/` | Config lifecycle management service | `main.go`, `run.go` | NATS-only operational surface |
| `derive/` | Observation to evidence/signal/decision/strategy/risk/execution derivation | `main.go`, `run.go` | Main downstream event fan-out runtime |
| `execute/` | Execution control service | `main.go`, `run.go` | Controlled action boundary runtime |
| `gateway/` | HTTP API gateway | `main.go`, `run.go`, `gateway.go` | Main HTTP entrypoint plus analytical read wiring |
| `ingest/` | Market data capture service | `main.go`, `run.go` | Exchange WebSocket to observations |
| `migrate/` | ClickHouse migration CLI | `main.go` | One-shot operational utility, not a long-running service |
| `migrate/engine/` | Migration engine package used by `cmd/migrate` | package files | Binary-local helper package; not an independent entrypoint |
| `store/` | Read-model materialization service | `main.go`, `run.go` | NATS KV projection runtime |
| `writer/` | Analytical writer | `main.go`, `run.go`, `pipeline.go` | ClickHouse write path runtime |

## Runtime Entry Rules

- Add a new `cmd/<binary>/README.md` only if that binary develops enough local
  complexity to justify a second-level index.
- Keep binary ownership descriptions short and operational.
- Do not duplicate architecture rationale here; link to
  [`../docs/architecture/README.md`](../docs/architecture/README.md) when the
  question is structural rather than navigational.
- When a new binary is introduced, update this file, [`../README.md`](../README.md),
  and [`../deploy/README.md`](../deploy/README.md) if runtime assets are added.
