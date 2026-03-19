# Stage S88: Pre-Venue Design Hardening Report

> Completed: 2026-03-19

## 1. Executive Summary

S88 closes the remaining architectural ambiguities identified in S86 that stand between the current paper-integrated execution system and a future real venue adapter. Four design documents formalize:

1. **Fill reconciliation model** — explicit invariants for how intents and fills are correlated, how divergence is detected, and what reconciliation states exist.
2. **Async fill model and venue intake design** — two-phase execution (submit + track), new event types for partial/rejected/expired fills, and the refined transition from the paper bridge to venue-specific intake subjects.
3. **Credential and activation prerequisites** — environment-variable credential delivery, 17-gate activation ceremony checklist, three-phase post-activation monitoring, and security safeguards.
4. **CI and validation baseline** — 7-stage CI pipeline design, embedded NATS integration test harness, config validation gate, and 8 explicit pre-venue CI criteria.

No venue real was opened. No multi-venue was introduced. No credential infrastructure was built. All deliverables are design artifacts that remove conceptual ambiguity and establish concrete prerequisites for the next frontier.

---

## 2. Hardening Applied

### 2.1 Fill Reconciliation (pre-venue-fill-reconciliation-model.md)

**Problem solved**: Paper mode trivially satisfies reconciliation (every intent gets an instant fill). This masks 7 failure modes that real venues introduce: timeouts, partial fills, rejections, orphan fills, stuck intents, stale fills, and duplicates.

**What was designed**:
- 7 reconciliation invariants (RC-1 through RC-7) with enforcement points
- `ReconciliationStatus` enum (matched, pending, diverged, orphaned, none) with derivation function
- Query-time reconciliation via composite status endpoint (backward-compatible extension)
- Background reconciliation actor design (periodic, store-owned, read-only)
- Health counters for reconciliation outcomes

### 2.2 Async Fill Model and Venue Intake (async-fill-and-venue-intake-design.md)

**Problem solved**: The current fill model is synchronous (PaperVenueAdapter returns filled receipt in one call). Real venues require a two-phase model with intermediate states and polling.

**What was designed**:
- Two-phase execution split: synchronous submission → asynchronous fill tracking
- Four new event types: VenueOrderAcceptedEvent, VenueOrderPartialFilledEvent, VenueOrderRejectedEvent, VenueOrderExpiredEvent
- FillTrackerActor design: polling-based, with recovery from KV on restart, configurable timeout and max active orders
- VenuePort interface extension: GetOrderStatus + CancelOrder (deferred implementation)
- Refined S85 migration plan: event types, subject taxonomy, consumer mapping, backward compatibility analysis
- Paper vs. venue intent semantic comparison table (Final, Status, Fills differences)
- Rate/timing budget for single-venue, single-symbol deployment

### 2.3 Credentials and Activation (venue-credentials-and-activation-prerequisites.md)

**Problem solved**: No credential delivery mechanism existed. No formal process for activating a real venue type.

**What was designed**:
- Environment variable naming convention: `MF_VENUE_{TYPE}_{CREDENTIAL}`
- `LoadCredentials()` contract with fail-fast behavior
- 17-gate activation ceremony checklist (AG-1 through AG-17)
- Three-phase post-activation monitoring: shadow (24h) → guarded (72h) → operational
- Credential security safeguards: no logging, no config files, no global state
- Docker Compose env_file integration with gitignore rules
- VenueConfig schema extension (timeout, testnet flag — credentials stay in env)

### 2.4 CI and Validation Baseline (post-paper-ci-and-validation-baseline.md)

**Problem solved**: Validation layers existed but had no unified pipeline or formal gate criteria.

**What was designed**:
- Current validation inventory: ~155 unit tests, 21 smoke steps, ~25 drift rules
- 5 CI gaps identified: no unified pipeline, no embedded NATS tests, no config validation in CI, no compose health gate, no drift regression protection
- 7-stage CI pipeline: build → test → quality-gate → config-validate → docker-build → compose-health → smoke
- New Makefile targets: `config-validate`, `ci`, `ci-full`, `test-integration`
- Embedded NATS test harness design: 8 scenarios with build tag isolation
- 8 explicit pre-venue CI gate criteria (CI-1 through CI-8)

---

## 3. Files Changed

| File | Change |
|------|--------|
| `docs/architecture/pre-venue-fill-reconciliation-model.md` | New — fill reconciliation invariants and status composition |
| `docs/architecture/async-fill-and-venue-intake-design.md` | New — async fill model and venue intake transition |
| `docs/architecture/venue-credentials-and-activation-prerequisites.md` | New — credential delivery, activation gates, security |
| `docs/architecture/post-paper-ci-and-validation-baseline.md` | New — CI pipeline design and gate criteria |
| `tools/raccoon-cli/src/analyzers/drift_detect.rs` | Updated — added 4 new docs to EXECUTION_DOCS drift rule |
| `docs/stages/stage-s88-pre-venue-design-hardening-report.md` | New — this report |

