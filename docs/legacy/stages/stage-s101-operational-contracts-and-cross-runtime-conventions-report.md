# Stage S101 — Operational Contracts and Cross-Runtime Conventions Report

> Consolidation of operational contracts and shared behavior rules across market-foundry runtimes.

---

## Objective

Consolidate operational contracts and cross-runtime conventions that were previously implicit, making them explicit, verifiable, and documented — without introducing unnecessary rigidity.

## Context

S100 confirmed that composition roots, DI patterns, and structural assembly are solid. However, several operational conventions existed only as patterns replicated across `cmd/*/run.go` files without documentation or enforcement. This stage makes those conventions explicit and fixes the inconsistencies found during review.

---

## 1. Executive Summary

S101 reviewed all 6 runtimes (gateway, store, derive, ingest, execute, configctl), identified 3 concrete inconsistencies, fixed them in code, and produced 2 architecture documents formalizing 10 cross-runtime invariants and 7 shared behavior rules.

**Key outcome:** The operational contract surface is now documented, verifiable, and bounded. The distinction between invariants (non-negotiable), conventions (with documented exemptions), and local behaviors (runtime-owned) is explicit.

---

## 2. Contracts and Conventions Consolidated

### Formalized as Invariants (10)

| ID | Invariant | Status Before | Status After |
|----|-----------|---------------|--------------|
| INV-1 | Single entrypoint via `bootstrap.Main` | Implicit | Documented |
| INV-2 | Logger installed before any work | Implicit | Documented |
| INV-3 | Engine failure is terminal | Implicit | Documented |
| INV-4 | Actors stop before health server | Implicit | Documented |
| INV-5 | Error key is `"error"` | Violated (gateway actor) | Fixed + documented |
| INV-6 | No `init()` registration | Implicit | Documented |
| INV-7 | No cross-domain imports | Verified in S100 | Documented |
| INV-8 | `*problem.Problem` across boundaries | Implicit | Documented |
| INV-9 | Compile-time interface proof | Implicit | Documented |
| INV-10 | Graceful shutdown timeout consistency | Implicit | Documented with rationale |

### Formalized as Shared Behavior Rules (7)

| ID | Rule | Notes |
|----|------|-------|
| BHV-1 | Startup log format | `"<runtime> starting"` as first word |
| BHV-2 | NATS readiness check | All NATS-dependent runtimes |
| BHV-3 | Health server on `config.HTTP.Addr` | Gateway exempt (uses main server) |
| BHV-4 | Tracker naming convention | `<family>-<role>` pattern |
| BHV-5 | Connection cleanup via defer | Composition root responsibility |
| BHV-6 | Registry as value object | No state, no lifecycle |
| BHV-7 | Config-driven activation | Venue adapters exempt (security) |

### Documented as Local Behaviors (5)

- Gateway connection topology
- Supervisor internal structure
- Tracker granularity
- Readiness check composition
- NATS consumer configuration

---

## 3. Files Changed

### Code Changes

| File | Change | Reason |
|------|--------|--------|
| `internal/actors/scopes/gateway/gateway.go` | `"err"` → `"error"` in error log key | INV-5 enforcement: consistent structured log key |
| `cmd/configctl/run.go` | Added health server with `/healthz`, `/readyz`, `/statusz` | Configctl was the only runtime without health endpoints. Aligns with cross-runtime health server contract. |
| `cmd/configctl/run.go` | Standardized import grouping | Canonical Go import grouping (stdlib, then internal) |

### Architecture Documents Created

| File | Purpose |
|------|---------|
| `docs/architecture/operational-contracts-and-cross-runtime-conventions.md` | Canonical reference for all operational contracts: lifecycle, health, logging, error handling, NATS, actors, trackers |
| `docs/architecture/runtime-invariants-and-shared-behavior-rules.md` | Classification of invariants vs. conventions vs. local behaviors, with verification checklist and trade-off rationale |

---

## 4. Inconsistencies Reduced

### Fixed

| Inconsistency | Before | After |
|---------------|--------|-------|
| Gateway actor error key | Used `"err"` in `slog.Error("failed to start gateway", "err", err)` | Corrected to `"error"` — matches all other log sites |
| Configctl health server | No health server. Only runtime without `/healthz`, `/readyz`, `/statusz`. | Added `healthz.NewHealthServer` with NATS readiness check and `nil` trackers |
| Configctl import ordering | stdlib and internal imports mixed in non-canonical grouping | Canonical grouping: stdlib first, then internal |

