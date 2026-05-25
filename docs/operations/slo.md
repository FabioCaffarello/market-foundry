# SLO / SLI — Service Level Objectives

**Status:** Active — all four SLOs in `Observing` (PROGRAM-0003 H-5)
**Date:** 2026-05-25
**Owner:** Repository maintainer
**Authority tier:** T1 (definitions) / T2 (concrete targets once
`Committed`) — see [`../AUTHORITY.md`](../AUTHORITY.md)
**Relates to:** [`runtime-invariants.md`](runtime-invariants.md),
[`observability.md`](observability.md),
[`../decisions/0024-metrics-policy.md`](../decisions/0024-metrics-policy.md),
[`../decisions/0025-alerting-strategy.md`](../decisions/0025-alerting-strategy.md),
[`../ARCHITECTURE.md`](../ARCHITECTURE.md) → "Data flow",
[`../TRUTH-MAP.md`](../TRUTH-MAP.md)

---

## SLO status taxonomy (per ADR-0025 AS-1)

Every SLO declared below carries one of three lifecycle states:

- **`Proposed`** — target declared, no measurement infrastructure.
  No alerts active. (Pre-PROGRAM-0003 state.)
- **`Observing`** — measurement infrastructure deployed
  (Prometheus + Grafana per PROGRAM-0003 H-5); target declared
  but not yet validated against observed baseline. Alerts
  active with severity capped at `ticket` per ADR-0025 (no
  paging on unvalidated targets). Promotion to `Committed`
  requires 7 contiguous days of compliance against the
  proposed target.
- **`Committed`** — target validated against baseline; foundry
  actively defends the target. Burn-rate alerts pages on-call
  for fast burn, tickets for slow burn.

**Current state (post-PROGRAM-0003 H-5)**: F1–F4 all `Observing`.
None `Committed` yet — promotion is a future-onda decision per
SLO individually after baseline collection.

---

## Purpose

Define the **service-level objectives** that the foundry's four
critical operational flows are measured against, the measurement
infrastructure backing each, and the lifecycle state of each
target.

Pre-PROGRAM-0003, this document was a template — F1–F4 declared
with proposed targets and explicit `Status: Not yet measured`.
PROGRAM-0003 H-5 installed the Prometheus + Grafana stack, the
recording rules computing burn rates, and the burn-rate alerts
firing at `ticket` severity per ADR-0025. The four SLOs are
now `Observing`.

---

## Measurement infrastructure

- **Prometheus** scrapes the 7 long-running binaries at 15s
  intervals; see `deploy/observability/prometheus/prometheus.yml`.
- **Recording rules** compute `error_ratio` per window (5m / 30m
  / 1h / 6h) and `burn_rate` (error_ratio / error_budget) per SLO;
  see `deploy/observability/prometheus/recording.rules.yml`.
- **Alert rules** fire burn-rate alerts per ADR-0025 AS-2
  (multi-window multi-burn-rate); see
  `deploy/observability/prometheus/alerts.rules.yml`. All four
  SLOs are `Observing` → all eight burn-rate alerts carry
  `severity: ticket` per ADR-0025 AS-3.
- **Grafana** provisions dashboards under
  `deploy/observability/grafana/dashboards/`; the SLO burn-rate
  panel per dashboard shows the four windows with thresholds at
  6 (yellow) and 14.4 (red).
- **Bring-up**: `make obs-up` (opt-in compose profile per
  PROGRAM-0003).

See [`observability.md`](observability.md) for operator workflow.

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
| **SLI** | `successful_publish_ratio = ok / (ok + parse_error + publish_error)` over a 5-minute rolling window. |
| **SLO target** | **99.5%** over rolling 30 days. |
| **Error budget** | **0.5%** per 30 days. |
| **Measurement** | Recording rule `slo:ingest:error_ratio_<window>` computed from `marketfoundry_ingest_messages_total{status}` per window. Burn rate: `slo:ingest:burn_rate_<window> = error_ratio / 0.005`. |
| **Status** | **Observing**. Recording rules + burn-rate alerts active per ADR-0025; ingest binary does not yet emit `marketfoundry_ingest_messages_total` (canonical metric name reserved). Until the counter is wired, recording rules produce empty series and alerts do not fire — by design. Counter wiring is a follow-up onda; promotion to `Committed` blocked on baseline collection of 7 contiguous days. |
| **Burn-rate alerts** | `SLOIngestBurnRateFast` (5m AND 1h > 14.4× for 2m, ticket). `SLOIngestBurnRateSlow` (30m AND 6h > 6× for 5m, ticket). See `alerts.rules.yml`. |
| **Rollback signal** | Sustained ratio below 99.0% over an hour → investigate WebSocket reconnect behaviour, JetStream publish error rate, schema drift between Binance and the foundry's parser. |

