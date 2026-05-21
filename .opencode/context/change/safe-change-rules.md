# Safe Change Rules

Before editing, re-check these repo-specific risks:

- do not break the layer flow `domain -> application -> adapters -> actors -> interfaces -> cmd`
- do not reintroduce Kafka, quality-service names, or `.context/`
- do not turn scripts or `raccoon-cli` into parallel public workflows
- keep `make smoke*` as runtime proof-of-record
- if topology, streams, subjects, configs, or compose move, update docs and guard rails in the same change
- if `Makefile`, `scripts/`, `.opencode/`, or `tools/raccoon-cli/` move together, keep the support surfaces aligned in one change

Use these owners when unsure:

- `../../../docs/architecture/stage-definition-of-done.md`
- `../../../docs/architecture/anti-debt-checklist.md`
- `../../../docs/architecture/opus-guidance-rules.md`
- `../../../docs/architecture/prohibited-carryovers.md`
