# Venue Credentials and Activation Prerequisites

> Stage S88 — Documents the credential, secret, and activation prerequisites that must be satisfied before any real venue adapter can be instantiated.
> Date: 2026-03-19
> Classification: DESIGN — no credential infrastructure implemented in this stage.

---

## 1. Purpose

Real venue adapters require API keys, secrets, and possibly additional authentication tokens. This document specifies:

- What credential infrastructure is needed.
- How credentials are provided to the execute binary.
- What activation gates must pass before credentials are loaded.
- What operational safeguards protect credential material.

This closes the design gap identified in S86 (HB-POST-4).

---

## 2. Current State

### What Exists

| Component | Status |
|-----------|--------|
| VenueConfig in schema.go | Has `Type` field (only `paper_simulator` allowed) |
| VenuePort interface | `SubmitOrder` only — no auth context |
| Config validation | Rejects unknown venue types |
| PaperVenueAdapter | No credentials needed |
| Kill switch | Operational (EXECUTION_CONTROL KV) |

### What Does Not Exist

| Component | Status |
|-----------|--------|
| Credential storage | Not designed |
| Secret injection | Not designed |
| Per-venue credential isolation | Not designed |
| Credential rotation | Not designed |
| Credential validation at startup | Not designed |
| Audit logging for credential access | Not designed |

---

## 3. Credential Model

### 3.1 Credential Types per Venue

| Venue Type | Required Credentials |
|------------|---------------------|
| paper_simulator | None |
| CEX (e.g., Binance Futures) | API key, API secret, optional passphrase |
| DEX (e.g., on-chain) | Private key or wallet connection (out of scope for first venue) |

### 3.2 Credential Delivery: Environment Variables

**Decision**: Credentials are injected via environment variables, not config files.

**Rationale**:
- Config files (JSONC) are committed to version control — credentials must never be in VCS.
- Environment variables are the standard secret delivery mechanism for containerized services.
- Docker Compose supports `env_file` and `environment` directives.
- Kubernetes supports `Secret` → env var injection.
- No custom secret manager needed for the first venue.

### 3.3 Environment Variable Naming Convention

```
MF_VENUE_{VENUE_TYPE}_{CREDENTIAL_NAME}

Examples:
  MF_VENUE_BINANCE_FUTURES_API_KEY=abc123
  MF_VENUE_BINANCE_FUTURES_API_SECRET=xyz789
  MF_VENUE_BINANCE_FUTURES_PASSPHRASE=optional
```

Prefix `MF_VENUE_` prevents collision with other environment variables. The venue type segment matches the config `venue.type` value in SCREAMING_SNAKE_CASE.

### 3.4 Credential Loading

```go
// CredentialSet holds venue-specific credentials loaded from environment.
type CredentialSet struct {
    APIKey     string
    APISecret  string
    Passphrase string // optional, venue-dependent
}

// LoadCredentials reads credentials from environment for the given venue type.
// Returns an error if required credentials are missing.
func LoadCredentials(venueType VenueType) (CredentialSet, error) {
    prefix := "MF_VENUE_" + strings.ToUpper(strings.ReplaceAll(string(venueType), "-", "_"))

    apiKey := os.Getenv(prefix + "_API_KEY")
    apiSecret := os.Getenv(prefix + "_API_SECRET")
    passphrase := os.Getenv(prefix + "_PASSPHRASE") // optional

    if apiKey == "" || apiSecret == "" {
        return CredentialSet{}, fmt.Errorf(
            "venue %s requires %s_API_KEY and %s_API_SECRET environment variables",
            venueType, prefix, prefix,
        )
    }

    return CredentialSet{
        APIKey:     apiKey,
        APISecret:  apiSecret,
        Passphrase: passphrase,
    }, nil
}
```

**Where this runs**: In `cmd/execute/run.go`, after config loading and venue type validation, before venue adapter construction.

**Fail-fast**: If required credentials are missing, the execute binary refuses to start. This prevents a running execute binary from silently operating without venue access.

---

## 4. Activation Gate Ceremony

### 4.1 Overview

Adding a new venue type to market-foundry is a deliberate governance act, not a code change. The activation gate ceremony is a checklist that must be completed and documented before the new venue type is added to `knownVenueTypes` in `schema.go`.

### 4.2 Pre-Activation Checklist

| # | Gate | Evidence Required |
|---|------|-------------------|
| AG-1 | Venue adapter implements VenuePort interface | Code review: SubmitOrder, GetOrderStatus, CancelOrder |
| AG-2 | Venue adapter has unit tests | Test coverage for success, rejection, timeout, partial fill |
| AG-3 | Venue adapter has integration test with mock venue API | Validates HTTP client behavior, auth headers, error parsing |
| AG-4 | Credential loading for venue type works | Test: LoadCredentials returns correct values from env |
| AG-5 | Config validation accepts new venue type | `knownVenueTypes` updated in schema.go |
| AG-6 | Docker Compose env_file template exists | `deploy/configs/execute.env.example` with placeholder credentials |
| AG-7 | Kill switch verified with real venue adapter | Halt → verify no new orders submitted → resume |
| AG-8 | Staleness guard verified with real venue timing | Confirm 120s default is appropriate for venue's execution speed |
| AG-9 | Fill reconciliation invariants hold | RC-1 through RC-7 validated with real venue responses |
| AG-10 | Drift rules updated | New venue type checked in execution-config-drift |
| AG-11 | Smoke test extended | New venue type has dedicated smoke test scenario |
| AG-12 | Architecture doc written | `docs/architecture/venue-{name}-adapter-design.md` |
| AG-13 | Risk review completed | Documented assessment of venue-specific risks |
| AG-14 | Single-symbol restriction confirmed | First activation limited to one symbol, one timeframe |
| AG-15 | Rollback plan documented | How to revert to paper_simulator if venue fails |
| AG-16 | Monitoring baseline defined | Which /statusz counters trigger alerts |
| AG-17 | Rate limit budget confirmed | Venue API rate limits vs expected polling frequency |

