# Wave B Family 02 — Decisions (RSI Oversold) — Definition

**Status:** Defined
**Stage:** S168
**Predecessor:** S167 (CONDITIONAL PASS)
**Family:** Decisions (RSI Oversold)
**Iteration:** Wave B, Family 02

---

## 1. Executive Summary

This document defines the second controlled Wave B family expansion: **Decisions (RSI Oversold)**. The S167 gate review authorized this family as the next iteration under strict conditions. This is the last expansion iteration before the mandatory hardening tranche (Family 03).

The Decisions family was selected because it introduces **two JSON-typed columns** (`signals` and `metadata`) compared to one in Signals, testing the pattern's ability to handle richer payloads without breaking boundaries, opcionalidade, or operational discipline.

---

## 2. Family Selection Rationale

### 2.1 Why Decisions (RSI Oversold)

| Criterion | Assessment |
|---|---|
| **Write path readiness** | `mapDecisionRow()` already exists and is active. Writer pipeline already consumes `decision.events.v1.rsi_oversold_evaluated`. Zero write-path changes required. |
| **Schema readiness** | Migration `003_create_decisions.sql` already applied. Table exists in ClickHouse. |
| **Domain complexity** | 10 domain columns vs 8 for Signals. Two JSON columns (`signals`, `metadata`) vs one. This is the minimum increase needed to prove JSON extensibility. |
| **Dependency chain** | Decisions depend on Signals, which depend on Evidence. The dependency chain is already proven end-to-end. |
| **NATS subject** | `decision.events.rsi_oversold.evaluated.<symbol>` — already producing events. |
| **S167 authorization** | Explicitly named as second family in S167 CONDITIONAL PASS verdict. |

### 2.2 Why Not Another Family

- **Strategy:** 3 JSON columns (`decisions`, `parameters`, `metadata`) + `direction` enum. Too much complexity increase for a single step.
- **Risk Assessment:** 5 JSON columns + `rationale` string. Would triple JSON column count vs Signals.
- **Execution:** 4 JSON columns + `side`/`status` enums + dual correlation/causation IDs. Most complex family — wrong candidate for controlled expansion.

### 2.3 Complexity Delta from Family 01 (Signals)

| Dimension | Signals (Family 01) | Decisions (Family 02) | Delta |
|---|---|---|---|
| Domain columns | 8 | 10 | +2 |
| JSON columns | 1 (`metadata`) | 2 (`signals`, `metadata`) | +1 |
| Enum-like columns | 1 (`type`) | 2 (`type`, `outcome`) | +1 |
| Float columns | 1 (`value`) | 1 (`confidence`) | 0 |
| Total SELECT columns | 8 | 10 | +2 |

The delta is minimal and controlled: +1 JSON column, +1 enum-like column, +1 numeric column (confidence replaces value semantically). This tests JSON array deserialization (`signals` is `[]SignalInput`) alongside JSON map deserialization (`metadata` is `map[string]string`).

---

## 3. What This Iteration Must Prove

1. **JSON array deserialization works in the read path.** The `signals` column stores `[]SignalInput` (JSON array of objects). The reader must deserialize this correctly. Signals only proved `map[string]string` deserialization.

2. **Two JSON columns don't break the pattern.** The mapper writes two `marshalJSON()` calls. The reader must parse both independently with appropriate fallbacks.

3. **Outcome filtering works as query parameter.** Decisions have a categorical `outcome` (triggered/not_triggered/insufficient). The endpoint should accept an optional `outcome` filter — the first family-specific query parameter beyond the shared key params.

4. **Constructor accumulation is manageable at 3 use cases.** Family 02 adds a third use case to `AnalyticalWebHandler`. This will confirm whether the current constructor pattern is viable or whether the struct-based DI (H-1) must be accelerated.

5. **The 9-artifact pattern scales without surprise.** Family 02 must produce exactly 9 artifacts, same template, same dependency chain. No structural deviation.

---

## 4. Payload JSON Analysis

### 4.1 `signals` Column — JSON Array

**Write path (existing):**
```go
marshalJSON(d.Signals) // []SignalInput → JSON string
```

**Stored format:**
```json
[{"type":"rsi","value":"28.5","timeframe":60}]
```

**Read path (new):**
```go
// Must deserialize JSON string → []SignalInput
// Fallback: empty slice on parse failure (mirrors metadata fallback pattern)
```

**Rationale for JSON array:** SignalInput is a heterogeneous collection (different signal types with different value semantics). A separate ClickHouse column per signal input would require schema changes for each new signal type. JSON array preserves schema stability at the cost of query opaqueness — acceptable since signal-level drill-down is served by the signals table directly.

### 4.2 `metadata` Column — JSON Map

**Write path (existing):**
```go
marshalJSON(d.Metadata) // map[string]string → JSON string
```

**Stored format:**
```json
{"threshold":"30","period":"14"}
```

**Read path (new):**
Identical pattern to Signal's metadata deserialization via `ParseMetadataJSON()`. No new code needed — reuse existing helper.

### 4.3 Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| JSON array parse failure on malformed data | Low | Low | Silent fallback to empty slice, matches metadata pattern |
| Large signal arrays degrade query perf | Very Low | Low | `signals` is not in WHERE clause; only deserialized post-fetch |
| Inconsistent JSON encoding between writer/reader | Low | Medium | Unit tests verify round-trip fidelity |

---

## 5. Domain Model Reference

From `internal/domain/decision/decision.go`:

```go
type Decision struct {
    Type       string            `json:"type"`       // "rsi_oversold"
    Source     string            `json:"source"`      // "binancef"
    Symbol     string            `json:"symbol"`      // "btcusdt"
    Timeframe  int               `json:"timeframe"`   // 60
    Outcome    Outcome           `json:"outcome"`     // triggered|not_triggered|insufficient
    Confidence string            `json:"confidence"`  // "0.85"
    Signals    []SignalInput     `json:"signals"`     // input signals
    Metadata   map[string]string `json:"metadata"`    // decision context
    Final      bool              `json:"final"`
    Timestamp  time.Time         `json:"timestamp"`
}

type Outcome string // "triggered" | "not_triggered" | "insufficient"

type SignalInput struct {
    Type      string `json:"type"`      // signal type
    Value     string `json:"value"`     // signal value
    Timeframe int    `json:"timeframe"` // signal timeframe
}
```

---

## 6. Constraints Inherited from S167

All S162/S167 constraints apply without exception:

- **C-1:** One family per iteration — this iteration covers Decisions only.
- **C-7:** No horizontal redesign — vertical expansion only.
- **C-9:** Additive only — never modify existing candle or signal artifacts.
- **S167-1:** Follow pattern v2 exactly (9 artifacts, CI gate, 5-point gate review).
- **S167-2:** Must pass its own gate review before Family 03 begins.
- **S167-3:** If >2 new frictions not in v2 emerge, expansion pauses.

---

## 7. Binding Pre-commitments for Family 03

Family 02 does NOT resolve any hardening debts. These remain committed for Family 03:

- **H-1:** Handler constructor → struct-based DI (`AnalyticalHandlerDeps`)
- **H-2:** Smoke test → extract `validate_analytical_family()` function
- **H-3:** Shared helpers → rename `parseEvidenceKeyParams` → `parseAnalyticalKeyParams`

If Family 02 reveals that any of these are blocking (not just friction), they may be pulled forward — but this requires explicit documented rationale and gate review acknowledgment.
