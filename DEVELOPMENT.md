# DEVELOPMENT.md

This file is preserved as a convention. The canonical development
guide is **[docs/DEVELOPMENT.md](docs/DEVELOPMENT.md)**.

docs/DEVELOPMENT.md covers:

- Setup (prerequisites, bootstrap, local.env)
- Daily workflow (check → tdd → implement → verify)
- Smoke tests (target selection, when to run which)
- Testing (unit, integration, ClickHouse, behavioral, boot test, codegen)
- Stack lifecycle (bring-up, inspection, tear-down, volume wipe)
- Troubleshooting (5 common scenarios with diagnosis)
- Architecture enforcement (raccoon-cli surfaces)
- Module scoping
- Contributing changes

For PR rules and protocols, see [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md).
For the canonical entry point for AI agents, see [CLAUDE.md](CLAUDE.md).
For current state and known gaps, see [docs/RESUMPTION.md](docs/RESUMPTION.md).

---

## Why this file is now a shim

The pre-reset DEVELOPMENT.md duplicated content that is now better
maintained in docs/DEVELOPMENT.md. Rather than keep both in sync, this
file redirects to the single source of truth.
