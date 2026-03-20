# Family 06 Lifecycle Record -- Aborted (No Viable Candidate)

**Stage range:** S189--S191
**Pattern:** Wave B v2 gate evaluation
**Predecessor:** Family 05 (Executions / paper_order)
**Outcome:** Manual expansion aborted. Codegen tranche initiated.

---

## Selection

### Trigger: Pre-Family-06 Hardening (S189)

Before Family 06 candidate evaluation, a mandatory hardening tranche was defined:

**Blocker H-5: Handler Parameter Extraction**
- `parseAnalyticalParams()` function to replace inline limit/since/until parsing in all 6 handler methods.
- Handler file target: <=501 lines (down from 615).
- All existing handler tests must pass without modification.

Non-blockers tracked: codegen scope definition, reader 10-parameter signature, smoke test line count.

### Candidate identification (S191)

All NATS registries, writer pipeline entries, domain types, and ClickHouse tables were inventoried.

| # | Candidate | Layer | Write Path Ready | Classification |
|---|-----------|-------|-----------------|---------------|
| A | EMA Crossover | L2 Signal | Missing pipeline entry | Partially ready |
| B | Venue Market Order | L6 Execution | Missing mapper + pipeline | Partially ready |
| C | Trade Burst | L1 Evidence | Missing everything | Uncovered |
| D | Volume Metrics | L1 Evidence | Missing everything | Uncovered |
| E | Observation Trades | L0 | Not analytical | Not applicable |

---

## Definition & Contract

### Gate conditions (S190, non-negotiable)

| ID | Condition |
|----|-----------|
| C1 | Candidate must NOT require write-path changes |
| C2 | Reader parameters must NOT exceed 11 |
| C3 | Family 06 must measure and report ceiling metrics |
| C4 | Codegen tranche must be scoped before Family 07 |

### Gate testing results

| Candidate | C1 (No write-path) | C2 (Params <=11) | Result |
|-----------|:---:|:---:|--------|
| EMA Crossover | FAILS (needs pipeline entry) | Passes (0 new params) | **Disqualified** |
| Venue Market Order | FAILS (needs mapper + pipeline) | Passes | **Disqualified** |
| Trade Burst | FAILS (full 9-artifact needed) | Unclear | **Disqualified** |
| Volume Metrics | FAILS (full 9-artifact needed) | Risk | **Disqualified** |
| Observation Trades | FAILS | N/A | **Disqualified** |

**All candidates fail C1. No viable candidate exists.**

---

## Implementation

No implementation occurred. Family 06 manual expansion was aborted.

---

## Validation

N/A -- no artifacts to validate.

---

## Runtime & Operability

N/A.

---

## Findings & Frictions

### Why abort was structurally inevitable

1. All 6 vertical layers already have analytical read-path coverage.
2. The readers are type-parameterized -- they already support querying any event type within their layer.
3. Every uncovered event type lacks a **writer pipeline entry** to persist data to ClickHouse.
4. Adding a writer pipeline entry is, by definition, a write-path change.

The analytical read-path is **more generic than the write-path**. Within-layer expansion is a write-side problem, not a read-side problem.

### Critical insight: EMA Crossover already works on the read path

`GET /analytical/signal/history?type=ema_crossover` returns 200 with empty results today. The `SignalReader` is type-parameterized. The bottleneck is entirely on the write side: no writer pipeline entry exists to persist EMA crossover events.

### What abort does NOT mean

- EMA Crossover is unimportant: it is a write-path enablement task, not an analytical expansion task.
- The analytical layer is finished: next improvements come from codegen or cross-family features.
- Wave B failed: it succeeded beyond expectations (6 families, full vertical coverage, proven pattern).
- Write-path work is blocked: writer can add pipeline entries independently of the analytical gate.

---

## Success Criteria & Blockers

### Deferred candidates

| Candidate | Proper scope | When |
|-----------|-------------|------|
| EMA Crossover | Writer pipeline extension (1 consumer + 1 pipeline entry) | When write-path extension is justified |
| Venue Market Order | Writer pipeline extension (new mapper + consumer + pipeline) | When venue execution flow is active |
| Trade Burst | Full new family -- codegen candidate | Post-codegen implementation |
| Volume Metrics | Full new family -- codegen candidate | Post-codegen implementation |

### Triggers for Family 07+

| Trigger | Condition | Unlocks |
|---------|-----------|---------|
| Codegen tranche implementation | Templates built and validated against 6 families | Generated families (F07+) |
| Writer pipeline extension | New consumer/pipeline entry for an event type | Data flow for codegen-generated reader |
| New vertical layer | Layer outside L1--L6 with analytical need | Manual or codegen expansion |

### Pattern conclusion

The manual Wave B expansion pattern is **retired** after 6 families. It delivered:
- 6 families with zero creative decisions across 5 expansions.
- Full vertical analytical coverage (evidence through execution).
- Complete specification for codegen (schema, mapper, reader, handler, test conventions documented across 289 tests).

Future analytical families are authorized only through codegen or exceptional manual expansion for genuinely new layers.
