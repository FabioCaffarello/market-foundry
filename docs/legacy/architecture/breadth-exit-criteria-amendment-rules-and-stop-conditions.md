# Breadth Exit Criteria, Amendment Rules, and Stop Conditions

**Charter:** BREADTH-WAVE-1
**Reference:** breadth-charter-and-scope-freeze.md

---

## 1. Exit Criteria

The charter is considered **passed** only when ALL of the following criteria are met. Partial exit is explicitly prohibited — all criteria must be satisfied or the charter must be formally amended.

### 1.1 Mandatory Exit Criteria (All Required)

| # | Criterion | Measurement | Pass Condition |
|---|-----------|-------------|----------------|
| E1 | Decision breadth | Distinct evaluator types in `internal/application/decision/` | ≥ 2 types with distinct family names and distinct signal sources |
| E2 | Strategy breadth | Distinct resolver types in `internal/application/strategy/` | ≥ 2 types with distinct family names and distinct resolution logic |
| E3 | Risk breadth | Distinct evaluator types in `internal/application/risk/` | ≥ 2 types with distinct family names and distinct risk dimensions |
| E4 | Domain validation | Each new type has domain-level validation tests | 100% of new types have domain unit tests |
| E5 | Application logic | Each new type has application-layer unit tests | 100% of new types have evaluator/resolver unit tests |
| E6 | Actor integration | Each new type is wired into the actor chain | Fan-out messages, publisher messages, actor tests present |
| E7 | Chain integration | End-to-end integration test covering both chains | ≥ 2 distinct chain paths exercised in integration tests |
| E8 | Codegen families | Each new type has a codegen family YAML | 3 new family YAMLs in `codegen/families/` |
| E9 | CI green | All tests pass in CI | Remote CI green on the breadth gate stage |

### 1.2 How Each Criterion Is Verified

- **E1–E3:** File count + code review confirming distinct logic (not wrappers or aliases of existing types)
- **E4–E5:** `go test ./internal/domain/... ./internal/application/...` passes with new test files present
- **E6:** Actor test files exist; fan-out routing confirmed in actor code
- **E7:** `actor_chain_integration_test.go` covers both Chain A and Chain B paths
- **E8:** `ls codegen/families/` shows ≥ 6 family YAMLs (3 existing + 3 new)
- **E9:** CI workflow passes on the gate commit

### 1.3 What Does NOT Count as Exit

- Adding fields to existing evaluators (depth, not breadth)
- Tests without corresponding feature code
- Infrastructure improvements without feature delivery
- Documentation-only stages
- "Placeholder" evaluators with stub logic

---

## 2. Amendment Rules

These rules are inherited from the S239 governance framework and hardened for this charter.

### 2.1 The Five Rules (Binding)

| Rule | Description | Enforcement |
|------|-------------|-------------|
| R1 | **Pre-execution documentation** | No scope change may be implemented before an amendment record is written. Implementation commits without prior amendment record are governance violations. |
| R2 | **Exit criteria explicitly updated** | Every amendment must state which exit criteria (E1–E9) are affected and what replaces them. Silent criterion removal is prohibited. |
| R3 | **Mid-charter gate mandatory** | After S242, a formal gate reviews breadth progress. If ≥ 2 domains still lack second types, the charter is suspended for review. |
| R4 | **No retroactive modification** | Original charter text is immutable. Amendments are appended to the Amendments Log in `breadth-charter-and-scope-freeze.md`. |
| R5 | **Post-hoc amendments flagged** | If a deviation is discovered after execution, it must be acknowledged with: what happened, why it wasn't caught, and corrective action. |

### 2.2 Amendment Triggers (Mandatory)

An amendment record MUST be filed before proceeding if any of the following occur:

1. **Target change:** A candidate evaluator/resolver is swapped for a different one
2. **Deferral:** A domain's second type is deferred to a future charter
3. **Depth creep:** Depth work (enriching existing types) exceeds 20% of a stage's effort
4. **Sequence change:** Implementation order diverges from S241→S242→S243
5. **Scope expansion:** Any work outside the three target types is undertaken
6. **Blocking prerequisite:** A depth or infrastructure change is required before a breadth deliverable can proceed

### 2.3 Amendment Record Format

