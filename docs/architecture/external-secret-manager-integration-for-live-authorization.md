# External Secret Manager Integration for Live Authorization

**Stage:** S439
**Status:** Implemented
**Scope:** Config-driven credential provider selection, file-based secret manager integration, fail-closed bootstrap

---

## Summary

S439 closes the external secret manager gap identified during the S438 live trading authorization ceremony preparation. The `CredentialProvider` interface (introduced in S434) now has a second canonical implementation — `FileCredentialProvider` — that reads credentials from mounted secret files. Config-driven provider selection allows the execute binary to switch between env vars and file-based secrets without code changes.

## Architecture

### Provider Implementations

| Provider | Backend | Config Value | Status |
|----------|---------|-------------|--------|
| `EnvCredentialProvider` | `os.Getenv("MF_VENUE_{TYPE}_{KEY}")` | `"env"` (default) | S434 |
| `FileCredentialProvider` | File at `{credential_path}/{venue_type}/{KEY}` | `"file"` | S439 |

### FileCredentialProvider

**Location:** `internal/application/execution/file_credential_provider.go`

**File layout convention:**

```
{credential_path}/
  binance_spot_mainnet/
    API_KEY
    API_SECRET
  binance_futures_mainnet/
    API_KEY
    API_SECRET
```

**Compatibility matrix:**

| Secret Manager | Mount Mechanism | Compatible |
|---------------|-----------------|------------|
| HashiCorp Vault | Agent sidecar / CSI driver | Yes |
| AWS Secrets Manager | External Secrets Operator | Yes |
| Docker secrets | `/run/secrets` mount | Yes |
| Kubernetes secrets | Projected volume | Yes |
| Manual files | Any mount | Yes |

**Resolution:**

```
Resolve("binance_spot_mainnet", "API_KEY")
  -> ReadFile("{credential_path}/binance_spot_mainnet/API_KEY")
  -> TrimSpace(contents)
  -> return value (or "" on any error)
```

### Config-Driven Provider Selection

**New config fields in `VenueConfig`:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `credential_provider` | string | `"env"` | Backend selector: `"env"` or `"file"` |
| `credential_path` | string | (none) | Base directory for `"file"` provider |

**Example config (file provider):**

```jsonc
{
  "venue": {
    "credential_provider": "file",
    "credential_path": "/run/secrets/venue",
    "dry_run": true,
    "segments": {
      "spot": { "enabled": true, "adapter": "binance_spot_mainnet" },
      "futures": { "enabled": true, "adapter": "binance_futures_mainnet" }
    }
  }
}
```

**Validation rules (fail-closed):**

| Condition | Result |
|-----------|--------|
| `credential_provider` is unknown | Rejected at config validation |
| `credential_provider=file` without `credential_path` | Rejected at config validation |
| `credential_path` set without `credential_provider=file` | Rejected at config validation |
| `credential_provider` omitted | Defaults to `"env"` |

### Bootstrap Wiring

**Location:** `cmd/execute/run.go`

```
Config parsed
  |
  v
Phase -1: Wire credential provider from config   <-- S439
  |  "env" -> no-op (already default)
  |  "file" -> NewFileCredentialProvider(path) -> SetCredentialProvider
  |
  v
Phase 0: Preflight
  |  CredentialPathCheck          <-- S439: validates base path exists
  |  MainnetCredentialCheck       <-- S434: validates all secrets resolvable
  |
  v
Phase 1: Adapter bootstrap (LoadCredentials via selected provider)
```

### Credential Resolution Flow (Updated from S434)

```
Config (credential_provider, credential_path, adapter type)
  |
  v
SetCredentialProvider(selected provider)   <-- S439
  |
  v
Preflight: CredentialPathCheck             <-- S439: file path exists?
  |
  v
Preflight: MainnetCredentialCheck          <-- S434: secrets resolvable?
  |
  v
buildVenueAdapterByType
  |
  v
LoadCredentials -> LoadCredentialsFrom(provider, venueType, keys)
  |
  v
provider.Resolve(venueType, key)           <-- dispatches to env or file
  |
  v
Format validation (mainnet only)           <-- S434: length + whitespace
  |
  v
CredentialSet (immutable, never logged)
```

## What Changed from S434

| Aspect | S434 | S439 |
|--------|------|------|
| Providers | EnvCredentialProvider only | + FileCredentialProvider |
| Selection | Hardcoded to env | Config-driven via `credential_provider` |
| Config surface | No provider fields | `credential_provider`, `credential_path` |
| Preflight | MainnetCredentialCheck | + CredentialPathCheck |
| External SM support | Interface only (future) | File-based integration (production-ready) |
| Compose | Env vars only | Can mount secret volumes |

## Security Invariants (Preserved + Extended)

1. **Credential values are never logged, printed, or included in error messages.** (S434, preserved)
2. **Missing mainnet credentials block startup.** (S434, preserved)
3. **Format validation is mainnet-only.** (S434, preserved)
4. **Provider is set once at init.** (S434, preserved)
5. **Fail-closed default.** Omitted `dry_run` defaults to true. (S379, preserved)
6. **File provider trims whitespace.** Handles trailing newlines from mount tooling. (S439)
7. **File provider returns empty on any read error.** No partial data, no exceptions. (S439)
8. **Config validation rejects unknown providers.** No silent fallback. (S439)
9. **Credential path must be a directory.** Preflight rejects files and missing paths. (S439)

## Limitations and Non-Goals

- **No rotation support.** Credentials are loaded once at startup. Rotation requires restart. (Unchanged from S434)
- **No multi-provider fallback.** Only one provider is active. No try-file-then-env chain.
- **No provider health check beyond path existence.** File provider does not verify files are readable at preflight — only that the directory exists. Actual file read errors are caught at credential load time.
- **No Vault API client.** The file provider works with Vault Agent sidecar or CSI driver, not the Vault HTTP API directly. A native Vault client would be a separate provider implementation.
- **No AWS SDK integration.** AWS Secrets Manager integration is via External Secrets Operator file sync, not the AWS SDK.
- **No encryption at rest in file provider.** The files are expected to be in a tmpfs or restricted-permission mount. The provider reads plaintext.
- **No secret rotation ceremony.** This stage provides the external manager surface; operational rotation procedures are out of scope.
- **No live trading authorization.** dry_run=true enforcement for mainnet remains in place. This stage closes the secret manager gap; the live authorization ceremony is a separate stage.
