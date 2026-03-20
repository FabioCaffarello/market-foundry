# Analytical Closure: Open vs Closed Items

> S206 — Explicit inventory of what was closed and what was consciously frozen.

## Closed Items

Items below are resolved and require no further action before S207.

| # | Area | Item | Action |
|---|------|------|--------|
| C-01 | Reader | 4 missing compile-time interface assertions | Added all 6 assertions |
| C-02 | Tooling | Compiled binary `cmd/writer/writer` staged in git | Unstaged, .gitignore updated |
| C-03 | Scripts | Writer missing from live-pipeline health/readiness/diagnostics | Added to all 4 phases |
| C-04 | Scripts | Analytical endpoints not validated in live-pipeline-activate | Added /analytical/* checks |
| C-05 | Scripts | Gateway port missing from SVC_PORTS map | Added `[gateway]=8080` |
| C-06 | Writer | No TODOs/FIXMEs in writer service | Confirmed clean |
| C-07 | Reader | All 6 adapters, use cases, handlers, routes complete | Confirmed 205+ tests |
| C-08 | Schema | All 7 migrations present and consistent (000-006) | Confirmed |
| C-09 | Config | Settings validation for writer (ClickHouse + Pipeline) | Confirmed complete |
| C-10 | NATS | All 7 families consistently registered across registries | Confirmed |
| C-11 | Gateway | Analytical wiring with optional ClickHouse lifecycle | Confirmed correct |
| C-12 | CI | Codegen golden equivalence + analytical E2E smoke in CI | Confirmed |
| C-13 | Diagnostics | diag-check.sh includes writer in readiness/statusz/diagz | Confirmed present |

## Frozen Items (Consciously Deferred)

Items below are known gaps that were explicitly reviewed and deferred.
They must NOT be addressed before S207 to prevent scope contamination.

| # | Area | Item | Why Frozen | Risk | Next Phase |
|---|------|------|-----------|------|-----------|
| F-01 | Dependencies | clickhouse-go v2.30.0 (adapters) vs v2.43.0 (migrate) | Version alignment requires coordinated update + regression testing. Both versions work for their respective use cases. | Low — separate binaries, no shared state | Post-S207 dependency sweep |
| F-02 | Writer | No backpressure between consumer and inserter actors | Requires Hollywood actor framework changes or reply-pattern implementation. Current fire-and-forget is acceptable for analytical projection (not transactional). | Medium — potential silent event loss under sustained load | Writer hardening phase |
| F-03 | Writer | parseFloat/marshalJSON default to 0/"{}\" on error | Silent data substitution. Acceptable for analytical projection where completeness > precision. No quarantine/DLQ mechanism. | Low — analytical data, not operational | Writer hardening phase |
| F-04 | Writer | Hardcoded 30s ClickHouse insert timeout | Not configurable per-family. Current value is adequate for batch sizes up to 1000 rows. | Low | Writer tuning phase |
| F-05 | Writer | No per-family batch tuning | All families share same batch_size/flush_interval. Adequate for current 7-family scope. | Low | Scaling phase |
| F-06 | Migrations | No transaction wrapping for migration + metadata record | ClickHouse has limited DDL transaction support. Bootstrap DDL uses CREATE IF NOT EXISTS for idempotency. | Medium — partial migration state possible on crash | Migration hardening |
| F-07 | Migrations | 60s context timeout hardcoded | Adequate for current 7 simple DDL migrations. Would need increase for complex data migrations. | Low | Migration hardening |
| F-08 | Smoke | smoke-analytical-e2e.sh only tests 60s timeframe | Phase 5 validates one timeframe per family. Multi-timeframe validated in operational smoke. | Low — analytical queries are timeframe-agnostic | Smoke expansion |
| F-09 | Smoke | Hardcoded credentials across config/scripts | ClickHouse password "clickhouse" appears in gateway.jsonc, writer.jsonc, smoke scripts. Acceptable for local/CI development. | Low — local only | Production readiness phase |
| F-10 | Tests | No integration tests for migration runner | Unit tests cover catalog/checksum. Runner requires real ClickHouse instance. CI analytical E2E implicitly validates migrations. | Low — CI provides indirect coverage | Test expansion phase |
| F-11 | Tests | No actor integration tests for writer | Unit tests cover mappers, inserter buffer, supervisor backoff. Full actor orchestration not unit-tested. CI E2E validates end-to-end. | Medium — failure recovery untested | Writer hardening phase |
| F-12 | CI | Codegen integrated check doesn't block merge | Runs in CI but exit code may not block PR merge if not marked required. | Low — golden snapshot check is the primary gate | CI hardening |

## Decision Criteria Applied

Items were classified as **closed** when:
- Implementation was complete and verified
- Fix was straightforward with no risk of scope expansion
- Gap was a tooling/scripting omission with clear correction

Items were classified as **frozen** when:
- Fix would require architectural changes (F-01, F-02, F-06)
- Fix would require coordinated multi-module changes (F-01, F-05)
- Current behavior is acceptable for the analytical projection use case (F-03, F-04)
- Fix is out of scope for closure (expanding test coverage, adding features)
