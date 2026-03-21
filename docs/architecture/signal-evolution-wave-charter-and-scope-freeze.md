# Signal Evolution Wave — Charter and Scope Freeze

**Stage:** S283
**Date:** 2026-03-21
**Status:** FROZEN
**Gate dependency:** S281 (feature gate PASS), S282 (CI enforcement COMPLETE)

---

## 1. Strategic Context

The Signal Evolution Wave is the first true feature delivery wave in the history of market-foundry. All prior waves (breadth S200–S244, behavioral S249–S257, codegen re-entry S258–S263, paper execution S264–S269, operational proof S270–S282) built and hardened infrastructure. This wave consumes that infrastructure to deliver domain value.

**S281 scored Signal Evolution at 25/30** — the highest of six candidates — on grounds of:

- Maximum domain value (5/5): each family is an independently useful trading signal
- Maximum architectural pressure (5/5): validates the full eight-context composition chain
- Maximum infrastructure reuse (5/5): codegen-first proven with Bollinger (S262)
- Bounded regression risk (3/5): each family is additive, not mutative

**S282 closed the hard prerequisite** by eliminating 40 auto-skipping tests and establishing a non-skipping CI baseline with NATS service containers.

---

## 2. Wave Charter

### 2.1 Mission

Deliver 3 new signal families and 1 new decision family end-to-end through the codegen-first pipeline, proving that the infrastructure investment translates into repeatable, low-cost domain expansion.

### 2.2 Families In Scope (Frozen)

| # | Family | Layer | Type | Rationale |
|---|--------|-------|------|-----------|
| 1 | **MACD** | Signal | Sampler | Well-defined momentum indicator; natural crossover decision pair |
| 2 | **VWAP** | Signal | Sampler | Volume-weighted anchor; evidence-layer volume data already exists in candle spec |
| 3 | **ATR** | Signal | Sampler | Volatility measure; natural risk evaluator input for dynamic stop-distance |
| 4 | **Bollinger Squeeze** | Decision | Evaluator | Consumes existing Bollinger signal; first decision evaluator via codegen-first |

### 2.3 Minimum Viable Delivery Per Family

Each family must deliver ALL of the following to be considered complete:

1. **Codegen YAML spec** in `codegen/families/{family}.yaml`
2. **Golden snapshots** — `consumer_spec.go.golden` + `pipeline_entry.go.golden`
3. **Integrated markers** — `codegen:begin`/`codegen:end` in target files
4. **Application-layer implementation** — sampler or evaluator in `internal/application/{signal,decision}/`
5. **Behavioral tests** — severity scaling, cross-domain composition, round-trip
6. **ClickHouse schema** — table definition and `map{Layer}Row` integration
7. **Equivalence check PASS** — `codegen-equivalence-check.sh` green with new family
8. **CI green** — zero regressions in existing test baseline

### 2.4 What This Wave Is NOT

This wave is bounded feature delivery. It is explicitly **not**:

- A codegen framework expansion (no new artifact templates, no generator changes)
- An architectural redesign (all eight bounded contexts are mature per S281)
- A platform observability program (Prometheus is interleaved, see §3)
- A multi-symbol scaling initiative (single-symbol sufficient per family)
- A venue-readiness preparation (paper execution is the ceiling)
- An infrastructure hardening wave (operational proof is closed)

---

## 3. Interleaved Observability (Prometheus Minimal)

S281 approved minimal Prometheus metrics as a secondary direction, interleaved with feature delivery — not as a parallel wave.

### 3.1 Scope of Interleaved Observability

| Metric | Type | Where | When |
|--------|------|-------|------|
| Pipeline event counter | Counter | `cmd/writer/pipeline.go` | With first new family integration |
| Writer batch gauge | Gauge | `cmd/writer/pipeline.go` | With first new family integration |
| Control gate state | Gauge | `natsexecution/` | After all 4 families delivered |

### 3.2 What Observability Does NOT Include

