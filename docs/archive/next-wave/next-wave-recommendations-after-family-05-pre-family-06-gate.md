# Next Wave Recommendations — After Family 05 / Pre-Family-06 Gate

## Recommendation Summary

| Priority | Recommendation | Rationale |
|----------|---------------|-----------|
| **1** | Proceed to Family 06 (conditional) | Pattern is sound, H-5 provides runway, within-layer expansion value remains |
| **2** | Scope codegen tranche formally | 5 families / 0 creative decisions = codegen is overdue for scoping |
| **3** | Defer codegen implementation to post-F06 | F06 is still within manual ceiling; codegen investment better serves F07+ |
| **4** | Do NOT pre-authorize Family 07 | F07 gate must require codegen scope as input |

## Detailed Recommendations

### 1. Family 06 — Conditional Authorization

**What**: Authorize one more manual family expansion, following the established 9-artifact template.

**Why**: The pattern is proven and healthy after H-5. Handler at 501 lines has 2–3 families of runway. Full vertical coverage is achieved; within-layer variants (different event types within existing layers) add analytical depth without structural risk.

**Conditions**:
- Candidate must NOT require write-path changes.
- Candidate must NOT push reader parameters past 11.
- Family 06 must measure and report ceiling metrics (handler lines, reader params, parser count).
- Gate review at Family 06 completion must evaluate codegen readiness.

**Candidate space** (from prior assessments):
- EMA Crossover (within-layer variant of Signals, L2)
- Volume metrics (may require write-path — evaluate carefully)
- Tradeburst events (may require infrastructure — evaluate carefully)

### 2. Codegen Tranche Scoping

**What**: A dedicated stage to define codegen scope, template approach, and implementation plan.

**Why**: The evidence is unambiguous:
- 0 creative decisions across 5 families.
- ~85% handler duplication, ~80% reader duplication, ~70% use case duplication.
- Manual cost: ~45 min / ~780 LOC per family.
- Codegen cost (once built): ~2 min per family.
- Break-even: Family 06–07.

**Scope to define**:
- Template language/approach (Go templates, code generation tool, or custom).
- Which artifacts are templated (reader, handler, use case, tests, routes).
- Schema-driven generation (derive from DDL/migration files) vs. config-driven.
- Generated code testing strategy.
- Integration with existing CI.

**When**: After Family 06 delivery, before Family 07 trigger assessment.

### 3. Defer Codegen Implementation

**What**: Do not implement codegen for Family 06.

**Why**: Family 06 is still within the manual pattern's structural capacity. Investing in codegen before Family 06 delays delivery without commensurate benefit. The correct sequence is:
1. Family 06 manually (validates pattern one more time under within-layer variant).
2. Codegen scope (uses all 7 families as specification evidence).
3. Family 07 via codegen (first generated family).

### 4. Do Not Pre-Authorize Family 07

**What**: Family 07 gate must independently evaluate whether codegen is scoped and whether the pattern can absorb another expansion.

**Why**: The gate discipline that has kept Wave B healthy depends on each family being individually authorized. Pre-authorization undermines this. Specifically:
- Family 07 will likely need 11+ reader parameters (requires query-object refactoring).
- Family 07 will likely add parser 9+ (requires generic parser consideration).
- Family 07 should be the first codegen-produced family if codegen is adopted.
- None of these decisions should be made implicitly.

## What Must NOT Happen

1. **Do not expand to Family 06 without candidate trigger assessment.** The trigger assessment determines whether a candidate exists that fits the conditions. If no candidate fits, the correct action is hardening or consolidation, not relaxing conditions.

2. **Do not treat codegen as optional past Family 07.** The evidence is clear: the manual pattern works but does not scale. Continuing manual expansion past 7 families is engineering waste by choice.

3. **Do not skip gate reviews.** The gate at Family 06 completion must evaluate: handler metrics, reader metrics, parser count, codegen scope status, and any new frictions. Every family gets a gate.

4. **Do not merge concerns.** Family 06 is an expansion stage. Codegen scoping is an architecture stage. These must remain separate deliverables with separate acceptance criteria.

## Sequence

```
S190: Post-Family-05 / Pre-Family-06 Gate ← this document
  │
  ├── IF acceptable candidate exists:
  │   ├── S191: Family 06 Trigger Assessment + Candidate Selection
  │   ├── S192: Family 06 Definition and Contract
  │   ├── S193: Family 06 Minimal Implementation
  │   ├── S194: Family 06 End-to-End Validation
  │   ├── S195: Post-Family-06 Hardening (if triggered)
  │   └── S196: Post-Family-06 Gate (must include codegen scope input)
  │
  └── IF no acceptable candidate:
      ├── S191: Codegen Tranche Scoping
      └── S192: Post-Codegen-Scope Gate
```

## Open Questions for Family 06 Candidate Assessment

1. Which within-layer variants have pre-staged infrastructure (migrations, writer mappers)?
2. Do any candidates introduce new ClickHouse types not yet proven?
3. Do any candidates require write-path changes (disqualifying)?
4. What is the expected reader parameter count for each candidate?
5. Does the candidate add analytical depth (same layer, different event type) or breadth (new concern)?

## Conclusion

The Wave B pattern is at a natural inflection point. It has proven itself through 6 families and full vertical coverage. The next phase is either one more controlled manual expansion (Family 06) or direct investment in codegen. Both paths are valid; the gate recommends the former because it produces one more family of specification evidence while the codegen scope is being defined.

The critical constraint is: **Family 07 does not proceed without codegen scope.** This is the bright line that prevents the manual pattern from becoming technical debt.
