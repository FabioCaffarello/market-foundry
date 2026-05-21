# Stage S38 — Decision Readiness Review Report

> **Status**: Complete
> **Date**: 2026-03-17
> **Predecessor**: S37 (Signal Projection Hardening)
> **Objective**: Formal readiness assessment for opening a decision/strategy
> domain in Market Foundry.

## Executive Summary

S38 conducted a rigorous, evidence-based readiness review across all layers
of Market Foundry to determine whether the system is structurally prepared
to introduce a decision/strategy domain. The review examined source code,
test coverage, governance tooling, documentation, and build health.

**Verdict**: **NOT READY** — but conditionally achievable in 2–3 targeted stages.

The foundational architecture (observation → evidence → signal → store → gateway)
is structurally consistent and follows proven patterns. However, three specific
blockers prevent safe decision entry:

1. Signal adapter tests are missing (registry + KV store)
2. raccoon-cli has no signal governance rules
3. Signal multi-symbol behavior is unverified

These are concrete, scoped gaps — not systemic problems requiring redesign.

---

## Assessment Summary

### What Is Mature

| Area | Evidence |
|------|----------|
| Observation domain | Trade model tested, ingest actors functional, multi-symbol proven |
| Evidence domain | 3 families (candle, trade_burst, volume), full client coverage, history projection, replay-safe |
| Signal domain model | Signal entity tested, RSI sampler tested, client tested, HTTP surface tested |
| Store authority | Single-writer invariant, monotonicity guards, health tracking |
| Gateway | Clean read-only surface, all routes tested |
| Activation model | Config-driven, opt-in semantics, configctl lifecycle |
| Build health | All 13 workspace modules compile cleanly, `go work sync` succeeds |

### What Is Not Yet Mature

| Area | Gap |
|------|-----|
| Signal NATS adapters | `signal_registry.go` — no test; `signal_kv_store.go` — no test |
| Evidence KV stores | `trade_burst_kv_store.go`, `volume_kv_store.go` — no tests |
| Signal governance | raccoon-cli is blind to signal contracts, subjects, buckets |
| Signal multi-symbol | Not verified under concurrent multi-symbol operation |
| Actor tests | Zero tests across all actor scopes (systemic, not decision-specific) |

---

## Readiness Questions — Answered

### Is observation mature enough?

**Yes.** Observation has a tested domain model, functional ingest actors,
a tested NATS registry, and proven multi-symbol support. Single-exchange
limitation (binancef) is acceptable and not decision-relevant.

### Is evidence mature enough?

**Yes, with a caveat.** The candle family has full end-to-end test coverage.
Trade burst and volume families work but their KV stores lack unit tests.
This is a low-effort gap to close.

### Is signal mature enough?

**Not yet.** Signal has structural parity with evidence in terms of pipeline
architecture. S37 hardened the projection path. But signal's adapter layer
(registry, KV store) has no tests, governance tooling doesn't cover signal
contracts, and multi-symbol behavior is unverified. A decision layer consuming
signals would depend on an untested and ungoverned contract.

### Does store continue as clear authority?

**Yes.** Projection ownership is well-documented. Single-writer invariant
is enforced. Monotonicity guards are consistent across evidence and signal.
Health tracking covers both evidence and signal families (since S37).

### Is gateway still clean?

**Yes.** Gateway is a read-only HTTP surface. It delegates all queries
to NATS request/reply. No business logic. All routes tested. Adding
decision routes would follow the existing pattern with zero changes
to the gateway framework.

### Can raccoon-cli audit the current state?

**Partially.** raccoon-cli governs architecture boundaries, evidence
contracts, topology, and drift. It does **not** govern signal contracts.
This means `make check` and `make verify` cannot catch signal drift.
The CLI needs signal rules before decision entry.

### What gaps still prevent decision from entering without generating debt?

1. **Signal adapter test coverage** — decision would consume from an
   untested contract
2. **Signal governance** — drift would propagate silently from signal
   to decision
3. **Signal multi-symbol proof** — decision operating on multiple symbols
   would depend on unverified behavior

### What is the smallest acceptable design for a future decision layer?

A decision domain following the exact structural pattern of existing domains:
domain entity → application evaluator → actor scope → NATS adapters → HTTP routes → cmd entrypoint.
The evaluator consumes signal + evidence via NATS request/reply, applies a
rule, publishes decision events, projects to a KV bucket. Zero new patterns.

See `decision-risks-and-blockers.md` § "Smallest Acceptable Decision Layer Design".

---

## Documents Delivered

| Document | Purpose |
|----------|---------|
| `docs/architecture/decision-readiness-review.md` | Layer-by-layer maturity assessment with consolidated readiness matrix |
| `docs/architecture/decision-entry-prerequisites.md` | 5 specific pre-conditions with definitions of done and staging proposal |
| `docs/architecture/decision-risks-and-blockers.md` | 3 blockers + 6 risks with mitigations and smallest acceptable design |
| `docs/stages/stage-s38-decision-readiness-review-report.md` | This report |

---

## Recommendation

**Do not open decision in the next stage.**

Instead, execute the following sequence:

| Stage | Focus | Outcome |
|-------|-------|---------|
| **S39** | Adapter test coverage sweep | P-1 (signal registry test), P-2 (signal KV store test), P-3 (evidence KV store tests) |
| **S40** | raccoon-cli signal governance | P-4 (signal drift rules in fast/deep profiles) |
| **S41** | Signal multi-symbol verification | P-5 (extend smoke-multi for signal queries) |
| **S42** | Decision domain design | Architecture doc, not implementation — contracts, stream families, evaluator model |
| **S43** | Decision first slice | Minimal implementation following the proven domain pattern |

This sequence closes all identified gaps, adds the decision gate to
the governance tooling, and ensures the decision domain is introduced
with the same rigor applied to observation, evidence, and signal.

---

## Guard Rail Compliance

| Guard Rail | Status |
|------------|--------|
| No decision/strategy implemented | Compliant — review only |
| Gaps not masked | 3 blockers + 6 risks documented explicitly |
| No vague abstractions | All findings cite specific files, tests, and patterns |
| Blockers are concrete and prioritized | 5 prerequisites with dependency map |
| Readiness vs implementation distinction maintained | Compliant — review produces gate, not code |

---

## Conclusion

Market Foundry has built a structurally sound foundation through 37 stages
of disciplined, incremental evolution. The observation → evidence → signal
pipeline follows consistent patterns, the store maintains clear authority,
and the gateway is clean. The governance tooling provides real enforcement.

The system is **close** to decision readiness but not there yet. The gaps
are specific, scoped, and achievable. Closing them will take 2–3 stages —
a worthwhile investment to ensure decision enters on a solid foundation
rather than compensating for uncertainties in layers below it.

The next wave of stages should be planned from the prerequisite map, not
from domain ambition. Evidence-driven progression, not anxiety-driven
expansion.
