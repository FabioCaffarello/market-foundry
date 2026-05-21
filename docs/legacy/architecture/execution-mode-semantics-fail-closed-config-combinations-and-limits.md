# Execution Mode Semantics, Fail-Closed Config Combinations, and Limits

## Companion documents

- [Exchange ingress contracts and runtime mode model](exchange-ingress-contracts-and-runtime-mode-model.md)
- [Wave charter and scope freeze](exchange-listening-and-dry-run-foundation-wave-charter-and-scope-freeze.md)
- [S377 stage report](../stages/stage-s377-exchange-ingress-contracts-and-runtime-mode-report.md)

---

## 1. Purpose

This document specifies:

1. The semantics of each `EffectiveMode` value — what happens in each mode.
2. The fail-closed properties of the activation surface — how the system
   prevents accidental live venue interaction.
3. Every valid and invalid configuration combination and its safety verdict.
4. The limits of the current model — what it does and does not cover.

---

## 2. Execution mode semantics

### 2.1 Mode: `paper`

| Property | Value |
|----------|-------|
| EffectiveMode | `paper` |
| Trigger | `AdapterState = paper` (any gate, any credentials) |
| Adapter | `PaperVenueAdapter` |
| Venue interaction | None — orders simulated locally |
| Fill source | Synthetic (instant fill, `Simulated: true`) |
| Output stream | `EXECUTION_FILL_EVENTS` (venue family fill events) |
| Credential requirement | None |
| Gate relevance | Ignored — paper mode always processes intents |

**Semantics:** All execution intents are routed to the paper venue adapter,
which simulates instant fills with synthetic prices. No HTTP call, no
WebSocket, no external system contact. The `Simulated: true` flag is set on
all fill records.

### 2.2 Mode: `venue_halted`

| Property | Value |
|----------|-------|
| EffectiveMode | `venue_halted` |
| Trigger | `AdapterState = venue` AND `GateStatus = halted` |
| Adapter | Real venue adapter loaded but blocked |
| Venue interaction | None — safety gate rejects all intents |
| Fill source | None (intents are dropped with reason `"kill_switch"`) |
| Output stream | No fill events produced |
| Credential requirement | Depends on adapter (may or may not be present) |
| Gate relevance | The gate IS the blocking mechanism |

**Semantics:** The venue adapter is loaded at startup (binary has the
capability to reach the venue), but the runtime gate blocks every intent
before it reaches the adapter. This mode exists for operational control —
the operator can halt execution without restarting the binary.

### 2.3 Mode: `venue_degraded`

| Property | Value |
|----------|-------|
| EffectiveMode | `venue_degraded` |
| Trigger | `AdapterState = venue` AND `GateStatus = active` AND `CredentialState = absent` |
| Adapter | Real venue adapter loaded but will fail on submission |
| Venue interaction | Attempted but will fail (no credentials) |
| Fill source | Error responses |
| Credential requirement | Missing — this is a misconfiguration state |
| Gate relevance | Gate is active but credentials are absent |

**Semantics:** This is a degraded state that should not occur in production.
The binary configured `venue.type` to a real venue but did not provide
credentials. The adapter is loaded, the gate is active, but any submission
will fail at the HTTP authentication layer. The binary should ideally reject
this configuration at startup (future hardening). Currently, it proceeds and
fails per-request.

### 2.4 Mode: `venue_live`

| Property | Value |
|----------|-------|
| EffectiveMode | `venue_live` |
| Trigger | `AdapterState = venue` AND `GateStatus = active` AND `CredentialState = present` |
| Adapter | Real venue adapter (e.g., `BinanceFuturesTestnetAdapter`) |
| Venue interaction | Real HTTP calls to exchange API |
| Fill source | Exchange API responses |
| Output stream | `EXECUTION_FILL_EVENTS` |
| Credential requirement | Present and valid |
| Gate relevance | Gate is active — allows execution |

**Semantics:** This is the only mode that produces real venue orders. All
three conditions must be simultaneously satisfied: venue adapter configured,
gate active, credentials present. This is the maximum-privilege mode.

---

## 3. Fail-closed semantics

### 3.1 Fail-closed definition

The activation model is **fail-closed**: in the absence of explicit
configuration or in any ambiguous state, the system defaults to the safest
mode (paper) or blocks execution entirely.

### 3.2 Fail-closed properties

