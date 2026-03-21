# Signal Evolution Wave — Family Ordering and Acceptance Criteria

**Stage:** S283
**Date:** 2026-03-21
**Charter:** signal-evolution-wave-charter-and-scope-freeze.md

---

## 1. Recommended Delivery Order

### Order: MACD → ATR → Bollinger Squeeze → VWAP

| Position | Family | Layer | Rationale |
|----------|--------|-------|-----------|
| **1st** | MACD | Signal | Closest analog to existing EMA sampler; zero new evidence fields needed; proves signal codegen-first for a momentum indicator |
| **2nd** | ATR | Signal | Pure price-range calculation; no volume dependency; natural pairing with risk evaluators for dynamic stop-distance |
| **3rd** | Bollinger Squeeze | Decision | Consumes existing Bollinger signal (S262); first codegen-first decision evaluator; validates cross-layer composition |
| **4th** | VWAP | Signal | Requires volume field from candle evidence (already in spec); delivered last because it exercises the most cross-layer wiring |

### Order Rationale

1. **MACD first** — lowest risk, highest analogy to existing code. EMA calculation is already proven; MACD is EMA-derived. This family proves the repeatable codegen-first signal delivery pattern before tackling families with more wiring.

2. **ATR second** — still pure signal layer, but introduces volatility semantics. ATR's natural consumer is the risk layer (dynamic stop-distance), creating the first signal→risk composition path that doesn't exist today.

3. **Bollinger Squeeze third** — shifts to decision layer. This is the first decision evaluator delivered via codegen-first. It consumes an existing signal (Bollinger, S262), so the only new work is the decision evaluator itself plus its codegen artifacts.

4. **VWAP last** — most wiring surface. Volume-weighted calculation uses the volume field from candle evidence. While the field exists in the evidence spec, VWAP exercises the evidence→signal pipeline more thoroughly than MACD or ATR.

### Resequencing Rules

The order may be adjusted if:

- A blocking dependency is discovered during implementation (document in stage report)
- A family's implementation reveals a spec gap that another family would resolve first
- CI regression requires prioritizing a simpler family to stabilize

Resequencing does NOT require a new charter — only documentation in the affected stage report.

---

## 2. Dependencies Per Family

### 2.1 MACD (Signal)

| Dependency | Status | Notes |
|------------|--------|-------|
| Candle evidence (OHLCV) | Exists | Close prices sufficient for MACD |
| EMA calculation | Exists | `ema_crossover_sampler.go` has EMA logic |
| Signal NATS stream | Exists | `SIGNAL_EVENTS` stream, `signal.events.>` subject |
| Signal ClickHouse table | Exists | Schema proven with RSI, EMA, Bollinger |
| Codegen toolchain | Exists | `codegen` binary, templates, equivalence checks |

**New work:** YAML spec, golden snapshots, `macd_sampler.go`, behavioral tests, registry markers.

**External dependencies:** None.

### 2.2 ATR (Signal)

| Dependency | Status | Notes |
|------------|--------|-------|
| Candle evidence (OHLCV) | Exists | High, low, close prices needed |
| Signal NATS stream | Exists | Shared with other signals |
| Signal ClickHouse table | Exists | Schema proven |
| Codegen toolchain | Exists | |

**New work:** YAML spec, golden snapshots, `atr_sampler.go`, behavioral tests, registry markers.

**External dependencies:** None.

**Downstream opportunity:** ATR output can feed into `drawdown_limit_evaluator` or `position_exposure_evaluator` as a dynamic volatility input. This is a post-wave enhancement, not a wave requirement.

### 2.3 Bollinger Squeeze (Decision)

| Dependency | Status | Notes |
|------------|--------|-------|
| Bollinger signal | Exists | `bollinger_sampler.go` (S262) |
| Bollinger codegen family | Exists | `codegen/families/bollinger.yaml` |
| Decision NATS stream | Exists | `DECISION_EVENTS` stream |
| Decision ClickHouse table | Exists | Schema proven with RSI Oversold, EMA Crossover |
| Codegen toolchain | Exists | |

**New work:** YAML spec (decision layer), golden snapshots, `bollinger_squeeze_evaluator.go`, behavioral tests, registry markers.

**External dependencies:** None. Consumes existing Bollinger signal; no upstream changes.

### 2.4 VWAP (Signal)

| Dependency | Status | Notes |
|------------|--------|-------|
| Candle evidence (OHLCV + Volume) | Exists | Volume field in candle evidence spec |
| Signal NATS stream | Exists | |
| Signal ClickHouse table | Exists | |
| Codegen toolchain | Exists | |

**New work:** YAML spec, golden snapshots, `vwap_sampler.go`, behavioral tests, registry markers.

**External dependencies:** None. Volume already flows through candle evidence pipeline.

---

## 3. Acceptance Criteria Per Family

### 3.1 Universal Acceptance Criteria (All Families)

Every family must satisfy ALL of the following:

