# Continuous Prioritization Model For The Development Platform

## Purpose

This document defines the lightweight continuous-prioritization model for the
`market-foundry` repository as the Foundry development platform.

Use it when deciding which future repository-platform improvement should come
first across:

- tooling;
- docs;
- workflow;
- CLI;
- governance;
- support surfaces.

This model exists to make repository evolution more strategic, less reactive,
and less dependent on momentary intuition.

Use it together with:

- [`strategic-operating-model-for-the-repository-as-a-development-platform.md`](strategic-operating-model-for-the-repository-as-a-development-platform.md)
- [`repository-platform-governance-health-review-and-sustainability-model.md`](repository-platform-governance-health-review-and-sustainability-model.md)
- [`strategic-checkpoints-for-the-development-platform.md`](strategic-checkpoints-for-the-development-platform.md)
- [`development-platform-readiness-model-for-future-foundry-waves.md`](development-platform-readiness-model-for-future-foundry-waves.md)
- [`criteria-for-opening-containing-or-rejecting-new-support-surfaces.md`](criteria-for-opening-containing-or-rejecting-new-support-surfaces.md)
- [`prioritization-criteria-buckets-and-decision-examples-for-repo-evolution.md`](prioritization-criteria-buckets-and-decision-examples-for-repo-evolution.md)

## Strategic Problem

The repository already has a meaningful support-system baseline:

- canonical entrypoints;
- health interpretation;
- review cadence;
- support-surface lifecycle rules;
- readiness criteria for future waves;
- criteria for opening or containing new support surfaces.

The remaining gap is prioritization.

The repository needs a practical way to decide:

- which improvement matters most now;
- which changes are quick wins versus structural work;
- when urgency should override normal sequencing;
- when a strategic platform investment should come before another wave.

Without this layer, repository-platform evolution tends to drift toward one of
two failure modes:

- reactive local fixes that solve the latest discomfort but do not improve the
  platform trajectory;
- broad strategic ideas that sound right but compete poorly against immediate
  development friction.

## Historical Pattern Across Codex Waves

The Codex support waves in this repository have repeatedly proposed five
improvement shapes.

### 1. Daily-path clarifications

Examples:

- root-doc and operations-doc navigation;
- `make` discoverability;
- onboarding and troubleshooting guidance;
- canonical workflow clarification.

Typical value:
high reduction of search cost and contributor hesitation.

### 2. Workflow trust and reliability hardening

Examples:

- lightweight repository checks;
- CLI reliability and trustworthiness work;
- smoke/proof ergonomics;
- stage tooling continuity.

Typical value:
higher predictability and greater confidence in the default engineering loop.

### 3. Structural coherence and surface control

Examples:

- command lifecycle rules;
- support-surface lifecycle and sunset strategy;
- criteria for opening or rejecting new support surfaces;
- extension-discipline rules.

Typical value:
entropy containment and lower long-term maintenance burden.

### 4. Sustainability and governance refinements

Examples:

- maintainability economics;
- health and review models;
- checkpoint model;
- readiness model;
- documentation governance refinements.

Typical value:
better long-horizon decisions and less accidental drift.

### 5. Strategic platform framing

Examples:

- repository-as-development-platform framing;
- strategic operating model;
- next-wave readiness and executive checkpoints.

Typical value:
better sequencing of future waves and less local optimization.

The main lesson is that the highest-value improvements have rarely been
"more surface." They have mostly been improvements that made the existing
surface easier to trust, easier to navigate, or cheaper to keep coherent.

## Prioritization Objective

The purpose of continuous prioritization is not to rank every idea precisely.
It is to create a stable decision habit:

1. identify what type of problem is being solved;
2. judge whether the value is immediate, structural, or strategic;
3. compare value against implementation and maintenance cost;
4. choose the smallest high-leverage move that improves the platform.

## Criteria

Evaluate repository-platform proposals with eight practical criteria.

| Criterion | Core question | Why it matters |
|---|---|---|
| impact on daily development | does this improve the normal contributor loop frequently enough to matter? | daily friction compounds quickly |
| friction reduction | does this remove repeated confusion, extra steps, or manual repair? | low-grade friction slows every wave |
| entropy risk reduction | does this prevent sprawl, overlap, or silent drift? | entropy becomes structural debt |
| maintenance cost | will this stay cheap to keep aligned over time? | local wins can create global upkeep |
| predictability improvement | does this make the expected workflow more stable and legible? | predictable systems scale better |
| discoverability gain | does this make the right owner path easier to find? | search cost is recurring platform tax |
| environment reliability | does this improve trust in docs, wrappers, checks, or tooling? | low trust pushes contributors off the canonical path |
| implementation cost | how expensive is the change to ship now? | sequencing must reflect real delivery cost |

## Scoring Style

