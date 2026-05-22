# Development

Daily development workflow for market-foundry. For higher-level
architecture, see [`ARCHITECTURE.md`](ARCHITECTURE.md). For operational
topology, see [`RUNTIME.md`](RUNTIME.md). For the canonical PR contract
and review checklist, see [`CONTRIBUTING.md`](CONTRIBUTING.md).

This document covers: setup, daily workflow, smoke tests, testing,
stack lifecycle, troubleshooting.

---

## Setup

### Prerequisites

The repository expects these tools available on the host machine:

- **Go 1.25.7** (workspace declares this exact version)
- **Rust toolchain** (stable) — for `raccoon-cli` build
- **Docker** with `docker compose` plugin
- **Make**, **bash**, **curl**, **git**, **python3**

### First-time bootstrap

After cloning the repository, validate that the machine is ready:

```bash
make bootstrap
```

This runs `scripts/bootstrap-check.sh`, which validates:

- Required commands present
- Docker daemon reachable
- `docker compose` configuration renders cleanly
- Canonical repository entrypoints exist

If `make bootstrap` fails, fix the missing tool or path it reports
before continuing.

### Local environment file

The stack reads environment variables from `deploy/envs/local.env`.
This file is gitignored. The first time you bring up the stack, copy
the example:

```bash
cp deploy/envs/local.env.example deploy/envs/local.env
```

For Binance testnet or mainnet credentials (only needed if running
non-paper modes), edit `local.env` and `deploy/configs/execute*.jsonc`.

### Discovery

Two helpers print the canonical surfaces:

```bash
make help        # grouped targets and common variables
make docs        # primary docs for workflows, targets, and tooling
```

---

## Daily workflow

The default change loop is four steps:

```bash
make check       # pre-code guard rail
make tdd         # impact-driven validation guidance
# < implement the smallest correct change >
make verify      # post-change validation
```

Each step has a specific role:

| Step | Purpose |
|---|---|
| `make check` | Repo consistency + fast quality gate. Catches structural drift before you start. |
| `make tdd` | `raccoon-cli` guidance on which tests to add/update for the current change set. |
| `make verify` | Tests across all workspace modules + repo-consistency + fast quality gate. Run before committing. |

For significant or risky changes, escalate to `make check-deep`
(slower, more thorough static analysis profile).

If you changed runtime behavior, also run an appropriate smoke test
(see below) before pushing.

---

## Smoke tests

Smoke tests exercise the system end-to-end with a real stack (compose
up + seed + probes). They are the canonical proof of operational
behavior — slower than unit tests but catch integration issues that
unit tests miss.

### Choosing the right smoke

The repository ships ~23 smoke targets covering specific operational
proofs. The most common day-to-day choices:

| Target | Purpose | When to use |
|---|---|---|
| `make smoke` | Single-symbol baseline operational proof | Default — most changes |
| `make smoke-multi` | Multi-symbol governed slice | After changes that touch multi-symbol behavior |
| `make smoke-analytical` | Analytical write/read path (NATS → writer → ClickHouse → reader) | Changes to writer or ClickHouse reader |
| `make smoke-round-trip` | Full persistence round-trip (S317: adapter → NATS → ClickHouse → HTTP) | Changes that span multiple layers |
| `make smoke-composed` | Stackless composed pipeline (S330) | Pipeline changes when you don't need full compose |
| `make smoke-live-stack` | Live stack: venue path + persistence + composite + kill-switch (S335) | Larger end-to-end changes |
| `make smoke-operational` | OS-process / container operational behavior | Changes to container lifecycle, signals, supervision |
| `make smoke-restart-recovery` | Restart/recovery resilience | Changes to durable consumers, KV, or supervisor restart logic |
| `make smoke-help` | Print the full smoke menu, prerequisites, and troubleshooting | Discovery |

For the remaining stage-tagged smokes (S372, S374, S378, S380, S394,
S397, S402, S405, S412, S416, S419, S435, S440), use `make smoke-help`
to see the full list with descriptions and prerequisites.

Choose the **narrowest** smoke that proves the behavior you changed.
`make live*` ergonomic wrappers do not replace `make smoke*` as
proof-of-record.

### Running a smoke test

```bash
make up                  # start the full stack
make seed                # seed configctl with single symbol
make smoke               # baseline E2E proof
make down                # tear down
```

Or, for the fastest single-symbol bring-up:

```bash
make live                # build + up + seed + validate
```

