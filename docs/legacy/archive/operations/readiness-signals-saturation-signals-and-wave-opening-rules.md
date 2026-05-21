# Readiness Signals, Saturation Signals, And Wave Opening Rules

## Purpose

This document turns the C30 development-platform readiness model into a compact
decision guide for opening or delaying future waves in `market-foundry`.

Use it when a strategic checkpoint reaches the question:
should the repository open the next wave now, after one prerequisite, or not
yet?

This is a repository-platform decision tool. It does not evaluate domain
correctness, runtime feature completeness, or product delivery value.

## Decision Boundary

Ask this first:

is the next decision about the health of the development platform, or about the
functional readiness of the system?

Use this document only when the concern is platform-side, such as:

- workflow stability;
- trust in entrypoints or wrappers;
- documentation ownership;
- CLI/support-surface drift;
- stage/proof governance load;
- support-surface saturation before expansion.

## Quick Readiness Pass

Before opening a new wave, scan these seven questions:

1. Is the normal workflow still predictable?
2. Are public entrypoints still reliable and coherent?
3. Are active docs still the current source of truth?
4. Is tooling still trusted and bounded?
5. Can proofs and stage artifacts still be governed cleanly?
6. Is maintenance fan-out still proportionate?
7. Can the repository absorb the new wave without obvious support drift?

If the answer is "no" in more than one area, default to "do not open yet"
unless the issues are demonstrably isolated and already contained.

## Readiness Signals

These signals support opening the next wave.

### Workflow signals

- The default change loop is still clear: `make check`, `make tdd`, implement,
  `make verify`.
- Contributors do not need script-first or CLI-first recovery paths for normal
  work.
- Proof-of-record remains explicit and narrow enough for the change at hand.

### Entrypoint signals

- `README.md`, `DEVELOPMENT.md`, `make`, and operations indexes describe one
  coherent workflow story.
- Public wrappers map to real maintained behavior.
- Troubleshooting entrypoints remain obvious and proportional.

### Documentation signals

- Active repository-platform rules live in `docs/operations/`.
- New platform guidance has an explicit owner and is indexed.
- Stage reports are evidence and rationale, not the only place where a current
  rule can be found.

### Tooling signals

- `raccoon-cli` remains trustworthy as inspection/governance tooling.
- `make` remains the public workflow surface.
- Helper growth is being absorbed by existing surfaces when possible.

### Governance signals

- Stage continuity and closure still work without exceptional manual handling.
- The current wave can promote durable rules into canonical docs at closure.
- Repository checks still catch real drift and remain cheap enough to run
  habitually.

### Expansion signals

- The next wave does not require a new support surface unless its owner is
  already explicit.
- Known platform hotspots are either quiet or already locally contained.
- The new wave can ride the current operating model instead of stretching it.

## Saturation Signals

These signals argue for delaying wave opening until the platform is stabilized.

### Workflow saturation

- Contributors must remember exceptions to use the normal workflow safely.
- Narrow proof selection is no longer obvious.
- Validation trust depends on tribal knowledge rather than on canonical docs.

### Entrypoint saturation

- Several surfaces claim to be the default answer for the same task.
- Raw scripts or direct CLI calls are becoming the normal path.
- Root entrypoints increasingly need reconciliations to stay aligned.

### Documentation saturation

- Important active rules are staying inside stage reports or recent wave notes.
- Indexes lag behind active support docs.
- The repository increasingly explains overlap instead of removing it.

### Tooling saturation

- `raccoon-cli` or helper surfaces are taking on workflow duties that belong to
  `make` or docs.
- Wrapper behavior and documentation drift apart.
- Contributors avoid guard rails because failures are noisy or unclear.

### Governance saturation

- Stage closure requires too much manual reconciliation across docs and index
  surfaces.
- Evidence/proof conventions are becoming harder to narrate than the change
  itself.
- The next wave would expand a hotspot already showing repeated friction.

### Structural-cost saturation

- Small platform changes require wide edit fan-out.
- New docs or wrappers are added faster than old overlap is trimmed.
- Each new wave leaves behind more support burden than durable clarity.

## Wave Opening Rules

Apply these rules in order.

### Rule 1. Open immediately only when the platform is clearly ready

Open the next wave when:

- no critical saturation signal is present;
- no more than one dimension shows mild drift;
- the next wave fits inside existing canonical owners;
- the repository can explain and validate the wave through current entrypoints.

### Rule 2. Prefer one prerequisite correction over a support-heavy detour

Delay wave opening briefly when:

- one hotspot is known and likely to worsen under new load;
- the correction is local and low-cost;
- the platform is otherwise healthy.

Typical prerequisite:

- one doc/index alignment;
- one wrapper or ownership clarification;
- one support-surface consolidation;
- one cheap drift guard.

### Rule 3. Do not open a new wave into unresolved ownership ambiguity

Do not open yet when:

- the next wave implicitly depends on a new support surface;
- canonical ownership of that surface is not clear;
- entrypoint overlap would grow if the wave proceeds now.

First action:
resolve ownership or reject the new surface.

### Rule 4. Do not let product urgency redefine platform readiness

A high-value functional wave is not automatically safe to open if the platform
cannot carry it coherently.

If the repository cannot route contributors through one clear workflow and one
clear documentation path, platform stabilization comes first.

### Rule 5. Prefer consolidation over new ceremony

If the platform is not ready, the first move is usually to simplify or align an
existing surface, not to create a new checklist, review ritual, or dashboard.

### Rule 6. Escalate to a support-focused follow-up only when local action is not enough

Open a support-focused prerequisite stage only when:

- more than one dimension is degrading together;
- the same hotspot has already recurred after local correction;
- the next wave would otherwise amplify platform weakness materially.

## Decision Outputs

Use one of these outputs.

| Output | Meaning | Typical next step |
|---|---|---|
| open wave | platform is ready enough | proceed with normal wave chartering |
| open after one prerequisite | platform needs one bounded fix first | land the fix, then open the wave |
| do not open yet | platform weakness is material | stabilize the hotspot before expansion |

## Practical Examples

### Open wave

Use when:
the next wave adds domain work, but current docs, entrypoints, checks, and
stage support already explain how to execute it.

### Open after one prerequisite

Use when:
the next wave is sensible, but one entrypoint stack inconsistency or one active
doc gap would make execution noisier than necessary.

### Do not open yet

Use when:
the next wave would introduce more platform load into a repository already
showing overlap, trust erosion, or repeated stage-governance friction.

## Relationship To Strategic Checkpoints

This document is the decision layer for the C29 readiness-for-next-wave
checkpoint.

Use the checkpoint model to know when the question should be asked.
Use this C30 rule set to answer the question proportionally.

## Related Documents

- [`development-platform-readiness-model-for-future-foundry-waves.md`](development-platform-readiness-model-for-future-foundry-waves.md)
- [`strategic-checkpoints-for-the-development-platform.md`](strategic-checkpoints-for-the-development-platform.md)
- [`development-platform-checkpoint-triggers-scope-and-decision-model.md`](development-platform-checkpoint-triggers-scope-and-decision-model.md)
- [`repository-platform-governance-health-review-and-sustainability-model.md`](repository-platform-governance-health-review-and-sustainability-model.md)
