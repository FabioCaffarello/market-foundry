# Preflight Checks: Startup Fail-Fast Semantics and Limitations

## Overview

Market-foundry binaries implement a two-phase startup model:

1. **Phase 0 — Preflight**: Pure validation of config and credentials. No I/O, no connections, no side effects.
2. **Phase 1+ — Runtime**: Open connections, build actors, start health servers.

This document details the fail-fast semantics, what is and is not validated, and the known limitations of this approach.

## Fail-Fast Semantics

### Exit Behavior

- Preflight checks run sequentially in declaration order.
- The **first** failing check logs an error and calls `os.Exit(1)`.
- No cleanup is needed because Phase 0 has not opened any connections or allocated external resources.
- The error message includes: service name, check name, and the specific failure reason.

### Why Sequential, Not Parallel

Checks are ordered by dependency. For example:
1. `nats-enabled` must pass before `nats-url-format` is meaningful.
2. `clickhouse-config` validates structural correctness before Phase 1 opens a connection.

Running them in parallel would produce confusing cascading errors when the root cause is a single missing field.

### Exit Code

All preflight failures exit with code `1`. There is no distinction between "missing credential" and "invalid format" at the exit code level. The structured log message provides the necessary detail.

## What Preflight Validates

### Config Structural Correctness

| Layer | Checks |
|-------|--------|
| JSON/JSONC parsing | Valid syntax, single document, no unknown fields |
| Type coercion | All duration strings parse correctly |
| Required fields | NATS URL when enabled; ClickHouse addr/database/username for writer |
| Value ranges | Batch sizes non-negative, timeframes in [10, 86400]s |
| Cross-field rules | Pipeline family dependencies (signal→evidence, decision→signal, etc.) |

### Credential Presence

| Credential | Validated at | How |
|------------|-------------|-----|
| NATS URL | Preflight | Format check: scheme + host |
| ClickHouse addr | Preflight (writer only) | Non-empty string |
| ClickHouse username | Preflight (writer only) | Non-empty string |
| Venue API keys | Adapter construction | Env var presence via `LoadCredentials()` |

### URL Format

The NATS URL format check validates:
- Scheme is one of: `nats`, `tls`, `wss`
- Host component is non-empty
- URL is parseable by `net/url.Parse`

This catches common errors like `http://` schemes, missing hosts, and typos — before the NATS client produces an opaque connection error.

## What Preflight Does NOT Validate

| Category | Reason |
|----------|--------|
| Network reachability | NATS/ClickHouse may be down temporarily; readiness probes handle this |
| Authentication validity | Credentials may be revoked/rotated; only presence is checked |
| Schema existence | ClickHouse database/tables may not exist yet; migrate handles this |
| NATS JetStream streams | Streams are created on demand by consumers |
| DNS resolution | Host may be resolvable at startup but not at connection time |
| Port availability | HTTP listen port conflicts are caught at `net.Listen` time |

## Startup Timeline

```
┌──────────────────────────────────────────────────────┐
│ main()                                                │
│  └─ bootstrap.Main(name, Run)                        │
│      ├─ flag.Parse()                                 │
│      ├─ LoadAndValidate(configPath)  ← config checks │
│      │   ├─ settings.Load()                          │
│      │   └─ cfg.Validate()                           │
│      └─ Run(cfg)                                     │
│          ├─ BuildLogger()                            │
│          ├─ RunPreflight()  ← Phase 0                │
│          │   ├─ NATSEnabledCheck                     │
│          │   ├─ NATSURLFormatCheck                   │
│          │   └─ (binary-specific checks)             │
│          ├─ Open connections  ← Phase 1              │
│          ├─ Build actors      ← Phase 2              │
│          └─ WaitTillShutdown                         │
└──────────────────────────────────────────────────────┘
```

## Interaction with Config Validation

Config validation (`AppConfig.Validate()`) runs **before** `Run()` is called, inside `bootstrap.Main()`. It catches:
- Invalid log level/format
- Invalid HTTP timeouts
- NATS URL empty when enabled
- ClickHouse batching param errors
- Pipeline family dependency violations

Preflight adds **binary-specific** checks that only make sense for a particular service:
- Gateway needs NATS but not ClickHouse
- Writer needs both NATS and ClickHouse
- Execute needs NATS but validates venue adapter separately

This separation avoids coupling binary-specific requirements into the shared config schema.

## Limitations and Non-Goals

### Not a Secret Management Platform
The preflight system checks that credentials are present. It does not:
- Encrypt or decrypt credentials
- Integrate with secret stores (Vault, AWS Secrets Manager, etc.)
- Rotate credentials at runtime
- Audit credential access

### Not a Compliance/Security Program
The system does not:
- Enforce password complexity requirements
- Check credential expiration dates
- Provide audit trails for credential usage
- Implement RBAC for credential access

### Single-Shot Validation
Preflight runs once at startup. It does not re-validate credentials during runtime. If a credential becomes invalid after startup (e.g., API key revoked), the system will fail at the point of use, not at a periodic re-validation cycle.

### Known Gaps
- **Venue credentials validated late**: For the execute binary, venue credentials are validated during adapter construction, not during the preflight phase. This is because the credential requirements depend on the venue type, which is config-driven.
- **No aggregate error report**: The fail-fast approach means only the first error is reported. Operators with multiple misconfigurations will need multiple fix-and-restart cycles.
