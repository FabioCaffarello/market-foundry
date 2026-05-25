# ADR 0025: Alerting strategy — SLO taxonomy, burn-rate windows, severity tiers

## Status

Proposed. Delivered in Onda H-5 of PROGRAM-0003 (Fase
Observability); promoted to `Accepted` when H-5 ships the
implementing artifacts (`deploy/observability/prometheus/alerts.rules.yml`,
`docs/operations/slo.md` flipped from template to `Observing`,
recording rules under the same path). See "Promoção para Accepted"
below.

## Date

2026-05-25.

## Context

`market-foundry`'s [`docs/operations/slo.md`](../operations/slo.md)
(template, shipped in PROGRAM-0001 H-1) declares four critical
operational flows with proposed SLO targets:

| Flow | SLI | SLO (proposed) | Error budget (30d) |
|---|---|---|---|
| F1 — Ingest | publish success ratio | 99.5% | 0.5% |
| F2 — Derive | p99 latency observation→downstream | < 500ms | 1% windows |
| F3 — Store read | p99 latency gateway GET | < 200ms | 1% windows |
| F4 — Writer | persist within 5s ratio | 99.9% | 0.1% |

Status of each: **"Not yet measured — target to be validated
against baseline once H-5 lands."**

H-5 lands the Prometheus + Grafana stack that finally measures
these flows. But the act of installing the stack does not, by
itself, "make the SLOs real" — there is a gap between *targets
were declared in a template* and *targets are actively defended by
the on-call rotation*. The gap is the **observation phase**:
measure, baseline, validate, revise if needed, *then* commit to
defending the target.

The alerting strategy needs to support this gap. An alert against
an unvalidated SLO target is operationally dangerous in two
opposite directions:

- **Too tight** (target too aggressive for observed reality) →
  pager fatigue, operators learn to ignore.
- **Too loose** (target trivially achievable) → false comfort,
  real degradations don't fire.

The conventional answer "wait until baseline validates before
turning on alerts" loses observability during the validation
period — operators need to see when something is broken even
*while* the SLO is being baselined. The honest answer is to
*tier* the alert response: alert exists, but its **severity**
reflects how much confidence we have in the SLO target it is
defending.

The other half of the context is the **burn-rate methodology**.
Google SRE's multi-window multi-burn-rate is the established
practice; the raccoon's `docs/observability/alerting-policy.md`
implements it; the foundry adopts the same shape.

## Decision

`market-foundry` adopts the following **alerting strategy** for
all SLO-driven alerts and runtime-safety alerts:

### AS-1 — SLO status taxonomy

Every SLO declared in `docs/operations/slo.md` carries one of
three lifecycle states:

#### `Proposed`

- **Meaning**: target declared, **no measurement infrastructure
  to validate it**.
- **State as of**: PROGRAM-0001 H-1 closure; this was the only
  state available before PROGRAM-0003.
- **Alerts**: none active.

#### `Observing`

- **Meaning**: measurement infrastructure deployed (Prometheus
  scraping, recording rules computing burn rates); target is
  **declared but not yet validated** against observed baseline.
  Baseline collection in progress (typical: 7–14 days).
- **State after**: PROGRAM-0003 H-5 closure for F1–F4.
- **Alerts**: active, but **severity capped at `ticket`** — a
  fast-burn alert during `Observing` produces a ticket, not a
  page. Rationale: false positives during baseline are a
  documented possibility (target may be too tight), and paging
  the on-call for an unvalidated target erodes trust in the
  alerting system. A ticket is a forcing function for human
  review.
- **Promotion criterion**: 7 contiguous days of measurement with
  observed performance ≥ proposed target. Promotion to
  `Committed` is a deliberate decision recorded in `slo.md`'s
  changelog.

#### `Committed`

- **Meaning**: target **validated against observed baseline**;
  the foundry actively defends the target with operational
  resources.
- **State**: no SLO is `Committed` as of PROGRAM-0003 H-5
  closure; promotion is a future-onda decision per F1–F4
  individually.
- **Alerts**: active, severity per the standard tier (see AS-3).
  Fast-burn pages, slow-burn tickets.

### AS-2 — Burn-rate windows (multi-window multi-burn-rate)

Adopts the standard Google SRE multi-window multi-burn-rate
pattern. For an SLO with error budget `EB` (e.g., 0.5% =
`0.005`), burn rate at window `W` is the ratio of observed error
rate over window `W` divided by `EB`:

```
slo:<flow>:burn_rate_<W> = slo:<flow>:error_ratio_<W> / EB
```

Recording rules compute `error_ratio` over windows `5m`, `30m`,
`1h`, `6h` per SLO. Burn-rate alerts AND two of those windows
to suppress noise:

| Alert | Windows | Burn-rate threshold | `for:` |
|---|---|---|---|
| **Fast burn** | 5m AND 1h | both `> 14.4` | 2m |
| **Slow burn** | 30m AND 6h | both `> 6` | 5m |

