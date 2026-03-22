# O15 Report

## Summary

O15 expands `.opencode` only around three high-value concerns: local runtime operation, safe repository change, and strategic tooling intelligence. The layer stays short, task-shaped, and anchored to the real `make`, scripts, `deploy/`, docs, and `raccoon-cli` surfaces.

## Rationale

- `.opencode/context/runtime/` now answers how to bring up the stack, what services exist, where compose/config/migration assets live, which smoke or live flow proves a path, and how to troubleshoot first.
- `.opencode/context/change/` now answers how to scope impact, run the official TDD and validation loop, use stage helpers correctly, and avoid unsafe repository drift.
- `.opencode/context/intelligence/` now answers when `make` is enough, when to use direct `raccoon-cli`, how wrappers map to expert commands, and which guard rails keep support surfaces aligned.

## Limits

- `.opencode` still does not duplicate large command catalogs, stage history, topology inventories, or architecture rationale.
- canonical ownership remains in `AGENTS.md`, `Makefile`, `README.md`, `DEVELOPMENT.md`, `docs/operations/`, `docs/tooling/`, and `docs/architecture/`.
- the layer is intentionally navigational; if a file starts carrying policy or full runbook depth, it should collapse back into pointers.

## Drift Risks

- `Makefile`, `scripts/*.sh`, and `.opencode` can drift when a new target or proof path is added without updating navigation.
- `raccoon-cli` taxonomy can drift from wrapper docs if grouped commands or profiles change.
- compose/config/runtime evolution can stale the short summaries if service ownership or runtime prerequisites move.

## Usage Guidance

- start with `.opencode/context/navigation.md`, then descend only into the needed block
- use `.opencode/context/runtime/` for stack/proof/troubleshooting questions
- use `.opencode/context/change/` for impact, validation, and governed-stage questions
- use `.opencode/context/intelligence/` for `raccoon-cli`, wrapper mapping, and structural inspection
- treat `.opencode` as a fast router; confirm details in the linked canonical owners before deeper edits
