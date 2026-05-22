# .claude/

Anthropic agentic-layer configuration and customization for
market-foundry. This directory replaces the previous `.opencode/`
layer (removed in Phase 1B).

For the primary instructions that Claude reads automatically, see
[CLAUDE.md](../CLAUDE.md) in the repository root.

## Structure

| Path | Purpose |
|---|---|
| `settings.json` | Default settings for Claude sessions in this repo |
| `commands/` | Custom slash commands (5 commands; codifying Phase 1+2 patterns — see below) |
| `agents/` | Sub-agent definitions for specialized tasks (2 templates) |
| `hooks/` | Workflow hooks for pre-commit, post-build, etc. (currently empty) |

## Available commands

Slash commands invocable as `/<name>` in Claude Code sessions:

| Command | Purpose |
|---|---|
| `/check-clean` | Verify working tree clean + baseline (`make verify`, `make bootstrap`) PASS at session start |
| `/check-refs <path>` | Find all active references to a path before deletion or rename |
| `/inventory <area>` | Produce structured inventory of an area before making changes |
| `/audit <area>` | Skeleton for read-only investigation of an area |
| `/version-check` | Verify version consistency across `go.work`, `rust-toolchain.toml`, `.tool-versions`, CI |

These codify patterns that recurred across Phase 1+2 work (working-
tree verification, cross-ref search, inventory production, audit
skeletons, version sync checks).

## Available agent templates

Agent role definitions in `agents/`:

| Agent | Purpose |
|---|---|
| `investigation-agent` | Read-only investigator; produces structured reports without modifying repo |
| `execution-agent` | Scoped executor following pause-and-report protocol |

These are templates documenting how Phase 1+2 distinguished
read-only investigation from scoped execution. They are descriptive,
not enforced — but useful as orientation when spawning sub-agents.

## Philosophy

The previous agentic layer (`.opencode/`) accumulated many definitions
before they were needed, then drifted out of sync with the system as
the system evolved. Phase 1B removed it entirely.

This `.claude/` layer is deliberately **minimal**. Commands, agents,
and hooks are added only when there is a concrete repeated need that
benefits from automation. The current population came from P3.8,
which codified patterns proven repetitive across Phase 1+2 work.
The empty `hooks/` subdirectory is intentional — Claude Code hooks
remain exploratory; populated only when concrete needs surface.

When adding a command, agent, or hook:

1. Document its purpose in this README's relevant table.
2. Keep it focused on one task.
3. If it encodes a structural decision, also write an ADR
   (see `../docs/decisions/`).

## What's NOT here

- **Documentation** — lives in `../docs/`.
- **Code** — lives in `../internal/`, `../cmd/`, `../tools/`.
- **Configuration of runtime services** — lives in `../deploy/`.

This directory is exclusively about agent-side configuration and
customization.
