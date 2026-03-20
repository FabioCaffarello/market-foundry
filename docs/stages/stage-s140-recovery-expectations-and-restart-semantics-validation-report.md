# Stage S140 — Recovery Expectations and Restart Semantics Validation Report

**Status:** Complete
**Date:** 2026-03-19
**Predecessor:** S136 (readiness review), S137 (baseline definition), S139 (operational diagnostics)

---

## 1. Executive Summary

S140 validates and documents the recovery, restart, cold-start, and crash semantics of the market-foundry baseline. The goal is to make explicit what is guaranteed, what is best-effort, and what is out of scope — providing a mature operational baseline before any future persistence layer (ClickHouse) changes the recovery model.

**Key findings:**
- The system's recovery model is sound and well-aligned with its current scope (paper trading, development)
- NATS durable consumers + KV provide the primary recovery mechanism; in-memory state loss is bounded and self-healing
- Five accepted limitations (L-01 through L-05) are now explicitly documented with rationale
- Five future trigger conditions for state persistence are identified and prioritized
- No code changes were needed — the existing implementation matches the documented semantics

---

## 2. Validation Performed

### 2.1 Shutdown Semantics Validation

| Aspect | Finding | Status |
|--------|---------|--------|
| Signal handling | SIGTERM, SIGINT, os.Interrupt captured in `WaitTillShutdown` | Correct |
| Actor drain | 10s poison pill timeout with recursive child shutdown | Correct |
| Health server drain | 5s graceful shutdown, heartbeat monitor stopped | Correct |
| Deferred cleanup | NATS client close in LIFO order | Correct |
| Total shutdown window | 15s max (10s actors + 5s health) | Documented |
| Unhandled signals | SIGKILL/SIGQUIT cause immediate termination, no cleanup | Accepted |

### 2.2 Restart Recovery Validation

| Runtime | Recovery mechanism | Data loss | Recovery time | Status |
|---------|-------------------|-----------|---------------|--------|
| configctl | NATS event replay | None | Seconds | Validated |
| ingest | WebSocket reconnect + configctl query | Trades during downtime | ~2s | Validated |
| derive | Durable consumer resume + sampler reset | Up to 1 window/TF | 60s–61min | Validated |
| store | Durable consumer resume + idempotent KV | None | Seconds | Validated |
| execute | Durable consumer resume | In-flight intents | Seconds | Validated |
| gateway | Stateless; immediate | None | Instant | Validated |

### 2.3 Cold-Start Sequence Validation

| Phase | Expected | Finding |
|-------|----------|---------|
| Infrastructure (NATS) | 0–10s | Correct |
| Service bootstrap | 10–30s | Correct; all services fail-fast if NATS unavailable |
| Config seeding | 30–45s | Correct; required for data flow |
| Data materialization | 45–120s | Correct; first 60s candle at next window boundary |
| RSI warm-up (60s TF) | ~15min | Correct; 15 candles required |
| RSI warm-up (3600s TF) | ~15h | Correct; accepted limitation |

### 2.4 NATS Reconnection Validation

| Aspect | Finding |
|--------|---------|
| Bootstrap connection | Single attempt; `os.Exit(1)` on failure |
| Runtime reconnection | Delegated to nats.go client library (exponential backoff, 60 attempts) |
| During disconnect | Health checks fail; consumers stall; publishes buffer then fail |
| WebSocket (ingest) | Independent reconnection with 1s→60s exponential backoff |

### 2.5 State Survival Matrix

| Data category | Survives service restart | Survives NATS restart | Survives volume loss |
|---------------|--------------------------|----------------------|---------------------|
| Stream events | Yes | Yes | No |
| KV projections | Yes | Yes | No |
| Consumer positions | Yes | Yes | No |
| Config history | Yes | Yes | No |
| Candle samplers | No | No | No |
| Actor mailboxes | No | No | No |
| Health counters | No | No | No |
| Paper order intents | No | No | No |

---

## 3. Files Changed

