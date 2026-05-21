# Pre-Slice Repository Alignment

## Purpose

Confirm the repository is aligned and ready for the next vertical slice execution after S107 cleanup.

## Alignment Checklist

### Infrastructure

- [x] All compose services use `market-foundry-*` naming consistently
- [x] All compose services use the same network (`market-foundry-network`)
- [x] All service healthchecks use HTTP readiness probes
- [x] ClickHouse env template exists for fresh clones
- [x] Docker images use `market-foundry/<service>:dev` prefix

### Naming Consistency

- [x] No `server.http` references remain in test or production code
- [x] No `quality-service` references in active (non-deprecated) code paths
- [x] AGENTS.md status reflects post first-slice phase
- [x] LSP workspace identity matches project name

### Tooling Health

- [x] raccoon-cli compiles cleanly (warnings are pre-existing, non-blocking)
- [x] Deprecated dead-code modules removed (`results_inspect`, `trace_pack`)
- [x] Deprecated CLI commands removed (`scenario-smoke`, `results-inspect`, `trace-pack`)
- [x] Quality-gate deep profile still functional (`runtime-smoke` preserved)
- [x] All Go tests pass after naming alignment

### Code Quality

- [x] No new features introduced
- [x] No architectural reorganization
- [x] No cosmetic-only changes
- [x] All changes have clear structural rationale

## Residual Items (Acceptable for Next Slice)

These items exist but do not block vertical slice execution:

1. **smoke module internal comments**: Contains `quality-service` references in error messages. Acceptable — these are runtime error strings in a deprecated-but-functional code path used only by `quality-gate --profile deep`.

2. **Topology compose test fixture**: Contains a `quality-service/consumer:dev` image reference as parse-test data. Not operational.

3. **Compiler warnings in raccoon-cli**: Pre-existing warnings about unused code in `codeintel`, `coverage_map`, `lsp`, `smoke`. Non-blocking; can be addressed in a dedicated tooling wave.

## Recommended Next Steps

1. Design and scope the next vertical slice (S108+)
2. Decide whether to evolve or replace the smoke module for E2E testing
3. Consider a focused raccoon-cli cleanup wave to suppress remaining warnings