---

### F2 — Derive latency (observation → first downstream event)

**Description.** Time from `OBSERVATION_EVENTS` consumption to the
first emission of an evidence / signal / decision / strategy /
risk event for the same partition. F2 measures **how fast the
derivation pipeline reacts to an observation**.

| Field | Value |
|---|---|
| **SLI** | Share of `marketfoundry_consumer_processing_duration_seconds` samples (filtered by `consumer=~"derive.*"`) below the 500ms bucket boundary. |
| **SLO target** | **p99 < 500ms** over rolling 30 days. |
| **Error budget** | 1% of windows may breach. |
| **Measurement** | Recording rule `slo:derive_latency:error_ratio_<window>` computed from `marketfoundry_consumer_processing_duration_seconds_bucket{consumer=~"derive.*", le="0.5"}` over `_count`. Burn rate: `slo:derive_latency:burn_rate_<window> = error_ratio / 0.01`. |
| **Status** | **Observing**. Counter `marketfoundry_consumer_processing_duration_seconds` is emitted today by consumer actors; recording rules and burn-rate alerts are active. Baseline collection starts on first `make obs-up` of the running stack. Promotion to `Committed` blocked on 7 contiguous days of compliance. |
| **Burn-rate alerts** | `SLODeriveLatencyBurnRateFast` (5m AND 1h > 14.4× for 2m, ticket). `SLODeriveLatencyBurnRateSlow` (30m AND 6h > 6× for 5m, ticket). |
| **Rollback signal** | p99 sustained > 1s → investigate actor mailbox depths, JetStream consumer ack lag, or specific family processor regressions. |

---

### F3 — Gateway HTTP read latency

**Description.** End-to-end latency of a gateway HTTP read (e.g.,
`GET /signal/rsi/:source/:symbol/:timeframe`) including the
internal NATS request/reply round-trip to `store`. F3 measures
**how fast the operational read surface responds to clients**.

| Field | Value |
|---|---|
| **SLI** | Share of `marketfoundry_http_request_duration_seconds` samples (filtered by `method="GET"`) below the 200ms bucket boundary. |
| **SLO target** | **p99 < 200ms** over rolling 30 days. |
| **Error budget** | 1% of windows may breach. |
| **Measurement** | Recording rule `slo:store_read_latency:error_ratio_<window>` computed from `marketfoundry_http_request_duration_seconds_bucket{method="GET", le="0.2"}` over `_count`. Burn rate: `slo:store_read_latency:burn_rate_<window> = error_ratio / 0.01`. |
| **Status** | **Observing**. Counter `marketfoundry_http_request_duration_seconds` is emitted today by every HTTP-instrumented handler in the gateway; recording rules and burn-rate alerts are active. Baseline collection starts on first `make obs-up` of the running stack. Promotion to `Committed` blocked on 7 contiguous days. |
| **Burn-rate alerts** | `SLOStoreReadLatencyBurnRateFast` (5m AND 1h > 14.4× for 2m, ticket). `SLOStoreReadLatencyBurnRateSlow` (30m AND 6h > 6× for 5m, ticket). |
| **Rollback signal** | p99 sustained > 500ms → investigate store responder mailbox depth, NATS request/reply timeout configuration, or KV bucket read patterns. |

---

### F4 — Analytical write success ratio (NATS → ClickHouse)

**Description.** Share of domain events published to a stream that
the `writer` binary successfully persists in ClickHouse within
5 seconds of the publish timestamp. F4 measures **the data-flow
integrity of the analytical persistence path**.