### Already Consistent (Verified)

| Area | Finding |
|------|---------|
| Shutdown sequence | All runtimes stop actors before health server ✅ |
| Signal handling | All runtimes use `WaitTillShutdown` ✅ |
| Logger setup | All runtimes call `BuildLogger` + `slog.SetDefault` in phase 1 ✅ |
| Engine creation | All runtimes exit on engine failure ✅ |
| Health server timeout | All runtimes use `5 * time.Second` for health shutdown ✅ |
| Actor poison timeout | Single canonical `10 * time.Second` in `WaitTillShutdown` ✅ |
| NATS readiness | All NATS-dependent runtimes include NATS readiness check ✅ |

---

## 5. Limits Maintained

### Standardization Boundaries

The following areas were explicitly kept as local runtime responsibilities:

1. **Supervisor topology** — Each supervisor owns its child structure. No unified supervisor framework.
2. **Consumer configuration** — AckWait, MaxDeliver, filter subjects are domain decisions.
3. **Stream configuration** — Retention, storage type, max age are per-domain.
4. **HTTP route structure** — Gateway-local concern.
5. **Venue adapter activation** — Explicit switch-case, not registry-driven (security).

### Why Not a Unified Runtime Interface

A shared `Runtime` interface was evaluated and rejected:
- Each runtime's startup is meaningfully different.
- A generic interface would be either too broad (useless) or too narrow (constraining).
- 6 runtimes are small enough to verify by pattern inspection.
- The 6-phase lifecycle achieves consistency through convention, not type constraint.

### Why Not Configurable Timeouts

Making shutdown timeouts configurable was considered and rejected:
- Must coordinate with orchestrator's `terminationGracePeriodSeconds`.
- Misconfiguration risk outweighs flexibility benefit.
- If a runtime needs different timeouts, it should be an architectural decision.

---

## 6. Preparation for S102

### Recommended Focus Areas

Based on the open debts identified in S100 and the contracts formalized in S101:

1. **Test infrastructure for operational contracts** — The contracts documented here have no automated verification. A lightweight test that validates composition root behavior (startup log, health endpoints, shutdown sequence) would prevent regression.

2. **Observability maturity** — Health trackers report binary-level status. Distributed tracing (correlation ID propagation across NATS boundaries) is partially implemented (`requestctx.CorrelationID`) but not systematically verified.

3. **Error handling policy** — INV-8 defines `*problem.Problem` as the boundary error type, but the policy for when to degrade vs. fail (e.g., optional NATS connections in gateway vs. required connections) is still runtime-local. A decision matrix would reduce ambiguity.

4. **Raccoon-cli rule expansion** — The architecture guardian currently enforces 11 structural rules. The invariants documented in this stage (INV-1 through INV-10) could become raccoon-cli lint checks for continuous enforcement.

---

## Criteria de Aceite — Verificacao

| Criterio | Status |
|----------|--------|
| Contratos operacionais antes implicitos ficam explicitos | ✅ 10 invariants + 7 behavior rules documented |
| Comportamento cross-runtime fica mais previsivel | ✅ 6-phase lifecycle, health server, logging, shutdown sequence formalized |
| Inconsistencias desnecessarias sao reduzidas | ✅ 3 code fixes (error key, health server, imports) |
| Padronizacao nao apaga responsabilidades locais | ✅ 5 local behaviors explicitly preserved |
| Base preparada para futuras ondas sem aumento de ambiguidade | ✅ Verification checklist + S102 recommendations provided |

---

## Guard Rails — Compliance

| Guard rail | Compliance |
|------------|------------|
| Nao adicionar features novas | ✅ Only standardization and documentation |
| Nao criar framework transversal excessivamente rigido | ✅ Rejected unified Runtime interface |
| Nao impor uniformidade artificial | ✅ 5 local behaviors preserved with rationale |
| Nao misturar contratos operacionais com logica de dominio | ✅ Contracts are infrastructure-level only |
| Documentar trade-offs de qualquer consolidacao | ✅ Trade-offs section in both architecture docs |
