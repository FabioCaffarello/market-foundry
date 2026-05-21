# Stage S498 — Operational Hardening Wave Charter and Scope Freeze — Report

**Status**: COMPLETE
**Date**: 2026-03-28
**Predecessor**: S497 (Cross-Session Position Continuity Evidence Gate — FULL PASS)
**Wave**: Operational Hardening (S498–S502)

---

## Objective

Open a formal, short wave to harden the operational behavior of the execution pipeline. The system has closed three consecutive analytical-layer waves (S452a–S497) and the analytical read layer is substantially complete. The next bottleneck is operational/runtime: fee persistence gaps, session lifecycle close edge cases, and writer stability under sustained multi-segment operation.

---

## What Was Done

### 1. State Analysis

Analyzed the consolidated state across 20+ closed waves to identify the operational bottleneck:

**Fee persistence gaps** (documented across S428, S482, S496):
- RG-22: Futures commission unavailable from Binance API (structural `Fee="0"`)
- NB-8: Pre-S428 historical data has overloaded fee semantics (cumQuote in Fee field)
- `fee_gap` flag universally raised for Futures — no cause classification
- Net P&L for Futures round-trips overstates return by actual fee amount

**Lifecycle close edge cases** (documented across S412, S496):
- L-S496-3: Duplicate legs from improper session closure — relies on operational discipline
- L-S412-1/L-S412-2: No time-based drift detection, synthetic endurance only
- L-S495-2: Session time overlap risk at close boundary — ±5min buffer is convention, not enforced
- Session close relies on supervisor `Stopped` callback without explicit close ceremony

**Writer and runtime stability** (documented in S412):
- L-S412-6: No Futures segment endurance (Spot-only, architecturally equivalent but unproven)
- L-S412-1: Endurance window is synthetic (in-process, no real clock progression)
- L-S412-3: Batch flush lag acknowledged but unbounded by SLO

### 2. Wave Charter

Defined a 5-stage wave (S498–S502):

| Stage | Title |
|-------|-------|
| S498 | Operational Hardening Wave Charter and Scope Freeze (this stage) |
| S499 | Fee Persistence and Reconciliation Hardening |
| S500 | Lifecycle Close Edge Cases Hardening |
| S501 | Sustained Runtime and Writer Stability Proof |
| S502 | Operational Hardening Evidence Gate |

### 3. Capabilities Registered

- **5 MUST**: Futures fee retrieval (C-OH1), fee reconciliation tightening (C-OH3), explicit session close (C-OH4), duplicate leg prevention (C-OH5), Futures endurance (C-OH7)
- **4 SHOULD**: Historical fee normalization (C-OH2), boundary timestamp guard (C-OH6), wall-clock stability (C-OH8), batch flush SLO (C-OH9)

### 4. Governing Questions Defined

5 governing questions (Q-OH1 through Q-OH5), all must answer YES for FULL PASS.

### 5. Non-Goals Frozen

10 explicit non-goals (NG-1 through NG-10) preventing scope inflation into OMS expansion, multi-exchange, observability platforms, strategy families, structural redesign, portfolio/risk engines, dashboards, streaming analytics, cross-binary health aggregation, or write-path schema changes beyond fee fields.

### 6. Guard Rails Set

8 guard rails (GR-1 through GR-8) constraining the wave: no new infrastructure, no schema changes except fee normalization, no new HTTP endpoints, no NATS subject changes, no event envelope changes, independent stage closure, no scope addition without new charter, maximum 5 stages.

---

## Deliverables Produced

| Artifact | Path |
|----------|------|
| Wave Charter and Scope Freeze | `docs/architecture/operational-hardening-wave-charter-and-scope-freeze.md` |
| Capabilities, Questions, and Non-Goals | `docs/architecture/operational-hardening-capabilities-questions-and-non-goals.md` |
| Stage Report | `docs/stages/stage-s498-operational-hardening-charter-report.md` |

---

## Risk Register

| R-ID | Risk | Likelihood | Impact | Mitigation |
|------|------|-----------|--------|------------|
| R-1 | Futures fee retrieval adds latency or rate-limit pressure | MEDIUM | MEDIUM | Async enrichment; estimation fallback |
| R-2 | Historical fee normalization may require migration | LOW | MEDIUM | Query-time only; GR-2 prevents schema changes |
| R-3 | Session close hardening may uncover undocumented edge cases | LOW | LOW | 200-cycle baseline exists |
| R-4 | Futures endurance may reveal segment-specific instability | LOW | MEDIUM | Architecture is segment-agnostic by design |

---

## Dependencies

### Requires (all met)

- Fee normalization canonical model (S428) — delivered
- Writer stability baseline (S412) — 200 cycles, zero failures
- Session lifecycle model (execute_supervisor) — open/close/error states
- Reconciliation framework (S482, S496) — flags operational
- Cross-session pairing (S493–S497) — boundary handling proven

### Does NOT require

- New API keys
- New live trading session
- New venue connectivity
- New infrastructure provisioning

---

## Next Stage

**S499 — Fee Persistence and Reconciliation Hardening**

Scope: Investigate and implement best-achievable Futures fee accuracy, historical fee normalization, and fee reconciliation rule tightening. Delivers capabilities C-OH1, C-OH2, and C-OH3.

Preparation:
- Read Binance Futures API documentation for `/fapi/v1/userTrades` response shape
- Inventory current fee-related code paths in `internal/domain/execution/` and `internal/application/execution/`
- Review reconciliation rule implementation in `internal/domain/pairing/` and `internal/domain/effectiveness/`
- Determine whether post-fill async enrichment is viable within existing adapter architecture

---

## References

- [Wave Charter](../architecture/operational-hardening-wave-charter-and-scope-freeze.md)
- [Capabilities and Non-Goals](../architecture/operational-hardening-capabilities-questions-and-non-goals.md)
- [S497 Evidence Gate](stage-s497-cross-session-continuity-evidence-gate-report.md)
- [Fee Semantics](../architecture/fees-commission-assets-cross-segment-semantics-and-limitations.md)
- [Writer Stability](../architecture/sustained-execution-state-consistency-writer-stability-and-limitations.md)
- [Fills and Fee Reconciliation](../architecture/fills-fees-pairing-result-reconciliation-semantics-and-limitations.md)
