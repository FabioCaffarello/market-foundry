# Documentation

## Purpose

`docs/` is the canonical human documentation surface for `market-foundry`.

Its primary navigation is now organized by real human context instead of by
documentation mechanics.

## Primary Contexts

| Context | Entry point | Role |
|---|---|---|
| Product | [`product/README.md`](product/README.md) | What the system is, which docs own product/runtime questions |
| Development | [`development/README.md`](development/README.md) | How to work in the repository day to day |
| Tooling | [`tooling/README.md`](tooling/README.md) | `raccoon-cli` internals, rule catalogs, and guardrails |
| Architecture | [`architecture/README.md`](architecture/README.md) | Deep canonical technical reference |
| Stages | [`stages/INDEX.md`](stages/INDEX.md) | Historical delivery evidence |
| Archive | [`archive/README.md`](archive/README.md) | Superseded, transitional, or legacy material |

## Fast Paths

| Need | Go to |
|---|---|
| Understand the system first | [`product/README.md`](product/README.md) |
| Start coding or operating the repo | [`development/README.md`](development/README.md) |
| Find the owner doc for a contributor question | [`development/owners.md`](development/owners.md) |
| Find the owner doc for a product/runtime question | [`product/owners.md`](product/owners.md) |
| Inspect CLI rules or analyzer internals | [`tooling/README.md`](tooling/README.md) |
| Research historical rationale | [`stages/INDEX.md`](stages/INDEX.md), [`archive/README.md`](archive/README.md) |

## Rules

- keep active human docs in the primary contexts above;
- keep historical rationale in `docs/stages/` or `docs/archive/`;
- keep `docs/architecture/` as deep technical reference, not as the first-stop
  navigation surface for every human question;
- classify new docs before creating them; use
  [`architecture/information-system-governance-and-classification.md`](architecture/information-system-governance-and-classification.md)
  when in doubt;
- do not recreate the old high-fan-out catalog model in new indexes.
