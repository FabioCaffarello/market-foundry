# First Generated Family — Findings, Generated Path Frictions, and Limits

**Stage:** S203
**Date:** 2026-03-20
**Family:** EMA (Exponential Moving Average)

---

## Friction Summary

| # | Friction | Severity | Impact | Mitigation |
|---|---------|----------|--------|------------|
| F-1 | Manual fragment insertion | Medium | Developer must copy generated output into target files and add markers | Automated marker-based insertion (deferred) |
| F-2 | Config registration not generated | Low | `knownSignalFamilies` and `signalDependsOnEvidence` require manual update | Config registration could be spec-derivable in future |
| F-3 | No live activation proof without event producer | Medium | EMA pipeline compiles but cannot be proven active without an EMA signal emitter | Synthetic event injection or derive service extension |
| F-4 | Test count assertion fragility | Low | `TestKnownFamiliesReturnsRegisteredNames` used hardcoded count (2), broke when EMA added | Changed to 3; consider using `>=` or dynamic count |
| F-5 | CODEGEN_ROOT environment variable required | Low | `codegen validate-all` and `check-all` fail when run via `go run` without CODEGEN_ROOT | Document requirement; consider auto-detection |
| F-6 | Golden snapshot duplication | Low | Golden files duplicate what the engine produces; no single source of truth beyond spec+template | By design for CI auditability; accepted cost |

## Detailed Findings

### F-1: Manual Fragment Insertion

**Observation:** After generating A1 and A2 from the EMA spec, the developer must manually:
1. Copy consumer spec output into `signal_registry.go`
2. Add `codegen:begin` / `codegen:end` markers around the fragment
3. Copy pipeline entry output into `pipeline.go`
4. Add markers around the fragment

**Friction cost:** ~5 minutes of careful copy-paste per family. Error-prone: wrong indentation, missing markers, or partial copy could break governance checks.

**Why this matters:** For a single family, the cost is negligible. For 10+ families, it becomes the dominant cost of the generated path. This is the primary friction that would motivate automated marker-based insertion.

**Status:** Accepted for this phase. Automation is deferred (not authorized).

### F-2: Config Registration Not Generated

**Observation:** Adding EMA required manual changes to `settings/schema.go`:
- Add `"ema": true` to `knownSignalFamilies`
- Add `"ema": {"candle"}` to `signalDependsOnEvidence`

These are spec-derivable: `family.name`, `family.layer`, and a hypothetical `family.depends_on` field could drive these registrations.

**Why not generated now:** The spec schema is frozen at 14 fields (S198 condition). Adding `depends_on` requires a schema evolution ceremony.

**Status:** Documented as future friction reduction opportunity.

### F-3: No Live Activation Proof

**Observation:** The EMA pipeline entry compiles into the writer binary and is registered with `isEnabled` pointing to `IsSignalFamilyEnabled("ema")`. With `"ema"` in `signal_families` config, the pipeline will be declared at startup.

However, no derive service currently emits `signal.events.ema.generated.>` events. The pipeline's consumer will subscribe to an empty subject and report zero events received.

**Impact:** SC-6 ("pipeline activates with config, smoke test observes EMA actors") can be partially validated — the pipeline is declared and the consumer attempts to subscribe — but cannot produce evidence of event flow.

**Possible resolutions for S204:**
1. Synthetic NATS event injection during smoke test
2. Extend derive service to emit EMA signals (requires domain work)
3. Accept structural proof (compilation + declaration + subscription) as sufficient

**Status:** Documented as known limit.

### F-4: Test Count Assertion Fragility

**Observation:** `TestKnownFamiliesReturnsRegisteredNames` in `settings_test.go` asserted `len(signals) != 2`. Adding EMA broke this assertion immediately.

**Root cause:** Hardcoded count in test, not synchronized with the canonical family registry.

**Fix applied:** Updated assertion from 2 to 3.

**Better pattern (deferred):** Assert `>= 1` or compute expected count from the registry itself. This is a test design issue, not a codegen issue.

### F-5: CODEGEN_ROOT Required for `go run`

**Observation:** `codegen validate-all` and `check-all` resolve `familiesDir` and `goldenDir` relative to the executable path. When invoked via `go run`, the executable is in a temporary cache directory, causing path resolution to fail.

**Workaround:** Set `CODEGEN_ROOT=$(pwd)` before running.

**Impact:** Minor — affects developer workflow but not CI (CI can set the variable). Already documented in codegen usage.

### F-6: Golden Snapshot Duplication

**Observation:** Golden snapshots duplicate the deterministic output of `codegen generate`. They exist solely for CI comparison — the engine could regenerate them at any time.

**Why this is by design:** Golden files provide:
1. Offline audit trail (no engine execution needed to review)
2. PR diffability (changes to golden files are visible in code review)
3. CI comparison target (faster than re-generating during CI)

**Cost:** 2 files per family (14 total with 7 families). Manageable at current scale.

## Limits of the Generated Path (This Phase)

| Dimension | Current Limit | What Would Change It |
|-----------|--------------|---------------------|
| Artifacts generated | A1 + A2 only | Template expansion authorization |
| Families per iteration | 1 | Gate review (S204) |
| Layers covered | Signal only (this family) | Cross-layer family selection |
| Infrastructure creation | Zero — reuse only | New table/stream families |
| Mapper generation | Not supported | `domain.columns` spec extension |
| Read-path generation | Not supported | Tier 2 authorization |
| Config registration | Manual | Spec schema extension |
| Fragment insertion | Manual | Automation authorization |
| Template evolution | Frozen | New authorization stage |
| Spec schema evolution | Frozen (14 fields) | New authorization stage |

## What the Generated Path Proved

1. **Mechanical correctness:** Spec → derive → template → output → golden → target → compile — all steps produce correct Go code without human editing
2. **Governance enforcement:** Markers + golden comparison + integrated check create a tamper-evident chain
3. **Infrastructure reuse works:** Signal-layer families can share all infrastructure; the only new code is the consumer spec function and pipeline entry
4. **Naming derivation is reliable:** `knownAbbreviations` handles `"ema" → "EMA"` correctly; all 10 derived fields match expectations
5. **Cross-spec validation scales:** 7 families validated with zero collisions on durable/subject/name

## What the Generated Path Did NOT Prove

1. **Cross-layer generation:** EMA is signal-layer only; decision/strategy/risk/execution layers not tested
2. **New infrastructure families:** EMA reuses everything; families requiring new tables/streams are not covered
3. **Mapper generation:** `mapSignalRow` was reused; no evidence that mapper code can be generated
4. **Multi-family iteration speed:** Only one family added; no evidence of batch efficiency
5. **Live event flow:** No EMA events exist in the pipeline; activation proof is structural only

## Recommendations for S204

1. **Gate decision:** The generated path works for same-layer, same-infrastructure families. Recommend authorizing a second codegen-first family if it shares existing infrastructure.
2. **Friction prioritization:** F-1 (manual insertion) is the only friction that scales with family count. Consider prioritizing automated insertion if multi-family expansion is planned.
3. **Live activation:** Decide whether SC-6 requires live event flow or structural proof. If live, plan a synthetic injection mechanism or derive extension.
4. **Config registration automation:** Track as a candidate for spec schema extension in a future phase.
