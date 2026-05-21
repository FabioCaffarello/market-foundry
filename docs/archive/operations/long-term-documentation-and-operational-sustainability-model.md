# Long-Term Documentation And Operational Sustainability Model

## Purpose

This document defines the lightweight sustainability model that should keep the
`market-foundry` development environment understandable, governable, and cheap
to evolve across many future waves.

The repository already has canonical entrypoints, support-surface rules, stage
governance, and tooling-evolution discipline. The remaining need is a single
model that ties those pieces together so long-term health does not depend on
contributors remembering implicit norms.

## Sustainability Goal

Keep the repository operable over time by preserving four properties at once:

- contributors can still find the right entrypoint quickly;
- support tooling remains small, intentional, and governable;
- active documentation stays current and indexed;
- the environment resists entropy without introducing a heavy process layer.

This is a sustainability model for repository operations and support surfaces,
not a new governance bureaucracy and not a change to runtime architecture.

## Sustainability Diagnosis

The current repository is already much healthier than an ad hoc monorepo, but
future entropy would likely appear through repeated small additions rather than
one large regression.

The main long-term pressure points are:

- active docs being added without entering the canonical indexes;
- support guidance spreading across root docs, area docs, operations docs, and
  stage reports without an explicit promotion path;
- `make`, scripts, and `raccoon-cli` all gaining new helpers for the same need;
- low-frequency scripts or checks remaining alive without a clear owner surface;
- new waves leaving stage-local support assets behind after the lasting rule was
  already promoted elsewhere;
- high-traffic entrypoints absorbing too much detail and becoming merge
  hotspots.

These are sustainability failures because they increase search cost, implicit
knowledge, and maintenance fan-out at the same time.

## Repository Sustainability Pillars

### 1. Canonical entrypoint discipline

One recurring repository question should have one obvious starting point.

The stable entrypoint stack remains:

- `README.md` for orientation;
- `DEVELOPMENT.md` for the daily engineering loop;
- `docs/README.md` for documentation-surface navigation;
- `docs/operations/README.md` for the detailed active support catalog;
- area `README.md` files for local tree navigation;
- `docs/tooling/README.md` for tooling-internal ownership;
- `docs/stages/INDEX.md` for immutable history only.

Sustainability rule:
do not solve discoverability by promoting several parallel start points for the
same question.

### 2. Promotion-path discipline

Lasting rules must move out of stage-local rationale into a canonical active
document.

The expected path is:

1. discover or prove something during a stage;
2. promote the lasting rule into `docs/operations/`, `docs/tooling/`, or
   `docs/architecture/`;
3. keep the stage report as historical evidence;
4. index the active document in the owning README;
5. protect only the active invariant with lightweight checks when justified.

Without this path, stage reports become accidental workflow dependencies.

### 3. Support-surface hierarchy discipline

Repository health depends on keeping the support layers distinct:

- `make` owns the public workflow contract;
- `scripts/` owns harness implementation and expert debugging entry;
- `raccoon-cli` owns structural analysis and machine-readable governance;
- `docs/operations/` owns active repository-usage rules;
- `docs/tooling/` owns tooling-internal rules;
- `docs/stages/` owns evidence.

Sustainability rule:
new needs should extend the smallest existing owning surface before creating a
new surface.

### 4. Indexed active-doc discipline

Active support docs are sustainable only if contributors can discover them from
the canonical index of their surface.

That means:

- active operations docs must appear in `docs/operations/README.md`;
- active tooling docs must appear in `docs/tooling/README.md`;
- root docs may link only to curated high-value entrypoints and should not act
  as exhaustive inventories.

An unindexed active doc is a latent orphan even if the file itself is good.

### 5. Lightweight review discipline

Long-term health should be preserved by short review routines attached to
normal work, not by separate ceremonies.

The repository should prefer:

- entrypoint review when a public workflow changes;
- index review when an active doc is added or promoted;
- surface-selection review before adding commands, scripts, or checks;
- periodic hotspot review for support areas that accumulate cost faster than the
  rest of the tree.

