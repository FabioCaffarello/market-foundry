# Config Activation and Dependency Map Model

## Overview

Market Foundry uses two complementary configuration surfaces:

1. **Static pipeline config** (`config.jsonc` → `PipelineConfig`) — controls which families (evidence, signal, decision, strategy, risk, execution) are activated at process startup. Changes require runtime restart.
2. **Dynamic ingestion config** (configctl domain) — controls which (source, symbol) binding pairs are active. Changes propagate at runtime via `IngestionRuntimeChangedEvent`.

This document describes how these surfaces interact, what dependencies exist between them, and how new entries should be added.

## Canonical Family Catalog

The single source of truth for recognized family names lives in `internal/shared/settings/schema.go`:

| Domain | Known Families | Activation Mode |
|--------|---------------|-----------------|
| Evidence | `candle`, `tradeburst`, `volume` | Backward-compatible default (all enabled when list empty) |
| Signal | `rsi` | Opt-in only |
| Decision | `rsi_oversold` | Opt-in only |
| Strategy | `mean_reversion_entry` | Opt-in only |
| Risk | `position_exposure` | Opt-in only |
| Execution | `paper_order`, `venue_market_order` | Opt-in only |

### Exported Catalog API

```go
settings.KnownFamilies(domain PipelineDomain) []string
settings.IsKnownFamily(domain PipelineDomain, family string) bool
settings.DependencyGraph() []FamilyDependency
```

These functions expose the canonical catalog for tooling, tests, and coherence checks without requiring consumers to duplicate knowledge.

## Cross-Layer Dependency Rules

Dependencies flow strictly downward through the pipeline:

```
Evidence (candle, tradeburst, volume)
    ↓
Signal (rsi → requires candle)
    ↓
Decision (rsi_oversold → requires rsi)
    ↓
Strategy (mean_reversion_entry → requires rsi_oversold)
    ↓
Risk (position_exposure → requires mean_reversion_entry)
    ↓
Execution (paper_order, venue_market_order → requires position_exposure)
```

These dependencies are **enforced at config validation time** by `PipelineConfig.ValidatePipeline()`. Enabling a downstream family without its upstream dependency produces a validation error that prevents the process from starting.

## Activation Flow: Static Pipeline

```
config.jsonc
  → bootstrap.LoadAndValidate()
    → settings.Load() + cfg.Validate()
      → PipelineConfig.ValidatePipeline()
        → Reject unknown families
        → Reject duplicate families
        → Enforce cross-layer dependency rules
  → Per-runtime processor/pipeline registration
    → derive: filterEnabled() on each processor slice
    → store: declarePipelines() with IsEnabled predicates
```

### What is canonical vs. derived

| Artifact | Type | Location |
|----------|------|----------|
| Known family registry | **Canonical** | `settings/schema.go` (knownXxxFamilies maps) |
| Cross-layer dependency rules | **Canonical** | `settings/schema.go` (xxxDependsOnYyy maps) |
| Derive processor list | **Derived** | `derive/derive_supervisor.go` (each entry references a canonical family) |
| Store pipeline catalog | **Derived** | `store/store_supervisor.go` (`declarePipelines()`) |
| Store tracker definitions | **Derived** | `store/store_supervisor.go` (`PipelineTrackerDefs()` — derived from pipeline catalog) |
| Health tracker map | **Derived** | `cmd/store/run.go` (derived from `PipelineTrackerDefs()`) |

## Activation Flow: Dynamic Bindings (Configctl)

```
ConfigDocument (YAML/JSON)
  → InspectDocument() + Validate()
    → Structural validation (metadata, bindings, fields, rules)
    → Binding topic format: must be 'source.symbol' (e.g., binancef.btcusdt)
    → Field type and rule reference validation
  → CreateDraft → ValidateVersion → CompileVersion → ActivateVersion
    → Artifact validation: schema_version and runtime_loader must be known
    → Scope-aware activation (global:default, tenant:br, etc.)
  → IngestionRuntimeChangedEvent published to NATS
    → Ingest binding watcher receives and activates/deactivates WebSocket connections
    → Derive binding watcher receives and activates/deactivates sampler actors
```

### Binding Topic Convention

Binding topics must follow the format `source.symbol`:
- Both segments must be lowercase alphanumeric with underscores
- Examples: `binancef.btcusdt`, `source_a.eth_usdt`
- This is validated in `ConfigDocument.Validate()` and enforced by `ParseBindingTopic()` in the ingest application layer

## Runtime Dependency Map

```
Configctl  ←── no dependencies (source of truth for bindings)
    ↓ (queries + event stream)
Ingest     ←── depends on configctl (binding activation)
    ↓ (NATS stream: TRADES_RECEIVED)
Derive     ←── depends on configctl (binding activation)
           ←── depends on ingest indirectly (consumes trade stream)
    ↓ (NATS streams: CANDLE_SAMPLED, RSI_SIGNAL, etc.)
Store      ←── depends on derive (consumes evidence/signal/decision/strategy/risk/execution streams)
           ←── does NOT depend on configctl at runtime
    ↓ (NATS KV buckets)
Gateway    ←── depends on configctl (required, HTTP façade for config management)
           ←── depends on evidence/signal/decision/strategy/risk/execution (optional, graceful degradation)
Execute    ←── depends on derive (consumes execution intent events)
           ←── depends on venue adapter (external market connection)
```

## How to Add a New Family

### Step 1 — Register in the canonical catalog (`settings/schema.go`)

1. Add the family to the appropriate `knownXxxFamilies` map.
2. Add cross-layer dependency entries if the family depends on upstream families.
3. Run `go test ./internal/shared/settings/...` to verify.

### Step 2 — Add derive processor (`derive/derive_supervisor.go`)

1. Add a new entry to the appropriate processor slice in `start()`.
2. The `Family` field must match the canonical name from Step 1.
3. Implement the processor actor.

### Step 3 — Add store pipeline (`store/store_supervisor.go`)

1. Add a new `Pipeline` entry in `declarePipelines()`.
2. The `Family` field must match the canonical name.
3. Implement the projection and consumer actors.
4. The tracker definition is automatically derived — no separate registration needed.

### Step 4 — Add gateway routes (if query surface needed)

1. Add use case in `internal/application/xxxclient/`.
2. Wire in `cmd/gateway/compose.go`.
3. Add route in gateway actor.

### Step 5 — Update configctl document schema (if binding changes needed)

1. No changes needed if the new family uses existing binding topics.
2. If new field types or rule operators are needed, extend the domain model.

## Limits and Manual Points

- **Pipeline family config is static** — adding/removing families requires process restart.
- **Derive and store registrations are independent** — adding a family to derive without the corresponding store pipeline will silently skip materialization. This is by design (not all families need materialization), but coherence should be verified.
- **Binding topic format is validated but not semantically verified** — configctl validates the `source.symbol` format but does not check whether the source actually exists as an exchange adapter.
- **Artifact schema versions and runtime loaders are whitelisted** — adding new versions requires explicit registration in `domain/configctl/runtime.go`.
