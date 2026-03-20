# Post-Consolidation Readiness Review

> Formal assessment after S137–S141: is the current capability baseline genuinely consolidated, and is the Foundry ready to plan the next strategic wave?

## 1. Executive Summary

The consolidation wave (S137–S141) succeeded in its primary mission: the current capability baseline is now **canonically defined, operationally observable, recovery-documented, and ergonomically governed**. The Foundry is no longer a collection of working services — it is a documented, validated, and governable system with explicit limits.

However, consolidation also surfaced structural realities that must be accepted, not papered over:

- **In-memory state remains the single largest architectural constraint.** Cold-start warm-up for RSI at 3600s timeframes takes ~15 hours. Candle samplers are lost on restart. This is accepted for the current baseline but is a hard gate for any expansion.
- **ClickHouse infrastructure is prepared but inert.** Entry principles, migration catalog conventions, and signal candidates are documented — but no migration tool, no schema, and no writer exist.
- **The operational loop is robust for its scope** (2 symbols, 4 timeframes, 9 families), but has not been proven under higher cardinality or extended uptime beyond manual validation windows.

**Verdict:** The baseline is consolidated. The system is ready to **plan** — not implement — the ClickHouse/migrations preparation wave.

---

## 2. What Consolidation Actually Proved

### 2.1 Baseline Is Canonical (S137)

| Claim | Evidence |
|---|---|
| The operational baseline is formally defined | 30 success criteria across 5 tiers in `current-capability-baseline-success-criteria.md` |
| The canonical loop is documented end-to-end | `current-capability-baseline-definition.md` covers all 7 runtime processes, 8 layers, 9 families |
| Cold-start behavior is explicit | `current-baseline-cold-start-and-state-limits.md` documents 5 phases, ~10–20 KB in-memory state, data loss windows per timeframe |
| Success criteria are testable | Tiers 1–3 validatable in 20 min; full validation in 75 min |

### 2.2 Operations Are Observable (S139)

| Claim | Evidence |
|---|---|
| Diagnostic endpoints exist on every service | `/healthz`, `/readyz`, `/statusz`, `/diagz` — all implemented in `internal/shared/healthz/` |
| Phase classification is automatic | starting → warming → active → idle → stalled, computed from tracker state |
| Runbook exists for common operations | `current-baseline-runbook.md` covers start, validate, diagnose, recover |
| Quick diagnostics are scripted | `scripts/diag-check.sh` provides single-command stack health snapshot |

### 2.3 Recovery Semantics Are Documented (S140)

| Claim | Evidence |
|---|---|
| Shutdown sequence is bounded | 15-second max window, documented per-service |
| What survives restart is explicit | NATS streams, KV projections, consumer positions, config |
| What is lost on restart is explicit | In-memory samplers, in-flight orders, health tracker counters |
| Accepted limitations are enumerated | L-01 through L-05 with explicit rationale |

### 2.4 Ergonomics and Governance Are Formalized (S141)

| Claim | Evidence |
|---|---|
| Shared script library exists | `scripts/utils/lib.sh` with logging, JSON helpers, compose helpers, canonical defaults |
| Configuration is self-documenting | `deploy/configs/CONFIG-REFERENCE.md`, inline JSONC comments on all config files |
| ClickHouse entry principles are defined | 7 principles in `future-clickhouse-and-migrations-entry-principles.md` |
| Migration catalog conventions exist | `future-migration-catalog-organization-guidelines.md` with naming, numbering, idempotency rules |
| Family/venue/timeframe addition is governed | `current-capability-ergonomics-and-governance.md` |

---

## 3. What Consolidation Did NOT Prove

These are not failures — they are honest limits of what consolidation can demonstrate:

1. **Extended uptime stability.** The baseline has been validated in windows of 20–75 minutes. Multi-day continuous operation has not been formally validated.

2. **Higher cardinality behavior.** 2 symbols × 4 timeframes × 9 families is the proven envelope. 10+ symbols or 8+ timeframes remain theoretical.

3. **Recovery under partial failure.** Documentation covers full restart and crash recovery. Partial failures (one service dies while others continue) are documented but not stress-tested.

4. **Operational cost of the in-memory constraint.** The 15-hour RSI warm-up for 3600s is documented and accepted, but its operational impact on a running system (e.g., after an unplanned restart) has not been experienced in production-like conditions.

5. **Cross-session continuity.** Each restart is effectively a clean slate for derived state. No mechanism exists to resume from where the system left off.

---

## 4. Consolidation Quality Assessment

| Dimension | Rating | Rationale |
|---|---|---|
| Baseline definition completeness | **Strong** | 30 criteria, 5 tiers, all layers covered |
| Operational observability | **Strong** | 4 diagnostic endpoints, phase classification, scripted checks |
| Recovery documentation | **Strong** | Explicit survival/loss matrix, bounded shutdown, accepted limitations |
| Ergonomic governance | **Strong** | Shared lib, config reference, family addition rules |
| ClickHouse preparation documentation | **Adequate** | Principles and conventions exist; no implementation artifacts |
| Extended operation confidence | **Insufficient** | No multi-day validation, no partial-failure testing |
| State persistence readiness | **Not started** | In-memory only; hard gate for expansion acknowledged but unaddressed |

---

## 5. Remaining Friction and Open Items

### From TC-01 (still relevant)

| ID | Item | Status after consolidation |
|---|---|---|
| D-01 | State persistence (candle samplers, RSI accumulators) | Still open — hard gate for TC-02 and ClickHouse writer |
| D-02 | Per-timeframe idle detection | Deferred — adequate at 4 TFs |
| D-03 | RSI convergence formal proof | Deferred — accepted on empirical basis |
| D-04 | Per-binding timeframe customization | Deferred — no heterogeneous demand yet |
| D-05 | Query observability (latency, throughput metrics) | Deferred — no external consumers |
| D-06 | Window state persistence | Same as D-01 |
| D-07 | Gateway aggregate view | Deferred — adequate at current cardinality |

### From consolidation wave

| ID | Item | Notes |
|---|---|---|
| C-01 | No `cmd/migrate` tool | Entry principles and catalog conventions defined; tool does not exist |
| C-02 | No ClickHouse schema | Signal candidates catalogued; no DDL written |
| C-03 | No ClickHouse writer service | Architecture clear; no code exists |
| C-04 | No multi-day uptime validation | Documented but untested |
| C-05 | Gateway lacks tracker integration | `/statusz` and `/diagz` return empty tracker data for gateway |

---

## 6. Verdict

The consolidation wave achieved its goals. The current capability baseline is **canonically defined, operationally observable, recovery-documented, and ergonomically governed**. The Foundry is a coherent, validated system within its proven envelope.

The system is **ready to plan** the next strategic wave. It is **not ready to implement** ClickHouse integration directly — the preparation phase (migration tooling, schema design, writer architecture) must come first as a distinct, scoped effort.

**Gate decision:** PASS for planning. Implementation requires the preparation gate defined in `clickhouse-and-migrations-preparation-gate.md`.