- Tracing (OpenTelemetry spans)
- Dashboards (Grafana JSON)
- Alerting rules
- Custom histogram buckets
- Per-family cardinality explosion
- Dedicated observability stages or gates

### 3.3 Delivery Rule

Observability work is permitted only when it naturally accompanies a family delivery stage. It must never:

- Block a family delivery
- Consume more than 20% of a stage's effort
- Open its own stage or gate
- Introduce new dependencies (e.g., Prometheus client library is acceptable; OpenTelemetry SDK is not)

---

## 4. Scope Freeze Rules

### 4.1 What Is Frozen

1. **Family list**: exactly MACD, VWAP, ATR, Bollinger Squeeze — no additions until post-wave gate
2. **Delivery standard**: the 8-item checklist in §2.3 — no reductions
3. **Architecture**: all eight bounded contexts, actor topology, NATS stream layout, ClickHouse schema pattern — no changes
4. **Codegen boundary**: consumer_spec + pipeline_entry artifacts only — no new artifact types

### 4.2 What May Evolve Within Freeze

1. **Family delivery order** — may be resequenced based on implementation findings (see ordering document)
2. **Test count** — expected to grow as families add behavioral tests
3. **Codegen YAML fields** — minor additions to spec schema if required by a new family (must not break existing families)
4. **ClickHouse columns** — new tables for new families; existing tables untouched

### 4.3 Scope Inflation Protection

Any request to add scope must satisfy ALL of the following:

1. It is required for a frozen family to pass its acceptance criteria
2. It does not open a new bounded context or architectural layer
3. It does not introduce a dependency not already in `go.mod`
4. It can be delivered within the stage that surfaces the need
5. It is documented as a scope amendment in that stage's report

Requests that fail any criterion are deferred to post-wave gate assessment.

---

## 5. Single-Front Discipline

This wave follows the single-front principle established in S258 and reaffirmed in S281:

- **One active delivery stage at a time**
- **No parallel feature branches** across different families
- **Gate between families** — each family's completion is verified before the next begins
- **Regression gate** — every stage must close with `make test` + `make test-integration` green

---

## 6. Items Explicitly Out of Scope

| Item | Reason | When to Revisit |
|------|--------|-----------------|
| Multi-symbol as a wave | JetStream subject isolation already sufficient | Post-wave gate |
| Venue readiness | Paper execution barely proven (S269) | After ≥2 feature waves |
| Full observability platform | No operational pressure yet | Post-wave gate |
| Codegen expansion (new artifact types) | Side-effect only per S263 | Post-wave gate |
| Configuration infrastructure | No user-facing config surface yet | Post-wave gate |
| New strategy families | Current 2 strategies sufficient for signal validation | Post-wave gate |
| New risk families | Current 2 risk evaluators sufficient | Post-wave gate |
| Actor topology changes | Proven in S270–S276 | Not planned |
| Domain model changes | Stable since S241–S244 | Only if family requires |

---

## 7. Success Criteria for Post-Wave Gate

The Signal Evolution Wave is complete when:

1. All 4 frozen families (MACD, VWAP, ATR, Bollinger Squeeze) are delivered end-to-end
2. Each family passes all 8 acceptance items from §2.3
3. `codegen-equivalence-check.sh` passes with ≥15 integrated families (11 existing + 4 new)
4. Zero regressions in the pre-wave test baseline
5. Interleaved Prometheus metrics are operational (pipeline counter + writer gauge minimum)
6. Multi-symbol smoke script passes with at least one new family
7. No secondary fronts were opened during the wave

---

## 8. Relation to Prior Governance

| Prior Decision | How This Charter Honors It |
|----------------|---------------------------|
| S263: feature evolution, not infrastructure | First pure feature wave; zero infrastructure changes |
| S258: codegen-first as means, not end | Codegen serves family delivery; no codegen framework changes |
| S264: paper execution ceiling | No venue or real-order logic |
| S281: single-front discipline | One family at a time, gated |
| S282: non-skipping CI baseline | All new tests must execute in CI; zero auto-skip |
