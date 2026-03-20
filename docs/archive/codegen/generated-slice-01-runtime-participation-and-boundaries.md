# Generated Slice 01: Runtime Participation and Boundaries

## Scope

This document defines how the first codegen-governed slice (RSI signal, A1 + A2) participates in the runtime and where the boundary between generated and manual code lies.

## Runtime participation

### What the generated code does at runtime

**A1 — `WriterRSISignalConsumer()` consumer spec**:
- Returns a `ConsumerSpec` struct with NATS durable consumer configuration.
- Called once during pipeline declaration in `declareWriterPipelines()`.
- Pure data — no side effects, no I/O, no state.
- Deterministic: always returns the same struct.

**A2 — RSI pipeline entry struct**:
- Declares the `writerPipeline` struct literal for the RSI family.
- Wires the NATS consumer to the inserter actor via the `startConsumer` closure.
- References manual artifacts: `mapSignalRow` mapper, `NewSignalConsumer` constructor, `reg.signal` registry.
- The `isEnabled` predicate delegates to `settings.PipelineConfig.IsSignalFamilyEnabled("rsi")`.

### Runtime flow

```
Pipeline declaration (startup)
  │
  ├── A1: WriterRSISignalConsumer()  ← codegen-governed
  │     returns ConsumerSpec{Durable, Event, AckWait, MaxDeliver}
  │
  └── A2: writerPipeline{...}       ← codegen-governed
        ├── family: "rsi"
        ├── consumerSpec: A1 result
        ├── isEnabled: settings check
        └── startConsumer: closure
              ├── NewSignalConsumer() ← manual (NATS adapter)
              ├── reg.signal          ← manual (registry)
              ├── mapSignalRow()      ← manual (mapper)
              └── tracker/actor       ← manual (infrastructure)
```

### What is governed vs. manual

| Component | Owner | Rationale |
|-----------|-------|-----------|
| Consumer spec function signature | Codegen | Deterministic from spec |
| Consumer spec return values | Codegen | Deterministic from spec (durable, subject, event_type, stream) |
| AckWait / MaxDeliver constants | Codegen | Fixed policy (30s / 5), frozen in template |
| Pipeline entry metadata (family, names, table, SQL) | Codegen | Deterministic from spec |
| Pipeline entry `isEnabled` predicate | Codegen | Deterministic from spec (layer + family key) |
| Pipeline entry `startConsumer` closure structure | Codegen | Template-driven; references manual functions by name |
| `mapSignalRow` function | Manual | Requires domain knowledge of ClickHouse columns |
| `NewSignalConsumer` constructor | Manual | NATS adapter — infrastructure concern |
| `SignalRegistry` structure | Manual | Registry design — architectural concern |
| Event type (`signal.SignalGeneratedEvent`) | Manual | Domain modeling — permanent human decision |
| Tracker/actor wiring | Manual | Infrastructure — shared across all families |

## Boundary rules

### Rule 1: Generated code references manual code, not the reverse

The generated pipeline entry calls `mapSignalRow`, `NewSignalConsumer`, and `reg.signal` by name. These manual functions do NOT reference or depend on the codegen system. The dependency is one-directional:

```
codegen output → manual functions (references by name)
manual functions → (no codegen dependency)
```

### Rule 2: Markers delimit, not replace

The `codegen:begin`/`codegen:end` markers are Go comments. They have zero runtime impact. Removing them would not change behavior — but would break CI governance.

### Rule 3: The spec is the source of truth, not the target file

If a discrepancy is found between the spec-derived golden snapshot and the target file, the **golden snapshot wins**. The target file must be updated to match, not the other way around.

### Rule 4: Template changes affect all families

If the `pipeline_entry.go.tmpl` template changes, ALL families' golden snapshots must be regenerated and verified. This is enforced by `make codegen-check` (12 comparisons across 6 families × 2 artifacts).

### Rule 5: Only A1 + A2 are governed for this slice

The mapper, tests, config, and smoke phases for RSI remain manual. Codegen governance does not extend beyond the marked regions.

## Observability

The governed slice does not alter observability. The tracker, counter, and event recording logic inside the `startConsumer` closure is part of the generated template — but the tracker infrastructure itself is manual.

Metrics emitted by the RSI pipeline at runtime:
- `events_received` — incremented in the generated closure
- `events_flushed` — incremented by the inserter (manual)
- `events_dropped` — recorded by the inserter (manual)
- `buffer_depth` — tracked by the inserter (manual)

The generated closure's `tracker.RecordEvent()` and `tracker.Counter("events_received").Add(1)` calls are part of the template. Any change to the observability pattern requires a template change → full regeneration.

## Failure modes

### Scenario 1: Spec drift

Someone modifies `codegen/families/rsi.yaml` but does not regenerate the golden snapshot or update the target file.

**Detection**: `make codegen-check` fails (golden ≠ generated). CI blocks.

### Scenario 2: Target file manual edit

Someone edits the RSI consumer spec or pipeline entry directly in the target file without going through the codegen pipeline.

**Detection**: `make codegen-integrated` fails (target ≠ golden). CI blocks.

### Scenario 3: Template change without regeneration

Someone modifies a template but does not regenerate all golden snapshots.

**Detection**: `make codegen-check` fails for affected families. CI blocks.

### Scenario 4: Marker deletion

Someone accidentally removes the `codegen:begin`/`codegen:end` markers.

**Detection**: `make codegen-integrated` fails (region not found). CI blocks.

### Scenario 5: Golden snapshot manually edited

Someone edits a `.go.golden` file directly.

**Detection**: `make codegen-check` fails (golden ≠ generated from spec). CI blocks. Additionally, this should be caught in PR review — golden files should only change via regeneration.

## Limitations

1. **Golden-to-target comparison is structural, not byte-exact**: Whitespace and comment differences are tolerated. A subtle structural difference could theoretically pass normalization. Mitigated by compilation + tests + smoke.

2. **No automated insertion**: The developer must manually copy generated output into the target file at the correct location. This is the primary friction point. Future stages may introduce marker-based automated insertion.

3. **Single family only**: This slice proves the pattern for ONE family. Extrapolation to other families requires per-family integration (adding markers, manifest entries, and checks).

4. **No Tier 2 coverage**: Read-path artifacts (readers, handlers, routes) are not part of this slice.

5. **No mapper generation**: The `mapSignalRow` function remains manual. This is the artifact most likely to benefit from codegen next, but requires the `domain.columns` spec extension.

## What this slice proves

1. A codegen-generated artifact can participate in the real runtime pipeline without modification.
2. The governance mechanism (markers + golden comparison + CI) detects drift in both directions (spec→golden and golden→target).
3. The boundary between generated and manual code is explicit, auditable, and enforceable.
4. The integration adds zero runtime overhead — markers are comments, verification is CI-only.
5. The existing manual families are unaffected — the integration is additive and per-family opt-in.
