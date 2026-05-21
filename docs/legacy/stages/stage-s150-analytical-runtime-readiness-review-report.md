# Stage S150 — Analytical Runtime Readiness Review Report

## Stage identity

| Field | Value |
|-------|-------|
| Stage | S150 |
| Title | Post-analytical runtime entry readiness review |
| Scope | Formal review of S143–S149 as a single capability wave |
| Predecessor | S149 (historical query surface minimal extension) |
| Date | 2026-03-19 |

---

## 1. Objective

Evaluate whether the first entry of ClickHouse, migrations, writer, and historical query surface into market-foundry was successful, what risks and limits remain, and what should happen next.

This stage produces no code. It produces a review, an accounting of gains and debts, and a recommendation.

---

## 2. Deliverables

| # | Document | Path | Status |
|---|----------|------|--------|
| 1 | Readiness review | `docs/architecture/post-analytical-runtime-entry-readiness-review.md` | Delivered |
| 2 | Gains, trade-offs, and open debts | `docs/architecture/analytical-runtime-gains-tradeoffs-and-open-debts.md` | Delivered |
| 3 | Next wave recommendations | `docs/architecture/next-wave-recommendations-after-analytical-runtime-entry.md` | Delivered |
| 4 | Stage report (this document) | `docs/stages/stage-s150-analytical-runtime-readiness-review-report.md` | Delivered |

---

## 3. Executive Summary

The S143–S149 wave achieved its primary structural objectives. ClickHouse was introduced as an optional analytical projection layer without contaminating the operational baseline. The migration tool, core schema, writer service, and historical query endpoint form a coherent minimal skeleton.

The wave did not achieve production-grade reliability. The writer and reader paths have zero test coverage. Failure handling in the writer diverges from documented specifications. No observability exists beyond basic health checks. The writer supervisor cannot recover failed pipelines without a process restart.

**Verdict: the analytical runtime is a valid first projection. It is ready for hardening. It is not ready for expansion.**

---

## 4. Review Findings

### 4.1 Was ClickHouse introduced without contaminating the baseline?

**Yes.** Gateway builds without ClickHouse. No operational service imports the ClickHouse driver. Readiness checks exclude ClickHouse. Smoke tests pass without it. Docker Compose declares no gateway dependency on ClickHouse. All analytical code is additive — no operational handler, route, or adapter was modified to support analytical queries.

Minor observation: `ClickHouseConfig` lives in the shared settings struct, meaning all binaries carry the field definition. This is cosmetic — the field is never populated in non-analytical binaries.

### 4.2 Is the migration tool robust enough?

**Adequate for current scale.** 8 unit tests cover catalog parsing and checksum logic. Forward-only model with drift detection works. Missing: integration tests for the runner, concurrent migration protection, and transaction semantics across multiple migrations.

### 4.3 Was the core schema a good first base?

**Yes.** 6 tables map 1:1 to pipeline domain events. Uniform metadata columns enable cross-table correlation. Conservative data types (Float64, LowCardinality, DateTime64). 90-day TTL prevents unbounded growth. 3 speculative tables were correctly deferred.

### 4.4 Is the writer service well-delimited?

**Well-delimited in scope. Fragile in failure handling.** Standalone binary, independent consumers, mechanical mapping, append-only writes. But: no tests, no pipeline recovery, single INSERT attempt (docs say retry with backoff), silent data loss on buffer overflow, mapper errors produce zero values silently.

### 4.5 Was the query surface introduced with clear boundaries?

**Yes.** Single endpoint with conditional registration, distinct route prefix, 503 fallback, max limit. No contamination of operational routes. Missing: tests for the reader adapter.

---

## 5. Gains and Trade-offs Summary

### Gains
- Structural optionality (binary separation, not feature flags)
- Operational baseline uncontaminated
- Migration infrastructure established
- Canonical 6-table schema
- Write and read paths functional
- 10 optionality rules enforced
- Deferral discipline maintained throughout

### Accepted trade-offs
- Float64 precision (adequate for paper trading)
- No deduplication (low event rate)
- JSON strings for nested structures (no ClickHouse-level indexing)
- Forward-only migrations (no automated rollback)
- Single query endpoint (proves pattern before expanding)

### Open debts (see full accounting in gains-tradeoffs document)
- **Critical:** Writer/reader test coverage, pipeline recovery, failure handling alignment
- **Significant:** Observability, mapper error visibility, migration integration tests
- **Deferred:** Cold-start bootstrap, event schema versioning, materialized views

---

## 6. Next Wave Recommendation

**Harden the analytical runtime before expanding it.**

| Wave | Description | Sequence |
|------|-------------|----------|
| **A. Hardening** | Tests, failure handling, observability | **Next** |
| B. Controlled expansion | Additional query endpoints, deferred families | After A |
| C. Cold-start bootstrap | Derive queries ClickHouse on startup | After B |
| D. Deliberate pause | No analytical work | Not recommended |

Wave A scope: writer tests (mappers, inserter, supervisor, integration), reader tests, pipeline recovery, INSERT retry alignment, buffer overflow metrics, mapper error visibility, write-path observability.

Wave A does NOT include: new tables, new endpoints, cold-start bootstrap, schema evolution, materialized views.

---

## 7. Acceptance Criteria Verification

| Criterion | Met? | Evidence |
|-----------|------|----------|
| Review is specific, honest, and evidence-based | Yes | Findings reference specific files, code paths, and implementation gaps |
| Gains, limits, and trade-offs are clear | Yes | Categorized as gains, accepted trade-offs, implicit trade-offs, and open debts |
| Analytical capability evaluated without bias | Yes | Structural success acknowledged alongside reliability gaps; no celebratory framing |
| Next wave has better-defined sequencing | Yes | Four candidates ranked with preconditions and exit criteria |
| Stage closes the wave with clarity and low drift risk | Yes | Hardening recommended before expansion; anti-patterns documented |

---

## 8. Guard Rail Compliance

| Guard rail | Status |
|------------|--------|
| Review is not automatic celebration | Compliant — reliability gaps, silent data loss, and test deserts called out explicitly |
| Next wave not opened without concrete basis | Compliant — hardening preconditions defined before any expansion |
| Remaining risks not hidden | Compliant — 5 critical risks, 5 significant risks enumerated with scope and priority |
| Analytical runtime not treated as irreversible victory | Compliant — described as "valid first projection, not proven capability" |
| What should remain small is clearly recorded | Compliant — expansion deferred; cold-start bootstrap explicitly deferred to Wave C |

---

## 9. Stage Disposition

**S150 complete.** The analytical runtime entry wave (S143–S149) is formally reviewed and closed. The next stage should begin Wave A (analytical runtime hardening) if the analytical runtime remains active, or disable the writer in docker-compose until hardening is scheduled.
