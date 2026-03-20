# Decision Readiness Review

> **Status**: S38 — Formal readiness assessment for opening a decision/strategy layer.
> **Date**: 2026-03-17
> **Verdict**: **NOT READY** — conditional readiness achievable in 2–3 targeted stages.
> **Predecessor**: S37 (Signal Projection Hardening)

---

## Executive Summary

This review evaluates whether Market Foundry has the structural maturity to
safely introduce a decision/strategy domain without creating architectural debt
or reopening drift. The assessment covers every layer from observation through
signal, the store authority model, gateway query surface, governance tooling,
and activation model.

**Conclusion**: The foundational layers (observation, evidence, signal) exist
and follow consistent patterns. However, specific gaps — primarily in test
coverage of the actor layer, signal governance automation, and evidence
family completeness — mean that opening a decision layer today would force
the new domain to compensate for uncertainties in layers below it.

The recommendation is to close 3 specific gaps before opening decision,
achievable in 2–3 focused stages.

---

## Layer-by-Layer Assessment

### 1. Observation — Maturity: **ADEQUATE**

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Domain model | ✅ Solid | `observation/trade.go` + test, `events.go` |
| Ingest actors | ✅ Functional | 5 actors: binding_watcher, exchange_scope, supervisor, publisher, websocket |
| Exchange adapter | ✅ Single exchange | `binancef/aggtrade.go` + test, `websocket.go` |
| NATS transport | ✅ Tested | `observation_registry.go` + test, publisher, consumer |
| Application binding | ✅ Tested | `ingest/binding.go` + test |
| Multi-symbol | ✅ Proven | `make smoke-multi` exercises btcusdt + ethusdt |
| Actor tests | ❌ Zero | No tests in `actors/scopes/ingest/` |

**Assessment**: Observation is mature enough to feed evidence reliably.
The single-exchange limitation (binancef only) is acceptable for the
current scope. The lack of actor-level tests is a known systemic gap
(shared across all scopes) but does not block decision entry — observation
actors are structurally simple (forward events, no business logic).

**Decision-blocking**: No.

---

### 2. Evidence — Maturity: **ADEQUATE WITH CAVEATS**

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Domain model | ✅ Complete | `candle.go` + test, `trade_burst.go`, `volume.go` + test |
| Derive samplers | ✅ Tested | `sampler.go` + test, `trade_burst_sampler.go` + test, `volume_sampler.go` + test |
| Evidence client | ✅ Full coverage | 4 use cases, all tested (latest candle, history, trade_burst, volume) |
| NATS registry | ✅ Tested | `evidence_registry.go` + test |
| KV stores | ⚠️ Partial | `candle_kv_store.go` + test; `trade_burst_kv_store.go` and `volume_kv_store.go` **no tests** |
| Store projections | ✅ Functional | candle, trade_burst, volume projection actors |
| Actor tests | ❌ Zero | No tests in `actors/scopes/derive/` or `actors/scopes/store/` |
| HTTP surface | ✅ Tested | `handlers/evidence.go` + test, `routes/evidence.go` + test |
| Multi-timeframe | ✅ Proven | 60s and 300s via config |
| Candle history | ✅ Exists | `get_candle_history.go` + test, history bucket pattern |
| Replay/idempotency | ✅ Documented | `replay-idempotency-rules.md` with 5 invariants |

**Assessment**: Evidence is the most mature domain layer. Candles have full
coverage from domain through HTTP. Trade burst and volume KV stores lack
unit tests — they follow the candle pattern but haven't been independently
verified at the adapter level. This is a minor gap but should be closed
before decision relies on these families as inputs.

**Decision-blocking**: No, but trade_burst/volume KV store tests should
be completed as a pre-condition.

---

### 3. Signal — Maturity: **MINIMUM VIABLE, NOT HARDENED**

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Domain model | ✅ Tested | `signal.go` + test, `events.go`, `Validate()`, `DeduplicationKey()` |
| RSI sampler | ✅ Tested | `rsi_sampler.go` + test |
| Signal client | ✅ Tested | `get_latest_signal.go` + test, contracts |
| NATS registry | ⚠️ No test | `signal_registry.go` exists, **no test file** |
| Signal KV store | ⚠️ Hardened, no test | Nil guards + error handling (S37), but **no unit test** |
| Signal publisher | ✅ Exists | `signal_publisher.go` |
| Signal consumer | ✅ Exists | `signal_consumer.go` |
| Store projection | ✅ Hardened (S37) | Health trackers, structured logging, defensive guards |
| HTTP surface | ✅ Tested | `handlers/signal.go` + test, `routes/signal.go` + test |
| Signal families | ⚠️ RSI only | MACD and others deferred; registry dispatch ready |
| Signal history | ❌ None | Latest-only KV bucket; no historical lookback |
| Raccoon-CLI rules | ❌ None | No signal drift rules in governance tooling |
| Multi-symbol proof | ❌ Not verified | Signal pipeline not smoke-tested with multiple symbols |