```markdown
## Amendment #N

**Date:** YYYY-MM-DD
**Stage:** S2XX
**Type:** [Target Change | Deferral | Depth Allowance | Sequence Change | Scope Expansion | Prerequisite]

### Original Commitment
[What was originally planned]

### Proposed Change
[What is changing and why]

### Exit Criteria Impact
[Which E1–E9 criteria are affected and how]

### Revised Metric
[New pass condition, if applicable]
```

### 2.4 Amendment Escalation

- **Minor amendment** (sequence change, single-domain target swap): Document and proceed
- **Major amendment** (deferral of any domain, scope expansion): Requires explicit stop-and-review before proceeding
- **Charter-breaking amendment** (dropping breadth as primary objective): Charter must be closed and a new charter opened

---

## 3. Stop Conditions

### 3.1 Automatic Suspension Triggers

The charter is **automatically suspended** and requires explicit review before continuation if:

| # | Condition | Detection Point |
|---|-----------|-----------------|
| S1 | Two consecutive stages fail to deliver their primary breadth deliverable | Stage gate review |
| S2 | Mid-charter gate (after S242) shows ≥ 2 domains still at single-type coverage | Mid-charter gate |
| S3 | Depth work consumption exceeds 20% of effort in any single stage | Stage gate review |
| S4 | A blocking architectural issue requires redesign affecting > 1 domain | During implementation |
| S5 | CI remains red for > 1 stage due to breadth-related changes | Stage gate review |

### 3.2 Suspension Protocol

When a stop condition is triggered:

1. **Halt** — No further feature implementation until review completes
2. **Diagnose** — Identify root cause: scope error, architectural gap, or governance failure
3. **Decide** — One of:
   - **Resume** with formal amendment documenting the correction
   - **Rescope** the charter with reduced breadth targets (must amend exit criteria)
   - **Close** the charter as failed and open a new one with lessons learned
4. **Document** — Decision and rationale recorded in the Amendments Log

### 3.3 Warning Indicators (Non-Blocking)

These do not trigger suspension but should be flagged in stage reports:

- Hardening budget (20%) approaching in any stage
- Test count growth disproportionate to feature delivery
- Actor integration complexity higher than expected
- Fan-out routing requiring architectural changes beyond message additions

---

## 4. Mid-Charter Gate Protocol (After S242)

### 4.1 Gate Timing

The mid-charter gate executes after S242 delivery (strategy resolver #2) and before S243 begins.

### 4.2 Gate Checklist

| # | Question | Expected Answer | Failure Action |
|---|----------|-----------------|----------------|
| G1 | Does Decision domain have ≥ 2 evaluator types? | Yes (rsi_oversold + ema_crossover) | Suspend charter |
| G2 | Does Strategy domain have ≥ 2 resolver types? | Yes (mean_reversion_entry + trend_following_entry) | Suspend charter |
| G3 | Were any amendments filed? | Record count | Review each for pattern |
| G4 | Did depth work stay within 20% budget? | Yes for both S241 and S242 | Flag if exceeded |
| G5 | Is CI green? | Yes | Fix before proceeding |
| G6 | Is the charter still achievable in remaining stages? | Yes | Amend or close |

### 4.3 Gate Outcomes

- **PASS:** Proceed to S243 (risk evaluator #2)
- **CONDITIONAL PASS:** Proceed with documented corrective action
- **FAIL:** Charter suspended; follow suspension protocol (§3.2)

---

## 5. Final Gate Protocol (S245)

### 5.1 Binary Evaluation

The final gate is a **pass/fail** evaluation against all nine exit criteria (E1–E9). There is no partial pass.

### 5.2 Final Gate Report Must Include

1. Exit criteria matrix with pass/fail per criterion
2. Amendment log review (were rules followed?)
3. Breadth measurement matrix (from targets document) with actual values
4. Chain integration evidence (both paths exercised)
5. CI evidence (green build on gate commit)
6. Lessons learned for next charter

### 5.3 Post-Gate

- **PASS:** Breadth wave complete. Next charter may address depth, codegen evolution, or additional breadth.
- **FAIL:** Document which criteria failed, why, and whether the charter should be re-attempted or closed with partial credit recorded.
