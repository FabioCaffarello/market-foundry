# Smoke tests

Smoke tests exercise market-foundry end-to-end with a real compose
stack — they are the canonical operational proof-of-record. Unit
tests cover logic; smokes cover **integration**.

For brief workflow context, see
[`../DEVELOPMENT.md`](../DEVELOPMENT.md) → "Smoke tests". This doc
goes deeper: which smoke to run when, what each verifies, and how
to diagnose failures.

---

## Target overview

The Makefile defines **23 smoke targets** (1 discovery helper +
22 actual smokes; verify with `make smoke-help`). They fall into
three groups:

1. **Daily-use smokes** (9) — what you run for routine validation.
2. **Specialized smokes** (14) — for specific scenarios (venue-live,
   compose variants, endurance, backup, etc.).
3. **Stage-tagged residue** (not target names — see below) — many
   smokes have **stage references in their `##` description** (e.g.,
   `## S317: Full persistence round-trip proof…`). The target names
   themselves are functional; only the descriptions still carry the
   stage numbers. This is a lighter form of D4 surface debt than
   what `make verify` would catch.

---

## Daily-use smokes (9)

These are the smokes you'll actually run:

| Target | Verifies | When to run |
|---|---|---|
| `make smoke` | Canonical baseline operational proof for the single-symbol slice | Default after any change |
| `make smoke-multi` | Canonical broad proof for the governed multi-symbol slice | When touching multi-symbol behavior |
| `make smoke-analytical` | Specialized proof for the analytical write/read path | After changes to writer or ClickHouse reader |
| `make smoke-round-trip` | Full persistence round-trip: adapter → NATS → ClickHouse → HTTP (S317) | After cross-layer changes |
| `make smoke-composed` | Composed pipeline operational smoke (no stack needed) (S330) | After composition wiring changes |
| `make smoke-live-stack` | Live stack: venue path + persistence + composite + kill-switch (S335) | Before pushing larger end-to-end changes |
| `make smoke-operational` | OS-process / container operational behavior | After container lifecycle or supervision changes |
| `make smoke-restart-recovery` | Restart/recovery resilience | After changes to durable consumers, supervisor restart logic, or session lifecycle |
| `make smoke-help` | Print the full smoke menu, prerequisites, troubleshooting | Discovery |

### When to run which

| If you changed... | Run |
|---|---|
| A domain type (signal, decision, etc.) | `smoke` then `smoke-multi` |
| HTTP routes or handlers | `smoke` then confirm `cmd/gateway/boot_test.go` updated |
| ClickHouse schema or analytical reads | `smoke-analytical` |
| Restart or recovery logic | `smoke-restart-recovery` |
| Composed pipeline composition | `smoke-composed` |
| Stack lifecycle scripts | `smoke-operational` |
| Anything material | `smoke` at minimum |

If unsure: start with plain `smoke`. It's the narrowest valid baseline.

---

## Specialized smokes (14)

Use these when their specific scenario is in scope.

### Venue and live data

| Target | Verifies |
|---|---|
| `make smoke-activation` | Activation acceptance transitions + controlled live path + extended observation (S340+S341+S342+S343) |
| `make smoke-live-listening` | Compose live exchange listening — real trades, paper mode (S378) |
| `make smoke-live-dry-run` | End-to-end live-listen + dry-run: live data → dry-run fill → read/explain (S380) |
| `make smoke-spot-ingest` | Spot ingest binding seed and runtime projection (S397) |
| `make smoke-spot-venue-live` | Spot real venue acceptance/fill (unit tests, no compose) (S405) |
| `make smoke-futures-venue-live` | Futures real venue acceptance/fill (unit tests, no compose) (S416) |

### Compose variants and multi-segment

| Target | Verifies |
|---|---|
| `make smoke-compose-wiring` | Compose-level orchestration: boot, streams, consumers, connectivity (S372) |
| `make smoke-segmented-compose` | Segmented Binance — Futures/Spot configs, dry-run, segment isolation (S394) |
| `make smoke-unified-coexistence` | Single-compose coexistence — Spot+Futures unified config, dry-run, isolation (S402) |

### Resilience

