# Segmented Config Examples, Fail-Closed Behavior, and Limitations

**Stage:** S393
**Date:** 2026-03-22
**Companion to:** [config-driven-enablement-for-binance-spot-and-futures.md](config-driven-enablement-for-binance-spot-and-futures.md)
**Authority:** This document provides canonical config examples and fail-closed behavior catalog.

---

## 1. Valid Configuration Examples

### 1.1 Paper Simulator (Default)

```jsonc
{
  "venue": {
    "type": "paper_simulator",
    "dry_run": true
    // No segments block needed -- paper has no segment.
  }
}
```

**Result:** Starts normally. Paper adapter loaded. DryRunSubmitter wraps it.

### 1.2 Binance Futures Testnet with Dry Run

```jsonc
{
  "venue": {
    "type": "binance_futures_testnet",
    "dry_run": true,
    "segments": {
      "futures_enabled": true
    }
  }
}
```

**Result:** Starts normally. Futures adapter loaded. DryRunSubmitter intercepts all calls.

### 1.3 Binance Futures Testnet Live Execution

```jsonc
{
  "venue": {
    "type": "binance_futures_testnet",
    "dry_run": false,
    "segments": {
      "futures_enabled": true
    }
  }
}
```

**Result:** Starts normally. Futures adapter loaded. Live venue calls. Requires credentials.

### 1.4 Binance Spot Testnet with Dry Run

```jsonc
{
  "venue": {
    "type": "binance_spot_testnet",
    "dry_run": true,
    "segments": {
      "spot_enabled": true
    }
  }
}
```

**Result:** Config validation passes. Adapter build fails (pending S392 implementation).

### 1.5 Both Segments Enabled (Futures Active)

```jsonc
{
  "venue": {
    "type": "binance_futures_testnet",
    "dry_run": true,
    "segments": {
      "spot_enabled": true,
      "futures_enabled": true
    }
  }
}
```

**Result:** Valid. Both segments are declared enabled, and the active VenueType (futures) matches.

---

## 2. Invalid Configurations and Fail-Closed Behavior

### 2.1 Futures Without Segments Config

```jsonc
{
  "venue": {
    "type": "binance_futures_testnet",
    "dry_run": true
    // Missing segments block
  }
}
```

**Rejection:** `venue type "binance_futures_testnet" requires segments config with futures explicitly enabled`

**Rationale:** Fail-closed. Absent segments block means no segments are enabled.

### 2.2 Futures With Segment Disabled

```jsonc
{
  "venue": {
    "type": "binance_futures_testnet",
    "segments": {
      "futures_enabled": false
    }
  }
}
```

**Rejection:** `venue type "binance_futures_testnet" requires segment "futures" to be explicitly enabled (set to true)`

### 2.3 Futures With Only Spot Enabled

```jsonc
{
  "venue": {
    "type": "binance_futures_testnet",
    "segments": {
      "spot_enabled": true
    }
  }
}
```

**Rejection:** Same as 2.2 -- futures is not enabled (absent = false).

### 2.4 Paper Simulator With Segments Enabled

```jsonc
{
  "venue": {
    "type": "paper_simulator",
    "segments": {
      "spot_enabled": true
    }
  }
}
```

**Rejection:** `segments config is not applicable to paper_simulator`

**Rationale:** Paper simulator has no market segment. Enabling segments on it is a configuration error.

### 2.5 Paper With Empty Segments (No Contradiction)

```jsonc
{
  "venue": {
    "type": "paper_simulator",
    "segments": {}
  }
}
```

**Result:** Valid. Empty segments block has all fields nil = all disabled = no contradiction with paper.

### 2.6 Spot With Futures Enabled But Spot Disabled

```jsonc
{
  "venue": {
    "type": "binance_spot_testnet",
    "segments": {
      "futures_enabled": true,
      "spot_enabled": false
    }
  }
}
```

**Rejection:** `venue type "binance_spot_testnet" requires segment "spot" to be explicitly enabled`

---

## 3. Fail-Closed Decision Table

| VenueType | segments | spot_enabled | futures_enabled | Result |
|-----------|----------|-------------|----------------|--------|
| `paper_simulator` | absent | -- | -- | PASS |
| `paper_simulator` | present | nil | nil | PASS |
| `paper_simulator` | present | true | -- | REJECT |
| `binance_futures_testnet` | absent | -- | -- | REJECT |
| `binance_futures_testnet` | present | -- | nil | REJECT |
| `binance_futures_testnet` | present | -- | false | REJECT |
| `binance_futures_testnet` | present | -- | true | PASS |
| `binance_spot_testnet` | absent | -- | -- | REJECT |
| `binance_spot_testnet` | present | nil | -- | REJECT |
| `binance_spot_testnet` | present | false | -- | REJECT |
| `binance_spot_testnet` | present | true | -- | PASS |

---

## 4. Interaction With dry_run

Segment enablement and dry_run are **independent validation dimensions**. Both must pass:

| Scenario | Segment check | dry_run check | Overall |
|----------|--------------|---------------|---------|
| futures + futures_enabled=true + dry_run=true | PASS | PASS | PASS |
| futures + futures_enabled=true + dry_run=false | PASS | PASS | PASS (live) |
| futures + futures_enabled=false + dry_run=true | REJECT | -- | REJECT |
| paper + no segments + dry_run=true | PASS | PASS | PASS |
| paper + no segments + dry_run=false | PASS | REJECT | REJECT |

---

## 5. Limitations

1. **Segments are Binance-only:** The current model has `spot_enabled` and `futures_enabled` as top-level fields. Multi-exchange would need a different structure (e.g., `exchanges.binance.segments`).
2. **One venue type per binary:** Even when both segments are enabled, only the configured `venue.type` is active. Enabling both is a declaration of intent, not multi-segment routing.
3. **Spot adapter pending:** `binance_spot_testnet` passes config validation but fails at adapter construction until S392 code lands.
4. **Mainnet not registered:** No mainnet VenueTypes are in the code registry. Adding them requires a new stage.
5. **No runtime segment switching:** Segment enablement is a startup-time decision. Changing it requires binary restart.
