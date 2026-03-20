# First Generated Family — Implementation and Validation

**Stage:** S203
**Date:** 2026-03-20
**Family:** EMA (Exponential Moving Average)
**Layer:** Signal (Tier 1)

---

## Summary

EMA is the first codegen-first family integrated into the market-foundry analytical pipeline. All generated artifacts (A1 consumer spec, A2 pipeline entry) were produced by the codegen engine from `codegen/families/ema.yaml` and inserted into the runtime codebase with governance markers. No manual edits were applied to generated code.

## Spec

```yaml
family:
  name: ema
  layer: signal
  tier: 1
nats:
  subject: "signal.events.ema.generated.>"
  event_type: "signal.events.v1.ema_generated"
  stream: SIGNAL_EVENTS
  durable: writer-signal-ema
writer:
  table: signals
  mapper: mapSignalRow
  pipeline_family_key: ema
  config_array: signal_families
domain:
  event_package: signal
  event_type: SignalGeneratedEvent
```

## Generated Artifacts

### A1 — Consumer Spec (`WriterEMASignalConsumer`)

- **Target:** `internal/adapters/nats/signal_registry.go`
- **Marker:** `codegen:begin consumer_spec family=ema`
- **Golden:** `codegen/golden-snapshots/ema/consumer_spec.go.golden`
- **Function:** Returns `ConsumerSpec` with durable `writer-signal-ema`, subject `signal.events.ema.generated.>`, stream `SIGNAL_EVENTS`

### A2 — Pipeline Entry

- **Target:** `cmd/writer/pipeline.go`
- **Marker:** `codegen:begin pipeline_entry family=ema`
- **Golden:** `codegen/golden-snapshots/ema/pipeline_entry.go.golden`
- **Content:** `writerPipeline` struct literal wiring EMA consumer spec → `mapSignalRow` → `signals` table

## Manual Artifacts

### A5 — Config Entry

- **File:** `deploy/configs/writer.jsonc`
- **Change:** Added `"ema"` to `signal_families` array
- **Rationale:** Operational decision — opt-in activation

### Settings Registration

- **File:** `internal/shared/settings/schema.go`
- **Changes:**
  - Added `"ema": true` to `knownSignalFamilies`
  - Added `"ema": {"candle"}` to `signalDependsOnEvidence`
- **Rationale:** Config validation requires known family registration

## Infrastructure Reuse (Zero New)

| Component | Shared With |
|-----------|-------------|
| `SIGNAL_EVENTS` NATS stream | RSI, EMA Crossover |
| `signals` ClickHouse table | RSI |
| `NewSignalConsumer` factory | RSI |
| `reg.signal` registry | RSI |
| `mapSignalRow` mapper | RSI |
| `IsSignalFamilyEnabled` config method | RSI |
| Signal reader / handler / route | RSI |

## Validation Evidence

### SC-1: Spec Validates Clean

```
$ codegen validate families/ema.yaml
VALID  families/ema.yaml (family=ema, layer=signal, tier=1)

$ codegen validate-all
7 families validated, cross-spec uniqueness: OK
```

### SC-2: Golden Snapshots Match

```
$ codegen check-all
14 passed, 0 failed (including ema/consumer_spec, ema/pipeline_entry)
```

### SC-3: Code Compiles

```
$ go build ./cmd/writer/...        → clean
$ go build ./internal/adapters/nats/...  → clean
```

### SC-4: Existing Tests Pass

```
$ go test ./cmd/writer/...                → ok
$ go test ./internal/shared/settings/...  → ok
$ go test ./internal/adapters/nats/...    → ok
$ go test ./internal/adapters/clickhouse/... → ok
$ go test ./internal/interfaces/http/...  → ok
$ go test ./internal/application/...      → ok (all subpackages)
$ go test ./codegen/...                   → ok
```

### SC-5: Integrated Check Passes

```
$ ./scripts/codegen-integrated-check.sh
PASS rsi/consumer_spec
PASS rsi/pipeline_entry
PASS ema/consumer_spec
PASS ema/pipeline_entry
4 passed, 0 failed (4 checked)
```

### SC-7: No Manual Edits to Generated Code

Generated fragments between `codegen:begin` and `codegen:end` markers are byte-identical to golden snapshots after normalization. Confirmed by SC-5 integrated check.

### SC-8: Manifest Updated

`codegen/integrated.yaml` now contains 4 entries:
- `rsi/consumer_spec` (S200)
- `rsi/pipeline_entry` (S200)
- `ema/consumer_spec` (S203)
- `ema/pipeline_entry` (S203)

## What Was Proven

1. **Codegen-first family participates in real runtime** — EMA pipeline entry is structurally identical to manual families and compiles into the writer binary
2. **Spec → generate → golden → insert → validate pipeline works end-to-end** — no step required human intervention beyond config opt-in
3. **Cross-spec validation catches collisions** — 7 families validated with no durable/subject/name conflicts
4. **Governance markers + CI checks detect drift** — integrated check verifies 4 slices (2 RSI + 2 EMA)
5. **Existing families unaffected** — zero test regressions across all packages
6. **Infrastructure reuse confirmed** — EMA adds zero new infrastructure, shares everything with RSI

## What Was NOT Proven

1. **Runtime activation under live traffic** — SC-6 requires `make up` with live NATS events; this stage validates compilation and governance, not live traffic
2. **EMA signal producer existence** — no derive service currently emits EMA signals; activation proof requires either a synthetic event injector or derive service extension
3. **Read path for EMA specifically** — EMA shares the signal reader which is already proven for RSI; no EMA-specific read path test was added
4. **Mapper generation** — `mapSignalRow` was reused, not generated (NG-4)
5. **Template changes** — templates are frozen (NG-5)
6. **Performance under load** — not in scope (NG-7)

## Derived Fields Verification

| Field | Expected | Actual |
|-------|----------|--------|
| ConsumerSpecFunc | `WriterEMASignalConsumer` | `WriterEMASignalConsumer` |
| ConsumerName | `writer-signal-ema-consumer` | `writer-signal-ema-consumer` |
| InserterName | `writer-signal-ema-inserter` | `writer-signal-ema-inserter` |
| IsEnabledMethod | `IsSignalFamilyEnabled` | `IsSignalFamilyEnabled` |
| RegistryField | `signal` | `signal` |
| NewConsumerFunc | `NewSignalConsumer` | `NewSignalConsumer` |
| PascalFamily | `EMA` | `EMA` |
| PascalLayer | `Signal` | `Signal` |
| InsertSQL | `INSERT INTO signals` | `INSERT INTO signals` |
| HyphenFamily | `ema` | `ema` |

All 10 derived fields match expectations. `knownAbbreviations` correctly maps `"ema" → "EMA"`.