For multi-symbol equivalents:

```bash
make live-multi          # build + up + seed-multi + validate
make live-check          # validate an already-running single-symbol stack
make live-multi-check    # validate an already-running multi-symbol stack
```

If the smoke fails, look at `make logs` and `make diag` to investigate.

---

## Testing

### Unit and integration tests

```bash
make test                # all workspace modules (alias: make test-unit)
make test-integration    # NATS-backed integration tests (requires NATS at localhost:4222)
make test-clickhouse     # ClickHouse-backed tests (requires CLICKHOUSE_DSN)
make test-behavioral     # charter-protected behavioral scenarios
make test-behavioral-roundtrip  # round-trip serialization scenarios
```

Single-module testing:

```bash
MODULE=./internal/shared make test
```

Or directly with `go test`:

```bash
go test ./internal/domain/execution/...
go test -run TestSpecific ./...
```

### Boot test (gateway route registration)

The gateway has a hermetic boot test that registers all routes in a
fresh httprouter to detect static-vs-wildcard trie conflicts before
they cause a production panic:

```bash
go test ./cmd/gateway/... -run TestGatewayRouteRegistrationDoesNotPanic
```

This test was added after three such conflicts were found in
production. **If you add a new HTTP route, you must also add it to
the test's `routes` slice in `cmd/gateway/boot_test.go`.**

### Code generation tests

```bash
make codegen-check         # generated output matches golden snapshots
make codegen-test          # codegen unit tests
make codegen-integrated    # integrated slices match golden snapshots
make codegen-equivalence   # cross-artifact equivalence wrapper
make codegen-validate-all  # spec validation, including cross-spec uniqueness
make codegen-status        # governance status of codegen families
```

---

## Stack lifecycle

### Bring-up paths

| Path | Command | When |
|---|---|---|
| Fastest official | `make live` | Local development, want one command |
| Multi-symbol fastest | `make live-multi` | Multi-symbol stack in one command |
| Controlled manual | `make up` + `make seed` + `make smoke` | Need to inspect or override per-step |
| Specific config | `docker compose -f docker-compose.yaml -f docker-compose.<variant>.yaml up -d` | Non-default compose variant |

The default compose variant is `deploy/compose/docker-compose.yaml`.
Alternative variants live alongside it for specific scenarios
(mainnet-dry-run, mainnet-live, unified, venue-live). See
[`RUNTIME.md`](RUNTIME.md) → "Deployment modes" for details.

`make up` waits for ClickHouse to become healthy and then applies any
pending migrations (`migrate-up`) before exiting.

### Seed variants

```bash
make seed                 # default single-symbol (Futures)
make seed-multi           # multi-symbol (Futures)
make seed-spot            # Spot single-symbol
make seed-spot-multi      # Spot multi-symbol
make seed-unified         # S400: merged Spot+Futures
make seed-unified-multi   # S400: merged Spot+Futures multi-symbol
```

### Service inspection

```bash
make ps                       # compose service status
make logs                     # stream all logs
make logs SERVICE=gateway     # one service only
make restart                  # restart whole stack
SERVICE=gateway make restart  # restart one service
make diag                     # diagnostic snapshot
```

### Migrations

```bash
make migrate-up         # apply pending ClickHouse migrations
make migrate-status     # show applied and pending
make migrate-validate   # verify checksums of applied migrations
```

### ClickHouse backup/restore

```bash
make ch-backup                                  # backup all tables (or TABLE=<name>)
make ch-backup-list                             # list available backups
make ch-restore BACKUP=mf_20260323_120000       # restore from a backup
make ch-backup-auto                             # automated backup + off-host replication
```

### Tear-down

```bash
make down                # stop services, remove containers
```

Persistent volumes (`market-foundry-nats-data`,
`market-foundry-clickhouse-data`, `market-foundry-clickhouse-logs`)
are **not** removed by `make down`. To wipe state completely:

```bash
docker volume rm market-foundry-nats-data market-foundry-clickhouse-data market-foundry-clickhouse-logs
```

This loses all NATS streams, KV state, and ClickHouse history.

---

## Troubleshooting

### First-line diagnostic

```bash
make diag                   # diagnostic snapshot
make ps                     # service status
make logs SERVICE=<name>    # focused logs
```

### Common scenarios

**Gateway in CrashLoopBackoff**