### 6. Low-fan-out maintenance discipline

Sustainability is strongest when one support change touches one owning doc plus
the behavior it governs.

Preferred pattern:

1. update the behavior or rule;
2. update the owning canonical doc;
3. update the owning index;
4. update root docs only if contributor orientation changed materially;
5. update lightweight checks only if the invariant is now stable and objective.

## Entropy Sources And Control Model

| Entropy source | Failure mode | Lightweight control |
|---|---|---|
| active docs outside indexes | good guidance becomes invisible | require index inclusion in owning README |
| root-doc overgrowth | entrypoints become broad and noisy | keep root docs curated and shallow |
| parallel entrypoints | contributors choose inconsistent paths | preserve `make` as canonical public workflow surface |
| rare scripts without a clear owner | maintenance survives only by memory | require script-to-owner mapping in `scripts/README.md` and operations catalog |
| stage rationale left as active policy | current rules drift into history docs | promote lasting rules into canonical docs and link back only when needed |
| ever-growing lightweight checks | local trust erodes and fixes slow down | admit only objective, cheap, high-signal invariants |
| tooling-surface creep | CLI, scripts, and Make compete | use C22/C23 inclusion and lifecycle rules before extending support tooling |

## Operating Model For Future Waves

When a future wave introduces a repository-support change, apply this order:

1. decide which surface should own the concern;
2. extend the existing canonical owner if possible;
3. add a new doc only if the concern is durable and lacks a clean home;
4. index the active doc in the owning README;
5. keep stage evidence historical;
6. add or extend a lightweight check only if the new invariant is objective and
   likely to drift silently.

This model keeps the repository adaptable while limiting growth-by-accumulation.

## Sustainability Signals

The repository is staying healthy when:

- contributors can still answer "where do I start?" from entrypoints rather
  than stage history or directory scanning;
- new support docs usually require one index update, not broad root-doc churn;
- new workflows usually extend an existing `make` family or script instead of
  creating a sibling surface;
- active docs in `docs/operations/` and `docs/tooling/` remain indexed;
- lightweight checks stay understandable enough that contributors trust them as
  part of habitual work;
- old support stages remain useful as evidence but unnecessary as current
  operational instructions.

## Sustainability Boundaries

This model does not require:

- approval queues for ordinary doc or tooling edits;
- ownership metadata systems outside the repository tree;
- monthly governance ceremonies;
- generated catalogs or registries for every support artifact;
- architecture or service-runtime refactors.

If a sustainability proposal needs those mechanisms, it is probably too heavy
for the current repository scale.

## Relationship To Existing Governance

This document complements and connects existing models rather than replacing
them:

- C19 provides the repository-navigation layer;
- C20 defines where automation helps continuity without replacing judgment;
- C21 defines structural-cost control;
- C22 defines support-surface extension discipline;
- C23 defines `raccoon-cli` command lifecycle discipline.

This C24 model is the umbrella rule: repository sustainability depends on those
governed surfaces staying connected and intentionally maintained together.

## Related Documents

- [`documentation-governance-entrypoints-and-taxonomy.md`](documentation-governance-entrypoints-and-taxonomy.md)
- [`repository-metadata-indexes-and-developer-navigation-system.md`](repository-metadata-indexes-and-developer-navigation-system.md)
- [`repository-maintainability-economics-and-structural-cost-control.md`](repository-maintainability-economics-and-structural-cost-control.md)
- [`repository-maintenance-hotspots-and-cost-reduction-principles.md`](repository-maintenance-hotspots-and-cost-reduction-principles.md)
- [`tooling-evolution-patterns-and-repository-extension-discipline.md`](tooling-evolution-patterns-and-repository-extension-discipline.md)
- [`tooling-inclusion-deprecation-and-consolidation-rules.md`](tooling-inclusion-deprecation-and-consolidation-rules.md)
- [`repository-sustainability-review-routines-and-entropy-control.md`](repository-sustainability-review-routines-and-entropy-control.md)
