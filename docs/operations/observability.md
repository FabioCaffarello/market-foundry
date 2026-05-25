# Observability — operator guide

**Status:** Active
**Date:** 2026-05-25
**Owner:** Repository maintainer
**Authority tier:** T2 — Operational
([`../AUTHORITY.md`](../AUTHORITY.md))
**Relates to:**
[`../programs/PROGRAM-0003-observability.md`](../programs/PROGRAM-0003-observability.md),
[`../decisions/0024-metrics-policy.md`](../decisions/0024-metrics-policy.md),
[`../decisions/0025-alerting-strategy.md`](../decisions/0025-alerting-strategy.md),
[`slo.md`](slo.md), [`runtime-invariants.md`](runtime-invariants.md)

---

## Purpose

How to bring the observability stack up, where to look at what,
and how to extend it when adding new metrics or dashboards. This
is the operator entry point for the PROGRAM-0003 surface.

The architectural decisions live in
[ADR-0024](../decisions/0024-metrics-policy.md) (metrics policy)
and [ADR-0025](../decisions/0025-alerting-strategy.md) (alerting
strategy). The SLO definitions live in [`slo.md`](slo.md). This
document covers *operation*, not *policy*.

---

## Quick start

```bash
make obs-up         # bring up prometheus + grafana
open http://127.0.0.1:9090     # prometheus (no auth)
open http://127.0.0.1:3000     # grafana (admin / admin)
make obs-down       # stop the stack (volumes persist)
make obs-reload     # hot-reload prometheus config
```

The observability stack is an **opt-in compose profile**. It does
not come up under `make up`; the dev workflow keeps the baseline
stack lean by default and brings observability up when needed.

---

## Architecture overview

```
┌─────────────────────────────────────────────────────────────────┐
│  Foundry binaries (7 long-running)                              │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐    │
│  │ ingest  │ │ derive  │ │  store  │ │ gateway │ │ execute │    │
│  └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘    │
│       │           │           │           │           │         │
│  Each binary serves /metrics on its HTTP port.                  │
│  (configctl + writer also serve /metrics, omitted for space.)   │
└───────┴───────────┴───────────┴───────────┴───────────┴─────────┘
                                │ scrape every 15s
                                ▼
                       ┌─────────────────┐
                       │   prometheus    │ :9090
                       │   - scrape      │
                       │   - recording   │ slo:<flow>:burn_rate_<window>
                       │   - alert       │ SLO* / Runtime* alerts
                       └────────┬────────┘
                                │ query
                                ▼
                       ┌─────────────────┐
                       │     grafana     │ :3000
                       │   5 dashboards  │ ingest/derive/store/gateway/determinism
                       └─────────────────┘
```

### Where /metrics lives per binary

| Binary | /metrics on | Mechanism |
|---|---|---|
| `configctl` | `:8080` | `healthz.NewHealthServer` auto-route |
| `derive` | `:8083` | `healthz.NewHealthServer` auto-route |
| `execute` | `:8084` | `healthz.NewHealthServer` auto-route |
| `gateway` | `:8080` | `routes.DefaultRoutes(deps)` → `routes/core.go:364` |
| `ingest` | `:8082` | `healthz.NewHealthServer` auto-route |
| `store` | `:8081` | `healthz.NewHealthServer` auto-route |
| `writer` | `:8085` | `healthz.NewHealthServer` auto-route |

`migrate` is a one-shot CLI — no /metrics, listed in
`tools/raccoon-cli/policies/binaries.toml` under `one_shot`.

The invariant "every long-running binary exposes /metrics" is
enforced statically by `make metrics-check` (raccoon-cli
`check metrics` analyzer, Step 8 of the quality gate).

---

## Provisioned dashboards (Grafana → Observability folder)

| Dashboard | UID | Focus |
|---|---|---|
| **Ingest Health** | `mf-ingest-health` | Observation event throughput per venue, F1 publish ratio, ingest consumer lag, F1 burn-rate |
| **Derive Health** | `mf-derive-health` | Derive consumer throughput, F2 p99 latency vs 500ms target, derive consumer lag, F2 burn-rate |
| **Store Health** | `mf-store-health` | Store consumer throughput, lag, store binary goroutine count, heap MB |
| **Gateway Health** | `mf-gateway-health` | HTTP request rate by method, F3 GET p99 latency vs 200ms target, error rate by status code, top routes table, F3 burn-rate |
| **Determinism Health** | `mf-determinism-health` | Sequencer gap rate per (venue, event_type), total gaps 1h, gate read failures per reason, execution gate active gauge |

