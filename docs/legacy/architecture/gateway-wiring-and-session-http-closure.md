# Gateway Wiring and Session HTTP Closure

**Stage**: S465
**Date**: 2026-03-24
**Status**: COMPLETE
**Wave**: Session Access & Verification Closure (S464--S468)

---

## 1. Context

The S463 evidence gate closed the Session Intelligence wave with two MEDIUM gaps:

- **G3**: `VerifySession` use case existed but was never wired in the gateway composition. The `/session/:id/verify` route returned 503 at runtime despite the handler, route, and use case all existing.
- **G4**: `AuditSessionUseCase` was constructed with `nil` for both `verifyUseCase` and `fillReader`. The `/session/:id/audit` endpoint returned a degraded audit bundle with no verification report and no fee analysis.

Both gaps were localized to `cmd/gateway/compose.go:248--269` -- the session wiring block.

---

## 2. Root Cause

The original S462 implementation created `AuditSessionUseCase` with placeholder `nil` parameters and included comments stating "wired below if ClickHouse available". However, the wiring was never implemented -- no code path after the session block connected verification or fill readers.

The verification use case (`VerifySessionUseCase`) was also never constructed in the composition root because it requires readers from both NATS (gate status, session metadata) and ClickHouse (intent records, venue responses, fill records). At the time of S462, no adapter bridged the existing `clickhouse.ExecutionReader` to the simpler `VerifyCHSummary` / `VerifyCHLister` interfaces.

---

## 3. Resolution

### 3.1 ClickHouse Session Reader Adapters

Created `cmd/gateway/session_reader.go` with two adapters:

| Adapter | Satisfies | Delegates To |
|---------|-----------|-------------|
| `sessionCHSummaryAdapter` | `executionclient.VerifyCHSummary` | `ExecutionReader.QueryExecutionList` with 24h window |
| `sessionCHListerAdapter` | `executionclient.VerifyCHLister` AND `executionclient.AuditCHFillReader` | `ExecutionReader.QueryExecutionList` with type/status filters |

Both adapters are constructed only when `chClient != nil` (ClickHouse configured). The lister adapter satisfies both the verification and audit fill reader interfaces since they share the `List24h` signature.

### 3.2 Gateway Composition Wiring

Updated `cmd/gateway/compose.go` session block to:

1. Build ClickHouse adapters when `chClient != nil`
2. Build gate reader from execution control gateway when available
3. Construct `VerifySessionUseCase` with all available readers -- **closes G3**
4. Pass the verify use case and fill reader into `AuditSessionUseCase` -- **closes G4**

### 3.3 Graceful Degradation

All reader dependencies remain optional:
- Without ClickHouse: verification checks that require CH return `verdict: skip`
- Without execution control gateway: gate halt check returns `verdict: skip`
- Without consistency checker: lifecycle consistency check returns `verdict: skip`

This preserves the existing degradation semantics -- each PO check independently handles missing readers.

---

## 4. Session HTTP Surface After Closure

| Endpoint | Method | Use Case | Status After S465 |
|----------|--------|----------|-------------------|
| `/session/list` | GET | `ListSessionsUseCase` | Fully wired (since S460) |
| `/session/:id` | GET | `GetSessionUseCase` | Fully wired (since S460) |
| `/session/:id/verify` | GET | `VerifySessionUseCase` | **Now wired** (was nil) |
| `/session/:id/audit` | GET | `AuditSessionUseCase` | **Now fully wired** (was degraded) |

All four endpoints are registered when the session gateway is available. Verification and audit now produce complete output when ClickHouse and execution control are also configured.

---

## 5. Limitations

1. **Consistency checker remains nil**: The `VerifyConsistencyChecker` parameter is passed as `nil` because cross-surface consistency reads (comparing CH and KV for the same partition key) require a composite reader not yet composed at the gateway level. The lifecycle consistency PO check returns `verdict: skip`. This is a LOW gap acceptable for S465 scope.

2. **24h fixed window**: The CH adapters use a fixed 24-hour lookback window for `Summary24h` and `List24h`. Session-bounded time windows require verification parameterization (S466 scope).

3. **Hardcoded scope symbols**: Several PO checks query `BTCUSDT` specifically. This is inherited from S461 and out of scope for this wiring stage.

---

## 6. Files Changed

| File | Change |
|------|--------|
| `cmd/gateway/session_reader.go` | NEW -- ClickHouse session reader adapters |
| `cmd/gateway/session_reader_test.go` | NEW -- compile-time interface assertions and structural tests |
| `cmd/gateway/compose.go` | MODIFIED -- session wiring block rewritten with full reader composition |

---

## 7. References

- [S464 Charter](../stages/stage-s464-session-access-verification-charter-report.md)
- [S463 Evidence Matrix](../architecture/session-intelligence-evidence-matrix-residual-gaps-and-next-ceremony.md)
- [Session HTTP Surface Readers](./session-http-surface-readers-composition-and-limitations.md)
