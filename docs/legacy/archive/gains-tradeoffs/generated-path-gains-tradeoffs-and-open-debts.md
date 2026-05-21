# Generated Path — Gains, Tradeoffs, and Open Debts

**Gate:** S204 (Post-Generated Family Gate)
**Scope:** Evidence from S199–S203, covering governance framework, first slice integration, coexistence hardening, EMA family definition, and EMA implementation/validation.

---

## Gains

### G-1: Naming Correctness Enforcement

The codegen engine deterministically derives 10 naming fields from each spec (ConsumerSpecFunc, ConsumerName, InserterName, IsEnabledMethod, RegistryField, NewConsumerFunc, PascalFamily, PascalLayer, InsertSQL, HyphenFamily). This eliminates a class of errors that occurred in manual family authoring: inconsistent durable names, wrong function names, mismatched registry references.

**Evidence**: 7 families × 10 derived fields = 70 derivations, all correct. Known abbreviations (RSI, EMA) handled without special-casing by the developer.

### G-2: Cross-Spec Uniqueness Validation

`codegen validate-all` enforces global uniqueness of `family.name`, `nats.durable`, and `nats.subject` across all 7 specs. This catches collisions at spec-authoring time, before any code is written.

**Evidence**: Cross-spec validation runs in CI, blocks merge on collision. No collisions found across 7 families.

### G-3: Governance Auditability

The spec → golden → marker → target → integrated check chain provides a tamper-evident trail. Any deviation from the generated output is detectable by CI. PR reviewers can verify generated fragments against golden snapshots without running the engine.

**Evidence**: 4/4 integrated checks pass. Marker integrity validated (begin/end presence, non-empty region). Golden snapshot diffs are visible in PRs.

### G-4: Deterministic Reproducibility

The same spec always produces the same output. Generation is not dependent on the developer's machine state, Go version (beyond template library), or any external service. This is a property worth preserving — it means generated code is auditable offline.

**Evidence**: Golden snapshots are version-controlled. `check-all` confirms determinism across regeneration.

### G-5: Structural Equivalence Proof

12/12 golden comparisons across all 6 existing families demonstrate that codegen output is structurally equivalent to hand-crafted code. The normalization pipeline (comment strip, whitespace collapse, empty line removal) handles cosmetic variation without false positives.

**Evidence**: S196 validated full coverage. S203 extended to 14/14 with EMA.

### G-6: CI Regression Gate

The `codegen-golden` CI job blocks merge on golden comparison failure. The `codegen-integrated` check blocks merge on governance drift. Together, they ensure that no generated family can regress without detection.

**Evidence**: CI chain operational since S200. Reordered to fail-fast in S201.

---

## Tradeoffs

### T-1: Manual Insertion Accepted Over Automated Integration

**What was accepted**: Developers must manually copy generated fragments from golden snapshots into target files and add codegen markers. This costs ~5 minutes per family and is error-prone.

**Why accepted**: Automated insertion adds complexity to the codegen tool, was not justified for 2 governed families, and risks introducing a new failure mode. The governance model works with manual insertion.

**When to revisit**: When ≥3 governed families make manual insertion the dominant cost of the generated path.

### T-2: Fragment Generation, Not File Generation

**What was accepted**: The codegen engine produces code fragments (a function, a struct literal), not complete files. Fragments are inserted into existing files alongside manually-authored code.

**Why accepted**: File generation would require managing file-level concerns (imports, package declaration, shared functions) that differ per target file. Fragment generation keeps the engine simple and the boundary clear.

**When to revisit**: Not expected to change. Fragment generation is a design choice, not a limitation.

### T-3: 2/6 Tier 1 Artifacts Generated

**What was accepted**: Only consumer spec (A1) and pipeline entry (A2) are generated. Mapper (A3), mapper tests (A4), config entry (A5), and smoke phase (A6) remain manual.

**Why accepted**: A1+A2 are fully spec-derivable with the current 14-field schema. A3 requires `domain.columns`, A4 depends on A3, A5 requires JSONC tooling, A6 requires shell template engine. Expanding scope without evidence is premature.

**When to revisit**: A3 (mapper) after ≥2 codegen-first families validate A1+A2. A5/A6 not expected to trigger soon.

### T-4: Frozen Schema and Templates