All dashboards default to a 1-hour time range with 30s refresh.
Burn-rate panels carry threshold lines at 6 (yellow, slow-burn)
and 14.4 (red, fast-burn) per ADR-0025 AS-2.

---

## Alerts (read-only summary)

Defined in
`deploy/observability/prometheus/alerts.rules.yml`. The full set:

### SLO burn-rate alerts (severity: `ticket` — Observing)

Per ADR-0025 AS-1, all four SLOs (F1–F4) are currently
`Observing`. Burn-rate alerts fire at `ticket` severity until
each SLO individually promotes to `Committed`. Fast burn
expression: `burn_rate_5m > 14.4 AND burn_rate_1h > 14.4 for 2m`;
slow burn: `burn_rate_30m > 6 AND burn_rate_6h > 6 for 5m`.

- `SLOIngestBurnRateFast` / `SLOIngestBurnRateSlow`
- `SLODeriveLatencyBurnRateFast` / `SLODeriveLatencyBurnRateSlow`
- `SLOStoreReadLatencyBurnRateFast` / `SLOStoreReadLatencyBurnRateSlow`
- `SLOWriterPersistBurnRateFast` / `SLOWriterPersistBurnRateSlow`

### Runtime-safety alerts (per ADR-0025 AS-6)

Defend invariants whose breakage causes data loss regardless of
any SLO target. The `slo` and `flow` labels are absent;
`category: runtime-safety` instead.

- `ConsumerStallNoProgress` (page) — consumer lag growing with
  zero ack rate for 5+ min.
- `SeqGapRateNonZero` (ticket) — sequencer gap counter
  incrementing for 15+ min per (venue, event_type).
- `GateReadFailureRateHigh` (ticket) — gate read failures
  incrementing for 15+ min (ADR-0012 fail-open observability).
- `ProcessGoroutinesHigh` (ticket) — `process_goroutines > 10000`
  for 5+ min.
- `ProcessHeapAllocHigh` (ticket) — `go_memstats_heap_alloc_bytes
  > 500 MB` for 5+ min.

Routing is currently `null` (logs to Prometheus self). Paging
integration is a future-phase concern per ADR-0025 non-goals.

---

## Common operator workflows

### Reading a fired alert

1. Find the alert in Prometheus UI (`/alerts`) or the dashboard's
   burn-rate panel.
2. Read the `description` annotation — it includes the SLO
   identifier, the observed values, and the threshold.
3. Follow `runbook_url` if a runbook exists; otherwise the
   alert's `description` is the operator's primary diagnostic.
4. Correlate with structured logs per ADR-0024 MP-5 log
   compensation pattern. Example for sequencer gaps:
   ```bash
   docker compose logs --tail=200 derive | grep sequencer.gap_detected
   ```
   The log record carries the high-cardinality `instrument`
   dimension that the metric label intentionally omits.

### Adding a new metric

1. Declare the counter / histogram / gauge in
   `internal/shared/metrics/` following ADR-0024 MP-1 (naming)
   and MP-2 (label budget).
2. Register in the package `init()`.
3. Export a helper function (`IncFoo`, `ObserveFoo`) so callers
   don't take a direct prometheus dependency.
4. Add a unit test in the same package asserting the metric
   appears on `/metrics`.
5. If the metric drives a new dashboard panel, edit the relevant
   JSON under `deploy/observability/grafana/dashboards/` and
   `make obs-reload` (Grafana picks up new dashboards on its
   provisioning interval; restart the grafana container to
   force).
6. If the metric drives a new SLO, follow the slo.md "How to
   evolve" section.

### Adding a new binary

