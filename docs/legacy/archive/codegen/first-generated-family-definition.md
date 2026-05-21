# First Generated Family Definition

> Stage: S202 — First Codegen-First Family Definition
> Status: Approved
> Date: 2026-03-20

## 1. Family Selected

**EMA signal family** — a second signal-layer family that shares all existing infrastructure with the RSI signal family.

## 2. Selection Rationale

The first codegen-first family must minimize risk while maximizing proof value. EMA was chosen because it satisfies every constraint simultaneously:

| Criterion | EMA Signal | Justification |
|-----------|-----------|---------------|
| Existing layer | Yes | Signal layer fully operational (RSI already governed) |
| Existing table | Yes | `signals` table already accepts signal rows |
| Existing domain type | Yes | `signal.SignalGeneratedEvent` is the shared event type |
| Existing consumer factory | Yes | `adapternats.NewSignalConsumer` handles all signal families |
| Existing registry | Yes | `reg.signal` registry already wired |
| Existing config pattern | Yes | `signal_families` array in writer config |
| Existing mapper | Yes | `mapSignalRow` works for all signal events |
| Known abbreviation | Yes | `"ema"` is in `knownAbbreviations` — anticipated by design |
| Single-family iteration | Yes | One family, one layer, no cross-layer changes |
| Tier 1 only | Yes | Write-path only, no reader generation |

### Why Not Other Candidates

| Candidate | Rejection Reason |
|-----------|-----------------|
| Second decision family | Decision layer has only one family (rsi_oversold); adding a second requires new domain event type or new decision logic — too much manual work |
| Second evidence family | Evidence layer has special naming rules; first codegen-first should use standard derivation |
| New layer family | Would require new DDL, new consumer factory, new registry — defeats "minimal risk" |
| Candle (evidence) as codegen-first | Already exists manually; this stage is about a NEW family, not retroactive governance |

## 3. Frozen Spec

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

### Derived Fields (Deterministic)

| Field | Value | Derivation Rule |
|-------|-------|----------------|
| ConsumerSpecFunc | `WriterEMASignalConsumer` | `"Writer" + PascalFamily + PascalLayer + "Consumer"` |
| ConsumerName | `writer-signal-ema-consumer` | `"writer-" + layer + "-" + hyphenFamily + "-consumer"` |
| InserterName | `writer-signal-ema-inserter` | `"writer-" + layer + "-" + hyphenFamily + "-inserter"` |
| IsEnabledMethod | `IsSignalFamilyEnabled` | `"Is" + PascalLayer + "FamilyEnabled"` |
| RegistryField | `signal` | Same as `family.layer` |
| NewConsumerFunc | `NewSignalConsumer` | `"New" + PascalLayer + "Consumer"` |
| PascalFamily | `EMA` | From `knownAbbreviations["ema"]` |
| PascalLayer | `Signal` | Standard PascalCase |
| InsertSQL | `INSERT INTO signals` | `"INSERT INTO " + table` |
| HyphenFamily | `ema` | No underscores to convert |

### Cross-Spec Uniqueness Verification

| Constraint | EMA Value | Conflicts? |
|------------|-----------|-----------|
| `family.name` | `ema` | No — unique across all 6 existing specs |
| `nats.durable` | `writer-signal-ema` | No — unique |
| `nats.subject` | `signal.events.ema.generated.>` | No — unique |

## 4. Generated Artifacts (A1 + A2)

### A1: Consumer Spec (`consumer_spec`)

**Target file:** `internal/adapters/nats/signal_registry.go`
**Marker:** `codegen:begin consumer_spec family=ema`

Expected generated output:

```go
// WriterEMASignalConsumer defines the durable consumer spec for writer consuming
// ema signal events.
func WriterEMASignalConsumer() ConsumerSpec {
	return ConsumerSpec{
		Durable: "writer-signal-ema",
		Event: EventSpec{
			Subject: "signal.events.ema.generated.>",
			Type:    "signal.events.v1.ema_generated",
			Stream: StreamSpec{
				Name: "SIGNAL_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}
```

### A2: Pipeline Entry (`pipeline_entry`)

**Target file:** `cmd/writer/pipeline.go`
**Marker:** `codegen:begin pipeline_entry family=ema`

Expected generated output:

```go
// ── Signal: ema → signals ───────────────────────────────────
{
	family:       "ema",
	consumerName: "writer-signal-ema-consumer",
	inserterName: "writer-signal-ema-inserter",
	table:        "signals",
	insertSQL:    "INSERT INTO signals",
	consumerSpec: adapternats.WriterEMASignalConsumer(),
	isEnabled:    func(p settings.PipelineConfig) bool { return p.IsSignalFamilyEnabled("ema") },
	startConsumer: func(natsURL string, spec adapternats.ConsumerSpec, inserterPID *actor.PID, tracker *healthz.Tracker, logger *slog.Logger, actorCtx *actor.Context) (io.Closer, error) {
		consumer := adapternats.NewSignalConsumer(natsURL, spec, reg.signal,
			func(event signal.SignalGeneratedEvent) {
				if tracker != nil {
					tracker.RecordEvent()
					tracker.Counter("events_received").Add(1)
				}
				actorCtx.Send(inserterPID, insertRowMsg{row: mapSignalRow(event)})
			},
			logger,
		)
		return consumer, consumer.Start()
	},
},
```

## 5. Integration Manifest Entry

Two new entries will be added to `codegen/integrated.yaml`:

```yaml
  - family: ema
    artifact: consumer_spec
    spec: codegen/families/ema.yaml
    golden: codegen/golden-snapshots/ema/consumer_spec.go.golden
    target: internal/adapters/nats/signal_registry.go
    marker: "codegen:begin consumer_spec family=ema"
    integrated_at: "<S203 date>"
    stage: S203

  - family: ema
    artifact: pipeline_entry
    spec: codegen/families/ema.yaml
    golden: codegen/golden-snapshots/ema/pipeline_entry.go.golden
    target: cmd/writer/pipeline.go
    marker: "codegen:begin pipeline_entry family=ema"
    integrated_at: "<S203 date>"
    stage: S203
```

## 6. What This Iteration Proves

1. **Spec-first authorship works** — the YAML spec is written before any implementation code exists.
2. **Generation produces correct, compilable code** — golden snapshots match template output exactly.
3. **Same-layer expansion works** — a second family in the signal layer reuses all existing infrastructure.
4. **Shared mapper pattern works** — `mapSignalRow` handles any `SignalGeneratedEvent` regardless of family.
5. **CI drift detection scales** — adding a governed family to an already-governed target file works cleanly.
6. **Minimal manual delta** — the only manual work is config entry and smoke validation.

## 7. What This Iteration Does NOT Prove

- Cross-layer family generation (different table, different domain type)
- New mapper generation (A3 artifact)
- New DDL or migration generation
- Evidence layer special-case handling for codegen-first
- Tier 2 read-path generation
- Multiple families in a single iteration
