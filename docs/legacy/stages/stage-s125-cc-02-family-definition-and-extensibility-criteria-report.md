# Stage S125 — CC-02 Family Definition and Extensibility Criteria

> **Status**: Complete
> **Date**: 2026-03-19
> **Predecessor**: S124 (Post-Capability Readiness Review)
> **Successor**: S126 (CC-02 Minimal Implementation)

## 1. Executive Summary

Stage S125 formally defines CC-02: the addition of `ema_crossover` as a new signal family to the market-foundry monorepo. CC-02 is not a product expansion — it is a controlled test of the codebase's ability to absorb new code paths with low friction, high discipline, and without disrupting proven patterns.

CC-01 proved that the architecture handles horizontal scaling (N symbols) through configuration alone. CC-02 now tests the complementary property: **vertical extensibility** — adding a genuinely new family end-to-end with bounded effort and zero structural regression.

## 2. Family Chosen: `ema_crossover`

**Exponential Moving Average Crossover** was selected over `moving_average_crossover` (SMA), `macd`, and `bollinger_bands` based on:

1. **Right-sized complexity** — stateless after warm-up (like RSI), but introduces a two-parameter computation (fast/slow periods) and non-numeric signal value (`bullish`/`bearish`/`neutral`).
2. **Full layer exercise** — touches every layer RSI touches (sampler → actor → publisher → consumer → projection → query → config) without requiring architectural changes.
3. **Domain model stress test** — uses `Metadata` for multiple parameters and `Value` for categorical output, validating that the unified `signal.Signal` struct is genuinely family-agnostic.
4. **Dependency alignment** — depends only on `candle` evidence, requiring no new upstream families.

## 3. Scope and Flow

### 3.1 Estimated Footprint

| Metric | Target |
|--------|--------|
| New files | 3 (sampler, sampler_test, sampler_actor) |
| Modified files | 7 (schema, registry, publisher, kv_store, derive_supervisor, store_supervisor, config JSONC) |
| New application logic | ≤ 120 lines |
| New actor code | ≤ 80 lines |

### 3.2 Data Flow

```
candle evidence → EMACrossoverSamplerActor → SignalPublisherActor → SIGNAL_EVENTS stream
                                                                          ↓
GET /signal/ema_crossover/latest ← KV bucket ← SignalProjectionActor ← SignalConsumerActor
```

All infrastructure (stream, publisher actor, projection actor, consumer actor, HTTP route) is **reused** from the existing signal pipeline. Only the sampler computation and registration touchpoints are new.

## 4. Extensibility Criteria

### 4.1 Categories

- **EX (Structural Extensibility)** — 7 criteria verifying that existing components are reusable without modification
- **RF (Registration Friction)** — 6 criteria measuring the overhead of adding a family
- **PL (Playbook Adherence)** — 4 criteria confirming that documented playbooks match reality
- **PP (Pipeline Proof)** — 6 criteria verifying end-to-end data flow
- **GV (Governance Holds)** — 4 criteria ensuring no architectural regression

### 4.2 Minimum Viable Success

All EX, RF (01–05), PL, PP (01–04), and GV (01–03) criteria must pass. This totals **22 binary criteria**.

### 4.3 Non-Success Indicators

7 indicators that would signal extensibility problems: domain model changes needed, store actors require modification, new route files needed, excessive new/modified file counts, existing test breakage, or playbook/reality drift.

## 5. Limits and Non-Objectives

### 5.1 Explicitly Out of Scope

- **No downstream families** — no decision, strategy, risk, or execution families for `ema_crossover`
- **No domain model changes** — `signal.Signal` must be sufficient as-is
- **No new abstractions** — two families is insufficient basis for shared patterns
- **No deferred debt resolution** — debts are documented if triggered, not resolved
- **No infrastructure changes** — no new streams, runtimes, Docker Compose services, or CI changes
- **No product features** — no multi-timeframe correlation, alerting, or ensemble logic

### 5.2 Deferred Debt Monitoring

| Debt | Trigger Watch |
|------|--------------|
| CF-08 (boilerplate) | Registration patterns in 7 modified files |
| CF-03 (correlation ID) | Copy-paste patterns in new actor |
| D4 (composition root tests) | Untested wiring in supervisors |

