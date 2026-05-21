# CC-02 — Extensibility Success Criteria

> Stage S125 — Objective measurement framework for signal family extensibility

## 1. Purpose

CC-01 proved horizontal scaling (more symbols, zero code). CC-02 must prove **vertical extensibility** (new code path, disciplined absorption). This document defines the binary criteria by which that property is evaluated.

## 2. Extensibility Criteria

### EX — Structural Extensibility

| ID | Criterion | Measurement | Pass Condition |
|----|-----------|-------------|----------------|
| EX-01 | **Domain model unchanged** | `git diff` on `internal/domain/signal/signal.go` | Zero lines changed |
| EX-02 | **No new domain types needed** | New files in `internal/domain/signal/` | Zero new files |
| EX-03 | **Projection actor reused** | `SignalProjectionActor` serves `ema_crossover` without code changes | Zero changes to projection actor logic |
| EX-04 | **Consumer actor reused** | `SignalConsumerActor` serves `ema_crossover` without code changes | Zero changes to consumer actor logic |
| EX-05 | **Publisher actor reused** | `SignalPublisherActor` publishes `ema_crossover` with only registry routing changes | Only `specForType()` switch case added |
| EX-06 | **HTTP route reused** | `/signal/:type/latest` serves `ema_crossover` without new route files | Zero new route/handler files |
| EX-07 | **Stream reused** | `SIGNAL_EVENTS` stream absorbs new subject pattern without reconfiguration | Stream wildcard `signal.events.>` covers new family |

### RF — Registration Friction

| ID | Criterion | Measurement | Pass Condition |
|----|-----------|-------------|----------------|
| RF-01 | **New file count** | `git diff --stat` new files | ≤ 4 new files (sampler + sampler_test + sampler_actor + optional) |
| RF-02 | **Modified file count** | `git diff --stat` modified files | ≤ 8 modified files |
| RF-03 | **Lines of new application logic** | `wc -l` on `ema_crossover_sampler.go` | ≤ 120 lines (comparable to RSI's 113) |
| RF-04 | **Lines of new actor code** | `wc -l` on `ema_crossover_sampler_actor.go` | ≤ 80 lines (comparable to RSI sampler actor) |
| RF-05 | **Registration boilerplate per layer** | Lines added per modified file | ≤ 15 lines per registration site |
| RF-06 | **Time to implement** | Wall-clock from start of S126 to passing tests | Measurable, no target — baseline capture |

### PL — Playbook Adherence

| ID | Criterion | Measurement | Pass Condition |
|----|-----------|-------------|----------------|
| PL-01 | **Playbook 1 followed** | Implementation steps match `expansion-playbooks-refined.md` Playbook 1 sequence | All 7 layers touched in documented order |
| PL-02 | **Naming conventions met** | Family name, durable name, KV bucket, subjects follow `naming-conventions-for-domains-families-and-runtimes.md` | Zero naming violations |
| PL-03 | **Dependency graph validated** | `schema.go` cross-layer dependency for `ema_crossover` → `candle` | Validation rejects config with `ema_crossover` enabled but `candle` disabled |
| PL-04 | **Config activation works** | `configctl` draft → validate → compile → activate cycle with `ema_crossover` | Full lifecycle completes without error |

### PP — Pipeline Proof

| ID | Criterion | Measurement | Pass Condition |
|----|-----------|-------------|----------------|
| PP-01 | **Signal published** | NATS subject `signal.events.ema_crossover.generated.>` receives messages | ≥ 1 event observed after warm-up |
| PP-02 | **Signal projected** | KV bucket `SIGNAL_EMA_CROSSOVER_LATEST` contains entries | ≥ 1 entry per active (source, symbol, timeframe) |
| PP-03 | **Signal queryable** | `GET /signal/ema_crossover/latest?source=binancef&symbol=btcusdt&timeframe=300` | HTTP 200 with valid signal payload |
| PP-04 | **Coexistence with RSI** | RSI pipeline unaffected by `ema_crossover` activation | RSI signals continue flowing; RSI query returns valid data |
| PP-05 | **Healthz reports** | `/healthz` includes `ema_crossover` components | All `ema_crossover` actors report healthy |
| PP-06 | **Diagz visibility** | `/diagz` or `/statusz` shows `ema_crossover` family metrics | Counters visible for new family |

### GV — Governance Holds

| ID | Criterion | Measurement | Pass Condition |
|----|-----------|-------------|----------------|
| GV-01 | **arch-guard passes** | `raccoon-cli arch-guard` | Zero violations |
| GV-02 | **Unit tests pass** | `make test` | Zero failures |
| GV-03 | **Existing tests unbroken** | No modifications to existing test files | Zero changes to non-ema_crossover test files |
| GV-04 | **No cross-domain imports** | `ema_crossover` code does not import from other domain packages | Zero forbidden imports |

## 3. Minimum Viable Success

**All of the following must pass for CC-02 to be considered successful:**

- EX-01 through EX-07 (structural extensibility fully proven)
- RF-01 through RF-05 (registration friction within bounds)
- PL-01 through PL-04 (playbook adherence confirmed)
- PP-01 through PP-04 (pipeline proof end-to-end)
- GV-01 through GV-03 (governance holds)

**Desirable but not blocking:**

- PP-05, PP-06 (diagnostic visibility — may trigger CF-08 boilerplate debt)
- GV-04 (enforcement — may depend on raccoon-cli rule)
- RF-06 (time baseline — informational only)

## 4. Non-Success Indicators

The following outcomes would indicate extensibility problems:

| ID | Indicator | Implication |
|----|-----------|-------------|
| NS-01 | Domain model requires changes | Signal abstraction is not family-agnostic |
| NS-02 | Projection or consumer actor requires code changes | Store layer is not type-parameterized |
| NS-03 | New route/handler files needed | Query surface is not type-driven |
| NS-04 | > 5 new files required | Family overhead too high |
| NS-05 | > 10 modified files required | Registration spread too wide |
| NS-06 | Existing tests break | Family addition has coupling side-effects |
| NS-07 | Playbook steps don't match reality | Documentation drifted from code |

## 5. Deferred Debt Trigger Evaluation

CC-02 implementation will naturally evaluate whether deferred debts should fire:

| Debt | Trigger Condition | Expected Outcome |
|------|-------------------|------------------|
| **CF-08** (UseCase boilerplate) | Adding `ema_crossover` reveals duplicated client patterns | Likely triggers — signal client is family-agnostic already, but registration may expose boilerplate |
| **CF-03** (Correlation ID) | First new actor (`EMACrossoverSamplerActor`) | Evaluate — if correlation ID propagation is copy-pasted, trigger is confirmed |
| **CF-02** (Symbols endpoint) | N/A — not triggered by family addition | No trigger expected |
| **D4** (Composition root tests) | New wiring in derive/store supervisors | Evaluate — if supervisor wiring is error-prone, trigger is confirmed |

## 6. Baseline Capture

CC-02 will establish the first **family addition baseline**:

- **File count**: new + modified files
- **Line count**: new application logic + actor boilerplate + registration lines
- **Playbook fidelity**: steps followed vs. steps documented
- **Friction log**: any surprises, undocumented steps, or workarounds

This baseline becomes the reference for future family additions (CC-03+).
