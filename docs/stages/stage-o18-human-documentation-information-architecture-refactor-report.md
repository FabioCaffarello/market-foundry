# Stage O18 Report: Human Documentation Information Architecture Refactor

## Summary

O18 executes the human-documentation refactor requested after O17.

The main editorial move is structural:

- restore `docs/` as a human-facing navigation surface;
- segment active docs by real context: `product` and `development`;
- reduce `docs/operations/` from a competing active catalog to a legacy bridge;
- preserve history by reclassifying legacy support and documentation-topology
  material into `docs/archive/`;
- keep `tooling`, `stages`, and `archive` with explicit non-competing roles.

## Editorial Rationale

The previous topology made humans navigate documentation mechanics before
reaching the subject they actually cared about. `docs/operations/` became a
high-fan-out support catalog, while `docs/architecture/` still carried some
meta-documentation and reconciliation artifacts that competed with current
owner docs.

O18 shifts the first question from "which documentation system file should I
open?" to:

1. is this a product/runtime question?
2. is this a contributor/development question?
3. is this tooling, history, or archive?

## Before / After Map

### Before

| Surface | Role in practice |
|---|---|
| `docs/README.md` | routing plus taxonomy discussion |
| `docs/operations/` | active support catalog and documentation governance hub |
| `docs/tooling/` | tooling internals |
| `docs/architecture/` | technical corpus plus some documentation-meta artifacts |
| `docs/stages/` | history |
| `docs/archive/` | historical archive |

### After

| Surface | Role |
|---|---|
| `docs/README.md` | small routing surface only |
| `docs/product/` | primary human context for product/runtime understanding |
| `docs/development/` | primary human context for contributor workflow and repository use |
| `docs/tooling/` | tooling-internal reference |
| `docs/architecture/` | deep canonical technical reference |
| `docs/stages/` | immutable history |
| `docs/archive/` | archived legacy and superseded material |
| `docs/operations/README.md` | legacy bridge, not a primary owner surface |

## Owners By Context

### Product

| Subject | Owner |
|---|---|
| Product/runtime overview | `docs/product/system-overview.md` |
| Product owner map | `docs/product/owners.md` |
| Deep technical reference | `docs/architecture/README.md` |

### Development

| Subject | Owner |
|---|---|
| Development owner map | `docs/development/owners.md` |
| Daily workflow | `docs/development/workflow.md` |
| Repository map | `docs/development/repository-map.md` |
| Commands and proofs | `docs/development/commands-and-proofs.md` |
| Stages and documentation hygiene | `docs/development/stages-and-governance.md` |

## Consolidations Applied

The following active concerns were compressed into smaller owner surfaces:

- daily workflow, onboarding, and first-line troubleshooting into
  `docs/development/workflow.md`;
- repository-shape and task-based navigation into
  `docs/development/repository-map.md`;
- command-surface, proof-of-record, and `make` vs tooling guidance into
  `docs/development/commands-and-proofs.md`;
- stage support and documentation hygiene into
  `docs/development/stages-and-governance.md`;
- product-facing runtime orientation into `docs/product/system-overview.md`
  plus `docs/product/owners.md`.

## Reclassified / Archived Material

### Reclassified from active navigation

- the pre-O18 `docs/operations/` corpus moved to `docs/archive/operations/`;
- documentation-topology and reconciliation artifacts moved to
  `docs/archive/documentation/`;
- `docs/operations/README.md` now survives only as a bridge into the new
  surfaces and the archive.

### Preserved but demoted from primary navigation

- `docs/architecture/` remains active, but `docs/architecture/README.md` was
  compressed so it no longer competes with product/development entrypoints;
- `docs/stages/INDEX.md` remains the historical entrypoint only.

## Tradeoffs

### Gains

- lower navigation fan-out for humans;
- explicit owner docs by context;
- less competition between workflow docs, meta-docs, and history;
- stronger separation between current guidance and historical rationale.

### Costs

- old `docs/operations/` deep links now route through archive/bridge surfaces;
- some legacy support essays are no longer first-stop reading, even though they
  remain available.

## Impact

- root docs, `make docs`, bootstrap, and repository consistency now point to
  the O18 topology;
- active contributor navigation is shorter and context-led;
- useful history remains preserved without staying in the main navigation path.