**Rule**: Document trigger evidence; do not resolve in CC-02.

## 6. Preparation for S126

### 6.1 Implementation Order (follows Playbook 1)

1. **Domain** — Confirm `signal.Signal` needs no changes (EX-01 checkpoint)
2. **Settings** — Register `ema_crossover` in `knownSignalFamilies` and `signalDependsOnEvidence`
3. **NATS** — Add registry specs, consumer spec, bucket constant, publisher routing
4. **Application** — Implement `EMACrossoverSampler` with unit tests
5. **Actors/Derive** — Implement `EMACrossoverSamplerActor`, wire in `derive_supervisor.go`
6. **Actors/Store** — Wire pipeline entry in `store_supervisor.go`
7. **Config** — Update JSONC to include `ema_crossover` in `signal_families`

### 6.2 Validation Sequence

1. `make test` — unit tests pass, no regressions
2. `raccoon-cli arch-guard` — governance holds
3. Config lifecycle — draft → validate → compile → activate
4. Live pipeline — `ema_crossover` signals produced and queryable
5. Coexistence — RSI pipeline unaffected

### 6.3 Friction Capture Protocol

During S126 implementation, maintain a friction log capturing:
- Any step that was not covered by Playbook 1
- Any file that needed changes beyond the predicted 3+7 envelope
- Any boilerplate that felt duplicative (CF-08 evidence)
- Any correlation ID propagation that was copy-pasted (CF-03 evidence)
- Any surprising test failures or compilation errors
- Wall-clock time for implementation (RF-06 baseline)

## 7. Artifacts Produced

| # | File | Purpose |
|---|------|---------|
| 1 | `docs/architecture/cc-02-family-definition.md` | Family choice, component map, data flow, NATS topology |
| 2 | `docs/architecture/cc-02-extensibility-success-criteria.md` | 27 binary criteria across 5 categories |
| 3 | `docs/architecture/cc-02-scope-contracts-and-out-of-scope.md` | 14 in-scope deliverables, 5 out-of-scope categories, guard rails |
| 4 | `docs/stages/stage-s125-cc-02-family-definition-and-extensibility-criteria-report.md` | This report |

## 8. Decision Record

| Decision | Rationale |
|----------|-----------|
| `ema_crossover` over `moving_average_crossover` | EMA is stateless after warm-up (like RSI); SMA requires full-window buffer. EMA also tests two-parameter metadata and categorical value output. |
| `ema_crossover` over `macd` | MACD has three components (MACD line, signal line, histogram) — over-complex for a first extensibility test. MACD is better suited for CC-03 if CC-02 succeeds. |
| No downstream families | Each domain layer is an independent extensibility test. Bundling signal + decision + strategy would conflate the measurement. |
| No shared abstractions | Two examples (RSI + ema_crossover) is the minimum to observe a pattern, but not enough to validate an abstraction. If CC-03 adds a third family and the pattern holds, abstraction becomes justified. |
| Measure deferred debts, don't resolve | CC-02's purpose is extensibility proof, not debt reduction. Mixing objectives would compromise both measurements. |

## 9. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Domain model insufficient for categorical values | Low | High (NS-01) | `Value` is string-typed; categorical values are valid strings |
| Store projection needs type-specific logic | Very Low | Medium (NS-02) | Projection is already type-agnostic; only KV bucket differs |
| Publisher switch statement becomes unwieldy | Low | Low | Two cases (rsi, ema_crossover) is well within ergonomic bounds |
| Playbook steps outdated | Medium | Low | Document divergences; update playbook in friction capture |
| Scope creep into decision family | Medium | Medium | Hard scope boundary: zero downstream families |

## 10. Conclusion

CC-02 is defined, bounded, and measurable. The `ema_crossover` signal family is the right choice: complex enough to test real extensibility (two-parameter, categorical output, full pipeline), simple enough to remain a single-stage implementation. The 22 mandatory success criteria provide objective pass/fail evaluation. The friction capture protocol ensures that even if all criteria pass, the qualitative cost of family addition is recorded for future optimization.

**Next step**: S126 — Minimal implementation of `ema_crossover` following Playbook 1.
