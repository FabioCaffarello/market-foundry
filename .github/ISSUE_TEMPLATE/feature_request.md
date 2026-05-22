---
name: Feature request
about: Propose a new capability for market-foundry
title: "[FEATURE] "
labels: enhancement
---

<!--
Feature request template.
Before submitting:
- Check docs/RESUMPTION.md → "Deliberate non-features" to see if
  this is a deliberate omission rather than a missing capability.
- Check docs/decisions/ to see if a related ADR already constrains
  what's feasible.
-->

## Motivation

<!-- Why is this capability needed? What does it unlock or improve? -->

## Proposed capability

<!-- What would the feature do? -->

## Use cases

1. <!-- concrete scenario where this would be used -->
2. <!-- ... -->

## Considered alternatives

<!-- What else was considered? Why is this approach preferred? -->

## Structural impact

<!-- Does this require:
     - New domain? (docs/domain/)
     - New stream or KV bucket? (single-writer invariant)
     - New HTTP route? (boot_test.go update)
     - New ADR? (docs/decisions/)
     - Layer changes? (raccoon-cli enforcement)
     - Cross-binary coordination?
-->

## Conflicts with current design?

<!-- Does this conflict with any documented decision in
     docs/decisions/? Examples:
     - ADR 0007 (paper-venue default): does this respect paper-first?
     - ADR 0008 (single-writer): does this introduce a second writer?
     - ADR 0011 (no-OMS in pairing): does this add OMS to pairing?
     If yes, an ADR superseding the existing one would be required.
-->

No conflicts identified.

## Out of scope

<!-- What is this feature NOT supposed to do? Prevents scope creep
     during implementation. -->
