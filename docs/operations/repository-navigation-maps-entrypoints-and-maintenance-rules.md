# Repository Navigation Maps, Entrypoints, And Maintenance Rules

## Purpose

This is the practical navigation map for contributors who are moving through the
repository tree itself.

Use it when you know the kind of problem you are solving, but not yet the exact
file or directory.

## Primary Navigation Paths

| You need to... | Start here | Why |
|---|---|---|
| understand the official developer workflow | `../../README.md` and `../../DEVELOPMENT.md` | these remain the daily operating entrypoints |
| find the right top-level repository area | this document | maps tasks to physical directories |
| find a runtime or service entrypoint | `../../cmd/README.md` | binary-oriented map |
| find implementation layers and code placement | `../../internal/README.md` | architecture-oriented code map |
| find runtime assets, configs, or migrations | `../../deploy/README.md` | deploy/runtime asset map |
| find harnesses and support scripts | `../../scripts/README.md` | explains the script layer behind `make` |
| find shared repository-level test assets | `../../tests/README.md` | shared test surface map |
| find canonical docs by taxonomy | `../README.md` | documentation-system entrypoint |

## Repository Area Map

| Area | What it owns | Entrypoint | Common reasons to open it |
|---|---|---|---|
| `cmd/` | binaries and process bootstrap | `../../cmd/README.md` | trace runtime startup, CLI utilities, service ownership |
| `internal/` | implementation layers and domain/application code | `../../internal/README.md` | locate behavior, decide placement, trace dependencies |
| `deploy/` | compose, configs, envs, migrations, runtime assets | `../../deploy/README.md` | inspect stack topology, config, local runtime wiring |
| `scripts/` | workflow harness implementations | `../../scripts/README.md` | debug wrappers, inspect smoke logic, maintain support scripts |
| `tests/` | shared repository test assets | `../../tests/README.md` | manual HTTP verification and cross-repo test assets |
| `tools/raccoon-cli/` | Rust tooling workspace | `../../tools/raccoon-cli/README.md` | evolve or inspect the architecture guardian |
| `docs/operations/` | workflow and documentation-system guidance | `README.md` | operating the repo and navigating active docs |
| `docs/tooling/` | tooling-internal references | `../tooling/README.md` | understand what the tooling enforces |
| `docs/architecture/` | binding architecture and governance | `../architecture/README.md` | confirm canonical structural rules |
| `docs/stages/` | immutable delivery evidence | `../stages/INDEX.md` | historical traceability |

## Entrypoints By Contributor Intent

### I need to change behavior

1. Start in `../../cmd/README.md` if you do not yet know which service owns the behavior.
2. Move to `../../internal/README.md` to find the correct implementation layer.
3. Use `../../deploy/README.md` only if the change depends on runtime config or local stack assets.

### I need to run or prove something

1. Start in `../../DEVELOPMENT.md`.
2. Use `../../scripts/README.md` only if you need to inspect the harness behind a `make` target.
3. Use `../README.md` and [`operational-proof-entrypoints-and-ownership.md`](operational-proof-entrypoints-and-ownership.md) when the proof surface is unclear.

### I need to understand repository history or rationale

1. Start in `../architecture/README.md` for current rules.
2. Then use `../stages/INDEX.md` for the delivery trail.
3. Use `../archive/README.md` only for superseded context.

### I need to add a new repository surface

1. Add it in the real directory tree first.
2. Update the nearest area entrypoint.
3. Update root docs only if the surface changes how contributors orient themselves.

## Maintenance Rules

### What must stay updated

- `README.md`
- `DEVELOPMENT.md`
- `docs/README.md`
- `docs/operations/README.md`
- the area entrypoints under `cmd/`, `internal/`, `deploy/`, `scripts/`, and `tests/`

### When to update an area entrypoint

- a new top-level responsibility appears in that area;
- the recommended first file to read changes materially;
- a developer would otherwise have to infer ownership from code archaeology.

### When not to update an area entrypoint

- minor file moves that do not affect orientation;
- package-internal refactors;
- changes that are already obvious from filenames and do not change ownership.

### Cross-linking rules

- Link outward to canonical docs instead of restating their content.
- Use relative links that follow the real tree.
- Do not create second copies of architecture rules in operations docs.
- Do not create second copies of workflow instructions in area `README.md` files.

## Practical Success Criteria

This navigation system is working when a contributor can answer these quickly:

- which directory owns the behavior or asset they need;
- which file to read first inside that directory;
- whether they are looking at current guidance, tooling internals, architecture policy, or historical evidence.
