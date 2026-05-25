# ADR 0024: Metrics policy — naming, labels, cardinality, log compensation

## Status

Accepted. Promoted from `Proposed` by Onda H-5 (PROGRAM-0003,
Fase Observability) — all four acceptance criteria below are
backed by tracked code (compose profile in commit 5, refactored
`consumer_seq_gap_total` in commit 4, raccoon-cli `check metrics`
in commit 9, `slo.md` flipped to Observing in commit 10). See
"Promoção para Accepted" for the criterion-by-criterion mapping
and the Changelog entry below.

## Date

2026-05-25.

## Context

`market-foundry` has 11 Prometheus metrics already registered in
`internal/shared/metrics/` (audit performed during the H-5 pre-flight):

| Metric | Type | Subsystem | Labels |
|---|---|---|---|
| `marketfoundry_http_request_duration_seconds` | Histogram | (top) | method, path, status_code |
| `marketfoundry_http_requests_total` | Counter | (top) | method, path, status_code |
| `marketfoundry_consumer_messages_total` | Counter | consumer | consumer, outcome |
| `marketfoundry_consumer_processing_duration_seconds` | Histogram | consumer | consumer |
| `marketfoundry_consumer_lag_messages` | Gauge | consumer | consumer |
| `marketfoundry_consumer_seq_gap_total` | Counter | consumer | stream_key |
| `marketfoundry_execution_strategy_evaluations_total` | Counter | execution | strategy_type, outcome |
| `marketfoundry_execution_gate_checks_total` | Counter | execution | gate, verdict |
| `marketfoundry_execution_intents_total` | Counter | execution | source_path, side |
| `marketfoundry_execution_gate_active` | Gauge | execution | (none) |
| `marketfoundry_execution_gate_read_failures_total` | Counter | execution | reason |

Naming follows `marketfoundry_<subsystem>_<name>` consistently;
the histograms terminate in `_seconds`, the counters terminate in
`_total`. The substrate is largely compliant with the policy this
ADR is about to ratify — except for one discrepancy that motivates
codifying the policy now rather than later:

**`marketfoundry_consumer_seq_gap_total{stream_key}` carries the
composite label value `"<venue>.<instrument>.<event_type>"`,
where `instrument` is a high-cardinality dimension (every traded
instrument adds a label value).** Without a written policy, this
pattern would propagate as new counters are added by future ondas
(insights, multi-venue, delivery). High-cardinality labels are
cheap to add and expensive to remove: every series adds memory,
scrape time, and rule-evaluation cost; the cumulative footprint
on Prometheus crosses pain thresholds well before any single
counter looks "too big" in isolation.

H-5 also opens the question of **what to do with high-cardinality
dimensions when they are diagnostically valuable but operationally
expensive as labels**. The naive answer "drop them" loses
diagnostic value at the moment of alerting; the naive answer
"include them" sacrifices Prometheus health. This ADR records a
third path — *log compensation pattern* — that preserves
diagnostics without compromising the metric system.

## Decision

`market-foundry` adopts the following **metrics policy** for every
Prometheus instrument declared in or under `internal/shared/metrics/`:

### MP-1 — Naming convention

- **Namespace**: every metric is prefixed `marketfoundry_`.
- **Subsystem**: every metric declares a subsystem
  (`http`, `consumer`, `execution`, `sequencer`, future
  `ingest`, `derive`, `store`, `writer`). Top-level metrics
  without a clear subsystem are discouraged; the current
  `marketfoundry_http_*` pair predates this ADR and is
  grandfathered.
- **Unit suffix** is required:
  - latency / duration → `*_seconds` (use `time.Duration.Seconds()`
    when observing).
  - size → `*_bytes`.
  - ratio → `*_ratio` (range `[0, 1]`, sourced as
    `success/total`).
  - cumulative count → `*_total`.
  - instantaneous gauge → no suffix or domain-specific suffix
    (e.g., `_messages`, `_active`).
- **Verbs in names** describe the measured quantity, not the
  action. `gap_total` (the quantity of gaps), not
  `gap_detect_total` (which suggests the act of detecting).

### MP-2 — Label budget

Labels permitted by default (stable partitioning dimensions):

- `venue` — `binance` / `binancef` / future `bybit` / etc.
- `event_type` — the canonical event-family identifier
  (`observation.trade`, `evidence.candle`, ...).
- `stream` — the JetStream stream name when the metric is
  intrinsically per-stream.
