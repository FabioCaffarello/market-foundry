# Stage S37 — Signal Projection Hardening Report

> **Status**: Complete
> **Date**: 2026-03-17
> **Predecessor**: S36 (Signal First Slice)
> **Objective**: Harden the signal domain's projection, replay/idempotency,
> health/readiness, and query path for structural reliability.

## Executive Summary

S37 applied targeted hardening to the signal domain introduced in S36.
The focus was on making `signal` prove structural reliability before
any expansion toward decision layers. All changes strengthen existing
patterns without adding new features or broadening scope.

Key outcomes:
- Health trackers for signal projection and consumer are now properly registered
- KV store guards match the mature evidence pattern (nil checks, explicit error handling)
- Response contracts enforce type-safe JSON output (no omitempty on nullable fields)
- Structured logging includes family context for all signal actors
- Projection ownership and replay/idempotency rules are formally documented

## Hardening Applied

### H-1: Signal Health Tracker Registration (cmd/store/run.go)

**Problem**: Signal tracker definitions (`signal-rsi-projection`, `signal-rsi-consumer`)
were missing from `Run()`. The tracker map was built only from evidence families.
Signal trackers were always `nil` in the supervisor, making `/statusz` blind
to signal pipeline activity.

**Fix**: Added signal tracker creation loop that mirrors the evidence pattern.
Signal trackers are created conditionally based on `pipeline.signal_families`
config, then passed to the supervisor and health server.

### H-2: SignalKVStore Defensive Guards (internal/adapters/nats/signal_kv_store.go)

**Problem (a)**: `Put()` lacked nil guard on `s` and `s.latest`, unlike
`CandleKVStore.Put()`. A nil store pointer would panic.

**Problem (b)**: `Put()` returned `PutSkippedStale` on marshal/write errors,
which is semantically wrong — `PutSkippedStale` means a valid guard decision,
not an error. This masked actual failures in stats counters.

**Problem (c)**: `Get()` returned `nil, nil` for all errors, swallowing
real connectivity issues. Evidence pattern distinguishes `ErrKeyNotFound`
(expected) from actual errors.

**Fixes**:
- Added nil guards on both `Put()` and `Get()` matching candle KV store pattern
- `Put()` now returns `PutWritten` + error problem on failures (consistent with candle)
- `Get()` explicitly checks `jetstream.ErrKeyNotFound` → returns `nil, nil` only for not-found
- Improved monotonicity guard comment alignment with evidence pattern
- Added inline comment for ErrKeyNotFound handling

### H-3: Response Contract Type Safety (internal/application/signalclient/contracts.go)

**Problem**: `SignalLatestReply.Signal` had `json:"signal,omitempty"` tag.
Per S10 hardening rules, nullable fields should always be present in JSON
output (null, not absent) to enable reliable consumer type checking.

**Fix**: Removed `omitempty` from `Signal` field tag. The HTTP handler's
`latestSignalResponse` already had this correct; the NATS reply contract
was the inconsistency.

### H-4: Structured Logging Context (actors/scopes/store/signal_*.go)

**Problem**: Signal consumer and projection actors logged `"actor"` but
not `"family"`, unlike evidence actors that carry full context per log line.

**Fix**: Added `"family", "rsi"` to both signal actor loggers, enabling
family-level filtering in log aggregation.

## Documentation Delivered

### docs/architecture/signal-projection-pattern.md

Canonical reference for how signal events are materialized into the read model:
- Pipeline architecture diagram
- Single-writer invariant rules
- Materialization gates (3 gates, with skip counters)
- Health tracking registration pattern
- Bucket ownership table
- Query path flow
- Activation rules and known limitations

### docs/architecture/signal-replay-idempotency-rules.md

Formal invariants for replay safety:
- INV-1 through INV-5 (Final gate, Validate gate, monotonicity, JetStream dedup, durable consumer)
- Replay safety matrix (7 scenarios)
- Write outcome enum documentation
- Accepted limitations with explicit rationale
- Partition key and deduplication key contracts

## Files Changed

| File | Change |
|------|--------|
| `cmd/store/run.go` | Signal tracker registration |
| `internal/adapters/nats/signal_kv_store.go` | Nil guards, error handling, ErrKeyNotFound |
| `internal/application/signalclient/contracts.go` | Remove omitempty from Signal field |
| `internal/actors/scopes/store/signal_consumer_actor.go` | Add family to logger |
| `internal/actors/scopes/store/signal_projection_actor.go` | Add family to logger |
| `docs/architecture/signal-projection-pattern.md` | NEW — projection ownership doc |
| `docs/architecture/signal-replay-idempotency-rules.md` | NEW — replay/idempotency rules |
| `docs/stages/stage-s37-signal-projection-hardening-report.md` | NEW — this report |

## Verification

- All three service binaries compile: `store`, `gateway`, `derive`
- All signal-related tests pass: domain, sampler, client, handlers, routes, adapters
- No new dependencies introduced
- No scope expansion (no new features, no new signal types)

## Known Remaining Limitations

1. **No signal history projection** — only latest KV bucket exists; no historical
   lookback. Acceptable: signals are refreshed every evidence window.
2. **RSI only** — MACD and other signal types are deferred. Registry has
   `LatestSpecByType()` dispatch ready for extension.
3. **No raccoon-cli signal drift rules** — signal contract governance is not
   yet automated in the CLI guardian.
4. **No signal expiration events** — stale signals remain in KV until overwritten.
   TTL-based expiration is deferred.
5. **Single binary assumption** — monotonicity guard is sufficient for single-writer
   deployments; multi-instance convergence is safe but produces redundant writes.

## S38 Preparation Recommendations

1. **Raccoon-CLI signal governance** — add drift rules for signal contracts,
   ensuring signal registry, KV buckets, and subjects remain consistent.
2. **Signal readiness review** — formal assessment of whether signal is mature
   enough to feed a decision layer.
3. **MACD sampler** — if signal proves reliable in production, MACD is the
   natural next signal family to implement.
4. **Signal history** — if decision-making needs lookback, add a history bucket
   following the candle history pattern.
5. **Multi-symbol proof for signal** — verify signal pipeline handles multiple
   concurrent symbols without contention.

## Guard Rail Compliance

| Guard Rail | Status |
|------------|--------|
| No strategy/decision layer opened | Compliant |
| No broad redesign | Compliant — targeted fixes only |
| No generic framework created | Compliant |
| Limitations not hidden | Documented in 5 items above |
| Scope limited to hardening | Compliant — 0 new features |
