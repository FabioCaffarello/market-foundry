# Startup Credential Validation and Operational Preflight

## Purpose

This document describes the startup credential validation and operational preflight system implemented across all market-foundry binaries. The system ensures that predictable configuration and credential errors are caught early — before any connection attempts or I/O — with actionable error messages.

## Design Principles

1. **Fail fast**: Missing preconditions cause immediate process exit with a clear message.
2. **Proportional scope**: Only validate what matters for the specific binary.
3. **No secret management**: Credentials are loaded from environment variables; the system validates *presence*, not *authority*.
4. **Actionable errors**: Every preflight failure message names the failing check and explains what to fix.

## Preflight Architecture

### Shared Framework

All preflight checks use the shared `bootstrap.RunPreflight()` function:

```go
bootstrap.RunPreflight("service-name", logger, []bootstrap.PreflightCheck{
    bootstrap.NATSEnabledCheck(config),
    bootstrap.NATSURLFormatCheck(config),
})
```

Checks run sequentially. The first failure logs an error and exits the process. This is intentional — if NATS is not enabled, there is no value in checking the URL format.

### Standard Checks

| Check | What it validates | Used by |
|-------|-------------------|---------|
| `NATSEnabledCheck` | `nats.enabled` is `true` | All runtime binaries |
| `NATSURLFormatCheck` | URL has valid scheme (`nats://`, `tls://`, `wss://`) and non-empty host | All runtime binaries |
| `clickhouse-config` | ClickHouse addr/database/username present + batching params valid | Writer |
| `pipeline-config` | At least one family enabled + pipeline dependency rules satisfied | Writer |

### Binary-Specific Preflight

| Binary | Preflight checks |
|--------|------------------|
| **gateway** | NATS enabled, NATS URL format |
| **configctl** | NATS enabled, NATS URL format |
| **derive** | NATS enabled, NATS URL format |
| **ingest** | NATS enabled, NATS URL format |
| **store** | NATS enabled, NATS URL format |
| **execute** | NATS enabled, NATS URL format |
| **writer** | NATS enabled, NATS URL format, ClickHouse config, pipeline config |
| **migrate** | None (uses direct database connection, not NATS) |

## Credential Validation

### NATS Credentials

NATS URL is the primary credential for inter-service communication:

- **Config path**: `nats.url` (with `NATS_URL` env var fallback in connection layer)
- **Validated at**: Preflight (format check) and config validation (`Validate()`)
- **Format**: Must be `nats://host:port`, `tls://host:port`, or `wss://host:port`

### ClickHouse Credentials

Used only by the writer binary:

- **Config path**: `clickhouse.addr`, `clickhouse.database`, `clickhouse.username`, `clickhouse.password`
- **Validated at**: Writer preflight via `ValidateForWriter()`
- **Required fields**: addr, database, username (password may be empty for some deployments)

### Venue Credentials

Used by the execute binary for non-paper venue adapters:

- **Env var convention**: `MF_VENUE_{VENUE_TYPE}_{CREDENTIAL_NAME}` (e.g., `MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY`)
- **Validated at**: Venue adapter construction in `buildVenueAdapter()`
- **Security invariant**: Values are never logged, printed, or included in error messages

## Error Messages

Preflight errors follow a consistent format:

```
{service} startup blocked: preflight check "{check-name}" failed  check={check-name}  error={message}
```

Examples:
```
gateway startup blocked: preflight check "nats-enabled" failed  check=nats-enabled  error=nats.enabled must be true — this binary requires NATS connectivity
writer startup blocked: preflight check "clickhouse-config" failed  check=clickhouse-config  error=CFG_INVALID: clickhouse config is invalid for writer
```

## Limitations

- **Presence only, not reachability**: Preflight validates configuration presence and format, not that services are actually reachable. Reachability is checked at connection time and via readiness probes.
- **No credential rotation**: The system validates credentials at startup only. Runtime credential rotation requires a process restart.
- **No secret store integration**: Credentials come from config files and environment variables. There is no Vault/KMS/HSM integration.
- **Single-shot exit**: The first failing check terminates the process. Remaining checks are not evaluated, so operators may need multiple fix-and-restart cycles for compound misconfiguration.