The `check metrics` analyzer enforces "every long-running
cmd/*/main.go exposes /metrics". When adding a new binary:

- If it uses `healthz.NewHealthServer` → /metrics is auto-routed,
  nothing else to do.
- If it has its own HTTP server with custom routes → register
  `metrics.HandlerFunc()` or `mux.Handle("GET /metrics", ...)`
  in the binary's package.
- If /metrics is delegated to an imported package (the gateway
  pattern) → add the binary name to
  `tools/raccoon-cli/policies/binaries.toml` under
  `transitive_registration` with a comment pointing at the
  registration site.
- If genuinely one-shot (no HTTP, no long-running process) → add
  the binary name under `one_shot`.

Also: add the new binary to the prometheus scrape config in
`deploy/observability/prometheus/prometheus.yml` with its port.

### Hot-reloading Prometheus config

```bash
# Edit prometheus.yml / recording.rules.yml / alerts.rules.yml.
# Then:
make obs-reload
```

Hot-reload uses the `/-/reload` endpoint enabled in the
prometheus container's command line. Grafana picks up dashboard
JSON changes from its filesystem provisioning at a fixed interval
(default 10s); to force, restart the grafana container.

### Validating rule changes before applying

```bash
docker run --rm -v "$PWD/deploy/observability/prometheus:/etc/prometheus:ro" \
  --entrypoint promtool prom/prometheus:v2.54.1 \
  check rules /etc/prometheus/recording.rules.yml /etc/prometheus/alerts.rules.yml
```

This is the same `promtool check rules` that PROGRAM-0003 H-5
ran before each commit. Adopt as a pre-commit habit when editing
rules.

---

## Layout

```
deploy/observability/
├── prometheus/
│   ├── prometheus.yml          # scrape config + global settings
│   ├── recording.rules.yml     # SLO error_ratio + burn_rate per window
│   └── alerts.rules.yml        # SLO burn-rate + runtime-safety alerts
└── grafana/
    ├── provisioning/
    │   ├── datasources/datasources.yml   # Prometheus datasource (uid: marketfoundry-prometheus)
    │   └── dashboards/dashboards.yml     # filesystem provider config
    └── dashboards/
        ├── ingest-health.json
        ├── derive-health.json
        ├── store-health.json
        ├── gateway-health.json
        └── determinism-health.json
```

`docker-compose.yaml` has prometheus + grafana under
`profiles: ["observability"]`. The Makefile wrappers
(`obs-up` / `obs-down` / `obs-reload`) live in `##@ Observability`.

The `marketfoundry-prometheus` datasource UID is declared in
`datasources.yml`; dashboard panels reference it
deterministically. Changing the UID breaks provisioned dashboards
— treat as a stable identifier.

---

## Persistence

- `market-foundry-prometheus-data` (named volume) — prometheus
  TSDB; retains 30 days by default
  (`--storage.tsdb.retention.time=30d` in the compose service
  command).
- `market-foundry-grafana-data` (named volume) — grafana
  user-created dashboards, snapshots, plugin state. The
  provisioned dashboards under `dashboards/` re-load on every
  startup regardless.

`make obs-down` preserves both volumes. To reset:

```bash
docker compose -f deploy/compose/docker-compose.yaml --profile observability down -v
```

`-v` removes the named volumes. Use deliberately.

---

## Known limitations (PROGRAM-0003 non-goals)

The following are explicit non-goals of PROGRAM-0003 and remain
absent until a future phase:

- **Distributed tracing** (OpenTelemetry, Jaeger). Cross-binary
  spans, exemplars linking metrics to trace IDs.
- **Log aggregation** (Loki, OpenSearch). Operators currently use
  `docker compose logs` for the log compensation pattern; the
  pattern is documented in ADR-0024 MP-5.
- **Paging integration** (PagerDuty, Opsgenie). Alerts carry
  `severity: page|ticket` labels but no Alertmanager routing
  destination configured.
- **Per-instrument labels.** Per ADR-0024 MP-2, the `instrument`
  dimension is omitted from labels and compensated via
  structured logs.
- **Incident-management workflow.** No `incident-log.md` /
  ticketing-system integration yet.

These land in successor phases when single-operator phase ends
and multi-operator on-call rotation is operationally
justified.

---

## Changelog

- **2026-05-25** — Initial version, shipped as PROGRAM-0003 H-5
  deliverable. Covers compose profile, dashboards, alerts,
  common workflows, and known limitations.
