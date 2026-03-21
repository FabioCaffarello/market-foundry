# Charter Amendment Rules and Breadth Governance

**Stage:** S239
**Date:** 2026-03-20
**Status:** Active

## Purpose

This document codifies the rules for formal charter amendments, derived from the governance deviation discovered during S238. It also establishes specific governance guardrails for the upcoming breadth charter.

## Charter Amendment Rules

### Rule 1: Pivots Must Be Documented Before Execution

When work reveals that the charter's original scope should change (e.g., breadth → depth, addition/removal of deliverables), the amendment must be documented **before** the pivot is executed.

**Document:** A dedicated amendment record in `docs/architecture/` with:
- Original charter commitment being changed
- Proposed new scope
- Rationale for the change
- Impact on exit criteria
- Revised success metrics

### Rule 2: Exit Criteria Must Be Updated

A charter amendment that changes scope must also update the exit criteria. The amendment document must explicitly state:
- Which original exit criteria are being removed or modified
- What replacement criteria apply
- Whether the hardening budget (≤20%) is affected

### Rule 3: Mid-Charter Gates

Charters spanning more than 3 stages must include at least one mid-charter checkpoint. At the checkpoint:
- Review progress against original exit criteria
- Identify scope drift early
- Formally amend if drift exceeds 1 deliverable

### Rule 4: Amendments Do Not Retroactively Modify Charters

Per the existing governance framework (`domain-evolution-entry-exit-and-stop-conditions.md`, Section 5): amendments are **appended**, never retroactively applied to the original charter document. This preserves auditability.

### Rule 5: Post-Hoc Amendments Are Permitted But Flagged

If a pivot was executed without a pre-amendment (as happened in S233–S237), a post-hoc amendment must be filed. Post-hoc amendments must:
- Acknowledge the governance deviation
- Explain why it was not caught earlier
- Document the corrective action (this rule set)
- Be flagged in the gate report as a governance finding

## Breadth Governance for S240+

### Definition of Breadth

"Breadth" means adding at least one additional evaluator/resolver **type** per domain, exercising a genuinely different evaluation logic path. Enriching an existing evaluator (e.g., adding severity to RSI oversold) is **depth**, not breadth.

### Minimum Acceptance for Breadth

For each domain (decision, strategy, risk), the breadth charter must deliver:
- A second evaluator/resolver with distinct type name
- Full domain object validation (Validate() passes)
- Unit tests at domain, application, and actor layers
- Integration into the actor chain (fan-out messages, publisher messages)
- The chain integration test pattern from S239 extended to cover the new types

### Anti-Drift Guardrails

1. **Scope freeze after S240 charter document is approved** — no scope changes without formal amendment
2. **Mid-charter gate at the halfway point** — mandatory progress check
3. **Depth work is prohibited** unless it is a blocking prerequisite for breadth (must be documented as such)
4. **Each new evaluator type must be complete before starting the next** — no partial implementations across types

### Breadth Candidates (Informational)

These are candidates identified during S233–S237 depth work. The S240 charter will select from this list:

**Decision domain:**
- Volume spike evaluator
- Price momentum evaluator
- Multi-timeframe RSI evaluator

**Strategy domain:**
- Breakout entry resolver
- Momentum continuation resolver

**Risk domain:**
- Drawdown limit evaluator
- Correlation exposure evaluator

Selection criteria for the charter: implementation complexity, reuse of existing domain infrastructure, and diversity of evaluation logic paths.

## Applicability

These rules apply to all future charters in the market-foundry project. They complement (not replace) the existing governance framework in `domain-evolution-entry-exit-and-stop-conditions.md`.
