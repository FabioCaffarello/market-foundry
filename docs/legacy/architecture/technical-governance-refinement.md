# Technical Governance Refinement

> Refinements to market-foundry's technical governance based on lessons from S96–S104. This document records what changed, why, and how the governance model now operates.

---

## Governance Model Overview

Market Foundry's governance operates at three levels:

| Level | Mechanism | Enforcement |
|-------|-----------|-------------|
| **Mechanical** | raccoon-cli (arch-guard, drift-detect, topology-doctor) | Automated — `make check` / `make verify` |
| **Structural** | Playbooks, naming conventions, lifecycle patterns | Convention — enforced through review and documentation |
| **Judgmental** | Expansion decision gates, cost budgets, anti-patterns | Human — informed by documentation, not replaced by it |

The S105 refinement strengthens the alignment between these three levels. The goal is not more rules — it's better correspondence between what the tooling checks, what the playbooks describe, and what developers need to decide.

---

## Refinement 1: Tooling ↔ Documentation Alignment

### Problem

The `drift_detect.rs` ARCH_DOCS constant referenced pre-consolidation documents but did not include the canonical governance documents produced by S96–S99. This meant:
- `make check` could not verify that the consolidation-era documents exist.
- A developer could delete `runtime-assembly-guidelines.md` without triggering a drift finding.
- The tooling and documentation were governing different things.

### Change

Updated `ARCH_DOCS` in `drift_detect.rs` to include the canonical governance documents from S96–S104:

**Added:**
- `docs/architecture/boundary-naming-and-interface-hygiene.md`
- `docs/architecture/dependency-injection-and-composition-roots.md`
- `docs/architecture/diagnostic-surfaces-and-runtime-signals.md`
- `docs/architecture/error-handling-and-degradation-policy.md`
- `docs/architecture/fail-fast-vs-graceful-degradation-rules.md`
- `docs/architecture/family-runtime-registration-rules.md`
- `docs/architecture/how-to-introduce-new-runtimes-domains-and-families.md`
- `docs/architecture/minimal-observability-foundation.md`
- `docs/architecture/monorepo-structure-and-engineering-conventions.md`
- `docs/architecture/naming-conventions-for-domains-families-and-runtimes.md`
- `docs/architecture/operational-contracts-and-cross-runtime-conventions.md`
- `docs/architecture/registry-driven-runtime-assembly.md`
- `docs/architecture/runtime-assembly-guidelines.md`
- `docs/architecture/runtime-invariants-and-shared-behavior-rules.md`
- `docs/architecture/config-activation-and-dependency-map-model.md`
- `docs/architecture/config-validation-and-sync-rules.md`
- `docs/architecture/expansion-playbooks-refined.md`
- `docs/architecture/structural-anti-patterns-and-when-not-to-expand.md`
- `docs/architecture/technical-governance-refinement.md`

### Rationale

Governance documents that define binding conventions must be mechanically verifiable. Adding them to `ARCH_DOCS` means `drift-detect` will flag if any of these files are removed or renamed. This closes the gap between "we have conventions" and "we can verify the conventions exist."

---

## Refinement 2: Two-Tier Architecture Documentation Model

### Problem

Market Foundry has accumulated architecture documents from two eras:
1. **Domain-specific documents** (S35–S93): `signal-domain-design.md`, `decision-stream-families.md`, `risk-activation-and-ownership.md`, etc. These were created per-domain as each domain was implemented.
2. **Consolidated governance documents** (S96–S104): `monorepo-structure-and-engineering-conventions.md`, `family-runtime-registration-rules.md`, etc. These are cross-cutting and supersede domain-specific guidance where they overlap.

Without explicit hierarchy, a developer doesn't know which document to trust when they conflict.

### Model

```
Consolidated governance docs (S96+)     ← AUTHORITATIVE for cross-cutting conventions
  ├── naming, lifecycle, registration, boundary rules
  ├── expansion playbooks, anti-patterns
  └── config activation, dependency maps

Domain-specific docs (S35–S93)           ← AUTHORITATIVE for domain-specific design decisions
  ├── domain design rationale
  ├── family contracts
  └── domain-specific stream/projection patterns
```

