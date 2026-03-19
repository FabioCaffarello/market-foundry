# Next Technical Wave Recommendations

> Concrete, evidence-based recommendations for what should follow the S96–S99 structural consolidation, and what should not.

---

## Decision Framework

The consolidation wave (S96–S99) focused on reducing the cost of evolution. The next wave should **use** that reduced cost to deliver value — either through product capability, operational confidence, or integration readiness.

**Criteria for recommending a wave:**
1. It addresses a bottleneck that is currently blocking or slowing real work.
2. It builds on the structural foundation established in S96–S99.
3. Its value is measurable or at least falsifiable.
4. It does not re-open structural questions that are already settled.

---

## Recommended: Vertical Slice Completion

**What:** Complete the first end-to-end vertical slice — from market data ingestion through evidence derivation, signal generation, decision evaluation, strategy resolution, risk assessment, and execution — with a single family chain running against a real (testnet) venue.

**Why this is the right next step:**
- The structural consolidation was investment in the system's ability to evolve. The return on that investment is realized when the system actually runs end-to-end.
- The first vertical slice will expose integration issues that no amount of structural refactoring can surface (NATS subject mismatches, actor message routing bugs, config activation gaps).
- The expansion playbooks from S99 exist precisely to guide this kind of work — using them validates their accuracy.

**Scope:**
- Family chain: `candle` → `rsi` → `rsi_oversold` → `mean_reversion_entry` → `position_exposure` → `paper_order`
- Venue: paper simulator (already implemented)
- Data source: Binance Futures testnet (adapter exists)
- Goal: A trade triggers the full pipeline chain and produces a paper order

**What this validates:**
- Catalog-driven assembly actually works end-to-end (all families activate from config)
- Cross-domain event routing through actors is correct
- NATS infrastructure (streams, consumers, KV buckets) is properly configured
- Health trackers report meaningful status

**What this does NOT require:**
- New architectural patterns
- New domains or runtimes
- Changes to composition roots or DI patterns

---

## Recommended: Operational Confidence Layer

**What:** Add minimal observability to the actor pipeline — structured logging at pipeline boundaries, basic latency metrics for the full chain, and a diagnostic endpoint that shows pipeline status.

**Why:**
- When the vertical slice runs, debugging failures will require tracing events through 6 domain boundaries. Without structured observability, debugging means reading logs across 4 binaries.
- Health trackers currently report binary status. They don't show whether a pipeline is processing events or stalled.
- This is not a "nice to have" — it's a prerequisite for confident operation.

**Scope (minimal):**
- Structured log entries at domain handoff points (candle finalized → signal sampler → decision evaluator → ...).
- Pipeline throughput counter (events processed per family per minute).
- Diagnostic endpoint on gateway showing: enabled families, pipeline status, last event timestamp per family.

**What this does NOT require:**
- Distributed tracing infrastructure (OpenTelemetry, Jaeger)
- Metrics aggregation (Prometheus, Grafana)
- Dashboard development

**Guard rail:** Start with structured logs and a single diagnostic endpoint. Do not introduce tracing infrastructure until the operational need is demonstrated.

---

## Conditionally Recommended: MarketMonkey Absorption

**What:** Absorb the MarketMonkey codebase into the Foundry monorepo, applying the structural patterns established in S96–S99.

**Precondition:** The vertical slice must be running end-to-end first. Absorbing MarketMonkey into a system that hasn't proven its runtime behavior would compound integration risk.

**Why (when ready):**
- MarketMonkey contains market data adapters and exchange integrations that the Foundry needs.
- The expansion playbooks from S99 provide a documented path for adding new adapters and data sources.
- The guardian tooling (raccoon-cli) can validate that absorbed code follows established conventions.

**Risk:**
- MarketMonkey may have patterns that conflict with Foundry conventions. These must be adapted during absorption, not preserved.
- The absorption scope must be bounded — only absorb what the Foundry needs, not the entire MarketMonkey codebase.

---

## Not Recommended Now

### Event Schema Formalization

Introducing Protobuf, Avro, or JSON Schema for domain events adds build complexity and deployment coupling. The system has single producers per event type and all communication is intra-cluster via NATS. Schema enforcement solves a problem that doesn't exist yet (multi-team, multi-language consumers).

**When to reconsider:** When a second team or a non-Go consumer needs to read domain events.

### Test Infrastructure Wave

Comprehensive integration tests for composition roots and contract tests for NATS subjects would improve confidence but are not the bottleneck. The vertical slice completion will naturally drive test creation where it's needed.

**When to reconsider:** When test failures become a recurring pain point, or when the team grows beyond solo development.

### Additional Structural Refactoring

The consolidation wave achieved its objectives. Further refactoring (unified supervisor framework, generic repository interface, automated doc generation) would consume effort without addressing a current bottleneck.

**When to reconsider:** When a specific pattern becomes a measurable drag on development velocity.

### Multi-Venue Support

The execute runtime supports paper simulator and Binance Futures testnet. Adding more venues (spot exchanges, other derivatives platforms) should wait until the first venue integration is proven end-to-end.

**When to reconsider:** After the paper simulator vertical slice runs reliably and a real venue integration is the next product goal.

---

## Recommended Wave Sequence

| Priority | Wave | Precondition | Estimated Scope |
|----------|------|--------------|-----------------|
| 1 | Vertical slice completion | None | 3-5 stages |
| 2 | Operational confidence layer | Vertical slice running | 1-2 stages |
| 3 | MarketMonkey absorption | Vertical slice + observability | 2-4 stages |

This sequence maximizes return on the structural investment while keeping each wave focused and falsifiable. Each wave validates the previous one — the vertical slice validates the structural patterns, the observability layer validates the runtime behavior, and MarketMonkey absorption validates the growth playbooks.
