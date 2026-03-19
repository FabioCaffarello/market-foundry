# Real Venue Activation and Secret Handling

> **Stage:** S90
> **Date:** 2026-03-19
> **Status:** Infrastructure implemented, no real venue activated

---

## 1. Credential Delivery Mechanism

### Convention

All venue credentials are delivered via environment variables with the pattern:

```
MF_VENUE_{VENUE_TYPE}_{CREDENTIAL_NAME}
```

Examples:
- `MF_VENUE_BINANCE_API_KEY`
- `MF_VENUE_BINANCE_API_SECRET`

### Loading

```go
creds, prob := execution.LoadCredentials("binance", []string{"API_KEY", "API_SECRET"})
if prob != nil {
    // Fail fast — binary will not start without credentials.
    os.Exit(1)
}
apiKey := creds.Get("API_KEY")
```

### Security Invariants

1. **No logging of values.** `LoadCredentials` error messages list the missing env var *names* but never their *values*.
2. **No config file storage.** Credentials must come from environment variables or `env_file`, never from JSONC configs.
3. **Fail-fast on missing.** If any required credential is absent, the binary exits immediately with a clear error listing which env vars are missing.
4. **Git protection.** `.gitignore` includes `*.env` to prevent accidental commit of real credentials.
5. **Immutable after load.** `CredentialSet` is read-only after construction.

### Docker Compose Integration

```yaml
execute:
  env_file:
    - ../configs/execute.env  # Not committed; copy from execute.env.example
```

Template at `deploy/configs/execute.env.example`:
```
# MF_VENUE_BINANCE_API_KEY=your_api_key_here
# MF_VENUE_BINANCE_API_SECRET=your_api_secret_here
```

## 2. Activation Flow

### Pre-Activation (current state)

1. `venue.type` is `"paper_simulator"` — no credentials needed.
2. `buildVenueAdapter()` returns `PaperVenueAdapter` directly.
3. All credential infrastructure exists but is not exercised.

### Activation Sequence (future)

1. Register new venue type in `knownVenueTypes` (settings schema).
2. Implement `VenuePort` adapter for the exchange.
3. Add case in `buildVenueAdapter()`:
   ```go
   case settings.VenueTypeBinance:
       creds, prob := appexec.LoadCredentials("binance", []string{"API_KEY", "API_SECRET"})
       if prob != nil {
           return nil, fmt.Errorf("binance credential load failed: %s", prob.Message)
       }
       return appexec.NewBinanceVenueAdapter(creds), nil
   ```
4. Create `execute.env` with real credentials.
5. Update `execute.jsonc` to set `venue.type` to the new type.
6. Run activation gate ceremony (AG-1..AG-17).

### Deactivation / Rollback

1. Set kill switch to `halted` via `PUT /execution/control`.
2. Change `venue.type` back to `"paper_simulator"` in `execute.jsonc`.
3. Restart execute service.
4. Remove `execute.env` (optional — paper mode ignores it).

## 3. Kill Switch Authority

The kill switch is the primary emergency control for venue execution:

| Aspect | Detail |
|--------|--------|
| Storage | `EXECUTION_CONTROL` KV bucket, key `"global"` |
| Write authority | Gateway binary (via `PUT /execution/control`) |
| Read authority | Execute binary (VenueAdapterActor checks before every submit) |
| Default | Active (fail-open if KV unavailable) |
| Scope | All execution families in this deployment |

### Halt Procedure

```bash
curl -X PUT http://gateway:8080/execution/control \
  -H 'Content-Type: application/json' \
  -d '{"status":"halted","reason":"emergency halt","updated_by":"operator"}'
```

### Resume Procedure

```bash
curl -X PUT http://gateway:8080/execution/control \
  -H 'Content-Type: application/json' \
  -d '{"status":"active","reason":"resume after verification","updated_by":"operator"}'
```

## 4. Config-Driven Safety Controls

| Control | Config Field | Default | Range |
|---------|-------------|---------|-------|
| Staleness guard | `venue.staleness_max_age` | `120s` | 30s–600s |
| Submit timeout | `venue.submit_timeout` | `10s` | 1s–60s |
| Venue type | `venue.type` | `paper_simulator` | Known types only |

All values are validated at startup via settings schema. Invalid values prevent binary startup.
