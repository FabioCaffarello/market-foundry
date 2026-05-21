# Next Wave Recommendations After Post-Codegen Reentry Gate

**Stage:** S263
**Date:** 2026-03-21
**Prerequisite:** Post-Codegen Reentry Gate PASSED (S258–S262)

---

## 1. Strategic Context

The Foundry has completed three consecutive infrastructure waves:

1. **Breadth wave** (S241–S244) — expanded domain from signal+evidence to all 6 layers.
2. **Behavioral wave** (S249–S257) — hardened domain logic with 47 behavioral tests, severity scaling, confidence mapping.
3. **Codegen reentry wave** (S258–S262) — reconciled specs, expanded governance to 22 artifacts, proved codegen-first.

After three infrastructure-focused waves, the Foundry has a mature domain surface, validated behavioral rules, and a trustworthy (if narrow) codegen system. The infrastructure now exists to support real feature evolution.

---

## 2. Options Evaluated

### Option A: Continue Expanding the Generated Path

**Description:** Dedicate the next wave to adding new artifact templates (store consumers, layer starters, mappers, config methods). Target: raise codegen coverage from 14% to ~39%.

**Pros:**
- Reduces manual boilerplate for future families.
- Store consumers and starters are the highest-ROI next candidates.
- Builds on momentum from S258–S262.

**Cons:**
- Three consecutive infrastructure waves risk stalling domain value delivery.
- Marginal ROI decreases: the 2 simplest artifact types are already governed.
- Store consumers may diverge from writer consumers by design — forcing codegen may create false equivalence.
- Requires spec schema evolution (OD-CG1) for mapper generation.

**Verdict:** NOT RECOMMENDED as dedicated wave. Incremental codegen improvements can proceed alongside feature work.

### Option B: Execute a Short Codegen Hardening Sprint

**Description:** 1–2 stages focused on hardening the existing codegen (CI integration, marker automation, error reporting).

**Pros:**
- Low cost, high confidence improvement.
- Makes codegen more trustworthy for ongoing use.

**Cons:**
- The codegen system already works well. The 7-phase equivalence check and integrated-check scripts are solid.
- Marker automation saves seconds per family — not worth a dedicated stage.
- Risk of over-engineering a narrow tool.

**Verdict:** NOT RECOMMENDED as dedicated wave. Any hardening can be folded into the first stage of the next wave as prep work.

### Option C: Feature Evolution — New Domain Capabilities

**Description:** Leverage the enriched infrastructure (6 layers, behavioral tests, codegen governance) to deliver new domain capabilities: new signal families, new decision strategies, new risk models, or new execution modes.

**Pros:**
- Directly delivers domain value — the reason the Foundry exists.
- Tests infrastructure under real load (do the layers, behavioral rules, and codegen actually support rapid feature development?).
- Validates the investment of the last three waves.
- Creates real feedback for where infrastructure gaps hurt most.

**Cons:**
- Feature work may expose infrastructure gaps that require backtracking.
- Codegen coverage at 14% means most new artifacts are still manual.

**Verdict:** RECOMMENDED. This is the highest-value next step.

### Option D: Pause Until a Specific Blocker Is Closed

**Description:** Hold all development until OD-BW2 (configurable scaling), OD-CG1 (column-opaque spec), or another debt is resolved.

**Pros:**
- Ensures clean foundation before building.

**Cons:**
- No current blocker is severe enough to justify a full pause.
- OD-BW2 and OD-CG1 are both Medium severity with known workarounds.
- Pausing loses momentum from three successful waves.

**Verdict:** NOT RECOMMENDED. No blocker warrants a full stop.

---

## 3. Recommendation: Option C — Feature Evolution

### Rationale

The Foundry's infrastructure is mature enough to support feature delivery:

- 6 domain layers are wired and operational.
- 47 behavioral tests guard against regression.
- 22 codegen-governed artifacts reduce wiring boilerplate.
- Equivalence validation prevents drift.
- Codegen-first workflow is proven for new families.

The best test of this infrastructure is to use it. Feature evolution will either confirm that the investment paid off or reveal specific gaps that need targeted fixing — both outcomes are valuable.

### Suggested Feature Wave Scope

The next wave should focus on a bounded set of domain capabilities that exercise multiple layers:

1. **New signal families** — candidates: MACD, VWAP, ATR. These exercise the codegen-first workflow at scale and validate that Bollinger wasn't an outlier.
2. **New decision families** — candidates: Bollinger Squeeze decision consuming the new Bollinger signal. This validates cross-layer wiring (signal → decision) with a codegen-first family.
3. **Enhanced risk models** — candidates: correlation-based position limits, volatility-adjusted sizing. These exercise the risk layer enrichments from the behavioral wave.

### Constraints on the Feature Wave

- **Codegen-first for new families:** All new signal/decision families must follow the spec-first workflow proven in S262. No manual-first retrofitting.
- **Behavioral tests required:** Every new behavioral rule must have a test before merge.
- **No infrastructure expansion:** The feature wave must work within current infrastructure. If a gap is found, document it as debt — do not expand scope.
- **Codegen improvements allowed as side-effects:** If adding a new family reveals a template bug or script gap, fix it. But don't plan codegen expansion as a primary objective.

### Success Criteria for the Feature Wave

- At least 2 new codegen-first families delivered and passing all validation.
- At least 1 cross-layer interaction (e.g., signal → decision) validated end-to-end.
- Zero regression in existing 47 behavioral tests.
- All new families governed by codegen (consumer_spec + pipeline_entry).
- Net assessment of whether infrastructure supports rapid feature delivery.

---

## 4. What Should NOT Happen Next

1. **Do not open a fourth consecutive infrastructure wave.** The Foundry needs to deliver domain value.
2. **Do not scale codegen to new artifact types as a primary objective.** Let feature work drive codegen evolution naturally.
3. **Do not attempt DDL generation or mapper generation.** These require spec schema evolution (OD-CG1) and are premature.
4. **Do not skip the charter/scope freeze for the feature wave.** The discipline that made S258–S262 successful should continue.
5. **Do not treat codegen-first as mandatory for all work.** Codegen-first applies to new families with repetitive wiring. Manual-first remains valid for unique or experimental features.
