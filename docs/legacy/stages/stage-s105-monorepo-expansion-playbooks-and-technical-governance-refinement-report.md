# Stage S105 — Monorepo Expansion Playbooks and Technical Governance Refinement

**Status:** Complete
**Date:** 2026-03-19

## 1. Executive Summary

S105 refines the expansion playbooks and technical governance of Market Foundry based on concrete experience from the S96–S104 consolidation wave and the full vertical slice implementation. The focus is on making the monorepo's growth mechanisms more precise, more actionable, and better aligned with tooling — without adding bureaucratic overhead.

**Key outcomes:**
- Refined expansion playbooks with explicit decision gates and worked examples from real history.
- Added venue adapter expansion playbook (closing D5 debt from `structural-gains-tradeoffs-and-open-debts.md`).
- Cataloged 10 concrete anti-patterns observed during market-foundry evolution with detection methods.
- Added explicit "when NOT to expand" criteria with cost budgets for each expansion type.
- Updated `drift_detect.rs` ARCH_DOCS to include all S96–S105 canonical governance documents (19 additions).
- Established two-tier architecture documentation model: consolidated governance docs (authoritative for conventions) vs. domain-specific docs (authoritative for design decisions).
- Documented governance health indicators for ongoing assessment.

## 2. Gap Analysis (Pre-S105)

### What S100 identified:
> "Playbooks and conventions already aggregate real value. Now makes sense to refine them from concrete consolidation experience."

### Concrete gaps found:

| Category | Impact |
|----------|--------|
| Playbooks lacked decision gates (when to expand vs. when not to) | Developers could follow the "how" without evaluating the "whether" |
| No venue adapter expansion playbook | D5 open debt — pattern for adding venues was undocumented |
| Anti-patterns were scattered across trade-off docs, not cataloged | No single reference for known mistakes and their corrections |
| `drift_detect.rs` ARCH_DOCS didn't include S96–S104 governance docs | Tooling couldn't verify that canonical governance documents exist |
| No explicit hierarchy between consolidated and domain-specific docs | Conflict resolution between overlapping documents was unclear |
| No cost budget for expansion types | Developers couldn't estimate the full cost of an expansion decision |

## 3. Changes Made

### 3.1 New Architecture Documents

| Document | Purpose |
|----------|---------|
| `docs/architecture/expansion-playbooks-refined.md` | Refined playbooks with decision gates, worked examples, venue adapter guide, configuration discipline, and raccoon-cli governance update requirements |
| `docs/architecture/structural-anti-patterns-and-when-not-to-expand.md` | 10 cataloged anti-patterns with detection methods, explicit "when NOT to expand" criteria, and expansion cost budgets |
| `docs/architecture/technical-governance-refinement.md` | Governance model documentation: three-level enforcement (mechanical/structural/judgmental), tooling alignment, two-tier doc hierarchy, governance health indicators |

### 3.2 Tooling — Raccoon-CLI Governance Alignment

| Change | Detail |
|--------|--------|
| `drift_detect.rs` ARCH_DOCS expanded | Added 19 canonical governance documents from S96–S105 to the architecture docs existence check |
| Comments added to ARCH_DOCS | Pre-consolidation and consolidated doc groups clearly delineated |

### 3.3 Existing Documents — No Modifications

The existing playbook (`how-to-introduce-new-runtimes-domains-and-families.md`) and registration rules (`family-runtime-registration-rules.md`) remain unchanged. They are still valid and referenced by the refined playbooks. The refinement is additive — it provides decision gates and anti-patterns that complement the existing step-by-step instructions.

## 4. Files Changed

| File | Type |
|------|------|
| `docs/architecture/expansion-playbooks-refined.md` | New |
| `docs/architecture/structural-anti-patterns-and-when-not-to-expand.md` | New |
| `docs/architecture/technical-governance-refinement.md` | New |
| `tools/raccoon-cli/src/analyzers/drift_detect.rs` | Modified — ARCH_DOCS expanded with 19 governance docs |
| `docs/stages/stage-s105-monorepo-expansion-playbooks-and-technical-governance-refinement-report.md` | New |

## 5. Playbook Refinements Applied

### 5.1 Decision Gates Added

Each expansion playbook now starts with a decision gate — a table of yes/no questions that must be answered before starting:

| Playbook | Key Gate Questions |
|----------|-------------------|
| New family (same domain) | Does it have its own event type? Can you name its upstream dependency? |
| New venue adapter | Is paper_order running end-to-end? Do you have testnet credentials? |
| New domain | Does it have at least one family ready? Can you define its dependency chain position? |
| New runtime | Does it have a distinct operational concern? Does it need independent scaling? |
| New adapter group | Is this genuinely new technology, not a new instance? |

