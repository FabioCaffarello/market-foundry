# ADR 0012: ControlGate fail-open posture

## Status

Accepted. Formalization of an existing intentional choice.

## Context

The **ControlGate** is the operator-controlled execution halt
mechanism. Its state is a single key (`"global"`) in a NATS KV bucket
(`EXECUTION_CONTROL`), with two possible statuses: `active` (trading
allowed) and `halted` (trading blocked). Phase 4.1 docs and the
Phase 3 audit refer to it as "the kill switch"; in code it is
exclusively `ControlGate`.

`ControlKVStore.IsHalted(ctx) bool` is consulted from four operational
sites:

| Site | File |
|---|---|
| Per-intent gate check before venue submit | `internal/application/execution/safety_gate.go:51` |
| Per-publish gate check in derive | `internal/actors/scopes/derive/execution_publisher_actor.go:106` |
| Between retry attempts | `internal/application/execution/retry_submitter.go:229` |
| Startup activation-surface log | `internal/actors/scopes/execute/venue_adapter_actor.go:166` |

Five distinct failure modes can interrupt a clean read:

1. **`nil_bucket`** — `s == nil` or `s.bucket == nil` (init race).
2. **`key_not_found`** — fresh deployment; no operator has written
   the gate yet.
3. **`ctx_timeout`** — caller's context expired or was cancelled.
4. **`kv_error`** — NATS reachability, JetStream errors, etc.
5. **`unmarshal_error`** — KV value present but corrupt.

