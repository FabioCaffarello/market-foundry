# Stages And Governance

## Purpose

This file owns the development-facing stage and documentation hygiene model.

## Stage Workflow

Use these entrypoints when you are working inside a governed stage:

```bash
make stage-help
make stage-status STAGE_ID=<id> STAGE_SLUG=<slug>
make stage-check STAGE_ID=<id> STAGE_SLUG=<slug>
```

## Source Of Truth

- Stage history index: [`../stages/INDEX.md`](../stages/INDEX.md)
- Definition of done: [`../architecture/stage-definition-of-done.md`](../architecture/stage-definition-of-done.md)
- Evolution playbook: [`../architecture/market-foundry-evolution-playbook.md`](../architecture/market-foundry-evolution-playbook.md)
- Information-system policy: [`../architecture/information-system-governance-and-classification.md`](../architecture/information-system-governance-and-classification.md)

## Editorial Rules

- active human docs belong in `docs/product/`, `docs/development/`,
  `docs/tooling/`, or `docs/architecture/`;
- stage reports are historical evidence, not the current owner of a recurring
  rule;
- superseded support docs move to `docs/archive/`, not to the primary
  navigation surface.
- `.opencode/` may compress this workflow, but it does not own the human rule.
