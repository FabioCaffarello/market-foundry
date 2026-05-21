# Stabilization Scope Freeze and Must-Finish Matrix

**Stage:** S205
**Date:** 2025-07-24
**Status:** FROZEN — This document is the authoritative scope boundary for the stabilization wave.

---

## Executive Summary

This matrix formally closes the current expansion wave (baseline → analytical runtime → Wave A → Wave B → codegen introduction) and defines the precise boundary of what must be completed, what may be deferred, and what is explicitly frozen before the next phase (strategic refactoring, architectural restructuring, and documentation cleanup).

The stabilization wave exists to **close open structural obligations** — not to add features, not to expand families, and not to begin cleanup. Every item below has been evaluated against a single criterion: **does leaving this undone create an unsafe foundation for the refactoring phase?**

---

## Stabilization Matrix

### MUST FINISH NOW (MF)

These items must be completed during the stabilization wave. Leaving them undone would create structural gaps that contaminate or block the refactoring phase.

| ID | Item | Current State | Why Must Finish | Owner Track | Effort |
|----|------|---------------|-----------------|-------------|--------|
| **MF-1** | H-5: Extract `parseAnalyticalParams()` helper | Scoped in S189, not implemented | Handler at 615/620 lines — hard ceiling. Without extraction, handler file is at physical limit. Refactoring phase cannot safely touch analytical handlers in this state. | Gateway / Handlers | Small (1–2h) |
| **MF-2** | CI smoke-analytical job stability verification | Job defined in ci.yml, not verified end-to-end on real PR | CI is the safety net for the refactoring phase. If smoke-analytical fails silently, refactoring can break analytical path without detection. | CI / DevOps | Small (1h) |
| **MF-3** | Codegen integrated check (`codegen-integrated-check.sh`) verified on all 7 families | Script exists, manifest-driven, needs full-chain verification | Codegen drift during refactoring would be invisible without this gate. All 7 specs must produce matching goldens and integrated check must pass. | Codegen | Small (1h) |
| **MF-4** | Writer binary removed from version control (`cmd/writer/writer`) | Binary committed (staged in git status) | Binary in repo is a hygiene issue that will propagate through refactoring. Must be gitignored and removed before stabilization closes. | Hygiene | Trivial |
| **MF-5** | Verify all 13 Go modules build cleanly (`go build ./...` per module) | Presumed working, not verified as gate | Refactoring phase will modify imports and dependencies. A clean build baseline is required as the entry condition. | Build | Small (30m) |
| **MF-6** | Verify all unit tests pass across all modules | Presumed passing, not verified as gate | Test green is the entry condition for refactoring. Any pre-existing failures will be confused with refactoring regressions. | Test | Small (30m) |
| **MF-7** | Codegen cross-spec validation (`codegen validate-all`) passes | Implemented in S201, needs gate verification | 7 specs must have unique family names, NATS durables, and subjects. Collision during refactoring would be catastrophic. | Codegen | Trivial |

### MAY DEFER (MD)

These items are real work but can safely wait until after the refactoring phase. They do not create structural risk if deferred.

| ID | Item | Current State | Why Safe to Defer | Risk of Deferral |
|----|------|---------------|-------------------|------------------|
| **MD-1** | Live event flow proof for generated families (D-1) | EMA is structural-only (compiles, subscribes, no producer) | Generated families participate in write path identically to manual ones. Structural proof is sufficient for refactoring safety. | Low — no operational risk, only validation gap |
| **MD-2** | Cross-layer codegen validation (D-2) | Signal layer only validated | Codegen produces fragments, not files. Cross-layer differences are in manual artifacts (mappers, handlers), not generated ones. | Low — codegen scope is A1+A2 only |
| **MD-3** | Mapper generation feasibility (A3/D-3) | Not designed | Mappers are fully manual. No dependency on this for refactoring. | None — pure expansion |
| **MD-4** | Automated fragment insertion (D-4) | Manual copy-paste, ~5 min/family | Manual process is low-error at 7 families. Automation is convenience, not correctness. | None |
| **MD-5** | Config registration automation (D-5) | Manual, spec-derivable | No family additions during stabilization, so no registration needed. | None |
| **MD-6** | Backoff jitter (DEF-U6) | No jitter, single writer instance | Thundering herd impossible with single instance. | None at current scale |
| **MD-7** | NATS consumer lag visibility (DEF-U3) | Writer health via `/statusz`/`/diagz` | Lag has never caused an operational incident. | Low — monitoring gap only |
| **MD-8** | ClickHouse client timeout configurability | Hardcoded | No timeout incidents observed. | Low |
| **MD-9** | Load testing baseline (D5 pre-Wave-B) | No performance baseline | Refactoring is structural, not performance-focused. Baseline can be established post-refactoring. | Low |
| **MD-10** | Reader 10-parameter positional signature refactoring (TRIG-3) | Escalation flag at 10 params | No new families during stabilization, so no 11th param. Refactoring phase may address this naturally. | None during stabilization |
| **MD-11** | Schema coherence compile-time verification (DEF-C2) | Review-enforced, 6 tables ~95 columns | Under 12-table/100-column threshold. | None — threshold not reached |
| **MD-12** | Gateway tracker integration (OD-03) | Gateway health inferred from downstream | Operational, not structural. | Low |
| **MD-13** | Automated baseline validation (OD-04) | 30 success criteria are manual checks | Refactoring does not change success criteria. | Low |
| **MD-14** | Test assertion hardcoded family count (D-6) | Tests break on new family addition | No new families during stabilization. Fix during refactoring. | None during stabilization |
| **MD-15** | `CODEGEN_ROOT` auto-detection (D-7) | Requires explicit env var | Developer experience issue, not correctness. | None |
| **MD-16** | Second codegen-first family (next-wave Wave 1) | EMA is first and only generated family | No family expansion during stabilization. | None — explicitly out of scope |
| **MD-17** | TC-01 deferred items (D-01 through D-06) | All deferred with TC-02 gate | TC-02 is not in scope. State persistence (D-06) is TC-02 hard gate. | None — TC-02 not started |