Almost certainly a route registration panic. Check
`docker logs <gateway-container>` directly (not via `make logs`,
because the panic happens before the structured logger initializes).
Look for `panic: ... conflicts with existing ...` — this means a new
route conflicts with httprouter's trie. The boot test
(`cmd/gateway/boot_test.go`) should have caught this in CI; if it
did not, the new route was added without updating the test.

**`make verify` fails on consistency check**

`raccoon-cli` runs lightweight repository checks (cross-references,
naming, doc topology). Most failures are:

- Broken Markdown link in a doc that was moved
- Reference to an old path (e.g., `docs/architecture/...` when the
  doc has been retired and consolidated into a root-level doc)
- Stage report not indexed

Read the actual error message — `raccoon-cli` is usually specific.

**Conditional endpoint returns 404 when it shouldn't**

Almost every gateway endpoint is conditionally registered, gated on
whether its backing dependency is wired in the gateway composition
root. Check [`HTTP-API.md`](HTTP-API.md) → "Conditional endpoints
summary" to see which dep gates the endpoint. If the dep is not
configured, the endpoint is silently absent. This is by design — it
allows gateway to start with partial dependencies.

**Smoke test hangs or times out**

Likely a dependency healthcheck not transitioning to `healthy`. Run
`make ps` to see which service is `Restarting` or `Starting`. Then
`make logs SERVICE=<that-one>` to investigate. Common causes:

- ClickHouse cold start (can take 30-60s on first up)
- NATS JetStream not ready (rare; usually <10s)
- Service blocked on configctl readiness when configctl itself failed

**`make verify` shows .opencode/ cross-reference failures**

These are a known carry-over from the documentation reset (P1A) and
will be resolved when `.opencode/` is removed in P1B. They do not
indicate code problems.

---

## Architecture enforcement

The Rust CLI `raccoon-cli` enforces architecture rules automatically.
The make surface:

```bash
make arch-guard            # layer boundary violations
make drift-detect          # cross-layer semantic drift
make quality-gate          # fast quality gate profile
make quality-gate-deep     # deeper, slower analysis
make quality-gate-ci       # CI profile with JSON output
make snapshot              # JSON code-intelligence snapshot
make snapshot-diff SNAP1=... SNAP2=...   # compare two snapshots
make baseline-drift BASELINE=...         # drift vs a baseline snapshot
make raccoon-build         # build the raccoon-cli release binary
make raccoon-test          # run raccoon-cli tests
```

`make check` runs `arch-guard` and the fast `quality-gate` as part of
its repo-consistency profile. `make verify` runs the same plus the Go
test suite. Direct invocation of these targets is for expert
inspection or for adding new checks — see `tools/raccoon-cli/README.md`
for the underlying command surface.

---

## CI-oriented composites

These wrappers exist for CI and pre-push validation:

```bash
make ci-smoke           # CI-safe smoke suite (stackless only)
make ci-preflight       # tests + consistency + quality gate + stackless smoke
make ci-analytical      # tests + smoke-analytical
make ci-wait-ready      # poll until ClickHouse+gateway are ready (used by stack-dependent CI smokes)
```

`ci-preflight` is the canonical pre-push check; passing it locally is
the closest equivalent to a green CI before pushing.

---

## Module scoping

The Go workspace has multiple modules; `make tidy` and `make test`
operate across all of them. To work in one:

```bash
cd internal/domain/execution
go test ./...
go build ./...
```

Or from the root with the `MODULE` variable:

```bash
MODULE=./internal/domain/execution make test
SERVICE=gateway make build
SERVICE=gateway make logs
```

---

## Contributing changes

Short version:

1. Make sure `make verify` passes locally (or `make ci-preflight` for
   a closer simulation of CI).
2. Open a PR with a clear description of intent and scope.
3. The PR template (when configured in P1D) will list the checklist
   for review.

Full PR rules, branch conventions, and review checklist live in
[`CONTRIBUTING.md`](CONTRIBUTING.md).

---

## Reading further

| If you want | Go to |
|---|---|
| System architecture | [`ARCHITECTURE.md`](ARCHITECTURE.md) |
| Operational topology | [`RUNTIME.md`](RUNTIME.md) |
| HTTP endpoints | [`HTTP-API.md`](HTTP-API.md) |
| Current state and gaps | [`RESUMPTION.md`](RESUMPTION.md) |
| PR rules | [`CONTRIBUTING.md`](CONTRIBUTING.md) |
| Domain deep dives | [`domain/`](domain/README.md) |
| Operational guides | [`operations/`](operations/README.md) |
