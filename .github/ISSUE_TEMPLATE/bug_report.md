---
name: Bug report
about: Report a defect in market-foundry
title: "[BUG] "
labels: bug
---

<!--
Bug report template.
Before submitting, check docs/RESUMPTION.md for known gaps (G1-G6).
If your bug matches a known gap, mention it explicitly.
-->

## Summary

<!-- One sentence describing the bug. -->

## Expected behavior

<!-- What should happen. -->

## Observed behavior

<!-- What actually happens. Include exact error messages or
     output if possible. -->

## Reproduction steps

1. <!-- step 1 -->
2. <!-- step 2 -->
3. <!-- step 3 -->

## Environment

- **Mode:** <!-- paper / testnet / mainnet / mainnet-dry-run -->
- **Compose variant:** <!-- default / mainnet-dry-run / mainnet-live / unified / venue-live -->
- **Affected binary:** <!-- gateway / ingest / derive / store / execute / writer / configctl / migrate -->
- **Commit SHA:** <!-- output of `git rev-parse HEAD` -->

## Diagnostic snapshot

<!-- If applicable, attach output of `make diag` or relevant
     `make logs SERVICE=<name>` excerpts. -->

```
<!-- paste output here -->
```

## Related to a known gap?

<!-- If this bug relates to G1-G6 in docs/RESUMPTION.md,
     name the gap. Otherwise leave "No". -->

No
