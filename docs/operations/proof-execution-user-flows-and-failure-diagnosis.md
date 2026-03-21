# Proof Execution User Flows And Failure Diagnosis

## Purpose

This document describes the intended operator flows for running repository
proofs and diagnosing failures without dropping immediately into substrate-level
commands.

## Primary User Flows

### Flow 1: Smallest valid runtime proof

Use when you changed the baseline single-symbol path.

```bash
make smoke-help
make up
make seed
make smoke
```

Interpretation:

- success means the baseline runtime slice is reachable and queryable
- a null candle can still be acceptable during warm-up
- a gateway health/readiness failure is an environment or startup problem, not a domain regression

### Flow 2: Broad multi-symbol runtime proof

Use when the change touches symbol isolation, broader pipeline coverage, or
execution surface behavior across the governed scenario.

```bash
make smoke-help
make up
make seed-multi
make smoke-multi
```

Interpretation:

- warnings can reflect warm-up or market-condition timing
- failures in cross-symbol isolation are high-signal and should be treated as correctness issues

### Flow 3: Analytical path proof

Use when the change touches writer, ClickHouse, analytical readers, or gateway
analytical endpoints.

```bash
make up
make seed
SMOKE_WAIT=180 make smoke-analytical
```

Interpretation:

- ClickHouse or writer preflight failures are stack/setup blockers
- non-zero ClickHouse rows with zero HTTP rows suggests read/query mismatch
- missing `Server-Timing` is a read-surface observability defect, not a domain defect

### Flow 4: Operational halt/resume proof

Use when the change touches process isolation, control gate behavior, or the
operator-facing execution control surface.

```bash
make up
make seed
make smoke-operational
```

Interpretation:

- service-not-running failures are environmental blockers
- control gate round-trip failures narrow the problem to gateway/control/KV propagation
- analytical endpoint warnings can still be acceptable if the runtime is young

### Flow 5: Restart/recovery proof

Use when the change touches resilience, consumer restart behavior, or
durability/recovery assumptions.

```bash
make up
make seed
SMOKE_WAIT=180 make smoke-restart-recovery
```

Interpretation:

- a pre-flight service failure means recovery cannot be evaluated yet
- count decreases after restart are strong signals
- explicitly documented design limits are not failures unless behavior exceeds those limits

## Failure Diagnosis Ladder

### Step 1: Read the first hard failure

The first hard failure now includes:

- failing endpoint or subsystem
- expected state
- setup reminder
- recommended next commands

Do not skip this and jump straight to arbitrary logs.

### Step 2: Confirm runtime state

```bash
make ps
make diag
```

Use these to separate:

- stack not running
- stack running but not ready
- runtime healthy but proof-specific surface failing

### Step 3: Inspect the nearest service

Use the first failing surface to choose the next log target:

| Failure shape | First log target |
|---|---|
| `/healthz` or `/readyz` failure | `make logs SERVICE=gateway` |
| Baseline runtime data missing | `make logs SERVICE=derive` |
| KV/query surface mismatch | `make logs SERVICE=store` |
| Analytical flush/read issue | `make logs SERVICE=writer` |
| Control gate / execution issue | `make logs SERVICE=execute` |

### Step 4: Re-run with a larger wait budget when the evidence suggests warm-up

Use this only when the failure points to timing rather than correctness:

```bash
SMOKE_WAIT=180 make smoke
SMOKE_WAIT=240 make smoke-analytical
```

Do not use larger waits to ignore deterministic 4xx/5xx failures or service
readiness failures.

## Common Failure Shapes

### Gateway returns `000` or `503`

Likely causes:

- stack not running
- gateway not ready
- upstream substrate unavailable

Start with:

```bash
make ps
make logs SERVICE=gateway
```

### Baseline smoke reaches gateway but no data materializes

Likely causes:

- config not seeded
- derive not active
- pipeline still warming up

Start with:

```bash
make logs SERVICE=derive
make logs SERVICE=store
```

Then retry with a longer wait only if the services look healthy.

### Analytical proof shows rows in ClickHouse but zero HTTP items

Likely causes:

- reader query mismatch
- gateway analytical adapter issue
- query parameters no longer aligned with persisted source/type semantics

Start with:

```bash
make logs SERVICE=gateway
make logs SERVICE=writer
```

### Restart/recovery proof shows counts going backwards

Treat as a real defect candidate.

This shape is higher signal than “no data yet” because it violates a direct
continuity expectation rather than a timing expectation.

## Reading Warnings Correctly

Warnings are acceptable when they reflect known warm-up or market-condition
limits, for example:

- no finalized long-window candle yet
- RSI/EMA family still warming up
- execution/fill path not triggered by current market conditions

Warnings are not equivalent to success when they indicate a blocked preflight or
an unavailable required surface.

## Escalation Boundary

Only move below the repository workflow contract after the documented surfaces
have been used:

1. `make smoke-help`
2. `make smoke*`
3. `make ps`
4. `make diag`
5. `make logs SERVICE=...`

After that, direct script invocation and raw `docker compose` are appropriate
for deeper debugging.