### 4.3 Activation Ceremony Document

When all gates pass, a document is created:

```
docs/architecture/venue-activation-{name}-ceremony.md
```

This document records:
- Date of ceremony
- Who reviewed each gate
- Evidence references (test results, PR links, etc.)
- Conditions and limitations of activation
- Emergency rollback procedure

### 4.4 Post-Activation Monitoring Period

After activation, the system enters a monitoring period:

| Phase | Duration | Constraints |
|-------|----------|------------|
| Shadow | 24h | Real venue adapter runs but orders are not submitted (dry-run mode) |
| Guarded | 72h | Orders submitted with reduced quantity (10% of paper quantity) |
| Operational | Ongoing | Full operational mode, monitoring continues |

**Transition between phases** is manual (operator decision), logged, and requires kill switch to be active before transition.

---

## 5. Credential Security Safeguards

### 5.1 Logging Protection

```
INVARIANT: No credential value (API key, secret, passphrase) may appear in any log output,
structured or unstructured, at any log level.
```

Implementation:
- Credential fields are never passed to logger.
- VenuePort implementations receive `CredentialSet` and must not log its contents.
- Startup log confirms credential *presence* (e.g., "API key loaded: yes"), never *value*.

### 5.2 Memory Protection

- Credentials are loaded once at startup.
- Stored in a `CredentialSet` struct, not in a global variable.
- Passed by value to the venue adapter constructor.
- No credential caching, no credential sharing between actors.

### 5.3 No Credentials in Config Files

```
INVARIANT: deploy/configs/*.jsonc files must never contain credential values.
Credential paths or environment variable names may appear, but not actual secrets.
```

Drift rule enforcement: `raccoon-cli` can check that no `*.jsonc` file contains patterns matching API keys (base64 strings of specific lengths, known key prefixes).

### 5.4 Credential Rotation

For the first venue, credential rotation is manual:

1. Generate new API key at venue.
2. Update environment variable (`MF_VENUE_{TYPE}_API_KEY`).
3. Restart execute binary (rolling restart if multiple instances).
4. Verify new credentials work (health check).
5. Revoke old API key at venue.

**Automated rotation is deferred** — it requires a secret manager (Vault, AWS Secrets Manager, etc.) and is not needed for the first single-venue deployment.

---

## 6. Docker Compose Credential Integration

### 6.1 Environment File Template

```bash
# deploy/configs/execute.env.example
# Copy to deploy/configs/execute.env and fill with real values.
# NEVER commit execute.env to version control.

# Venue credentials (required for real venue, ignored for paper_simulator)
MF_VENUE_BINANCE_FUTURES_API_KEY=
MF_VENUE_BINANCE_FUTURES_API_SECRET=
MF_VENUE_BINANCE_FUTURES_PASSPHRASE=
```

### 6.2 Docker Compose Configuration

```yaml
execute:
  # ... existing config ...
  env_file:
    - ./deploy/configs/execute.env  # only for real venue, not needed for paper
```

### 6.3 Gitignore

```
# Venue credentials
deploy/configs/*.env
!deploy/configs/*.env.example
```

---

## 7. VenueConfig Schema Extension (Design Only)

When the first real venue type is added, `VenueConfig` gains credential-related fields:

```go
type VenueConfig struct {
    Type            VenueType `json:"type"`
    TimeoutSeconds  int       `json:"timeout_seconds,omitempty"`  // default: 10
    FillDelayMs     int       `json:"fill_delay_ms,omitempty"`    // paper only
    TestnetEnabled  bool      `json:"testnet_enabled,omitempty"`  // use venue's testnet API
}
```

**Note**: Credentials are NOT in VenueConfig (they come from environment). The config file specifies *behavior*, not *secrets*.

`TestnetEnabled` is important: the first real venue activation should always target testnet, not production.

---

## 8. Gaps Closed by This Design

| Gap | Before S88 | After S88 |
|-----|-----------|-----------|
| Credential delivery mechanism | Not designed | Environment variables with naming convention |
| Credential loading contract | Not designed | `LoadCredentials()` with fail-fast |
| Activation gate ceremony | Mentioned but not specified | 17-gate checklist with evidence requirements |
| Credential security invariants | Not documented | Logging, memory, config file protection |
| Docker Compose credential integration | Not designed | env_file template + gitignore |
| Post-activation monitoring | Not designed | Three-phase rollout (shadow → guarded → operational) |

---

## 9. What Remains Deferred

| Item | Reason | Earliest Stage |
|------|--------|---------------|
| Credential loading implementation | No real venue adapter yet | S89+ (with activation gate) |
| env.example file creation | No real venue adapter yet | S89+ |
| Automated credential rotation | Requires secret manager | Future |
| Per-venue credential isolation (multi-venue) | Single venue first | Future |
| HSM/Vault integration | Enterprise feature | Future |
| Testnet-to-mainnet migration ceremony | No testnet activation yet | S90+ |
