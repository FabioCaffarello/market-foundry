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
| `commands/` | Custom slash commands (currently empty; populates as needs emerge) |
| `agents/` | Sub-agent definitions for specialized tasks (currently empty) |
| `hooks/` | Workflow hooks for pre-commit, post-build, etc. (currently empty) |

## Philosophy

The previous agentic layer (`.opencode/`) accumulated many definitions
before they were needed, then drifted out of sync with the system as
the system evolved. Phase 1B removed it entirely.

This `.claude/` layer is deliberately **minimal**. Commands, agents,
and hooks are added only when there is a concrete repeated need that
benefits from automation. The empty subdirectories are intentional —
they signal "here's where this kind of thing goes if and when it's
useful".

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
