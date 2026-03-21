# Cross-Symbol Contamination: Risks, Findings, and Remediation

> S301 deliverable — catalogs all contamination vectors identified during the isolation audit,
> their assessed severity, and the applied (or deferred) remediation.
> Date: 2026-03-21

---

## 1. Contamination Vectors Identified

### VECTOR-1: Correlation-Based Queries Without Symbol Filter (CRITICAL — FIXED)

**Location**: `composite_reader.go` — `querySignalByCorrelation`, `queryDecisionByCorrelation`, `queryStrategyByCorrelation`, `queryRiskByCorrelation`, `queryExecutionByCorrelation`

**Mechanism**: All five methods queried their respective tables using only `WHERE correlation_id = ?`. If two symbols ever shared a correlation_id (by collision, misconfiguration, or multi-leg strategy), the query would return a row from the wrong symbol.

**Impact**: A chain lookup for symbol A could include stages belonging to symbol B. Attribution would be computed from contaminated data. Q1–Q4 answers would be incorrect.

**Likelihood pre-fix**: LOW in current system (correlation_ids are generated per-chain with symbol context), but **architecturally unprotected** — no schema constraint or query guard prevented it.

**Remediation**: Added `AND symbol = ?` to all five query methods. Updated `QueryChainByCorrelationID` signature to require symbol. Applied in S301.

**Status**: FIXED.

---

### VECTOR-2: HTTP Single-Chain Endpoint Missing Symbol (CRITICAL — FIXED)

**Location**: `composite.go` — `GetChain` handler

**Mechanism**: `GET /analytical/composite/chain?correlation_id=...` accepted only correlation_id. No symbol parameter meant:
1. The use case had no symbol to pass to the reader.
2. Any user could query any symbol's data by correlation_id alone.
3. No access scoping at the API boundary.

**Impact**: API-level violation of symbol isolation principle. Even if the reader were fixed, the handler would pass empty symbol, defeating the filter.

**Remediation**: Added mandatory `symbol` query parameter. Missing symbol returns HTTP 400. Applied in S301.

**Status**: FIXED.

---

### VECTOR-3: Batch Enrichment Losing Symbol Context (HIGH — FIXED)

**Location**: `composite_reader.go` — `QueryChainsBatch`, line calling `QueryChainByCorrelationID`

**Mechanism**: The batch path correctly scoped the initial execution query by symbol, but the enrichment step called `QueryChainByCorrelationID(ctx, corrID)` without symbol. This meant the enrichment queries (5 per chain) ran without symbol filter.

**Impact**: In a multi-symbol dataset, if a correlation_id from symbol A's execution query also existed in symbol B's signals table (theoretical but not prevented), the enrichment would pull symbol B's signal into symbol A's chain.

**Remediation**: `QueryChainsBatch` now passes `symbol` to `QueryChainByCorrelationID`. Applied in S301.

**Status**: FIXED.

---

### VECTOR-4: Correlation ID Not Symbol-Scoped in Construction (MEDIUM — ACCEPTED)

**Location**: Write-side event production (out of S301 read-path scope)

**Mechanism**: Correlation IDs are generated as opaque strings (e.g., `"s296-full-001"`). They do not embed the symbol as a structural component.

**Impact**: Two symbols could theoretically receive the same correlation_id if the generation logic has insufficient entropy or if cross-symbol event production shares a generator. Post-S301, this is mitigated by the `AND symbol = ?` filter — even a collision would not cause contamination.

**Remediation**: No code change. The read-path fix (VECTOR-1) makes this a defense-in-depth concern, not a live risk. If multi-symbol event production is introduced, correlation_id uniqueness per symbol should be validated.

**Status**: ACCEPTED — risk mitigated by query-level fix.

---

### VECTOR-5: Event ID Collision Across Symbols (LOW — ACCEPTED)

**Location**: Write-side event production

**Mechanism**: Event IDs follow patterns like `"sig-" + correlationID`. If correlation_ids collide across symbols, event_ids would also collide.

**Impact**: Post-S301, queries filter by `correlation_id AND symbol`, so event_id collisions do not cause data mixing. However, if event_id is ever used as a standalone lookup key (not currently the case), this would be a vector.

**Remediation**: No code change needed. Monitor for any future event_id-only lookup paths.

**Status**: ACCEPTED — no standalone event_id lookup exists.

---

## 2. Vectors NOT Found (Confirmed Safe)

| Component | Why Safe |
|-----------|----------|
| Individual domain readers | Mandatory `BuildQuery()` enforces symbol in WHERE |
| Pipeline funnel queries | Explicit `symbol = ?` in every stage count |
| Disposition breakdown | Explicit `symbol = ?` in GROUP BY query |
| ClickHouse schema | Symbol is part of ORDER BY — efficient pruning |
| NATS KV buckets | Keyed by `source.symbol.timeframe` — inherently scoped |
| Config-driven activation | Symbols activated independently per config |
| Attribution computation | Pure read-side derivation from already-scoped chain data |

---

## 3. Residual Risks

| ID | Risk | Severity | Mitigation |
|----|------|----------|-----------|
| RR-1 | Correlation ID collision across symbols | Low | Query-level symbol filter blocks contamination |
| RR-2 | Future event_id-only lookup path | Low | No such path exists; monitor for additions |
| RR-3 | Write-side symbol mismatch (event tagged with wrong symbol) | Medium | Out of read-path scope; write-side validation is a separate concern |

---

## 4. Test Evidence

### Unit Tests (no ClickHouse required)

| Test | What It Proves |
|------|---------------|
| `TestGetCompositeChain_Single_MissingSymbol` | Use case rejects single-chain query without symbol |
| `TestCompositeGetChain_MissingSymbol` | HTTP handler returns 400 for missing symbol |
| All existing single-chain tests updated with `Symbol: "btcusdt"` | Backward compatibility maintained |

### Integration Tests (requireclickhouse)

| Test | Criterion |
|------|-----------|
| `TestCompositeReader_SymbolIsolation_SingleChain` (S301-ISO-1) | 3 symbols inserted; each queried independently; all 5 stages match requested symbol |
| `TestCompositeReader_SymbolIsolation_CrossSymbolBlocked` (S301-ISO-2) | btcusdt correlation_id queried with symbol=ethusdt returns 0 stages |
| `TestCompositeReader_SymbolIsolation_BatchScoping` (S301-ISO-3) | 3 symbols inserted; batch query for btcusdt returns only btcusdt chains |
| `TestCompositeReader_SymbolIsolation_Funnel` (S301-ISO-4) | Multi-symbol data; per-symbol funnel counts are accurate |
| `TestCompositeReader_SymbolIsolation_Dispositions` (S301-ISO-5) | Multi-symbol data; per-symbol disposition counts are independent |
