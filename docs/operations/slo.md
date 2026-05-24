# SLO / SLI — Service Level Objectives

**Status:** Active (template — targets not yet measured)
**Date:** 2026-05-24
**Owner:** Repository maintainer
**Authority tier:** T1 (definitions) / T2 (concrete targets once
measured) — see [`../AUTHORITY.md`](../AUTHORITY.md)
**Relates to:** [`runtime-invariants.md`](runtime-invariants.md),
[`../ARCHITECTURE.md`](../ARCHITECTURE.md) → "Data flow",
[`../TRUTH-MAP.md`](../TRUTH-MAP.md)

---

## Purpose

Define the **service-level objectives** that the foundry's four
critical operational flows will be measured against once
automated observability lands (H-5 per
[PROGRAM-0001](../programs/PROGRAM-0001-foundation.md) trajectory,
not before).

This document is **deliberately a template**. The flows, the SLIs
(Service Level Indicators), and the structure are stable now;
the **SLO targets** below are **proposals to validate against
baseline data once measurement is automated**. Until then, the
foundry runs without active error-budget tracking and without
alerting — a deliberate choice consistent with the single-operator
phase.

---

## Why no alerting now

- **No Prometheus yet.** The foundry exposes counters and
  histograms through Prometheus client libraries (
  `internal/shared/metrics/`), but the scrape and alerting stack
  is not deployed in the local default and is scoped for a future
  onda (likely H-5).
- **No baseline data.** Targets like "p99 read latency < 200ms"
  must be validated against observed reality; quoting them as
  binding before measurement risks both **false alarms** (target
  too tight) and **false complacency** (target too loose).
- **Single-operator phase.** Page-style alerting is operationally
  premature; smoke-test green is the operational signal that
  matters today.

When H-5 (or a later onda) installs Prometheus + alerting, the
targets below are the starting set to validate. Targets confirmed
against baseline data flip from `Proposed` to `Committed`; targets
contradicted by data are revised.

---

## Critical flows (F1–F4)

The foundry has four operational flows where a degradation has
end-to-end impact on system value. Each gets its own SLO entry.

```
                      ┌─────────────┐
        Binance WS →  │  F1 Ingest  │ → OBSERVATION_EVENTS
                      └──────┬──────┘
                             │
                      ┌──────▼──────┐
                      │  F2 Derive  │ → evidence / signal /
                      └──────┬──────┘    decision / strategy /
                             │            risk / execution
                  ┌──────────┼──────────┐
                  ▼                     ▼
            ┌──────────┐         ┌──────────────┐
            │ F3 Store │         │ F4 Writer    │
            │   read   │         │ → ClickHouse │
            └──────────┘         └──────────────┘
```

---

### F1 — Ingest success ratio

**Description.** Binance WebSocket message → published
`OBSERVATION_EVENTS` JetStream message. F1 measures the **share
of incoming exchange messages that the foundry successfully
parses and publishes**, end-to-end.

| Field | Value |
|---|---|
| **SLI** | `successful_publish_ratio = ok / (ok + parse_error + publish_error)` over a 5-minute rolling window, evaluated against a 30-day window. |
| **SLO (proposed)** | **99.5%** over rolling 30 days. |
| **Error budget (proposed)** | **0.5%** per 30 days. |
| **Measurement (planned)** | PromQL against `marketfoundry_ingest_messages_total{status}` once exposed; sum-rate over the 5m window. |
| **Status** | **Not yet measured.** Counters exist in `internal/shared/metrics/`; Prometheus stack does not yet scrape them in the local default. Target to be validated against baseline once H-5 lands. |
| **Burn-rate alerts (planned)** | Fast: 5m/1h burn > 14.4× → page (after Prometheus). Slow: 30m/6h burn > 6× → ticket. |
| **Rollback signal** | Sustained ratio below 99.0% over an hour → investigate WebSocket reconnect behaviour, JetStream publish error rate, schema drift between Binance and the foundry's parser. |

---

### F2 — Derive latency (observation → first downstream event)

**Description.** Time from `OBSERVATION_EVENTS` consumption to the
first emission of an evidence / signal / decision / strategy /
risk event for the same partition. F2 measures **how fast the
derivation pipeline reacts to an observation**.

| Field | Value |
|---|---|
| **SLI** | p99 of `derive_latency_ms` (observation receipt → first downstream publish on the same partition) over a 5-minute window. |
| **SLO (proposed)** | **p99 < 500ms** over rolling 30 days. |
| **Error budget (proposed)** | 1% of windows may breach (i.e., p99 ≥ 500ms in any 5m window). |
| **Measurement (planned)** | Histogram `marketfoundry_derive_processing_seconds_bucket` once exposed; quantile evaluation at 0.99. |
| **Status** | **Not yet measured.** Per-actor processing histograms exist via Hollywood instrumentation; aggregation to derive-level latency is the H-5 work. Target to validate. |
| **Burn-rate alerts (planned)** | Fast: 5m/1h burn > 14.4× → page. Slow: 30m/6h burn > 6× → ticket. |
| **Rollback signal** | p99 sustained > 1s → investigate actor mailbox depths, JetStream consumer ack lag, or specific family processor regressions. |

---

### F3 — Gateway HTTP read latency

