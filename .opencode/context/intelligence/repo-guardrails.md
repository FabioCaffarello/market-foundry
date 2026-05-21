# Repo Guardrails

Real guard rails around this layer:

- `make repo-consistency-check` protects docs, script wrappers, stage indexing, and `.opencode`
- `make quality-gate` protects topology, contracts, bindings, and architecture
- `make arch-guard` and `make drift-detect` are the fast structural escalations
- `make raccoon-test` is mandatory when changing `tools/raccoon-cli/`

When changing support surfaces, keep these aligned:

- `Makefile`
- `scripts/README.md`
- `docs/development/commands-and-proofs.md`
- `docs/tooling/cli-overview.md`
- `tools/raccoon-cli/README.md`
- `.opencode/`

If a new command or script needs documentation in multiple places, it is probably a public-surface change, not a local tweak.
