# Triggered vs Deferred Items Before Family 04

## Purpose

Explicit classification of all accumulated debts, frictions, and predicted triggers into two categories: items that have been **triggered** (require action) and items that remain **deferred** (tracked but not yet actionable). This document prevents silent accumulation and ensures nothing is hidden.

---

## 1. Triggered Items

Items that have been activated by evidence from Family 03 or prior stages.

### TRIG-1: Codegen Evaluation (D-4)

**Status: TRIGGERED — evaluation required, implementation deferred.**

- **Origin:** S167, committed trigger at Family 04.
- **Evidence:** 4 families produce ~800 lines of mechanically duplicated code across readers (552 lines), handler methods (~320 lines), and use cases (~240 lines). Pattern is ~80% identical per artifact.
- **Evaluation result:** Codegen is justified but not cost-effective until Family 06. At 4-5 families, manual duplication cost < template maintenance cost. At 6+ families, the equation inverts.
- **Required action:** Document this evaluation as D-4 resolved. Set new committed trigger: codegen implementation mandatory before Family 06.
- **Family 04 impact:** None. Family 04 proceeds without codegen.

### TRIG-2: Documentation Correction (PF-4/CI Smoke)

**Status: TRIGGERED — documentation is stale.**

- **Origin:** PF-4 flagged in Family 01, 02, and 03 validation documents as "no CI integration for analytical smoke test."
- **Evidence:** `.github/workflows/ci.yml` contains `smoke-analytical` job validating all 4 families. The gap was resolved (likely S166/hardening tranche) but documentation was not updated.
- **Required action:** Close PF-4 in all future documents. Note resolution date.
- **Family 04 impact:** None. CI is already operational.

---

## 2. Deferred Items — Tracked With Committed Triggers

Items with explicit activation conditions that have not yet been met.

### DEF-C1: Codegen Implementation

- **Trigger:** Family 06 boundary (6 analytical families in read path).
- **Rationale:** At 6 families, ~1200 lines of mechanical duplication. Template-based generation becomes cheaper than manual copy-paste-modify.
- **Scope:** Reader files, handler methods, use case files, contracts structs, route registrations.
- **Risk if deferred beyond trigger:** Increasing maintenance burden, higher chance of copy-paste errors.

### DEF-C2: Schema Coherence Compile-Time Verification

- **Trigger:** ~12 analytical tables (currently at 5 with pre-staged migration for family 05).
- **Rationale:** DDL ↔ mapper ↔ reader column alignment is currently review-enforced. At 12+ tables with 100+ columns, review-based verification becomes unreliable.
- **Risk if deferred beyond trigger:** Silent column mismatches between DDL, writer, and reader.

### DEF-C3: Handler File Split

- **Trigger:** Handler file exceeds ~600 lines (projected at Family 06).
- **Rationale:** Single file with 6+ methods becomes hard to navigate. Split by family or generate from template.
- **Dependency:** Likely resolved by DEF-C1 (codegen) if implemented first.

### DEF-C4: Friction Count Gate

- **Trigger:** >2 new frictions in any single family expansion.
- **Rationale:** S167 gate condition. Ensures pattern health is re-evaluated when friction accelerates.
- **Current status:** Family 03 introduced 2 new frictions (PF-3, PF-6), both low severity. Threshold not crossed.

---

## 3. Deferred Items — Tracked Without Committed Triggers

Items that are known debts but have no specific activation condition. They are tracked for awareness, not scheduled.

### DEF-U1: Filter Case-Sensitivity (PF-3 / PF-4)

- **Description:** Optional domain filters (`type`, `outcome`, `direction`) pass values to ClickHouse without case normalization or enum validation.
- **Severity:** Low.
- **Why deferred:** Consistent behavior across all families. No operational incidents. ClickHouse string comparison is case-sensitive by design.
- **Revisit condition:** If users report confusion or if a domain introduces case-variant values.

### DEF-U2: No Pagination (D-9)

- **Description:** All analytical queries bounded by `limit=500`, no cursor-based pagination.
- **Severity:** Low.
- **Why deferred:** Current data volumes don't warrant it. TTL is 90 days. No consumer has requested scan beyond 500 rows.
- **Revisit condition:** If analytical consumers need full historical scans or data volumes grow significantly.

