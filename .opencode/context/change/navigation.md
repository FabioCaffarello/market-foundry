# Change Navigation

Canonical owner docs:

- `../../../AGENTS.md`
- `../../../DEVELOPMENT.md`
- `../../../docs/operations/development-lifecycle-entrypoints-and-canonical-flows.md`
- `../../../docs/architecture/stage-definition-of-done.md`
- `../../../docs/architecture/anti-debt-checklist.md`
- `../../../docs/architecture/opus-guidance-rules.md`

Start here by question:

- what breaks if I touch this area -> `impact-analysis.md`
- what do I run before and after editing -> `tdd-and-validation.md`
- how do stage helpers fit without becoming workflow -> `stage-execution.md`
- what changes are unsafe even if tests pass -> `safe-change-rules.md`

Default loop:

- `make check`
- `make tdd`
- implement the smallest correct change
- `make verify`
- add the narrowest relevant `make smoke*`

Use `docs/stages/INDEX.md` for immutable evidence, not `.opencode`.
