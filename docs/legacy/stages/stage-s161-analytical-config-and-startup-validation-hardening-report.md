# Stage S161 — Analytical Config and Startup Validation Hardening

**Status:** Complete
**Scope:** Config validation, startup semantics, fail-fast behavior for writer and gateway analytical runtime.
**Constraint:** No analytical functionality expansion. No schema changes. No new dependencies.

## 1. Executive Summary

S161 hardens the analytical layer's configuration validation and startup
semantics. Writer misconfiguration now fails immediately with actionable
messages that name the offending field, aggregate all issues per restart cycle,
and complete all validation before any connection attempt. The gateway now
validates ClickHouse config before connecting, preventing opaque connection
errors from masking simple config problems. The baseline remains unaffected —
ClickHouse is still fully optional for non-analytical services.

## 2. Hardening Applied

### 2.1 ClickHouseConfig.ValidateForWriter()

New method that enforces writer-specific invariants:
- `addr` must not be empty (writer requires ClickHouse).
- `database` must not be empty.
- `username` must not be empty (ClickHouse requires auth credentials).
- All batching fields validated (batch_size, max_pending, max_retries ≥ 0; durations parseable).
- All issues aggregated into a single structured error.

The generic `Validate()` remains unchanged (returns nil on empty addr) for
gateway and other binaries.

### 2.2 PipelineConfig.ValidateForWriter()

New method that extends standard pipeline validation:
- Runs full `ValidatePipeline()` (known families, duplicates, cross-layer dependencies).
- Additionally requires at least one family configured across any layer.
- Empty pipeline config → hard exit with clear message.

### 2.3 Writer Startup Consolidation

`cmd/writer/run.go` now follows a strict three-phase pattern:

**Phase 0 — Validate config (no I/O):**
1. Check NATS enabled.
2. `ClickHouseConfig.ValidateForWriter()` — replaces the ad-hoc addr check + separate `Validate()` call.
3. `PipelineConfig.ValidateForWriter()` — replaces the separate `ValidatePipeline()` call.
4. Log validated config summary (effective values for all batching params).

**Phase 1 — Open connections:**
5. Open ClickHouse (with addr in error message on failure).
6. Create actor engine.
7. Build health trackers.

**Phase 2 — Run:**
8. Spawn supervisor, start health server, wait for shutdown.

### 2.4 Gateway Analytical Config Validation

`buildAnalyticalClient()` now calls `Validate()` before `Open()`:
- Invalid config → log warning with problem details, return nil (analytical disabled).
- Connection failure → log warning with addr, return nil.
- Success → log with addr and database.

### 2.5 Improved Error Messages

All writer startup errors now use the prefix `writer startup blocked:` for
consistent log filtering. Error messages name the specific field and explain what
is expected, not just that something is wrong.

## 3. Files Changed

| File | Change |
|------|--------|
| `internal/shared/settings/schema.go` | Added `ValidateForWriter()` for ClickHouseConfig and PipelineConfig; refactored `validateFields()` and `validateBatchingFields()` internal helpers |
| `internal/shared/settings/settings_test.go` | 11 new tests covering writer-specific validation |
| `cmd/writer/run.go` | Consolidated validation with fail-fast phasing and config summary log |
| `cmd/gateway/compose.go` | Added ClickHouse config validation before connection; improved log attributes |
| `docs/architecture/analytical-config-and-startup-validation-hardening.md` | Validation rules and startup sequence reference |
| `docs/architecture/analytical-runtime-activation-rules-and-failure-modes.md` | Activation rules and failure mode catalog |

## 4. Validations and Failure Modes

| ID | Condition | Writer Behavior | Gateway Behavior |
|----|-----------|-----------------|------------------|
| F-01 | Empty clickhouse.addr | Hard exit | Analytical disabled (info log) |
| F-02 | Empty clickhouse.database | Hard exit | Analytical disabled (warn log) |
| F-03 | Empty clickhouse.username | Hard exit | Passes (generic Validate() allows) |
| F-04 | Invalid batching config | Hard exit | Analytical disabled (warn log) |
| F-05 | NATS not enabled | Hard exit | N/A (gateway has separate NATS handling) |
| F-06 | Empty pipeline | Hard exit | N/A (gateway doesn't run writer pipelines) |
| F-07 | Dependency violation | Hard exit | N/A |
| F-08 | Connection failure | Hard exit | Analytical disabled (warn log) |
| F-09 | Runtime degradation | Family degraded, others continue | N/A |

## 5. Test Coverage

11 new tests in `settings_test.go`:

- `TestClickHouseValidateForWriterRejectsEmptyAddr`
- `TestClickHouseValidateForWriterRejectsEmptyDatabase`
- `TestClickHouseValidateForWriterRejectsEmptyUsername`
- `TestClickHouseValidateForWriterAcceptsValidConfig`
- `TestClickHouseValidateForWriterRejectsNegativeBatchSize`
- `TestClickHouseValidateForWriterAggregatesMultipleIssues`
- `TestPipelineValidateForWriterRejectsEmptyConfig`
- `TestPipelineValidateForWriterAcceptsWithEvidenceFamily`
- `TestPipelineValidateForWriterAcceptsWithSignalFamily`
- `TestPipelineValidateForWriterPropagatesStandardValidationErrors`

All 42 settings tests pass. Both `cmd/writer` and `cmd/gateway` compile cleanly.

## 6. Remaining Limits

| Limit | Rationale |
|-------|-----------|
| No schema existence check at startup | Migrations are a separate concern (`cmd/migrate`) |
| No runtime reconnection validation | Delegated to ClickHouse driver |
| No cross-service config coherence | Writer and store configs are independent; validated per-binary |
| `password` not validated for emptiness | Empty password may be intentional in dev/test environments |
| Zero-value batching fields silently default | `batch_size=0` → 1000 is a safe fallback; adding a warning would require a logger in a pure validation method |
| No ClickHouse reachability pre-check | Validated by `Open()` and readiness probe, not config validation |

## 7. Preparation for S162

S161 satisfies the third Wave B precondition (writer config validation at startup)
identified in S156. The analytical layer is now ready for the pre-Wave-B gate:

- **Config validation:** Writer fails fast on every known invariant.
- **Startup semantics:** Phased, predictable, observable.
- **Failure modes:** Cataloged with resolution steps.
- **Optionality:** Intact — ClickHouse remains optional for the baseline.

Recommended S162 focus areas:
1. **Pre-Wave-B readiness gate ceremony** — formal review of all Wave B preconditions.
2. **Cross-service config coherence validation** (optional) — tooling to verify writer families match store projections.
3. **Schema migration validation at writer startup** (optional) — verify expected tables exist before spawning pipelines.