Today, `IsHalted` returns `false` (i.e. "not halted, continue
trading") on every one of these modes. Two code comments call this
"fail-open" explicitly, but no ADR formalizes the choice or its
trade-offs.

The asymmetric counterpart matters: query/admin paths
(`QueryResponderActor.handleExecutionControlGet`,
`VerifySessionUseCase.checkGate`, the HTTP CRUD routes) propagate
read errors back to the caller. Operators reading the gate via
HTTP see infrastructure issues; the silent path is only the
operational hot loop.

P4.4 surfaced two gaps:
- The decision was undocumented as an ADR.
- The silent failure was unobservable: the `marketfoundry_execution_gate_active`
  gauge tracks the runtime verdict (allowed/blocked), not the KV
  state, so a transient KV outage looked identical to "operator
  wants active" on dashboards.

## Decision

**Operational paths fail-open.** When any of the five failure modes
occurs inside `ControlKVStore.IsHalted`, the method returns `false`
and the calling pipeline continues submitting orders.

**Query/admin paths surface the error.** Read failures via the HTTP
control endpoints, the activation surface query, and post-session
verification all return the problem to the caller.

**Default state at init is `Active`.** `DefaultControlGate()` returns
`{Status: GateActive}`. Combined with `key_not_found` being treated
as a failure (and thus producing `false` from `IsHalted`), a fresh
deployment is operationally indistinguishable from one whose
operator has just written `active`.

**Each failure mode is now observable.** A new counter
`marketfoundry_execution_gate_read_failures_total{reason}` is
incremented at each error path inside `IsHalted`, with five
canonical `reason` label values matching the failure modes above.
Operators can distinguish a `nil_bucket` race at startup from
sustained `kv_error` rates or one-off `unmarshal_error` events
without changing any runtime semantics.

## Rationale

Trading remains protected by eight layers of defense-in-depth:

1. `paper_simulator` as the default venue (ADR 0007).
2. `dry_run` mode default (`DryRunSubmitter` decorator).
3. Credentials gating (`AdapterVenue` without credentials degrades
   to `ModeVenueDegraded` and skips execution).
4. **ControlGate** (this layer).
5. Staleness guard (5-minute default).
6. Segment source guard.
7. Rate limiter (P4.2).
8. Retry submitter halt re-check between attempts.

For fail-open to result in actual harm, **all** of these must
align: live venue + valid credentials + dry-run off + non-stale
intent + a transient KV failure simultaneous with an operator's
halt intent. This is a narrow compound failure surface.

Inside that surface, the trade-off is **availability vs.
conservative halt**. Fail-open prefers availability for these
reasons:

- The KV layer (NATS JetStream FileStorage, single-writer per
  ADR 0008, 1 MB bucket) is operationally cheap and reliable.
- Operator-initiated halts are not time-critical in the same sense
  as automated kill conditions would be — there are no automated
  conditions today (Section "Automated halts" below).
- Defense-in-depth catches every other compound failure mode.
- Fail-closed would convert routine NATS deployment churn into
  business-impacting outages, eroding operator trust in the
  signal when a real halt event happens.

The observability counter closes the most acute objection to the
posture: that operational metrics could no longer distinguish "all
healthy" from "we have no idea". A non-zero rate on any failure
reason is now a first-class operational signal.

## Consequences

### Positive

- **Availability preserved during transient KV issues.** Routine
  NATS deployment churn does not halt trading.
- **Operator trust preserved.** Halts (when they happen) are not
  diluted by frequent infrastructure-noise halts.
- **Aligned with ADR 0007's manual-control stance** — the system
  is deliberately conservative about automated mode changes.
- **Now monitorable.** Every IsHalted failure mode increments a
  labeled counter; dashboards and alerts can react to sustained
  non-zero rates per reason.

### Negative

- **A genuine halt intent simultaneous with a KV failure is
  missed.** This is the irreducible risk that the new counter
  surfaces but does not eliminate.
- **Implementation duplication.** `IsHalted` no longer delegates
  to `Get` — it inlines the JetStream read so each failure mode
  can be categorized. `Get` retains its existing semantics for
  query/admin callers, who get a `*problem.Problem` rather than
  a label.
- **Five reason labels are documentation-light by themselves.**
  Operators reading metrics need to know which labels are
  expected at low rate (e.g., `key_not_found` during first boot)
  vs. concerning (`kv_error` sustained).

### Mitigation

- The counter (`marketfoundry_execution_gate_read_failures_total`)
  is the primary mitigation; without it the choice would be
  defensible only on faith.
- Tier E backlog item: dashboard + alert rules for the counter
  (out of scope for this ADR — operations concern).
- Tests in `control_kv_store_unit_test.go` and
  `control_kv_store_gate_read_failures_test.go` lock the
  contract: each reason increments exactly its label, and
  IsHalted returns `false` in every failure mode.

## When to revisit

This decision should be reconsidered if **any** of the following
becomes true:

1. The counter shows a sustained non-zero rate on `kv_error` or
   `ctx_timeout` (i.e., KV reliability is not what this ADR
   assumes).
2. A production incident occurs where a halt was needed but
   missed because of fail-open.
3. Compliance or regulatory requirements change such that a
   provable halt becomes externally required.
4. The eight-layer defense-in-depth weakens — for example, if
   `dry_run` ceases to be the default, or `paper_simulator` is
   removed.
5. Multi-region or multi-venue deployment changes the KV
   reliability profile (a single regional NATS outage halting
   global trading is operationally different from today's
   single-deployment posture).
6. An automated halt condition is introduced (e.g., excessive
   rejection rate trips the gate). At that point the meaning of
   "halt missed" shifts from operator intent to safety
   automation, and the trade-off changes.

## Alternatives considered

Three hybrid strategies were enumerated in the P4.4 design
discussion (`/tmp/p4.4-kill-switch-design-discussion.md`,
§7.3 H1/H2/H3). All three are deferred to the design-meta queue
pending data from the new counter:

- **M16 (H1) — Cached state with staleness threshold.** Cache the
  last-known gate state in-process; on KV read failure, return
  the cached value if its age is below a configured threshold,
  fail-closed otherwise. Provides availability under brief
  transients with bounded divergence from operator intent.

- **M17 (H2) — Conditional fail-closed.** Fail-closed for the
  live-venue + credentials-present configuration; fail-open for
  paper/dry-run. Matches blast radius to risk surface, at the
  cost of bifurcated semantics.

- **M18 (H3) — Distinguish `ErrKeyNotFound`.** Treat
  `key_not_found` (fresh-boot ergonomics) as a non-failure that
  defaults to active; treat the other four modes as
  potentially-fail-closed under M16 or M17 semantics.

All three are data-driven decisions once the counter has accumulated
operational history; until then the simpler fail-open posture
documented here is the chosen baseline.

## References

- P4.4 design discussion: `/tmp/p4.4-kill-switch-design-discussion.md`
  (read-only investigation that fed this ADR).
- Code: `internal/adapters/nats/natsexecution/control_kv_store.go`
  (`IsHalted`, `Get`, `Put`).
- Code: `internal/domain/execution/control.go` (`ControlGate`,
  `DefaultControlGate`).
- Code: `internal/shared/metrics/metrics.go`
  (`executionGateReadFailuresTotal`, `IncGateReadFailure`,
  `GateReadFailureCount`, reason-label constants).
- Tests: `internal/adapters/nats/natsexecution/control_kv_store_unit_test.go`,
  `…/control_kv_store_gate_read_failures_test.go`.
- ADR [0007](0007-paper-venue-default.md) — paper venue default
  underwrites the "narrow compound failure surface" claim.
- ADR [0008](0008-single-writer-invariant.md) — single-writer
  invariant constrains the KV's failure modes.
