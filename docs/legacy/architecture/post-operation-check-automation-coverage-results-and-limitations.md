# Post-Operation Check Automation: Coverage, Results, and Limitations

Authority: S461 -- PO Automation and Verification Pipeline
Predecessor: S447 (Post-Session Operational Verification), S456A (Evidence Gate — PO Automation PARTIAL)

## Coverage Matrix

| Check | Pre-S461 | Post-S461 | Automation level | Surface |
|-------|----------|-----------|------------------|---------|
| PO-1: Kill-switch halt | Scripted, log output | Structured JSON, pass/fail | Full | Script + HTTP |
| PO-2: Post-session backup | Scripted, log output | Filesystem check, structured verdict | Partial (script-only) | Script |
| PO-3: ClickHouse intent records | Scripted, log output | Structured JSON with count evidence | Full | Script + HTTP |
| PO-4: ClickHouse venue responses | Scripted, log output | Structured JSON with count evidence | Full | Script + HTTP |
| PO-5: NATS KV state | Scripted, recorded for review | Structured verdict with KV status | Full | Script + HTTP |
| PO-6: System status | Scripted, recorded for review | Health check with pass/warn verdict | Full | Script + HTTP |
| PO-7: Fee/commission fields | Scripted, pattern-match grep | Structured fill-level fee inspection | Full | Script + HTTP |
| PO-8: Lifecycle consistency | Scripted, **manual review** | Automated via session-explain endpoint | Full | Script + HTTP |
| PO-9: Scope containment | Scripted, log output | Structured verdict with scope counts | Full | Script + HTTP |

### Before S461

- **7 of 9** checks were scripted but produced unstructured log output.
- **PO-8** required manual comparison of CH and KV status values.
- **PO-5, PO-6** were record-only (captured data but no pass/fail verdict).
- No structured output, no session binding, no persistence.

### After S461

- **8 of 9** checks are fully automated with structured JSON verdicts.
- **PO-8** is now automated via the session-explain endpoint (S455A) which performs structured CH-vs-KV consistency checking.
- **PO-2** remains script-only (requires filesystem access for backup verification).
- All checks produce structured evidence, timing, and automation flags.
- Reports are bound to session IDs from S460.
- Reports can be persisted to `backups/sessions/<id>/po-report.json`.

## Automation improvement

| Metric | Pre-S461 | Post-S461 |
|--------|----------|-----------|
| Checks with structured verdict | 0 / 9 | 9 / 9 |
| Checks fully automated (no human review) | 5 / 9 | 8 / 9 |
| Checks with machine-parseable evidence | 0 / 9 | 9 / 9 |
| Session-bound reports | No | Yes |
| Rerunnable | Partially (new log each time) | Yes (same session, fresh report) |
| Persistable | Log files only | Structured JSON + log files |
| Programmatic access (HTTP) | No | Yes (8/9 checks) |

## Test coverage

| Test file | Tests | Coverage |
|-----------|-------|----------|
| `internal/domain/execution/s461_verification_test.go` | 3 tests (5 subtests) | Domain model: POCheck ordering, summary computation, AllPassed logic |
| `internal/application/executionclient/s461_verify_session_test.go` | 10 tests | Use case: validation, nil deps, gate halted/active, intent records, scope pass/fail, fee fields, consistency, full report |

Total: 13 tests, all passing.

## Limitations

### Structural

1. **PO-2 not automatable at HTTP level**: backup verification requires filesystem access. The script surface handles this; the HTTP surface marks it as `manual`.

2. **Hardcoded scope parameters**: checks assume Binance Spot, BTCUSDT, 24h window. When additional symbols or segments are authorized for live trading, checks must be parameterized.

3. **Time window approximation**: checks query "last 24h" rather than exact session start/end times. This is acceptable for single-session scenarios but may produce false positives with multiple sessions in 24h.

### Operational

4. **Gateway must be running**: the HTTP verification endpoint requires the gateway binary with ClickHouse and NATS available. The script surface can partially work with degraded infrastructure.

5. **No automatic persistence**: reports are not automatically saved. The operator must use `--save` (script) or S462 will add automatic persistence.

6. **No diff between runs**: multiple verification runs against the same session are independent. There is no built-in comparison of "previous run vs current run" — this is left to S462.

### Not in scope

7. **No rules engine**: checks are procedural code, not declarative rules. Adding new checks requires code changes. This is intentional per guard rails.

8. **No observability platform**: this pipeline is verification-focused, not monitoring-focused. It runs on demand, not continuously.

9. **No alerting**: there is no notification mechanism for failed checks. The exit code (script) or response structure (HTTP) is the signal surface.

## Residual gap

The only check that remains partially manual is **PO-2 (backup verification)** at the HTTP level. At the script level, all 9 checks produce structured verdicts.

The PO Automation capability moves from **PARTIAL** (S456A) to **SUBSTANTIAL** with S461.
