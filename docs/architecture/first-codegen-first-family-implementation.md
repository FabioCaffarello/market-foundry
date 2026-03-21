# First Codegen-First Family Implementation

## Family Selected: `bollinger` (Bollinger Bands — Signal Layer)

### Selection Rationale

The `bollinger` family was chosen as the first codegen-first family based on these criteria:

1. **Proven layer infrastructure** — The signal layer already has two established families (`rsi`, `ema`) with all infrastructure in place: starters, mappers, consumers, publishers, schema. Adding a new signal family exercises only the codegen path and domain logic, not new infrastructure.

2. **Shared structural artifacts** — All signal families share the same ClickHouse table (`signals`), NATS stream (`SIGNAL_EVENTS`), mapper (`mapSignalRow`), and starter (`NewSignalStarter`). The codegen-first family requires zero new infrastructure beyond what the spec generates.

3. **Simple, well-understood domain logic** — Bollinger Bands (SMA ± k×StdDev with %B output) is a standard technical indicator with well-known behavior, making the manual domain logic small and auditable.

4. **Low structural risk** — A signal family has no downstream dependents unless explicitly wired. The `bollinger` family does not affect any existing decision, strategy, or risk family. It can be enabled/disabled independently.

5. **High proof value** — This family proves that the codegen pipeline can bootstrap a family from spec-first (YAML → golden → markers → production) rather than manual-first (write code → retrofit spec).

### Implementation Flow (Codegen-First Order)

The critical difference from all 10 previous families is the **order of operations**:

| Step | Previous families (manual-first) | Bollinger (codegen-first) |
|------|----------------------------------|---------------------------|
| 1 | Write production code manually | Create YAML spec |
| 2 | Retrofit YAML spec to match | Generate golden snapshots from spec |
| 3 | Generate golden to prove equivalence | Insert generated code into production files |
| 4 | Validate no drift | Write manual domain logic |

### What the Spec Produces

From `codegen/families/bollinger.yaml`, the system deterministically derives:

| Derived Name | Value |
|---|---|
| ConsumerSpecFunc | `WriterBollingerSignalConsumer` |
| ConsumerName | `writer-signal-bollinger-consumer` |
| InserterName | `writer-signal-bollinger-inserter` |
| IsEnabledMethod | `IsSignalFamilyEnabled` |
| StarterFunc | `NewSignalStarter` |
| PackageAlias | `natssignal` |
| InsertSQL | `INSERT INTO signals (event_id, ...)` |

### Artifacts Created

**Codegen-governed (spec → generate → golden → markers):**
- `codegen/families/bollinger.yaml` — family spec
- `codegen/golden-snapshots/bollinger/consumer_spec.go.golden`
- `codegen/golden-snapshots/bollinger/pipeline_entry.go.golden`
- Marked region in `internal/adapters/nats/natssignal/registry.go`
- Marked region in `cmd/writer/pipeline.go`
- Entry in `codegen/integrated.yaml` manifest

**Manual (human-written, spec does not govern):**
- `internal/application/signal/bollinger_sampler.go` — domain logic
- `internal/application/signal/bollinger_sampler_test.go` — unit tests
- Registry entries: `BollingerGenerated`, `BollingerLatest`, `StoreBollingerSignalConsumer`
- Config registration: `knownSignalFamilies["bollinger"]`, `signalDependsOnEvidence["bollinger"]`

### Validation Results

| Check | Result |
|---|---|
| `codegen validate bollinger.yaml` | VALID |
| `codegen check-all` (22 artifacts) | 22/22 PASS |
| `codegen validate-all` (11 families) | 11 VALID, 0 collisions |
| `codegen-integrated-check.sh` | 22/22 PASS |
| `codegen-equivalence-check.sh` (7 phases) | 65/65 PASS |
| Bollinger sampler unit tests | 6/6 PASS |
| Full application test suite | PASS |
| Full shared test suite | PASS |
| `go build` all affected modules | Clean |
