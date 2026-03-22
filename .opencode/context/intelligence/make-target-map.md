# Make Target Map

Use this when you know the workflow need but not the underlying intelligence command.

- `make tdd` -> `raccoon-cli change tdd`
- `make coverage-map` -> `raccoon-cli inspect coverage`
- `make briefing TARGETS=...` -> `raccoon-cli change briefing`
- `make recommend TARGETS=...` -> `raccoon-cli change recommend`
- `make arch-guard` -> `raccoon-cli check arch`
- `make drift-detect` -> `raccoon-cli check drift`
- `make quality-gate` -> `raccoon-cli check gate`
- `make quality-gate-ci` -> `raccoon-cli check gate --profile ci --json`
- `make check-deep` -> `raccoon-cli check gate --profile deep` plus repo consistency

If the answer is a stable public step, keep it in `make`; if the answer is inspection depth, go to direct CLI.
