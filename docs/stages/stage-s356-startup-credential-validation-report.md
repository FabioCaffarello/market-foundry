# S356 — Startup Credential Validation and Operational Preflight

> Stage type: Implementation (OF-2)
> Wave: Operational Foundation (S353–S358)
> Status: **Closed**
> Date: 2026-03-22

## Governing Questions

| ID | Question | Answer |
|----|----------|--------|
| OFQ-7 | Do binaries fail fast on missing credentials/preconditions? | **YES** — all 7 runtime binaries run preflight before I/O |
| OFQ-8 | Are preflight error messages actionable? | **YES** — include check name, field, and fix guidance |
| OFQ-9 | Is NATS URL format validated before connection? | **YES** — scheme + host check catches common misconfigurations |
| OFQ-10 | Does writer validate ClickHouse + pipeline config before connecting? | **YES** — already existed, now uses shared preflight framework |

## Summary

S356 delivers a shared preflight validation framework and applies it uniformly across all market-foundry runtime binaries. The S352 Production Readiness Assessment identified startup credential validation as a small, high-value delivery. This stage closes that gap.

**Before S356**: Only the writer binary had explicit Phase 0 validation. Other binaries (gateway, execute, derive, ingest, store, configctl) would fail with opaque connection errors when NATS was misconfigured or disabled.

**After S356**: All 7 runtime binaries run a `RunPreflight()` sequence before any I/O. Common checks (NATS enabled, NATS URL format) are shared. Binary-specific checks (ClickHouse config, pipeline config) compose cleanly into the same sequence.

## Deliverables

### Code

| File | Change | Purpose |
|------|--------|---------|
| `internal/shared/bootstrap/preflight.go` | **New** | Shared preflight framework: `RunPreflight()`, `NATSEnabledCheck`, `NATSURLFormatCheck`, `ValidateNATSURL` |
| `internal/shared/bootstrap/preflight_test.go` | **New** | 13 unit tests covering all preflight checks and URL validation |
| `cmd/gateway/run.go` | Modified | Added Phase 0 preflight (NATS enabled + URL format) |
| `cmd/configctl/run.go` | Modified | Added Phase 0 preflight (NATS enabled + URL format) |
| `cmd/execute/run.go` | Modified | Added Phase 0 preflight (NATS enabled + URL format) |
| `cmd/derive/run.go` | Modified | Added Phase 0 preflight (NATS enabled + URL format) |
| `cmd/ingest/run.go` | Modified | Added Phase 0 preflight (NATS enabled + URL format) |
| `cmd/store/run.go` | Modified | Added Phase 0 preflight (NATS enabled + URL format) |
| `cmd/writer/run.go` | Modified | Migrated ad-hoc Phase 0 to shared `RunPreflight()` framework |

### Documentation

| File | Purpose |
|------|---------|
| `docs/architecture/startup-credential-validation-and-operational-preflight.md` | Architecture: credential flow, binary-specific checks, error message format |
| `docs/architecture/preflight-checks-startup-fail-fast-semantics-and-limitations.md` | Semantics: fail-fast behavior, startup timeline, what is and isn't validated |
| `docs/stages/stage-s356-startup-credential-validation-report.md` | This report |

## Preflight Coverage Matrix

| Binary | NATS enabled | NATS URL format | ClickHouse config | Pipeline config | Venue credentials |
|--------|:---:|:---:|:---:|:---:|:---:|
| gateway | Y | Y | — | — | — |
| configctl | Y | Y | — | — | — |
| derive | Y | Y | — | — | — |
| ingest | Y | Y | — | — | — |
| store | Y | Y | — | — | — |
| execute | Y | Y | — | — | at adapter build |
| writer | Y | Y | Y | Y | — |
| migrate | — | — | — | — | — |

## Tests

```
=== RUN   TestNATSEnabledCheck
=== RUN   TestNATSEnabledCheck/passes_when_enabled              PASS
=== RUN   TestNATSEnabledCheck/fails_when_disabled              PASS
=== RUN   TestNATSURLFormatCheck
=== RUN   TestNATSURLFormatCheck/skips_when_NATS_disabled       PASS
=== RUN   TestNATSURLFormatCheck/passes_valid_nats_URL          PASS
=== RUN   TestNATSURLFormatCheck/fails_on_empty_URL             PASS
=== RUN   TestNATSURLFormatCheck/fails_on_bad_scheme            PASS
=== RUN   TestValidateNATSURL
=== RUN   TestValidateNATSURL/valid_nats                        PASS
=== RUN   TestValidateNATSURL/valid_tls                         PASS
=== RUN   TestValidateNATSURL/valid_wss                         PASS
=== RUN   TestValidateNATSURL/empty                             PASS
=== RUN   TestValidateNATSURL/whitespace_only                   PASS
=== RUN   TestValidateNATSURL/http_scheme                       PASS
=== RUN   TestValidateNATSURL/no_host                           PASS
=== RUN   TestValidateNATSURL/no_scheme                         PASS
```

All 13 tests pass. All 7 runtime binaries compile successfully.

## Residual Gaps

| Gap | Severity | Notes |
|-----|----------|-------|
| Venue credentials validated at adapter build, not preflight | LOW | Correct for now — credential requirements depend on venue type |
| No aggregate error report (first-fail exit) | LOW | Operators may need multiple fix-restart cycles for compound misconfiguration |
| No runtime credential re-validation | LOW | Would require process restart on credential rotation |
| No secret store integration | OUT OF SCOPE | Not a goal of this stage or this wave |

## Preparation for S357

With S356 closed, the operational foundation wave has:
- S353: Charter and scope freeze
- S354: Metrics and operational signals foundation
- S355: CI smoke integration
- S356: Startup credential validation and operational preflight

Remaining items for the wave:
- **S357**: Operational runbook hardening — validate that the runbook procedures match the actual binary behavior post-S354/S355/S356.
- **S358**: Operational foundation gate — evidence evaluation and wave closure.

S356 provides the fail-fast semantics that S357 runbook procedures can reference. When documenting "how to diagnose a startup failure," the preflight error messages are now the first thing operators will see.
