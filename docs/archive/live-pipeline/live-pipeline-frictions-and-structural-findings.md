# Live Pipeline Frictions and Structural Findings

> S115 — Honest friction capture from operational validation of the minimal live pipeline.

## Classification Legend

| Category | Definition |
|----------|-----------|
| **Bug** | Incorrect behavior that must be fixed — the system does not do what it claims |
| **Operational fragility** | Works today but will break or mislead under plausible conditions |
| **Structural debt** | Architectural misalignment that creates ongoing maintenance cost |
| **Trade-off** | Intentional limitation accepted for the current stage |

---

## F1: Raccoon-CLI Stream Heuristic False Positive (Bug — Fixed)

**What happened:** The `find_stream_name_near` function in raccoon-cli scanned linearly (top-to-bottom) within a 15-line radius to associate a durable consumer with its stream. When a KV bucket constant (`EXECUTION_VENUE_MARKET_ORDER_LATEST`) appeared above a durable definition and the actual stream (`EXECUTION_FILL_EVENTS`) appeared below it, the heuristic picked the wrong one.

**Impact:** Quality gate reported a false error (`durable-stream-alignment FAIL`), blocking development.

**Fix:** Changed both copies of `find_stream_name_near` (in `topology/source.rs` and `runtime_bindings/source.rs`) to scan outward from center. Nearest match wins.

**Residual risk:** The heuristic is still text-based, not AST-based. It could theoretically pick up a non-stream UPPER_SNAKE_CASE constant that happens to be closest. This has not manifested in practice.

---

## F2: Gateway Actor Layer Violation (Structural Debt — Fixed)

**What happened:** `internal/actors/scopes/gateway/gateway.go` imported `internal/interfaces/http/webserver`, violating the clean architecture rule that actors (layer 3) must not depend on interfaces (layer 4). The `webserver` package is HTTP server infrastructure with no interface-layer concerns — it depended only on `internal/shared/settings`.

**Impact:** arch-guard failed. Layer boundary discipline was broken.

**Fix:** Moved `webserver` package from `internal/interfaces/http/webserver/` to `internal/shared/webserver/`. Updated 9 import paths. All tests pass.

**Why this matters:** Clean architecture enforcement is mechanical via raccoon-cli. One violation silently permits others. Fixing early preserves the boundary.

---

## F3: Test Fixture Drift (Structural Debt — Fixed)

**What happened:** The `make_source_topology()` test fixture in `topology.rs` was missing 3 streams (RISK_EVENTS, EXECUTION_EVENTS, EXECUTION_FILL_EVENTS), 4 durable consumers, and 12 subjects that were added during execution pipeline stages S80-S81.

**Impact:** Tests passed but with incomplete coverage. The fixture did not reflect the real system topology, meaning tests validated against a stale model.

**Fix:** Added all missing streams, durables, and subjects to align the fixture with the actual 9-stream architecture.

---

## F4: Stale "consumer"/"validator" Naming Residue (Operational Fragility)

**What it is:** The old architecture used service names like "consumer" and "validator" that have since been renamed. ~260 references to "consumer" remain across 35 files (mostly documentation). 3 test files still reference "validator" as a string literal.

**Impact:** Low. These are in documentation and test assertions, not in runtime code. They cause confusion when reading docs but do not affect behavior.

**Recommendation:** Low-priority batch cleanup. Not worth a dedicated stage. Can be addressed opportunistically during future doc updates.

**Classification:** Trade-off (accepted for now).

---

## F5: TEST_STREAM Non-Canonical Warning (Trade-off)

**What it is:** Drift-detect flags `TEST_STREAM` as a non-canonical stream name.

**Impact:** None. It is used exclusively in test infrastructure and does not exist in the runtime topology.

**Recommendation:** No action needed. The warning is informational and correctly identifies a test-only stream.

**Classification:** Trade-off (intentional).

---

## F6: Raccoon-CLI Heuristic Parser Limitations (Operational Fragility)

**What it is:** The raccoon-cli source scanner uses regex/heuristic parsing rather than Go AST parsing. It extracts streams, durables, and subjects by scanning quoted strings and nearby context. This works well for the current codebase's consistent style but is fundamentally fragile.

**Known edge cases:**
- A constant with UPPER_SNAKE_CASE that is not a stream could be misidentified (mitigated by outward-from-center scan)
- A durable defined far from its stream (>15 lines) falls back to whole-file scan, which may pick up the wrong stream
- Comment-line filtering skips `//` but not `/* */` block comments

**Impact:** Medium. The heuristic has produced one false positive (F1) in the current codebase. As the codebase grows, more edge cases may emerge.

**Recommendation:** Monitor. If another false positive surfaces, consider adding a `// stream:EXECUTION_FILL_EVENTS` annotation convention that the scanner can use as a reliable hint.

**Classification:** Operational fragility.

---

## F7: No Long-Running Stability Validation (Trade-off)

**What it is:** S114 activated the live pipeline and validated snapshot behavior (health, readiness, single-query). No sustained-load test, no overnight soak, no memory leak detection.

**Impact:** Unknown. The system may have issues that only manifest under continuous operation (goroutine leaks, NATS consumer backlog, KV store growth).

**Recommendation:** Not in scope for S115. Consider a future stability validation stage with: (a) 1-hour continuous run, (b) `/statusz` event count growth curve, (c) goroutine count monitoring.

**Classification:** Trade-off (explicitly out-of-scope per S114 definition).

---

## F8: No Data Correctness Verification (Trade-off)

**What it is:** The pipeline validates structural correctness (events flow, KV materializes, endpoints respond) but does not verify that computed values (RSI, candle OHLCV, strategy decisions) are mathematically correct.

**Impact:** A logic bug in a sampler or evaluator would pass all current tests.

**Recommendation:** Add domain-specific golden-file tests for key computations (RSI calculation, candle aggregation). This is a feature concern, not an S115 scope item.

**Classification:** Trade-off.

---

## Friction Priority Matrix

| ID | Category | Severity | Fixed? | Recurrence Risk | Next Action |
|----|----------|----------|--------|-----------------|-------------|
| F1 | Bug | High | Yes | Low (outward scan) | Monitor |
| F2 | Structural debt | High | Yes | None | Done |
| F3 | Structural debt | Medium | Yes | Medium (new families) | Update fixture when adding families |
| F4 | Stale naming | Low | No | None | Opportunistic cleanup |
| F5 | Trade-off | None | N/A | None | Accept |
| F6 | Operational fragility | Medium | Partially | Medium | Add annotation convention if recurs |
| F7 | Trade-off | Low | N/A | N/A | Future stability stage |
| F8 | Trade-off | Low | N/A | N/A | Golden-file tests when adding families |
