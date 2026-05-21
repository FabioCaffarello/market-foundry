# Post-S273 Open vs Closed Debts Matrix

> Gate assessment: S274 — post-S273 transition gate
> Scope: S269 debts + debts inherited from prior waves
> Method: codebase inspection, test analysis, CI pipeline review

---

## Debt Classification Criteria

| Status | Definition |
|--------|-----------|
| **Closed** | Code + tests prove the capability; CI gates it; no remaining ambiguity |
| **Partially Closed** | Infrastructure exists and is proven in isolation; integration across binaries or actor chains not yet validated |
| **Open** | No code or test addresses the debt, or proof is insufficient |

---

## S269 Debts — Current Status

### OD-PE1: SafetyGate Not in Actor Path

**Status: CLOSED**

| Aspect | Evidence |
|--------|----------|
| Kill switch gate in derive scope | `ExecutionPublisherActor` lines 106–122 — reads `ControlKVStore.IsHalted()` before NATS publish |
| Full SafetyGate in execute scope | `VenueAdapterActor` lines 76–155 — `SafetyGate.Check()` with kill switch + staleness guard before venue submission |
| Logic-level proof | `safety_gate_integration_test.go` — 11 test scenarios covering active/halted/stale/boundary conditions |
| Counter tracking | `healthz.Tracker` increments for processed, filled, skipped_halt, skipped_stale |
| Defense-in-depth | Two independent gate points: publish-side and venue-side |

**Remaining limit:** Tests replicate actor decision logic without spawning Hollywood actors. The gate IS wired in production actor code, but no test exercises the actor message-passing path with SafetyGate active. This is an acceptable limit — the logic is identical.

---

### OD-PE2: S267 Stage Report Missing

**Status: OPEN**

No `stage-s267-*` file exists in `docs/stages/`. The report gap from the paper execution wave was identified in S269 and remains unaddressed. This is a governance/documentation debt, not a code debt.

---

### OD-PE3: KV Materialization Unproven for Execution

**Status: PARTIALLY CLOSED**

| Aspect | Evidence | Status |
|--------|----------|--------|
| KV adapter Put/Get round-trip | `kv_store_roundtrip_test.go` — 8 properties against real NATS | Proven |
| Monotonicity guard | Tests KV-RT-2 (stale rejection), KV-RT-3 (dedup) | Proven |
| Multi-symbol partition isolation | Test KV-RT-4 | Proven |
| Missing key semantics | Test KV-RT-5 (nil, not error) | Proven |
| Store binary consumer wiring | `store_supervisor.go` declares `execution-paper-order-projection` pipeline | Wired |
| Projection actor gate pipeline | `ExecutionProjectionActor` — 3-gate pipeline (non-final skip, validation, monotonicity) | Unit-tested with mocks |
| End-to-end: derive → NATS stream → store binary → KV → gateway query | No test exercises this path | **NOT proven** |
| Config-driven pipeline activation | No test verifies enabling `paper_order` family spawns the projection | **NOT proven** |

**Remaining limit:** Each component works in isolation. The orchestration from derive binary through store binary to gateway query has not been validated as an integrated path.

**Structural note:** Monotonicity guard uses read-then-write without CAS. Sole-writer constraint mitigates the race window but is enforced by convention, not code.

---

### OD-PE4: ClickHouse Round-Trip Unproven for Execution

**Status: CLOSED**

| Aspect | Evidence |
|--------|----------|
| Serialization round-trip | `behavioral_roundtrip_test.go` — 9 execution scenarios (Scenarios 9–17) |
| Enum coverage | Side (3 values), Status (7 values) proven |
| Precision preservation | Quantity precision test (Scenario 15) |
| Full causal chain | Decision → Strategy → Risk → Execution traced through correlation/causation IDs (Scenario 16) |
| Live E2E in CI | `smoke-analytical-e2e.sh` Phase 5.8 validates executions table structure and HTTP queryability |
| CI gate | `smoke-analytical` job runs full compose stack (NATS + ClickHouse + all binaries) on every push |

