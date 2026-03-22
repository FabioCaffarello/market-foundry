# Strategic Operating Model For The Repository As A Development Platform

## Purpose

This document consolidates the long-term operating model for the
`market-foundry` repository as the development platform of the Foundry.

The goal is practical coherence. The repository should stay easy to operate,
maintain, evolve, and protect without becoming a parallel product or a heavy
governance program.

## Strategic Position

The repository is not just where application code lives. It is the active
development platform through which contributors:

- discover the canonical workflow;
- build and validate changes;
- run proofs and recover from failures;
- navigate architecture, tooling, and stage history;
- evolve support surfaces that keep engineering work governable.

The repository therefore needs an operating model for itself, not only for the
runtime system.

## Operating Goal

Preserve five properties at the same time:

1. contributors can enter the correct workflow quickly;
2. the public support surface stays small and coherent;
3. support tooling evolves without uncontrolled fan-out;
4. active documentation remains durable, indexed, and current;
5. repository entropy is contained through light recurring discipline.

## Strategic Foundations Consolidated From Earlier Waves

The current model stands on the support-platform waves already completed:

- C20: lightweight automation for continuity and stage hygiene;
- C21: structural-cost control and support-surface economics;
- C22: tooling-extension discipline across `make`, scripts, docs, checks, and
  "do nothing";
- C23: `raccoon-cli` command lifecycle and deprecation discipline;
- C24: long-term documentation and operational sustainability;
- C25: strategic health model for the developer environment;
- C26: periodic review cadence, triggers, and proportional follow-through;
- C27: support-surface lifecycle states, sunset, consolidation, and retirement.

C28 does not replace those layers. It turns them into one repository operating
model with explicit ownership, rhythms, and decision boundaries.

## Canonical, Periodic, And Flexible Layers

### What should stay canonical

These surfaces define the stable operating contract of the repository:

- `make` as the canonical public workflow surface;
- `README.md`, `DEVELOPMENT.md`, `docs/README.md`, and
  `docs/operations/README.md` as the entrypoint stack;
- `docs/architecture/` for binding architecture and governance;
- `docs/tooling/` for tool-internal rules;
- `docs/stages/INDEX.md` plus stage reports as historical evidence only;
- `scripts/` as harness implementation behind canonical workflows;
- `raccoon-cli` as the expert structural-analysis and governance surface.

Canonical rule:
one recurring repository question should have one obvious public starting point.

### What should be reviewed periodically

These surfaces drift through accumulation and therefore require recurring
review:

- repository health dimensions and their practical signals;
- `Makefile` target shape and alias value;
- script and harness sprawl;
- `raccoon-cli` command maturity and lifecycle;
- documentation indexes, entrypoints, and active-doc placement;
- support-surface lifecycle states: active, auxiliary, legacy, retired;
- lightweight checks and consistency coverage;
- low-volume automation and stage-support helpers.

### What should remain flexible

These areas should stay adaptable and problem-driven:

- exact helper wording and UX in docs or scripts;
- narrow advisory automation when it removes repeated friction;
- local harness flags and debugging paths;
- which hotspot deserves the next support-focused cleanup wave;
- whether a change is solved by clarify, align, consolidate, guard, or
  governed follow-up.

Flexible rule:
adapt tactics freely without weakening the canonical ownership model.

## Operating Model

### 1. Ownership model

The repository platform stays governable when ownership is explicit:

| Concern | Owning surface | Primary responsibility |
|---|---|---|
| public development workflow | `Makefile` + `DEVELOPMENT.md` | stable contributor path |
| harness implementation | `scripts/` + `scripts/README.md` | implementation and debug depth |
| structural analysis and machine-readable governance | `raccoon-cli` + `docs/tooling/` | repository guard rails and inspection |
| active repository operations guidance | `docs/operations/` | workflow, navigation, support policy |
| binding system rules | `docs/architecture/` | architectural truth and runtime governance |
| historical evidence | `docs/stages/` | traceability and closure record |

Ownership rule:
extend the smallest correct owner before creating a sibling surface.

### 2. Rhythm model

The repository platform operates on three lightweight rhythms.

#### Continuous rhythm

Applied during normal change work:

- use `make check`, `make tdd`, and `make verify`;
- review local doc/index alignment when changing support surfaces;
- prefer the narrowest smoke or proof that matches the change;
- use `make stage-status` and `make stage-check` when the work belongs to a
  governed stage.

#### Periodic rhythm

Applied when review triggers from C26 appear:

- growth in commands, wrappers, docs, or checks;
- drift between docs, behavior, and supported entrypoints;
- discoverability degradation;
- operational confusion;
- duplication or reliability erosion.

The expected response remains proportional:
clarify, align, consolidate, guard, or open a small governed follow-up.

#### Strategic rhythm

Applied at support-heavy wave closure or when several periodic signals recur
together:

- re-evaluate repository health at platform level;
- identify the most expensive hotspot;
- decide whether the next step is editorial, lifecycle, tooling, or guard-rail
  oriented;
