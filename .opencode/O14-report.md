# O14 Report

## Objective

Canonize the minimum `.opencode` core for `market-foundry` around repository
navigation, semantic compression, and context handoff, without creating a new
owner surface.

## Ownership Map

| Surface | Owner |
|---|---|
| workflow contract and command surface | `AGENTS.md`, `Makefile`, `DEVELOPMENT.md` |
| repository identity and physical navigation | `README.md`, `cmd/README.md`, `internal/README.md`, `deploy/README.md`, `scripts/README.md`, `tests/README.md` |
| operations and support navigation | `docs/operations/` |
| tooling rules and `raccoon-cli` | `docs/tooling/`, `tools/raccoon-cli/README.md` |
| architecture, boundaries, governance | `docs/architecture/` |
| historical evidence | `docs/stages/INDEX.md`, stage reports |
| compression and handoff only | `.opencode/` |

## Editorial Rationale

- keep `.opencode` short, task-shaped, and link-heavy
- compress only the recurring decisions: where work belongs, which workflow to
  follow, which surface owns a rule, and where docs should live
- push policy, catalogs, and rationale back to owner docs immediately

## Naming Decisions

- `.opencode/context/repo/repository-shape.md` replaces the old
  `overview` surface because the file answers placement, not product overview
- `.opencode/context/repo/documentation-topology.md` replaces the old `docs`
  surface because the problem is doc ownership
  routing, not a generic doc list
- `.opencode/context/repo/architecture-boundaries.md`,
  `.opencode/context/repo/development-workflow.md`, and
  `.opencode/context/repo/tooling-contracts.md` remain because they describe
  concrete task questions

## Maintenance Rules

- change owner docs first when behavior or policy changes
- change `.opencode` only when routing, compression, or handoff changed
- keep context files short enough to route, not teach
- every new `.opencode` file must point to real owner docs and be linked from a
  navigation or profile entrypoint
- do not add `.context/`, generic plugin surfaces, or parallel command catalogs
