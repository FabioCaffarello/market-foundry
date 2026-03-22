# Repository Maintainability Economics And Structural Cost Control

## Purpose

This document defines how `market-foundry` should reason about repository
maintenance cost outside the runtime architecture itself.

The goal is to keep the support surface sustainable as new waves add more docs,
checks, scripts, stage reports, and navigation entrypoints.

## Structural-Economics Model

Repository maintenance cost is driven mostly by repeated edits across support
surfaces, not by single-file size in isolation.

The practical cost buckets are:

- entrypoint duplication: the same support map repeated in root docs, indexes,
  and helper outputs;
- governance drag: checks that keep expanding their required artifact list even
  when older artifacts are already covered elsewhere;
- public-surface inflation: too many equally promoted commands or docs for the
  same question;
- ownership ambiguity: when contributors must infer whether a concern belongs to
  `README.md`, `docs/operations/`, an area `README.md`, or a stage report;
- historical spillover: stage evidence or old support decisions leaking into
  current operating entrypoints.

## Cost Classes

### Justifiable structural cost

This repository should intentionally pay for:

- clear root entrypoints (`README.md`, `DEVELOPMENT.md`, `docs/README.md`);
- one canonical operational index in `docs/operations/README.md`;
- area-local entrypoints such as `cmd/README.md` and `scripts/README.md`;
- lightweight guard rails that protect core entrypoints, links, wrappers, and
  stage index alignment;
- stage evidence in `docs/stages/` because governed delivery depends on it.

This cost is justified because it reduces search time, preserves governance, and
keeps evolution predictable.

### Accidental structural cost

This repository should avoid:

- mirroring the full support-doc catalog in multiple root-level files;
- making `make docs` behave like a second large documentation index;
- growing `repository-consistency-check.sh` by appending historical stage files
  forever;
- forcing root docs to change for every new operational guide;
- keeping old support-stage artifacts on the critical path when canonical docs
  already cover the current rule.

This cost is accidental because it increases edit fan-out without increasing
clarity or safety proportionally.

## Control Rules

### Entrypoint depth

- Root docs should stay shallow and point to canonical indexes.
- Detailed support-document catalogs belong in `docs/operations/README.md`.
- `make docs` should print a curated shortlist of current entrypoints, not an
  exhaustive catalog.

### Guard-rail scope

- Lightweight checks should protect current entrypoints and invariant-bearing
  docs.
- Historical stage reports should be protected by naming/index rules, not by a
  forever-growing required-file list.
- New checks should fail only on high-signal drift that contributors can fix
  locally and quickly.

### Ownership boundaries

- `README.md` owns orientation.
- `DEVELOPMENT.md` owns the daily workflow.
- `docs/operations/README.md` owns the detailed support-document catalog.
- area `README.md` files own local tree orientation.
- `docs/stages/` owns historical evidence, not current workflow authority.

### Change proportionality

- Prefer deleting duplicate support guidance over introducing a new index.
- Prefer tightening one canonical surface over updating many derivative ones.
- Prefer additive docs only when they introduce a new governing concept that
  cannot fit cleanly in the existing canonical index.

## What C21 Changes

C21 reduces structural cost by:

- curating `make docs` down to primary entrypoints;
- moving detailed support-document discovery back to `docs/operations/README.md`
  instead of repeating it in root docs;
- narrowing the lightweight required-document guard rail to canonical support
  surfaces instead of accumulated historical stage reports;
- documenting cost-control rules explicitly so future waves can add support
  assets without reintroducing broad edit fan-out.

## Success Criteria

This model is working when:

- adding one new support doc usually requires changing only
  `docs/operations/README.md` plus the new document itself;
- root entrypoints change only when contributor orientation materially changes;
- lightweight guard rails stay stable instead of growing linearly with stage
  history;
- contributors can tell which support surface is canonical without reading stage
  reports first.