---

## 4. Limitations Remaining

### 4.1 Unresolved from Prior Stages

| ID | Blocker | Status After S88 |
|----|---------|-----------------|
| HB-POST-1 | No embedded NATS integration tests | DESIGNED (test harness specified), NOT IMPLEMENTED |
| A-1 | Derive actor test coverage | UNRESOLVED (not in S88 scope) |
| A-2 | Automated traceability verification | UNRESOLVED (not in S88 scope) |
| A-3 | Trace metadata persistence in projections | PARTIALLY RESOLVED (domain model has fields, projection copies them) |
| SR-1 | Transitional bridge coupling | DESIGNED (migration plan refined in S88), NOT MIGRATED |
| SR-3 | Synchronous fill model | DESIGNED (async model specified in S88), NOT IMPLEMENTED |

### 4.2 New Limitations Identified in S88

| ID | Limitation | Impact |
|----|-----------|--------|
| S88-L1 | Reconciliation invariants are design-only — no runtime enforcement yet | Real venue would need implementation before activation |
| S88-L2 | FillTrackerActor is polling-based — WebSocket optimization deferred | Higher latency for fill detection (~1s) |
| S88-L3 | Credential model assumes single deployment — no multi-tenant isolation | Acceptable for first venue |
| S88-L4 | CI pipeline has no GitHub Actions/CI service backing — local only | Must be formalized before team scaling |
| S88-L5 | Background reconciliation runs in store — no dedicated reconciliation service | Acceptable for single-venue, may need separation at scale |

### 4.3 Conceptual Ambiguities Resolved

| Ambiguity | Before S88 | After S88 |
|-----------|-----------|-----------|
| How are fills reconciled with intents? | Implicit (paper always matches) | 7 invariants + ReconciliationStatus enum |
| What happens with async venue fills? | Not designed | Two-phase model with 4 new event types |
| How does paper bridge transition to venue intake? | 5-step outline | Full event types, subjects, consumers, backward compat |
| How are venue credentials delivered? | Not designed | Environment variables with fail-fast loading |
| What gates must pass before real venue? | "Activation gate ceremony" (vague) | 17 explicit gates with evidence requirements |
| What does the CI pipeline look like? | Ad-hoc targets | 7-stage pipeline with 8 gate criteria |
| What order should CI improvements be implemented? | Not prioritized | 5-step implementation order |

---

## 5. Guard Rail Compliance

| Guard Rail | Status |
|------------|--------|
| No venue real implemented | Compliant — all deliverables are design documents |
| No multi-venue opened | Compliant — single-venue assumed throughout |
| No credential infrastructure built | Compliant — only naming convention and loading contract designed |
| No premature implementation | Compliant — code changes limited to drift rules (4 new doc entries) |
| Limits and dependencies documented | Compliant — each document has "What Remains Deferred" section |

---

## 6. Preparation for S89

S88 closes the design gap. The system now has explicit answers for fill reconciliation, async fills, credentials, and CI. The natural next stage depends on priority:

### Option A: Activation Gate Ceremony (Recommended)

**S89: Formal Activation Gate Evaluation**

Run the 17-gate checklist from `venue-credentials-and-activation-prerequisites.md` against the current system state. This produces a clear GO/NO-GO for the first real venue adapter and identifies exactly which implementation work is required.

**Why recommended**: Converts design into a decision. No more design documents — the system needs to know which gates pass and which require implementation.

### Option B: CI Pipeline Implementation

**S89: CI Pipeline Hardening**

Implement the CI targets designed in S88: `config-validate`, `ci`, `test-integration` (embedded NATS harness). This closes HB-POST-1 and makes the pipeline concrete.

**Why alternative**: Higher immediate technical value, but doesn't advance the venue frontier directly.

### Option C: Combined

**S89: Activation Gate Dry-Run + CI Implementation**

Run the gate evaluation in parallel with CI implementation. The gate evaluation identifies which gates fail; CI implementation closes the most critical gap (embedded NATS tests).

---

## 7. Acceptance Criteria Verification

| Criterion | Met? |
|-----------|------|
| System gains clearer design for next frontier | Yes — 4 design documents with concrete contracts |
| Conceptual gaps stop being implicit | Yes — 7 ambiguities explicitly resolved |
| Fill reconciliation better defined | Yes — 7 invariants, reconciliation status, background design |
| Async fill model formalized | Yes — two-phase model, event types, fill tracker design |
| Credentials/activation have explicit prerequisites | Yes — 17 gates, env var convention, security safeguards |
| CI/validation baseline documented | Yes — 7-stage pipeline, 8 gate criteria |
| Drift rules updated | Yes — 4 new docs in EXECUTION_DOCS |
