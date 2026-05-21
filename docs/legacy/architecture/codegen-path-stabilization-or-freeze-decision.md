# Codegen Path: Stabilization Decision

**Stage**: S207
**Date**: 2026-03-20
**Decision**: Controlled Stabilization
**Status**: Final

---

## 1. Decision Statement

The codegen path is **stabilized for controlled use** within its current scope boundaries. It is neither frozen nor expanded. It exits this phase as a narrow, proven, CI-gated tool for writer consumer specs and pipeline entries.

---

## 2. Decision Rationale

### Evidence For Stabilization (not freeze)

| Factor | Evidence |
|--------|----------|
| Mechanical correctness | 14/14 golden comparisons PASS across all 7 families × 2 artifacts |
| CI integration | 4 gates operational and blocking: validate-all, check, test, integrated |
| Governance value | codegen:begin/end markers enforce ownership; manifest tracks provenance |
| Spec freeze | S193 spec is frozen — no uncontrolled schema evolution possible |
| Cross-spec validation | 7-family uniqueness checks prevent collision |
| Test coverage | 26+ unit tests covering derivation, parsing, comparison, cross-family |
| Drift detection | Structural normalization + line comparison catches deviations |

### Evidence Against Freeze

| Factor | Why freezing is wrong |
|--------|----------------------|
| Working infrastructure | CI gates, golden snapshots, validation scripts are all operational |
| Governance continuity | Freezing removes enforcement of codegen:begin/end markers |
| Low maintenance cost | 2 templates, 7 specs, 14 goldens — no active churn |
| Next-phase compatibility | Refactoring phase benefits from codegen drift detection |

### Evidence Against Expansion

| Factor | Why expansion is premature |
|--------|--------------------------|
| Narrow artifact coverage | Only 2 of 6 Tier 1 artifacts generated (A1, A2) |
| Partial integration | Only 2 of 7 families integrated with governance markers |
| Cross-layer gap | Both integrated families (RSI, EMA) are signal layer only |
| Mapper generation absent | A3 (mappers) requires spec extension not yet designed |
| Live event proof missing | Generated families not yet proven in production event flow |
| Modest time savings | ~15 min per family — governance, not speed, is the value |

---

## 3. What "Controlled Stabilization" Means

### Permitted

- Using codegen to generate A1 (consumer_spec) and A2 (pipeline_entry) for new families
- Running all 4 CI gates on every build
- Adding new family specs that follow the frozen S193 schema
- Integrating generated slices into target files using governance markers
- Regenerating existing golden snapshots after template fixes

### Prohibited

- Expanding codegen to new artifact types (A3–A6) without a new stage
- Adding new template types without architecture review
- Modifying the spec schema without unfreezing S193
- Bypassing CI gates for any reason
- Treating codegen as the sole path for family expansion
- Integrating remaining 5 families in bulk without per-family validation

### Required

- All integrated slices must pass golden comparison in CI
- New integrations must follow the manifest pattern (integrated.yaml entry)
- codegen:begin/end markers must be present in target files
- Each new family integration requires per-family validation

---

## 4. Scope Ceiling

The codegen path stabilizes at this exact scope:

```
Artifacts generated:  A1 (consumer_spec), A2 (pipeline_entry)
Families specified:   7 (candle, ema, mean_reversion_entry, paper_order,
                         position_exposure, rsi, rsi_oversold)
Families integrated:  2 (rsi, ema) — both signal layer
Templates:            2 (consumer_spec.go.tmpl, pipeline_entry.go.tmpl)
CI gates:             4 (validate-all, check, test, integrated)
Golden snapshots:     14 files
```

Any change beyond this scope requires a new architecture stage.

---

## 5. Alternatives Considered

### Full Freeze
Rejected. The codegen infrastructure is working and CI-gated. Freezing removes governance enforcement during the refactoring phase, which is exactly when drift detection is most valuable.

### Aggressive Expansion
Rejected. Only 2 of 7 families are integrated. Cross-layer viability is unproven. Mapper generation requires spec schema extension. Expansion without evidence would violate the project's evidence-over-enthusiasm principle.

### Deprecation
Rejected. The codegen pipeline is mechanically correct and provides auditability. Deprecating working infrastructure that took 13 stages (S192–S204) to build is wasteful.

---

## 6. Risk Assessment

| Risk | Mitigation |
|------|------------|
| Codegen diverges during refactoring | CI gates catch any drift automatically |
| New contributors bypass markers | CI fails on integrated check |
| Spec becomes stale | Specs are source of truth; golden check enforces freshness |
| Temptation to expand scope | This document explicitly caps scope; new stage required |
| Next phase ignores codegen | Stabilization means it stays active, not dormant |
