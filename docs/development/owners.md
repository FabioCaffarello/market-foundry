# Development Owners

## Purpose

This file defines the canonical owner docs for recurring development questions.

## Owner Map

| Subject | Owner doc | Reference docs | Historical trail |
|---|---|---|---|
| Daily engineering workflow | [`workflow.md`](workflow.md) | [`../../DEVELOPMENT.md`](../../DEVELOPMENT.md) | [`../archive/operations/README.md`](../archive/operations/README.md) |
| Repository map and entrypoints by task | [`repository-map.md`](repository-map.md) | [`../../cmd/README.md`](../../cmd/README.md), [`../../deploy/README.md`](../../deploy/README.md), [`../../scripts/README.md`](../../scripts/README.md), [`../../tests/README.md`](../../tests/README.md) | [`../archive/operations/README.md`](../archive/operations/README.md) |
| Command surface, proofs, and troubleshooting | [`commands-and-proofs.md`](commands-and-proofs.md) | [`../../Makefile`](../../Makefile), [`../tooling/README.md`](../tooling/README.md) | [`../archive/operations/README.md`](../archive/operations/README.md) |
| Stage support and documentation hygiene | [`stages-and-governance.md`](stages-and-governance.md) | [`../stages/INDEX.md`](../stages/INDEX.md), [`../architecture/stage-definition-of-done.md`](../architecture/stage-definition-of-done.md) | [`../archive/operations/README.md`](../archive/operations/README.md) |
| Tooling-internal reference | [`../tooling/README.md`](../tooling/README.md) | tool-specific `cli-*.md` docs under `docs/tooling/` | [`../stages/INDEX.md`](../stages/INDEX.md) |

## Rules

- one recurring development question should have one owner doc;
- root docs stay shallow and route into these owners;
- legacy workflow/governance essays belong in archive, not in the primary
  contributor surface.