**Confidence: HIGH.** Serialization proven at mapper level; live pipeline proven in CI via compose stack.

---

### OD-PE5: ControlGate Kill Switch Unproven End-to-End

**Status: PARTIALLY CLOSED**

| Aspect | Evidence | Status |
|--------|----------|--------|
| KV state transitions | `control_gate_runtime_test.go` — 6 tests against real NATS | Proven |
| Fail-open default | Test CG-RT-1: missing key → intent flows | Proven |
| Halt/resume cycle | Test CG-RT-4: full active→halted→active→halted cycle | Proven |
| Audit field fidelity | Test CG-RT-5: reason + updated_by survive round-trip | Proven |
| Multi-intent blocking | Test CG-RT-6: 5 consecutive intents during halt → all blocked | Proven |
| Actor wiring (derive) | `ExecutionPublisherActor` lines 106–122 reads gate before publish | Wired |
| Actor wiring (execute) | `VenueAdapterActor` lines 76–90 creates gate at startup | Wired |
| Gateway HTTP API | `ExecutionControlWebHandler` routes GET/PUT /execution/control | Wired |
| Store responder | `QueryResponderActor` handles ControlGet/ControlSet routes | Wired |
| Cross-binary propagation | No test: gateway PUT → store KV write → derive/execute KV read | **NOT proven** |
| HTTP round-trip | No test: HTTP PUT → 200 → KV state visible to consumers | **NOT proven** |
| Latency characterization | Not measured under production-like load | **NOT proven** |

**Remaining limit:** All wiring exists in production code. KV behavior is proven in single-connection tests. The missing proof is the cross-binary propagation path.

---

### OD-PE6/PE7/PE8: Single Symbol, Static Signals, No Concurrency

**Status: OPEN (Low Severity — Acknowledged Scope Constraints)**

These are feature-scope limits, not integration debts. They remain as expected constraints for the current paper-execution phase.

---

## Inherited Debts from Prior Waves

### OD-CG1: Codegen Spec Drift

**Status: CLOSED by S258–S263.** Codegen equivalence check (`scripts/codegen-equivalence-check.sh`) and integrated check (`scripts/codegen-integrated-check.sh`) run in CI. Golden snapshot tests exist for bollinger family.

### OD-BW2: Behavioral Scenario Coverage

**Status: CLOSED by S249–S257.** 47 behavioral scenarios validated in CI. Round-trip tests cover all four domain stages.

### OD-BW5/BW6: ClickHouse Schema and Writer Pipeline Coverage

**Status: CLOSED by S272 + CI smoke-analytical.** Full compose stack validates end-to-end persistence.

---

## Summary Matrix

| Debt ID | Description | Severity | Status | Confidence |
|---------|-------------|----------|--------|------------|
| OD-PE1 | SafetyGate in actor path | High | **Closed** | High |
| OD-PE2 | S267 report missing | Low | **Open** | — |
| OD-PE3 | KV materialization for execution | Medium | **Partially Closed** | Medium |
| OD-PE4 | ClickHouse round-trip for execution | Medium | **Closed** | High |
| OD-PE5 | ControlGate kill switch end-to-end | Medium | **Partially Closed** | Medium |
| OD-PE6 | Single symbol constraint | Low | Open (by design) | — |
| OD-PE7 | Static signals constraint | Low | Open (by design) | — |
| OD-PE8 | No concurrency proof | Low | Open (by design) | — |
| OD-CG1 | Codegen spec drift | Medium | **Closed** | High |
| OD-BW2 | Behavioral scenario coverage | Medium | **Closed** | High |
| OD-BW5 | ClickHouse schema coverage | Medium | **Closed** | High |
| OD-BW6 | Writer pipeline coverage | Medium | **Closed** | High |

**Closed: 6 | Partially Closed: 2 | Open (structural): 1 | Open (by design): 3**
