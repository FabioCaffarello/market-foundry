# Manual-to-Generated Equivalence on Current Families

> S261 — Validating that the codegen generated path reproduces the structural
> recorte of real, production-integrated families without semantic drift.

## 1. Purpose

The generated path (codegen specs + templates + golden snapshots) has been expanding
since S200. Until now, correctness was validated against golden snapshots — static
files maintained alongside the codegen tooling itself.

This document validates the generated path against **real production artifacts** in
the current repository state: the manually-written code that is live, compiles,
passes CI, and runs in the writer pipeline.

## 2. Families Selected for Comparison

All 10 tier-1 families were included — this is not a sample, it is the full
population of codegen-governed families:

| Family | Layer | Codegen Since |
|---|---|---|
| candle | evidence | S260 |
| rsi | signal | S200 |
| ema | signal | S203 |
| rsi_oversold | decision | S260 |
| ema_crossover | decision | S260 |
| mean_reversion_entry | strategy | S260 |
| trend_following_entry | strategy | S260 |
| position_exposure | risk | S260 |
| drawdown_limit | risk | S260 |
| paper_order | execution | S260 |

## 3. Artifact Types Under Equivalence

### 3.1 Codegen-Governed Artifacts (2 per family = 20 total)

| Artifact | Template | Target File |
|---|---|---|
| consumer_spec | `consumer_spec.go.tmpl` | `nats{layer}/registry.go` |
| pipeline_entry | `pipeline_entry.go.tmpl` | `cmd/writer/pipeline.go` |

### 3.2 Manual Artifacts Compared (coexisting alongside codegen)

| Artifact Type | Count | Location | Pattern Repeatable? |
|---|---|---|---|
| Store consumer specs | 10 | `nats{layer}/registry.go` | YES — `NewConsumerSpec(durable, subject, type, stream)` |
| Writer starters | 6 (per-layer) | `writerpipeline/support.go` | YES — identical closure structure |
| Writer mappers | 6 (per-layer) | `writerpipeline/support.go` | YES — `[]any` from event fields |
| Config methods | 6 (per-layer) | `settings/schema.go` | YES — identical `for-range` loop |
| Registry struct + defaults | 6 (per-layer) | `nats{layer}/registry.go` | Partially — layer-specific fields |
| Domain event types | 8 | `domain/{layer}/events.go` | YES — standard Metadata + payload |
| ClickHouse DDL | 6 (per-layer) | `deploy/migrations/*.sql` | YES — deterministic schema |

## 4. Equivalence Validation Method

### 4.1 Three-Layer Verification

1. **Golden Snapshot Match** (`codegen check-all`): Spec + template output matches
   the golden snapshot file.

2. **Integrated Slice Match** (`codegen-integrated-check.sh`): The code between
   `codegen:begin`/`codegen:end` markers in production files matches the golden
   snapshot after structural normalization.

3. **Cross-Artifact Consistency** (`codegen-equivalence-check.sh`): The spec values
   (durable names, table names, column lists) are consistent with the manual
   artifacts they coexist with (store consumers, starters, mappers, config methods).

### 4.2 What Was Checked

| Check | Scope | Count |
|---|---|---|
| Golden snapshot equivalence | 10 families x 2 artifacts | 20 |
| Integrated slice equivalence | 10 families x 2 artifacts | 20 |
| Durable naming convention | 10 families | 10 |
| INSERT table alignment | 10 families | 10 |
| Column list alignment | 10 families | 10 |
| Store consumer coexistence | 10 families | 10 |
| Starter existence per layer | 6 layers | 6 |
| Mapper existence per layer | 6 layers | 6 |
| Config method per layer | 6 layers | 6 |
| Spec validation + cross-spec | 10 families | 10+1 |
| **Total checks** | | **109** |

## 5. Equivalence Results

**All 109 checks passed.** No drift, no warnings.

### 5.1 Structural Equivalence: Confirmed

For every codegen-governed family, the generated output is byte-identical (after
normalization) to the production code between markers.

### 5.2 Cross-Artifact Consistency: Confirmed

- Every spec `durable` matches the naming convention (`writer-{layer}-{family}`).
- Every spec `writer.table` matches the INSERT target in `pipeline.go`.
- Every spec `writer.columns` matches the column list in the production INSERT SQL.
- Every codegen writer consumer has a corresponding manual store consumer.

### 5.3 Manual Infrastructure Completeness: Confirmed

All 6 layers have the required manual infrastructure:
- A starter function in `writerpipeline/support.go`.
- A mapper function in `writerpipeline/support.go`.
- A config method in `settings/schema.go`.

## 6. What This Proves

1. **The codegen output is production-identical.** The generated path does not
   produce code that differs from what was manually written and integrated.

2. **The spec is the real source of truth.** Every value in the spec (durable,
   subject, type, stream, table, columns) is verifiably consistent with the
   production code it governs and the manual code it coexists with.

3. **No hidden manual overrides.** The code between codegen markers contains
   exactly what the templates produce — no manual edits were made to override
   generated content.

4. **The generated path is ready for expansion.** Since the 20 governed artifacts
   are proven equivalent to production, the same templates can reliably produce
   artifacts for new families.

## 7. What This Does NOT Prove

See [generated-equivalence-results-drift-and-limitations.md](generated-equivalence-results-drift-and-limitations.md)
for the explicit list of limitations and unproven claims.