- `outcome` — bounded enum (`ok` / `error` / `dropped` / ...).
- `severity` — bounded enum (`info` / `warn` / `error`).
- `status_code` — HTTP status code (bounded by HTTP spec).
- `method` — HTTP method (bounded).
- `path` — HTTP **route template** (`/signal/rsi/:source/:symbol/:timeframe`),
  **never** the actual request URL with parameters substituted.
- `reason` — bounded enum for failure classification
  (`nil_bucket`, `key_not_found`, `ctx_timeout`, ...).
- `consumer` — bounded by the number of declared durable
  consumers (low tens).
- `strategy_type` — bounded by the registered strategy list.
- `gate` / `verdict` — bounded enums.
- `side` — bounded enum (`buy` / `sell`).
- `source_path` — bounded enum.

Labels **prohibited** (high or unbounded cardinality):

- ❌ `instrument` — every traded instrument adds a series.
- ❌ `symbol` — synonym of `instrument`.
- ❌ `request_id`, `correlation_id`, `causation_id` — one
  series per request.
- ❌ `subject` (full NATS subject including dynamic key) —
  encodes instrument + stream.
- ❌ `window_id`, `seq`, `order_id`, `venue_order_id`, `session_id` —
  per-message / per-session identifiers.
- ❌ Composite labels that **encode** a prohibited dimension
  (e.g., `stream_key = "<venue>.<instrument>.<event_type>"`
  encodes `instrument`).

### MP-3 — Histogram bucket guidance

- **Latency / duration histograms** (`*_seconds`): default bucket
  set is
  `{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}`. This
  covers the foundry's operational regime (sub-millisecond NATS
  round-trips through multi-second ClickHouse writes) with
  Prometheus's default `histogram_quantile` behaviour. The
  HTTP request duration histogram (the only existing example)
  already follows this set.
- **Size histograms** (`*_bytes`): default
  `{1<<10, 1<<12, 1<<14, 1<<16, 1<<18, 1<<20}` (1 KB → 1 MB).
  No size histogram exists today; this guidance is for future
  ondas.
- **Ratio histograms**: avoid. Ratios are better expressed as
  recording rules `success_rate = success_total / total_total`,
  evaluated server-side at scrape time.
- **Per-metric override allowed** when the operational regime
  diverges (e.g., a multi-second batch flush would tolerate
  larger upper buckets). Override is documented inline at the
  declaration site.

### MP-4 — Cardinality budget per subsystem

The total number of label-value combinations per subsystem is the
operational cost. As of 2026-05-25, the foundry sits comfortably
below any threshold of concern; the budget below is the early
warning, not a quota:

| Subsystem | Current series count (rough) | Target ceiling |
|---|---|---|
| `http` | (method × path × status_code) ≤ ~60 routes × 6 methods × ~10 status codes = ~3.6k upper bound, ~600 observed | 5k |
| `consumer` | (consumer × outcome) ≤ ~20 durables × 4 outcomes = ~80 | 500 |
| `consumer_seq_gap_total` | (venue × event_type) ≤ ~5 venues × ~10 event types = ~50 (after H-5 refactor) | 200 |
| `execution` | mostly bounded enums; ~50 series | 500 |

When a subsystem nears its target ceiling, the discussion is:
(a) is the cardinality legitimate (new venues, new event types)?
(b) is a high-cardinality label leaking via a composite? (c)
should a sub-cohort metric replace it?

### MP-5 — Log compensation pattern

When a high-cardinality dimension is **diagnostically valuable
at alert-firing time** but **operationally infeasible as a
Prometheus label**, the call site MUST emit a structured log
record alongside the metric increment containing the omitted
dimension(s).

**Reference shape**:

```go
slog.Warn("sequencer.gap_detected",
    "venue", venue,
    "instrument", instrument,  // omitted from metric label
    "event_type", eventType,
    "last_seq", lastSeq,
    "current_seq", currentSeq,
    "gap_size", currentSeq-lastSeq-1)
metrics.IncSeqGap(venue, eventType)
```

The metric carries the bounded labels (`venue`, `event_type`); the
log carries the same dimensions plus the high-cardinality
diagnostic (`instrument`). When an alert fires on
`marketfoundry_consumer_seq_gap_total{venue="binance",event_type="observation.trade"}`,
the operator runs `docker logs <binary> | grep sequencer.gap_detected | grep binance`
and sees the per-instrument fan-out without polluting Prometheus.

**Naming convention for the log event**: dotted lowercase,
`<subsystem>.<event>`. Examples: `sequencer.gap_detected`,
`store.commit_failed`, `ingest.parse_error`.

