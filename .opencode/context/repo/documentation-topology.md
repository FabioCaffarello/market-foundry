# Documentation Topology

Use this file when deciding which doc should own new or updated explanation.

Owner surfaces:

- `../../../README.md` for repository identity and quick orientation
- `../../../DEVELOPMENT.md` for the day-to-day change loop
- `../../../docs/development/owners.md` for contributor owner mapping
- `../../../docs/product/owners.md` for system/product owner mapping
- `../../../docs/tooling/README.md` for `raccoon-cli` internals and references
- `../../../docs/architecture/README.md` for architecture and governance
- `../../../docs/architecture/information-system-governance-and-classification.md`
  for classification and evolution policy
- `../../../docs/stages/INDEX.md` for historical stage evidence
- `../../../docs/archive/README.md` for superseded material

Routing rules:

- update owner docs when behavior, workflow, contract, or governance changes
- update `.opencode` only when routing, compression, or handoff changed
- stage reports are evidence, not recurring owner answers
- archive superseded guidance instead of leaving compatibility sprawl in active surfaces

What `.opencode` intentionally absorbs from `docs/`:

- owner-doc routing
- workflow and proof entrypoint compression
- short stage/governance entrypoints
- tooling boundary reminders

What stays in `docs/`:

- human explanation
- durable rationale
- rule catalogs
- historical evidence
