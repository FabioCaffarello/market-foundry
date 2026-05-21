# Raccoon CLI Usage

Use `make` first for the public workflow. Drop to direct CLI when you need JSON, narrower scope, or you are changing the tool itself.

Grouped command map:

- `check` -> repo, topology, contracts, bindings, arch, drift, gate
- `inspect` -> symbol, lsp, contract-usage, coverage
- `change` -> impact, tdd, briefing, recommend, rename
- utilities -> `snapshot`, `snapshot-diff`, `baseline-drift`
- compatibility only -> `legacy runtime-smoke` and hidden flat aliases

Do not use it for:

- stack bring-up
- live orchestration
- replacing `make smoke*`

Primary references:

- `../../../docs/tooling/cli-overview.md`
- `../../../tools/raccoon-cli/README.md`
