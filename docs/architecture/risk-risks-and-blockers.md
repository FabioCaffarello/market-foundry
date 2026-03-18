# Risk Risks and Blockers

> Detailed analysis of gaps, risks, and blockers that could affect a future `risk` domain in Market Foundry.
> Date: 2026-03-18 | Stage: S59

## Purpose

This document catalogs concrete risks and blockers for the `risk` domain. Items are categorized by severity and include remediation paths with estimated effort.

---

## Blocking Gaps

### BG-1: Adapter Test Debt Is Systemic

**Severity**: HIGH

**Affected files**:
- `internal/adapters/nats/observation_publisher.go` (no tests)
- `internal/adapters/nats/observation_consumer.go` (no tests)
- `internal/adapters/nats/evidence_publisher.go` (no tests)
- `internal/adapters/nats/evidence_consumer.go` (no tests)
- `internal/adapters/nats/signal_publisher.go` (no tests)
- `internal/adapters/nats/signal_consumer.go` (no tests)
- `internal/adapters/nats/decision_publisher.go` (no tests)
- `internal/adapters/nats/decision_consumer.go` (no tests)
- `internal/adapters/nats/strategy_publisher.go` (no tests)
- `internal/adapters/nats/strategy_consumer.go` (no tests)

**Risk**: Adding a 6th domain (risk) with untested publisher/consumer adapters compounds a debt that spans 10 untested files across 5 domains. If a subtle encoding, subject routing, or ack/nak bug exists in the publisher/consumer pattern, it could silently affect all domains including risk.

**Why this matters for risk**: Risk is a downstream consumer of strategy. If the strategy publisher has a latent bug (e.g., incorrect subject formatting under edge cases), risk would inherit that bug and potentially evaluate stale or missing data. The further downstream, the harder to diagnose.

**Remediation**:
- Establish publisher and consumer test patterns using one domain (e.g., evidence or strategy)
- Replicate pattern across remaining domains
- Estimated effort: 1 stage (S60), 2-3 days

---

### BG-2: Derive Actor Scope Has Zero Tests

**Severity**: HIGH

**Affected files**:
- `internal/actors/scopes/derive/sampler_actor.go`
- `internal/actors/scopes/derive/trade_burst_sampler_actor.go`
- `internal/actors/scopes/derive/volume_sampler_actor.go`
- `internal/actors/scopes/derive/signal_sampler_actor.go`
- `internal/actors/scopes/derive/signal_publisher_actor.go`
- `internal/actors/scopes/derive/decision_evaluator_actor.go`
- `internal/actors/scopes/derive/decision_publisher_actor.go`
- `internal/actors/scopes/derive/strategy_resolver_actor.go`
- `internal/actors/scopes/derive/strategy_publisher_actor.go`
- `internal/actors/scopes/derive/source_scope_actor.go`
- `internal/actors/scopes/derive/derive_supervisor.go`
- `internal/actors/scopes/derive/binding_watcher_actor.go`

**Risk**: The derive binary is the most complex binary (15 actor files, multi-family pipelines, cross-domain message routing). Zero actor-level tests means message routing, error handling, and actor lifecycle are verified only through end-to-end smoke tests.

**Why this matters for risk**: A risk evaluator actor in derive would follow the same untested actor pattern. Any message routing bug (wrong PID, dropped message, incorrect type assertion) would be invisible until runtime. The derive binary is the hardest to debug in production because of its fan-out architecture.

**Remediation**:
- Add actor-level unit tests using proto.go test harness for at least: one sampler, one publisher, one evaluator
- Estimated effort: 1 stage (S61), 2-3 days

---

### BG-3: Ingest Actor Scope Has Zero Tests

**Severity**: MEDIUM

**Affected files**:
- `internal/actors/scopes/ingest/websocket_actor.go`
- `internal/actors/scopes/ingest/exchange_scope_actor.go`
- `internal/actors/scopes/ingest/publisher_actor.go`
- `internal/actors/scopes/ingest/binding_watcher_actor.go`
- `internal/actors/scopes/ingest/ingest_supervisor.go`

**Risk**: Ingest is the pipeline entry point. WebSocket actor handles reconnection and message parsing from external exchanges. Without tests, edge cases (connection drops, malformed messages, rate limiting) are untested.

**Why this matters for risk**: Risk is at the end of the pipeline. Ingest quality determines the quality of all downstream domains. However, this gap exists independently of risk and is not a direct blocker for risk design.

**Remediation**:
- Add at minimum: websocket actor message handling test, publisher actor dispatch test
- Estimated effort: 1 sub-stage within S60 or S61

---

## Non-Blocking Risks

### NR-1: Risk Domain Scope Creep