The `14.4` and `6` constants are the standard SRE thresholds:
14.4× burn over 5m/1h consumes ~1 hour's worth of monthly
budget every 5 minutes (fast-burn); 6× burn over 30m/6h consumes
~10% of monthly budget every 6 hours (slow-burn). Together they
form the "fast and slow" pair that catches sudden outages and
gradual degradation.

`error_budget` constants are encoded in the recording rules
file; updating an SLO target updates the constant in one place.

### AS-3 — Severity tiers

Severity labels on the alert determine the operational response:

#### `severity: page`

- Wakes the on-call operator immediately.
- Used for: SLO fast-burn against `Committed` targets, runtime
  invariants whose breakage causes data loss (consumer-stall,
  store-flush failure, gate-read-failure rate sustained).
- **Not used for**: `Observing` SLOs (per AS-1), service-mesh
  glitches, transient configuration mismatches.

#### `severity: ticket`

- Creates a ticket for next-business-day review.
- Used for: SLO slow-burn against `Committed` targets, **all**
  burn alerts against `Observing` SLOs, gradual resource-leak
  signals (`process_goroutines` rising, `process_heap_alloc_bytes`
  rising), latency degradation that hasn't yet broken SLO.
- Rationale: ticket-grade signals deserve human attention but
  do not warrant interrupting the operator's day.

#### `severity: info`

- Logged for trend analysis; no human action expected at the
  moment of firing.
- Used for: deprecation warnings, scheduled-maintenance
  reminders, "this metric will be removed in onda N" notices.
- The foundry does not currently emit `info` alerts; reserved
  for future use.

### AS-4 — Alert label conventions

Every alert MUST carry the following labels:

- `severity` — `page` / `ticket` / `info` (per AS-3).
- `slo` — the SLO identifier when the alert defends an SLO
  (`ingest_success`, `derive_latency`, `store_read_latency`,
  `writer_persist`). Absent when the alert is runtime-safety
  (`consumer_stall`, `goroutine_leak`).
- `service` — `market-foundry`. Constant, but explicit so
  Alertmanager routing rules can filter by service.
- `flow` — the canonical flow identifier (`F1` / `F2` / `F3` /
  `F4`) when the alert maps to a `docs/operations/slo.md`
  entry. Absent for runtime-safety alerts.

Every alert MUST carry the following annotations:

- `summary` — one-line human-readable headline (≤80 chars).
- `description` — full explanation including the observed
  values, the threshold, and the time window.
- `runbook_url` — relative path under `docs/operations/runbooks/`
  pointing to the response procedure. For H-5, runbooks are
  stubs (created later in the operations-doc onda); the URLs
  resolve to actual files post-H-5.

### AS-5 — Silence conventions

When an alert fires for a known cause that does not warrant
operator response (planned maintenance, intentional baseline
shift), the silence is created with:

- `creator` — operator identity.
- `comment` — reason + expiry rationale.
- `expires_at` — explicit expiration; no open-ended silences.

Silences ≥ 24 hours require documentation in
`docs/operations/incident-log.md` (future file, currently
optional). H-5 does not enforce this — it is documented here as
the target practice.

### AS-6 — Alerting against runtime invariants

In addition to SLO-defending alerts, the foundry emits
**runtime-safety alerts** that defend invariants whose breakage
predicts data loss or system unavailability:

- `consumer_stall` — JetStream consumer lag growing with zero
  acks for ≥ 5 minutes. `severity: page` (data path stalled).
- `goroutine_leak` — `process_goroutines > 10000` sustained for
  5 minutes. `severity: ticket`.
- `heap_alloc_high` — `process_heap_alloc_bytes > 500MB`
  sustained for 5 minutes. `severity: ticket`.
- `gate_read_failure_rate_high` — sustained non-zero
  `marketfoundry_execution_gate_read_failures_total` increment
  rate (ADR-0012 fail-open posture observability).
  `severity: ticket`.
- `seq_gap_rate_nonzero` — sustained non-zero
  `marketfoundry_consumer_seq_gap_total` increment rate
  per (venue, event_type). `severity: ticket`. ADR-0020
  documents gaps as recoverable, not fatal; ticket is the right
  tier.

These do **not** belong to an SLO and so omit the `slo` and
`flow` labels (per AS-4).

## Non-goals

- **Paging integration.** H-5 emits alerts with the `severity`
  label, but does not configure PagerDuty / Opsgenie / phone
  call infrastructure. Single-operator phase; paging arrives
  with a multi-operator on-call rotation, future phase.
- **Alertmanager routing rules.** The label set defined here
  supports Alertmanager routing; the routing rules themselves
  are deployment concern, not policy. H-5 ships Prometheus with
  alert rules; routing destination is `null` (logs to
  Prometheus self).
- **Incident-management workflow.** Tickets create ticket files
  somewhere; the choice of system (GitHub Issues, Linear, Jira)
  is out of scope. The `severity: ticket` label is a signal,
  not a configuration of the ticketing system.
- **Service-level *agreements* (SLAs).** SLOs are internal
  targets; SLAs are external contracts. The foundry has no
  external clients in this phase, so no SLAs.

