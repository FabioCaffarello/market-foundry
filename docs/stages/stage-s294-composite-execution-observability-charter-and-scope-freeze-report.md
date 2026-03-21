# Stage S294 — Composite Execution Observability Charter and Scope Freeze Report

**Stage:** S294
**Type:** Charter and scope freeze
**Status:** COMPLETE
**Predecessor:** S293 (Post-Squeeze Vertical Slice Gate — PASS)

---

## 1. Executive Summary

Stage S294 opens the Composite Execution Observability Wave, defining its scope, boundaries, capabilities, delivery order, and explicit non-goals. The wave increases a systemic capability of the Foundry — legibility, explainability, and end-to-end operability — rather than expanding domain families.

The Foundry has proven three independent vertical slices (EMA, Trend, Squeeze) with CorrelationID/CausationID flowing through all five layers. Individual domain readers exist for every layer in ClickHouse. What is missing is the ability to **compose** these individual views into a single, coherent explanation of why a specific execution happened, was blocked, or was modified.

This wave closes that gap with five focused blocks: correlation spine validation, composite read model, explainability query surface, attribution of blockages/rejections, and a post-wave gate.

---

## 2. Analysis Performed

### Codebase analysis

- **Correlation infrastructure:** CorrelationID and CausationID are embedded in every event across all five domain layers (signal, decision, strategy, risk, execution). Propagation flows through actor messages (`messages.go`), publisher actors, NATS JetStream events, and ClickHouse rows.
- **Individual readers:** Six ClickHouse readers exist (candle, signal, decision, strategy, risk, execution), each supporting type/source/symbol/timeframe/time-range filters. All return domain objects with query timing metadata.
- **HTTP analytical surface:** Six GET endpoints on the gateway expose individual domain queries. No composite or cross-domain endpoints exist.
- **Publisher counters (S292):** Domain counters at the publisher level track event counts by family/outcome. Available via `/statusz` endpoint. No cross-domain correlation or funnel metrics.
- **KV projection (S271):** Execution latest state projected to NATS KV with monotonicity guards. Read-on-demand, no history.
- **Domain models:** All domain types preserve causal metadata — decisions carry `signals` JSON, strategies carry `decisions` JSON, risk carries `strategies` JSON and `constraints` JSON, execution carries `risk` JSON.

### Gap analysis

| Gap | Impact | This wave addresses? |
|-----|--------|---------------------|
| No composite execution query (cross-table JOIN by correlation/causation) | Cannot explain why execution X happened in a single call | Yes (Block 2) |
| No attribution of rejections (which constraint blocked, why) | Cannot debug risk gate behavior without manual JSON inspection | Yes (Block 4) |
| No pipeline funnel metrics (conversion rates per stage per family) | Cannot assess pipeline efficiency or identify drop-off points | Yes (Block 4) |
| No pipeline health per symbol (staleness detection per layer) | Cannot detect when a symbol stops flowing through a specific layer | Yes (Block 3) |
| No confidence/severity flow tracing | Cannot see how values transform across the chain | Yes (Block 2) |
| No cross-symbol correlation | Cannot trace cross-asset effects | No (out of scope — requires portfolio layer) |
| No monitoring/alerting infrastructure | No production-grade observability | No (out of scope — premature) |

---

## 3. Wave Charter

### Governing questions (Q1–Q7)

The wave is designed to make seven specific operational questions answerable through documented, tested API endpoints:

1. **Q1:** Why was execution X submitted? What signal chain produced it?
2. **Q2:** Why was execution X rejected or modified by risk? What constraint triggered?
3. **Q3:** Which signals contributed to decision D? With what values?
4. **Q4:** What was the confidence/severity flow from signal through execution?
5. **Q5:** Why did symbol S stop receiving executions? Where did the pipeline break?
6. **Q6:** How many executions were blocked vs approved in period T? Why?
7. **Q7:** What is the conversion rate at each pipeline stage for family F?

### Wave blocks

| Block | Name | Purpose |
|-------|------|---------|
| 1 | Correlation/Causation Spine | Validate causal chain integrity across all three slices |
| 2 | Composite Execution Read Model | ClickHouse + application layer joining five domain tables |
| 3 | Explainability Query Surface | HTTP endpoints for composite execution explanations |
| 4 | Attribution of Blockages/Rejections | Structured extraction of risk constraint causation |
| 5 | Post-Wave Gate | Validate Q1–Q7 are answerable; zero regression |

### Constraint: read-side only

The wave is bounded to read-side capabilities. No write-side changes (new columns, new NATS subjects, modified event schemas) are permitted. The sole exception is adding ClickHouse indexes on existing columns for query performance.

---

## 4. Capabilities Delivered by Wave Completion