Do not use numeric scoring.

For each criterion, classify the proposal as:

- high;
- medium;
- low.

Then make a judgment using patterns, not arithmetic.

That keeps the model lightweight while still forcing explicit comparison.

## Priority Buckets

Every proposal should land in one of three buckets.

### Quick win

Shape:

- narrow scope;
- low implementation cost;
- low maintenance cost;
- immediate gain in daily workflow or friction reduction.

Typical examples:

- clarifying an owner doc;
- improving one `make` help path;
- removing a repeated naming ambiguity;
- tightening an existing wrapper or README index.

### Structural improvement

Shape:

- medium scope;
- moderate implementation cost;
- strong effect on predictability, reliability, or entropy control;
- usually affects more than one support surface.

Typical examples:

- consolidating overlapping entrypoints;
- hardening stage/tooling continuity;
- reducing fan-out in docs and support workflows;
- aligning lifecycle rules across CLI, docs, and wrappers.

### Strategic initiative

Shape:

- larger scope or cross-wave importance;
- can unlock several future improvements;
- usually improves prioritization quality, readiness, or platform direction.

Typical examples:

- readiness and checkpoint models;
- support-surface expansion discipline;
- platform-level prioritization model;
- high-value governance updates that change how future waves are chosen.

## Decision Order

Apply this model in five steps.

### 1. Frame the proposal correctly

State:

- the problem;
- the affected surface;
- whether the proposal is a fix, consolidation, extension, or strategic model
  change.

If the problem is unclear, do not prioritize the solution yet.

### 2. Determine the dominant value

Pick the main reason the change matters most:

- daily throughput;
- friction reduction;
- reliability;
- predictability;
- entropy control;
- discoverability;
- future-wave leverage.

The dominant value should be singular even when multiple benefits exist.

### 3. Assign the priority bucket

Choose:

- quick win;
- structural improvement;
- strategic initiative.

This prevents small useful work from competing incorrectly against broader
structural needs.

### 4. Compare value against cost and load

Judge both:

- implementation cost now;
- maintenance cost later.

A proposal with strong value but bad long-term maintenance shape should usually
be contained, simplified, or deferred.

### 5. Decide the next action

The output should be one of:

1. do now;
2. do next;
3. defer intentionally;
4. contain inside another change;
5. reject.

## Urgency Versus Structural Value

Urgency is real, but it should not erase strategy.

Use this rule:

- if the issue breaks or seriously undermines the canonical path, workflow
  trust, or wave execution, treat it as operationally urgent;
- otherwise prioritize by structural and strategic value rather than by recent
  visibility.

Operational urgency should override normal ordering only when one of these is
true:

- the default workflow is unreliable;
- canonical docs or entrypoints are actively misleading;
- checks or stage support are blocking normal execution;
- a known hotspot would be amplified immediately by the next wave.

When urgency does override ordering, choose the smallest durable correction.
Do not use urgency as a reason to open a broader platform program than the
problem requires.

## Selection Heuristics

When two proposals compete, prefer:

1. the one that improves the daily path used most often;
2. the one that removes repeated ambiguity rather than adding another surface;
3. the one that reduces maintenance fan-out;
4. the one that increases trust in the canonical path;
5. the one that better prepares the repository for the next wave.

When a strategic initiative competes with several quick wins, prefer the
strategic initiative only if it will materially improve the quality of future
decisions or unlock several bounded improvements.

## Default Portfolio Shape

Continuous prioritization should normally maintain a mixed portfolio:

- always keep a small stream of quick wins available;
- reserve capacity for one structural improvement when a hotspot is repeating;
- open a strategic initiative only when the platform needs a better decision
  model, readiness rule, or major coherence correction.

This keeps the repository improving without becoming either purely reactive or
purely theoretical.

## Operating Rules For Future Waves

Use this model at three moments:

### During a support-heavy stage

Ask:
is the current change only solving the immediate symptom, or is there a
higher-value structural correction nearby?

### At a strategic checkpoint

Ask:
which candidate improvement has the best ratio of platform value to added
maintenance burden?

### Before opening the next wave

Ask:
is there one platform improvement that would make the next wave materially
safer, cheaper, or more predictable?

## Anti-Bureaucracy Rules

- do not rank large backlogs with fake precision;
- do not require every quick fix to go through a full prioritization ritual;
- do not turn qualitative criteria into a spreadsheet-only exercise;
- do not open strategic work just because it sounds more important;
- do not treat a new support surface as the default answer to friction.

## Recommended Usage

For future Codex waves, the lightweight default is:

1. write the candidate improvement in one sentence;
2. note the dominant value;
3. classify the bucket;
4. assess value and cost qualitatively;
5. choose do now, do next, defer, contain, or reject.

Use the companion examples document when the comparison is not obvious.
