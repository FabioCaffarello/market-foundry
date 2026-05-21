# Mainnet Secret Manager Integration

**Stage:** S434
**Status:** Implemented
**Scope:** Credential resolution abstraction, fail-closed bootstrap, mainnet format validation

---

## Summary

S434 introduces a `CredentialProvider` interface that decouples credential resolution from a specific backend. The default implementation (`EnvCredentialProvider`) reads from environment variables, preserving full backward compatibility. The interface enables future integration with external secret managers (Vault, AWS Secrets Manager, etc.) without changing adapter bootstrap code.

## Architecture

### CredentialProvider Interface

```
CredentialProvider
  Resolve(venueType, key) string   -- returns value or "" if missing
  Name() string                    -- "env", "vault", etc. (for audit logs)
```

**Location:** `internal/application/execution/credentials.go`

**Implementations:**

| Provider | Backend | Status |
|----------|---------|--------|
| `EnvCredentialProvider` | `os.Getenv("MF_VENUE_{TYPE}_{KEY}")` | Canonical default |
| (future) VaultProvider | HashiCorp Vault | Not implemented |
| (future) AWSSecretsProvider | AWS Secrets Manager | Not implemented |

### Credential Resolution Flow

```
Config (adapter type)
  |
  v
Preflight: MainnetCredentialCheck  <-- S434: fails fast if mainnet creds missing
  |
  v
buildVenueAdapterByType
  |
  v
LoadCredentials -> LoadCredentialsFrom(provider, venueType, keys)
  |
  v
provider.Resolve(venueType, key)  <-- pluggable backend
  |
  v
Format validation (mainnet only)  <-- S434: length + whitespace checks
  |
  v
CredentialSet (immutable, never logged)
```

### What Changed from S433

| Aspect | S433 (Before) | S434 (After) |
|--------|---------------|--------------|
| Resolution | Direct `os.Getenv` in `LoadCredentials` | `CredentialProvider.Resolve` via interface |
| Validation | Presence-only (empty = missing) | Presence + format (mainnet: min length, no whitespace) |
| Preflight | None for credentials | `MainnetCredentialCheck` in bootstrap chain |
| Audit | Logged credential state (present/absent) | Logged credential state + provider name |
| Extensibility | Hardcoded to env vars | Pluggable via `SetCredentialProvider` |
| Compose | No mainnet overlay | `docker-compose.mainnet-dry-run.yaml` |

## Security Invariants

1. **Credential values are never logged, printed, or included in error messages.** Format validation reports length and structural issues, never the value itself.

2. **Missing mainnet credentials block startup.** The `MainnetCredentialCheck` preflight runs before any I/O or connection attempt.

3. **Format validation is mainnet-only.** Testnet credentials accept any non-empty value (test environments use short placeholders).

4. **Provider is set once at init.** `SetCredentialProvider` must be called before `Run`. Concurrent mutation is not supported.

5. **Fail-closed default.** If `dry_run` is omitted with mainnet adapters, it defaults to `true`. Config validation rejects `dry_run=false` with mainnet.

## Config and Compose Artifacts

### Mainnet Dry-Run Config

**File:** `deploy/configs/execute-mainnet-dry-run.jsonc`

- Targets mainnet endpoints (api.binance.com, fapi.binance.com)
- Enforces `dry_run: true`
- Requires mainnet credentials via CredentialProvider

### Mainnet Dry-Run Compose Overlay

**File:** `deploy/compose/docker-compose.mainnet-dry-run.yaml`

- Passes mainnet env vars into the execute container
- Uses `${VAR:-}` pattern (compose starts even if vars unset; binary preflight catches missing creds)

## Format Validation Rules (Mainnet Binance)

| Check | Threshold | Rationale |
|-------|-----------|-----------|
| Minimum length | 16 characters | Binance keys are 64-char; anything shorter is truncated or placeholder |
| No whitespace | Rejects `\s\t\n\r` | Common copy-paste error |

## Limitations and Non-Goals

- **No rotation support.** Credentials are loaded once at startup. Rotation requires a process restart.
- **No encryption at rest.** The env var provider stores credentials in plaintext in the process environment. External secret managers should be used for production deployments.
- **No audit trail of access.** The provider interface does not enforce access logging. Future Vault/AWS providers can add this.
- **No multi-provider fallback.** Only one provider is active at a time. Chaining (try Vault, fall back to env) is not implemented.
- **No runtime re-resolution.** Credentials cannot be refreshed without restart.
- **No secret rotation ceremony.** This stage provides the integration surface; operational rotation procedures are out of scope.
