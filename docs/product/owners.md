# Product Owners

## Purpose

This file makes the product-facing owner docs explicit.

Use it when the question is "which current document owns this product or
architecture subject?"

## Owner Map

| Subject | Owner doc | Reference docs | Historical trail |
|---|---|---|---|
| Product/runtime overview | [`system-overview.md`](system-overview.md) | [`../../README.md`](../../README.md) | [`../stages/INDEX.md`](../stages/INDEX.md) |
| System identity and design direction | [`../architecture/system-vision.md`](../architecture/system-vision.md) | [`../architecture/system-principles.md`](../architecture/system-principles.md), [`../architecture/non-goals.md`](../architecture/non-goals.md) | [`../archive/README.md`](../archive/README.md) |
| Runtime topology and binary responsibilities | [`../architecture/runtime-target.md`](../architecture/runtime-target.md) | [`../architecture/actor-ownership.md`](../architecture/actor-ownership.md), [`../architecture/stream-ownership-matrix.md`](../architecture/stream-ownership-matrix.md) | [`../stages/INDEX.md`](../stages/INDEX.md) |
| Evolution governance | [`../architecture/market-foundry-evolution-playbook.md`](../architecture/market-foundry-evolution-playbook.md) | [`../architecture/stage-definition-of-done.md`](../architecture/stage-definition-of-done.md), [`../architecture/anti-debt-checklist.md`](../architecture/anti-debt-checklist.md) | [`../stages/INDEX.md`](../stages/INDEX.md) |
| Architecture corpus entrypoint | [`../architecture/README.md`](../architecture/README.md) | domain and runtime docs under `docs/architecture/` | [`../archive/README.md`](../archive/README.md) |

## Rules

- keep one owner per recurring product question;
- route to `docs/architecture/` for deep technical detail instead of cloning it
  into product docs;
- move superseded product narratives to `docs/archive/`, not to the main
  navigation surface.
