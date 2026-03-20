# TC-01 Prioritized Friction Matrix

> **Wave:** TC-01 (Timeframe Coverage)
> **Stage:** S134 (Friction Capture)
> **Date:** 2026-03-19
> **Source:** `timeframe-coverage-01-frictions-and-findings.md`

---

## 1. Purpose

This matrix orders every friction point by actionability and impact, enabling informed decisions about what to fix, what to defer, and what to accept permanently. The matrix is designed to answer one question: **"What should we do next?"**

---

## 2. Priority Tiers

| Tier | Meaning | Action Timeline |
|------|---------|----------------|
| **P1** | Fix before TC-02 planning begins | Next 1-2 stages |
| **P2** | Address as part of TC-02 preparation | Before TC-02 execution |
| **P3** | Track; revisit if conditions change | No action now |
| **P4** | Accept permanently | No action planned |

---

## 3. Prioritized Matrix

### Tier P1 — Fix Before TC-02 Planning

| ID | Friction | Classification | Effort | Justification |
|----|----------|---------------|--------|---------------|
| F-02 | No timeframe validation in config | Operational Fragility | Small (~10 lines) | An operator typo can silently spawn meaningless samplers. Adding `ValidateTimeframes()` to `schema.go` (reject duplicates, reject <10s/>86400s) is trivial and prevents a class of silent misconfiguration. |
| F-17 | No post-crash recovery runbook for high TFs | Operational Fragility | Small (doc only) | A brief paragraph in the validation procedure documenting expected data loss on restart. Zero code change. Immediate operational value. |

### Tier P2 — Address as Part of TC-02 Preparation

| ID | Friction | Classification | Effort | Justification |
|----|----------|---------------|--------|---------------|
| F-05 | No per-timeframe idle detection | Operational Fragility | Medium | At 8+ TFs, distinguishing a stalled 4h sampler from one correctly waiting is operationally important. Requires timeframe-aware health reporting in tracker/diagnostic output. Should be in place before TC-02 adds 4h/daily timeframes. |
| F-04 | Single tracker for evidence publisher | Structural Debt | Medium | Per-timeframe tracker granularity becomes important for diagnosing stalls at higher cardinality. Natural companion to F-05. |
| F-13 | 3600s window state loss on crash | Structural Debt | Large | The highest-impact structural debt. At TC-02 (4h candles), a crash can lose 4 hours of accumulated state. Interim snapshots or WAL for in-progress candles should be evaluated before committing to 4h+ timeframes. |
| F-15 | No interim candle snapshots | Structural Debt | Large | Related to F-13. In-progress candle projection (`final=false`) eliminates the "dead zone" for long timeframes. Same implementation substrate as F-13 state persistence. |
| F-01 | Global timeframe list (not per-binding) | Structural Debt | Medium | If TC-02 introduces heterogeneous timeframe needs per symbol, binding-level timeframe overrides will be needed. Evaluate during TC-02 scoping. |

### Tier P3 — Track; Revisit If Conditions Change

| ID | Friction | Classification | Trigger for Revisit |
|----|----------|---------------|-------------------|
| F-07 | No "list available timeframes" endpoint | Structural Debt | When external consumers (beyond internal operators) query the system. |
| F-08 | Null response ambiguity (200 with null) | Operational Fragility | When the system is exposed to non-expert consumers who cannot cross-reference config. |
| F-19 | Gateway has no aggregate view | Structural Debt | When manual N-query checks become too tedious (likely at 10+ symbols or when dashboard integration is needed). |

### Tier P4 — Accept Permanently

| ID | Friction | Classification | Rationale for Acceptance |
|----|----------|---------------|------------------------|
| F-03 | Timeframe as integer seconds | Acceptable Boilerplate | The integer representation is unambiguous, machine-friendly, and consistent across all surfaces (config, NATS, KV, HTTP). Human-readable labels are a cosmetic nicety, not a structural need. |
| F-06 | Log verbosity scaling | Acceptable Boilerplate | Linear log growth is inherent to linear actor growth. Structured logging with `slog` already supports filtering. |
| F-09 | HTTP test file duplication | Acceptable Boilerplate | Each block is independently executable. The duplication provides documentation value. |
| F-10 | Actor count growth | Acceptable Boilerplate | Linear growth by design. Hollywood engine handles thousands of actors. |
| F-11 | KV key cardinality | Acceptable Boilerplate | NATS KV handles millions of keys. |
| F-12 | NATS subject cardinality | Acceptable Boilerplate | NATS handles millions of subjects. |
| F-14 | Signal warmup latency at high TFs | Accepted Limitation | Physics constraint of low-frequency analysis. Not an architectural issue. |
| F-16 | Smoke test wait time assumptions | Operational Fragility | The three-tier validation procedure correctly separates wiring validation (fast) from data validation (slow). The smoke test scope is appropriate. |
| F-18 | Configctl has no timeframe concept | Acceptable Boilerplate | Correct separation of concerns: configctl manages bindings, derive manages timeframes. |

---

## 4. Effort Summary

| Tier | Count | Code Changes | Doc Changes | Total Effort |
|------|-------|-------------|-------------|-------------|
| P1 | 2 | 1 small | 1 small | ~1 hour |
| P2 | 5 | 3 medium + 1 large | — | ~2-3 stages |
| P3 | 3 | 0 | 0 | None now |
| P4 | 9 | 0 | 0 | None |

---

## 5. Dependency Map

```
F-13 (window state loss) ←→ F-15 (interim snapshots)
  └── Same implementation substrate: state persistence for in-progress candles
  └── Evaluate together during TC-02 preparation

F-04 (single tracker) ←→ F-05 (idle detection)
  └── Both require per-timeframe diagnostic granularity
  └── Can be implemented together as a single enhancement

F-01 (global TF list) → depends on TC-02 scoping
  └── Only needed if TC-02 requires heterogeneous TF sets per symbol
```

---

## 6. Decision Framework for TC-02

Before starting TC-02 (4h/daily timeframes), the following P2 items should be resolved or explicitly accepted:

| Decision | Options | Recommendation |
|----------|---------|---------------|
| In-progress candle state (F-13/F-15) | (a) Accept 4h state loss risk, (b) Add WAL/snapshot, (c) Limit TC-02 to 4h only | Evaluate (b) cost. If prohibitive, accept (a) with documented risk. |
| Per-timeframe diagnostics (F-04/F-05) | (a) Keep aggregate, (b) Split tracker per TF | Implement (b) — low cost, high operational value. |
| Per-binding timeframes (F-01) | (a) Keep global, (b) Add binding-level override | Defer unless TC-02 design requires heterogeneous TFs. |

---

## 7. What This Matrix Does NOT Prioritize

The following are explicitly excluded from the friction matrix because they are **not frictions:**

- **NF-01 through NF-06:** Items that were anticipated as problems but did not materialize. These validate the architecture rather than pressure it.
- **Performance optimization:** No performance bottleneck was revealed by TC-01. The system grows linearly.
- **New features:** No new capability is proposed. This matrix is strictly about operational quality.

---

## 8. Next Steps

1. **S134 complete:** This matrix and the findings document provide the evidence base.
2. **P1 items:** Can be addressed in a lightweight follow-up (S135 or standalone commits).
3. **P2 items:** Form the preparation checklist for TC-02 scoping.
4. **TC-02 gate:** P2 decisions must be resolved before TC-02 execution begins.