| # | Criterion | Verification |
|---|-----------|--------------|
| AC-1 | Codegen YAML spec exists and validates | `codegen validate-all` passes |
| AC-2 | Golden snapshots generated and match | `codegen check-all` passes |
| AC-3 | Integration markers in target files | `codegen-integrated-check.sh` passes |
| AC-4 | Full equivalence check green | `codegen-equivalence-check.sh` exits 0 |
| AC-5 | Application-layer implementation exists | File exists in `internal/application/{layer}/` |
| AC-6 | Behavioral test(s) in CI | `make test` or `make test-integration` executes them (zero skip) |
| AC-7 | ClickHouse schema defined | Migration or table definition present |
| AC-8 | Zero regression in existing tests | Full CI pipeline green |

### 3.2 Family-Specific Acceptance Criteria

#### MACD

| # | Criterion | Verification |
|---|-----------|--------------|
| MACD-1 | Computes MACD line (fast EMA - slow EMA) | Unit test with known input/output |
| MACD-2 | Computes signal line (EMA of MACD line) | Unit test with known input/output |
| MACD-3 | Publishes to `signal.events.macd` | Integration test or registry check |
| MACD-4 | Severity scaling follows established pattern | Behavioral test analogous to Bollinger severity test |

#### ATR

| # | Criterion | Verification |
|---|-----------|--------------|
| ATR-1 | Computes true range (max of H-L, |H-Cp|, |L-Cp|) | Unit test with known input/output |
| ATR-2 | Computes ATR as smoothed average of true range | Unit test with configurable period |
| ATR-3 | Publishes to `signal.events.atr` | Integration test or registry check |
| ATR-4 | Severity scaling follows established pattern | Behavioral test |

#### Bollinger Squeeze

| # | Criterion | Verification |
|---|-----------|--------------|
| BSQ-1 | Detects squeeze condition (bandwidth below threshold) | Unit test with known Bollinger values |
| BSQ-2 | Produces decision event from existing Bollinger signal | Integration test proving signal→decision flow |
| BSQ-3 | Publishes to `decision.events.bollinger_squeeze` | Integration test or registry check |
| BSQ-4 | Does NOT modify existing Bollinger signal family | Bollinger golden snapshots unchanged |

#### VWAP

| # | Criterion | Verification |
|---|-----------|--------------|
| VWAP-1 | Computes cumulative volume-weighted typical price | Unit test with known OHLCV input |
| VWAP-2 | Resets correctly on session boundary | Unit test proving reset behavior |
| VWAP-3 | Publishes to `signal.events.vwap` | Integration test or registry check |
| VWAP-4 | Consumes volume field from candle evidence | Evidence→signal flow test |

---

## 4. Multi-Symbol as Acceptance Criterion

Per S281 directive, multi-symbol is a **validation criterion**, not a delivery wave.

| Rule | Detail |
|------|--------|
| Each new family must pass the existing multi-symbol smoke script | `scripts/smoke-analytical-e2e.sh` or equivalent |
| No family may hardcode a single symbol | Code review criterion; no `if symbol == "X"` patterns |
| JetStream subject isolation (wildcard `>`) is sufficient | No new stream topology needed |
| Multi-symbol scaling optimization is out of scope | Post-wave gate concern |

---

## 5. Stage Mapping (Recommended)

| Stage | Content | Gate |
|-------|---------|------|
| S284 | MACD signal family delivery | Family gate: AC-1 through AC-8 + MACD-1 through MACD-4 |
| S285 | ATR signal family delivery | Family gate: AC-1 through AC-8 + ATR-1 through ATR-4 |
| S286 | Bollinger Squeeze decision family delivery | Family gate: AC-1 through AC-8 + BSQ-1 through BSQ-4 |
| S287 | VWAP signal family delivery | Family gate: AC-1 through AC-8 + VWAP-1 through VWAP-4 |
| S288 | Post-Signal-Evolution-Wave gate | Wave success criteria from charter §7 |

Interleaved observability (Prometheus minimal) enters with S284 (pipeline counters) and closes with S288 (control gate gauge).

### Stage Atomicity Rule

Each family stage is atomic: it either delivers the complete family (all 8 acceptance criteria + family-specific criteria) or it does not close. Partial delivery creates a scope leak that this charter prohibits.

---

## 6. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| MACD implementation discovers EMA logic needs refactoring | Low | Medium | EMA logic is proven in ema_crossover_sampler; MACD reuses, doesn't replace |
| Bollinger Squeeze requires Bollinger signal schema change | Low | High | Squeeze reads bandwidth from existing Bollinger output; no schema mutation |
| VWAP session reset logic is ambiguous | Medium | Low | Define reset boundary in YAML spec before implementation |
| New families increase CI time significantly | Low | Medium | Each family adds ~2-3 tests; total increase bounded |
| Codegen YAML schema needs new fields | Medium | Low | Minor additions permitted per charter §4.2; must not break existing families |

---

## 7. What Happens After This Wave

The post-wave gate (S288) will assess:

1. Whether the codegen-first pipeline scales to 15+ families reliably
2. Whether interleaved observability is sufficient or needs dedicated investment
3. Whether multi-symbol optimization should become a wave
4. Which candidate wave to open next (from S281 options matrix, minus Signal Evolution)
5. Whether any debts accumulated during the wave require remediation before the next wave
