# Stage C11 Documentation System Hardening Report

## Summary

Stage C11 hardened the `market-foundry` documentation system so it behaves more
like a governable information architecture and less like a large collection of
good-but-loosely-related documents.

The stage did not rewrite the architecture corpus. It focused on:

- canonical entrypoints by document type;
- stronger taxonomy and naming guidance;
- clearer links between operations, tooling, architecture, stages, and archive;
- reduced governance duplication;
- maintenance rules reinforced by repository checks.

## Diagnosis

Before C11, the repository already had strong documentation depth and multiple
useful indexes. The remaining weakness was systemic:

1. documentation governance was spread across several valid documents with no
   single obvious canonical authority;
2. the repository lacked one explicit entrypoint map from document type to
   canonical owner;
3. root docs, operations docs, and architecture governance all carried parts of
   the taxonomy, which increased overlap;
4. historical documentation-system documents remained useful but were not
   clearly framed as baseline or historical context;
5. the lightweight repository checks did not yet require the new hardened
   documentation-system entrypoints.

## Changes Applied

### New canonical documentation-system documents

Added:

- `docs/operations/documentation-system-hardening.md`
- `docs/operations/documentation-governance-entrypoints-and-taxonomy.md`

These two files now provide the canonical documentation-system map and the
canonical governance/taxonomy rules.

### Index and entrypoint updates

Updated:

- `README.md`
- `DEVELOPMENT.md`
- `docs/README.md`
- `docs/operations/README.md`
- `docs/architecture/README.md`
- `docs/architecture/monorepo-documentation-and-stage-governance.md`

This made the new C11 documentation-system surfaces first-class repository
entrypoints instead of secondary references.

### Duplication reduction

Updated:

- `docs/operations/documentation-taxonomy-and-authoring-conventions.md`
- `docs/operations/documentation-reorganization-and-operational-navigation.md`

These docs remain useful, but now clearly point to the C11 documents as the
current canonical governance surface.

### Guard-rail integration

Updated:

- `Makefile`
- `scripts/bootstrap-check.sh`
- `scripts/repository-consistency-check.sh`
- `docs/stages/INDEX.md`

This wired the new documentation-system entrypoints and the C11 report into the
same lightweight guard rails that already protected the broader support surface.

## Validation

Validation executed for C11:

- `./scripts/repository-consistency-check.sh`

Observed outcome:

- required repository documents present;
- stage index aligned after adding C11;
- support-doc links resolved after wiring the new entrypoints;
- Makefile and bootstrap surfaces now include the new documentation-system docs.

## Outcome

After C11:

- the repository has a clearer canonical path for documentation navigation;
- operations owns the documentation system as a support/governance concern;
- architecture remains the canonical source for system design and binding rules;
- stages and archive are more explicitly historical, not competing sources of
  current truth;
- future documentation growth has simpler rules and lower entropy risk.

## Preparation For C12

1. use the new C11 entrypoint map as the default lens before creating any new
   support or governance doc;
2. if future documentation growth creates new clusters inside
   `docs/architecture/`, prefer sub-indexing and canonical maps before moving
   files;
3. keep measuring whether additional repository-consistency checks should verify
   canonical-source markers or duplicate-governance drift, but only if real
   entropy reappears.