### 5.2 Venue Adapter Playbook (New — Closes D5 Debt)

Documented the intentionally non-catalog-driven venue adapter expansion path:
- Security-motivated explicit switch statement in `cmd/execute/run.go`
- Steps covering domain extension, application adapter, exchange SDK adapter, settings registration, runtime wiring
- Credential management security considerations

### 5.3 Worked Examples

Added worked example from RSI signal family addition (S35–S41) — what went right (catalog pattern), what nearly went wrong (processor type copy-paste), and the lesson (type-level distinction prevents runtime routing bugs).

### 5.4 Cost Budgets

Documented the full cost of each expansion type:
- New family (same domain): ~8-12 files + ongoing config/test/governance maintenance
- New family (new scope): ~15-20 files + derive processor type + store scope injection
- New domain: ~20-30 files + ~50 LOC raccoon-cli governance constants
- New runtime: ~10-15 files + full deployment config + operational overhead
- New venue adapter: ~5-8 files + credential management + exchange API versioning

## 6. Anti-Patterns Cataloged

| ID | Anti-Pattern | Source | Detection |
|----|-------------|--------|-----------|
| AP-1 | Duplicated catalog synchronization points | S97 correction | PR touches catalog AND separate list in same runtime |
| AP-2 | Phase interleaving in composition roots | S96 correction | Infrastructure creation after wiring starts |
| AP-3 | Semantic drift in terminology | S98 correction | `drift-detect` defunct name check |
| AP-4 | Premature domain creation | Prevented by design | `drift-detect` prohibited streams guard |
| AP-5 | Force-fitting divergent shapes into uniform catalogs | Correctly avoided in derive | `interface{}` params or reflection in catalog entries |
| AP-6 | Cross-domain imports in domain layer | Prevented since S1 | `make arch-guard` |
| AP-7 | Infrastructure types leaking into domain/application | Enforced by arch-guard | `make arch-guard` type expression check |
| AP-8 | Hardcoded family activation | Prevented by design | Family name literals in non-test/non-config Go code |
| AP-9 | Test fixtures diverging from canonical conventions | S104 correction | `grep -r "old_value" --include="*_test.go"` |
| AP-10 | Documentation without corresponding tooling enforcement | S105 correction | Governance docs missing from drift-detect ARCH_DOCS |

## 7. Governance Gains

| Before | After |
|--------|-------|
| Playbooks described HOW to expand but not WHETHER to expand | Decision gates force evaluation before execution |
| Venue adapter expansion path undocumented (D5 debt) | Full playbook with security considerations |
| Anti-patterns scattered across trade-off docs | 10 cataloged patterns with detection methods in single reference |
| `drift-detect` ARCH_DOCS missed 19 governance docs | All canonical governance documents mechanically verified |
| No explicit doc hierarchy between eras | Two-tier model: consolidated (conventions) vs. domain-specific (design decisions) |
| No expansion cost visibility | Full cost budgets per expansion type |
| No governance health indicators | 5 measurable indicators for ongoing assessment |

## 8. Limits Maintained

| Aspect | Decision | Rationale |
|--------|----------|-----------|
| No governance approval workflow | Decision gates are self-assessed | Adding approval gates would slow solo development without proportionate risk reduction |
| No automated playbook enforcement | Playbooks guide, tooling enforces mechanically verifiable rules only | Judgment-requiring decisions cannot be mechanized |
| Existing playbook unchanged | Refinement is additive, not a rewrite | The original `how-to-introduce` playbook is correct; it was incomplete, not wrong |
| No document consolidation/pruning | Both eras of docs remain | Domain-specific docs still contain authoritative design rationale |
| No new raccoon-cli analyzers | Existing analyzers strengthened, not multiplied | Adding analyzers increases maintenance burden; prefer strengthening existing ones |

## 9. Recommended Preparation for S106

Based on remaining gaps and natural next steps from the S100 readiness review:

1. **Vertical slice completion** — The governance infrastructure is now robust. The next value-creating work is running the full pipeline chain end-to-end (`candle → rsi → rsi_oversold → mean_reversion_entry → position_exposure → paper_order`) as recommended in `next-technical-wave-recommendations.md`.

2. **Cross-registration coherence test** — Deferred from S104. A test-time check that verifies derive processor families and store pipeline families align with the canonical catalog in `settings/schema.go`. The exported `KnownFamilies()` API from S104 provides the foundation.

3. **Operational confidence layer** — When the vertical slice runs, structured logging at pipeline boundaries and a diagnostic endpoint showing pipeline status will be needed for debugging. This was identified as Priority 2 in `next-technical-wave-recommendations.md`.
