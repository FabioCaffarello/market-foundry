# Wave B Pattern Hardening After Family-01

> Lessons learned from the first Wave B expansion (Signals/RSI) and adjustments applied to the pattern before the second family.

## Context

S164 expanded the analytical layer with the first Wave B family (Signals/RSI). S165 validated it end-to-end. Both stages produced friction findings (PF-1 through PF-6) that inform this hardening pass.

This document captures what the first iteration taught and what changed as a result.

## What Family-01 Confirmed

1. **The 9-artifact unit works.** All nine artifacts were delivered mechanically. No ambiguity about scope or completeness.
2. **Schema coherence is testable without ClickHouse.** Unit tests for mapper row length and query builder column count catch DDL drift at compile time.
3. **Write path was future-proof.** Zero changes required to the writer infrastructure when adding signals — the pipeline catalog pattern absorbed the new family cleanly.
4. **Observability parity is automatic.** The inserter/supervisor infrastructure provides counters, `/statusz`, and `/diagz` entries without per-family code.
5. **Additive-only (C-9) is viable.** No existing code was modified beyond additive registrations (routes, compose, handler).
6. **Smoke test extension is straightforward.** Adding Phase 5b for signals followed the candle smoke pattern exactly.

## Friction Findings and Dispositions

### PF-1: Naming Residue — `parseEvidenceKeyParams()`

**Finding:** The shared param parser in the analytical handler carries the name `parseEvidenceKeyParams()` even though the parameters (source, symbol, timeframe, limit, since, until) are universal across all analytical families.

**Disposition:** Accept at 2 families. Rename to `parseAnalyticalKeyParams()` at family 3.

**Rationale:** C-9 (additive only) prevents renaming within the same iteration that introduced the friction. The name is misleading but not broken.

### PF-2: Constructor Accumulation

**Finding:** `AnalyticalWebHandler` currently takes 2 reader arguments. At family 3, this becomes 3+, which is awkward.

**Disposition:** Switch to `AnalyticalHandlerDeps` struct at family 3.

**Rationale:** Struct-based DI is a one-line change per existing reader. Not justified at 2 families but warranted at 3.

### PF-3: Mechanical Duplication (~80%)

**Finding:** Reader adapter, use case, and handler code is ~80% identical between candle and signal paths.

**Disposition:** Accept through family 3. Evaluate codegen at family 4.

**Rationale:** Duplication is explicit and greppable. Premature abstraction would obscure the pattern more than the duplication costs.

### PF-4: No Signal-Type Validation

**Finding:** The signal endpoint accepts any `type` value. ClickHouse returns empty for unknown types rather than an error.

**Disposition:** Accept. Empty result for unknown type is correct behavior — the reader layer is a query pass-through, not a type registry.

### PF-5: Smoke Test Linear Growth

**Finding:** Each family adds ~60 lines to `smoke-analytical-e2e.sh`. At 6 families, the script would exceed 800 lines.

**Disposition:** Extract `validate_analytical_family()` helper function at family 3.

**Rationale:** The function signature is clear from the first two families. Extracting at 2 would be premature (only one data point for parameterization).

### PF-6: No CI Integration — RESOLVED

**Finding:** Smoke-analytical ran only manually. This was the sole hard blocker before the second family per constraint C-3.

**Disposition:** **Resolved in S166.** GitHub Actions workflow created with unit test and smoke-analytical jobs. See `ci-smoke-analytical-integration.md`.

## Adjustments Applied to the Pattern

### A-1: CI Gate Added to Checklist

The Wave B checklist entry conditions now include:

> CI smoke-analytical passes on the branch before merge (required, not recommended).

This upgrades the S162 "recommended" to "required" based on the S165 finding that manual-only validation is insufficient for a repeatable process.

### A-2: Explicit Family Gate Criteria

The gate review (Step 8) now requires:

1. All unit tests pass (`make test`).
2. Smoke-analytical passes end-to-end (`make smoke-analytical`).
3. CI pipeline passes on the branch.
4. No regressions in existing families' smoke phases.
5. Schema coherence table documented for the new family.

### A-3: Documentation Artifact Clarified

Family documentation (artifact 9) must include:

- **Schema coherence table:** DDL column ↔ mapper field ↔ reader column alignment.
- **Endpoint specification:** HTTP method, path, query parameters, response contract.
- **Known limits:** Any simplifications or deferred validations.
- **Friction log:** New frictions discovered during the iteration.

### A-4: Smoke Test Parameterization Threshold

When the third family is added, the smoke script MUST extract a reusable `validate_analytical_family()` function. This is a hard requirement at family 3, not optional.

### A-5: Constructor Refactor Threshold

When the third family is added, `AnalyticalWebHandler` MUST switch to struct-based dependency injection (`AnalyticalHandlerDeps`). This is a hard requirement at family 3, not optional.

## What Did NOT Change

- **9-artifact unit** — Still the canonical expansion unit. No artifacts added or removed.
- **Left-to-right dependency chain** — Still enforced. No step may begin before its predecessor completes.
- **One family per iteration** — Still non-negotiable.
- **Additive-only (C-9)** — Still enforced within each iteration.
- **Observability parity** — Still automatic via inserter/supervisor infrastructure.
- **Schema coherence rule** — Still blocking. Unit tests are the verification mechanism.

## Debts Carried Forward

| Debt | Priority | Trigger |
|---|---|---|
| Rename `parseEvidenceKeyParams()` | Low | Family 3 |
| Constructor → struct DI | Medium | Family 3 |
| Smoke test extraction | Medium | Family 3 |
| Codegen evaluation | Low | Family 4 |
| Backoff jitter in writer retry | Low | Not scheduled |
| Consumer lag visibility | Medium | Not scheduled |
