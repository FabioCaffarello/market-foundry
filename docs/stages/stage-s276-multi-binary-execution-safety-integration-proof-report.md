# Stage S276 — Multi-Binary Execution Safety Integration Proof

**Status**: Complete
**Date**: 2026-03-21

## Objective

Prove the integration between control, execution safety, and paper order flow when derive, execute, and control surface components operate as distinct binaries communicating exclusively through NATS.

## Prior State

| Stage | What it proved | Gap |
|-------|---------------|-----|
| S271 | KV round-trip persistence | Single store instance |
| S273 | Control gate runtime halt/resume | All components in same process |
| S275 | Control plane full path (publisher gate → stream) | All components in same process |

All prior proofs used shared Go references within a single test process. No test validated the cross-binary boundary where NATS is the sole communication medium.

## Approach

Simulated three independent binaries, each with its own NATS connections:

1. **Control surface**: writes gate state to `EXECUTION_CONTROL` KV bucket
2. **Derive binary**: reads gate state + publishes paper orders to `EXECUTION_EVENTS` stream
3. **Execute binary**: reads gate state + subscribes to paper orders + applies SafetyGate + fills via PaperVenueAdapter

Binary isolation enforced by:
- Separate `natsclient.Connect()` calls per binary (6-7 total connections)
- No shared Go-level references between binaries
- NATS as the sole shared medium

## Test Results

Six tests, all passing (0% flakiness across 5× repeated runs):

| Test | Property | Result |
|------|----------|--------|
| MB-1 | Normal flow: derive → stream → execute fills | PASS |
| MB-2 | Halt propagates: both binaries see halted state | PASS |
| MB-3 | Cross-binary safety: active→halt→derive+execute both blocked | PASS |
| MB-4 | Resume propagates: halt→resume→both sides allow | PASS |
| MB-5 | Full cycle: active→halt→resume coherent across boundary | PASS |
| MB-6 | KV materialization: causal trace + fill record survive boundary | PASS |

### Test Location

`internal/adapters/nats/natsexecution/multi_binary_integration_test.go`

## Deliverables

| Deliverable | Path |
|-------------|------|
| Integration test (6 scenarios) | `internal/adapters/nats/natsexecution/multi_binary_integration_test.go` |
| Architecture proof document | `docs/architecture/multi-binary-execution-safety-integration-proof.md` |
| Operational shape findings | `docs/architecture/multi-binary-operational-shape-findings.md` |
| Stage report (this file) | `docs/stages/stage-s276-multi-binary-execution-safety-integration-proof-report.md` |

## Key Findings

1. **KV state propagation is immediate** across independent NATS connections — no eventual consistency delay
2. **Dual-gate safety holds** across binary boundary — both derive-side and execute-side gates observe the same KV state
3. **Causal trace survives** the full derive→stream→execute path (correlation ID, causation ID, fill records)
4. **Core NATS subscriptions** are more reliable than JetStream push consumers for test scenarios requiring deterministic activation
5. **Fail-open default** works correctly — binaries can start before control surface writes any gate state

## Acceptance Criteria

| Criterion | Status |
|-----------|--------|
| Reproducible proof of multi-binary flow | Met — 6 tests, 0% flakiness |
| Control/safety observable between processes | Met — MB-2, MB-3 prove cross-binary gate propagation |
| System closer to operational validation | Met — proves NATS-as-IPC shape with dual-gate safety |
| Closes structural gap without unnecessary infra | Met — no Docker, no compose, no external dependencies beyond NATS |

## Limitations

- Tests run in a single Go process (separate connections, not separate OS processes)
- Single NATS server (not clustered)
- No network partition testing
- No JetStream consumer durability/redelivery testing across binary restarts
- Paper venue adapter only (no real venue)

## Open Debts

| Debt | Severity | Notes |
|------|----------|-------|
| OS-level process isolation test | Medium | Would require shell script or Docker harness |
| NATS cluster resilience | Medium | Single server sufficient for correctness, cluster needed for availability |
| Consumer redelivery across binary restart | Low | JetStream guarantees cover this; not tested at integration level |
| Real venue adapter integration | Low | Out of scope — paper mode is the current operational target |

## Recommendations for S277

1. **Compose-level smoke test**: Extend `scripts/smoke-analytical-e2e.sh` to include control gate halt/resume verification across running Docker containers
2. **Consumer durability proof**: Prove that JetStream durable consumer resumes correctly after execute binary restart (simulated via consumer stop/restart)
3. **Observability wiring**: Prove that writer binary materializes paper orders and fills from the multi-binary flow to ClickHouse, closing the analytical round-trip across binary boundaries
