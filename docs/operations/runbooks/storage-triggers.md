# Runbook — Storage-tier triggers (ADR-0023)

Storage Stage 2 (TimescaleDB, Onda H-10) is **trigger-gated**. This
runbook covers the alerts that signal a trigger *may* be firing. Opening
H-10 is the **maintainer's decision** after confirming a sustained,
non-transient breach — the alerts are early signals, not auto-triggers.

## Triggers (ADR-0023)

| Trigger | Signal | Threshold | Alert | Recording rule |
|---|---|---|---|---|
| T1 | Gateway operational-query p99 (vs ClickHouse/KV) | > 50 ms sustained 7d | `StorageTriggerT1GatewayLatency` | `storage:gateway_op_query_p99_5m` |
| T2 | `store` RSS (proxy for KV-projection memory) | > 4 GB | `StorageTriggerT2StoreMemory` | `storage:store_rss_bytes` |
| T3 | Cliente Odin (H-12+) SLO miss on an un-precomputed query | client SLO miss | — (no alert — Odin does not exist yet) | — |

## When an alert fires

1. **Confirm it is sustained, not transient.** T1 requires ~7 contiguous
   days above 50 ms; the 1h alert window is only an early signal. Check
   the trend in Grafana against the `storage:*` recording series.
2. **Confirm the cause is structural** (not a deploy, backfill, or noisy
   neighbor). For T2, check whether KV projection growth (insight shapes,
   cross-venue overlays) is the driver vs. a leak.
3. **If confirmed**, record the trigger as fired (with evidence) in
   [`../../RESUMPTION.md`](../../RESUMPTION.md) and open **Onda H-10** per
   [ADR-0023](../../decisions/0023-storage-tier-roadmap.md) → "Stage 2":
   add TimescaleDB to the stack, `internal/adapters/storage/timescale/`,
   migrate the specific query pattern that fired, promote ADR-0023 fully.
4. **If transient**, no action — staying in Stage 1 is a legitimate
   steady state (ADR-0023).

## What NOT to do

- Do **not** adopt TimescaleDB preemptively because an alert blipped.
  ADR-0023's rule is **"triggers first, onda second"**.
