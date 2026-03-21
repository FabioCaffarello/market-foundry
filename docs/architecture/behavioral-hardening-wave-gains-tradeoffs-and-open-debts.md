# Behavioral Hardening Wave — Gains, Trade-offs, and Open Debts

**Stage:** S257 (post-hardening assessment)
**Scope:** Hardening tranche S255–S256
**Date:** 2026-03-21

---

## 1. Gains

### G-H1: Full-Stack Behavioral Proof (S255)

The behavioral wave's single medium-risk gap — no proof that behavioral properties survive serialization and persistence — is now closed.

- 12 round-trip tests prove decision severity, strategy confidence, and risk metadata survive `mapRow()` → `FormatFloat()` → `ParseJSON()` cycles.
- 6 smoke-analytical checks validate the same properties against live NATS → ClickHouse → HTTP infrastructure.
- Float64 precision is lossless (delta < 1e-10 across 8 test values).
- Confidence ordering invariant (`risk ≤ strategy ≤ decision`) proven stable through serialization.

**Impact:** The behavioral model is now operationally provable, not just logically correct.

### G-H2: Silent Failure Modes Eliminated (S256)

Two categories of silent failure were removed:

1. **Severity input mismatch:** `"HIGH"`, `" high "`, `"Moderate"` previously defaulted silently to neutral (×1.00). Now normalized via `TrimSpace` + `ToLower` at lookup boundary.
2. **Degenerate zero-confidence approval:** Confidence ≤ 0 previously produced an "approved" assessment with 0-size position. Now produces `DispositionRejected` with observable metadata.

**Impact:** The behavioral surface fails explicitly instead of silently degrading.

### G-H3: Three-Layer Evidence Stack (Cumulative)

Post-hardening, the behavioral wave has a complete evidence pyramid:

| Layer | Tests | Proves |
|-------|-------|--------|
| In-process actor chain | 6 scenarios | Domain logic correctness |
| Unit (risk + strategy scaling) | 23 tests | Component behavioral accuracy + edges |
| Serialization round-trip | 12 tests | Write→read field fidelity |
| Full-stack smoke | 6 checks | Infrastructure semantic survival |
| **Total** | **47** | **Full behavioral surface** |

### G-H4: Zero Infrastructure Cost

The entire hardening tranche added:
- 0 new NATS subjects or streams
- 0 new ClickHouse tables or columns
- 0 new binaries or actors
- 1 stdlib dependency (`strings`)
- 0 domain model changes

### G-H5: Disposition Enum Completion

`DispositionRejected` was defined but unused before S256. The rejection path in both risk evaluators now exercises it, making the disposition enum functionally complete for the current behavioral surface.

---

## 2. Trade-offs Accepted

### T-H1: Normalization at Lookup, Not at Source

Severity normalization happens at the `lookupSeverityFactor` / `severityFactor` boundary, not at the point where severity enters the system. Original severity values are preserved in metadata for observability.

**Why accepted:** Normalizing at source would require upstream producer audits and contract changes. Normalizing at lookup is local, safe, and backward-compatible.

**Risk:** If a new lookup map is added without normalization, the same class of bug could recur. Mitigated by test coverage.

### T-H2: Round-Trip Tests Simulate ClickHouse Types

The 12 round-trip tests use Go native types to simulate ClickHouse column behavior rather than actual ClickHouse protocol encoding.

**Why accepted:** The gap is covered by the smoke-analytical script running against a live ClickHouse instance in Docker Compose. Protocol-level encoding bugs would surface there.

**Risk:** Very low — Go's database/sql driver handles protocol encoding; behavioral data uses standard types (String, Float64, DateTime).

### T-H3: Rejection Threshold Fixed at Zero

The rejection path triggers at confidence ≤ 0. There is no configurable rejection threshold.

**Why accepted:** Zero is the natural boundary between meaningful and degenerate assessments. A configurable threshold would require the configuration infrastructure (OD-BW2) that was explicitly deferred.

**Risk:** None for current use — negative or zero confidence should never produce a real position.

### T-H4: Partial OD-BW4 Closure

Severity normalization covers casing and whitespace but does not enforce strict enum validation (e.g., rejecting `"critical"` or `"unknown"` as invalid).

**Why accepted:** Default-to-neutral (×1.00) is safe and backward-compatible. Strict rejection could break upstream producers that haven't been audited. Full validation should accompany configuration infrastructure (OD-BW2).

**Risk:** Low — unrecognized severity silently defaults to neutral, which is conservative and observable.

---

## 3. Open Debts (Post-Hardening)

### OD-BW2: Configurable Scaling Factors

**Risk:** Low
**Status:** Deferred — requires non-existent configuration infrastructure
**Current state:** Hardcoded maps in `risk_scaling.go` and `severity_scaling.go`
**When to address:** When operational feedback indicates current values are inadequate, or when configctl reaches maturity for runtime reconfiguration
**Safe to freeze:** Yes — current values produce measurably correct behavioral divergence (2.56× position sizing proven)

### OD-BW5: Performance Budgets

**Risk:** Very low
**Status:** Deferred — no performance pressure exists
**Current state:** 47 behavioral tests execute in < 1s; pipeline is I/O-bound
**When to address:** When pipeline scaling creates CPU contention or when test suite grows beyond 200+ tests
**Safe to freeze:** Yes — enforcement would add CI complexity without current benefit

### OD-BW6: Configctl Activation (EX8)

**Risk:** Low
**Status:** Deferred — depends on OD-BW2 and configctl maturity
**Current state:** Behavioral scaling is active by default, not feature-flagged
**When to address:** When configctl is mature enough to manage runtime feature toggles
**Safe to freeze:** Yes — always-on is safer than a half-implemented toggle

### OD-BW4 (Remainder): Full Severity Validation

**Risk:** Low
**Status:** Partially closed (normalization done, strict validation deferred)
**Current state:** Unrecognized severity defaults to neutral (×1.00)
**When to address:** When upstream producer contracts are formalized
**Safe to freeze:** Yes — default-to-neutral is conservative and backward-compatible

### OD-BW7: Execution Layer

**Risk:** Out of scope
**Status:** Future charter
**Current state:** Behavioral properties stop at risk assessment; no execution/order layer exists
**When to address:** Dedicated execution charter
**Safe to freeze:** Yes — execution is a separate domain boundary

---

## 4. Debt Risk Summary

| Category | Count | Blocking? |
|----------|-------|-----------|
| Closed (S255–S256) | 3 | — |
| Deferred, Low risk | 3 | No |
| Deferred, Very low risk | 1 | No |
| Out of scope | 1 | No |
| **Total remaining** | **4 deferred + 1 out-of-scope** | **None blocking** |

All remaining debts are either low/very-low risk or explicitly out of scope. None require action before reopening the codegen/generated path.
