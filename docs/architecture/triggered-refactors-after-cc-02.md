# Triggered Refactors After CC-02

## Context

This document records the refactors executed in S129, directly triggered by CC-02 (EMA Crossover Signal Family) extensibility proof findings captured in S128.

**Governing principle:** Only refactors whose trigger condition was definitively met by CC-02 are executed. No speculative, aesthetic, or horizontal improvements.

---

## R1: HTTP Correlation ID Middleware (CF-03)

### Trigger

CF-03 trigger condition: *"first new actor that copies the `events.NewMetadata().WithCorrelationID(msg.CorrelationID)` pattern."*

**Met by:** `EMACrossoverSignalSamplerActor` (ema_crossover_signal_sampler_actor.go:70), which copies the identical pattern. CC-02 also confirmed that every HTTP handler independently performed `requestctx.WithCorrelationID(r.Context(), r.Header.Get("X-Correlation-ID"))` — 12 occurrences across 7 handler files.

### What Was Done

Introduced `webserver.CorrelationID` HTTP middleware that:
1. Extracts `X-Correlation-ID` header from incoming requests.
2. Injects it into the request context via `requestctx.WithCorrelationID`.
3. Applies globally at the `WebServer.buildHTTPServer()` level.

Removed manual correlation ID extraction from all HTTP handlers:
- `signal.go` (1 occurrence)
- `decision.go` (1 occurrence)
- `risk.go` (1 occurrence)
- `strategy.go` (1 occurrence)
- `execution.go` (2 occurrences)
- `execution_control.go` (2 occurrences)
- `evidence.go` (4 occurrences)
- `configctl.go` (1 inline + 8 via `withCorrelationID` helper)

Removed the `withCorrelationID` private helper from `configctl.go`.

Cleaned up `requestctx` import from all handler files that no longer reference it.

### Scope of CF-03 Addressed

| Surface | Status | Notes |
|---------|--------|-------|
| HTTP handler layer | **Addressed** | Middleware eliminates per-handler boilerplate |
| Actor correlation propagation | **Not addressed** | Actor `events.NewMetadata().WithCorrelationID(msg.CorrelationID)` pattern remains manual; addressing this requires actor framework changes beyond the trigger scope |

The actor-layer pattern remains because:
- It operates in Hollywood actor message passing, not HTTP context.
- Centralizing it requires a generic actor middleware or message wrapper — a deeper abstraction.
- At N=2 families, N=10 actors, the copy-paste pattern has produced zero incidents.
- The natural trigger for actor-layer centralization is N=3 signal families (CF-08 threshold).

### Evidence

- **Lines removed:** ~20 lines of manual extraction across 7 handler files.
- **Lines added:** ~10 lines (middleware function + wiring).
- **Net reduction:** ~10 lines.
- **Extensibility gain:** Every future HTTP handler automatically inherits correlation ID support. Zero per-handler boilerplate for new families.
- **Test coverage:** New middleware tests (`middleware_test.go`); existing handler tests updated to reflect new pattern.
- **Zero behavioral change:** Middleware performs identical logic to the per-handler extraction it replaces.

---

## Summary

| Refactor | Friction ID | Trigger Met | Lines Changed | Risk |
|----------|-------------|-------------|---------------|------|
| HTTP Correlation ID Middleware | CF-03 | Yes (CC-02 actor + 12 handler occurrences) | ~30 | Minimal — identical behavior |

Only one refactor was executed because only one trigger was definitively met by CC-02 at the HTTP layer. This is by design.
