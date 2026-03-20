# Generated Slice Validation Against Existing Families

> S196 deliverable. Documents the cross-family validation of Slice 01 (consumer spec + pipeline entry) against all 6 existing manual families.

## Objective

Prove that the codegen engine from S195 reproduces the 2 covered artifact types identically across all 6 existing families — not just the 2 baseline families (RSI, Paper Order) used in S195.

## Families Validated

| # | Family | Layer | S195 Baseline? | S196 Result |
|---|--------|-------|----------------|-------------|
| 1 | candle | evidence | No | PASS |
| 2 | rsi | signal | Yes | PASS |
| 3 | rsi_oversold | decision | No | PASS |
| 4 | mean_reversion_entry | strategy | No | PASS |
| 5 | position_exposure | risk | No | PASS |
| 6 | paper_order | execution | Yes | PASS |

## Artifacts Validated Per Family

| Artifact | Template | Comparison Points | Result |
|----------|----------|-------------------|--------|
| A1: Consumer spec function | `consumer_spec.go.tmpl` | 6 families × 1 = 6 | 6/6 PASS |
| A2: Pipeline entry struct | `pipeline_entry.go.tmpl` | 6 families × 1 = 6 | 6/6 PASS |
| **Total** | | **12** | **12/12 PASS** |

## Validation Method

### Step 1: Spec Authoring
For each of the 4 families not previously covered (candle, rsi_oversold, mean_reversion_entry, position_exposure), a YAML spec was authored following the S193 frozen schema. Spec values were extracted directly from the live codebase.

### Step 2: Golden Snapshot Extraction
Consumer spec functions and pipeline entry structs were extracted verbatim from the live code:
- Consumer specs from `internal/adapters/nats/{layer}_registry.go`
- Pipeline entries from `cmd/writer/pipeline.go`

### Step 3: Template Generation
The codegen engine rendered each spec through both templates.

### Step 4: Structural Comparison
Generated output was compared against golden snapshots using the S194 normalization rules:
1. Strip single-line comments
2. Normalize tabs to spaces
3. Trim whitespace per line
4. Remove empty lines
5. Line-by-line equality

### Step 5: Live-Code Cross-Check
Generated output was manually compared against the actual source files to identify cosmetic drift that normalization accepts but that would be visible in a diff.

## Cosmetic Drift Detected (Acceptable)

### D1: Comment Phrasing
- **Generated**: `// WriterCandleConsumer defines the durable consumer spec for writer consuming\n// candle evidence events.`
- **Live**: `// WriterCandleConsumer defines the durable consumer spec for writer consuming candle events.`
- **Impact**: None. Comments are stripped by normalization. The template uses `{family_name} {layer} events` while hand-crafted code varies in phrasing.
- **Severity**: INFO — cosmetic only.

### D2: Comment Line Decorations
- **Generated**: `// ── Evidence: candle → evidence_candles ──`
- **Live**: `// ── Evidence: candle → evidence_candles ──────────────────`
- **Impact**: None. Trailing dash count differs. Stripped by normalization.
- **Severity**: INFO — cosmetic only.

### D3: Paper Order Comment Multi-line
- **Generated**: Two-line comment following template pattern.
- **Live**: Multi-line comment with additional context about EXECUTION_EVENTS stream.
- **Impact**: None. Comments are stripped. The extra context is in the registry file, not in the consumer spec function body.
- **Severity**: INFO — cosmetic only.

## Structural Drift Detected

**None.** All 12 comparison points are structurally equivalent after normalization.

## Evidence Layer Exceptions Validated

The evidence layer (candle family) exercises all 3 documented exceptions:
1. **Consumer name omits layer**: `writer-candle-consumer` (not `writer-evidence-candle-consumer`) ✓
2. **Function name omits layer**: `WriterCandleConsumer` (not `WriterCandleEvidenceConsumer`) ✓
3. **IsEnabled omits layer prefix**: `p.IsFamilyEnabled("candle")` (not `p.IsEvidenceFamilyEnabled("candle")`) ✓

## Known Abbreviation Handling Validated

| Family | Input | PascalCase | Status |
|--------|-------|------------|--------|
| rsi | `rsi` | `RSI` | ✓ (known abbreviation) |
| rsi_oversold | `rsi_oversold` | `RSIOversold` | ✓ (known abbreviation + standard) |
| mean_reversion_entry | `mean_reversion_entry` | `MeanReversionEntry` | ✓ (standard) |
| position_exposure | `position_exposure` | `PositionExposure` | ✓ (standard) |
| paper_order | `paper_order` | `PaperOrder` | ✓ (standard) |
| candle | `candle` | `Candle` | ✓ (standard) |

## Files Created/Modified

### New Spec Files
- `codegen/families/candle.yaml`
- `codegen/families/rsi_oversold.yaml`
- `codegen/families/mean_reversion_entry.yaml`
- `codegen/families/position_exposure.yaml`

### New Golden Snapshots
- `codegen/golden-snapshots/candle/{consumer_spec,pipeline_entry}.go.golden`
- `codegen/golden-snapshots/rsi_oversold/{consumer_spec,pipeline_entry}.go.golden`
- `codegen/golden-snapshots/mean_reversion_entry/{consumer_spec,pipeline_entry}.go.golden`
- `codegen/golden-snapshots/position_exposure/{consumer_spec,pipeline_entry}.go.golden`

### Modified Test Files
- `codegen/render_test.go` — Added 8 per-family golden tests + `TestCheckAllFamilies` cross-validation gate

### CI / Tooling
- `Makefile` — Added `codegen-check` and `codegen-test` targets
- `.github/workflows/ci.yml` — Added `codegen-golden` CI job

## Conclusion

The codegen engine's Slice 01 (consumer spec + pipeline entry) reproduces all 6 existing families with structural equivalence. The engine is validated across the full complexity spectrum: evidence (layer exception), signal (known abbreviation), decision (compound abbreviation), strategy (multi-word), risk (multi-word), execution (multi-word + ceiling complexity).
