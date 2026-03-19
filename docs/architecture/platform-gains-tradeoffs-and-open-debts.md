# Platform Gains, Trade-offs, and Open Debts (Post-S100)

> Honest accounting of what the S101–S105 hardening wave delivered, what it cost, and what remains unresolved. Companion to the S100 equivalent (`structural-gains-tradeoffs-and-open-debts.md`).

---

## Gains: What Actually Improved

### PG-1. Operational Contract Visibility

**Before:** 10 operational conventions existed only as patterns replicated across `cmd/*/run.go` files. A developer had to reverse-engineer conventions by comparing multiple composition roots.

**After:** 10 invariants and 7 shared behavior rules are documented with classification (invariant vs. convention vs. local behavior). Verification is by inspection — no new automation, but the checklist exists.

**Value:** Prevents silent regression when adding or modifying runtimes. The classification prevents over-standardization — local behaviors are explicitly preserved as runtime-owned decisions.

### PG-2. Diagnostic Surface Completeness

**Before:** Log lines had no runtime identity. `/statusz` reported tracker data without self-identification. No single-request diagnostic overview existed. Shutdown cause was indistinguishable in logs.

**After:** Every log line carries `runtime=<name>`. `/statusz` includes runtime metadata and uptime. `/diagz` provides combined readiness + tracker summary. Shutdown signal type is logged. All supervisors emit consistent startup summaries.

**Value:** Directly reduces debugging time. When aggregating logs from 6 services, `runtime=store` filtering is immediate vs. parsing message text. The `/diagz` endpoint eliminates the need to check `/readyz` and `/statusz` separately.

### PG-3. Error Tracking Accuracy

**Before:** 7 publisher actors and 7 projection actors logged errors but didn't call `tracker.RecordError()`. The `/statusz` and `/diagz` endpoints showed `error_count: 0` even when errors were occurring. This made the diagnostic surface actively misleading.

**After:** All 17 error paths are tracked. A new invariant links `slog.Error` calls to `tracker.RecordError()` in actors with trackers.

**Value:** This is the highest-impact change in the wave. The diagnostic surface went from misleading to accurate. When the vertical slice runs, operators can trust `/diagz` error counts to reflect reality.

### PG-4. Config Validation Hardening

**Before:** Pipeline config accepted duplicates silently. Binding topics accepted any non-empty string. Artifact metadata accepted free-form strings. The family catalog was opaque to tooling.

**After:** Duplicate families rejected. Binding topics format-validated. Artifact metadata whitelisted. Family catalog exported as queryable API (`KnownFamilies()`, `IsKnownFamily()`, `DependencyGraph()`).

**Value:** Catches misconfiguration at startup rather than at runtime. The exported catalog API is the most forward-looking change — it enables cross-registration coherence tests without requiring them now.

### PG-5. Governance Precision

**Before:** Playbooks described how to expand but not whether to expand. Anti-patterns were scattered. drift-detect ARCH_DOCS didn't include governance documents. No cost visibility for expansion decisions.

**After:** Decision gates for all expansion types. 10 anti-patterns cataloged with detection methods. ARCH_DOCS expanded from 8 to 27 entries. Cost budgets per expansion type.

**Value:** Reduces the risk of wrong expansion decisions. The ARCH_DOCS alignment means drift-detect can verify that governance documents exist — closing the gap between "we have conventions" and "we can verify the conventions exist."

---

## Trade-offs: What This Wave Cost

### PT-1. Documentation Volume (Continued)

S101–S105 produced 9 architecture documents and 5 stage reports. Combined with S96–S100, the total is now 18 architecture documents and 9 stage reports for the two consolidation waves.

**Assessment:** This is the dominant cost of the wave. The documentation is proportionate to the complexity, but the maintenance burden is real. Every convention change requires updating both code and documents. The two-tier hierarchy (consolidated > domain-specific) helps by establishing precedence, but doesn't reduce the total volume.

**Mitigation:** Documents should be reviewed when their subject area changes, not on a calendar schedule. raccoon-cli drift-detect verifies document existence but not content freshness — this is an accepted limitation.

### PT-2. Governance Abstraction Layer

The governance model now has three explicit levels (mechanical, structural, judgmental). This meta-documentation is useful for understanding where to invest in tooling vs. process, but it's an abstraction on top of the actual governance.

**Assessment:** Proportionate at current scale. If the governance model itself starts requiring documentation updates, it has become too abstract.

**Mitigation:** The governance model document (`technical-governance-refinement.md`) should be updated only when the governance model genuinely changes — not as a tracking document.

### PT-3. Tooling Maintenance Pressure

drift-detect ARCH_DOCS went from 8 to 27 entries. Each entry is a file existence check — low maintenance, but the list must be updated when governance documents are added, renamed, or removed.

**Assessment:** The maintenance cost scales linearly with governance document count. At 27 entries, this is manageable. At 50+, it would warrant a discovery-based approach.

**Mitigation:** Keep ARCH_DOCS to documents that define binding conventions. Do not add informational or historical documents to the list.

### PT-4. Test Coverage Asymmetry

S104 added 9 tests for config validation. S102 healthz tests cover endpoint response formats. But the actor error tracking fixes from S103 (17 RecordError additions) have no automated test coverage. The error tracking invariant ("every ERROR log must pair with RecordError()") is verified by grep, not by tests.

**Assessment:** This is the most significant quality gap in the wave. The RecordError additions are correct (verified by code review) and purely additive (no behavioral change), but they could regress without detection.

**Mitigation:** A future test infrastructure investment could add a lint rule or grep-based CI check for the RecordError invariant. This is lower priority than running the vertical slice.

---