**Assessment**: Signal has structural parity with evidence in terms of
pipeline architecture (consumer → projection → KV → query). S37 hardened
the projection path significantly. However, signal remains a single-type
(RSI) domain with no governance automation, no adapter-level tests, and
no history bucket. A decision layer that consumes signals would be consuming
from a domain that has not yet proven:
- Registry contract stability (no test)
- KV store correctness independently (no test)
- Multi-symbol behavior
- More than one signal type

**Decision-blocking**: **Yes** — signal registry and KV store need tests,
and raccoon-cli needs signal drift rules before decision can safely depend
on signal outputs.

---

### 4. Store (Projection Authority) — Maturity: **ADEQUATE**

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Single-writer invariant | ✅ Enforced | One projection actor per family per deployment |
| KV bucket ownership | ✅ Clear | Each family owns its bucket exclusively |
| Monotonicity guard | ✅ Consistent | All KV stores implement read-before-write |
| Health tracking | ✅ Complete (S37) | Evidence + signal trackers registered |
| Supervisor | ✅ Exists | `store_supervisor.go` |
| Actor tests | ❌ Zero | No tests in `actors/scopes/store/` |

**Assessment**: The store maintains clear authority over the read model.
Projection ownership is well-documented and consistently implemented.
The monotonicity guard pattern is proven across evidence and signal.
The absence of actor-level tests is the systemic gap — not unique to store.

**Decision-blocking**: No. Store's authority model is sound for decision
to read from.

---

### 5. Gateway (Query Surface) — Maturity: **ADEQUATE**

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Read-only surface | ✅ Enforced | Gateway never writes; delegates to NATS request/reply |
| Evidence routes | ✅ Tested | Latest candle, history, trade_burst, volume |
| Signal routes | ✅ Tested | Latest signal by type |
| Configctl routes | ✅ Tested | Full lifecycle surface |
| Health/readiness | ✅ Tested | `readiness.go` + test, `healthz` |
| Clean separation | ✅ | No business logic in gateway |

**Assessment**: The gateway is clean, tested, and follows the read-only
pattern consistently. A decision layer would add its own query routes
following the same pattern — no gateway changes needed beyond new routes.

**Decision-blocking**: No.

---

### 6. Governance / raccoon-cli — Maturity: **ADEQUATE WITH GAP**

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Architecture guard | ✅ | Layer boundary enforcement |
| Contract audit | ✅ | Messaging pattern validation |
| Drift detect | ✅ | Cross-layer semantic drift detection |
| Topology audit | ✅ | Service topology validation |
| Quality gate profiles | ✅ | fast, ci, deep profiles |
| Integration tests | ✅ | `tests/cli_integration.rs` |
| Signal governance | ❌ | No signal-specific drift rules |

**Assessment**: raccoon-cli is a mature governance tool for the evidence
domain. It does not yet know about signal contracts, subjects, or KV
buckets. Adding a decision layer without signal governance would mean
two ungoverned domains (signal + decision) — compounding the governance
gap rather than closing it.

**Decision-blocking**: **Yes** — signal drift rules must exist before
decision is introduced, to prevent two ungoverned layers stacking.

---

### 7. Activation Model — Maturity: **ADEQUATE**

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Config-driven activation | ✅ | `pipeline.evidence_families`, `pipeline.signal_families` |
| Opt-in semantics | ✅ | Absent family = no actors spawned |
| configctl lifecycle | ✅ | Draft → Validate → Compile → Activate → Deactivate |
| Settings schema | ✅ Tested | `schema.go` + test |

**Assessment**: The activation model is well-designed and proven. A decision
layer would follow the same `pipeline.decision_families` pattern. No changes
needed to the activation framework.

**Decision-blocking**: No.

---

## Consolidated Readiness Matrix

| Layer | Maturity | Decision-Blocking? |
|-------|----------|-------------------|
| Observation | Adequate | No |
| Evidence | Adequate with caveats | No |
| Signal | Minimum viable | **Yes** |
| Store | Adequate | No |
| Gateway | Adequate | No |
| Governance | Adequate with gap | **Yes** |
| Activation | Adequate | No |

---

## Test Coverage Summary

| Layer | Domain | Application | Adapters | Actors | HTTP |
|-------|--------|-------------|----------|--------|------|
| Observation | ✅ | ✅ | ✅ | ❌ | N/A |
| Evidence | ✅ | ✅ | ⚠️ Partial | ❌ | ✅ |
| Signal | ✅ | ✅ | ❌ | ❌ | ✅ |
| Store | N/A | N/A | ⚠️ Partial | ❌ | N/A |
| Gateway | N/A | N/A | N/A | N/A | ✅ |

**Systemic gap**: Zero actor-level tests across all scopes. This is not a
decision-specific blocker but increases risk for any new domain introduction.

---

## Verdict

**NOT READY** for decision layer entry today.

The foundational architecture is sound. The patterns are consistent. The
governance framework exists. But two specific domains need targeted
hardening before decision can safely depend on them:

1. **Signal needs adapter-level test coverage** (registry + KV store)
2. **raccoon-cli needs signal governance rules**

These are achievable in 2–3 focused stages without broad redesign.

See [decision-entry-prerequisites.md](decision-entry-prerequisites.md) for
the specific pre-conditions and [decision-risks-and-blockers.md](decision-risks-and-blockers.md)
for the risk analysis.
