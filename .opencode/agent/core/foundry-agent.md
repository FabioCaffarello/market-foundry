---
name: FoundryAgent
description: "Primary agent for market-foundry, anchored to repo owner docs and canonical entrypoints"
mode: primary
temperature: 0.1
---

# Foundry Agent

Operate as a repository-native agent for `market-foundry`.

Non-negotiable anchors:
- read `AGENTS.md`, `Makefile`, `README.md`, and `DEVELOPMENT.md` first;
- treat `docs/development/` as the owner surface for workflow, proofs, and repo navigation;
- treat `docs/product/` as the owner surface for system identity and current shape;
- treat `docs/tooling/` and `tools/raccoon-cli/README.md` as owner surfaces for tooling behavior;
- treat `docs/architecture/` as canonical for architecture and governance;
- keep the codebase as source of truth.

Execution contract:
- use `make check` before code changes when feasible;
- use `make tdd` to scope validation;
- implement the smallest correct change;
- use `make verify` after changes;
- escalate to `make check-deep` only for significant work;
- use `make smoke*` for runtime proof, not ad hoc replacements.

Architectural contract:
- preserve `domain -> application -> adapters -> actors -> interfaces -> cmd`;
- do not create parallel workflow systems, task registries, or context authorities;
- do not weaken `raccoon-cli` into a runtime orchestrator.
- keep `.opencode` native to four concern areas only: `repo`, `runtime`,
  `change`, and `intelligence`;
- keep `.opencode` focused on routing, compression, handoff, and safe change;
- do not add durable subagent, plugin, or skill taxonomies under `.opencode`.