| Property | Mechanism | Evidence |
|----------|-----------|----------|
| **FC-1: Default is paper** | Empty `venue.type` → `paper_simulator` | `cmd/execute/run.go: buildVenueAdapter()` default case |
| **FC-2: Paper dominates** | `AdapterState = paper` → `paper` regardless of gate/credentials | `ComputeEffectiveMode()` first branch |
| **FC-3: Gate blocks venue** | `GateStatus = halted` → `venue_halted` regardless of credentials | `ComputeEffectiveMode()` second branch |
| **FC-4: Credentials gate venue_live** | Missing credentials → `venue_degraded`, not `venue_live` | `ComputeEffectiveMode()` third branch |
| **FC-5: Kill switch is runtime-fail-safe** | Gate check timeout (2s) → `IsHalted` returns true on error | `SafetyGate.Check()` timeout handling |
| **FC-6: Stale intents rejected** | Intent timestamp > `staleness_max_age` → rejected | `StalenessGuard.IsStale()` |
| **FC-7: Unknown venue type rejected** | Unknown `venue.type` → config validation error → binary exits | `VenueConfig.Validate()` |

### 3.3 Fail-closed evaluation order

The system evaluates safety in a strict priority chain:

```
1. Is AdapterState == paper?          → paper (safe, done)
2. Is GateStatus == halted?           → venue_halted (safe, done)
3. Is CredentialState == absent?      → venue_degraded (degraded, done)
4. All three conditions met           → venue_live (real orders)
```

Each check is a **hard gate**. There is no fallthrough, no partial state, no
"almost live." The mode is one of exactly four values, and only one of those
values (`venue_live`) produces real orders.

---

## 4. Configuration combinations matrix

### 4.1 Complete configuration space

The following table exhausts all valid input combinations and their outcomes.

| # | `venue.type` | Env credentials | Gate status | AdapterState | CredentialState | EffectiveMode | Safe? | Notes |
|---|-------------|----------------|------------|-------------|-----------------|---------------|-------|-------|
| 1 | `""` (absent) | absent | active | paper | absent | `paper` | YES | Default deployment |
| 2 | `""` (absent) | absent | halted | paper | absent | `paper` | YES | Gate irrelevant in paper |
| 3 | `""` (absent) | present | active | paper | absent | `paper` | YES | Credentials ignored in paper |
| 4 | `""` (absent) | present | halted | paper | absent | `paper` | YES | Both ignored in paper |
| 5 | `paper_simulator` | absent | active | paper | absent | `paper` | YES | Explicit paper |
| 6 | `paper_simulator` | absent | halted | paper | absent | `paper` | YES | Gate irrelevant |
| 7 | `paper_simulator` | present | active | paper | absent | `paper` | YES | Credentials ignored |
| 8 | `paper_simulator` | present | halted | paper | absent | `paper` | YES | All overrides ignored |
| 9 | `binance_futures_testnet` | absent | active | venue | absent | `venue_degraded` | DEGRADED | Misconfiguration — no creds |
| 10 | `binance_futures_testnet` | absent | halted | venue | absent | `venue_halted` | YES | Gate protects even without creds |
| 11 | `binance_futures_testnet` | present | halted | venue | present | `venue_halted` | YES | Gate is the kill switch |
| 12 | `binance_futures_testnet` | present | active | venue | present | **`venue_live`** | **LIVE** | All conditions met — real orders |

### 4.2 Safety verdicts

| Verdict | Combinations | Description |
|---------|-------------|-------------|
| **SAFE** | 1–8, 10, 11 | No real venue interaction possible |
| **DEGRADED** | 9 | Venue configured but credentials missing — submissions will fail |
| **LIVE** | 12 | Real venue orders — requires all three conditions |

### 4.3 Configuration for S377–S380 (this wave)

The compose stack during this wave uses:

```jsonc
// deploy/configs/execute.jsonc
{
  "venue": {
    "type": "paper_simulator"
  }
}
```

No `MF_VENUE_*` environment variables are set. This maps to combination **#5**:
`paper` mode, unconditionally safe.

### 4.4 Ambiguous or dangerous configuration patterns

| Pattern | Risk | Mitigation |
|---------|------|-----------|
| `venue.type` set to real venue without credentials | MEDIUM | Binary starts in `venue_degraded` — submissions fail per-request. Future: reject at startup |
| `venue.type` set to real venue with credentials but gate halted | LOW | `venue_halted` — safe. Operator must explicitly activate gate |
| `venue.type` set to unknown string | NONE | `VenueConfig.Validate()` rejects at startup |
| Empty config file | NONE | Defaults to `paper_simulator` |
| `venue.type = "paper_simulator"` with real credentials in env | NONE | Paper adapter ignores credentials entirely |

---

## 5. Ownership matrix

