# Config-Driven Activation Hardening

> Defines the activation model for Market Foundry binaries: what runs, when it starts, and how config controls it.
> Established: S34 (2026-03-17).

---

## Activation Layers

Market Foundry uses a two-layer activation model:

| Layer | What It Controls | Where | Mechanism | Runtime Dynamic? |
|-------|-----------------|-------|-----------|-----------------|
| **Family activation** | Which evidence types are processed | derive, store | `pipeline.families` in JSONC config | No (requires restart) |
| **Binding activation** | Which sources/symbols are active | ingest, derive | Configctl bindings via BindingWatcherActor | Yes (live via events) |

### Family Activation (Structural)

`pipeline.families` is an optional string array in the service config. It controls which evidence families a binary will instantiate:

```jsonc
"pipeline": {
  "families": ["candle", "tradeburst", "volume"]
}
```

**Behavior:**
- If `families` is present and non-empty: only listed families are activated.
- If `families` is absent or empty (`[]`): all available families are activated (backward compatible).
- If all listed families are unknown: the binary fails to start with a clear error.

**Affected binaries:**
- **derive** — filters FamilyProcessor entries; only enabled families get sampler actors.
- **store** — filters ProjectionPipeline entries; only enabled families get consumer/projection actors and health trackers.
- **ingest** — not affected (ingest has no family concept; it produces raw observations).
- **gateway** — not affected (gateway is stateless; evidence use cases are nil-safe).

### Binding Activation (Runtime)

Configctl bindings control which market data sources and symbols are active. This is fully runtime-dynamic:

1. **At startup:** BindingWatcherActor queries configctl for active bindings via `ListActiveIngestionBindings`.
2. **At runtime:** BindingWatcherActor subscribes to `IngestionRuntimeChangedEvent` for dynamic activation.
3. **On activation:** Supervisor ensures source scope exists, then spawns adapters (ingest) or samplers (derive) for the new symbol.

**Affected binaries:**
- **ingest** — creates ExchangeScopeActor per source, WebSocketAdapterActor per symbol.
- **derive** — creates SourceScopeActor per source, sampler actors per symbol × timeframe × family.
- **store** — not affected (store consumes from evidence stream; it doesn't need per-symbol activation).

---

## Activation Matrix

| Binary | Family activation | Binding activation | Evidence types | Source/symbol control |
|--------|------------------|--------------------|---------------|---------------------|
| **ingest** | N/A | BindingWatcherActor | N/A | Dynamic (configctl) |
| **derive** | `pipeline.families` | BindingWatcherActor | Config-driven | Dynamic (configctl) |
| **store** | `pipeline.families` | N/A | Config-driven | N/A (consumes all from stream) |
| **gateway** | N/A | N/A | N/A | N/A (stateless) |

---

## How to Add a New Evidence Family

With config-driven activation, adding a new evidence family (e.g., `stats`) follows this pattern:

1. **Implement** the sampler actor (derive), consumer/projection actors (store), and KV store adapter.
2. **Register** the family in derive's `allProcessors` and store's `allPipelines` slices.
3. **Add** the family name to `pipeline.families` in `deploy/configs/derive.jsonc` and `store.jsonc`.
4. **Deploy** — only the listed families activate.

To test in isolation, deploy with `"families": ["stats"]` only. To run alongside existing families, add to the list.

---

## How to Add a New Domain (e.g., Signal)

Signal would follow a different activation path because it introduces a new stream family, not just a new evidence type:

1. **Create** a new binary (`cmd/signal-derive/`) or extend derive with signal-specific processors.
2. **Define** `SIGNAL_EVENTS` stream in a new registry.
3. **Add** signal families to `pipeline.families` when ready.
4. **Gate** signal activation behind configctl bindings if per-symbol signal control is needed.

The `pipeline.families` mechanism ensures signal code can exist in the binary without activating until config explicitly enables it.

---

## Deactivation Model

### Family deactivation
Remove the family from `pipeline.families` and restart the binary. The family's actors will not be spawned.

### Binding deactivation (known limitation)
When configctl clears a binding, the `IngestionRuntimeChangeCleared` event is received by BindingWatcherActor but only logged. Full deactivation (stopping samplers, closing WebSocket connections) requires scope→binding tracking that is not yet implemented.

**Workaround:** Restart the binary to deactivate a binding. The startup query will only activate currently-active bindings.

**Impact:** Low. Binding deactivation is rare in normal operation (typically only during config changes). The system is safe — stale samplers consume CPU but do not corrupt data.

---

## Backward Compatibility

The `pipeline.families` field is fully backward compatible:

| Config state | Behavior |
|-------------|----------|
| Field absent | All families enabled (pre-S34 behavior) |
| Field present, empty array `[]` | All families enabled |
| Field present, non-empty | Only listed families enabled |
| Field present, all unknown names | Binary fails to start with clear error |

Existing configs without `pipeline.families` continue to work unchanged.

---

## Config Schema Reference

```jsonc
{
  "pipeline": {
    // Optional: timeframes in seconds (derive only).
    "timeframes": [60, 300],

    // Optional: evidence families to enable.
    // Omit or set to [] for all available families.
    // Known families: "candle", "tradeburst", "volume"
    "families": ["candle", "tradeburst", "volume"]
  }
}
```

**Validation:**
- `IsFamilyEnabled(family)` — returns true if family is in the list or list is empty.
- `EnabledFamilies()` — returns the list, or nil if all are enabled.
- `TimeframeDurations()` — returns configured timeframes as `[]time.Duration`, fallback to `[60s]`.
