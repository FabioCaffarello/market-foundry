# S372: Compose-Level Orchestration Wiring Validation — Report

**Status:** Complete
**Phase:** 38 — Multi-Binary Orchestration Proof Wave
**Predecessor:** S371 (Binary Boundary and Event-Flow Contract Audit)
**Successor:** S373 (End-to-End Multi-Binary Pipeline Proof)

---

## Executive Summary

S372 transforms the topology audited in S371 into validated compose-level
orchestration. The stage delivers a proven Docker Compose wiring layer that
boots all 9 services (7 Go binaries + NATS + ClickHouse) in correct
dependency order, establishes NATS JetStream connectivity with all 9 streams
and 44 durable consumers, and confirms cross-binary communication via NATS
request/reply — all without artificial shortcuts.

The existing compose infrastructure was already substantially correct. S372
added the **validation layer** (automated wiring smoke) and **formal
documentation** (boot order, readiness protocol, limitations) that were
missing between the compose file and the S373 end-to-end proof.

---

## What Was Delivered

### 1. Compose Wiring Validation Script

**File:** `scripts/smoke-compose-wiring.sh`
**Entrypoint:** `make smoke-compose-wiring`

An 8-phase automated validation that proves compose-level orchestration:

| Phase | Validation | Scope |
|-------|-----------|-------|
| 1 | Compose boot order | All 9 services reach healthy state |
| 2 | NATS infrastructure readiness | JetStream operational, all services pass /readyz |
| 3 | JetStream stream existence | All 9 canonical streams present |
| 4 | Consumer bindings | All 44 durable consumers bound across binaries |
| 5 | Cross-binary connectivity | gateway ↔ store, gateway ↔ configctl via NATS |
| 6 | Service isolation | Separate containers, correct PID namespace |
| 7 | Port allocation | Host exposure (gateway only), internal isolation |
| 8 | Boot dependency chain | All declared dependencies met at runtime |

### 2. Architecture Documentation

**Compose-level orchestration wiring:**
`docs/architecture/compose-level-orchestration-wiring.md`

Covers: topology overview, service inventory, dependency graph, NATS stream/consumer
wiring, health contract, configuration wiring, validation instructions, limitations.

**Boot order, readiness, and limitations:**
`docs/architecture/multi-binary-runtime-boot-order-readiness-and-limitations.md`

Covers: 5-phase boot order, readiness protocol (preflight + continuous), stream
creation timing, 10 documented limitations (L1–L10), operational commands,
S373 preparation.

### 3. Makefile Integration

- New target: `make smoke-compose-wiring`
- Added to `.PHONY` declarations
- Added to `smoke-help` output

### 4. Stage Index Update

`docs/stages/INDEX.md` updated with S372 entry.

---

## Compose Wiring Analysis

### What Was Already Correct

The existing Docker Compose file (`deploy/compose/docker-compose.yaml`) was
already well-structured:

- All 9 services defined with correct images and build contexts
- `depends_on` with `service_healthy` conditions on all edges
- Health checks using `/readyz` for Go services, native checks for infrastructure
- Shared bridge network (`market-foundry-network`)
- Security hardening (cap_drop ALL, no-new-privileges, non-root user)
- JSONC config files mounted read-only
- Restart policy (unless-stopped) with init: true

### What S372 Added

1. **Automated validation of stream existence** — prior to S372, there was no
   automated check that all 9 JetStream streams were created after boot.

2. **Consumer binding verification** — the 44 durable consumers were
   documented in S371 but never verified automatically at runtime.

3. **Cross-binary connectivity proof** — gateway ↔ store and gateway ↔ configctl
   NATS request/reply was exercised by existing smoke scripts but not isolated
   as a wiring-level check.

4. **Service isolation verification** — container count and PID namespace
   checks confirm each binary runs in its own process.

5. **Port allocation audit** — automated verification that only gateway, NATS,
   and ClickHouse are exposed on the host.

6. **Boot dependency chain integrity** — runtime verification that all
   declared dependencies are healthy when the dependent service is running.

### What Was NOT Changed

- No changes to the Docker Compose file itself
- No changes to Go service code
- No changes to NATS or ClickHouse configuration
- No changes to JSONC config files
- No changes to the Dockerfile

The existing compose wiring was correct. S372's contribution is the
**validation and documentation layer** that proves it.

---

## Governing Questions Addressed

From the S370 charter (MBO-Q1 through MBO-Q8):

| Question | S372 Contribution |
|----------|-------------------|
| MBO-Q1: Binary startup and readiness orchestration | **Answered.** Boot order validated, readiness protocol documented, all 9 services boot to healthy. |
| MBO-Q2: Cross-binary pipeline correctness | Partially. Structural wiring proven; data flow deferred to S373. |
| MBO-Q3: Correlation chain preservation | Not addressed. Deferred to S373. |
| MBO-Q4: Single-binary restart recovery | Not addressed. Deferred to S374. |
| MBO-Q5: NATS transient disconnection | Not addressed. Deferred to S374. |
| MBO-Q6: Kill-switch propagation | Not addressed. Deferred to S374. |
| MBO-Q7: ClickHouse writer path | Partially. Writer boots and binds consumers; data flow deferred to S373. |
| MBO-Q8: Automated smoke command | **Answered.** `make smoke-compose-wiring` is the automated command. |

---

## Risk Register Update (from S371)