**Severity**: MEDIUM

**Risk**: The word "risk" in financial systems carries enormous conceptual weight — portfolio risk, counterparty risk, market risk, liquidity risk, execution risk. Without tight scoping, the domain could absorb concerns that belong to execution, portfolio management, or external risk feeds.

**Mitigation**:
- risk-domain-design.md must include an explicit "what risk is NOT" section
- First family must be narrowly scoped (e.g., single-position exposure risk, not portfolio-level)
- Boundary invariants (RBI-*) must prohibit importing from execution, portfolio, or external feed domains

---

### NR-2: Binary Placement Decision

**Severity**: MEDIUM

**Risk**: Should risk evaluation run in derive (alongside strategy) or in a separate binary? Both options have trade-offs:
- **In derive**: Simpler deployment, shared actor system, but derive is already the most complex binary
- **Separate binary**: Better isolation, independent scaling, but adds operational complexity

**Mitigation**:
- risk-domain-design.md must include a binary placement section with explicit justification
- Default recommendation: Start in derive (consistent with how signal → decision → strategy were added), extract if complexity warrants

---

### NR-3: Strategy History Not Yet Available

**Severity**: LOW

**Risk**: Risk evaluation might benefit from historical strategy data (e.g., "how many times has this strategy triggered in the last hour?"). Currently only latest-strategy KV buckets exist.

**Mitigation**:
- If risk requires strategy history, add it as a prerequisite (new KV bucket + projection)
- First risk family should be designed to work with latest-only data
- History can be added incrementally in a later stage

---

### NR-4: Single Exchange Source

**Severity**: LOW

**Risk**: Currently only `binancef` is implemented as an exchange adapter. Risk evaluation in production would ideally consider multiple sources for robustness.

**Mitigation**:
- Not a blocker — risk evaluation per-source is valid
- Multi-source aggregation is a future concern, not a first-slice requirement
- Partition keys already include `source`, so multi-source is structurally supported

---

### NR-5: Binding Deactivation Is Incomplete

**Severity**: LOW

**Risk**: Clearing a binding is logged but does not fully stop actors. Requires service restart. If risk actors spawn per binding and a binding is cleared, risk actors would continue processing stale data until restart.

**Mitigation**:
- Document this limitation in risk activation model
- Same limitation exists for all domains — not risk-specific
- Full deactivation is a platform-level improvement, not a risk prerequisite

---

### NR-6: QueryResponderActor Linear Growth

**Severity**: LOW

**Risk**: Adding risk query routes to the QueryResponderActor increases its responsibility linearly. At 10+ evidence types, the actor may need splitting.

**Mitigation**:
- Currently at 7 route types (4 evidence + 1 signal + 1 decision + 1 strategy)
- Adding 1 risk route brings it to 8 — well within acceptable limits
- Split into per-domain responders if/when it reaches 12+

---

### NR-7: Raccoon-CLI Cannot Verify Evaluator Purity

**Severity**: LOW

**Risk**: The CLI checks that risk evaluator files exist but cannot verify they are free of I/O side effects. A risk evaluator that makes HTTP calls or reads files would violate domain purity.

**Mitigation**:
- Code review enforces purity
- All existing evaluators/resolvers/samplers follow the pure pattern
- Consider adding a simple import-path check to raccoon-cli (flag files that import `net/http`, `os`, `io`)

---

## Risk Assessment Matrix

| ID | Description | Severity | Blocking? | Remediation Stage |
|----|-------------|----------|-----------|-------------------|
| BG-1 | Adapter test debt (10 untested files) | HIGH | Recommended | S60 |
| BG-2 | Derive actor tests (12 untested files) | HIGH | Recommended | S61 |
| BG-3 | Ingest actor tests (5 untested files) | MEDIUM | No | S60/S61 |
| NR-1 | Risk scope creep | MEDIUM | — | S62 (design) |
| NR-2 | Binary placement | MEDIUM | — | S62 (design) |
| NR-3 | No strategy history | LOW | — | Future |
| NR-4 | Single exchange | LOW | — | Future |
| NR-5 | Binding deactivation incomplete | LOW | — | Platform |
| NR-6 | QueryResponder linear growth | LOW | — | Future |
| NR-7 | Cannot verify evaluator purity | LOW | — | Future |

---

## Conclusion

The blocking gaps (BG-1, BG-2) are test coverage debts that span the entire codebase — they are not specific to risk and not caused by risk. However, opening a 6th domain without addressing them deepens the debt. The recommended path is to resolve BG-1 and BG-2 in hardening stages before risk implementation (not before risk design).

No architectural or governance blockers prevent risk design from beginning. The foundation is sound.