**Required fields**: all metric labels MUST be present in the log
record (so log and metric correlate by dimension); plus the
high-cardinality dimension(s) omitted from the metric.

This pattern is **operationally compensated** by the fact that
log aggregation (Loki / Elasticsearch) is non-scope for
PROGRAM-0003. Until it lands, operators read logs from
`docker logs`; the pattern is documented so the diagnostic value
is preserved at structured-log level even without aggregation.

### MP-6 — Migration of existing metrics

The audit identified one violation: `consumer_seq_gap_total`
carries composite `stream_key` that encodes `instrument`. H-5
commit 4 refactors this counter to `{venue, event_type}` per
MP-2 and documents the log compensation pattern inline at the
declaration site (per MP-5) so future callers (none today)
follow the pattern.

Future violations: any counter that adds a prohibited label or
encodes one via composite must be flagged in code review and
either (a) split into multiple counters with bounded labels, or
(b) drop the dimension to log (MP-5).

## Non-goals

- **Tracing / span context labels.** Tracing is out of scope for
  PROGRAM-0003 (non-goal in the PRD). This policy governs
  metrics only.
- **Custom Prometheus exporters.** All metrics flow through the
  client_golang library and the `internal/shared/metrics/`
  package; no separate exporter binary.
- **Per-instrument exemplars.** Exemplars (sampled trace IDs
  attached to histogram buckets) are interesting but out of
  scope; they require tracing to be useful, which is non-scope.
- **Metric retention policies.** Prometheus storage retention is
  a deployment concern handled in `prometheus.yml` per
  environment, not policy.
- **External system metrics (NATS, ClickHouse).** The foundry
  scrapes its own binaries; NATS and ClickHouse export their
  own metrics via their own scrape endpoints, governed by their
  own conventions.

## Alternatives considered

- **(A) Allow `instrument` as a label, monitor cardinality.**
  Rejected: gives up the policy and forces case-by-case
  argument; once allowed, every counter adds it "just in case".
  The raccoon reference (`docs/observability/metrics-policy.md`)
  has the same prohibition for the same reason.
- **(B) Drop the high-cardinality dimension entirely (no log).**
  Rejected: loses diagnostics at the worst moment. Operators
  paged at 3am cannot debug "gap in binance/observation.trade"
  if the per-instrument fan-out is absent.
- **(C) Encode multiple counters per dimension cohort
  (one counter per top-K instruments, one "other" counter).**
  Rejected: introduces label drift (which K? when does an
  instrument leave the top-K?) and operational complexity
  disproportionate to the diagnostic gain.
- **(D) Use Prometheus exemplars to attach instrument context to
  histogram samples.** Rejected: exemplars require tracing
  infrastructure to be useful (Jaeger/Tempo back-reference),
  which is non-scope.

## Consequences

### Positive

- **Cardinality bounded.** Prometheus cost is predictable as the
  foundry scales venues / instruments / event types — high-cardinality
  dimensions never leak as labels.
- **Diagnostic value preserved.** Log compensation pattern keeps
  fine-grained diagnostics available at alert-firing time without
  paying Prometheus cost.
- **Policy is mechanically checkable.** The raccoon-cli `check
  metrics` analyzer (H-5 commit 9) can be extended in a future
  onda to validate label declarations against the
  permitted/prohibited list. H-5 itself only validates
  `/metrics` presence; label validation is a follow-up.
- **Existing instrumentation grandfathered.** The audit confirms
  current state is largely compliant; only one counter (the H-4
  `consumer_seq_gap_total`) needs refactor, scoped to a single
  H-5 commit.

### Negative

- **Log compensation pattern requires discipline.** Reviewer must
  verify that a counter declaration with a deliberately-reduced
  label set has an accompanying log call in the increment path.
  Eventually mechanizable via analyzer; not in H-5.
- **No exemplars** means no per-sample tracing link. Acceptable
  while tracing is non-scope.
- **Subsystem inflation.** Every new family wants its own
  subsystem; the namespace prefix grows. Mitigated by the
  prefix being optional in display (Grafana queries can hide
  prefix) and required only at the wire level.

## Promoção para Accepted

This ADR is promoted from `Proposed` to `Accepted` when **Onda
H-5** ships:

1. `consumer_seq_gap_total` refactored per MP-2 and MP-5
   (H-5 commit 4).
2. Log compensation pattern documented inline at the metric
   declaration site (H-5 commit 4).
