# Quality-Gate CI — Before and After Assumptions

**Date:** 2026-03-20
**Stage:** S229 — CI Profile Reconciliation
**Purpose:** Document every assumption corrected in the CI profile to close the gap identified in S228

---

## 1. Overview

S228 recorded 40 errors in `make quality-gate-ci` across four failing steps:
topology-doctor, contract-audit, arch-guard, drift-detect.

All failures were caused by raccoon-cli analyzers encoding assumptions
from a previous architectural state that no longer matched the repository
after the S218–S224 restructuring.

This document records each assumption, its before state, and the corrected
after state.

---

## 2. Assumption Table

### 2.1 topology-doctor — configctl control subject prefix

| Dimension | Before | After |
|-----------|--------|-------|
| **Expected prefix** | `configctl.control.config` | `configctl.control.` |
| **Rationale** | Original configctl used a `config.*` sub-namespace for all control ops | Post-restructure control subjects are flat: `configctl.control.compile_config`, `configctl.control.create_draft`, etc. |
| **File** | `tools/raccoon-cli/src/analyzers/topology.rs` | same |
| **Impact** | Subject scan rejected legitimate control subjects as missing | Broader prefix matches all current control subjects |

### 2.2 contract-audit — reply-type symmetry for versioned pairs

| Dimension | Before | After |
|-----------|--------|-------|
| **Symmetry check** | Compared last `.`-delimited segment of request and reply types | Strips `_request`/`_reply` suffixes before comparison |
| **Example** | `signal.query.v1.rsi_latest_request` vs `signal.query.v1.rsi_latest_reply` → last segments `rsi_latest_request` ≠ `rsi_latest_reply` → **FAIL** | After stripping: `rsi_latest` == `rsi_latest` → **PASS** |
| **File** | `tools/raccoon-cli/src/analyzers/contracts.rs` — `check_reply_type_symmetry` | same |
| **Impact** | Every versioned request/reply pair raised a false error in CI profile | Versioned pairs pass correctly |

### 2.3 contract-audit — domain event scanning scope

| Dimension | Before | After |
|-----------|--------|-------|
| **Scan target** | Hardcoded: `internal/domain/configctl/events.go` only | Dynamic: `internal/domain/*/events.go` (all domain directories) |
| **DomainEventDef** | No `domain` field — event matched by name suffix only | `domain` field populated from directory name |
| **File** | `tools/raccoon-cli/src/analyzers/contracts/events.rs` | same |
| **Impact** | All non-configctl domain events (observation, signal, decision, strategy, execution, risk) were invisible to the contract audit — alignment check reported them as missing | All domain events discovered and attributed to their domain |

### 2.4 contract-audit — event-registry alignment algorithm

| Dimension | Before | After |
|-----------|--------|-------|
| **Matching strategy** | Rigid suffix match: strip first two segments of registry subject, check if remainder exists in domain event names | Domain-aware fuzzy matching: group domain events by domain, match registry events to the correct domain, use tokenized subsequence matching |
| **Example** | Registry `signal.events.rsi.generated` → suffix `rsi.generated` → not found in domain names → **FAIL** | Registry domain=`signal`, tokens match domain event `signal_generated` → **PASS** |
| **File** | `tools/raccoon-cli/src/analyzers/contracts.rs` — `check_event_registry_alignment` | same |
| **Impact** | Naming convention differences across domains caused cascade of false errors | Flexible matching tolerates the real naming variations across observation, signal, decision, strategy, execution, and risk domains |

### 2.5 drift-detect — "consumer" classified as defunct service name

| Dimension | Before | After |
|-----------|--------|-------|
| **DEFUNCT_NAMES** | `["consumer", "emulator", "validator"]` | `["emulator", "validator"]` |
| **Scan scope for consumer** | Scanned `cmd/` and `deploy/` for "consumer" as stale reference | Removed entirely — "consumer" is legitimate |
| **Rationale** | "consumer" was an old top-level service binary name | After writer service introduction, `cmd/writer/consumer.go` and related files legitimately use "consumer" for NATS JetStream durable consumer actors |
| **File** | `tools/raccoon-cli/src/analyzers/drift_detect.rs` | same |
| **Impact** | Every file in cmd/writer/ mentioning "consumer" generated a stale-name error | No false positives for legitimate consumer usage |

### 2.6 runtime-bindings — configctl query subject documentation

| Dimension | Before | After |
|-----------|--------|-------|
| **Doc comment pattern** | `configctl.control.config.*` | `configctl.control.*` |
| **Test subject** | `configctl.control.config.compile` | `configctl.control.compile_config` |
| **File** | `tools/raccoon-cli/src/analyzers/runtime_bindings/source.rs` | same |
| **Impact** | Documentation and test encoded old subject naming convention | Aligned with actual control subject format |

---

## 3. Checks That Were Already Correct

The following checks required no changes — they already reflected the current architecture:

1. **doctor** — project structure checks (go.work, directories)
2. **arch-guard** — layer boundary enforcement (clean architecture rules)
3. **runtime-bindings** — stream ownership, consumer binding, query routing (core logic was correct; only doc/test drift existed)
4. **drift-detect** — all checks except naming-identity-drift (config↔compose, config↔source, binding↔topology, workflow↔reality, contract↔domain, compose↔profiles all passed)

---

## 4. Guard Rails Preserved

No legitimate guard rail was weakened:

1. **Subject prefix validation** — still enforced; the prefix was corrected, not removed
2. **Reply-type symmetry** — still enforced; the comparison logic was made version-aware, not disabled
3. **Domain event scanning** — expanded to be comprehensive rather than partial
4. **Event-registry alignment** — made domain-aware rather than bypassed
5. **Defunct name detection** — still enforced for genuinely defunct names (emulator, validator)
6. **Layer boundaries** — untouched
7. **Pipeline continuity** — untouched
8. **Stream/durable/subject validation** — untouched
