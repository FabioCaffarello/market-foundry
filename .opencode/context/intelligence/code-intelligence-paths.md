# Code Intelligence Paths

Use these paths when the task is “inspect before changing”:

- symbol or contract tracing -> `raccoon-cli inspect symbol`, `raccoon-cli inspect contract-usage`
- coverage and blind spots -> `make coverage-map`
- change blast radius -> `raccoon-cli change impact`, `make briefing`
- baseline drift over time -> `make snapshot`, `make snapshot-diff`, `make baseline-drift`

When they help most in this repo:

- before renaming stream, subject, config, or service identity
- before changing `deploy/compose/`, `deploy/configs/`, or `tools/raccoon-cli/`
- when docs, code, and runtime seem inconsistent but tests still pass

These paths inform the change loop; they do not replace `make verify` or `make smoke*`.