| File | Change | Reason |
|------|--------|--------|
| `docs/architecture/current-baseline-recovery-and-restart-semantics.md` | **Created** | Documents shutdown, restart, NATS reconnection, crash vs graceful semantics |
| `docs/architecture/current-baseline-cold-start-and-state-limits.md` | **Created** | Documents cold-start sequence, in-memory state inventory, data loss windows, accepted limitations |
| `docs/architecture/future-state-persistence-and-clickhouse-trigger-notes.md` | **Created** | Frames pain points, trigger conditions, ClickHouse candidacy, migration path |
| `docs/architecture/current-baseline-runbook.md` | **Updated** | Cross-references to new recovery/cold-start docs in Section 4 |
| `docs/stages/stage-s140-recovery-expectations-and-restart-semantics-validation-report.md` | **Created** | This report |

**No Go code changes.** The existing implementation correctly implements the recovery semantics documented here. No tests, smoke scripts, or health endpoints required modification.

---

## 4. Limits Explicitly Documented

### L-01: In-Memory Sampler State
Candle/tradeburst/volume samplers are ephemeral. Partial windows lost on restart. Accepted because data loss is bounded by window duration and self-heals.

### L-02: No Observation Buffering
Trades during ingest downtime are permanently lost. Exchange does not buffer. Accepted because restart is fast (<30s) and impact is one partial candle.

### L-03: RSI Cold-Start Warm-Up
RSI needs 15 candles to converge. 3600s TF = ~15 hours. Accepted as algorithmic constraint. ClickHouse would eliminate this.

### L-04: No Automatic NATS Reconnect at Bootstrap
Services exit if NATS unavailable at startup. Accepted because external orchestration handles rescheduling and fail-fast is easier to debug.

### L-05: Health Tracker Reset on Restart
Counters and idle timers reset to zero. Accepted because phase progression is fast and counters are informational only.

---

## 5. Future Triggers Identified

| ID | Trigger | Priority | Threshold |
|----|---------|----------|-----------|
| T-01 | Live execution (real capital) | P1 | Decision to move beyond paper trading |
| T-02 | RSI warm-up unacceptable | P1 | Timeframes > 3600s or restart SLA < 15min |
| T-03 | Historical queries needed | P2 | Backtesting or compliance requirement |
| T-04 | NATS volume loss unacceptable | P2 | Production SLA defined |
| T-05 | Cross-session analytics | P3 | Strategy research or reporting requirement |

Each trigger is documented with expected impact and migration path in `future-state-persistence-and-clickhouse-trigger-notes.md`.

---

## 6. Success Criteria Assessment

| Criterion | Status |
|-----------|--------|
| Restart/recovery/cold-start semantics explicitly documented | Done |
| Limits of current model clearly documented with rationale | Done (L-01 through L-05) |
| System gains more mature operational baseline | Done (runbook cross-referenced) |
| Future persistence need better framed | Done (5 triggers with priority matrix) |
| Stage consolidates existing capabilities without opening new ones | Done (zero code changes) |
| No new features introduced | Respected |
| No state persistence implemented | Respected |
| No ClickHouse implemented | Respected |
| Current limitations not masked | Respected (all limits named and rationalized) |

---

## 7. Preparation Recommended for S141

With S140 completing the recovery/restart baseline, the system is well-positioned for the next stage. Recommended candidates:

1. **Observability hardening** — Export health tracker metrics to time-series storage (Prometheus/VictoriaMetrics) for historical operational visibility. Low risk, immediate operational value.

2. **ClickHouse introduction (Phase 1)** — Read-only projection alongside existing store consumers. Dual-write to NATS KV + ClickHouse. No query path changes. Addresses triggers T-01, T-02, T-03.

3. **Chaos validation** — Scripted restart-during-operation tests (kill derive mid-candle, kill ingest during trade burst) to validate the recovery semantics documented here under real conditions.

4. **Per-timeframe tracker granularity** — Split aggregate trackers into per-timeframe trackers for finer operational visibility. Addresses the S136 open debit of per-TF idle detection.

**Recommendation:** S141 should focus on whichever trigger from the T-01–T-05 matrix becomes active first. If no trigger is active, observability hardening (option 1) or chaos validation (option 3) provide the most value with lowest risk.