| Component | Owner binary | Config source | Mutability |
|-----------|-------------|--------------|-----------|
| WebSocket connection | `ingest` | configctl bindings | Runtime (binding watch) |
| Trade normalization | `ingest` | None (hardcoded per exchange) | Immutable |
| NATS publish (observation) | `ingest` | NATS URL | Immutable per process |
| Pipeline families (evidence→risk) | `derive` | `pipeline.*` in config | Immutable per process |
| Execution family (paper_order intent) | `derive` | `pipeline.execution_families` | Immutable per process |
| Venue adapter selection | `execute` | `venue.type` in config | Immutable per process |
| Credential loading | `execute` | `MF_VENUE_*` env vars | Immutable per process |
| Gate state | `execute` (KV), `gateway` (HTTP) | NATS KV `EXECUTION_CONTROL` | Runtime-mutable |
| Safety gate (kill switch + staleness) | `execute` | `venue.staleness_max_age` | Immutable per process |
| Activation surface computation | `execute` | Derived from 3 dimensions | Computed on query |
| Fill publishing | `execute` | NATS registry | Immutable per process |

---

## 6. Limits and boundaries of the current model

### 6.1 What the model covers

- Binary-startup adapter selection (paper vs venue)
- Runtime gate control (halt/activate execution)
- Credential presence check (present/absent)
- Staleness rejection for old intents
- Kill switch with timeout-based fail-safe
- Observability via HTTP activation surface endpoint

### 6.2 What the model does NOT cover

| Gap | Category | Severity | When to address |
|-----|----------|----------|-----------------|
| Per-symbol execution control | Control | LOW | Post-OMS — requires symbol-level gate granularity |
| Per-family execution control | Control | LOW | Post-OMS — current gate is global |
| Startup rejection of `venue_degraded` | Activation | LOW | Future hardening stage |
| Hot-reload of `venue.type` | Activation | NONE | Explicitly a non-goal (NG-10 in charter) |
| Multi-venue routing | Execution | NONE | Explicitly a non-goal (NG-3 in charter) |
| OMS / order lifecycle | Execution | NONE | Explicitly a non-goal (NG-1 in charter) |
| Position tracking | Execution | NONE | Explicitly a non-goal (NG-2 in charter) |
| Rate limiting on venue submissions | Execution | LOW | Post-OMS — not needed while paper-only |
| Circuit breaker on venue errors | Execution | LOW | Post-OMS — not needed while paper-only |

### 6.3 Transitional bridge note

The execute binary currently consumes paper_order events from
`EXECUTION_EVENTS` as a transitional bridge (see
`ExecuteVenueMarketOrderIntakeConsumer()`). When real venue intent subjects
are introduced in a future stage, the intake consumer will migrate to
venue-specific subjects. This bridge is safe because the execute binary's
venue adapter (paper or real) handles the intent regardless of its origin
subject.

---

## 7. Invariant summary

| ID | Invariant | Enforcement |
|----|-----------|-------------|
| ELDR-I1 | Default configuration results in paper-mode execution | `buildVenueAdapter()` default case |
| ELDR-I2 | Live venue requires venue.type + credentials + active gate | `ComputeEffectiveMode()` truth table |
| ELDR-I3 | Ingest binary does not require venue credentials | No venue dependency in ingest code |
| ELDR-I4 | Trade deduplication prevents duplicates on reconnect | NATS `Msg-Id` via `DeduplicationKey()` |
| ELDR-I5 | Read path independent of write path | No shared config between ingest/derive and execute |
| ELDR-I6 | Live data flows through same NATS subjects as simulated | Single subject hierarchy |
| FC-1 | Empty venue.type defaults to paper | `buildVenueAdapter()` default case |
| FC-2 | Paper adapter dominates over gate and credentials | `ComputeEffectiveMode()` first branch |
| FC-5 | Gate check timeout fails closed (halted) | `SafetyGate.Check()` timeout |
| CI-1 | Market data WebSocket is always mainnet | Hardcoded URL |
| CI-10 | Read path independent of write path config | Architectural separation |

---

## 8. Preparation for S378

S378 (Compose Live Exchange Listening Proof) requires:

1. **No code changes to ingest.** The WebSocket client already connects to
   mainnet. configctl bindings control which symbols are watched.

2. **No code changes to derive.** The derive pipeline processes
   `TradeReceivedEvent` regardless of data origin.

3. **No code changes to execute.** The default `paper_simulator` config
   keeps execution safe.

4. **Compose validation:** A smoke script must verify that live trades flow
   from Binance → NATS `OBSERVATION_EVENTS` → derive consumers → domain
   events. The script validates the read path only.

5. **Staleness guard compatibility:** Confirmed — 120s default window is
   sufficient for live 60s-timeframe operation (see companion doc section 8).
