# Stage C32 Report: Continuous Prioritization Model For The Development Platform

## 1. Executive Summary

The Stage C32 consolidated a lightweight continuous-prioritization model for
the `market-foundry` repository as the Foundry development platform.

Stages C25 through C31 already defined health, review cadence, lifecycle,
strategic checkpoints, readiness, and support-surface opening discipline. The
remaining gap was decision ordering. The repository needed a practical model
for evaluating, comparing, and sequencing future improvements across tooling,
docs, workflow, CLI, governance, and support surfaces.

The result of C32 is:

- a canonical prioritization model for the development platform;
- a companion decision-examples document that turns the model into practical
  usage;
- light integration into indexes, governance entrypoints, and repository
  consistency checks.

## 2. Scope Boundaries

### In scope

- analyze the kinds of improvements repeatedly proposed across Codex support
  waves;
- identify which prioritization criteria are actually useful;
- define a lightweight model for continuous prioritization;
- differentiate quick wins, structural improvements, and strategic
  initiatives;
- define how operational urgency should interact with structural value;
- document how future waves should use this model.

### Out of scope

- changes to the functional system architecture;
- creation of a heavy prioritization framework or numeric scorecard;
- broad refactors outside the minimum needed to integrate the model into the
  repository platform;
- automation of prioritization decisions.

### Not changed

- the repository functional architecture;
- the role of `make` as canonical public workflow;
- the role of `scripts/` as harness layer;
- the role of `raccoon-cli` as structural-analysis and governance tooling;
- the role of `docs/stages/` as historical evidence.

## 3. Strategic Diagnosis

The repository no longer lacks support-surface rules. By the end of C31, it
already had enough decision models to govern:

- repository-platform health;
- recurring review;
- support-surface lifecycle;
- strategic checkpoints;
- readiness for future waves;
- whether new support surfaces should be opened at all.

The remaining weakness was prioritization quality.

Without a continuous prioritization model, future repository-platform work
would still be chosen too reactively:

- the most visible friction might win even when a structural fix matters more;
- broad strategic work might be opened without clear leverage;
- low-cost local ideas could compete unfairly with higher-value structural
  improvements;
- urgency could be used to justify unnecessary platform expansion.

The repository therefore needed a decision layer that is:

- light enough for normal use;
- explicit enough to improve quality;
- strategic enough to guide future Codex waves.

## 4. Improvement Patterns Observed Across Waves

The support-history pattern shows that most repository-platform proposals fall
into a small set of recurring shapes:

1. daily-path clarification and discoverability improvements;
2. workflow trust and reliability hardening;
3. structural coherence and support-surface control;
4. sustainability and governance refinements;
5. strategic platform framing for future-wave sequencing.

The key finding is that the best improvements have usually not been additive
surface growth. They have been changes that reduced ambiguity, improved trust
in the canonical path, or lowered maintenance fan-out.

That observation directly shaped the C32 model.

## 5. Prioritization Criteria Chosen

The final model uses eight criteria:

1. impact on daily development;
2. friction reduction;
3. entropy-risk reduction;
4. maintenance cost;
5. predictability improvement;
6. discoverability gain;
7. environment reliability;
8. implementation cost.

These criteria were selected because they are concrete enough to apply without
turning prioritization into bureaucracy, and because together they capture both
value and burden.

Two strategic choices were made:

- keep the evaluation qualitative using `high`, `medium`, and `low`;
- forbid score-first prioritization so judgment remains grounded in repository
  reality rather than spreadsheet precision.

## 6. Model Delivered

The model delivered in C32 works as follows:

1. frame the proposal clearly;
2. identify its dominant value;
3. place it in one bucket:
   - quick win;
   - structural improvement;
   - strategic initiative;
4. compare value against implementation cost and maintenance cost;
5. choose one output:
   - do now;
   - do next;
   - defer intentionally;
   - contain;
   - reject.

The model also defines when urgency overrides normal sequencing:

- only when the canonical path, workflow trust, or near-term wave execution is
  materially at risk;
- and even then, only with the smallest durable correction.

## 7. Changes Applied

Created:

- `docs/operations/continuous-prioritization-model-for-the-development-platform.md`
- `docs/operations/prioritization-criteria-buckets-and-decision-examples-for-repo-evolution.md`
- `docs/stages/stage-c32-continuous-prioritization-model-for-the-development-platform-report.md`

Updated lightly:

- `docs/operations/README.md`
- `docs/README.md`
- `docs/operations/documentation-governance-entrypoints-and-taxonomy.md`
- `docs/operations/repository-platform-governance-health-review-and-sustainability-model.md`
- `Makefile` (`make docs`)
- `scripts/repository-consistency-check.sh`
- `docs/stages/INDEX.md`

These changes made the C32 model discoverable, governable, and protected
against silent omission from canonical indexes.

## 8. Validation

- `make repo-consistency-check`
- `make stage-check STAGE_ID=C32 STAGE_SLUG=continuous-prioritization-model-for-the-development-platform STAGE_REQUIRE=docs/operations/continuous-prioritization-model-for-the-development-platform.md,docs/operations/prioritization-criteria-buckets-and-decision-examples-for-repo-evolution.md`

## 9. Limits And Non-Goals

- C32 does not create a numeric scoring framework.
- C32 does not require a ceremony for every small improvement.
- C32 does not replace technical judgment with templates.
- C32 does not authorize broader support-surface growth by itself.
- C32 does not alter the system runtime or functional architecture.

## 10. Final Outcome

The repository now has a practical way to order future development-platform
improvements.

The main gain is not more governance surface. The main gain is better decision
quality:

- faster separation of quick wins from structural work;
- clearer handling of urgency versus long-term value;
- better sequencing for future Codex waves;
- stronger reinforcement of the repository as a development platform for the
  Foundry.

## 11. Preparation For C33

The next Codex wave should be a selective application wave, not another
abstract governance wave.

Formal recommendation:

open the next wave only if it applies the C32 model to one concrete
repository-platform hotspot that is already recurring, with preference for a
structural improvement that:

- improves workflow trust or canonical-path clarity;
- reduces maintenance fan-out;
- and increases readiness for the next broader Foundry expansion.