**Rule:** When a consolidated doc and a domain-specific doc disagree on a convention (naming, lifecycle, registration), the consolidated doc wins. When a question is domain-specific (why does RSI use Wilder's smoothing? what are the decision evaluation thresholds?), the domain doc is authoritative.

### No Changes Required

This hierarchy was already implicit. Making it explicit prevents confusion when the document count grows.

---

## Refinement 3: Expansion Decision Gates in Playbooks

### Problem

The original `how-to-introduce-new-runtimes-domains-and-families.md` provided step-by-step instructions for HOW to expand, but not decision criteria for WHETHER to expand. A developer following the playbook could add a new domain for a concept that should have been a family.

### Change

The refined playbooks (`expansion-playbooks-refined.md`) now include explicit Decision Gates — tables of yes/no questions that must be answered before starting each expansion type. These are not checklists to satisfy bureaucracy; they're calibration questions based on real decisions from market-foundry history.

Additionally, `structural-anti-patterns-and-when-not-to-expand.md` provides a consolidated "when NOT to expand" reference with concrete cost budgets.

### Rationale

The cost of a wrong expansion is high (see AP-4: Premature Domain Creation). Decision gates reduce this risk without adding process overhead — they're questions to answer, not approvals to seek.

---

## Refinement 4: Raccoon-CLI as Governance Backbone

### Current State

raccoon-cli provides 18 analyzers organized into three quality-gate profiles:

| Profile | Checks | Requires Infra |
|---------|--------|:--------------:|
| fast | arch-guard, drift-detect, doctor, topology-doctor, contract-audit, runtime-bindings, coverage-map | No |
| ci | All fast checks + snapshot + baseline-drift | No |
| deep | All ci checks + smoke-e2e | Yes |

### Governance Alignment

The following raccoon-cli capabilities directly support governance:

| Governance Concern | Analyzer | What It Checks |
|-------------------|----------|----------------|
| Layer isolation | arch-guard | Import direction, infra type leakage |
| Naming conventions | drift-detect | Defunct names, naming identity |
| Document existence | drift-detect | ARCH_DOCS + domain-specific doc lists |
| Stream/durable alignment | topology-doctor, runtime-bindings | NATS wiring consistency |
| Contract integrity | contract-audit, contract-usage-map | Message contract completeness |
| Domain artifact completeness | drift-detect | Per-domain file, subject, bucket, durable checks |
| Structural drift | baseline-drift, snapshot-diff | Semantic changes between snapshots |

### What Raccoon-CLI Should NOT Do

- **Type resolution across packages** — use optional gopls integration (`--lsp`) for deep analysis.
- **Business logic validation** — that's what tests are for.
- **Runtime behavior verification** — that requires the `deep` profile with running infrastructure.
- **Replace architectural judgment** — tooling catches known patterns; novel decisions require thought.

---

## Refinement 5: Stage Governance Integration

### Problem

Stages (S1–S104) are the primary unit of architectural evolution. Each stage has a report in `docs/stages/`. But the relationship between stages and governance artifacts was not explicit.

### Model

Every stage that modifies governance artifacts must:

1. **Document what changed** in the stage report (section: "Changes Made").
2. **Update affected raccoon-cli constants** if governance enforcement changes.
3. **Reference the governance documents modified** in the stage report.
4. **Not create governance artifacts without corresponding tooling** when the convention is mechanically verifiable.

### Not Required

- Stages that only modify implementation code (adding a family, fixing a bug) do not need governance updates unless they change conventions.
- Not every convention needs mechanical enforcement. Conventions about code style, comment quality, or architectural taste are properly enforced through review, not tooling.

---

## Governance Health Indicators

These signals suggest governance is working as intended:

| Indicator | Healthy | Unhealthy |
|-----------|---------|-----------|
| `make check` pass rate | >95% on first run | Frequent false positives or ignored failures |
| Playbook usage | Developers follow playbooks and report gaps | Developers skip playbooks and cargo-cult from existing code |
| Anti-pattern recurrence | New instances of documented anti-patterns are rare | Same anti-patterns keep appearing despite documentation |
| Documentation freshness | Docs match code behavior | Docs describe conventions that code no longer follows |
| raccoon-cli maintenance burden | Updates are incremental and scoped | Every code change requires extensive raccoon-cli rewrites |

---

## What This Refinement Did NOT Change

1. **The layer model** (domain → application → adapters → actors → interfaces) — stable and validated.
2. **The 6-phase runtime lifecycle** — working correctly for all 6 runtimes.
3. **The catalog-driven assembly pattern** — proven effective for current scale.
4. **The naming conventions** — stable since S98.
5. **The stage-based evolution model** — continues to work well for serialized development.

These are load-bearing structures. Modifying them requires evidence of failure, not improvement ideas.

---

## Related Documents

- `expansion-playbooks-refined.md` — refined expansion guidance
- `structural-anti-patterns-and-when-not-to-expand.md` — when not to expand
- `structural-gains-tradeoffs-and-open-debts.md` — consolidation accounting
- `next-technical-wave-recommendations.md` — what comes after governance refinement
