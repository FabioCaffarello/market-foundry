# Cross-Domain Consistency Checks for Decision Quality

> S472 | Introduced 2026-03-25

## Purpose

The decision pipeline flows through five stages: signal, decision, strategy, risk, execution. Each stage owns its domain model and validates internal invariants. However, **cross-domain invariants** --- relationships that span two or more stages --- were previously unchecked. Silent divergence between domains could produce inconsistent chains without detection.

This document defines the cross-domain consistency checks introduced in S472, their coverage, and their limitations.

## Architecture

### Package: `internal/domain/consistency`

A pure domain package with zero I/O dependencies. It operates on `ChainSnapshot` --- a flat struct of primitive values extracted from domain artifacts --- to avoid coupling between domain packages.

```
decision -> strategy -> risk -> execution
    |            |          |         |
    +------ ChainSnapshot ------+----+
                  |
         consistency.Check()
                  |
              Report{Findings}
```

### Integration Point

Consistency checks are integrated into the **decision review surface** (`DecisionReviewBundle`). Every review bundle now includes a `Consistency` field containing the full `Report` with findings, counts, and clean/dirty status.

This means consistency violations are surfaced:
- In the `/api/v1/decision-review` HTTP endpoint
- In the `Explanation` text summary
- As structured JSON for programmatic consumption

## Checks Implemented

| # | Check ID | Domain Boundary | Severity | Description |
|---|----------|----------------|----------|-------------|
| 1 | `severity_outcome` | decision | violation | triggered decisions must have severity != none; non-triggered must have severity = none |
| 2 | `direction_side` | strategy -> execution | violation | long -> buy, short -> sell, flat -> none; rejected risk -> none |
| 3 | `disposition_action` | risk -> execution | violation/warning | rejected risk must produce side=none, quantity=0; approved non-flat with no action is a warning |
| 4 | `symbol_coherence` | all stages | violation | symbol must be identical across all present stages |
| 5 | `source_coherence` | all stages | violation | source must be identical across all present stages |
| 6 | `timeframe_coherence` | all stages | violation | timeframe must be identical across all present stages |
| 7 | `confidence_progression` | strategy -> risk | warning | risk confidence should not exceed strategy confidence (discount factor expected) |
| 8 | `disposition_propagation` | risk -> execution | violation | execution's risk.disposition must match originating risk assessment |
| 9 | `direction_propagation` | strategy -> risk | violation | risk's strategy input direction must match originating strategy direction |

## Severity Model

- **Violation**: Hard invariant broken. This state should never occur in a correctly wired pipeline. Presence indicates a bug or data corruption.
- **Warning**: Soft invariant or suspicious state. May be legitimate in edge cases but warrants investigation.

## Output Structure

```json
{
  "consistency": {
    "correlation_id": "corr-001",
    "findings": [
      {
        "check": "severity_outcome",
        "severity": "violation",
        "domain": "decision",
        "message": "triggered decision must have severity != none",
        "got": "outcome=triggered severity=none",
        "expected": "severity in {low, moderate, high} when outcome=triggered"
      }
    ],
    "checks_run": 9,
    "violations": 1,
    "warnings": 0,
    "clean": false
  }
}
```

## Partial Chains

Checks gracefully handle incomplete chains. If a stage is missing, checks involving that stage are skipped (not failed). This is by design:
- A `not_triggered` decision may never produce a strategy.
- A `rejected` risk may never produce an execution.
- Events in flight may result in temporarily incomplete chains.

## Limitations

1. **Read-side only**: Checks run at query time on the review surface. They do not block the write path. A violation means the data is already persisted.
2. **No historical repair**: Checks report findings but do not fix them. Remediation is manual.
3. **Confidence comparison is lexicographic**: The current comparison works for fixed-width decimal strings (e.g., `"0.8500"`) but would not work for variable-width representations.
4. **No signal-level checks**: Signal -> decision consistency is not yet checked (signal types vs decision inputs). The signal domain has a different shape that would require additional modeling.
5. **No quantity validation**: The check does not validate that execution quantity is consistent with risk constraints (MaxPositionSize). This would require numeric parsing of decimal strings.
6. **Single-link fan-out**: The current model assumes 1:1 relationships at each stage boundary. Multi-strategy or multi-risk fan-out would need a different snapshot shape.