### EXPLICITLY FREEZE (EF)

These items are **prohibited** during the stabilization wave. Attempting them would contaminate scope, introduce risk, or conflict with the refactoring phase.

| ID | Item | Why Frozen | What Happens If Violated |
|----|------|-----------|--------------------------|
| **EF-1** | New analytical family expansion (Family 06+) | Stabilization is closure, not expansion. H-5 must complete first regardless. | Handler file exceeds ceiling, pattern regressions, scope creep |
| **EF-2** | Codegen template modification | Templates are frozen (S193). Changes require re-validation of all 14 golden snapshots. | Golden snapshot drift, CI failures, validation chain broken |
| **EF-3** | Codegen spec schema extension | 14-field schema sufficient for A1+A2. Extension requires its own architecture stage. | Schema evolution ceremony bypassed, spec/golden misalignment |
| **EF-4** | Retroactive manual-to-generated family conversion | 6 manual families are permanently golden references (S193 decision). | Loss of baseline, golden comparison invalidated |
| **EF-5** | Tier 2 codegen authorization (read-path generation) | Not designed, not validated. Requires Tier 1 production proof first. | Premature automation of untested patterns |
| **EF-6** | Massive documentation cleanup/archival | Next phase responsibility. Cleanup during stabilization mixes triaging with restructuring. | Scope confusion, lost context, premature deletion |
| **EF-7** | Architectural restructuring of module boundaries | Refactoring phase responsibility. Stabilization only verifies current state is sound. | Premature restructuring without clean baseline |
| **EF-8** | New NATS stream definitions or domain event types | Infrastructure expansion is not stabilization. | Coupling expansion, untested paths |
| **EF-9** | ClickHouse schema changes (new tables, column additions) | 7 migrations cover all 6 families. Schema is complete for current scope. | Migration ordering disruption, writer/reader misalignment |
| **EF-10** | Batch codegen generation | One-at-a-time validation required (S204 decision). | Untested batch behavior, validation gaps |
| **EF-11** | Writer pipeline structural changes | Write path has been immutable across 5 family expansions. | Regression risk to proven stable component |
| **EF-12** | New service introduction | No new `cmd/*` services. 8 services cover current scope. | Infrastructure sprawl |

---

## Scope Boundary Summary

| Category | Count | Total Effort | Timeline |
|----------|-------|-------------|----------|
| **Must Finish** | 7 items | ~5–6 hours | Before stabilization closes |
| **May Defer** | 17 items | Variable | Post-refactoring phase |
| **Explicitly Frozen** | 12 items | N/A | Prohibited until next expansion wave |

---

## Decision Record

This matrix was produced by reviewing:
- All `*gains-tradeoffs-and-open-debts*` documents (8 documents)
- All `*gate*` documents (7 documents)
- All `*next-wave*` documents (9 documents)
- All `*deferred*` and `*triggered*` documents (6 documents)
- Stage reports S200–S204
- Full implementation state of all 13 Go modules
- CI pipeline configuration
- Git staging area (uncommitted work)

No item was classified without evidence from at least one architectural document or implementation inspection.
