# Stage Execution

Use stage tooling only when work is tied to governed stage continuity.

Entrypoints:

- `make stage-help`
- `make stage-status STAGE_ID=... STAGE_SLUG=...`
- `make stage-check STAGE_ID=... STAGE_SLUG=...`
- `make stage-scaffold STAGE_ID=... STAGE_SLUG=... STAGE_TITLE=...`

What the helper is for:

- scaffold one report
- show continuity gaps
- validate report naming, indexing, links, and required artifacts

What it is not:

- not a task tracker
- not a session memory system
- not a replacement for `docs/stages/INDEX.md` or the stage report itself