| Target | Verifies |
|---|---|
| `make smoke-failure-isolation` | Multi-binary failure isolation — restart one, others survive (S374) |
| `make smoke-endurance-soak` | Endurance soak and persistence hardening (S412). Phases 1-4 stackless; phases 5-8 require `make up && make seed-unified`. |
| `make smoke-runtime-preflight` | Consolidated runtime smoke & Futures preflight, stackless (S419) |

### Backup

| Target | Verifies |
|---|---|
| `make smoke-backup-restore` | ClickHouse backup/restore round-trip (S435). Requires ClickHouse running. |
| `make smoke-backup-offhost` | Automated backup + off-host replication (S440). Requires ClickHouse running. |

These last two are tied to the backup operations — see
[backups.md](backups.md).

---

## Stage references in smoke names — D4 debt nuance

The 23 smoke targets all have **functional names** (e.g.,
`smoke-compose-wiring`, `smoke-failure-isolation`,
`smoke-backup-restore`). None of them are literally named
`smoke-s372` or similar. The stage-tagged debt lives in the **`##`
descriptions** of these targets, where lines like
`## S372: Compose-level orchestration wiring validation` still appear.

This is a milder form of the D4 surface debt described in
[`../RESUMPTION.md`](../RESUMPTION.md). The resolution path is the
same: progressively drop the `Sxxx:` prefix from descriptions when
their stage context is no longer load-bearing.

---

## How a smoke runs

A typical smoke target performs:

1. **Stack bring-up** — `make up` (compose, healthchecks).
2. **Seed** — apply a configctl config (sometimes via `make seed`).
3. **Wait for readiness** — wait until `/readyz` succeeds for all
   services (sometimes via `ci-wait-ready`).
4. **Probes** — make HTTP queries to verify expected data is flowing.
5. **Optional teardown** — `make down`.

The scripts driving smokes live under `scripts/`. Read the relevant
script if you want to know exactly what's checked.

---

## Diagnosing smoke failures

### Step 1: identify the failing assertion

The smoke output names what failed. Look for `expected X but got Y`
or `wait timeout`. The line number in the script tells you where.

### Step 2: check service health

```bash
make ps
```

Any service in `Restarting` or `Unhealthy`? Fix it first.

### Step 3: check service logs

```bash
make logs SERVICE=<the-failing-service>
```

Look for panic messages, error logs, or warnings about missing
dependencies.

### Step 4: narrow the smoke

If `smoke-multi` fails, does `smoke` alone fail? If yes, the problem
is broader. If no, the multi-symbol path is broken specifically.

### Step 5: read the smoke source

Open `scripts/smoke*.sh` (or wherever the smoke is defined) to see
what data shape is expected at which step.

### Step 6: when stuck

```bash
make diag                # diagnostic snapshot
```

`diag` collects logs, configs, and runtime state into a single output.
Useful for sharing with someone helping debug.

---

## Smoke design principles

For consistency when adding new smokes:

1. **Each smoke proves a specific path end-to-end.** Don't mix
   concerns in one smoke.
2. **Smokes assume a fresh stack.** Wipe volumes before if state matters.
3. **Smokes assert on specific data shapes.** Use jq to extract and
   compare, not just exit codes.
4. **Smokes name their target endpoints.** Future readers know what
   the path under test is.
5. **Document the smoke target in Makefile with `##`.** This makes
   it visible in `make smoke-help`.
6. **Prefer functional names over stage-tagged ones.** Future you (and
   future contributors) will thank you. Use the `## Sxxx:` description
   prefix sparingly and only when the stage context is currently
   load-bearing.

---

## Reading further

| If you want | Go to |
|---|---|
| Daily workflow | [`../DEVELOPMENT.md`](../DEVELOPMENT.md) |
| Mode-specific deployment | [deployment.md](deployment.md) |
| Backup smoke targets | [backups.md](backups.md) |
| When a smoke is hanging | [troubleshooting.md](troubleshooting.md) |
| Why the system is structured this way | [`../ARCHITECTURE.md`](../ARCHITECTURE.md) |
| D4 (stage-tagged) surface debt context | [`../RESUMPTION.md`](../RESUMPTION.md) |