| # | Capability | Questions answered |
|---|------------|-------------------|
| 1 | Execution chain reconstruction | Q1, Q3, Q4 |
| 2 | Rejection and modification attribution | Q2 |
| 3 | Pipeline funnel metrics | Q6, Q7 |
| 4 | Pipeline health per symbol | Q5 |
| 5 | Confidence/severity flow tracing | Q4 |

See `composite-execution-observability-capabilities-and-non-goals.md` for detailed capability descriptions and validation criteria.

---

## 5. Explicit Non-Goals

Ten items are formally excluded from this wave:

| ID | Non-Goal | Reason |
|----|----------|--------|
| NG-1 | Monitoring and alerting infrastructure | Not in production; premature |
| NG-2 | Distributed tracing (OTEL/Jaeger) | CorrelationID/CausationID sufficient |
| NG-3 | Real-time streaming views | Pull-based by design |
| NG-4 | Dashboard or UI delivery | JSON API first; visualization is separate |
| NG-5 | Cross-symbol correlation | Requires portfolio layer |
| NG-6 | Historical replay/backtest | Consumes read model; separate feature |
| NG-7 | Venue readiness/compliance | Different domain entirely |
| NG-8 | New signal families | S294 directive: no new families this wave |
| NG-9 | Write-side schema changes | Read-side only wave |
| NG-10 | Codegen integration | Manual first; codegen later |

See `composite-execution-observability-capabilities-and-non-goals.md` for detailed non-goal rationale and revisit conditions.

---

## 6. Recommended Stage Sequence

```
S294  Charter and scope freeze                              ← this stage (COMPLETE)
S295  Correlation/causation spine validation and gap closure
S296  Composite execution read model
S297  Explainability query surface (HTTP endpoints)
S298  Attribution of blockages, rejections, and reductions
S299  Post-Composite-Observability-Wave gate
```

### Stage-level entry/exit conditions

| Stage | Entry condition | Exit condition |
|-------|----------------|----------------|
| S295 | S294 complete | Correlation chain validated for all 3 slices; gaps documented or closed |
| S296 | S295 complete; chain is intact | Composite query returns full chain for any execution_id; tested with 3 slices |
| S297 | S296 complete; read model works | HTTP endpoints return composite explanations; integration tests pass |
| S298 | S297 complete; endpoints work | Attribution queries return structured rejection/modification reasons; funnel metrics work |
| S299 | S298 complete; all capabilities delivered | Q1–Q7 answerable; zero regression; honest debt assessment |

---

## 7. Preparation for S295

S295 (Correlation/Causation Spine Validation) should:

1. **Write deterministic test data** covering all three slices (EMA, Trend, Squeeze) with known CorrelationIDs and CausationIDs.
2. **Query each ClickHouse table** for a specific CorrelationID and verify all expected events appear.
3. **Follow CausationID links** from execution → risk → strategy → decision → signal and verify the chain is complete and unbroken.
4. **Document any gaps** where correlation breaks (e.g., fan-out producing new IDs, publisher actors dropping metadata).
5. **Fix gaps** if they are minor (missing ID propagation); escalate if they require write-side changes.

**Expected deliverables:**
- `docs/architecture/correlation-causation-spine-validation-and-findings.md`
- Integration test(s) proving chain integrity
- `docs/stages/stage-s295-correlation-causation-spine-validation-report.md`

---

## 8. Stop Conditions

The wave MUST stop and reassess if:

1. A proposed change requires modifying the write-side event schema.
2. A proposed change requires new NATS subjects or JetStream consumers.
3. A proposed change requires new external dependencies.
4. A proposed change touches actor message flow or fan-out logic.
5. Implementation scope exceeds two new ClickHouse queries and two new HTTP endpoints per stage.

---

## 9. Relation to S263 Directive

S263 said: "Feature evolution, not more infrastructure."

This wave is consistent with S263:
- It builds **read-side features** on existing infrastructure.
- It adds **zero new infrastructure components**.
- It delivers **operator-facing capabilities** (explainability, attribution, funnel metrics).
- It is bounded to **5 stages** with clear entry/exit conditions.

The wave is not infrastructure — it is a feature that makes the existing infrastructure legible and explainable.

---

## 10. Artifacts Delivered

| Artifact | Path |
|----------|------|
| Wave charter and scope freeze | `docs/architecture/composite-execution-observability-wave-charter-and-scope-freeze.md` |
| Capabilities and non-goals | `docs/architecture/composite-execution-observability-capabilities-and-non-goals.md` |
| Stage report (this document) | `docs/stages/stage-s294-composite-execution-observability-charter-and-scope-freeze-report.md` |

---

## 11. Verdict

**S294: COMPLETE**

The Composite Execution Observability Wave is formally open with frozen scope. Five blocks are defined, seven governing questions are formalized, ten non-goals are explicit, and five stages (S295–S299) are ordered with entry/exit conditions. The wave is read-side only, builds on proven infrastructure, and is bounded against inflation toward a generic observability platform.
