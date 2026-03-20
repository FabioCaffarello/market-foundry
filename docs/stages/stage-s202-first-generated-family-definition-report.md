# Stage S202 — First Generated Family Definition Report

> Status: Complete
> Date: 2026-03-20
> Predecessor: S201 (Generated/Manual Coexistence Hardening)
> Successor: S203 (First Generated Family Implementation)

## Executive Summary

Stage S202 selects and formally defines the first codegen-first family for the Market Foundry analytical pipeline. After evaluating all candidates against the frozen constraints from S193–S201, **EMA (Exponential Moving Average) signal family** was chosen as the optimal first codegen-first family. This choice maximizes proof value while minimizing implementation risk: EMA shares the signal layer with RSI (already governed since S200), reuses all existing infrastructure, requires zero new mapper code, and needs only a config line and smoke extension as manual work.

## Family Selected: EMA Signal

### Why EMA

1. **Same layer as RSI** — the signal layer is the most proven path; RSI governed fragments are already integrated and CI-validated.
2. **Shares all infrastructure** — `SIGNAL_EVENTS` stream, `signals` table, `NewSignalConsumer` factory, `reg.signal` registry, `mapSignalRow` mapper, `IsSignalFamilyEnabled` config method.
3. **Anticipated by design** — `"ema"` appears in `knownAbbreviations`, confirming the codegen engine was designed to handle this family.
4. **Minimal manual delta** — only A5 (config entry) and A6 (smoke extension) require new human-authored code.
5. **True codegen-first** — the spec YAML is authored before any implementation code exists; A1+A2 are generated, not retroactively governed.

### Why Not Others

- **Second decision/strategy/risk/execution family:** Would require new domain event types or new domain logic — too much manual work for a first iteration.
- **Evidence layer family:** Has special naming rules (evidence exception); should not be the first codegen-first proof.
- **New layer family:** Would require new DDL, new consumer factory, new registry — defeats the "minimal risk" constraint.
- **Existing family as codegen-first:** The existing 6 families are already implemented manually; retroactive governance was the S200 concern, not this stage.

## Generated vs Manual Coverage

| Artifact | Status | Owner | New Code? |
|----------|--------|-------|-----------|
| A1: Consumer spec | Generated | Machine | Yes — `WriterEMASignalConsumer()` |
| A2: Pipeline entry | Generated | Machine | Yes — pipeline struct literal |
| A3: Mapper | Reused | Human | No — `mapSignalRow` shared with RSI |
| A4: Mapper tests | Reused | Human | No — existing tests cover shared mapper |
| A5: Config entry | Manual | Human | Yes — 1 line in `writer.jsonc` |
| A6: Smoke test | Manual | Human | Yes — small extension of existing smoke |

**Generated artifacts:** 2 (A1, A2)
**Reused artifacts:** 2 (A3, A4)
**New manual artifacts:** 2 (A5, A6) — minimal effort

Full coverage details: [first-generated-family-generated-vs-manual-coverage.md](../architecture/first-generated-family-generated-vs-manual-coverage.md)

## Success Criteria

| ID | Criterion | Verification |
|----|-----------|-------------|
| SC-1 | Spec validates clean | `validate` + `validate-all` commands |
| SC-2 | Golden snapshots match | `check-all` — all families PASS |
| SC-3 | Code compiles | `go build` for writer + NATS adapter |
| SC-4 | Existing tests pass | `go test` — no regressions |
| SC-5 | Integrated check passes | `codegen-integrated-check.sh` exits 0 |
| SC-6 | Pipeline activates with config | Smoke test observes EMA actors |
| SC-7 | No manual edits to generated code | Code review confirms golden match |
| SC-8 | Manifest updated | 2 new entries in `integrated.yaml` |

## Risk Summary

| ID | Risk | Severity | Mitigation |
|----|------|----------|-----------|
| R1 | Fragment insertion error | Medium | RSI markers as reference; CI gates |
| R2 | Cross-family interference | Low | Independent struct literals; existing tests |
| R3 | Config omission | Low | Explicit checklist; smoke test |
| R4 | Overconfidence extrapolation | Medium | Non-goals explicitly documented |
| R5 | NATS durable collision | Low | Cross-spec validation |

Full risk model: [first-generated-family-success-criteria-risks-and-non-goals.md](../architecture/first-generated-family-success-criteria-risks-and-non-goals.md)

## Non-Goals

- Not full family generation (A1+A2 only)
- Not multi-family iteration
- Not cross-layer proof
- Not new mapper generation proof
- Not template changes
- Not new table proof
- Not performance benchmarking
- Not automatic insertion

## What This Iteration Proves

1. **Spec-first authorship works** — YAML spec written before implementation.
2. **Same-layer scaling works** — a second family reuses all infrastructure.
3. **Shared mapper pattern works** — `mapSignalRow` handles any signal family.
4. **CI governance scales** — drift detection covers multiple governed families in the same target files.
5. **Codegen-first is viable** — the generated path produces correct, compilable, runtime-ready code from scratch.

## What This Iteration Does NOT Prove

- Cross-layer generation (different table, different domain type)
- New mapper generation (A3 artifact)
- Evidence layer codegen-first (special naming rules)
- Tier 2 read-path generation
- Families requiring new DDL

## Deliverables

| # | Deliverable | Status |
|---|------------|--------|
| 1 | [first-generated-family-definition.md](../architecture/first-generated-family-definition.md) | Complete |
| 2 | [first-generated-family-generated-vs-manual-coverage.md](../architecture/first-generated-family-generated-vs-manual-coverage.md) | Complete |
| 3 | [first-generated-family-success-criteria-risks-and-non-goals.md](../architecture/first-generated-family-success-criteria-risks-and-non-goals.md) | Complete |
| 4 | This report | Complete |

## Acceptance Criteria Verification

| Criterion | Met? |
|-----------|------|
| First generated family formally defined | Yes — EMA signal, spec frozen |
| Generated vs manual coverage explicit | Yes — A1+A2 generated, A3+A4 reused, A5+A6 manual |
| Boundaries remain clear | Yes — machine-owned markers, human-owned manual artifacts |
| First iteration risks delimited | Yes — 5 risks, 5 failure modes, response plans |
| Base ready for S203 implementation | Yes — spec, targets, manifest structure, success criteria defined |

## Preparation for S203

The following tasks are ready for S203 execution:

1. **Author spec file:** Write `codegen/families/ema.yaml` with the frozen spec defined above.
2. **Generate golden snapshots:** Run codegen engine to produce `codegen/golden-snapshots/ema/consumer_spec.go.golden` and `pipeline_entry.go.golden`.
3. **Validate:** Run `validate`, `validate-all`, `check-all` to confirm correctness.
4. **Insert fragments:** Place generated code into target files with `codegen:begin`/`codegen:end` markers.
5. **Update manifest:** Add 2 entries to `codegen/integrated.yaml`.
6. **Add config:** Add `"ema"` to `signal_families` in `deploy/configs/writer.jsonc`.
7. **Extend smoke:** Add EMA pipeline activation check to smoke test.
8. **Verify:** Run full CI chain — compilation, unit tests, codegen checks, integrated check, smoke.
9. **Measure:** Record time from spec authorship to passing CI as a baseline for future families.
