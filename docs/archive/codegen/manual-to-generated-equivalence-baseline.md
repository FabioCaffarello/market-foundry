# Manual-to-Generated Equivalence Baseline

> S194 — Defines which hand-crafted families serve as reference, which artifacts are compared, and how equivalence is measured.

## 1. Baseline Family Selection

### Selected Families

| Family | Layer | Tier | Role in Baseline | Rationale |
|--------|-------|------|------------------|-----------|
| **RSI** | L2 Signal | 1 | Minimal Reference | Fewest columns (12), shared table, 1 JSON transform, 1 `parseFloat` — represents the simplest within-layer expansion case |
| **Paper Order** | L6 Execution | 1 | Complexity Ceiling | Most columns (20), 4 JSON fields, 2 enums, 2 correlation IDs — exercises every transform type and the widest column spread |

### Why Two, Not One or Six

- **One is insufficient**: a single family cannot distinguish pattern from accident. RSI alone would miss multi-JSON, multi-enum, and correlation-ID patterns.
- **Six is excessive**: all families follow the same structural pattern with zero creative decisions. Covering the simplest and most complex brackets the entire space.
- **Two brackets the range**: any future family's complexity will fall between RSI and Paper Order. If codegen reproduces both correctly, intermediate families are covered by interpolation.

### Deferred Families (Not in Initial Baseline)

| Family | Why Deferred |
|--------|-------------|
| Candle (L1) | Unique layer (evidence), but structurally identical pattern; adds cost without new signal |
| RSI Oversold (L3) | Mid-complexity; covered by RSI–Paper Order bracket |
| Mean Reversion Entry (L4) | Mid-complexity; covered by bracket |
| Position Exposure (L5) | Near Paper Order complexity; marginal incremental value |

These families remain available as **expansion evidence** if the initial baseline proves insufficient during S195 implementation.

## 2. Artifacts Under Comparison

### Tier 1 Artifact Inventory (Per Family)

| # | Artifact | Source Location | Equivalence Type | In Baseline |
|---|----------|----------------|------------------|:-----------:|
| A1 | Consumer spec function | `internal/adapters/nats/{layer}_registry.go` | Structural | ✅ |
| A2 | Pipeline entry | `cmd/writer/pipeline.go` | Structural | ✅ |
| A3 | Mapper function | `cmd/writer/mappers.go` | Structural + Semantic | ✅ |
| A4 | Mapper unit tests | `cmd/writer/mappers_test.go` | Structural | ✅ |
| A5 | Config array entry | `deploy/configs/writer.jsonc` | Structural | ✅ |
| A6 | Smoke test phase | `scripts/smoke-analytical-e2e.sh` | Structural | ✅ |

### Artifact Extraction Points (RSI)

```
A1: WriterRSISignalConsumer()           → internal/adapters/nats/signal_registry.go
A2: pipeline entry for "rsi"            → cmd/writer/pipeline.go
A3: mapSignalRow()                      → cmd/writer/mappers.go
A4: TestMapSignalRow*                   → cmd/writer/mappers_test.go
A5: "signal_families": ["rsi"]          → deploy/configs/writer.jsonc
A6: Signal history curl + assertions    → scripts/smoke-analytical-e2e.sh
```

### Artifact Extraction Points (Paper Order)

```
A1: WriterPaperOrderExecutionConsumer() → internal/adapters/nats/execution_registry.go
A2: pipeline entry for "paper_order"    → cmd/writer/pipeline.go
A3: mapExecutionRow()                   → cmd/writer/mappers.go
A4: TestMapExecutionRow*                → cmd/writer/mappers_test.go
A5: "execution_families": ["paper_order"] → deploy/configs/writer.jsonc
A6: Execution history curl + assertions → scripts/smoke-analytical-e2e.sh
```

## 3. Golden Spec Construction

For each baseline family, a **golden spec** YAML will be manually authored to describe the existing hand-crafted code. These specs live in `codegen/golden/` and are never auto-generated.

### Golden Spec for RSI

```yaml
# codegen/golden/rsi.yaml
family:
  name: rsi
  layer: signal
  tier: 1

nats:
  subject: "signal.events.rsi.generated.>"
  event_type: "signal.events.v1.rsi_generated"
  stream: "SIGNAL_EVENTS"
  durable: "writer-signal-rsi"

writer:
  table: signals
  mapper: mapSignalRow
  pipeline_family_key: rsi
  config_array: signal_families

domain:
  event_package: signal
  event_type: SignalGeneratedEvent
```

### Golden Spec for Paper Order

```yaml
# codegen/golden/paper_order.yaml
family:
  name: paper_order
  layer: execution
  tier: 1

nats:
  subject: "execution.events.paper_order.submitted.>"
  event_type: "execution.events.v1.paper_order_submitted"
  stream: "EXECUTION_EVENTS"
  durable: "writer-execution-paper-order"

writer:
  table: executions
  mapper: mapExecutionRow
  pipeline_family_key: paper_order
  config_array: execution_families

domain:
  event_package: execution
  event_type: PaperOrderSubmittedEvent
```

## 4. Comparison Procedure

### Step-by-Step Equivalence Check

```
1. Load golden spec (codegen/golden/{family}.yaml)
2. Run codegen templates against spec → produce candidate artifacts
3. Extract corresponding hand-crafted artifacts from source
4. Normalize both (gofmt, strip comments, sort imports)
5. Apply structural comparison rules (see equivalence-scope doc)
6. Apply semantic comparison rules (see equivalence-scope doc)
7. Report: PASS (equivalent) / FAIL (divergence list)
```

### What "Pass" Means

A baseline family **passes** equivalence when:

- All 6 artifact types are generated
- Structural comparison shows no divergence after normalization
- Semantic comparison confirms behavioral equivalence
- Generated code compiles (`go build ./...`)
- Generated tests pass (`go test ./...`)

### What "Fail" Means

Any of the following constitutes a **failure**:

- Missing artifact (artifact count mismatch)
- Structural field missing or extra
- SQL shape difference (column order, types)
- Different validation logic
- Different error handling paths
- Compilation failure
- Test failure

## 5. Baseline Limits

### What This Baseline Covers

- Tier 1 artifacts only (write-path)
- Within-layer expansion pattern
- Named mapper reuse (mapper already exists)
- Shared table reuse (no DDL generation)

### What This Baseline Does NOT Cover

- Tier 2 artifacts (read-path, handler, route, contracts)
- New-layer expansion (new table, new DDL)
- Generated mapper (`mapper: "generate"` with `domain.columns`)
- Cross-family interactions
- Performance characteristics
- Operational behavior under load

### Expansion Triggers

| Trigger | Action |
|---------|--------|
| Tier 1 templates proven correct against RSI + Paper Order | Extend baseline to include generated-mapper case |
| Tier 2 authorized | Add read-path artifacts to baseline using same two families |
| New layer needed | Add Candle (L1) to baseline for DDL generation validation |
