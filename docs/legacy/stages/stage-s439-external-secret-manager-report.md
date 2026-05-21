# Stage S439 — External Secret Manager Integration

**Status:** Complete
**Predecessor:** S438 (Live Trading Authorization Charter)
**Scope:** External secret manager integration for live authorization credentials

---

## Objective

Close the external secret manager gap identified in the S438 live trading authorization preparation. Replace the env-var-only credential surface with a config-driven provider model that supports mounted secret files from external managers (Vault, AWS Secrets Manager, K8s secrets, Docker secrets).

## What Was Done

### 1. FileCredentialProvider Implementation

**File:** `internal/application/execution/file_credential_provider.go`

- New `FileCredentialProvider` that reads credentials from `{basePath}/{venue_type}/{KEY}`
- Whitespace trimming (handles newlines from mount tooling)
- Fail-closed: read errors return empty string (treated as missing)
- `ValidateBasePath()` for preflight verification
- Compatible with Vault Agent, AWS ESO, K8s projected volumes, Docker secrets

### 2. Config-Driven Provider Selection

**File:** `internal/shared/settings/schema.go`

- Added `credential_provider` field to `VenueConfig` (`"env"` default, `"file"`)
- Added `credential_path` field for file provider base directory
- Added `CredentialProviderName()` accessor with default
- Added `validateCredentialProvider()` with fail-closed rules:
  - Unknown provider -> rejected
  - `file` without path -> rejected
  - Path without `file` -> rejected

### 3. Bootstrap Wiring

**File:** `cmd/execute/run.go`

- Phase -1 added: wire credential provider from config before preflight
- `CredentialPathCheck` added to preflight chain (validates file mount)
- Provider selection logged for auditability

### 4. Preflight Check

**File:** `internal/shared/bootstrap/preflight.go`

- `CredentialPathCheck`: validates `credential_path` exists and is a directory
- No-op when provider is not `file`

### 5. Config Reference

**File:** `deploy/configs/execute-mainnet-dry-run.jsonc`

- Updated comments to document S439 credential_provider option

## Artifacts Created

| Artifact | Path |
|----------|------|
| FileCredentialProvider | `internal/application/execution/file_credential_provider.go` |
| Provider tests (11 cases) | `internal/application/execution/s439_external_secret_manager_test.go` |
| Config tests (8 cases) | `internal/shared/settings/s439_credential_provider_config_test.go` |
| Architecture doc | `docs/architecture/external-secret-manager-integration-for-live-authorization.md` |
| Credentials doc | `docs/architecture/live-credentials-bootstrap-lookup-rotation-assumptions-and-fail-closed-semantics.md` |
| Stage report | `docs/stages/stage-s439-external-secret-manager-report.md` |

## Artifacts Modified

| Artifact | Change |
|----------|--------|
| `internal/shared/settings/schema.go` | Added `CredentialProvider`, `CredentialPath` fields + validation |
| `internal/shared/bootstrap/preflight.go` | Added `CredentialPathCheck` |
| `cmd/execute/run.go` | Added provider wiring phase + path preflight |
| `deploy/configs/execute-mainnet-dry-run.jsonc` | Updated credential documentation |

## Test Results

| Suite | Tests | Status |
|-------|-------|--------|
| `internal/application/execution` (S439) | 11 | All pass |
| `internal/shared/settings` (S439) | 8 | All pass |
| `internal/application/execution` (full) | All | No regressions |
| `internal/shared/settings` (full) | All | No regressions |
| `internal/shared/bootstrap` (full) | All | No regressions |
| `cmd/execute` (build) | Compiles | Clean |

## Acceptance Criteria Verification

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Secrets live leave fragile/informal surface | Met | FileCredentialProvider reads from mounted files; config-driven selection |
| Bootstrap and fail-safe are explicit and auditable | Met | CredentialPathCheck + MainnetCredentialCheck in preflight; provider logged |
| Closes a material authorization criterion | Met | External secret manager gap from S438 is closed |
| Base ready for backup/off-host proof in S440 | Met | File provider works with any secret mount mechanism |

## What Remains Out of Scope

- **Credential rotation ceremony** — operational procedures for rotating secrets
- **Vault HTTP API client** — native Vault client (file provider covers Vault Agent)
- **AWS SDK integration** — covered by ESO file sync pattern
- **Multi-provider fallback** — single active provider per process
- **Hot-reload of credentials** — restart required for rotation
- **Live trading authorization** — dry_run=true enforcement preserved
- **Compliance/security program** — this is an integration stage, not a program
