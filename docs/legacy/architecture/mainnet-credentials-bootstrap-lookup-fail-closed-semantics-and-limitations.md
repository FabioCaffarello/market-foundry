# Mainnet Credentials: Bootstrap, Lookup, Fail-Closed Semantics, and Limitations

**Stage:** S434
**Status:** Implemented
**Scope:** How mainnet credentials are resolved, validated, and enforced at bootstrap

---

## Bootstrap Sequence

The execute binary validates mainnet credentials in two phases:

### Phase 0: Preflight (fail-fast, before I/O)

```
RunPreflight("execute", logger, []PreflightCheck{
    NATSEnabledCheck(config),
    NATSURLFormatCheck(config),
    MainnetCredentialCheck(config, provider.Resolve),  // S434
})
```

`MainnetCredentialCheck` iterates over all enabled segments. For each segment with a mainnet adapter, it calls `provider.Resolve(venueType, "API_KEY")` and `provider.Resolve(venueType, "API_SECRET")`. If any call returns empty, the binary exits immediately with an actionable error:

```
mainnet credential missing: segment=spot adapter=binance_spot_mainnet key=API_KEY
  -- set via secret manager or environment
```

This check is a **no-op** for testnet and paper_simulator configurations.

### Phase 1: Adapter Bootstrap (credential loading + format validation)

```
LoadCredentials(venueType, []string{"API_KEY", "API_SECRET"})
  -> LoadCredentialsFrom(defaultProvider, venueType, keys)
    -> provider.Resolve(venueType, key) for each key
    -> validateCredentialFormats(venueType, creds)  // mainnet only
```

If preflight passed, adapter bootstrap should succeed. The double-check exists because:
1. Preflight checks all segments; adapter bootstrap loads one at a time
2. Format validation (length, whitespace) runs only at load time, not preflight
3. A race-free provider (env vars don't change) makes this redundant, but a remote provider could return different values between preflight and load

## Lookup Semantics

### CredentialProvider Contract

```go
type CredentialProvider interface {
    Resolve(venueType, key string) string  // "" = not found
    Name() string                          // for audit logging
}
```

**Resolution key format:** The provider receives the raw venue type string (e.g., `"binance_spot_mainnet"`) and key name (e.g., `"API_KEY"`). How these map to the backend is the provider's responsibility.

**EnvCredentialProvider mapping:**
```
Resolve("binance_spot_mainnet", "API_KEY")
  -> os.Getenv("MF_VENUE_BINANCE_SPOT_MAINNET_API_KEY")
```

### Required Keys by Adapter

| Adapter | Required Keys |
|---------|---------------|
| `binance_spot_mainnet` | `API_KEY`, `API_SECRET` |
| `binance_futures_mainnet` | `API_KEY`, `API_SECRET` |
| `binance_spot_testnet` | `API_KEY`, `API_SECRET` |
| `binance_futures_testnet` | `API_KEY`, `API_SECRET` |
| `paper_simulator` | (none) |

## Fail-Closed Semantics

### Layer 1: Config Validation (S433, preserved)

- `dry_run=false` with any mainnet adapter -> **rejected at config parse time**
- Binary refuses to start

### Layer 2: Preflight Check (S434, new)

- Any mainnet adapter with missing credentials -> **rejected before I/O**
- Error message names segment, adapter, and key (never the value)

### Layer 3: Credential Loading (S434, enhanced)

- Missing credential -> `Problem{Code: VAL_INVALID_ARGUMENT}`
- Truncated credential (mainnet, <16 chars) -> `Problem{Code: VAL_INVALID_ARGUMENT}`
- Whitespace in credential (mainnet) -> `Problem{Code: VAL_INVALID_ARGUMENT}`
- Provider failure -> treated as missing (empty string)

### Layer 4: DryRunSubmitter (S379, preserved)

- Even if credentials are loaded, `dry_run=true` (the only allowed mainnet mode) wraps all venue calls in DryRunSubmitter
- No real orders reach the exchange

### Layer 5: Kill Switch (EXECUTION_CONTROL KV, preserved)

- Global halt gate via NATS KV
- Can stop execution without process restart

## Failure Modes

| Failure | When Detected | Behavior | User Action |
|---------|---------------|----------|-------------|
| Mainnet cred missing | Preflight | Exit(1) with message | Set env var or configure secret manager |
| Mainnet cred truncated | Adapter bootstrap | Exit(1) with message | Fix credential value |
| Mainnet cred whitespace | Adapter bootstrap | Exit(1) with message | Trim credential value |
| Provider unavailable | Preflight (returns "") | Exit(1) with message | Fix provider connectivity |
| dry_run=false + mainnet | Config validation | Exit(1) with message | Remove dry_run=false or use testnet |

## Audit Trail

At startup, the execute binary logs:

```
INFO venue adapter selected  type=binance_spot_mainnet  credential_provider=env  dry_run=true
INFO activation surface at startup  adapter=venue  credentials=present  dry_run=true
```

The `credential_provider` field identifies which backend resolved the credentials. The `credentials` field confirms presence without revealing values.

## Limitations

1. **Single-provider model.** Only one `CredentialProvider` is active per process. No fallback chain.

2. **No runtime refresh.** Credentials are loaded once at startup. Secret rotation requires restart.

3. **No provider health check.** If the secret manager is down at startup, the binary fails to start. There is no retry or backoff for provider connectivity.

4. **Format validation is heuristic.** The min-length and whitespace checks catch common errors but cannot validate that a credential is actually valid for the target exchange. Only the exchange can authoritatively validate credentials.

5. **Env vars visible to all processes.** The default `EnvCredentialProvider` stores credentials in the process environment, which is readable by other processes with the same UID. External secret managers mitigate this.

6. **No credential isolation between segments.** Spot and futures mainnet credentials are loaded into the same process. A compromised process has access to all loaded credentials.

7. **Compose overlay uses `${VAR:-}` pattern.** This means compose starts even if env vars are unset. The binary's preflight check catches this, but docker-compose logs may show a "starting" message before the exit.
