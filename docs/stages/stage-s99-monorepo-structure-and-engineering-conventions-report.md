# Stage S99 — Monorepo Structure and Engineering Conventions

## Objective

Consolidate the structural conventions of the monorepo and its engineering practices, reducing organizational drift, improving readability, and reinforcing healthy growth patterns — without adding features or inflating bureaucracy.

## Changes Made

### Structural Drift Removed

1. **Stale binary artifact**: Removed `bin/server` — a residual build output from the pre-S96 server→gateway rename. The binary had no corresponding `cmd/server/` source and would confuse contributors.

2. **AGENTS.md service table**: Updated from 3 entries (configctl, gateway, nats) to 7 entries reflecting all current services (configctl, gateway, ingest, derive, store, execute, nats). AI agents operating on this repo were working with an incomplete picture of the system.

3. **DEVELOPMENT.md project structure**: Updated the project structure section to include:
   - Missing services: `cmd/execute/`, `cmd/store/`.
   - Missing domains in the domain layer description: `signal`, `decision`, `strategy`, `risk`, `execution`.
   - Missing `docs/` subdirectory breakdown (`architecture/`, `stages/`, `tooling/`).
   - Updated descriptions for accuracy (e.g., `tools/raccoon-cli/` described as "architecture guardian CLI" rather than "quality CLI").

4. **DEVELOPMENT.md services table**: Added the missing `execute` service entry.

### Convention Documents Created

1. **`docs/architecture/monorepo-structure-and-engineering-conventions.md`**
   - Formalizes the monorepo layout, Go workspace conventions, module boundaries, dependency direction, service layout, package conventions per layer, configuration conventions, build/workflow conventions, tooling conventions, and naming invariants.
   - Consolidates structural knowledge that was previously implicit or scattered across multiple docs.

2. **`docs/architecture/how-to-introduce-new-runtimes-domains-and-families.md`**
   - Step-by-step guide for adding new domains, families, runtimes, and adapters.
   - Includes concrete file paths, naming rules, validation steps, and anti-patterns.
   - Complements `family-runtime-registration-rules.md` (which covers family registration details per runtime) with broader guidance on all expansion types.

3. **`docs/architecture/monorepo-documentation-and-stage-governance.md`**
   - Formalizes documentation structure, document categories, writing rules, and maintenance guidelines.
   - Defines stage governance: what stages are, how reports are structured, numbering conventions, and the relationship between stage reports (historical) and architecture docs (canonical).
   - Clarifies when to create vs. update docs, and how to avoid documentation drift.

## Conventions Established

### Monorepo Structure
- Canonical layout map with every top-level and internal directory defined.
- Module boundaries formalized: 14 modules, one per architectural boundary.
- Explicit dependency direction rule with diagram.
- Service binary layout standardized: `main.go` + `run.go` + optional `compose.go`/`{service}.go`.

### Documentation Governance
- Three-tier doc structure: `architecture/` (canonical), `stages/` (historical), `tooling/` (mirror).
- Architecture doc categories defined: foundation, conventions, patterns, how-to, governance, audit, domain.
- Stage report format standardized with required sections.
- Immutability rule for stage reports formalized.
- "Stage records the decision, architecture doc captures the convention" relationship made explicit.

### Expansion Playbook
- Step-by-step checklists for each expansion type (domain, family, runtime, adapter).
- Validation requirements per expansion type.
- Anti-patterns documented per expansion type.

## Drift Removed

| Item | Type | Impact |
|------|------|--------|
| `bin/server` | Stale artifact | Eliminated confusion from pre-rename residue |
| AGENTS.md service table | Incomplete metadata | AI agents now see all 7 services |
| DEVELOPMENT.md structure | Incomplete reference | Developers now see accurate project map |
| DEVELOPMENT.md services | Missing entry | Execute service now documented |

## Trade-offs and Limitations

1. **No docs/ reorganization**: The `docs/architecture/` directory contains 166+ files in a flat structure. Introducing subdirectories (e.g., `docs/architecture/patterns/`, `docs/architecture/domains/`) was considered but deferred — the disruption of moving 166 files and updating all cross-references outweighs the navigation benefit, especially since grep/search is the primary access pattern.

2. **No automated doc-code sync**: Documentation correctness still depends on manual discipline. The quality gate enforces structural and naming conventions in code, but does not validate that architecture docs match current implementation.

3. **Convention density**: The monorepo now has substantial convention documentation. The trade-off is between comprehensive guidance (reduces ambiguity) and reading burden (new contributors must absorb more). The how-to guide and cross-references mitigate this by providing clear entry points.

4. **zip/ directory**: The `zip/` directory at the root contains archive snapshots. It is gitignored and serves as local development convenience. No action taken — it has no structural impact.

## Files Changed

| File | Action |
|------|--------|
| `bin/server` | Deleted (stale artifact) |
| `AGENTS.md` | Updated (service table expanded) |
| `DEVELOPMENT.md` | Updated (project structure, services table) |
| `docs/architecture/monorepo-structure-and-engineering-conventions.md` | Created |
| `docs/architecture/how-to-introduce-new-runtimes-domains-and-families.md` | Created |
| `docs/architecture/monorepo-documentation-and-stage-governance.md` | Created |
| `docs/stages/stage-s99-monorepo-structure-and-engineering-conventions-report.md` | Created |

## Preparation for S100

S99 completes the foundation consolidation arc (S96–S99). The monorepo now has:

- Formalized structural conventions and growth playbook.
- Clean documentation governance with clear stage/architecture separation.
- Accurate metadata in entry-point files (AGENTS.md, DEVELOPMENT.md).
- No residual artifacts from historical renames.

**Recommended focus for S100:**

1. **Operational maturity**: CI/CD pipeline formalization (currently no `.github/` or CI config is visible in the repo).
2. **Testing strategy formalization**: Document the testing pyramid — unit tests, integration tests (build-tagged), smoke tests, and their boundaries.
3. **Marketmonkey absorption preparation**: The repo is structurally ready; the next phase can focus on absorbing marketmonkey functionality into the established patterns.
4. **Quality gate evolution**: Consider adding doc-code consistency checks to raccoon-cli (e.g., verify that all services in `BUILDABLE_SERVICES` appear in AGENTS.md).
