# Next Wave Recommendations After Post-Behavioral Hardening Gate

**Stage:** S257
**Date:** 2026-03-21
**Prerequisite:** BEHAVIORAL-WAVE-1 formally closed (S257 gate PASS)

---

## 1. Recommendation

**Return to the codegen/generated path.**

The behavioral wave is closed. The hardening tranche delivered its objectives. No blockers remain. The project should resume the codegen/generated path with raccoon-cli as architecture guardian.

---

## 2. Options Evaluated

### Option A: Return to Codegen/Generated Path — RECOMMENDED

The behavioral wave proved that the analytical domain model supports real behavioral composition (decision → strategy → risk). The codegen path can now expand the system's functional surface with confidence that behavioral properties are CI-protected and serialization-safe.

**Preconditions met:**
- Behavioral charter closed (S254 PASS)
- Full-stack smoke closed (S255 PASS, OD-BW1 closed)
- Edge hardening closed (S256 complete, OD-BW3 + OD-BW4 closed)
- Transition gate closed (S257 PASS)
- 47 behavioral tests in CI, 0 failures
- Zero medium+ risk debts remaining

**What this enables:**
- New analytical types or actor chains
- Infrastructure expansion (new NATS subjects, ClickHouse tables)
- raccoon-cli guardian enforcement of architectural boundaries
- Future behavioral evolution via new charters (not ad-hoc scope creep)

### Option B: Execute Another Hardening Micro-Tranche

Would address one or more of OD-BW2, OD-BW5, OD-BW6.

**Not recommended because:**
- All remaining debts are low/very-low risk
- OD-BW2 and OD-BW6 require configuration infrastructure that doesn't exist — building it as "hardening" would be scope creep
- OD-BW5 has no performance pressure to justify enforcement
- Further hardening delays codegen without proportional risk reduction
- Diminishing returns: the highest-value edges are already closed

### Option C: Open New Behavioral Charter

Would define a BEHAVIORAL-WAVE-2 targeting execution layer, configurable factors, or new behavioral dimensions.

**Not recommended now because:**
- The current behavioral model needs validation against real market data before deepening
- Execution layer (OD-BW7) is a separate domain boundary that deserves its own charter
- Opening a new behavioral charter before codegen produces more functional surface would create a narrow, top-heavy system

**When to revisit:** After 2–3 codegen stages produce enough functional surface to motivate new behavioral requirements.

### Option D: Pause for Operational Feedback

Would freeze all development and wait for operational metrics before deciding next steps.

**Not recommended because:**
- The system is not yet in production — there is no operational feedback to wait for
- The behavioral surface is CI-protected and safe to freeze
- Pausing delays value delivery without new information

---

## 3. Constraints for Codegen Re-Entry

When returning to the codegen path, the following constraints apply:

1. **Behavioral tests must remain green.** The `behavioral-scenarios` CI job is a hard gate. Any codegen change that breaks behavioral tests must be fixed before merge.

2. **No ad-hoc behavioral changes.** New behavioral dimensions, scaling factors, or strategy-type awareness require a formal charter. The behavioral surface is frozen until a new charter is opened.

3. **raccoon-cli guardian enforces boundaries.** Architectural constraints from the breadth and behavioral waves are guardian-checked. New code must comply.

4. **Deferred debts are not forgotten.** OD-BW2, OD-BW5, OD-BW6 remain in the debt ledger. They should be addressed when their preconditions materialize (configuration infrastructure, performance pressure, configctl maturity), not opportunistically during codegen.

---

## 4. Suggested First Codegen Stage

The first stage after re-entry should be scoped to:
- Validate that the codegen path works smoothly alongside the behavioral CI gates
- Produce a small, complete functional increment
- Confirm raccoon-cli guardian compatibility with new generated code

Specific scope should be determined by the next planning session based on current project priorities.

---

## 5. Decision Required

Accept one of:
- **Option A** — Return to codegen/generated path (recommended)
- **Option B** — Execute another hardening micro-tranche
- **Option C** — Open new behavioral charter
- **Option D** — Pause for operational feedback