### DEF-U3: NATS Consumer Lag Visibility (D-6)

- **Description:** No metrics or diagnostics for NATS JetStream consumer lag in the writer service.
- **Severity:** Medium.
- **Why deferred:** Writer health is visible through `statusz`/`diagz` endpoints. No lag-related incidents observed.
- **Revisit condition:** If write latency increases or data freshness becomes a concern.

### DEF-U4: Sticky Degradation Without Auto-Recovery (D-7)

- **Description:** Writer pipeline enters degraded state on persistent ClickHouse errors and does not auto-recover when ClickHouse returns.
- **Severity:** Medium.
- **Why deferred:** Manual restart resolves. ClickHouse downtime has been brief and infrequent.
- **Revisit condition:** If ClickHouse instability increases or if operational team requests auto-recovery.

### DEF-U5: Silent Mapper Fallbacks (D-10)

- **Description:** Writer mappers use fallback values (empty string, 0.0) for unparseable fields instead of rejecting the event.
- **Severity:** Low.
- **Why deferred:** Fallback behavior is intentional — analytical data is best-effort. Losing events is worse than storing partial data.
- **Revisit condition:** If analytical consumers depend on field completeness for correctness.

### DEF-U6: Backoff Jitter (D-5)

- **Description:** Writer retry uses exponential backoff without jitter, risking thundering herd on ClickHouse recovery.
- **Severity:** Low.
- **Why deferred:** Single writer instance. No thundering herd possible.
- **Revisit condition:** If writer scales to multiple instances.

### DEF-U7: Smoke Test JSON Content Verification (PF-6)

- **Description:** Smoke test validates HTTP response structure and row counts but not JSON column content correctness.
- **Severity:** Low.
- **Why deferred:** Unit tests verify JSON parsing. Smoke validates the full pipeline flow. Adding content assertions to smoke increases brittleness.
- **Revisit condition:** If JSON parsing bugs escape unit tests.

### DEF-U8: Consumer/Inserter Naming (H-4)

- **Description:** Writer internal types use generic names (`consumerActor`, `inserterActor`) instead of domain-specific names.
- **Severity:** Low.
- **Why deferred:** Names are correct for the generic pipeline design. Domain specificity comes from configuration, not type names.
- **Revisit condition:** If writer handles non-analytical events.

### DEF-U9: Metadata Validation (D-11)

- **Description:** Writer accepts and stores metadata JSON without schema validation.
- **Severity:** Low.
- **Why deferred:** Analytical layer stores data as-received from operational pipeline. Validation belongs upstream.
- **Revisit condition:** If metadata quality issues affect analytical consumers.

---

## 4. Resolved Items (Closed)

Items that were previously tracked and are now resolved.

| ID | Description | Resolution | Resolved At |
|----|-------------|-----------|-------------|
| D-1 | `parseEvidenceKeyParams` naming | Renamed to `parseAnalyticalKeyParams` | S172 (H-3) |
| D-2 | Struct-based DI | `AnalyticalHandlerDeps` / `AnalyticalFamilyDeps` | S172 (H-1) |
| D-3 | Smoke test extraction | `validate_analytical_family()` helper | S172 (H-2) |
| PF-4 | No CI for analytical smoke | `smoke-analytical` job in `.github/workflows/ci.yml` | S166/S172 |
| D-4 | Codegen evaluation | Evaluated at S178 — justified but deferred to Family 06 boundary | S178 |

---

## 5. Summary

| Category | Count | Blocking Family 04? |
|----------|-------|---------------------|
| Triggered (action required) | 2 | No |
| Deferred with committed trigger | 4 | No |
| Deferred without trigger | 9 | No |
| Resolved | 5 | — |
| **Total tracked** | **20** | **None blocking** |

**Net trajectory:** Debt count decreased from 14 (pre-hardening) to 11 (post-hardening) to the current inventory of 15 active items (2 triggered + 4 committed + 9 unscheduled). The increase from 11 to 15 reflects more granular tracking, not acceleration. No item is high-severity. No item blocks Family 04.