- avoid opening a broad support-program if a small correction is sufficient.

### 3. Decision model

When a repository-platform change is proposed, decide in this order:

1. what recurring problem is being solved;
2. which current owner surface should absorb it;
3. whether the issue is canonicality, review cadence, lifecycle, or health;
4. whether the smallest valid response is clarify, align, consolidate, guard,
   or do nothing;
5. whether the change introduces more maintenance cost than it removes.

This decision model keeps the platform intentional instead of additive.

### 4. Protection model

The repository should resist entropy through light protection only:

- canonical indexes in area READMEs;
- curated root entrypoints;
- lightweight consistency checks for objective drift;
- lifecycle rules for aliases, wrappers, docs, and compatibility surfaces;
- promotion of lasting rules out of stage reports into canonical docs.

Protection rule:
guard stable, objective, silent-drift invariants only. Keep subjective
judgment manual.

## Governance By Support Surface

### Repository health

Use the C25 health dimensions as the main diagnostic lens:

- discoverability;
- operational reliability;
- entrypoint coherence;
- navigability;
- documentation governance;
- tooling sustainability;
- maintenance cost control.

Health is used for prioritization, not scoring.

### Review cadence

Use C26 as the cadence model:

- local review inside normal work;
- strategic periodic review only when triggers justify it;
- proportional follow-through, never ceremony by default.

### CLI governance

`raccoon-cli` stays bounded:

- `stable core` for recurring structural/governance workflows;
- `stable utility` for durable specialist commands;
- `legacy` only for compatibility or migration;
- expansion only after overlap review and lifecycle justification.

The CLI must not become a parallel operator control plane.

### Makefile, scripts, and harness governance

- `make` remains the public contract;
- scripts remain implementation-facing or expert-facing;
- harnesses should consolidate before forking;
- aliases must keep paying for their maintenance cost;
- proof-of-record stays with the canonical `make smoke*` layer.

### Documentation and navigation

- active rules live in active canonical docs;
- area READMEs own discoverability for their domain;
- root docs stay curated and shallow;
- stage reports remain evidence, not active operating instructions.

### Support-surface lifecycle

Every support surface should fit one explicit state:

- active canonical;
- active auxiliary;
- legacy;
- retired.

Default response to overlap is consolidation, not additive coexistence.

### Lightweight automation

Automation is justified when the routine is frequent, objective, and
transparent. It should:

- reduce continuity friction;
- expose what it checked;
- avoid hidden state or orchestration layers;
- live behind `make` or a simple documented script.

## Repository Platform Responsibilities

The repository platform should make these responsibilities explicit even when
the same contributors play several roles:

| Responsibility | Expected behavior |
|---|---|
| platform stewarding | preserve canonical entrypoints and support-surface coherence |
| workflow maintenance | keep `make`, scripts, and docs aligned |
| tooling governance | keep CLI/checks objective, bounded, and trusted |
| documentation sustainability | promote lasting rules and keep indexes current |
| lifecycle discipline | consolidate or demote surfaces before they become drift debt |

These are repository responsibilities, not separate team structures.

## Long-Term Success Conditions

The operating model is working when:

- normal contributors can work from canonical entrypoints without history
  archaeology;
- support changes usually touch one owner doc plus one behavior surface;
- reviews focus on real drift and real overlap rather than theoretical process;
- the CLI, Makefile, scripts, docs, and checks behave like one system;
- support-stage outputs increasingly unify the platform instead of adding more
  permanent fragments.

## Boundaries

This operating model must not become:

- a product roadmap for repository tooling itself;
- a workflow engine for stages or repository governance;
- a mandatory review calendar with logs and approvals;
- a reason to refactor functional architecture;
- a platform with more ceremony than the development work it supports.

## Canonical Companion

Use the companion document for the applied governance, health, review, and
sustainability rules:

- [`repository-platform-governance-health-review-and-sustainability-model.md`](repository-platform-governance-health-review-and-sustainability-model.md)

## Related Documents

- [`long-term-documentation-and-operational-sustainability-model.md`](long-term-documentation-and-operational-sustainability-model.md)
- [`developer-environment-strategic-health-model.md`](developer-environment-strategic-health-model.md)
- [`repository-health-dimensions-signals-and-decision-usage.md`](repository-health-dimensions-signals-and-decision-usage.md)
- [`periodic-review-model-for-repository-development-environment.md`](periodic-review-model-for-repository-development-environment.md)
- [`repository-review-cadence-triggers-and-follow-through-rules.md`](repository-review-cadence-triggers-and-follow-through-rules.md)
- [`support-surface-sunset-consolidation-and-retirement-strategy.md`](support-surface-sunset-consolidation-and-retirement-strategy.md)
- [`tooling-evolution-patterns-and-repository-extension-discipline.md`](tooling-evolution-patterns-and-repository-extension-discipline.md)
- [`repository-automation-boundaries-high-value-routines-and-sustainability-rules.md`](repository-automation-boundaries-high-value-routines-and-sustainability-rules.md)
