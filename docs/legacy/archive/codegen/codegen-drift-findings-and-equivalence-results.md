# Codegen Drift Findings and Equivalence Results

> S196 deliverable. Objective measurement of drift between generated output and existing manual code.

## Equivalence Summary

| Metric | Value |
|--------|-------|
| Families validated | 6 / 6 |
| Artifacts per family | 2 (consumer_spec, pipeline_entry) |
| Total comparison points | 12 |
| Structural passes | 12 / 12 (100%) |
| Cosmetic drift instances | 3 (all INFO severity) |
| Dangerous drift instances | 0 |
| Blocking drift instances | 0 |

## Equivalence Matrix

| Family | consumer_spec | pipeline_entry |
|--------|:------------:|:--------------:|
| candle (evidence) | PASS | PASS |
| rsi (signal) | PASS | PASS |
| rsi_oversold (decision) | PASS | PASS |
| mean_reversion_entry (strategy) | PASS | PASS |
| position_exposure (risk) | PASS | PASS |
| paper_order (execution) | PASS | PASS |

## Drift Classification

### Severity Levels

| Severity | Definition | Action |
|----------|-----------|--------|
| CRITICAL | Structural divergence — wrong field, missing value, different behavior | Block merge, fix immediately |
| WARNING | Semantic divergence — different approach to same behavior | Review, decide case-by-case |
| INFO | Cosmetic divergence — visible but behaviorally identical | Log, no action required |

### Drift Instances Found

#### D1: Comment Phrasing Variation (INFO)

**Where**: Consumer spec function doc comments across all families.

**Generated pattern**:
```go
// Writer{Family}{Layer}Consumer defines the durable consumer spec for writer consuming
// {family_name} {layer} events.
```

**Live code pattern** (varies):
```go
// WriterRSISignalConsumer defines the durable consumer spec for writer consuming RSI signal events.
```

**Differences**:
- Template wraps to 2 lines; some live comments are single-line
- Template uses raw `family_name` (e.g., `rsi_oversold`); live code uses human-readable form (e.g., `RSI oversold`)
- Paper order live comment includes additional stream context

**Impact**: Zero — comments stripped by normalization. No behavioral difference.

**Decision**: Acceptable. Comments are human-authored context. The template produces structurally correct code; comment phrasing is cosmetic.

#### D2: Section Comment Dash Decoration Length (INFO)

**Where**: Pipeline entry section comments.

**Generated**: Fixed short suffix `──`
**Live**: Variable-length trailing dashes for visual alignment

**Impact**: Zero — comments stripped by normalization.

**Decision**: Acceptable. The dash length is purely aesthetic and has no bearing on correctness.

#### D3: Evidence Layer Comment Omits "evidence" (INFO)

**Where**: `WriterCandleConsumer` doc comment.

**Generated**: `candle evidence events`
**Live**: `candle events`

**Impact**: Zero — comments stripped.

**Decision**: Acceptable. The evidence layer exception in code is correctly handled; the comment variation is cosmetic.

## What the Engine Generates Correctly

For each comparison point, the following structural elements were validated:

### Consumer Spec Function (A1)
- ✓ Function name matches derived naming convention
- ✓ Durable consumer name matches spec
- ✓ NATS subject filter matches spec
- ✓ Event type matches spec
- ✓ Stream name matches spec
- ✓ AckWait = 30 * time.Second (hardcoded constant)
- ✓ MaxDeliver = 5 (hardcoded constant)
- ✓ Evidence layer exception (omit layer from function name)

### Pipeline Entry Struct (A2)
- ✓ family string matches spec
- ✓ consumerName matches derived naming
- ✓ inserterName matches derived naming
- ✓ table matches spec
- ✓ insertSQL matches derived INSERT INTO {table}
- ✓ consumerSpec call matches derived function name
- ✓ isEnabled lambda calls correct Is{Layer}FamilyEnabled method
- ✓ startConsumer lambda calls correct New{Layer}Consumer constructor
- ✓ startConsumer lambda uses correct registry field (reg.{layer})
- ✓ startConsumer lambda uses correct domain event type
- ✓ startConsumer lambda calls correct mapper function
- ✓ Evidence layer exception (omit layer from consumer/inserter names, IsEnabled)

## What the Engine Cannot Generate (Current Limits)

### Artifact Coverage Ceiling

| Artifact | Can Generate? | Blocking Reason |
|----------|:------------:|-----------------|
| A1: Consumer spec | ✓ Yes | — |
| A2: Pipeline entry | ✓ Yes | — |
| A3: Mapper function | ✗ No | Requires column-order knowledge, type transforms, DDL awareness |
| A4: Mapper unit tests | ✗ No | Depends on A3 |
| A5: Config entry | ✗ No | JSONC manipulation tooling |
| A6: Smoke test phase | ✗ No | Shell script generation |
| Tier 2 artifacts | ✗ No | Not authorized |

### Spec Completeness Ceiling

Current specs contain 14 fields covering NATS routing, writer declaration, and domain type references. To generate A3 (mappers), specs would need:
- `domain.columns` — ordered list of column definitions
- Per-column: `name`, `go_field`, `ch_type`, `transform` (optional)
- This is the `mapper: "generate"` extension defined in S193 but not yet implemented.

### File Integration Ceiling

The engine produces code fragments, not complete files. To integrate generated code into existing files (pipeline.go, registry files), the engine would need:
- Marker section detection
- Append/replace logic within marker boundaries
- This is explicitly deferred per S195 D5.

## Normalization Pipeline Validation

The S194 normalization rules were validated against real drift:

| Rule | Purpose | Exercised By |
|------|---------|-------------|
| Strip `//` comments | Ignore comment phrasing | D1, D2, D3 |
| Tab → space | Normalize indentation | All pipeline entries |
| Trim whitespace | Ignore trailing spaces | All comparisons |
| Remove empty lines | Ignore blank line differences | Consumer spec line breaks |

The normalization pipeline correctly distinguishes cosmetic from structural differences. No false positives or false negatives detected.

## Conclusion

The codegen engine achieves **100% structural equivalence** for its covered slice (A1 + A2) across all 6 existing families. All detected drift is cosmetic (comment text) and correctly handled by the normalization pipeline.

The engine is validated but **not yet sufficient** for autonomous family generation — it covers 2 of 6 Tier 1 artifacts. The mapper (A3) is the primary blocker for end-to-end write-path generation.
