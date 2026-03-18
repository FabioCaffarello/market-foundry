# CLI Drift Detection Rules

> Defines the specific drift rules enforced by `raccoon-cli drift-detect`.
> Each rule compares two sources of truth and reports discrepancies.

---

## Rule 1: Config-Compose Drift

**Sources compared:** `deploy/configs/*.jsonc` vs `deploy/compose/docker-compose.yaml`

**What it checks:**
- Every service config file has a corresponding compose service
- Every compose service (excluding `nats`) has a corresponding config file
- NATS URLs in configs match the compose service network topology

**Expected alignment:**
| Config File | Compose Service |
|-------------|-----------------|
| `configctl.jsonc` | `configctl` |
| `gateway.jsonc` | `gateway` |
| `ingest.jsonc` | `ingest` |
| `derive.jsonc` | `derive` |
| `store.jsonc` | `store` |

**Common causes of drift:** Adding a new service binary without updating compose, or vice versa.

---

## Rule 2: Binary-Compose Drift

**Sources compared:** `cmd/*/` directories vs `deploy/compose/docker-compose.yaml`

**What it checks:**
- Every `cmd/{name}/` directory has a matching compose service
- No compose service points to a binary that doesn't exist

**Expected binaries:** configctl, gateway, ingest, derive, store

**Common causes of drift:** Renaming a binary without updating compose (e.g., the server→gateway rename).

---

## Rule 3: Naming Identity Drift

**Sources scanned:** All Go source files, config files, compose file, Makefile, scripts

**What it checks:**
- No references to `cmd/server` in active code (should be `cmd/gateway`)
- No references to old service names (`consumer`, `emulator`, `validator`) in active code paths
- No `"server.http"` source identifiers in NATS client construction
- No `actorserver` package references

**Exclusions:**
- `docs/stages/` — historical stage reports are preserved as-is
- `zip/` — archived reference material
- Comments documenting the rename history

**Common causes of drift:** Incomplete rename, copy-paste from old code, or forgotten references in scripts.

---

## Rule 4: Docs-Reality Drift

**Sources compared:** `docs/architecture/runtime-target.md` vs actual `cmd/` directories and compose services

**What it checks:**
- Every binary listed in runtime-target.md exists as a `cmd/` directory
- Every `cmd/` directory is documented in runtime-target.md
- DEVELOPMENT.md service table matches actual services
- Makefile BUILDABLE_SERVICES matches actual `cmd/` directories

**Common causes of drift:** Adding a service without updating architecture docs, or updating docs aspirationally before implementation.

---

## Rule 5: Actor Scope Drift

**Sources compared:** `cmd/*/` directories vs `internal/actors/scopes/*/` directories

**What it checks:**
- Every service binary has a corresponding actor scope directory
- No orphaned actor scope directories exist without a binary

**Expected mapping:**
| Binary | Actor Scope |
|--------|-------------|
| `cmd/configctl` | `internal/actors/scopes/configctl` |
| `cmd/gateway` | `internal/actors/scopes/gateway` |
| `cmd/ingest` | `internal/actors/scopes/ingest` |
| `cmd/derive` | `internal/actors/scopes/derive` |
| `cmd/store` | `internal/actors/scopes/store` |

---

## Rule 6: Stream Registry Drift

**Sources scanned:** Go source files in `internal/adapters/nats/`

**What it checks:**
- JetStream stream names in source match canonical streams: `CONFIGCTL_EVENTS`, `OBSERVATION_EVENTS`, `EVIDENCE_EVENTS`, `SIGNAL_EVENTS`, `DECISION_EVENTS`, `STRATEGY_EVENTS`
- No references to removed streams (`DATA_PLANE_INGESTION`, etc.)
- Durable consumer names follow the pattern `{service}-{stream}` (e.g., `derive-observation`, `store-candle`, `store-signal-rsi`, `store-strategy-mean-reversion-entry`)

**Common causes of drift:** Adding a new stream without following naming conventions, or residual references to old stream names.

---

## Rules 7–11: Signal Domain Drift

Signal-specific drift rules (SD-1 through SD-5) are documented in [cli-signal-drift-rules.md](cli-signal-drift-rules.md).

---

## Rules 12–16: Decision Domain Drift

Decision-specific drift rules (DD-1 through DD-5) are documented in [cli-decision-drift-rules.md](cli-decision-drift-rules.md).

---

## Rules 17–21: Strategy Domain Drift

Strategy-specific drift rules (STD-1 through STD-5) are documented in [cli-strategy-drift-rules.md](cli-strategy-drift-rules.md).

---

## Severity Levels

| Severity | Meaning | Gate Impact |
|----------|---------|-------------|
| Error | Definitive drift that must be fixed | Fails quality-gate |
| Warning | Likely drift that should be investigated | Fails in CI profile only |
| Info | Observation that may indicate drift | Does not affect gate |

---

## Running Drift Detection

```bash
# Quick check
raccoon-cli drift-detect

# Verbose output with all findings
raccoon-cli -v drift-detect

# JSON output for CI
raccoon-cli --json drift-detect
```
