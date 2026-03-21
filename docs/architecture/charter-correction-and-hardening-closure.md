# Charter Correction and Hardening Closure

**Stage:** S239
**Date:** 2026-03-20
**Status:** Executed

## Context

The S233–S237 charter committed to delivering **domain breadth** — at least 2 evaluator/resolver types per domain (decision, strategy, risk). The actual delivery was **domain depth**: the existing single evaluator per domain was significantly enriched with severity classification, human-readable rationale, metadata enrichment, and end-to-end decision context threading.

This pivot was pragmatically correct and delivered genuine value. However, it was never formalized as a **charter amendment**, violating the governance framework defined in `domain-evolution-entry-exit-and-stop-conditions.md` (Section 5: Charter Amendment Process).

The S238 gate concluded with a **CONDITIONAL PASS** — 6 of 9 exit criteria met. The 3 unmet criteria were all breadth targets.

## Formal Correction

### What Happened

1. **Charter S233 committed to:** ≥2 evaluator types per domain (decision, strategy, risk)
2. **Charter S233–S237 delivered:** Semantic depth enrichment of existing single evaluators
3. **Governance violation:** The breadth→depth pivot was executed as an informal runtime decision without triggering the documented amendment process

### Why It Happened

- The depth work emerged organically as prerequisite understanding for breadth (you must deeply understand the first evaluator to correctly design the second)
- Each depth enrichment (severity, rationale, metadata) pulled the next one naturally
- The immediate value of depth improvements made the pivot feel pragmatically justified
- No formal gate existed mid-charter to catch scope drift before the final gate

### Classification

This is classified as a **governance process deviation**, not a technical failure. The delivered code is correct, tested, and valuable. The issue is exclusively about charter-level accountability and traceability.

### Corrective Actions Applied

1. **This document** serves as the formal post-hoc amendment record, acknowledging the scope change
2. **Charter amendment rules** have been codified in `charter-amendment-rules-and-breadth-governance.md` to prevent recurrence
3. **The S238 CONDITIONAL PASS stands** — the delivered depth work is genuine and the breadth gap is real
4. **The next charter (S240+)** will explicitly target breadth with stricter governance

## Hardening Gaps Closed

### Strategy Test Coverage

**Before S239:**
| Layer | Strategy | Decision | Risk |
|-------|----------|----------|------|
| Domain | 13 | 17 | 19 |
| Application | 12 | 24 | 17 |
| Actor (Derive) | 5 | 7 | 6 |
| **Total** | **30** | **48** | **42** |

**After S239:**
| Layer | Strategy | Decision | Risk |
|-------|----------|----------|------|
| Domain | 17 (+4) | 17 | 19 |
| Application | 12 | 24 | 17 |
| Actor (Derive) | 9 (+4) | 7 | 6 |
| **Total** | **38** (+8) | **48** | **42** |

New strategy tests added:
- **Domain:** Multi-symbol partition key isolation, multi-symbol deduplication key isolation, negative timeframe validation, empty decisions slice validation
- **Actor:** Severity/rationale propagation, nil ScopePID publish without fan-out, fan-out with decision context, invalid confidence rejection

### Inter-Actor Chain Integration Test

**Before S239:** No test verified the full decision → strategy → risk actor chain.

**After S239:** `actor_chain_integration_test.go` provides 3 integration tests:
1. **Full triggered path:** signal → triggered decision → long strategy → approved risk, with decision severity/rationale verified at each stage
2. **Not-triggered path:** signal → not_triggered decision → flat strategy → approved risk
3. **Correlation ID preservation:** Verifies correlation ID survives the entire chain end-to-end

These tests wire real actor instances (not mocks) and manually forward inter-actor messages to simulate the SourceScopeActor routing, proving:
- Decision context (severity, rationale) propagates correctly through the full chain
- Correlation IDs are preserved end-to-end
- Domain isolation is maintained (DBI-9 primitive-only messages)
- Both triggered and not-triggered paths produce valid, finalized domain objects

## Non-Objectives

- This stage does **not** open the breadth charter
- This stage does **not** add new evaluator/resolver types
- This stage does **not** refactor existing code beyond test additions
- This stage does **not** address the risk confidence scaling (0.95 factor) debt

## Impact on Next Charter

The next charter (S240+) starts with:
- A formally corrected governance baseline
- Strategy test coverage closer to parity with other domains
- A proven inter-actor chain integration test pattern that breadth work can extend
- Explicit charter amendment rules preventing informal scope drift
