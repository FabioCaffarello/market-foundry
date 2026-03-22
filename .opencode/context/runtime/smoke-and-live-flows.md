# Smoke And Live Flows

Choose by intent:

- fastest bring-up: `make live`, `make live-multi`
- inspect each step: `make up` + `make seed*`
- choose proof: `make smoke-help`

Proof-of-record mapping:

- baseline slice -> `make smoke`
- multi-symbol -> `make smoke-multi`
- analytical writer/read path -> `make smoke-analytical`
- full persistence -> `make smoke-round-trip`
- live stack or kill-switch path -> `make smoke-live-stack`, `make smoke-activation`
- no full stack needed -> `make smoke-composed`
- lifecycle/recovery -> `make smoke-operational`, `make smoke-restart-recovery`

Script boundary:

- use `../../../scripts/live-pipeline-activate.sh` or `../../../scripts/smoke-*.sh` directly only for extra flags like `--wait` or harness debugging
- `make smoke*` stays the evidence surface; `make live*` is orchestration, not proof ownership
