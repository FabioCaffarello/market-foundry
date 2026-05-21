# Stage C35 Report: Documentary Topology Compression And Canonical Navigation Hardening

## Summary

Stage C35 treated the repository explicitly as a development platform and
hardened the documentation topology around that operating model.

The change stayed intentionally conservative:

- define documentary ownership by subject in one explicit owner doc;
- compress high-fan-out indexes so they route to owners instead of repeating
  large catalogs;
- separate owner docs from reference and historical docs without removing
  useful history;
- keep the active support catalog in one place instead of letting root docs and
  helper surfaces become secondary indexes.

## Diagnosis

The repository already had strong documentary depth, but the active navigation
layer had started to drift toward catalog duplication.

The main issues were:

- `README.md`, `DEVELOPMENT.md`, `docs/README.md`, `docs/operations/README.md`,
  and `make docs` all carried overlapping slices of the same operational and
  strategic doc inventory;
- some older governance docs still mattered, but their role as historical
  bridge docs was not explicit enough;
- operations and docs indexes mixed subject ownership with exhaustive listing,
  which made the navigation layer harder to maintain than the underlying
  documents themselves;
- the owner/reference split was implied by convention rather than stated as an
  explicit map.

This was a maintenance risk, not a content-depth problem.

## Scope Boundaries

### In scope

- define a canonical ownership map by subject;
- reduce repeated navigation catalogs in root docs and active indexes;
- preserve historical docs and stage history as non-owner surfaces;
- keep the changes compatible with existing guard rails and index discipline.

### Out of scope

- mass-moving documents between directories;
- deleting historical governance or reorganization docs;
- collapsing the support corpus into one mega-document;
- changing architecture ownership or stage-history semantics.

## Decisions

### Decision 1. Add one explicit ownership map instead of expanding existing indexes

The new canonical owner for "who owns this subject?" is:

- `docs/operations/documentary-ownership-and-canonical-navigation.md`

This keeps ownership logic in one short doc instead of forcing every index to
restate it.

### Decision 2. Keep exactly one detailed active support catalog

`docs/operations/README.md` remains the detailed active support index.

Root docs and `docs/README.md` now stay curated and point into that catalog or
into owner docs. `make docs` was reduced to a true shortlist again.

### Decision 3. Treat older doc-governance docs as historical bridge surfaces

The older reorganization and hardening docs remain useful because they explain
how the system got here. They now read as bridge/history material, not as
competing owners for current navigation.

### Decision 4. Preserve directory ownership while separating owner and reference docs

This stage does not relocate operations, tooling, architecture, or stage files.
Instead it hardens the split through naming, index wording, and explicit
owner/reference tables.

## Changes Applied

- added `docs/operations/documentary-ownership-and-canonical-navigation.md`;
- compressed `docs/README.md` into a routing surface instead of a long
  cross-surface catalog;
- reorganized `docs/operations/README.md` around owner docs, reference docs,
  and historical bridge docs while still indexing all active operations docs;
- clarified owner/reference split in `docs/tooling/README.md`;
- updated `README.md` and `DEVELOPMENT.md` to point to the ownership map
  instead of expanding more cross-surface narrative;
- reduced `make docs` to a smaller canonical shortlist;
- added the new owner doc to bootstrap and repository-consistency required-path
  checks;
- added this stage to `docs/stages/INDEX.md`.

## Ownership Outcome

### Canonical owners after C35

- repository identity: `README.md`
- daily workflow: `DEVELOPMENT.md`
- cross-surface doc routing: `docs/README.md`
- ownership-by-subject map: `docs/operations/documentary-ownership-and-canonical-navigation.md`
- active support catalog: `docs/operations/README.md`
- tooling-internal catalog: `docs/tooling/README.md`
- architecture corpus entrypoint: `docs/architecture/README.md`
- immutable history index: `docs/stages/INDEX.md`

### Reference and history split after C35

- reference docs remain in their existing directories and are grouped under the
  owner docs that they deepen;
- historical governance and reorganization docs remain active for rationale but
  are no longer treated as first-stop owners;
- stage reports remain the evidence trail, not the current operating contract.

## Tradeoffs

### Gains

- lower fan-out when a canonical topic changes;
- clearer separation between routing, ownership, reference depth, and history;
- less pressure to edit root docs for every new support document;
- better alignment with the repository-as-development-platform model.

### Costs

- one additional owner doc exists in `docs/operations/`;
- readers now rely slightly more on explicit routing into the detailed
  operations catalog instead of seeing large doc inventories everywhere.

### Why the tradeoff is acceptable

One small ownership map is materially cheaper than keeping four or five
competing summary surfaces synchronized.

## Guard Rails

- keep `docs/operations/README.md` as the only detailed active support catalog;
- keep `README.md`, `DEVELOPMENT.md`, `docs/README.md`, and `make docs` curated;
- add new durable concerns first to the ownership map, then to the active
  support catalog if needed;
- preserve historical bridge docs, but avoid promoting them back into
  first-stop entrypoints;
- do not move or delete stage history to solve navigation problems that can be
  solved with routing and ownership clarity.

## Validation

- `make repo-consistency-check`
- `make stage-check STAGE_ID=C35 STAGE_SLUG=documentary-topology-compression-and-canonical-navigation-hardening STAGE_REQUIRE=docs/operations/documentary-ownership-and-canonical-navigation.md,docs/stages/stage-c35-documentary-topology-compression-and-canonical-navigation-hardening-report.md`

## Preparation For Next Stage

1. If a future stage adds a durable support concern, update the ownership map
   before broadening any index.
2. If `docs/operations/README.md` grows past useful scan length again, compress
   it by grouping or bridging, not by cloning the catalog into new indexes.
3. If root docs start to reacquire large support catalogs, treat that as drift
   against the C21/C35 maintenance model.
