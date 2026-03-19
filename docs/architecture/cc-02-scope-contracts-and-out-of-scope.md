# CC-02 — Scope Contracts and Out of Scope

> Stage S125 — Boundary discipline for the extensibility capability

## 1. In-Scope Contracts

### 1.1 Code Deliverables

| # | Deliverable | Layer | Status |
|---|------------|-------|--------|
| C-01 | `ema_crossover_sampler.go` — EMA computation + crossover detection | Application | New file |
| C-02 | `ema_crossover_sampler_test.go` — Unit tests for sampler | Application | New file |
| C-03 | `ema_crossover_sampler_actor.go` — Actor wrapper for sampler | Actors/Derive | New file |
| C-04 | `knownSignalFamilies["ema_crossover"]` registration | Settings | Modification |
| C-05 | `signalDependsOnEvidence["ema_crossover"]` dependency | Settings | Modification |
| C-06 | `SignalRegistry.EMACrossoverGenerated` EventSpec | Adapters/NATS | Modification |
| C-07 | `SignalRegistry.EMACrossoverLatest` ControlSpec | Adapters/NATS | Modification |
| C-08 | `StoreEMACrossoverSignalConsumer()` consumer spec | Adapters/NATS | Modification |
| C-09 | `specForType("ema_crossover")` routing in publisher | Adapters/NATS | Modification |
| C-10 | `LatestSpecByType("ema_crossover")` routing in registry | Adapters/NATS | Modification |
| C-11 | `SignalEMACrossoverLatestBucket` constant | Adapters/NATS | Modification |
| C-12 | `SignalFamilyProcessor` entry in `derive_supervisor.go` | Actors/Derive | Modification |
| C-13 | `Pipeline` entry in `store_supervisor.go` | Actors/Store | Modification |
| C-14 | Config JSONC updated with `ema_crossover` in `signal_families` | Deploy/Config | Modification |

### 1.2 Validation Deliverables

| # | Deliverable | Purpose |
|---|------------|---------|
| V-01 | `make test` passes | No regressions |
| V-02 | `raccoon-cli arch-guard` passes | Architecture governance holds |
| V-03 | Config lifecycle (draft → validate → compile → activate) works | Config-driven activation proven |
| V-04 | Live pipeline produces `ema_crossover` signals | End-to-end data flow confirmed |
| V-05 | Query endpoint returns `ema_crossover` data | Read path proven |

### 1.3 Documentation Deliverables

| # | Deliverable | File |
|---|------------|------|
| D-01 | Family definition | `cc-02-family-definition.md` (this stage) |
| D-02 | Success criteria | `cc-02-extensibility-success-criteria.md` (this stage) |
| D-03 | Scope contracts | `cc-02-scope-contracts-and-out-of-scope.md` (this stage) |
| D-04 | Implementation notes | `cc-02-implementation-notes.md` (S126) |
| D-05 | Friction capture | `cc-02-frictions-and-structural-findings.md` (S127+) |
| D-06 | Gains/tradeoffs/debts | `cc-02-gains-tradeoffs-and-open-debts.md` (S128+) |

## 2. Explicitly Out of Scope

### 2.1 Product Expansion — NOT CC-02

| Item | Why Out of Scope |
|------|-----------------|
| New decision family for `ema_crossover` | CC-02 tests signal extensibility only; downstream families are a separate wave |
| New strategy, risk, or execution families | Same reason — each domain family is an independent extensibility test |
| Multi-timeframe crossover correlation | Product feature, not extensibility proof |
| Signal combination or ensemble logic | Requires cross-family wiring not in scope |
| Alerting or notification integration | Product feature, not structural test |

### 2.2 Architectural Changes — NOT CC-02

| Item | Why Out of Scope |
|------|-----------------|
| Domain model modifications to `signal.Signal` | CC-02 must prove the existing model is sufficient (EX-01) |
| New shared abstractions for family registration | Would hide the real friction; measure boilerplate first, abstract later |
| Generic signal sampler interface/trait | Premature abstraction — two families is not enough signal |
| Publisher actor refactoring | Publisher already works via `specForType()` switch; refactor only if switch becomes unmanageable (3+ families) |
| Store projection generalization | Already type-agnostic; no change expected |

### 2.3 Deferred Debts — NOT CC-02 (Unless Triggered)

| Debt | Trigger Rule | Action if Triggered |
|------|-------------|-------------------|
| CF-03 (Correlation ID) | If `EMACrossoverSamplerActor` requires copy-pasted correlation ID code | Document friction; do NOT implement middleware in CC-02 |
| CF-08 (UseCase boilerplate) | If signal client registration reveals duplicated patterns | Document friction; do NOT refactor in CC-02 |
| CF-02 (Symbols endpoint) | Not triggered by family addition | No action |
| D4 (Composition root tests) | If supervisor wiring introduces untested code | Document friction; add test only for new wiring |

**Rule**: CC-02 documents friction from deferred debts but does **not** resolve them. Resolution is a separate stage (S128 or S129) following evidence capture.

### 2.4 Infrastructure — NOT CC-02

| Item | Why Out of Scope |
|------|-----------------|
| New NATS streams | `SIGNAL_EVENTS` already covers all signal families via wildcard |
| Docker Compose changes | No new runtimes introduced |
| CI/CD pipeline changes | Family addition should not require pipeline changes |
| Soak testing infrastructure | Deferred from CC-01; still no concrete trigger |
| ClickHouse integration | Product feature, orthogonal to extensibility |
| OpenTelemetry integration | Observability enhancement, not extensibility test |

### 2.5 Tooling — NOT CC-02

| Item | Why Out of Scope |
|------|-----------------|
| raccoon-cli rule changes | Unless arch-guard fails on valid code, no changes needed |
| Smoke test modifications | Only if needed for validation (V-04); minimal additions only |
| New CLI commands | No new operational surface |

## 3. Scope Boundary Rules

1. **If a change touches a file outside the 3-new + 7-modified envelope**, stop and evaluate whether the scope has leaked.
2. **If an existing test needs modification**, the change may be coupling-unsafe — investigate before proceeding.
3. **If a new abstraction is tempting**, document it as a friction finding instead. Two examples (RSI + ema_crossover) is not enough to justify an abstraction.
4. **If a deferred debt fires**, capture the evidence in `cc-02-frictions-and-structural-findings.md` and continue. Do not resolve in-flight.
5. **If the domain model seems insufficient**, this is a non-success indicator (NS-01). Document clearly and halt for architectural review.

## 4. Dependency Map

```
                    ┌──────────────┐
                    │   candle     │ (evidence family, already exists)
                    │  (evidence)  │
                    └──────┬───────┘
                           │ depends on
                    ┌──────┴───────┐
                    │ ema_crossover│ (NEW — CC-02)
                    │   (signal)   │
                    └──────────────┘
                           │
                           │ NOT in CC-02 scope
                           ▼
                    ┌──────────────┐
                    │  (decision)  │ ← future CC-03 or later
                    └──────────────┘
```

CC-02 adds exactly **one node** to the dependency graph. No downstream nodes are created.

## 5. Guard Rails Summary

- **Do not** open new domains, runtimes, or streams.
- **Do not** refactor existing patterns — measure them.
- **Do not** add downstream families (decision, strategy, risk, execution) for ema_crossover.
- **Do not** resolve deferred debts — only document their trigger evidence.
- **Do not** modify the signal domain model.
- **Do not** create shared abstractions from only two examples.
- **Do** follow Playbook 1 exactly as documented.
- **Do** capture every friction point encountered.
- **Do** measure the file/line/time cost of family addition.