## Alternatives considered

- **(A) No alerts until SLOs are `Committed`.** Rejected: loses
  observability during the validation window. Operators need to
  see broken systems even while we baseline.
- **(B) Same severity tier for all SLO alerts regardless of
  status.** Rejected: defeats the purpose of the taxonomy.
  `Observing` and `Committed` are operationally different
  states; paging on an unvalidated target is the noise we want
  to avoid.
- **(C) Single-window burn-rate (only fast).** Rejected: misses
  slow degradation that would consume the budget over a day or
  two. Multi-window catches both modes.
- **(D) Per-flow distinct burn-rate constants.** Rejected:
  Google SRE's 14.4 and 6 are validated against a wide range
  of SLO targets; per-flow tuning is premature optimization
  before any flow is `Committed`. If a flow's burn-rate
  thresholds prove wrong post-`Committed`, we adjust then.

## Consequences

### Positive

- **Honest staging of SLO confidence.** The taxonomy makes
  explicit what was implicit: that the foundry's SLO targets
  start as proposals and earn the right to page the on-call
  only after measurement.
- **Operators trust the alerting system.** The cost of being
  paged for a false positive on an unvalidated target is high
  (trust erodes). Capping `Observing` at `ticket` preserves
  the contract that a `page` is real.
- **Conventional toolchain.** Multi-window multi-burn-rate is
  the established Google SRE pattern; on-call rotations new to
  the foundry recognize it without ramp-up.
- **Severity labels survive deployment changes.** The label set
  defined in AS-4 supports any Alertmanager routing in the
  future; H-5's choice to leave routing as `null` does not
  paint a future onda into a corner.

### Negative

- **`Observing` alerts may be ignored** if operators decide
  "tickets are not urgent" and they accumulate without review.
  Mitigated by the explicit promotion criterion in AS-1
  (`Observing` → `Committed` requires 7 days of compliance
  before being eligible for promotion; tickets during this
  period are the data informing promotion).
- **Three-state taxonomy adds complexity** to `slo.md`. Each
  SLO carries an extra field. The alternative (binary
  "active/inactive") is simpler but loses the validation phase.
- **Runtime-safety alerts overlap conceptually with SLOs.**
  `consumer_stall` could be argued as defending F2 (derive
  latency). The split here is operational: SLO alerts defend a
  *target* that the operator agreed to; runtime-safety alerts
  defend an *invariant* that breaks the system regardless of
  any target.

## Promoção para Accepted

This ADR is promoted from `Proposed` to `Accepted` when **Onda
H-5** ships:

1. `deploy/observability/prometheus/recording.rules.yml` contains
   recording rules for F1–F4 implementing the burn-rate windows
   per AS-2 (H-5 commit 6).
2. `deploy/observability/prometheus/alerts.rules.yml` contains
   burn-rate alerts for F1–F4 plus the runtime-safety alerts of
   AS-6, with labels and annotations per AS-4 (H-5 commit 7).
   Severity tiers per AS-3, with all SLO alerts at `Observing`
   tier (capped to `ticket`).
3. `docs/operations/slo.md` flipped from template to
   `Status: Active`. F1–F4 each carry the taxonomy state
   `Observing` per AS-1 (H-5 commit 10).

H-5 commit 11 flips the `Status` field of this ADR to `Accepted`
in the same commit that closes the onda.

## References

- ADR [0024](0024-metrics-policy.md) — sibling ADR; this policy
  governs *alerting* over the metrics that policy governs.
- ADR [0012](0012-control-gate-fail-open-posture.md) — the
  `gate_read_failure_rate_high` runtime-safety alert defends
  the fail-open observability ADR-0012 requires.
- ADR [0020](0020-sequencing-and-time-normalization.md) — the
  `seq_gap_rate_nonzero` runtime-safety alert observes the
  gap-detection invariant; ADR-0020 explicitly classifies gaps
  as recoverable (not fatal), justifying `ticket` tier.
- [`../operations/slo.md`](../operations/slo.md) — the
  document this ADR governs.
- [`../operations/runtime-invariants.md`](../operations/runtime-invariants.md)
  — the invariants that the runtime-safety alerts in AS-6 defend.
- [`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest" — P3
  (capacidade portada passa por documento primeiro).
- [PROGRAM-0003](../programs/PROGRAM-0003-observability.md) — H-5
  scope.
- raccoon `docs/observability/alerting-policy.md` (if present)
  and `deploy/observability/prometheus/alerts.rules.yml` —
  inspiração. Foundry diverges by (a) introducing the explicit
  three-state **SLO status taxonomy** (`Proposed`/`Observing`/
  `Committed`) and capping `Observing` at `ticket` severity —
  raccoon does not formalize this state machine; (b) tying the
  taxonomy promotion criterion to *7 contiguous days of
  compliance* rather than indefinite manual review;
  (c) defining `runtime-safety` as a distinct alert category
  separate from SLO-defending alerts, with documented label
  conventions (`slo` absent, `flow` absent for these).
