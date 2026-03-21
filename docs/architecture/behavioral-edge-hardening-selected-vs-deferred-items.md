# S256: Selected vs Deferred Items

## Selection Criteria

Items were evaluated on three axes:
1. **Cost** — lines of code, risk of regression, review burden
2. **Value** — robustness gain, failure mode eliminated
3. **Dependency** — whether the fix stands alone or requires infrastructure

## Selected Items

| ID | Debt | Rationale | Cost |
|----|------|-----------|------|
| OD-BW4 (partial) | Severity casing/whitespace edge cases | Prevents silent behavioral degradation from upstream casing variance; 4 lines of normalization logic | Very low |
| OD-BW3 | No rejection path in risk evaluators | `DispositionRejected` was a dead constant; zero-confidence produced degenerate approved assessments; adds observable rejection | Low |
| — | Boundary confidence tests | Validates rejection and approval at confidence extremes (0.0, 0.0001, 1.0) | Very low |

## Deferred Items

| ID | Debt | Reason for Deferral | Risk |
|----|------|---------------------|------|
| OD-BW2 | Hardcoded scaling factors | Requires configuration infrastructure that doesn't exist yet; current values are adequate for behavioral proof; premature extraction adds complexity without operational feedback | Low |
| OD-BW5 | No performance budget enforcement | Current overhead is <1s for 37 tests; pipeline is I/O-bound in production, not CPU-bound; budget enforcement adds CI cost without measurable benefit today | Very low |
| OD-BW6 | EX8/configctl activation | Depends on OD-BW2 (configurable factors) and configctl infrastructure maturity; addressing alone creates partial, unused configuration surface | Low |
| OD-BW4 (remainder) | Full severity validation (reject unknown values) | Current behavior (default to neutral) is safe and backward-compatible; strict rejection could break upstream producers that haven't been audited; defer until severity vocabulary is formalized | Low |
| — | Float precision consistency across platforms | IEEE 754 double precision is consistent for current value ranges; documented in S255 round-trip evidence | Very low |
| — | Concurrent evaluation safety | Evaluators are stateless pure functions instantiated per-call; no shared mutable state exists to race on | None |

## Decision Record

The selected items share three properties:
1. They eliminate failure modes that are **silent** (no error, wrong behavior)
2. They require **zero infrastructure** changes
3. They have **zero risk** of breaking existing behavior (normalization is additive; rejection is a new code path for previously-undefined inputs)

The deferred items share the property that their benefit requires either:
- Infrastructure that doesn't exist yet (config system), or
- Operational evidence that the current approach is insufficient (performance, precision), or
- Upstream contract formalization (severity vocabulary)

None of the deferred items represent active risks to the behavioral wave.