**What was accepted**: The 14-field spec schema and both templates are frozen. No evolution during generated path operation.

**Why accepted**: Freezing eliminates a class of coordination problems — template changes affect all families; schema changes require migration of all existing specs. Stability is more valuable than expressiveness at this stage.

**When to revisit**: Schema extension (e.g., `domain.columns`) requires a formal evolution ceremony with its own validation stage. Template changes require re-validation of all golden snapshots.

### T-5: Golden Snapshot Maintenance Cost

**What was accepted**: 2 golden files per family, growing linearly. Currently 14 files for 7 families.

**Why accepted**: Golden files provide offline auditability, PR diffability, and fast CI comparison. The alternative (regenerate in CI) is slower and less transparent.

**When to revisit**: If family count exceeds ~20 and golden snapshot maintenance becomes noisy in PRs. Not a near-term concern.

### T-6: Structural-Only Activation Proof for EMA

**What was accepted**: EMA's pipeline entry compiles, is registered, and the consumer subscribes — but no EMA events are emitted by any producer. SC-6 is PARTIAL.

**Why accepted**: Building an EMA signal producer is domain work outside codegen scope. Structural proof (compilation + declaration + subscription attempt) is sufficient to validate the generated path's correctness.

**When to revisit**: Before the third generated family. Structural-only proof is acceptable for the next iteration but cannot become permanent.

---

## Open Debts

### Priority: HIGH

| Debt | Description | Trigger |
|------|-------------|---------|
| D-1: Live event flow proof | No generated family has proven live event flow end-to-end | Before third generated family |
| D-2: Cross-layer validation | Generated path tested on signal layer only; other layers unvalidated | Before first non-signal generated family |
| D-3: Mapper generation feasibility | A3 is 100% manual; `domain.columns` spec extension not designed | After ≥2 codegen-first families validate A1+A2 model |

### Priority: MEDIUM

| Debt | Description | Trigger |
|------|-------------|---------|
| D-4: Automated fragment insertion | Manual copy-paste costs ~5 min/family; error-prone at scale | When ≥3 governed families make toil dominant |
| D-5: Config registration automation | `knownSignalFamilies` and `signalDependsOnEvidence` are spec-derivable but manually maintained | After spec schema evolution ceremony is warranted |
| D-6: Test assertion robustness | Hardcoded family count in tests breaks on each new family | Opportunistic fix; not blocking |

### Priority: LOW

| Debt | Description | Trigger |
|------|-------------|---------|
| D-7: CODEGEN_ROOT auto-detection | `go run` fails without explicit env var | Developer experience improvement; not blocking |
| D-8: Batch generation efficiency | Only 1 family per iteration; no evidence of batch efficiency | After pattern proven with ≥3 families |
| D-9: Automated scope guard | No technical mechanism prevents generating artifacts beyond A1+A2 | Only if review discipline is insufficient; currently working |

### NOT SCHEDULED

| Debt | Description | Why Deferred |
|------|-------------|-------------|
| D-10: Tier 2 (read-path) generation | Not authorized, not designed | Requires its own architecture stage |
| D-11: Template evolution | Templates frozen; evolution requires re-validation of all goldens | No template pressure observed |
| D-12: Spec schema extension | 14-field schema is sufficient for A1+A2; extension needed only for A3+ | Schema evolution ceremony not warranted yet |
| D-13: Retroactive manual-to-generated conversion | 6 manual families remain permanently manual | By design — golden references |

---

## Items That Do Not Justify the Cost Now

| Item | Why Not Now |
|------|-----------|
| Automated file patching / insertion | Only 2 governed families; ~10 min total manual work per iteration. Automation effort exceeds savings. |
| Mapper generation (A3) | Requires `domain.columns` spec extension, new template, new golden snapshots, new validation. EMA reuses `mapSignalRow` — no mapper generation pressure exists. |
| Config entry generation (A5) | Requires JSONC tooling. 1 line per family. Manual cost: ~1 minute. |
| Smoke test phase generation (A6) | Requires shell template engine. Each family's smoke is ~3 lines. Manual cost: ~2 minutes. |
| Multi-family batch generation | No process or evidence supports batch authoring. Single-family iteration is the proven model. |
| Pre-commit hooks for codegen | CI latency is acceptable (~2s for codegen checks). Pre-commit hooks add developer friction without clear benefit. |
