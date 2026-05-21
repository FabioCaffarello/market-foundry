# Live Credentials: Bootstrap, Lookup, Rotation Assumptions, and Fail-Closed Semantics

**Stage:** S439 (extends S434)
**Status:** Implemented
**Scope:** How live credentials are resolved, validated, and enforced with external secret manager support

---

## Bootstrap Sequence

The execute binary validates credentials across three phases:

### Phase -1: Provider Wiring (S439)

```
config.venue.credential_provider
  |
  "env"  -> default (EnvCredentialProvider)
  "file" -> NewFileCredentialProvider(config.venue.credential_path)
            -> SetCredentialProvider(fp)
```

The provider is wired before preflight runs. This ensures all subsequent credential resolution uses the configured backend.

### Phase 0: Preflight (fail-fast, before I/O)

```
RunPreflight("execute", logger, []PreflightCheck{
    NATSEnabledCheck(config),
    NATSURLFormatCheck(config),
    CredentialPathCheck(config),           // S439: file path exists?
    MainnetCredentialCheck(config, ...),   // S434: secrets resolvable?
})
```

**CredentialPathCheck (S439):** When `credential_provider=file`, verifies that `credential_path` exists and is a directory. This catches missing secret mounts before any credential resolution.

**MainnetCredentialCheck (S434, preserved):** For each enabled segment with a mainnet adapter, calls `provider.Resolve` for API_KEY and API_SECRET. If any returns empty, the binary exits with an actionable error.

### Phase 1: Adapter Bootstrap

```
LoadCredentials(venueType, ["API_KEY", "API_SECRET"])
  -> provider.Resolve(venueType, key) for each key
  -> validateCredentialFormats(venueType, creds)  // mainnet only
```

## Lookup Semantics by Provider

### EnvCredentialProvider (default)

```
Resolve("binance_spot_mainnet", "API_KEY")
  -> os.Getenv("MF_VENUE_BINANCE_SPOT_MAINNET_API_KEY")
```

- Uppercase venue type and key
- Returns empty string if env var is not set
- Thread-safe (reads process environment)

### FileCredentialProvider (S439)

```
Resolve("binance_spot_mainnet", "API_KEY")
  -> ReadFile("{credential_path}/binance_spot_mainnet/API_KEY")
  -> TrimSpace(contents)
```

- Lowercase venue type (canonical directory name)
- Uppercase key (file name matches credential key)
- Returns empty string on any read error (file missing, permissions, etc.)
- Trims whitespace (handles trailing newlines from echo, heredoc, mount tooling)
- Thread-safe (stateless file reads)

### Required Keys by Adapter

| Adapter | Required Keys |
|---------|---------------|
| `binance_spot_mainnet` | `API_KEY`, `API_SECRET` |
| `binance_futures_mainnet` | `API_KEY`, `API_SECRET` |
| `binance_spot_testnet` | `API_KEY`, `API_SECRET` |
| `binance_futures_testnet` | `API_KEY`, `API_SECRET` |
| `paper_simulator` | (none) |

## Fail-Closed Semantics

### Layer 0: Config Validation

| Condition | Behavior |
|-----------|----------|
| Unknown `credential_provider` value | Rejected at config parse |
| `credential_provider=file` without `credential_path` | Rejected at config parse |
| `credential_path` without `credential_provider=file` | Rejected at config parse |
| `dry_run=false` with mainnet adapter | Rejected at config parse (S433) |

### Layer 1: Preflight — Credential Path (S439)

| Condition | Behavior |
|-----------|----------|
| `credential_path` directory missing | Exit(1) with actionable message |
| `credential_path` is a file, not directory | Exit(1) with actionable message |
| `credential_path` not accessible | Exit(1) with actionable message |
| Provider is `env` | Check is no-op |

### Layer 2: Preflight — Credential Presence (S434)

| Condition | Behavior |
|-----------|----------|
| Mainnet adapter with missing credential | Exit(1) with actionable message |
| Testnet adapter | Check is no-op |
| Paper simulator | Check is no-op |

### Layer 3: Credential Loading + Format Validation (S434)

| Condition | Behavior |
|-----------|----------|
| Missing credential | Problem{Code: VAL_INVALID_ARGUMENT} |
| Truncated credential (mainnet, <16 chars) | Problem{Code: VAL_INVALID_ARGUMENT} |
| Whitespace in credential (mainnet) | Problem{Code: VAL_INVALID_ARGUMENT} |
| Provider failure | Treated as missing (empty string) |

### Layer 4: DryRunSubmitter (S379)

- Even if credentials are loaded, `dry_run=true` wraps all venue calls
- No real orders reach the exchange

### Layer 5: Kill Switch (EXECUTION_CONTROL KV)

- Global halt gate via NATS KV
- Can stop execution without process restart

## Rotation Assumptions

S439 does not implement credential rotation. The following assumptions apply:

1. **Credentials are loaded once at startup.** Neither provider re-reads after initial load.
2. **Rotation requires process restart.** To pick up new credentials: update the source (env var or file), then restart the binary.
3. **File provider enables zero-downtime rotation via orchestrator.** With Kubernetes or Docker Swarm, the orchestrator can:
   - Update the secret
   - Rolling-restart pods/containers
   - The new process reads the new files at startup
4. **Vault Agent can update files in place.** The binary must be restarted to pick up the change. Vault Agent's `command` template feature can trigger a restart.
5. **No hot-reload is planned.** Hot-reload of credentials introduces complexity (cache invalidation, mid-request credential swap) that is out of proportion to the operational need at this stage.

## Audit Trail

At startup, the execute binary logs:

```
INFO credential provider set to file  path=/run/secrets/venue
INFO execute preflight passed  checks=4
INFO venue adapter selected  type=binance_spot_mainnet  credential_provider=file  dry_run=true
INFO activation surface at startup  adapter=venue  credentials=present  dry_run=true
```

The `credential_provider` field identifies which backend resolved the credentials. The `credentials` field confirms presence without revealing values. The path is logged for the file provider to aid operational troubleshooting.

## Failure Modes

| Failure | When Detected | Behavior | User Action |
|---------|---------------|----------|-------------|
| Unknown provider in config | Config validation | Exit(1) | Fix `credential_provider` value |
| Missing `credential_path` | Config validation | Exit(1) | Set `credential_path` |
| `credential_path` not mounted | Preflight | Exit(1) | Mount secrets directory |
| Secret file missing | Preflight or load | Exit(1) | Create secret file |
| Secret file empty | Credential load | Exit(1) | Write credential value to file |
| Secret truncated (mainnet) | Format validation | Exit(1) | Fix credential value |
| Secret has whitespace (mainnet) | Format validation | Exit(1) | Trim credential value |
| Provider returns stale value | Not detected | Binary uses stale credential | Restart after rotation |

## Limitations

1. **Single-provider model.** Only one provider active per process. No fallback chain.
2. **No runtime refresh.** Credentials loaded once at startup.
3. **No provider health check beyond path existence.** File readability is checked at load time, not preflight.
4. **Format validation is heuristic.** Cannot validate credentials against the exchange — only structural checks.
5. **Env vars visible to all processes.** The env provider stores credentials in plaintext in the process environment.
6. **File permissions are the operator's responsibility.** The file provider does not enforce 0600 permissions — it reads whatever is mounted.
7. **No credential isolation between segments.** Spot and futures credentials exist in the same process.