**Description.** End-to-end latency of a gateway HTTP read (e.g.,
`GET /signal/rsi/:source/:symbol/:timeframe`) including the
internal NATS request/reply round-trip to `store`. F3 measures
**how fast the operational read surface responds to clients**.

| Field | Value |
|---|---|
| **SLI** | p99 of `gateway_http_request_duration_seconds` for `GET` routes, over a 5-minute window. |
| **SLO (proposed)** | **p99 < 200ms** over rolling 30 days. |
| **Error budget (proposed)** | 1% of windows may breach. |
| **Measurement (planned)** | Histogram `marketfoundry_gateway_http_request_duration_seconds_bucket` once exposed; quantile evaluation at 0.99 with `method=GET` label filter. |
| **Status** | **Not yet measured.** Gateway request handlers are instrumented; aggregation to per-route quantiles is H-5 work. Target to validate. |
| **Burn-rate alerts (planned)** | Fast: 5m/1h burn > 14.4× → page. Slow: 30m/6h burn > 6× → ticket. |
| **Rollback signal** | p99 sustained > 500ms → investigate store responder mailbox depth, NATS request/reply timeout configuration, or KV bucket read patterns. |

---

### F4 — Analytical write success ratio (NATS → ClickHouse)

**Description.** Share of domain events published to a stream that
the `writer` binary successfully persists in ClickHouse within
5 seconds of the publish timestamp. F4 measures **the data-flow
integrity of the analytical persistence path**.

| Field | Value |
|---|---|
| **SLI** | `successful_persist_ratio = persisted_within_5s / total_published` per stream, evaluated over rolling 30 days. |
| **SLO (proposed)** | **99.9%** over rolling 30 days. |
| **Error budget (proposed)** | **0.1%** per 30 days. |
| **Measurement (planned)** | PromQL against `marketfoundry_writer_persist_total{status, stream}` and `marketfoundry_writer_persist_lag_seconds_bucket`; sum-rate of `status="ok" AND lag<=5s`. |
| **Status** | **Not yet measured.** `writer` instruments persist counts and ClickHouse insert outcomes; lag-bucket histogram is H-5 work. Target to validate. |
| **Burn-rate alerts (planned)** | Fast: 5m/1h burn > 14.4× → page. Slow: 30m/6h burn > 6× → ticket. |
| **Rollback signal** | Ratio sustained below 99.5% → investigate ClickHouse availability, `writer` consumer ack lag, batch flush configuration, or migration drift. |

---

## SLO targets summary

| Flow | SLI | SLO target (proposed) | Error budget (30d) |
|---|---|---|---|
| F1 — Ingest | publish success ratio | 99.5% | 0.5% |
| F2 — Derive | p99 latency observation→downstream | < 500ms | 1% windows |
| F3 — Store read | p99 latency gateway GET | < 200ms | 1% windows |
| F4 — Writer | persist within 5s ratio | 99.9% | 0.1% |

All targets are **proposed**. They become **committed** only after
baseline measurement validates them or the maintainer revises them
based on data.

---

## Out of scope (deliberate)

- **Per-strategy PnL SLOs.** Effectiveness classifies round-trips;
  there is no aggregator yet (N2 in [`../RESUMPTION.md`](../RESUMPTION.md)).
- **Cross-session risk SLOs.** No portfolio-level position tracking
  (N3 in RESUMPTION). Risk SLOs would require cross-session
  aggregate state that does not exist.
- **WebSocket delivery SLOs.** The foundry currently exposes only
  HTTP (no client WS delivery surface). When the Odin client and
  WS delivery arrive (H-11+), a delivery-latency SLO joins this
  document; until then, omitted.
- **Multi-venue parity SLOs.** Single venue family (Binance Spot +
  Futures). Multi-venue is a future onda; SLOs for cross-venue
  parity belong with that work.
- **Heatmap / volume-profile / candle delivery SLOs.** Capabilities
  catalogued in the raccoon (per ADR-0016) but not yet implemented
  in the foundry; no SLO is meaningful until the capability ships.

---

## How to evolve this document

When automated measurement lands (H-5 or successor):

1. **Capture baseline.** Run the SLI for at least one week of
   representative traffic; compute observed p99 / ratio.
2. **Compare to proposed target.** If observed ≥ proposed target,
   the proposed target is conservative — flip to **Committed**.
3. **If observed < proposed target,** revise the target downward
   (or invest in fixing the underlying cause). Either action is
   honest; an unattainable SLO is worse than a realistic one.
4. **Update the row's `Status` field** to `Committed` (or to
   `Revised — target was X, observed Y, new target Z`).
5. **Promote the row from T1-template to T2** in
   [`../AUTHORITY.md`](../AUTHORITY.md)'s file-to-tier inventory
   (the *definitions* stay T1; the *target values* become T2 once
   measured and operationally live).

Once alerting is wired (burn-rate based), the alert routes go in
`docs/operations/troubleshooting.md` or a dedicated runbook —
not here. This file stays the SLO *definition*; runbooks own the
*response*.

---

## Changelog

- **2026-05-24** — Initial version, shipped as H-1 deliverable.
  F1–F4 critical flows declared; SLI structure defined; SLO
  targets marked **proposed** pending H-5 measurement. No
  alerting wired (deliberate). Out-of-scope items enumerated so
  future readers see deliberate omission rather than oversight.