3. The 4 SLOs in `docs/operations/slo.md` flipped to `Observing`
   reference this ADR for the label conventions their recording
   rules and alerts assume (H-5 commit 10).
4. raccoon-cli `check metrics` analyzer integrated in `make
   verify` (H-5 commit 9). Label validation against this policy
   is a future-onda extension and is documented as such in the
   analyzer's source.

H-5 commit 11 flips the `Status` field of this ADR to `Accepted`
in the same commit that closes the onda.

### Criterion-by-criterion mapping (post-H-5)

1. ✅ `consumer_seq_gap_total` refactored to `{venue, event_type}`
   per MP-2 in H-5 commit 4 (`0a48887`); composite `stream_key`
   label encoding `instrument` eliminated.
2. ✅ Log compensation pattern documented inline in
   `internal/shared/metrics/sequencer_metrics.go` (commit 4) at
   `IncSeqGap` with the canonical `slog.Warn("sequencer.gap_detected", ...)`
   reference shape.
3. ✅ `docs/operations/slo.md` flipped to `Active — Observing`
   in H-5 commit 10 (`ae49920`); references this ADR for label
   conventions and `consumer_seq_gap_total`'s new label shape
   that the runtime-aggregates recording rule consumes.
4. ✅ raccoon-cli `check metrics` analyzer integrated in
   `make verify` as gate Step 8 in H-5 commit 9 (`1467661`).
   Allowlist-driven (`tools/raccoon-cli/policies/binaries.toml`)
   with two exemption categories: `one_shot` and
   `transitive_registration`. Label validation against MP-2's
   permitted/prohibited list is a future-onda extension —
   documented as such in the analyzer's source.

## Changelog

- **2026-05-25** — ADR-0024 created (PROGRAM-0003 H-5 commit 2,
  status `Proposed`).
- **2026-05-25** — **Promoted to `Accepted`**. PROGRAM-0003 H-5
  delivered all four acceptance criteria across the H-5 PR:
  commit 4 refactored `consumer_seq_gap_total` per MP-2; commit
  9 introduced `check metrics` analyzer; commit 10 flipped
  `slo.md` to Observing referencing this ADR; this commit 11
  closes H-5 and promotes. See the H-5 PR for full diff.

## References

- ADR [0004](0004-raccoon-cli-static-enforcement.md) — analyzer
  framework that the future label-validation extension will build
  on; P5 of the Fase Harvest applies.
- ADR [0019](0019-deterministic-replay-time-invariants.md) — the
  determinism analyzer (sibling to the metrics analyzer) sets
  precedent for scope-bounded static enforcement.
- ADR [0020](0020-sequencing-and-time-normalization.md) — defines
  the gap-detection invariant that `consumer_seq_gap_total`
  surfaces; H-5 commit 4 refactor preserves that semantics.
- ADR [0025](0025-alerting-strategy.md) — sibling ADR; this
  policy and the alerting policy together govern PROGRAM-0003's
  artifacts.
- [`../operations/runtime-invariants.md`](../operations/runtime-invariants.md)
  — list of invariants the policy implicitly governs (gap rate,
  request latency, persist latency).
- [`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest" — P3
  (capacidade portada passa por documento primeiro) and P5
  (cada invariante traz seu enforcement, with the analyzer
  extension being the planned mechanism).
- [PROGRAM-0003](../programs/PROGRAM-0003-observability.md) — H-5
  scope.
- raccoon
  `docs/observability/metrics-policy.md` —
  inspiração. Foundry diverges by (a) documenting the **log
  compensation pattern** explicitly (MP-5) rather than leaving
  the operational alternative implicit; (b) declaring an explicit
  cardinality budget table (MP-4); (c) grandfathering existing
  pre-policy metrics (`marketfoundry_http_*`) instead of forcing
  immediate refactor; and (d) tying enforcement to a raccoon-cli
  analyzer (future) rather than reviewer discipline alone.
- **H-5 implementation note (analyzer scope)**: the `check
  metrics` analyzer (H-5 commit 9) uses a flat scan of each
  binary's main package; binaries that register `/metrics` in
  imported packages (e.g., `gateway` via
  `internal/interfaces/http/routes/core.go:364`) are listed in
  `tools/raccoon-cli/policies/binaries.toml` under
  `transitive_registration`. The allowlist is declarative — adding
  a binary to it requires editing the policy file and the reviewer
  auditing that the imported registration is correct. A future
  refactor may replace this list with transitive import closure
  scanning via `go list -deps` or AST-level import analysis; the
  technical debt is recorded inline at the policy file header.
