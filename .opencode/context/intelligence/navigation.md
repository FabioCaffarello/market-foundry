# Intelligence Navigation

Canonical owner docs:

- `../../../docs/development/commands-and-proofs.md`
- `../../../docs/tooling/README.md`
- `../../../docs/tooling/cli-overview.md`
- `../../../tools/raccoon-cli/README.md`

Use this surface for:

- `raccoon-cli` boundaries and grouped usage
- fast mapping from `make` wrappers to direct expert commands
- repository guard rails that shape safe tooling changes
- code-intelligence paths like snapshots, drift, and coverage

Start here by question:

- grouped CLI usage and when to drop below `make` -> `raccoon-cli-usage.md`
- which `make` target wraps which intelligence path -> `make-target-map.md`
- what must stay aligned across docs, wrappers, and checks -> `repo-guardrails.md`
- snapshots, coverage, diff, and drift -> `code-intelligence-paths.md`

Do not treat `raccoon-cli` as runtime orchestration or proof-of-record.