| Field | Value |
|---|---|
| **SLI** | `successful_persist_ratio = persisted_within_5s / total_published`, evaluated over rolling 30 days. |
| **SLO target** | **99.9%** over rolling 30 days. |
| **Error budget** | **0.1%** per 30 days. |
| **Measurement** | Recording rule `slo:writer_persist:error_ratio_<window>` computed from `marketfoundry_writer_persist_total{status}` per window. Burn rate: `slo:writer_persist:burn_rate_<window> = error_ratio / 0.001`. |
| **Status** | **Observing**. Recording rules + burn-rate alerts active; writer binary does not yet emit `marketfoundry_writer_persist_total` (canonical name reserved). Until the counter is wired, rules produce empty series and alerts do not fire — by design. Counter wiring is a follow-up onda; promotion to `Committed` blocked on baseline collection. |
| **Burn-rate alerts** | `SLOWriterPersistBurnRateFast` (5m AND 1h > 14.4× for 2m, ticket). `SLOWriterPersistBurnRateSlow` (30m AND 6h > 6× for 5m, ticket). |
| **Rollback signal** | Ratio sustained below 99.5% → investigate ClickHouse availability, `writer` consumer ack lag, batch flush configuration, or migration drift. |

---

## SLO targets summary

| Flow | SLI | SLO target | Error budget (30d) | Status |
|---|---|---|---|---|
| F1 — Ingest | publish success ratio | 99.5% | 0.5% | Observing |
| F2 — Derive | p99 latency observation→downstream | < 500ms | 1% windows | Observing |
| F3 — Store read | p99 latency gateway GET | < 200ms | 1% windows | Observing |
| F4 — Writer | persist within 5s ratio | 99.9% | 0.1% | Observing |

All four SLOs flipped from `Proposed` to `Observing` in
PROGRAM-0003 H-5. None are `Committed` yet — promotion is
per-SLO, requires 7 contiguous days of compliance per ADR-0025
AS-1, and is a future-onda decision.

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

## How to promote an SLO from `Observing` to `Committed`

1. **Capture baseline.** Let the stack run against representative
   traffic for at least **7 contiguous days** with the SLI
   continuously producing non-empty series. Confirm the recording
   rules emit values and the burn-rate alerts have not fired at
   ticket severity for 7 days.
2. **Validate target against observed compliance.**
   - If observed compliance ≥ target (e.g., observed 99.6% for a
     99.5% target), the SLO is **promotion-eligible**.
   - If observed compliance < target (e.g., observed 98.7% for
     99.5%), revise the target downward OR invest in fixing the
     underlying cause. Either action is honest; an unattainable
     SLO is worse than a realistic one.
3. **Update the row's `Status` field** in this file to
   `Committed`. Record the promotion in the Changelog with the
   observed baseline value.
4. **Update `alerts.rules.yml`**: flip the fast-burn alert for
   the promoted SLO from `severity: ticket` to `severity: page`.
   The slow-burn stays at `severity: ticket` (per ADR-0025 AS-3).
5. **Update [`../AUTHORITY.md`](../AUTHORITY.md)**: promote the
   row's *target values* from T1-template to T2 (the *definitions*
   stay T1; the *commitments* are operational truth once the
   pager is on the line).

Once a runbook is written for the SLO (in
`docs/operations/runbooks/<flow>.md`), the alert annotation
`runbook_url` points at it. Runbook files themselves are a
follow-up onda; stubs ship with annotations referencing them
from H-5 onwards.

---

## Changelog

- **2026-05-24** — Initial version, shipped as H-1 deliverable.
  F1–F4 critical flows declared; SLI structure defined; SLO
  targets marked **proposed** pending H-5 measurement. No
  alerting wired (deliberate). Out-of-scope items enumerated so
  future readers see deliberate omission rather than oversight.
- **2026-05-25** — **PROGRAM-0003 H-5 closure**: F1–F4 all flip
  `Proposed` → `Observing`. Status taxonomy formalized per
  ADR-0025 AS-1 (Proposed / Observing / Committed). Measurement
  infrastructure section added (recording rules + burn-rate
  alerts + Grafana dashboards). Per-SLO `Status` field updated
  with explicit measurement source (recording rule name,
  underlying counter) and current Observing reasoning (counter
  wired vs. canonical-name-reserved). Alerts at `ticket`
  severity per ADR-0025 AS-3 until each SLO promotes to
  Committed; promotion requires 7 contiguous days of baseline
  compliance.
