# O12 Report

## Scope

O12 applies light hardening to `.opencode` so it keeps tracking the real
`market-foundry` codebase without becoming a second governance layer.

## Delivered

1. Lightweight invariants were added to [`.opencode/README.md`](./README.md).
2. Orphaned context coverage was fixed by linking all compressed context files
   from the relevant navigation entrypoints.
3. `scripts/opencode-consistency-check.sh` was added and wired into
   `make repo-consistency-check` through
   [`../scripts/repository-consistency-check.sh`](../scripts/repository-consistency-check.sh).
4. The existing guard-rail owner docs now mention the `.opencode` checks and
   their intentionally narrow scope.

## Validation

- `../scripts/opencode-consistency-check.sh`
- `../scripts/repository-consistency-check.sh`

## Limitations

- The new check validates only cheap, objective invariants; it does not judge
  whether a summary is the best possible summary.
- It does not scan the full repository for all broken links; it stays on the
  `.opencode` surface plus the pre-existing primary-doc guard rail.
- It uses a thin-size threshold to catch obvious owner-doc duplication, not a
  semantic diff against canonical docs.

## Next Steps

- Extend the check only if real drift recurs and the new rule is equally cheap
  and objective.
- If `.opencode` gains new profiles or context areas, wire them from profile or
  navigation entrypoints in the same change.
- If owner docs or the public `make` surface change, update `.opencode` in the
  same commit so the guard rail stays green.