| Risk | S371 Rating | S372 Status |
|------|-------------|-------------|
| RISK-1: Transitional bridge in execute | LOW | Unchanged. Bridge exists but does not affect wiring. |
| RISK-2: Dual intake paths in execute | MEDIUM | Unchanged. Both consumers bind correctly. |
| RISK-3: Event metadata loss in KV | LOW | Unchanged. Not a wiring concern. |
| RISK-4: Compose startup order vs stream creation | MEDIUM | **Mitigated.** Verified: all streams exist after boot. Consumer retry handles the race. |
| RISK-5: Gateway readiness depends on store | LOW | **Verified.** Compose dependency chain ensures store is healthy before gateway starts. |
| RISK-6: Writer buffer eviction | LOW | Unchanged. Not a wiring concern. |
| RISK-7: NATS reconnection during restart | LOW | Unchanged. Deferred to S374. |

---

## Capabilities Validated (from S370 charter)

| Capability | Status |
|------------|--------|
| MBO-C1: Binary startup and readiness orchestration | **Validated** |
| MBO-C2: Cross-binary event delivery via NATS JetStream | **Structurally validated** (streams + consumers exist; data flow deferred to S373) |
| MBO-C3: KV materialization across process boundaries | Not addressed (S373) |
| MBO-C4: End-to-end pipeline correctness | Not addressed (S373) |
| MBO-C5: Correlation chain preservation | Not addressed (S373) |
| MBO-C6: Binary restart recovery | Not addressed (S374) |
| MBO-C7: NATS reconnection resilience | Not addressed (S374) |
| MBO-C8: Kill-switch cross-binary propagation | Not addressed (S374) |
| MBO-C9: ClickHouse analytical write path | Partially (writer boots, binds; data flow deferred to S373) |
| MBO-C10: Compose-level operational smoke | **Validated** |

---

## Key Invariants Verified

From the S371 handoff document (MBI-1 through MBI-10):

| Invariant | S372 Status |
|-----------|-------------|
| MBI-1: Startup ordering | **Verified** — compose dependency graph correct |
| MBI-2: Event flow single-writer | **Verified** — stream ownership confirmed |
| MBI-3: Correlation chain | Deferred to S373 |
| MBI-4: KV materialization | Deferred to S373 |
| MBI-5: Consumer paths | **Verified** — all 44 consumers bound |
| MBI-6: Fill materialization | Deferred to S373 |
| MBI-7: Analytical persistence | Deferred to S373 |
| MBI-8: Binary restart recovery | Deferred to S374 |
| MBI-9: Kill-switch propagation | Deferred to S374 |
| MBI-10: Duplicate prevention | Deferred to S373 |

---

## Files Changed

| File | Action | Purpose |
|------|--------|---------|
| `scripts/smoke-compose-wiring.sh` | Created | 8-phase compose wiring validation |
| `docs/architecture/compose-level-orchestration-wiring.md` | Created | Compose topology, wiring, health contract |
| `docs/architecture/multi-binary-runtime-boot-order-readiness-and-limitations.md` | Created | Boot order, readiness, 10 limitations |
| `docs/stages/stage-s372-compose-level-orchestration-wiring-report.md` | Created | This report |
| `docs/stages/INDEX.md` | Updated | Added S372 entry |
| `Makefile` | Updated | Added `smoke-compose-wiring` target |

---

## Limitations Identified

10 limitations documented in detail:

1. **L1:** No stream pre-creation (lazy by publishers)
2. **L2:** No cross-stream ordering guarantee
3. **L3:** No TLS between services
4. **L4:** No resource limits on Go services
5. **L5:** Restart policy masks persistent failures
6. **L6:** Single NATS server (no clustering)
7. **L7:** Single ClickHouse node (no replication)
8. **L8:** Writer consumer timing (may start before publishers)
9. **L9:** No readiness beyond NATS TCP check
10. **L10:** configctl/gateway port collision (internal only)

None of these limitations block S373. All are documented for future hardening.

---

## Preparation for S373

S373 (End-to-End Multi-Binary Pipeline Proof) can now proceed with confidence
that the structural wiring layer is correct. The recommended S373 sequence:

1. `make up` — boot the full compose stack
2. `make smoke-compose-wiring` — verify structural wiring (S372 gate)
3. `make seed` — activate ingestion bindings
4. Wait for pipeline data flow (candle materialization)
5. Verify end-to-end correlation chain (configctl → ingest → derive → store → gateway)
6. Verify ClickHouse analytical path (writer → ClickHouse → gateway /analytical/*)
7. Verify all terminal sinks receive correct data
8. Document the multi-binary pipeline proof

---

## Acceptance Criteria Evaluation

| Criterion | Status |
|-----------|--------|
| Pipeline can be levantado em binarios separados | **Met** — 9 containers, 7 Go binaries |
| Conectividade compose-level provada | **Met** — NATS streams, consumers, request/reply |
| Reduz dependencia de execucao monolitica | **Met** — no single-process shortcuts |
| Base pronta para prova ponta a ponta | **Met** — structural wiring validated, S373 can proceed |

---

## Guard Rails Compliance

| Guard Rail | Status |
|------------|--------|
| Nao abrir producao plena | **Compliant** — development/proof only |
| Nao inflar para orchestration platform ampla | **Compliant** — Docker Compose only, no K8s/Nomad |
| Nao abrir multi-venue | **Compliant** — paper simulator only |
| Nao mascarar dependencias frageis | **Compliant** — 10 limitations explicitly documented |

---

*S372 complete. Pipeline structurally wired. Ready for S373.*
