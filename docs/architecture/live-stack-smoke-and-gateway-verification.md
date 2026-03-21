# Live Stack Smoke and Gateway Verification

> S318 — Reproducible operational smoke for the full venue path with live stack.

## Purpose

This document defines the design and rationale for the S318 live stack smoke:
a single, reproducible script that validates the complete data path from venue
adapter through NATS, ClickHouse persistence, and gateway HTTP composite surface.

The smoke exists to reduce operational verification cost. Before S318, proving
that the live stack works end-to-end required running multiple separate smokes
(`smoke-round-trip`, `smoke-analytical`, individual curl probes). S318 unifies
these into a single command with clear PASS/FAIL semantics.

## Scope

The smoke validates six phases in sequence:

| Phase | What it proves |
|-------|---------------|
| 1. Stack readiness | ClickHouse, writer, gateway, and NATS are healthy and accepting requests |
| 2. NATS streams | EXECUTION_FILL_EVENTS and EXECUTION_EVENTS streams exist; venue-fill consumer is registered |
| 3. ClickHouse data | All six analytical tables (evidence, signals, decisions, strategies, risk_assessments, executions) have data or report clearly when empty |
| 4. Composite surface | `/analytical/composite/chains`, `/funnel`, `/dispositions` return HTTP 200 with valid JSON |
| 5. Single-family endpoints | All six `/analytical/{family}/...` endpoints return HTTP 200 |
| 6. Structural test gate | S317 round-trip Go tests pass |

## Data Path Under Validation

```
Binance Testnet (or paper path)
  → VenueAdapterActor / PaperOrderActor
    → NATS EXECUTION_FILL_EVENTS / EXECUTION_EVENTS
      → writer (venue_market_order / paper_order pipeline)
        → ClickHouse executions table
          → CompositeReader (5-table application-side composition)
            → gateway HTTP /analytical/composite/*
```

## Design Decisions

1. **Single script, single exit code.** The smoke is `scripts/smoke-live-stack.sh`
   invoked via `make smoke-live-stack`. It exits 0 on success, 1 on any failure.

2. **Warnings vs failures.** Empty tables or missing streams produce WARN, not
   FAIL. The smoke is designed to work on a freshly started stack where data
   may not yet have propagated. HTTP 503 or unexpected status codes produce FAIL.

3. **No data injection.** The smoke does not publish synthetic events to NATS.
   It validates whatever data the running pipeline has produced. This keeps the
   smoke side-effect free and safe to run repeatedly.

4. **Reuses lib.sh conventions.** Same color-coded output, same `ERRORS` counter,
   same `smoke_banner` / `smoke_fail_summary` patterns as all other smokes.

5. **Subsumes smoke-round-trip scope.** S318 covers everything `smoke-round-trip`
   validates plus the full analytical endpoint surface and disposition/funnel
   aggregation. The older smoke remains available for targeted S317 regression.

## Relationship to Existing Smokes

| Smoke | Scope | Overlap with S318 |
|-------|-------|--------------------|
| `smoke-first-slice` | NATS KV candle path | None (KV, not ClickHouse) |
| `smoke-multi-symbol` | Multi-symbol KV path | None |
| `smoke-analytical` | Writer → ClickHouse → HTTP families | Partial (S318 covers all families + composite) |
| `smoke-round-trip` | Venue fill persistence round-trip | Subsumed (S318 includes phases 1-4 of S317) |
| `smoke-live-stack` | Full live stack verification | — |
| `smoke-operational` | OS process behavior | None |
| `smoke-restart-recovery` | Restart resilience | None |

## Non-Goals

- Not a production CI gate (no SLA on runtime).
- Not a load test (single query per endpoint).
- Does not inject synthetic data.
- Does not validate WebSocket or async fills.
- Does not open dashboards or generate reports beyond stdout.
