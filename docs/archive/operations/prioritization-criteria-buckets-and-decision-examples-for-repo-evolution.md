# Prioritization Criteria, Buckets, And Decision Examples For Repo Evolution

## Purpose

This document makes the C32 prioritization model operational by showing:

- how to read the prioritization criteria in practice;
- how quick wins, structural improvements, and strategic initiatives differ;
- how to choose between urgency and long-term value;
- how analogous repository-platform proposals should be ordered.

Use it with:

- [`continuous-prioritization-model-for-the-development-platform.md`](continuous-prioritization-model-for-the-development-platform.md)
- [`repository-platform-governance-health-review-and-sustainability-model.md`](repository-platform-governance-health-review-and-sustainability-model.md)
- [`development-platform-readiness-model-for-future-foundry-waves.md`](development-platform-readiness-model-for-future-foundry-waves.md)
- [`criteria-for-opening-containing-or-rejecting-new-support-surfaces.md`](criteria-for-opening-containing-or-rejecting-new-support-surfaces.md)

## How To Read The Criteria

### Impact On Daily Development

High when:

- the improvement changes the default contributor path;
- the issue appears in most stages or normal validation loops;
- contributors are likely to hit the friction without specialized context.

Low when:

- the problem only affects rare expert use;
- the issue is stage-local or highly situational.

### Friction Reduction

High when:

- the improvement removes repeated branch decisions, manual repair, or
  navigation overhead;
- the same question keeps returning in nearby forms.

Low when:

- the proposal mostly feels nicer but does not remove real repetition.

### Entropy Risk Reduction

High when:

- the change prevents support-surface duplication;
- the repository is accumulating nearby owners for the same problem;
- silent drift is already plausible.

Low when:

- the proposal is isolated and does not change the repository shape.

### Maintenance Cost

High cost when:

- the change creates new docs, wrappers, checks, help text, and cross-links
  that must stay aligned;
- upkeep will spread across more than one owner surface.

Low cost when:

- one owner doc or one canonical surface absorbs the change cleanly.

### Predictability Improvement

High when:

- the change makes the expected engineering loop more obvious;
- contributors can better predict which command, doc, or proof path should be
  used.

Low when:

- the platform still offers several plausible answers after the change.

### Discoverability

High when:

- the change makes the correct owner path easier to find from canonical
  entrypoints;
- it reduces time spent searching documentation or command surfaces.

Low when:

- it only helps contributors who already know where to look.

### Environment Reliability

High when:

- the change improves trust in checks, wrappers, docs, or support tooling;
- it reduces the chance that contributors abandon the canonical path.

Low when:

- it does not materially change whether the platform feels safe to rely on.

### Implementation Cost

High when:

- the change spans several surfaces;
- it requires non-trivial coordination or new governance burden;
- the repository would need follow-up work before the change pays off.

Low when:

- the change is local and can be validated cheaply.

## Bucket Rules

### Quick wins

Choose this bucket when:

- value is immediate;
- scope is narrow;
- maintenance burden stays local;
- the platform shape does not materially change.

Decision bias:
prefer these when they reduce recurring daily friction cheaply.

### Structural improvements

Choose this bucket when:

- a hotspot is repeating;
- the change improves coherence, reliability, or entropy control;
- the result reduces future maintenance or ambiguity across more than one
  surface.

Decision bias:
prefer these when local fixes are starting to repeat.

### Strategic initiatives

Choose this bucket when:

- the repository needs a better rule for choosing future work;
- the improvement changes how future waves should be sequenced;
- the value is cross-wave rather than local.

Decision bias:
open these sparingly and only when they clearly upgrade platform judgment.

## Decision Examples

### Example 1: improve one confusing `make` help path

Likely profile:

- daily impact: high;
- friction reduction: high;
- entropy risk reduction: low;
- maintenance cost: low;
- predictability: medium to high;
- discoverability: high;
- reliability: low to medium;
- implementation cost: low.

