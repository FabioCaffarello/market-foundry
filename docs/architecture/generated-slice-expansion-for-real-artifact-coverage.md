# Generated Slice Expansion for Real Artifact Coverage

> Stage S260 — Expanding codegen governance from 2 to 10 families across the writer pipeline.

## Objective

Expand the codegen-governed slice from the initial 2 signal families (rsi, ema) to cover all 10 tier-1 families, governing both `consumer_spec` and `pipeline_entry` artifacts across all six domain layers.

## Scope

### What Was Expanded

| Layer | Families | Artifacts | Target Files |
|-------|----------|-----------|--------------|
| Evidence | candle | consumer_spec, pipeline_entry | `natsevidence/registry.go`, `cmd/writer/pipeline.go` |
| Signal | rsi, ema | *(already governed — unchanged)* | *(unchanged)* |
| Decision | rsi_oversold, ema_crossover | consumer_spec, pipeline_entry | `natsdecision/registry.go`, `cmd/writer/pipeline.go` |
| Strategy | mean_reversion_entry, trend_following_entry | consumer_spec, pipeline_entry | `natsstrategy/registry.go`, `cmd/writer/pipeline.go` |
| Risk | position_exposure, drawdown_limit | consumer_spec, pipeline_entry | `natsrisk/registry.go`, `cmd/writer/pipeline.go` |
| Execution | paper_order | consumer_spec, pipeline_entry | `natsexecution/registry.go`, `cmd/writer/pipeline.go` |

### Quantitative Summary

| Metric | Before (S259) | After (S260) |
|--------|--------------|--------------|
| Governed families | 2 (rsi, ema) | 10 (all tier-1) |
| Integrated slices | 4 | 20 |
| Target files with markers | 2 | 7 |
| Golden snapshots | 20 | 20 (unchanged — already existed) |
| Family specs | 10 | 10 (unchanged) |

## Implementation Details

### Consumer Spec Migration

For all 8 newly governed families, the writer consumer spec functions were migrated from the factory-style `natskit.NewConsumerSpec(...)` to the expanded struct literal form that matches the golden snapshots:

**Before (factory form):**
```go
func WriterRSIOversoldDecisionConsumer() natskit.ConsumerSpec {
    return natskit.NewConsumerSpec("writer-decision-rsi-oversold", ...)
}
```

**After (expanded form, codegen-governed):**
```go
// codegen:begin consumer_spec family=rsi_oversold source=codegen/families/rsi_oversold.yaml
func WriterRSIOversoldDecisionConsumer() natskit.ConsumerSpec {
    return natskit.ConsumerSpec{
        Durable: "writer-decision-rsi-oversold",
        Event: natskit.EventSpec{...},
        AckWait:    30 * time.Second,
        MaxDeliver: 5,
    }
}
// codegen:end consumer_spec family=rsi_oversold
```

The expanded form is semantically equivalent, more readable, and diff-friendly. Function signatures are unchanged — all call sites remain valid.

### Pipeline Entry Markers

All 10 pipeline entries in `cmd/writer/pipeline.go` are now wrapped with `codegen:begin`/`codegen:end` markers. No structural changes to the entries themselves — only governance markers were added.

### Integrated Check Script Hardening

The `scripts/codegen-integrated-check.sh` extraction logic was upgraded from regex-based sed to exact-match awk. This prevents substring false positives where `family=rsi` would incorrectly match `family=rsi_oversold`. The new `exact_match()` function verifies the marker is followed by space, tab, or end-of-line.

### Store Consumer Specs

Store consumer specs (`StoreXxx()` functions) remain manual:owned. They use the factory form and are NOT governed by codegen. This is intentional — store consumer specs may evolve independently (different AckWait, different stream configurations) and are less repetitive than writer specs.

## Verification Results

```
codegen check-all:     20/20 PASS
codegen validate-all:  10/10 VALID (no collisions)
codegen-integrated:    20/20 PASS
codegen-test:          OK
go build:              OK (all affected packages)
```

## What Was NOT Expanded

- No new templates were added
- No new family specs were created
- No domain logic was generated
- Store consumer specs remain manual
- NATS consumer/publisher/gateway/kv_store files remain manual
- Actor evaluator/projection files remain manual
- ClickHouse reader/mapper files remain manual

## Risk Assessment

| Risk | Mitigation |
|------|------------|
| Semantic drift between factory and expanded form | Verified equivalence via golden snapshot comparison |
| Substring matching in integrated check | Fixed with exact-match extraction (awk) |
| Breaking existing callers | Function signatures unchanged — only body changed |
| Evidence naming conventions | Validated — candle uses layer-omitted naming per spec |