## Open Debts: What Remains Unresolved

### PD-1. Composition Root Integration Tests

**Status:** Open (carried from S100 D1)
**Severity:** Medium

No automated tests verify that composition roots assemble correctly (infrastructure → composition → wiring → spawn → health → shutdown). Testing is manual — start the service, check `/healthz`, observe logs.

**Why it matters:** A wiring error in a composition root (wrong dependency injected, missing closer) is caught at manual test time, not at CI time.

**Why not addressed:** Composition root tests require running infrastructure (NATS) and are more integration test than unit test. The vertical slice will exercise these paths naturally.

**Recommendation:** Defer until after vertical slice completion. If composition root bugs appear during slice execution, invest in composition root smoke tests.

### PD-2. Cross-Registration Coherence

**Status:** Open (deferred from S104)
**Severity:** Low-Medium

Derive processor families, store pipeline families, and the canonical catalog in `settings/schema.go` are maintained independently. A family could be registered in settings but missing from derive or store without automated detection.

**Why it matters:** Adding a new family requires touching 3 independent declaration sites. Forgetting one produces a runtime that accepts the family in config but doesn't process it.

**Why not addressed:** Cross-package test requires a shared test harness that doesn't exist. The exported `KnownFamilies()` API from S104 provides the foundation.

**Recommendation:** Implement when the next family is added. The cost is low (one test file querying KnownFamilies and comparing against derive/store declarations).

### PD-3. Correlation ID Propagation

**Status:** Open (deferred from S102, S103)
**Severity:** Low (currently), Medium (after vertical slice runs)

`CorrelationID` exists in domain events but is not injected into slog attributes. Cross-runtime event tracing requires manual log correlation by timestamp.

**Why it matters:** When the pipeline runs end-to-end across 4 binaries (ingest → derive → store + execute), tracing a single market event through the full chain requires reading logs from all 4 services and matching by timestamp.

**Why not addressed:** Requires architectural decision on slog context propagation vs. structured event correlation. Both approaches have trade-offs.

**Recommendation:** Address when cross-runtime debugging becomes the primary blocker during vertical slice operation.

### PD-4. Error Classification Taxonomy

**Status:** Open (deferred from S103)
**Severity:** Low

Error logs use free-form strings. A structured classification (connectivity, validation, timeout, internal) would enable automated error categorization.

**Why it matters:** For automated alerting or error dashboards. Not needed for manual operation.

**Recommendation:** Not justified until alerting infrastructure exists. The current ERROR/WARN log level distinction is sufficient.

### PD-5. Raccoon-CLI Governance Constant Maintenance

**Status:** Ongoing
**Severity:** Low

Every new domain requires ~50 lines of governance constants in drift_detect.rs. Every governance document addition requires ARCH_DOCS update.

**Why it matters:** If raccoon-cli constants drift from code reality, governance checks produce false positives or false negatives.

**Recommendation:** Include raccoon-cli updates as part of the expansion playbook (already documented in S105). This is maintenance, not debt.

---

## Refactors That Do NOT Warrant the Cost Now

### PR-1. Automated RecordError Enforcement

A lint rule or raccoon-cli check that verifies every `slog.Error` in an actor with a tracker is paired with `tracker.RecordError()`.

**Why not:** The 17 fixes in S103 cover all current actors. New actors follow the established pattern by reference (execution actors are the template). A grep-based check (`grep -A2 'logger.Error' | grep -v RecordError`) catches regressions during review. A full lint rule would require Go AST parsing in raccoon-cli with scope resolution — disproportionate investment.

### PR-2. Unified Runtime Test Framework

A test framework that boots a runtime with mock infrastructure, verifies the 6-phase lifecycle, and checks health endpoints.

**Why not:** Each runtime's infrastructure dependencies are different (gateway: HTTP+NATS, execute: NATS+venue, configctl: NATS only). A unified framework would either be too generic or require per-runtime specialization that negates the abstraction.

### PR-3. Automated Documentation Freshness Checking

A tool that compares documentation claims to code reality (e.g., verifying that documented API signatures match actual signatures).

**Why not:** Architecture documents capture intent and rationale, not API specifications. The value is in the human-written "why", not the mechanically-verifiable "what". Documentation freshness is best maintained through ownership discipline, not automation.

### PR-4. Structured Error Types with Category Field

Adding an optional `Category` field to `*problem.Problem` for automated error classification.

**Why not:** No automated error handling pipeline exists. The category would be written but never read. Invest when alerting infrastructure justifies it.

### PR-5. OpenTelemetry/Distributed Tracing

Full distributed tracing with span propagation across NATS boundaries.

**Why not:** Requires collector infrastructure, SDK integration, and span propagation middleware. The observability foundation from S102 (runtime-tagged logs, /diagz, lifecycle signals) is sufficient for current development. Invest when operational issues (latency diagnosis across services) become the primary blocker.

---

## Wave Summary: S101–S105 in Numbers

| Metric | Count |
|--------|-------|
| Code files modified | 28 |
| Code locations fixed (RecordError) | 17 |
| New tests added | 9 (S104) |
| Architecture documents created | 9 |
| Stage reports created | 5 |
| drift-detect ARCH_DOCS entries added | 19 |
| Invariants formalized | 10 |
| Shared behavior rules formalized | 7 |
| Anti-patterns cataloged | 10 |
| S100 open debts closed | 3 of 5 |
| New open debts introduced | 0 |

---

## Related Documents

- `post-s100-technical-platform-readiness-review.md` — readiness assessment
- `next-platform-wave-recommendations.md` — next steps
- `structural-gains-tradeoffs-and-open-debts.md` — S96–S99 equivalent