Decision:
quick win, usually do now.

Why:
the gain is immediate, local, and cheap.

### Example 2: add a new helper script for a narrow runtime variant

Likely profile:

- daily impact: low to medium;
- friction reduction: medium for one case;
- entropy risk reduction: negative;
- maintenance cost: high;
- predictability: low;
- discoverability: low;
- reliability: low;
- implementation cost: low now, higher later.

Decision:
usually reject or contain.

Why:
creation is easy, but long-term platform shape gets worse.

### Example 3: consolidate overlapping proof or smoke entrypoints

Likely profile:

- daily impact: medium to high;
- friction reduction: high;
- entropy risk reduction: high;
- maintenance cost: medium now, lower later;
- predictability: high;
- discoverability: medium to high;
- reliability: medium to high;
- implementation cost: medium.

Decision:
structural improvement, usually do next or do now if the overlap is already
hurting trust.

### Example 4: create a new governance document that only restates active rules

Likely profile:

- daily impact: low;
- friction reduction: low;
- entropy risk reduction: low or negative;
- maintenance cost: medium to high;
- predictability: low;
- discoverability: low or negative;
- reliability: low;
- implementation cost: low.

Decision:
reject.

Why:
this adds surface without adding differentiated value.

### Example 5: harden a noisy or stale lightweight check

Likely profile:

- daily impact: high when the check is on the default path;
- friction reduction: high;
- entropy risk reduction: medium;
- maintenance cost: low to medium;
- predictability: high;
- discoverability: low;
- reliability: high;
- implementation cost: low to medium.

Decision:
quick win or structural improvement depending on scope.

Why:
trust in guard rails is a platform-multiplier.

### Example 6: define a new readiness or prioritization rule before a larger wave

Likely profile:

- daily impact: medium;
- friction reduction: medium;
- entropy risk reduction: high;
- maintenance cost: low to medium;
- predictability: high;
- discoverability: medium;
- reliability: medium;
- implementation cost: medium.

Decision:
strategic initiative.

Why:
it improves decision quality for several future changes, not only one local
problem.

## Urgency Override Examples

### Urgent and should jump the queue

- `make check` or `make verify` stops being credible as the default validation
  path;
- canonical docs point contributors to broken or misleading commands;
- stage continuity support is failing in a way that blocks wave execution;
- a live hotspot will be amplified by the next already-planned wave.

### Visible but should not jump the queue

- desire for one more convenience command when a current owner can absorb it;
- preference for new taxonomy wording without current ambiguity;
- a new support document for a rule that already has an owner;
- a niche expert path that does not affect the normal contributor loop.

## Recommended Comparison Pattern

When two repository-platform improvements compete, compare them in this order:

1. which one improves the default development path more often;
2. which one reduces future entropy more credibly;
3. which one lowers maintenance fan-out rather than increasing it;
4. which one improves readiness for the next wave;
5. which one can be shipped with the smaller durable change.

## Decision Templates

### Template: do now

Use when:

- value is high and immediate;
- cost is low enough;
- deferral would keep recurring friction in the daily path.

### Template: do next

Use when:

- the change is structural or strategic;
- it is not blocking today, but it should precede another support-heavy wave.

### Template: defer intentionally

Use when:

- the idea is valid;
- value exists, but the current wave has a higher-leverage item first.

### Template: contain

Use when:

- the need is real;
- an existing owner surface can absorb it more cheaply.

### Template: reject

Use when:

- differentiated value is weak;
- maintenance cost is disproportionate;
- the change would mostly add another plausible path.

## Portfolio Guidance For Future Waves

For future Codex planning, a healthy platform backlog usually looks like:

- a short list of ready quick wins;
- one active structural hotspot at a time;
- a strategic initiative only when the repository lacks a decision rule,
  readiness rule, or coherence model needed for the next sequence of waves.

This keeps the Foundry repository strategic without making platform evolution
ceremonial.
