# Refactors Deferred After Live Pipeline

> S116 — Items considered but explicitly deferred, with justification.

## Guiding Principle

A refactor is deferred when: (a) the pain is real but the cost/risk of fixing exceeds the current benefit, (b) the pain is speculative and hasn't manifested in practice, or (c) the fix would open a horizontal wave that contradicts stage governance.

---

## D1: Raccoon-CLI AST Parsing (F6)

**Pain:** The source scanner uses regex/heuristic parsing, which produced one false positive (F1, fixed in S115 via outward-from-center scan). The heuristic is fundamentally fragile.

**Why deferred:** The heuristic works correctly for the current codebase after the S115 fix. Building a Go AST parser in Rust would be a significant effort (~1-2 weeks) with no proven ongoing pain. The annotation convention (`// stream:X`) is a cheaper mitigation if another false positive surfaces.

**Trigger to revisit:** Second false positive from the heuristic parser.

---

## D2: Stale Naming Cleanup in Documentation (F4)

**Pain:** ~40 references to "consumer service" and "validator service" remain across documentation files (`docs/architecture/`, `docs/stages/`). These are historical records of the sanitization process.

**Why deferred:** Documentation files are historical artifacts. Rewriting them to eliminate old names would alter the historical record without operational benefit. The stale names do not appear in active code after R1-R4.

**Trigger to revisit:** Only if a new doc references old names (indicates incomplete onboarding).

---

## D3: Use-Case Pattern Unification (not in S115)

**Pain:** Two patterns coexist for use cases: `configctlclient` uses type-alias thin wrappers via `usecase.CommandUseCase`, while other clients (evidence, risk, signal, strategy, decision, execution) use concrete struct implementations with inline validation. This creates inconsistency when adding new use cases.

**Why deferred:** This was not flagged in S115. Both patterns work. The inconsistency has not caused bugs or confusion. Unifying them would touch 20+ files across 7 domains — a horizontal refactor that contradicts S116 guard rails.

**Trigger to revisit:** Adding a new domain where the developer is confused about which pattern to follow.

---

## D4: Long-Running Stability Validation / Soak Test (F7)

**Pain:** No sustained-load test exists. The system may have issues (goroutine leaks, NATS backlog growth) that only manifest under continuous operation.

**Why deferred:** Requires infrastructure (long-running environment, monitoring, alerting) that doesn't exist yet. The cost of building this exceeds the current value for a single-symbol/paper-trading setup.

**Trigger to revisit:** Moving to multi-symbol or live trading where stability is critical.

---

## D5: Domain-Specific Golden-File Tests (F8)

**Pain:** No verification that computed values (RSI, candle OHLCV, strategy decisions) are mathematically correct.

**Why deferred:** This is a feature concern, not an architectural one. The pipeline validates structural correctness. Adding golden-file tests should accompany the addition of new signal families or strategy types.

**Trigger to revisit:** Adding a second signal family or strategy type.

---

## D6: Script Hardening (live-pipeline-activate.sh, smoke tests)

**Pain:** Scripts embed Python one-liners for JSON parsing, hardcode URLs, and don't support atomic step re-entry.

**Why deferred:** Scripts work. The friction is low-frequency (scripts are run occasionally, not in CI). Refactoring them would not improve the architecture or reduce ongoing maintenance cost.

**Trigger to revisit:** Adding CI/CD pipeline that needs robust script automation.

---

## D7: Config Parameterization (NATS URLs in 6+ config files)

**Pain:** NATS URLs are duplicated across 6 deploy config files. Updating the URL requires changes in multiple places.

**Why deferred:** The local development environment uses a single NATS instance at a fixed URL. The duplication has not caused a bug. Templating would add complexity (envsubst, Docker configs, or a config generator) without current benefit.

**Trigger to revisit:** Adding a second deployment environment (staging, production).

---

## Summary

| ID | Friction | Decision | Trigger |
|----|----------|----------|---------|
| D1 | Heuristic parser fragility | Defer | Second false positive |
| D2 | Stale names in docs | Defer | New doc with old names |
| D3 | Use-case pattern inconsistency | Defer | New domain confusion |
| D4 | No soak test | Defer | Multi-symbol or live trading |
| D5 | No golden-file tests | Defer | New signal/strategy family |
| D6 | Script fragility | Defer | CI/CD pipeline |
| D7 | Config duplication | Defer | Second environment |
